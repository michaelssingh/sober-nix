package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
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
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	client := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, "GET", streamURL, nil)
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

	// Filter out Cloudflare challenge pages, HTML web pages, and HLS playlists corrupted with PNG ad inserts
	if strings.Contains(content, "Cloudflare") || strings.Contains(content, "Attention Required!") || strings.Contains(content, "cf-mitigated") || strings.Contains(content, ".png") || strings.Contains(content, "ibyteimg") || strings.Contains(content, "ad-site") {
		return fmt.Errorf("preflight rejected stream: Cloudflare challenge page or PNG ad detected")
	}

	if strings.Contains(content, "<html") || strings.Contains(content, "<!DOCTYPE") || strings.Contains(content, "<iframe") {
		if !strings.Contains(content, "#EXTM3U") {
			return fmt.Errorf("preflight rejected stream: URL is an HTML webpage, not a direct HLS/MP4 stream")
		}
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
						if strings.Contains(subContent, "Cloudflare") || strings.Contains(subContent, "Attention Required!") {
							return fmt.Errorf("preflight rejected stream: Cloudflare challenge in sub-playlist")
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
	} else if strings.Contains(urlVal, "cloudorchestranova") || strings.Contains(urlVal, "zealotsofzenith") {
		return "https://cloudorchestranova.com/"
	} else if strings.Contains(urlVal, "nextgen") || strings.Contains(urlVal, "quietmidnight") || strings.Contains(urlVal, "influencerstrategy") || strings.Contains(urlVal, "profitablelaunchsystem") || strings.Contains(urlVal, "vaplayer") || strings.Contains(urlVal, ".site/") {
		return "https://nextgencloudfabric.com/"
	}
	return StreamReferer
}

type MultiProviderResolver struct {
	providers []Provider
}

func NewMultiProviderResolver() *MultiProviderResolver {
	cfg := loadConfig()
	allProviders := []Provider{
		&AniDBProvider{},
		&AllAnimeProvider{},
		&VidSrcProvider{},
		&FlikhubProvider{},
		&GogoanimeProvider{},
	}
	var active []Provider
	for _, p := range allProviders {
		if cfg.IsProviderEnabled(p.Name()) {
			active = append(active, p)
		}
	}
	return &MultiProviderResolver{
		providers: active,
	}
}

func rankSearchMatch(showName, query string) int {
	cleanShow := strings.ToLower(strings.TrimSpace(showName))
	cleanQuery := strings.ToLower(strings.TrimSpace(query))
	cleanShowNoSpace := strings.ReplaceAll(cleanShow, " ", "")
	cleanQueryNoSpace := strings.ReplaceAll(cleanQuery, " ", "")

	// Tier 1: Exact match
	if cleanShow == cleanQuery || cleanShowNoSpace == cleanQueryNoSpace {
		return 100
	}
	// Tier 2: Show title starts with query
	if strings.HasPrefix(cleanShow, cleanQuery) || strings.HasPrefix(cleanShowNoSpace, cleanQueryNoSpace) {
		return 80
	}
	// Tier 3: Query is contained in show title
	if strings.Contains(cleanShow, cleanQuery) || strings.Contains(cleanShowNoSpace, cleanQueryNoSpace) {
		return 60
	}
	// Tier 4: Weak match
	return 10
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

	// Sort search results by title relevance score (stable sort preserving provider quality order for equal scores)
	sort.SliceStable(allShows, func(i, j int) bool {
		scoreI := rankSearchMatch(allShows[i].Name, query)
		if english := allShows[i].EnglishName; english != "" {
			if s := rankSearchMatch(english, query); s > scoreI {
				scoreI = s
			}
		}

		scoreJ := rankSearchMatch(allShows[j].Name, query)
		if english := allShows[j].EnglishName; english != "" {
			if s := rankSearchMatch(english, query); s > scoreJ {
				scoreJ = s
			}
		}

		return scoreI > scoreJ
	})

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
		provName := ""
		if idx := strings.Index(s.ID, ":"); idx > 0 {
			provName = s.ID[:idx]
		}
		for _, p := range r.providers {
			if provName != "" && p.Name() != provName {
				continue
			}
			streams, err := p.ResolveStreams(s.ID, mode, episodeNo, quality)
			if err != nil || len(streams) == 0 {
				continue
			}

			type preflightRes struct {
				stream ResolvedStream
				err    error
			}
			resChan := make(chan preflightRes, len(streams))
			for _, st := range streams {
				go func(candidate ResolvedStream) {
					headers := map[string]string{"Referer": getRefererForURL(candidate.URL)}
					err := PreflightStreamURL(candidate.URL, headers)
					resChan <- preflightRes{stream: candidate, err: err}
				}(st)
			}

			for i := 0; i < len(streams); i++ {
				res := <-resChan
				if res.err == nil {
					LogEventInfo(DomainResolve, "preflight success",
						slog.String("provider", p.Name()),
						slog.String("show", s.Name),
						slog.String("url", res.stream.URL),
					)
					return s, res.stream, nil
				} else {
					LogEventError(DomainResolve, "preflight failed for stream candidate", res.err,
						slog.String("provider", p.Name()),
						slog.String("url", res.stream.URL),
					)
				}
			}
		}
	}
	return AnimeShow{}, ResolvedStream{}, fmt.Errorf("failed to resolve a preflighted working stream across all providers")
}
