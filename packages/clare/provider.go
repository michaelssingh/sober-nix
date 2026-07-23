package main

import (
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
	client := &http.Client{Timeout: 4 * time.Second}
	req, err := http.NewRequest("GET", streamURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Range", "bytes=0-4096")
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

	// If master playlist, fetch the first sub-playlist to check for corrupted PNG ad segments
	if strings.Contains(content, "#EXTM3U") && strings.Contains(content, ".m3u8") {
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") && strings.HasSuffix(line, ".m3u8") {
				subURL := line
				if !strings.HasPrefix(subURL, "http") {
					lastIdx := strings.LastIndex(streamURL, "/")
					if lastIdx != -1 {
						subURL = streamURL[:lastIdx+1] + subURL
					}
				}
				subReq, err := http.NewRequest("GET", subURL, nil)
				if err == nil {
					for k, v := range headers {
						subReq.Header.Set(k, v)
					}
					subResp, err := client.Do(subReq)
					if err == nil {
						subBuf := make([]byte, 4096)
						sn, _ := subResp.Body.Read(subBuf)
						subResp.Body.Close()
						subContent := string(subBuf[:sn])
						if strings.Contains(subContent, ".png") || strings.Contains(subContent, "ibyteimg") {
							return fmt.Errorf("preflight rejected stream: sub-playlist contains PNG ad segments")
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
