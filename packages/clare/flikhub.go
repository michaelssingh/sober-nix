package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

type FlikhubProvider struct{}

func (p *FlikhubProvider) Name() string {
	return "flikhub"
}

func cleanFlikhubTitle(raw string) string {
	if raw == "" {
		return raw
	}
	reHash := regexp.MustCompile(`-[a-z0-9]{5}$`)
	cleaned := reHash.ReplaceAllString(raw, "")
	cleaned = strings.ReplaceAll(cleaned, "-", " ")
	words := strings.Fields(cleaned)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

type FlikhubSearchItem struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Image string `json:"image"`
}

func (p *FlikhubProvider) Search(query string, mode string) ([]AnimeShow, error) {
	// Query Flikhub search API directly with URL-encoded query
	searchURL := fmt.Sprintf("https://api.flikhub.net/search?q=%s", url.QueryEscape(query))
	headers := map[string]string{
		"User-Agent": UserAgent,
	}

	body, err := doHTTPReqWithRetry("GET", searchURL, nil, headers)
	if err != nil {
		return nil, fmt.Errorf("Flikhub search failed: %w", err)
	}

	var items []FlikhubSearchItem
	if err := json.Unmarshal(body, &items); err != nil {
		return nil, fmt.Errorf("failed to parse Flikhub search JSON: %w", err)
	}

	var shows []AnimeShow
	for _, item := range items {
		cleanTitle := cleanFlikhubTitle(item.Title)
		shows = append(shows, AnimeShow{
			ID:          item.ID,
			Provider:    "flikhub",
			Name:        cleanTitle,
			EnglishName: cleanTitle,
			Thumbnail:   item.Image,
		})
	}

	return shows, nil
}

type FlikhubInfoResponse struct {
	Title         string   `json:"title"`
	JapaneseTitle string   `json:"japaneseTitle"`
	Synopsis      string   `json:"synopsis"`
	Image         string   `json:"image"`
	Rating        string   `json:"rating"`
	Genres        []string `json:"genres"`
	MalRating     float64  `json:"malRating"`
	Duration      string   `json:"duration"`
	Premiered     string   `json:"premiered"`
	TotalEpisodes int      `json:"totalEpisodes"`
}

type FlikhubEpisodeItem struct {
	Number int    `json:"number"`
	HasSub bool   `json:"hasSub"`
	HasDub bool   `json:"hasDub"`
	MALID  string `json:"malId"`
}

func (p *FlikhubProvider) FetchEpisodes(showID string, mode string) (AnimeShow, []string, error) {
	headers := map[string]string{
		"User-Agent": UserAgent,
	}

	// 1. Fetch info details
	infoURL := fmt.Sprintf("https://api.flikhub.net/info?id=%s", showID)
	infoBody, err := doHTTPReqWithRetry("GET", infoURL, nil, headers)
	if err != nil {
		return AnimeShow{}, nil, fmt.Errorf("failed to fetch Flikhub show info: %w", err)
	}

	var info FlikhubInfoResponse
	if err := json.Unmarshal(infoBody, &info); err != nil {
		return AnimeShow{}, nil, fmt.Errorf("failed to parse Flikhub info JSON: %w", err)
	}

	// 2. Fetch episodes list
	episodesURL := fmt.Sprintf("https://api.flikhub.net/episodes?id=%s", showID)
	episodesBody, err := doHTTPReqWithRetry("GET", episodesURL, nil, headers)
	if err != nil {
		return AnimeShow{}, nil, fmt.Errorf("failed to fetch Flikhub episodes: %w", err)
	}

	var episodesList []FlikhubEpisodeItem
	if err := json.Unmarshal(episodesBody, &episodesList); err != nil {
		return AnimeShow{}, nil, fmt.Errorf("failed to parse Flikhub episodes list JSON: %w", err)
	}

	// Find the MAL ID from the episode items
	var malID string
	for _, ep := range episodesList {
		if ep.MALID != "" {
			malID = ep.MALID
			break
		}
	}

	show := AnimeShow{
		ID:          showID,
		Provider:    "flikhub",
		Name:        info.Title,
		EnglishName: info.Title,
		NativeName:  info.JapaneseTitle,
		Thumbnail:   info.Image,
		Description: info.Synopsis,
		MALID:       malID,
		Score:       info.MalRating,
		Duration:    info.Duration,
		Genres:      info.Genres,
		Rating:      info.Rating,
	}

	var episodes []string
	subMap := make(map[string][]string)
	var subEps []string
	var dubEps []string

	for _, ep := range episodesList {
		epStr := strconv.Itoa(ep.Number)
		episodes = append(episodes, epStr)
		if ep.HasSub {
			subEps = append(subEps, epStr)
		}
		if ep.HasDub {
			dubEps = append(dubEps, epStr)
		}
	}

	subMap["sub"] = subEps
	subMap["dub"] = dubEps
	show.AvailableEpisodesDetail = subMap

	return show, episodes, nil
}

