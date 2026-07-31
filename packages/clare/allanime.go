package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
)

type AllAnimeProvider struct{}

func (p *AllAnimeProvider) Name() string {
	return "allanime"
}

func (p *AllAnimeProvider) Search(query string, mode string) ([]AnimeShow, error) {
	return p.searchAnime(query, mode)
}

func (p *AllAnimeProvider) FetchEpisodes(showID string, mode string) (AnimeShow, []string, error) {
	return p.fetchEpisodeList(showID, mode)
}

func (p *AllAnimeProvider) ResolveStreams(showID, mode, episodeNo, quality string) ([]ResolvedStream, error) {
	return p.fetchAllResolvedStreams(showID, mode, episodeNo)
}

func (p *AllAnimeProvider) searchAnime(query, mode string) ([]AnimeShow, error) {
	searchGQL := `query( $search: SearchInput $limit: Int $page: Int $translationType: VaildTranslationTypeEnumType $countryOrigin: VaildCountryOriginEnumType ) { shows( search: $search limit: $limit page: $page translationType: $translationType countryOrigin: $countryOrigin ) { edges { _id name availableEpisodes englishName nativeName thumbnail description malId aniListId type score season __typename } }}`

	payload := map[string]any{
		"variables": map[string]any{
			"search": map[string]any{
				"allowAdult":   false,
				"allowUnknown": false,
				"query":        query,
			},
			"limit":           40,
			"page":            1,
			"translationType": mode,
			"countryOrigin":   "ALL",
		},
		"query": searchGQL,
	}
	jsonPayload, _ := json.Marshal(payload)

	headers := map[string]string{
		"Content-Type": "application/json",
		"User-Agent":   UserAgent,
		"Referer":      AllAnimeReferer,
	}

	body, err := doHTTPReqWithRetry("POST", AllAnimeAPI, jsonPayload, headers)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data struct {
			Shows struct {
				Edges []AnimeShow `json:"edges"`
			} `json:"shows"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		snippet := string(body)
		if len(snippet) > 150 {
			snippet = snippet[:150]
		}
		return nil, fmt.Errorf("search failed to parse API JSON: %w (body: %q)", err, snippet)
	}

	var validShows []AnimeShow
	for _, s := range result.Data.Shows.Edges {
		s.Provider = "allanime"
		if s.EpCount() > 0 {
			validShows = append(validShows, s)
		}
	}

	return validShows, nil
}

func (p *AllAnimeProvider) fetchEpisodeList(showID, mode string) (AnimeShow, []string, error) {
	showID = stripProviderPrefix(showID)
	if show, eps, found := loadShowCache(showID); found {
		debugLog("fetchEpisodeList: loaded show %s from cache", showID)
		return show, eps, nil
	}

	showGQL := `query ($showId: String!) { show( _id: $showId ) { _id name englishName nativeName thumbnail description malId aniListId type score season availableEpisodes availableEpisodesDetail }}`
	payload := map[string]any{
		"variables": map[string]any{
			"showId": showID,
		},
		"query": showGQL,
	}
	jsonPayload, _ := json.Marshal(payload)

	headers := map[string]string{
		"Content-Type": "application/json",
		"User-Agent":   UserAgent,
		"Referer":      AllAnimeReferer,
	}

	body, err := doHTTPReqWithRetry("POST", AllAnimeAPI, jsonPayload, headers)
	if err != nil {
		return AnimeShow{}, nil, err
	}

	var result struct {
		Data struct {
			Show AnimeShow `json:"show"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		snippet := string(body)
		if len(snippet) > 150 {
			snippet = snippet[:150]
		}
		return AnimeShow{}, nil, fmt.Errorf("episodes failed to parse API JSON: %w (body: %q)", err, snippet)
	}

	episodes := result.Data.Show.AvailableEpisodesDetail[mode]
	if len(episodes) == 0 {
		for k, v := range result.Data.Show.AvailableEpisodesDetail {
			if strings.EqualFold(k, mode) {
				episodes = v
				break
			}
		}
	}

	_ = saveShowCache(showID, result.Data.Show, episodes)
	return result.Data.Show, episodes, nil
}

func generateAABoot(buildID, lane string, epoch int64, maskHex string) string {
	if maskHex == "" {
		maskHex = "948f4e192f9b462ec946efc1996bbc8e66c7ef768c184f4aabf512b3505c9247"
	}
	maskBytes, err := hex.DecodeString(maskHex)
	if err != nil {
		return ""
	}

	h1 := hmac.New(sha256.New, maskBytes)
	h1.Write([]byte("x-aa-boot:" + buildID))
	key2 := h1.Sum(nil)

	domain := "allanime.day"
	if u, err := url.Parse(AllAnimeReferer); err == nil && u.Host != "" {
		domain = u.Host
	}

	payload := fmt.Sprintf("%s:k7:%s:%d:%s", buildID, domain, epoch, lane)
	h2 := hmac.New(sha256.New, key2)
	h2.Write([]byte(payload))
	return hex.EncodeToString(h2.Sum(nil))
}

