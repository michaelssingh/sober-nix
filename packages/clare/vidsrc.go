package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

type VidSrcProvider struct{}

func (p *VidSrcProvider) Name() string {
	return "vidsrc"
}

func (p *VidSrcProvider) Search(query, mode string) ([]AnimeShow, error) {
	tmdbSearchURL := fmt.Sprintf("https://api.themoviedb.org/3/search/multi?api_key=***REDACTED***&query=%s", url.QueryEscape(query))
	body, err := doHTTPReqWithRetry("GET", tmdbSearchURL, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("vidsrc search request failed: %w", err)
	}

	var searchResp struct {
		Results []struct {
			ID           int    `json:"id"`
			MediaType    string `json:"media_type"`
			Title        string `json:"title"`
			Name         string `json:"name"`
			OriginalName string `json:"original_name"`
			Overview     string `json:"overview"`
			ReleaseDate  string `json:"release_date"`
			FirstAirDate string `json:"first_air_date"`
		} `json:"results"`
	}

	if err := json.Unmarshal(body, &searchResp); err != nil {
		return nil, fmt.Errorf("vidsrc search parse failed: %w", err)
	}

	var shows []AnimeShow
	for _, item := range searchResp.Results {
		name := item.Title
		if name == "" {
			name = item.Name
		}
		if name == "" {
			name = item.OriginalName
		}
		if name == "" {
			continue
		}

		mediaType := item.MediaType
		if mediaType != "movie" && mediaType != "tv" {
			continue
		}

		year := 0
		dateStr := item.ReleaseDate
		if dateStr == "" {
			dateStr = item.FirstAirDate
		}
		if len(dateStr) >= 4 {
			fmt.Sscanf(dateStr[:4], "%d", &year)
		}

		showID := fmt.Sprintf("vidsrc:%s:%d", mediaType, item.ID)
		show := AnimeShow{
			ID:          showID,
			Name:        name,
			Description: item.Overview,
			Provider:    "vidsrc",
			Type:        strings.ToUpper(mediaType),
		}
		show.Season.Year = FlexInt(year)
		shows = append(shows, show)
	}

	return shows, nil
}

func (p *VidSrcProvider) FetchEpisodes(showID, mode string) (AnimeShow, []string, error) {
	cleanID := strings.TrimPrefix(showID, "vidsrc:")
	parts := strings.Split(cleanID, ":")
	if len(parts) < 2 {
		return AnimeShow{}, nil, fmt.Errorf("invalid vidsrc show ID: %s", showID)
	}
	mediaType := parts[0]
	tmdbID := parts[1]

	if mediaType == "movie" {
		return AnimeShow{
			ID:       showID,
			Name:     "Movie",
			Provider: "vidsrc",
		}, []string{"1"}, nil
	}

	tmdbDetailURL := fmt.Sprintf("https://api.themoviedb.org/3/tv/%s?api_key=***REDACTED***", tmdbID)
	body, err := doHTTPReqWithRetry("GET", tmdbDetailURL, nil, nil)
	if err != nil {
		return AnimeShow{}, nil, fmt.Errorf("vidsrc tv details failed: %w", err)
	}

	var tvDetail struct {
		Name             string `json:"name"`
		NumberOfEpisodes int    `json:"number_of_episodes"`
		Seasons          []struct {
			SeasonNumber int `json:"season_number"`
			EpisodeCount int `json:"episode_count"`
		} `json:"seasons"`
	}

	if err := json.Unmarshal(body, &tvDetail); err != nil {
		return AnimeShow{}, nil, fmt.Errorf("vidsrc tv details parse failed: %w", err)
	}

	var epList []string
	epNum := 1
	for _, season := range tvDetail.Seasons {
		if season.SeasonNumber <= 0 {
			continue
		}
		for e := 1; e <= season.EpisodeCount; e++ {
			epList = append(epList, fmt.Sprintf("%d", epNum))
			epNum++
		}
	}

	if len(epList) == 0 {
		epList = []string{"1"}
	}

	return AnimeShow{
		ID:       showID,
		Name:     tvDetail.Name,
		Provider: "vidsrc",
	}, epList, nil
}

