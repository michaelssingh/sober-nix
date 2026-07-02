package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	AllAnimeReferer    = "https://allmanga.to"
	AllAnimeBase       = "allanime.day"
	AllAnimeAPI        = "https://api.allanime.day/api"
	UserAgent          = "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/121.0"
	allAnimeKeyPhrase  = "Xot36i3lK3:v1"
	allAnimeQueryHash  = "d405d0edd690624b66baba3068e0edc3ac90f1597d898a1ec8db4e5c43c00fec"
	allAnimeQueryOrigin = "https://youtu-chan.com"
)

var (
	allAnimeKey = func() []byte {
		h := sha256.Sum256([]byte(allAnimeKeyPhrase))
		return h[:]
	}()

	hexSubstitutionTable = map[string]string{
		"79": "A", "7a": "B", "7b": "C", "7c": "D", "7d": "E", "7e": "F", "7f": "G",
		"70": "H", "71": "I", "72": "J", "73": "K", "74": "L", "75": "M", "76": "N", "77": "O",
		"68": "P", "69": "Q", "6a": "R", "6b": "S", "6c": "T", "6d": "U", "6e": "V", "6f": "W",
		"60": "X", "61": "Y", "62": "Z",
		"59": "a", "5a": "b", "5b": "c", "5c": "d", "5d": "e", "5e": "f", "5f": "g",
		"50": "h", "51": "i", "52": "j", "53": "k", "54": "l", "55": "m", "56": "n", "57": "o",
		"48": "p", "49": "q", "4a": "r", "4b": "s", "4c": "t", "4d": "u", "4e": "v", "4f": "w",
		"40": "x", "41": "y", "42": "z",
		"08": "0", "09": "1", "0a": "2", "0b": "3", "0c": "4", "0d": "5", "0e": "6", "0f": "7",
		"00": "8", "01": "9",
		"15": "-", "16": ".", "67": "_", "46": "~",
		"02": ":", "17": "/", "07": "?", "1b": "#",
		"63": "[", "65": "]", "78": "@",
		"19": "!", "1c": "$", "1e": "&",
		"10": "(", "11": ")", "12": "*", "13": "+", "14": ",",
		"03": ";", "05": "=", "1d": "%",
	}

	linkPriorities = []string{"wixmp", "sharepoint", "mpv", "youtube"}
)

type AnimeShow struct {
	ID                string `json:"_id"`
	Name              string `json:"name"`
	AvailableEpisodes any    `json:"availableEpisodes"`
	EnglishName       string `json:"englishName"`
	NativeName        string `json:"nativeName"`
	Thumbnail         string `json:"thumbnail"`
	Description       string `json:"description"`
	MALID             string `json:"malId"`
	AniListID         string `json:"aniListId"`
	Type              string `json:"type"`
	Score             float64 `json:"score"`
	Season            struct {
		Quarter string `json:"quarter"`
		Year    int    `json:"year"`
	} `json:"season"`
	Duration          string `json:"duration"`
}

func (s AnimeShow) EpCount() int {
	if s.AvailableEpisodes == nil {
		return 0
	}
	if episodes, ok := s.AvailableEpisodes.(map[string]any); ok {
		if subEpisodes, ok := episodes["sub"].(float64); ok {
			return int(subEpisodes)
		}
	}
	return 0
}

func (s AnimeShow) SubCount() int {
	if s.AvailableEpisodes == nil {
		return 0
	}
	if episodes, ok := s.AvailableEpisodes.(map[string]any); ok {
		if subEpisodes, ok := episodes["sub"].(float64); ok {
			return int(subEpisodes)
		}
	}
	return 0
}

func (s AnimeShow) DubCount() int {
	if s.AvailableEpisodes == nil {
		return 0
	}
	if episodes, ok := s.AvailableEpisodes.(map[string]any); ok {
		if dubEpisodes, ok := episodes["dub"].(float64); ok {
			return int(dubEpisodes)
		}
	}
	return 0
}

type SourceInfo struct {
	SourceName string
	SourceURL  string
}

type LinkEntry struct {
	Link          string `json:"link"`
	ResolutionStr string `json:"resolutionStr"`
	HLS           bool   `json:"hls"`
}