func (p *AllAnimeProvider) fetchEpisodeSources(showID, mode, episodeNo string) ([]SourceInfo, error) {
	showID = stripProviderPrefix(showID)
	aareq, err := generateAAReq(allAnimeQueryHash)
	if err != nil {
		return nil, fmt.Errorf("failed to generate aaReq: %w", err)
	}

	queryVars := fmt.Sprintf(`{"showId":"%s","translationType":"%s","episodeString":"%s"}`, showID, mode, episodeNo)
	queryExt := fmt.Sprintf(`{"persistedQuery":{"version":1,"sha256Hash":"%s"},"k":"k7","aaReq":"%s"}`, allAnimeQueryHash, aareq)

	reqURL := fmt.Sprintf("%s?variables=%s&extensions=%s", AllAnimeAPI, url.QueryEscape(queryVars), url.QueryEscape(queryExt))

	epoch, _, buildID, err := getDerivedKey()
	if err != nil || buildID == "" {
		buildID = "64"
	}
	aaBoot := generateAABoot(buildID, "k7", epoch, allAnimeClientMaskHex)
	debugLog("[ALLANIME] Generated aaReq length %d, buildID: %s, aaBoot: %s", len(aareq), buildID, aaBoot)

	headers := map[string]string{
		"User-Agent": UserAgent,
		"Referer":    AllAnimeReferer,
		"Origin":     allAnimeQueryOrigin,
		"x-build-id": buildID,
		"x-aa-boot":  aaBoot,
	}

	body, err := doHTTPReqWithRetry("GET", reqURL, nil, headers)
	if err != nil {
		return nil, err
	}
	debugLog("[ALLANIME] Episode sources response body snippet: %s", string(body)[:min(150, len(body))])

	if strings.Contains(string(body), "NEED_CAPTCHA") || strings.Contains(string(body), "AA_CRYPTO_STALE") || strings.Contains(string(body), "AA_CRYPTO_EXPIRED") || strings.Contains(string(body), "INTERNAL_SERVER_ERROR") {
		debugLog("[ALLANIME] Stale/captcha crypto token detected, invalidating cache and retrying...")
		invalidateDerivedKeyCache()
		epoch, _, buildID, _ = getDerivedKey()
		if buildID == "" {
			buildID = "64"
		}
		aareq, _ = generateAAReq(allAnimeQueryHash)
		aaBoot = generateAABoot(buildID, "k7", epoch, allAnimeClientMaskHex)
		queryExt = fmt.Sprintf(`{"persistedQuery":{"version":1,"sha256Hash":"%s"},"k":"k7","aaReq":"%s"}`, allAnimeQueryHash, aareq)
		reqURL = fmt.Sprintf("%s?variables=%s&extensions=%s", AllAnimeAPI, url.QueryEscape(queryVars), url.QueryEscape(queryExt))
		headers["x-build-id"] = buildID
		headers["x-aa-boot"] = aaBoot
		body, err = doHTTPReqWithRetry("GET", reqURL, nil, headers)
		if err != nil {
			return nil, err
		}
		debugLog("[ALLANIME] Retried episode sources response body snippet: %s", string(body)[:min(150, len(body))])
	}

	re := regexp.MustCompile(`"tobeparsed"\s*:\s*"([^"]*)"`)
	match := re.FindStringSubmatch(string(body))
	if len(match) >= 2 && match[1] != "" {
		return decodeToBeParsed(match[1])
	}

	var fallbackResult struct {
		Data struct {
			Episode struct {
				SourceUrls []struct {
					SourceURL  string `json:"sourceUrl"`
					SourceName string `json:"sourceName"`
				} `json:"sourceUrls"`
			} `json:"episode"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &fallbackResult); err == nil && len(fallbackResult.Data.Episode.SourceUrls) > 0 {
		var sources []SourceInfo
		for _, su := range fallbackResult.Data.Episode.SourceUrls {
			sources = append(sources, SourceInfo{
				SourceName: su.SourceName,
				SourceURL:  decodeSourceURL(su.SourceURL),
			})
		}
		return sources, nil
	}

	snippet := string(body)
	if len(snippet) > 200 {
		snippet = snippet[:200]
	}
	return nil, fmt.Errorf("no source urls found in response (body snippet: %s)", snippet)
}

func (p *AllAnimeProvider) fetchProviderLinks(sourceURL string) (map[string]string, error) {
	debugLog("[RESOLVE-CLOCK] Resolving clock payload on URL: %s", sourceURL)

	// 1. Intercept direct fast4speed redirector (no clock query needed)
	if strings.Contains(sourceURL, "fast4speed.rsvp") {
		debugLog("[RESOLVE-CLOCK] fast4speed.rsvp direct intercept matched.")
		return map[string]string{
			"best": sourceURL,
		}, nil
	}

	// 2. Intercept third-party embeds (played natively via yt-dlp)
	if strings.Contains(sourceURL, "ok.ru/videoembed/") {
		debugLog("[RESOLVE-CLOCK] Native third-party embed matched (Ok.ru): %s", sourceURL)
		return map[string]string{
			"best": sourceURL,
		}, nil
	}
	if strings.Contains(sourceURL, "filemoon") ||
		strings.Contains(sourceURL, "bysekoze") ||
		strings.Contains(sourceURL, "streamwish") ||
		strings.Contains(sourceURL, "awish") ||
		strings.Contains(sourceURL, "dwish") ||
		strings.Contains(sourceURL, "listeamed") ||
		strings.Contains(sourceURL, "vidguard") {
		debugLog("[RESOLVE-CLOCK] Unsupported third-party embed host skipped: %s", sourceURL)
		return nil, fmt.Errorf("unsupported third-party embed host: %s", sourceURL)
	}

	// 2. Intercept Mp4Upload landing page and parse the raw video source
	if strings.Contains(sourceURL, "mp4upload.com") {
		debugLog("[RESOLVE-CLOCK] Mp4Upload direct intercept matched.")
		headers := map[string]string{
			"User-Agent": UserAgent,
			"Referer":    "https://www.mp4upload.com",
		}
		body, err := doHTTPReqWithRetry("GET", sourceURL, nil, headers)
		if err != nil {
			return nil, err
		}

		re := regexp.MustCompile(`src:\s*"([^"]+)"`)
		if m := re.FindStringSubmatch(string(body)); len(m) >= 2 {
			debugLog("[RESOLVE-CLOCK] Mp4Upload stream URL resolved: %s", m[1])
			return map[string]string{
				"best": m[1],
			}, nil
		}
		return nil, fmt.Errorf("mp4upload: failed to parse source link")
	}

	// 3. Fallback: Standard AllAnime internal clock endpoint (Must use StreamReferer)
	headers := map[string]string{
		"User-Agent": UserAgent,
		"Referer":    StreamReferer, // Force https://allanimenews.com/ to unlock hidden links
	}
	body, err := doHTTPReqWithRetry("GET", sourceURL, nil, headers)
	if err != nil {
		return nil, err
	}

	debugLog("[RESOLVE-CLOCK] Received raw response body: %s", string(body))

	links := make(map[string]string)

	extractLinks := func(entries []LinkEntry) bool {
		directPlay := false
		for _, l := range entries {
			if l.Link != "" {
				quality := l.ResolutionStr
				if quality == "" && l.HLS {
					quality = "hls"
				}
				if quality == "" {
					quality = "best"
				}
				links[quality] = strings.ReplaceAll(l.Link, "\\", "")
			} else if l.Mp4 {
				directPlay = true
			}
		}
		return directPlay
	}

	var linksArr []LinkEntry
	if err := json.Unmarshal(body, &linksArr); err == nil {
		debugLog("[RESOLVE-CLOCK] Successfully unmarshaled raw array []LinkEntry")
		if extractLinks(linksArr) && len(links) == 0 {
			debugLog("[RESOLVE-CLOCK] mp4:true with no link URL — skipping broken source")
			return nil, fmt.Errorf("mp4:true response with no link URL (server returned incomplete payload)")
		}
	} else {
		debugLog("[RESOLVE-CLOCK] Array []LinkEntry unmarshal failed: %v. Retrying as ClockResponse object...", err)
		var clockResp ClockResponse
		if err := json.Unmarshal(body, &clockResp); err == nil {
			debugLog("[RESOLVE-CLOCK] Successfully unmarshaled ClockResponse object.")
			if extractLinks(clockResp.Links) && len(links) == 0 {
				debugLog("[RESOLVE-CLOCK] mp4:true with no link URL — skipping broken source")
				return nil, fmt.Errorf("mp4:true response with no link URL (server returned incomplete payload)")
			}
		} else {
			debugLog("[RESOLVE-CLOCK] ClockResponse object unmarshal failed: %v", err)
		}
	}

	if len(links) == 0 {
		debugLog("[RESOLVE-CLOCK] Zero links unmarshaled. Falling back to default regexes...")
		reVideo := regexp.MustCompile(`"link":"([^"]*)".*?"resolutionStr":"([^"]*)"`)
		for _, m := range reVideo.FindAllStringSubmatch(string(body), -1) {
			if len(m) >= 3 {
				links[m[2]] = strings.ReplaceAll(m[1], "\\", "")
			}
		}

		reHLS := regexp.MustCompile(`"hls":true.*?"link":"([^"]*)"`)
		for _, m := range reHLS.FindAllStringSubmatch(string(body), -1) {
			if len(m) >= 2 {
				links["hls"] = strings.ReplaceAll(m[1], "\\", "")
			}
		}
	}

	if len(links) == 0 {
		debugLog("[RESOLVE-CLOCK] Zero links parsed by standard regexes. Falling back to ultimate wildcard matching...")
		re := regexp.MustCompile(`"(link|url)"\s*:\s*"([^"]+)"`)
		matches := re.FindAllStringSubmatch(string(body), -1)
		for _, m := range matches {
			if len(m) >= 3 {
				urlVal := strings.ReplaceAll(m[2], "\\", "")
				if strings.Contains(urlVal, "m3u8") {
					links["hls"] = urlVal
				} else {
					links["best"] = urlVal
				}
			}
		}
	}

	debugLog("[RESOLVE-CLOCK] Final resolved maps: %+v", links)
	return links, nil
}