func (p *VidSrcProvider) ResolveStreams(showID, mode, episodeNo, quality string) ([]ResolvedStream, error) {
	cleanID := strings.TrimPrefix(showID, "vidsrc:")
	parts := strings.Split(cleanID, ":")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid vidsrc show ID: %s", showID)
	}
	mediaType := parts[0]
	tmdbID := parts[1]

	var embedURL string
	if mediaType == "movie" {
		embedURL = fmt.Sprintf("https://vidsrcme.su/embed/movie/%s", tmdbID)
	} else {
		embedURL = fmt.Sprintf("https://vidsrcme.su/embed/tv/%s/1/%s", tmdbID, episodeNo)
	}

	streamURL, err := unpackVidSrcEmbed(embedURL)
	if err != nil {
		return nil, fmt.Errorf("vidsrc unpack failed: %w", err)
	}

	return []ResolvedStream{
		{
			Provider:   "vidsrc",
			SourceName: "VidSrc-HD",
			Quality:    "best",
			URL:        streamURL,
		},
	}, nil
}

func unpackVidSrcEmbed(embedURL string) (string, error) {
	headers := map[string]string{
		"User-Agent": UserAgent,
		"Referer":    "https://vidsrcme.su/",
	}

	body1, err := doHTTPReqWithRetry("GET", embedURL, nil, headers)
	if err != nil {
		return "", fmt.Errorf("stage 1 embed page failed: %w", err)
	}
	html1 := string(body1)

	reIframe := regexp.MustCompile(`<iframe[^>]*src="([^"]+)"`)
	match1 := reIframe.FindStringSubmatch(html1)
	if len(match1) < 2 {
		return "", fmt.Errorf("stage 1 iframe src not found")
	}
	rcpURL := match1[1]
	if strings.HasPrefix(rcpURL, "//") {
		rcpURL = "https:" + rcpURL
	}

	body2, err := doHTTPReqWithRetry("GET", rcpURL, nil, headers)
	if err != nil {
		return "", fmt.Errorf("stage 2 rcp page failed: %w", err)
	}
	html2 := string(body2)

	reProrcp := regexp.MustCompile(`src:\s*["'](/prorcp/[^"']+)["']`)
	match2 := reProrcp.FindStringSubmatch(html2)
	if len(match2) < 2 {
		return "", fmt.Errorf("stage 2 prorcp path not found")
	}
	prorcpURL := "https://cloudorchestranova.com" + match2[1]

	prorcpHeaders := map[string]string{
		"User-Agent": UserAgent,
		"Referer":    rcpURL,
	}
	body3, err := doHTTPReqWithRetry("GET", prorcpURL, nil, prorcpHeaders)
	if err != nil {
		return "", fmt.Errorf("stage 3 prorcp page failed: %w", err)
	}
	html3 := string(body3)

	reM3u8 := regexp.MustCompile(`https?://[^\s"'\<>]+?\.m3u8[^\s"'\<>]*`)
	rawM3u8 := reM3u8.FindString(html3)
	if rawM3u8 == "" {
		return "", fmt.Errorf("stage 3 m3u8 url not found")
	}

	parsedM3u8, err := url.Parse(rawM3u8)
	if err != nil {
		return "", fmt.Errorf("stage 3 parse m3u8 url failed: %w", err)
	}
	masterHost := parsedM3u8.Host

	tokenURL := fmt.Sprintf("https://%s/generate.php", masterHost)
	tokenHeaders := map[string]string{
		"User-Agent": UserAgent,
		"Referer":    prorcpURL,
	}
	tokenBody, err := doHTTPReqWithRetry("GET", tokenURL, nil, tokenHeaders)
	if err != nil {
		return "", fmt.Errorf("stage 4 generate token failed: %w", err)
	}
	token := strings.TrimSpace(string(tokenBody))

	finalM3u8 := strings.Replace(rawM3u8, "__TOKEN__", token, 1)
	return finalM3u8, nil
}
