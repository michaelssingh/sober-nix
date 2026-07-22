package main

import (
	"fmt"
	"regexp"
	"strings"
)

type GogoanimeProvider struct{}

func (p *GogoanimeProvider) Name() string {
	return "gogoanime"
}

func (p *GogoanimeProvider) Search(query string, mode string) ([]AnimeShow, error) {
	// Gogoanime.by search URL
	searchURL := fmt.Sprintf("https://gogoanime.by/search.html?keyword=%s", strings.ReplaceAll(query, " ", "%20"))
	headers := map[string]string{
		"User-Agent": UserAgent,
	}

	body, err := doHTTPReqWithRetry("GET", searchURL, nil, headers)
	if err != nil {
		return nil, err
	}

	// Scrape search results
	var shows []AnimeShow
	reShow := regexp.MustCompile(`<span class="ellipsis"><a href="https://gogoanime\.by/series/([^/]+)/">([^<]+)</a></span>`)
	
	matches := reShow.FindAllStringSubmatch(string(body), -1)

	for _, m := range matches {
		if len(m) >= 3 {
			shows = append(shows, AnimeShow{
				ID:        m[1],
				Provider:  "gogoanime",
				Name:      m[2],
				Thumbnail: "",
			})
		}
	}

	return shows, nil
}

func (p *GogoanimeProvider) FetchEpisodes(showID string, mode string) (AnimeShow, []string, error) {
	// Fetch Gogoanime show page (e.g. https://gogoanime.by/series/gachiakuta-tv-serial/)
	showURL := fmt.Sprintf("https://gogoanime.by/series/%s/", showID)
	headers := map[string]string{
		"User-Agent": UserAgent,
	}

	body, err := doHTTPReqWithRetry("GET", showURL, nil, headers)
	if err != nil {
		return AnimeShow{}, nil, err
	}

	// Extract Title
	title := showID
	reTitle := regexp.MustCompile(`<h1 class="entry-title" itemprop="name">([^<]+)</h1>`)
	if m := reTitle.FindStringSubmatch(string(body)); len(m) >= 2 {
		title = strings.TrimSpace(m[1])
	}

	// Extract Description
	desc := ""
	reDesc := regexp.MustCompile(`<span itemprop="description">([^<]+)</span>`)
	if m := reDesc.FindStringSubmatch(string(body)); len(m) >= 2 {
		desc = strings.TrimSpace(m[1])
	}

	// Scrape episodes from the episodes-container
	var episodes []string
	reEp := regexp.MustCompile(`<div[^>]*class="episode-item"[^>]*data-episode-number="(\d+)"`)
	epMatches := reEp.FindAllStringSubmatch(string(body), -1)

	// Episode list in WordPress theme is usually descending (latest first). Reverse to ascending.
	for i := len(epMatches) - 1; i >= 0; i-- {
		episodes = append(episodes, epMatches[i][1])
	}

	show := AnimeShow{
		ID:          showID,
		Provider:    "gogoanime",
		Name:        title,
		Description: desc,
	}

	return show, episodes, nil
}

func (p *GogoanimeProvider) ResolveStreams(showID, mode, episodeNo, quality string) ([]ResolvedStream, error) {
	// 1. Fetch the series page to get the exact episode URL
	showURL := fmt.Sprintf("https://gogoanime.by/series/%s/", showID)
	headers := map[string]string{
		"User-Agent": UserAgent,
	}

	body, err := doHTTPReqWithRetry("GET", showURL, nil, headers)
	if err != nil {
		return nil, err
	}

	// Find the matching episode item and its link
	reEpLink := regexp.MustCompile(fmt.Sprintf(`<div[^>]*class="episode-item"[^>]*data-episode-number="%s"[^>]*>\s*<a href="([^"]+)">`, episodeNo))
	linkMatch := reEpLink.FindStringSubmatch(string(body))
	if len(linkMatch) < 2 {
		return nil, fmt.Errorf("unable to find Gogoanime link for episode %s on series page", episodeNo)
	}

	epURL := linkMatch[1]

	// 2. Fetch the episode page
	epBody, err := doHTTPReqWithRetry("GET", epURL, nil, headers)
	if err != nil {
		return nil, err
	}

	// 3. Extract the embed_url from the script tag or iframe src
	reEmbed := regexp.MustCompile(`var embed_url = "([^"]+)"`)
	embedMatch := reEmbed.FindStringSubmatch(string(epBody))
	
	var embedURL string
	if len(embedMatch) >= 2 {
		embedURL = embedMatch[1]
	} else {
		// Fallback to iframe src search
		reIframe := regexp.MustCompile(`<iframe class="gov-embed-iframe" src="([^"]+)"`)
		iframeMatch := reIframe.FindStringSubmatch(string(epBody))
		if len(iframeMatch) < 2 {
			return nil, fmt.Errorf("unable to find player embed URL on episode page")
		}
		embedURL = iframeMatch[1]
	}

	// 4. Fetch the embed URL to get the download redirect link
	embedBody, err := doHTTPReqWithRetry("GET", embedURL, nil, headers)
	if err != nil {
		return nil, err
	}

	reDownload := regexp.MustCompile(`href="([^"]+download[^"]+)"`)
	downloadMatch := reDownload.FindStringSubmatch(string(embedBody))
	if len(downloadMatch) < 2 {
		return nil, fmt.Errorf("unable to find download link on embed page")
	}

	downloadURL := downloadMatch[1]
	downloadURL = strings.ReplaceAll(downloadURL, "&amp;", "&")

	// 5. Fetch the GogoPlay download page which hosts direct high quality MP4 links
	dlBody, err := doHTTPReqWithRetry("GET", downloadURL, nil, headers)
	if err != nil {
		return nil, err
	}

	// 6. Scrape direct MP4 download links from the page
	var results []ResolvedStream
	reDirect := regexp.MustCompile(`<a href="([^"]+)"[^>]*>Download\s*\(([^)]+)\)`)
	matches := reDirect.FindAllStringSubmatch(string(dlBody), -1)

	for _, m := range matches {
		if len(m) >= 3 {
			qualityStr := strings.ToLower(m[2])
			qualityStr = strings.ReplaceAll(qualityStr, " - mp4", "")
			qualityStr = strings.ReplaceAll(qualityStr, " - hls", "")
			qualityStr = strings.TrimSpace(qualityStr)

			results = append(results, ResolvedStream{
				Provider:   "gogoanime",
				SourceName: "Gogo",
				Quality:    qualityStr,
				URL:        m[1],
			})
		}
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no streams resolved on Gogoanime download page")
	}

	return results, nil
}
