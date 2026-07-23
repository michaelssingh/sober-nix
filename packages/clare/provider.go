package main

import (
	"bytes"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

type Provider interface {
	Name() string
	Search(query string, mode string) ([]AnimeShow, error)
	FetchEpisodes(showID string, mode string) (AnimeShow, []string, error)
	ResolveStreams(showID, mode, episodeNo, quality string) ([]ResolvedStream, error)
}

// PreflightStreamURL checks if a stream URL responds with 200 OK and valid video content
func PreflightStreamURL(streamURL string, headers map[string]string) error {
	client := &http.Client{Timeout: 2 * time.Second}
	req, err := http.NewRequest("GET", streamURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Range", "bytes=0-2048")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", UserAgent)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("preflight connection failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("preflight returned status %d", resp.StatusCode)
	}

	buf := make([]byte, 4096)
	n, _ := resp.Body.Read(buf)
	content := string(buf[:n])

	// Filter out Cloudflare challenge pages and HLS playlists corrupted with PNG ad inserts (e.g. ibyteimg)
	if strings.Contains(content, "Cloudflare") || strings.Contains(content, "Attention Required!") || strings.Contains(content, "cf-mitigated") || strings.Contains(content, ".png") || strings.Contains(content, "ibyteimg") || strings.Contains(content, "ad-site") {
		return fmt.Errorf("preflight rejected stream: Cloudflare challenge page or PNG ad detected")
	}

	// If master playlist, fetch the first sub-playlist to check for corrupted PNG ad segments
	if strings.Contains(content, "#EXTM3U") {
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				subURL := line
				if !strings.HasPrefix(subURL, "http") {
					lastIdx := strings.LastIndex(streamURL, "/")
					if lastIdx != -1 {
						subURL = streamURL[:lastIdx+1] + subURL
					}
				}
				subReq, err := http.NewRequest("GET", subURL, nil)
				if err == nil {
					subReq.Header.Set("Range", "bytes=0-2048")
					for k, v := range headers {
						subReq.Header.Set(k, v)
					}
					subResp, err := client.Do(subReq)
					if err == nil {
						subBuf := make([]byte, 4096)
						sn, _ := subResp.Body.Read(subBuf)
						subResp.Body.Close()
						subContent := string(subBuf[:sn])
						if strings.Contains(subContent, "Cloudflare") || strings.Contains(subContent, "Attention Required!") || strings.Contains(subContent, ".png") || strings.Contains(subContent, "ibyteimg") || strings.Contains(subContent, "ad-site") {
							return fmt.Errorf("preflight rejected stream: Cloudflare challenge or PNG ad in sub-playlist")
						}

						// Probe first segment URL to detect disguised PNG image ads
						subLines := strings.Split(subContent, "\n")
						for _, sline := range subLines {
							sline = strings.TrimSpace(sline)
							if sline != "" && !strings.HasPrefix(sline, "#") {
								segURL := sline
								if !strings.HasPrefix(segURL, "http") {
									lastIdx := strings.LastIndex(subURL, "/")
									if lastIdx != -1 {
										segURL = subURL[:lastIdx+1] + segURL
									}
								}
								segReq, err := http.NewRequest("GET", segURL, nil)
								if err == nil {
									segReq.Header.Set("Range", "bytes=0-511")
									for k, v := range headers {
										segReq.Header.Set(k, v)
									}
									segResp, err := client.Do(segReq)
									if err == nil {
										segBuf := make([]byte, 512)
										sgn, _ := segResp.Body.Read(segBuf)
										segResp.Body.Close()
										cType := segResp.Header.Get("Content-Type")
										if strings.Contains(cType, "image/") || strings.Contains(cType, "png") || bytes.Contains(segBuf[:sgn], []byte("PNG")) || bytes.Contains(segBuf[:sgn], []byte("\x89PNG")) {
											return fmt.Errorf("preflight rejected stream: segment is image/png ad insert (%s)", cType)
										}
									}
								}
								break
							}
						}
					}
				}
				break
			}
		}
	}

	return nil
}

func getRefererForURL(urlVal string) string {
	if strings.Contains(urlVal, "mp4upload.com") {
		return "https://www.mp4upload.com/"
	} else if strings.Contains(urlVal, "flikhub") || strings.Contains(urlVal, "kotocdn") || strings.Contains(urlVal, "megap") {
		return "https://megaplay.buzz/"
	} else if strings.Contains(urlVal, "allanime") || strings.Contains(urlVal, "alltropic") || strings.Contains(urlVal, "ok.ru") {
		return "https://youtu-chan.com/"
	}
	return StreamReferer
}

type MultiProviderResolver struct {
	providers []Provider
}

func NewMultiProviderResolver() *MultiProviderResolver {
	return &MultiProviderResolver{
		providers: []Provider{
			&AllAnimeProvider{},
			&FlikhubProvider{},
			&GogoanimeProvider{},
		},
	}
}

func (r *MultiProviderResolver) Search(query, mode string) ([]AnimeShow, error) {
	var allShows []AnimeShow
	for _, p := range r.providers {
		shows, err := p.Search(query, mode)
		if err == nil && len(shows) > 0 {
			allShows = append(allShows, shows...)
		}
	}
	if len(allShows) == 0 {
		return nil, fmt.Errorf("no shows found across all providers for query %q", query)
	}
	LogEventInfo(DomainSearch, "multi-provider search completed",
		slog.String("query", query),
		slog.Int("total_shows", len(allShows)),
	)
	return allShows, nil
}

func (r *MultiProviderResolver) ResolveWithFallback(query, mode, episodeNo, quality string) (AnimeShow, ResolvedStream, error) {
	shows, err := r.Search(query, mode)
	if err != nil {
		return AnimeShow{}, ResolvedStream{}, err
	}

	for _, s := range shows {
		for _, p := range r.providers {
			streams, err := p.ResolveStreams(s.ID, mode, episodeNo, quality)
			if err != nil || len(streams) == 0 {
				continue
			}
			for _, st := range streams {
				headers := map[string]string{"Referer": getRefererForURL(st.URL)}
				if err := PreflightStreamURL(st.URL, headers); err == nil {
					LogEventInfo(DomainResolve, "preflight success",
						slog.String("provider", p.Name()),
						slog.String("show", s.Name),
						slog.String("url", st.URL),
					)
					return s, st, nil
				} else {
					LogEventError(DomainResolve, "preflight failed for stream candidate", err,
						slog.String("provider", p.Name()),
						slog.String("url", st.URL),
					)
				}
			}
		}
	}
	return AnimeShow{}, ResolvedStream{}, fmt.Errorf("failed to resolve a preflighted working stream across all providers")
}