func (p *AllAnimeProvider) fetchAllResolvedStreams(showID, mode, episodeNo string) ([]ResolvedStream, error) {
	sources, err := p.fetchEpisodeSources(showID, mode, episodeNo)
	if err != nil {
		return nil, err
	}

	var results []ResolvedStream
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, src := range sources {
		wg.Add(1)
		go func(s SourceInfo) {
			defer wg.Done()
			links, err := p.fetchProviderLinks(s.SourceURL)
			if err == nil && len(links) > 0 {
				mu.Lock()
				for qual, urlVal := range links {
					finalURL := urlVal
					if strings.Contains(finalURL, "ok.ru/videoembed/") {
						if directURL, err := unpackOkRuEmbed(finalURL); err == nil && directURL != "" {
							finalURL = directURL
							debugLog("[ALLANIME-OKRU] Successfully unpacked ok.ru embed to direct video URL: %s", finalURL)
						} else {
							debugLog("[ALLANIME-OKRU] Failed to unpack ok.ru embed %s: %v", finalURL, err)
							continue
						}
					}

					results = append(results, ResolvedStream{
						Provider:   "allanime",
						SourceName: s.SourceName,
						Quality:    qual,
						URL:        finalURL,
					})
				}
				mu.Unlock()
			}
		}(src)
	}
	wg.Wait()

	if len(results) == 0 {
		return nil, fmt.Errorf("no streams resolved for episode %s (%s)", episodeNo, mode)
	}

	return results, nil
}