type ClockResponse struct {
	Links []LinkEntry `json:"links"`
}

func decodeSourceURL(encoded string) string {
	encoded = strings.TrimPrefix(encoded, "--")
	var result strings.Builder
	result.Grow(len(encoded))
	for i := 0; i+1 < len(encoded); i += 2 {
		pair := encoded[i : i+2]
		if val, exists := hexSubstitutionTable[pair]; exists {
			result.WriteString(val)
		} else {
			result.WriteString(pair)
		}
	}
	decoded := result.String()
	decoded = strings.ReplaceAll(decoded, "/clock", "/clock.json")
	if strings.HasPrefix(decoded, "/") {
		decoded = fmt.Sprintf("https://%s%s", AllAnimeBase, decoded)
	}
	return decoded
}

func decodeToBeParsed(blob string) ([]SourceInfo, error) {
	data, err := base64.StdEncoding.DecodeString(blob)
	if err != nil {
		data, err = base64.URLEncoding.DecodeString(blob)
		if err != nil {
			data, err = base64.RawURLEncoding.DecodeString(blob)
			if err != nil {
				return nil, fmt.Errorf("base64 decode failed: %w", err)
			}
		}
	}

	if len(data) < 30 {
		return nil, fmt.Errorf("tobeparsed blob too short (%d bytes)", len(data))
	}

	nonce := data[1:13]
	ciphertext := data[13 : len(data)-16]

	block, err := aes.NewCipher(allAnimeKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	iv := make([]byte, 16)
	copy(iv[:12], nonce)
	iv[15] = 0x02

	stream := cipher.NewCTR(block, iv)
	plaintext := make([]byte, len(ciphertext))
	stream.XORKeyStream(plaintext, ciphertext)

	var result struct {
		Data struct {
			Episode struct {
				SourceUrls []struct {
					SourceURL  string `json:"sourceUrl"`
					SourceName string `json:"sourceName"`
				} `json:"sourceUrls"`
			} `json:"episode"`
		} `json:"data"`
	}

	if err := json.Unmarshal(plaintext, &result); err == nil && len(result.Data.Episode.SourceUrls) > 0 {
		var sources []SourceInfo
		for _, su := range result.Data.Episode.SourceUrls {
			sources = append(sources, SourceInfo{
				SourceName: su.SourceName,
				SourceURL:  decodeSourceURL(su.SourceURL),
			})
		}
		return sources, nil
	}

	re1 := regexp.MustCompile(`"sourceUrl"\s*:\s*"--([^"]*)"[^}]*"sourceName"\s*:\s*"([^"]*)"`)
	matches1 := re1.FindAllSubmatch(plaintext, -1)
	if len(matches1) > 0 {
		var sources []SourceInfo
		for _, m := range matches1 {
			sources = append(sources, SourceInfo{
				SourceName: string(m[2]),
				SourceURL:  decodeSourceURL(string(m[1])),
			})
		}
		return sources, nil
	}

	re2 := regexp.MustCompile(`"sourceName"\s*:\s*"([^"]*)"[^}]*"sourceUrl"\s*:\s*"--([^"]*)"`)
	matches2 := re2.FindAllSubmatch(plaintext, -1)
	if len(matches2) > 0 {
		var sources []SourceInfo
		for _, m := range matches2 {
			sources = append(sources, SourceInfo{
				SourceName: string(m[1]),
				SourceURL:  decodeSourceURL(string(m[2])),
			})
		}
		return sources, nil
	}

	return nil, fmt.Errorf("no source URLs decoded from tobeparsed plaintext")
}

func fetchEpisodeSources(showID, mode, episodeNo string) ([]SourceInfo, error) {
	queryVars := fmt.Sprintf(`{"showId":"%s","translationType":"%s","episodeString":"%s"}`, showID, mode, episodeNo)
	queryExt := fmt.Sprintf(`{"persistedQuery":{"version":1,"sha256Hash":"%s"}}`, allAnimeQueryHash)
	reqURL := fmt.Sprintf("%s?variables=%s&extensions=%s", AllAnimeAPI, url.QueryEscape(queryVars), url.QueryEscape(queryExt))

	headers := map[string]string{
		"User-Agent": UserAgent,
		"Referer":    AllAnimeReferer,
		"Origin":     allAnimeQueryOrigin,
	}

	body, err := doHTTPReqWithRetry("GET", reqURL, nil, headers)
	if err != nil || len(body) == 0 || !strings.Contains(string(body), "tobeparsed") {
		episodeEmbedGQL := `query ($showId: String!, $translationType: VaildTranslationTypeEnumType!, $episodeString: String!) { episode( showId: $showId translationType: $translationType episodeString: $episodeString ) { episodeString sourceUrls }}`
		payload := map[string]any{
			"variables": map[string]any{
				"showId":          showID,
				"translationType": mode,
				"episodeString":   episodeNo,
			},
			"query": episodeEmbedGQL,
		}
		jsonPayload, _ := json.Marshal(payload)

		postHeaders := map[string]string{
			"Content-Type": "application/json",
			"User-Agent":   UserAgent,
			"Referer":      AllAnimeReferer,
		}
		body, err = doHTTPReqWithRetry("POST", AllAnimeAPI, jsonPayload, postHeaders)
		if err != nil {
			return nil, err
		}
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

	return nil, fmt.Errorf("no source urls found in response")
}

func fetchProviderLinks(sourceURL string) (map[string]string, error) {
	headers := map[string]string{
		"User-Agent": UserAgent,
		"Referer":    AllAnimeReferer,
	}
	body, err := doHTTPReqWithRetry("GET", sourceURL, nil, headers)
	if err != nil {
		return nil, err
	}

	links := make(map[string]string)
	var clockResp ClockResponse
	if err := json.Unmarshal(body, &clockResp); err == nil {
		for _, l := range clockResp.Links {
			quality := l.ResolutionStr
			if quality == "" && l.HLS {
				quality = "hls"
			}
			if quality != "" {
				links[quality] = strings.ReplaceAll(l.Link, "\\", "")
			}
		}
	}

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

	return links, nil
}

func searchAnime(query, mode string) ([]AnimeShow, error) {
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

	return result.Data.Shows.Edges, nil
}

func fetchEpisodeList(showID, mode string) (AnimeShow, []string, error) {
	if show, eps, found := loadShowCache(showID); found {
		debugLog("fetchEpisodeList: loaded show %s from cache", showID)
		return show, eps, nil
	}

	showGQL := `query ($showId: String!) { show( _id: $showId ) { _id name englishName nativeName thumbnail description malId aniListId type score season availableEpisodesDetail }}`
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
			Show struct {
				AnimeShow
				AvailableEpisodesDetail map[string][]string `json:"availableEpisodesDetail"`
			} `json:"show"`
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

	_ = saveShowCache(showID, result.Data.Show.AnimeShow, episodes)
	return result.Data.Show.AnimeShow, episodes, nil
}

var (
	streamCache        = make(map[string]string)
	streamCacheMu      sync.RWMutex
	activePrefetches   = make(map[string]bool)
	activePrefetchesMu sync.Mutex
)

func prefetchEpisodeStream(showID, mode, epNo, quality string) {
	if showID == "" || epNo == "" {
		return
	}
	cacheKey := fmt.Sprintf("%s-%s-%s-%s", showID, mode, epNo, quality)

	streamCacheMu.RLock()
	_, cached := streamCache[cacheKey]
	streamCacheMu.RUnlock()
	if cached {
		return
	}

	activePrefetchesMu.Lock()
	if activePrefetches[cacheKey] {
		activePrefetchesMu.Unlock()
		return
	}
	activePrefetches[cacheKey] = true
	activePrefetchesMu.Unlock()

	go func() {
		defer func() {
			activePrefetchesMu.Lock()
			delete(activePrefetches, cacheKey)
			activePrefetchesMu.Unlock()
		}()

		debugLog("prefetchEpisodeStream: starting background prefetch for %s", cacheKey)
		if mode == "dual" {
			_, _ = resolveStreamURL(showID, "sub", epNo, quality)
			_, _ = resolveStreamURL(showID, "dub", epNo, quality)
		} else {
			_, _ = resolveStreamURL(showID, mode, epNo, quality)
		}
	}()
}

func resolveStreamURL(showID, mode, episodeNo, quality string) (string, error) {
	cacheKey := fmt.Sprintf("%s-%s-%s-%s", showID, mode, episodeNo, quality)
	streamCacheMu.RLock()
	if val, ok := streamCache[cacheKey]; ok {
		streamCacheMu.RUnlock()
		debugLog("resolveStreamURL: cache hit for %s", cacheKey)
		return val, nil
	}
	streamCacheMu.RUnlock()

	sources, err := fetchEpisodeSources(showID, mode, episodeNo)
	if err != nil {
		return "", err
	}

	for _, prioDomain := range linkPriorities {
		for _, src := range sources {
			if strings.Contains(strings.ToLower(src.SourceName), prioDomain) {
				links, err := fetchProviderLinks(src.SourceURL)
				if err == nil {
					best := selectBestLink(links, quality)
					if best != "" {
						streamCacheMu.Lock()
						streamCache[cacheKey] = best
						streamCacheMu.Unlock()
						return best, nil
					}
				}
			}
		}
	}

	for _, src := range sources {
		links, err := fetchProviderLinks(src.SourceURL)
		if err == nil {
			best := selectBestLink(links, quality)
			if best != "" {
				streamCacheMu.Lock()
				streamCache[cacheKey] = best
				streamCacheMu.Unlock()
				return best, nil
			}
		}
	}

	return "", fmt.Errorf("no stream links resolved for episode %s (%s)", episodeNo, mode)
}

func selectBestLink(links map[string]string, requested string) string {
	if len(links) == 0 {
		return ""
	}

	priorities := []string{"1080p", "720p", "480p", "360p", "hls"}
	if requested == "worst" {
		priorities = []string{"360p", "480p", "720p", "1080p", "hls"}
	} else if requested != "best" && requested != "" {
		if val, exists := links[requested]; exists {
			return val
		}
	}

	for _, p := range priorities {
		if val, exists := links[p]; exists {
			return val
		}
	}

	for _, val := range links {
		return val
	}
	return ""
}

func doHTTPReqWithRetry(method, url string, payload []byte, headers map[string]string) ([]byte, error) {
	var body []byte
	var err error
	client := newLoggingHttpClient(10 * time.Second)

	for attempt := 1; attempt <= 3; attempt++ {
		var req *http.Request
		if len(payload) > 0 {
			req, err = http.NewRequest(method, url, bytes.NewReader(payload))
		} else {
			req, err = http.NewRequest(method, url, nil)
		}
		if err != nil {
			return nil, err
		}

		for k, v := range headers {
			req.Header.Set(k, v)
		}

		var resp *http.Response
		resp, err = client.Do(req)
		if err == nil {
			defer resp.Body.Close()
			body, err = io.ReadAll(resp.Body)
			if err == nil {
				isTransientError := resp.StatusCode == 502 || resp.StatusCode == 503 || resp.StatusCode == 504 || resp.StatusCode == 429
				bodyStr := string(body)
				if !isTransientError && (strings.Contains(bodyStr, "error code: 502") || strings.Contains(bodyStr, "error code: 503")) {
					isTransientError = true
				}

				if !isTransientError {
					return body, nil
				}
				err = fmt.Errorf("transient HTTP error (status %d): %s", resp.StatusCode, strings.TrimSpace(bodyStr))
			}
		}

		debugLog("HTTP request attempt %d failed for %s: %v", attempt, url, err)
		if attempt < 3 {
			time.Sleep(time.Duration(attempt) * time.Second)
		}
	}
	return nil, fmt.Errorf("after 3 attempts, request failed: %w", err)
}

type loggingRoundTripper struct {
	next http.RoundTripper
}

func (l *loggingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	debugLog("HTTP Request: %s %s", req.Method, req.URL.String())
	resp, err := l.next.RoundTrip(req)
	if err != nil {
		debugLog("HTTP Error: %s %s -> %v", req.Method, req.URL.String(), err)
		return nil, err
	}
	debugLog("HTTP Response: %s %s -> Status %d (Length: %d)", req.Method, req.URL.String(), resp.StatusCode, resp.ContentLength)
	return resp, nil
}

func newLoggingHttpClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &loggingRoundTripper{
			next: http.DefaultTransport,
		},
	}
}