type FlikhubMegaplayResponse struct {
	M3u8       string `json:"m3u8"`
	ProxiedUrl string `json:"proxiedUrl"`
	Tracks     []struct {
		File  string `json:"file"`
		Label string `json:"label"`
		Kind  string `json:"kind"`
	} `json:"tracks"`
}

func (p *FlikhubProvider) ResolveStreams(showID, mode, episodeNo, quality string) ([]ResolvedStream, error) {
	// Resolve MAL ID from showID or cached show details
	malID := showID
	showName := ""
	if cachedShow, _, found := loadShowCache(showID); found {
		if cachedShow.MALID != "" {
			malID = cachedShow.MALID
		}
		if cachedShow.Name != "" {
			showName = cachedShow.Name
		} else if cachedShow.EnglishName != "" {
			showName = cachedShow.EnglishName
		}
	} else {
		// Fallback to fetch episodes to populate cache
		show, eps, err := p.FetchEpisodes(showID, mode)
		if err == nil {
			_ = saveShowCache(showID, show, eps)
			if show.MALID != "" {
				malID = show.MALID
			}
			if show.Name != "" {
				showName = show.Name
			}
		}
	}

	// If malID is not numeric (e.g. it's an AllAnime ID), perform a Flikhub title lookup to resolve the MAL ID
	if _, err := strconv.Atoi(malID); err != nil && showName != "" {
		if shows, err := p.Search(showName, mode); err == nil && len(shows) > 0 {
			for _, s := range shows {
				if flikShow, eps, err := p.FetchEpisodes(s.ID, mode); err == nil && len(eps) > 0 && flikShow.MALID != "" {
					malID = flikShow.MALID
					break
				}
			}
		}
	}

	if _, err := strconv.Atoi(malID); err != nil {
		return nil, fmt.Errorf("Flikhub requires a numeric MAL ID, unable to resolve from: %s", showID)
	}

	flikhubURL := fmt.Sprintf("https://api.flikhub.net/megaplay?mal=%s&ep=%s&type=%s", malID, episodeNo, mode)
	headers := map[string]string{
		"User-Agent": UserAgent,
	}

	body, err := doHTTPReqWithRetry("GET", flikhubURL, nil, headers)
	if err != nil {
		return nil, fmt.Errorf("Flikhub resolve streams failed: %w", err)
	}

	var res FlikhubMegaplayResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, fmt.Errorf("failed to parse Flikhub megaplay JSON: %w", err)
	}

	var subtitles []SubtitleTrack
	for _, track := range res.Tracks {
		if track.Kind == "captions" || track.Kind == "subtitles" {
			subtitles = append(subtitles, SubtitleTrack{
				Label: track.Label,
				URL:   track.File,
			})
		}
	}

	var streams []ResolvedStream
	if res.ProxiedUrl != "" {
		streams = append(streams, ResolvedStream{
			Provider:   "flikhub",
			SourceName: "Flikhub-Proxy",
			Quality:    "best",
			URL:        res.ProxiedUrl,
			Subtitles:  subtitles,
		})
	}
	if res.M3u8 != "" {
		streams = append(streams, ResolvedStream{
			Provider:   "flikhub",
			SourceName: "Flikhub-Direct",
			Quality:    "best",
			URL:        res.M3u8,
			Subtitles:  subtitles,
		})
	}

	if len(streams) == 0 {
		return nil, fmt.Errorf("no stream URLs found in Flikhub response")
	}

	return streams, nil
}
