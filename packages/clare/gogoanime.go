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
	// Gogoanime standard search URL
	searchURL := fmt.Sprintf("https://gogoanime3.co/search.html?keyword=%s", strings.ReplaceAll(query, " ", "%20"))
	headers := map[string]string{
		"User-Agent": UserAgent,
	}

	body, err := doHTTPReqWithRetry("GET", searchURL, nil, headers)
	if err != nil {
		return nil, err
	}

	// Scrape search results
	var shows []AnimeShow
	reShow := regexp.MustCompile(`<p class="name">\s*<a href="([^"]+)" title="([^"]+)">`)
	reImg := regexp.MustCompile(`<div class="img">\s*<a href="[^"]+" title="[^"]+">\s*<img src="([^"]+)"`)
	
	matches := reShow.FindAllStringSubmatch(string(body), -1)
	imgMatches := reImg.FindAllStringSubmatch(string(body), -1)

	for i, m := range matches {
		if len(m) >= 3 {
			showID := m[1]
			// IDs are typically like "/category/gachiakuta" - strip the prefix
			showID = strings.TrimPrefix(showID, "/category/")
			
			thumbnail := ""
			if i < len(imgMatches) && len(imgMatches[i]) >= 2 {
				thumbnail = imgMatches[i][1]
			}

			shows = append(shows, AnimeShow{
				ID:        showID,
				Provider:  "gogoanime",
				Name:      m[2],
				Thumbnail: thumbnail,
			})
		}
	}

	return shows, nil
}

func (p *GogoanimeProvider) FetchEpisodes(showID string, mode string) (AnimeShow, []string, error) {
	// Fetch Gogoanime show page (e.g. https://gogoanime3.co/category/gachiakuta)
	showURL := fmt.Sprintf("https://gogoanime3.co/category/%s", showID)
	headers := map[string]string{
		"User-Agent": UserAgent,
	}

	body, err := doHTTPReqWithRetry("GET", showURL, nil, headers)
	if err != nil {
		return AnimeShow{}, nil, err
	}

	// Extract Title & Description
	title := showID
	reTitle := regexp.MustCompile(`<h1>([^<]+)</h1>`)
	if m := reTitle.FindStringSubmatch(string(body)); len(m) >= 2 {
		title = strings.TrimSpace(m[1])
	}

	desc := ""
	reDesc := regexp.MustCompile(`<p class="type"><span>Plot Summary:</span>([^<]+)</p>`)
	if m := reDesc.FindStringSubmatch(string(body)); len(m) >= 2 {
		desc = strings.TrimSpace(m[1])
	}

	// Gogoanime loads episodes via an ajax request using episode start/end IDs
	// Let's scrape the ep_start and ep_end parameters from the page
	reStart := regexp.MustCompile(`ep_start\s*=\s*'(\d+)'`)
	reEnd := regexp.MustCompile(`ep_end\s*=\s*'(\d+)'`)
	reMovieID := regexp.MustCompile(`id="movie_id"\s*value="(\d+)"`)

	startMatch := reStart.FindStringSubmatch(string(body))
	endMatch := reEnd.FindStringSubmatch(string(body))
	movieIDMatch := reMovieID.FindStringSubmatch(string(body))

	var episodes []string
	if len(startMatch) >= 2 && len(endMatch) >= 2 && len(movieIDMatch) >= 2 {
		movieID := movieIDMatch[1]
		epStart := startMatch[1]
		epEnd := endMatch[1]

		// Call the episode list Ajax endpoint
		ajaxURL := fmt.Sprintf("https://ajax.gogo-load.com/ajax/load-list-episode?ep_start=%s&ep_end=%s&id=%s&default_ep=0", epStart, epEnd, movieID)
		ajaxBody, err := doHTTPReqWithRetry("GET", ajaxURL, nil, headers)
		if err == nil {
			// Parse episode list links (e.g. <li><a href="/gachiakuta-episode-1">)
			reEp := regexp.MustCompile(`href="([^"]+)"`)
			epMatches := reEp.FindAllStringSubmatch(string(ajaxBody), -1)
			
			// Gogoanime returns episodes in reverse order, reverse them to ascending
			for i := len(epMatches) - 1; i >= 0; i-- {
				epLink := epMatches[i][1]
				epLink = strings.TrimSpace(epLink)
				// Extract the episode number from slug (e.g., /gachiakuta-episode-1 -> 1)
				epNo := "1"
				if parts := strings.Split(epLink, "-episode-"); len(parts) >= 2 {
					epNo = parts[1]
				}
				episodes = append(episodes, epNo)
			}
		}
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
	// Construct the episode player page slug
	epSlug := fmt.Sprintf("%s-episode-%s", showID, episodeNo)
	epURL := fmt.Sprintf("https://gogoanime3.co/%s", epSlug)
	headers := map[string]string{
		"User-Agent": UserAgent,
	}

	body, err := doHTTPReqWithRetry("GET", epURL, nil, headers)
	if err != nil {
		return nil, err
	}

	// 1. Scrape Gogoanime Download Page Link (embed page contains a download button redirecting to player download pages)
	reDownload := regexp.MustCompile(`href="([^"]+download[^"]+)"`)
	downloadMatch := reDownload.FindStringSubmatch(string(body))
	if len(downloadMatch) < 2 {
		return nil, fmt.Errorf("unable to find Gogoanime download button on player page")
	}

	downloadURL := downloadMatch[1]
	downloadURL = strings.ReplaceAll(downloadURL, "&amp;", "&")

	// 2. Fetch the GogoPlay download page which hosts direct high quality MP4 links
	dlBody, err := doHTTPReqWithRetry("GET", downloadURL, nil, headers)
	if err != nil {
		return nil, err
	}

	// 3. Scrape direct MP4 download links from the page
	// Format is: <div class="downdown"><a href="https://..." download>Download (1080p - mp4)</a></div>
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
