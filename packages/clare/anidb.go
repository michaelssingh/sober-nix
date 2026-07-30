package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

type AniDBProvider struct{}

func (p *AniDBProvider) Name() string {
	return "anidb"
}

func (p *AniDBProvider) Search(query, mode string) ([]AnimeShow, error) {
	searchURL := fmt.Sprintf("https://anidb.app/browse?q=%s", url.QueryEscape(query))
	headers := map[string]string{
		"User-Agent": UserAgent,
		"Referer":    "https://anidb.app/",
	}

	body, err := doHTTPReqWithRetry("GET", searchURL, nil, headers)
	if err != nil {
		return nil, fmt.Errorf("anidb search failed: %w", err)
	}

	re := regexp.MustCompile(`<a href="https://anidb\.app/anime/([a-zA-Z0-9_-]+-([0-9]+))"[^>]*title="([^"]+)"`)
	matches := re.FindAllStringSubmatch(string(body), -1)

	seen := make(map[string]bool)
	var shows []AnimeShow

	for _, m := range matches {
		if len(m) >= 4 {
			slugWithID := m[1]
			idNum := m[2]
			title := strings.TrimSpace(m[3])

			if seen[idNum] {
				continue
			}
			seen[idNum] = true

			shows = append(shows, AnimeShow{
				ID:          "anidb:" + idNum,
				Name:        title,
				EnglishName: title,
				Type:        "TV",
				Description: slugWithID,
			})
		}
	}

	return shows, nil
}

func (p *AniDBProvider) FetchEpisodes(showID, mode string) (AnimeShow, []string, error) {
	cleanID := stripProviderPrefix(showID)
	episodesURL := fmt.Sprintf("https://anidb.app/api/frontend/anime/%s/episodes", cleanID)
	headers := map[string]string{
		"User-Agent": UserAgent,
		"Referer":    "https://anidb.app/",
	}

	body, err := doHTTPReqWithRetry("GET", episodesURL, nil, headers)
	if err != nil {
		return AnimeShow{}, nil, fmt.Errorf("anidb fetch episodes failed: %w", err)
	}

	var resp struct {
		Episodes []struct {
			ID     int  `json:"id"`
			Number int  `json:"number"`
			Filler bool `json:"filler"`
		} `json:"episodes"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return AnimeShow{}, nil, fmt.Errorf("anidb unmarshal episodes failed: %w", err)
	}

	var epList []string
	for _, ep := range resp.Episodes {
		epList = append(epList, strconv.Itoa(ep.Number))
	}

	show := AnimeShow{
		ID:   "anidb:" + cleanID,
		Name: "AniDB Show (" + cleanID + ")",
	}

	return show, epList, nil
}

func (p *AniDBProvider) ResolveStreams(showID, mode, episodeNo, quality string) ([]ResolvedStream, error) {
	cleanID := stripProviderPrefix(showID)
	episodesURL := fmt.Sprintf("https://anidb.app/api/frontend/anime/%s/episodes", cleanID)
	headers := map[string]string{
		"User-Agent": UserAgent,
		"Referer":    "https://anidb.app/",
	}

	body, err := doHTTPReqWithRetry("GET", episodesURL, nil, headers)
	if err != nil {
		return nil, fmt.Errorf("anidb resolve streams fetch episodes failed: %w", err)
	}

	var resp struct {
		Episodes []struct {
			ID     int `json:"id"`
			Number int `json:"number"`
		} `json:"episodes"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("anidb unmarshal episodes failed: %w", err)
	}

	targetEpNo, _ := strconv.Atoi(episodeNo)
	var epID int
	for _, ep := range resp.Episodes {
		if ep.Number == targetEpNo {
			epID = ep.ID
			break
		}
	}

	if epID == 0 && len(resp.Episodes) > 0 {
		epID = resp.Episodes[0].ID
	}
	if epID == 0 {
		return nil, fmt.Errorf("anidb episode %s not found for show %s", episodeNo, cleanID)
	}

	languagesURL := fmt.Sprintf("https://anidb.app/api/frontend/episode/%d/languages", epID)
	langBody, err := doHTTPReqWithRetry("GET", languagesURL, nil, headers)
	if err != nil {
		return nil, fmt.Errorf("anidb fetch languages failed: %w", err)
	}

	var langResp struct {
		Languages []struct {
			Code     string `json:"code"`
			Name     string `json:"name"`
			EmbedURL string `json:"embed_url"`
		} `json:"languages"`
	}

	if err := json.Unmarshal(langBody, &langResp); err != nil {
		return nil, fmt.Errorf("anidb unmarshal languages failed: %w", err)
	}

	if len(langResp.Languages) == 0 {
		return nil, fmt.Errorf("anidb no languages available for episode %d", epID)
	}

	targetCode := "jpn"
	if strings.EqualFold(mode, "dub") {
		targetCode = "eng"
	}

	var selectedEmbed string
	for _, l := range langResp.Languages {
		if strings.EqualFold(l.Code, targetCode) {
			selectedEmbed = l.EmbedURL
			break
		}
	}

	if selectedEmbed == "" {
		selectedEmbed = langResp.Languages[0].EmbedURL
	}

	embedBody, err := doHTTPReqWithRetry("GET", selectedEmbed, nil, headers)
	if err != nil {
		return nil, fmt.Errorf("anidb fetch embed page failed: %w", err)
	}

	reM3U8 := regexp.MustCompile(`https?://[^\s\"\'\<\>]+\.m3u8[^\s\"\'\<\>]*`)
	m3u8Matches := reM3U8.FindAllString(string(embedBody), -1)
	if len(m3u8Matches) == 0 {
		return nil, fmt.Errorf("anidb m3u8 stream URL not found in embed page")
	}

	streamURL := m3u8Matches[0]
	streamHeaders := map[string]string{
		"User-Agent": UserAgent,
		"Referer":    "https://anidb.app/",
	}

	if err := PreflightStreamURL(streamURL, streamHeaders); err != nil {
		return nil, fmt.Errorf("anidb stream preflight failed: %w", err)
	}

	return []ResolvedStream{
		{
			Provider:   "anidb",
			SourceName: "AniDB-HLS",
			Quality:    "1080p",
			URL:        streamURL,
		},
	}, nil
}