func unpackOkRuEmbed(embedURL string) (string, error) {
	req, err := http.NewRequest("GET", embedURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", UserAgent)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	htmlStr := string(body)
	reDataOpt := regexp.MustCompile(`data-options="([^"]+)"`)
	m := reDataOpt.FindStringSubmatch(htmlStr)
	if len(m) < 2 {
		return "", fmt.Errorf("ok.ru data-options not found")
	}

	cleanJSON := strings.ReplaceAll(m[1], "&quot;", "\"")
	cleanJSON = strings.ReplaceAll(cleanJSON, "&amp;", "&")

	var dataOpt map[string]interface{}
	if err := json.Unmarshal([]byte(cleanJSON), &dataOpt); err != nil {
		return "", err
	}

	flashvars, ok := dataOpt["flashvars"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("ok.ru flashvars not found")
	}

	metaStr, ok := flashvars["metadata"].(string)
	if !ok {
		return "", fmt.Errorf("ok.ru metadata not found")
	}

	var meta map[string]interface{}
	if err := json.Unmarshal([]byte(metaStr), &meta); err != nil {
		return "", err
	}

	if hls, ok := meta["hlsManifestUrl"].(string); ok && hls != "" {
		return hls, nil
	}

	if videos, ok := meta["videos"].([]interface{}); ok {
		for _, v := range videos {
			if vMap, ok := v.(map[string]interface{}); ok {
				if u, ok := vMap["url"].(string); ok && u != "" {
					return u, nil
				}
			}
		}
	}

	return "", fmt.Errorf("no valid video stream URL found in ok.ru metadata")
}
