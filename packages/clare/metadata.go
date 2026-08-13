package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// FetchEpisodeMetadataCascade attempts to fetch episode metadata across Jikan -> AniList -> Kitsu
func FetchEpisodeMetadataCascade(malID, showName string, page int) (map[string]JikanEpInfo, error) {
	metadata := make(map[string]JikanEpInfo)

	// Tier 1: Jikan (MAL API)
	if malID != "" && malID != "0" {
		client := newLoggingHttpClient(4 * time.Second)
		reqURL := fmt.Sprintf("https://api.jikan.moe/v4/anime/%s/episodes?page=%d", malID, page)
		req, err := http.NewRequest("GET", reqURL, nil)
		if err == nil {
			req.Header.Set("User-Agent", UserAgent)
			resp, err := client.Do(req)
			if err == nil {
				defer resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					var res struct {
						Data []struct {
							MalID    int    `json:"mal_id"`
							Title    string `json:"title"`
							Aired    string `json:"aired"`
							Synopsis string `json:"synopsis"`
							Filler   bool   `json:"filler"`
							Recap    bool   `json:"recap"`
						} `json:"data"`
					}
					body, _ := io.ReadAll(resp.Body)
					if json.Unmarshal(body, &res) == nil && len(res.Data) > 0 {
						for _, d := range res.Data {
							epNum := fmt.Sprintf("%d", d.MalID)
							airedStr := d.Aired
							if len(airedStr) >= 10 {
								airedStr = airedStr[:10]
							}
							metadata[epNum] = JikanEpInfo{
								Title:    d.Title,
								Aired:    airedStr,
								Synopsis: d.Synopsis,
								Filler:   d.Filler,
								Recap:    d.Recap,
							}
						}
						return metadata, nil
					}
				}
			}
		}
	}

	// Tier 2: AniList GraphQL
	if showName != "" {
		alMeta, err := fetchAniListEpisodeMetadata(showName)
		if err == nil && len(alMeta) > 0 {
			debugLog("[INFO] Successfully fetched %d episode titles from AniList fallback for %s", len(alMeta), showName)
			return alMeta, nil
		}

		// Tier 3: Kitsu API
		kitsuMeta, err := fetchKitsuEpisodeMetadata(showName)
		if err == nil && len(kitsuMeta) > 0 {
			debugLog("[INFO] Successfully fetched %d episode titles from Kitsu fallback for %s", len(kitsuMeta), showName)
			return kitsuMeta, nil
		}
	}

	return nil, fmt.Errorf("all episode metadata fallbacks failed for malID=%s show=%s", malID, showName)
}

func fetchAniListEpisodeMetadata(showName string) (map[string]JikanEpInfo, error) {
	cleanTitle := showName
	if idx := strings.Index(cleanTitle, "("); idx > 0 {
		cleanTitle = strings.TrimSpace(cleanTitle[:idx])
	}
	query := `query ($search: String) {
		Media(search: $search, type: ANIME) {
			streamingEpisodes {
				title
			}
		}
	}`
	payload := map[string]any{
		"query": query,
		"variables": map[string]any{
			"search": cleanTitle,
		},
	}
	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	headers := map[string]string{
		"Content-Type": "application/json",
		"User-Agent":   UserAgent,
	}

	respBody, err := doHTTPReqWithRetry("POST", "https://graphql.anilist.co", jsonBody, headers)
	if err != nil {
		return nil, err
	}

	var alResp struct {
		Data struct {
			Media struct {
				StreamingEpisodes []struct {
					Title string `json:"title"`
				} `json:"streamingEpisodes"`
			} `json:"Media"`
		} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &alResp); err != nil {
		return nil, err
	}

	eps := alResp.Data.Media.StreamingEpisodes
	if len(eps) == 0 {
		return nil, fmt.Errorf("no streaming episodes found on AniList for %s", cleanTitle)
	}

	metadata := make(map[string]JikanEpInfo)
	for i, ep := range eps {
		title := strings.TrimSpace(ep.Title)
		epNum := fmt.Sprintf("%d", i+1)
		if idx := strings.Index(title, " - "); idx > 0 {
			title = strings.TrimSpace(title[idx+3:])
		}
		if title != "" {
			metadata[epNum] = JikanEpInfo{
				Title: title,
			}
		}
	}

	if len(metadata) == 0 {
		return nil, fmt.Errorf("no valid episode titles extracted from AniList")
	}
	return metadata, nil
}

func fetchKitsuEpisodeMetadata(showName string) (map[string]JikanEpInfo, error) {
	cleanTitle := showName
	if idx := strings.Index(cleanTitle, "("); idx > 0 {
		cleanTitle = strings.TrimSpace(cleanTitle[:idx])
	}
	searchURL := fmt.Sprintf("https://kitsu.io/api/edge/anime?filter[text]=%s", url.QueryEscape(cleanTitle))
	body, err := doHTTPReqWithRetry("GET", searchURL, nil, map[string]string{"User-Agent": UserAgent})
	if err != nil {
		return nil, err
	}
	var searchRes struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &searchRes); err != nil || len(searchRes.Data) == 0 {
		return nil, fmt.Errorf("no kitsu anime found for %s", cleanTitle)
	}

	kitsuID := searchRes.Data[0].ID
	episodesURL := fmt.Sprintf("https://kitsu.io/api/edge/anime/%s/episodes?page[limit]=20", kitsuID)
	epBody, err := doHTTPReqWithRetry("GET", episodesURL, nil, map[string]string{"User-Agent": UserAgent})
	if err != nil {
		return nil, err
	}

	var epRes struct {
		Data []struct {
			Attributes struct {
				Number         int    `json:"number"`
				CanonicalTitle string `json:"canonicalTitle"`
				Synopsis       string `json:"synopsis"`
				Airdate        string `json:"airdate"`
				Titles         struct {
					En string `json:"en"`
				} `json:"titles"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.Unmarshal(epBody, &epRes); err != nil {
		return nil, err
	}

	metadata := make(map[string]JikanEpInfo)
	for _, ep := range epRes.Data {
		title := ep.Attributes.Titles.En
		if title == "" {
			title = ep.Attributes.CanonicalTitle
		}
		if title == "" {
			continue
		}
		epNum := fmt.Sprintf("%d", ep.Attributes.Number)
		metadata[epNum] = JikanEpInfo{
			Title:    title,
			Synopsis: ep.Attributes.Synopsis,
			Aired:    ep.Attributes.Airdate,
		}
	}
	if len(metadata) == 0 {
		return nil, fmt.Errorf("no kitsu episode titles found")
	}
	return metadata, nil
}
