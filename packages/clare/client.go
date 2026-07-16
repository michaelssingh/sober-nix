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
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	AllAnimeReferer     = "https://youtu-chan.com"
	StreamReferer       = "https://allanimenews.com/" // Required for Wixmp/HLS CDN and Clock Handshakes
	AllAnimeBase        = "allanime.day"
	AllAnimeAPI         = "https://api.allanime.day/api"
	UserAgent           = "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/121.0"
	allAnimeKeyPhrase   = "Xot36i3lK3:v1"
	allAnimeQueryHash   = "d405d0edd690624b66baba3068e0edc3ac90f1597d898a1ec8db4e5c43c00fec"
	allAnimeQueryOrigin = "https://youtu-chan.com"
)

var (
	allAnimeKey = func() []byte {
		return []byte{
			0x22, 0x19, 0x6f, 0xa6, 0xaf, 0xca, 0x95, 0x30, 0x9f, 0xda, 0xbe, 0x9a, 0x35, 0x34, 0xb8, 0x7c,
			0xd2, 0x45, 0x4e, 0x50, 0xef, 0xea, 0xbf, 0xcb, 0xdb, 0xdf, 0xd3, 0xde, 0x67, 0x8b, 0x39, 0x82,
		}
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
	linkPriorities = []string{"wixmp", "default", "mp4", "s-mp4", "luf-mp4", "sharepoint", "mpv", "youtube", "yt-mp4", "ok"}
	// linkPriorities = []string{"wixmp", "sharepoint", "mpv", "youtube"}
)

type AnimeShow struct {
	ID                      string              `json:"_id"`
	Name                    string              `json:"name"`
	AvailableEpisodes       any                 `json:"availableEpisodes"`
	AvailableEpisodesDetail map[string][]string `json:"availableEpisodesDetail"`
	EnglishName             string              `json:"englishName"`
	NativeName              string              `json:"nativeName"`
	Thumbnail               string              `json:"thumbnail"`
	Description             string              `json:"description"`
	MALID                   string              `json:"malId"`
	AniListID               string              `json:"aniListId"`
	Type                    string              `json:"type"`
	Score                   float64             `json:"score"`
	Season                  struct {
		Quarter string  `json:"quarter"`
		Year    FlexInt `json:"year"`
	} `json:"season"`
	Duration string `json:"duration"`
}

func (s AnimeShow) EpCount() int {
	if s.AvailableEpisodesDetail != nil {
		if eps, ok := s.AvailableEpisodesDetail["sub"]; ok {
			return len(eps)
		}
	}
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
	if s.AvailableEpisodesDetail != nil {
		if eps, ok := s.AvailableEpisodesDetail["sub"]; ok {
			return len(eps)
		}
	}
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
	if s.AvailableEpisodesDetail != nil {
		if eps, ok := s.AvailableEpisodesDetail["dub"]; ok {
			return len(eps)
		}
	}
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

func (s AnimeShow) HasSub(ep string) bool {
	if s.AvailableEpisodesDetail != nil {
		if eps, ok := s.AvailableEpisodesDetail["sub"]; ok {
			for _, e := range eps {
				if e == ep {
					return true
				}
			}
			return false
		}
	}
	epVal := parseEpisodeNumber(ep)
	subCount := s.SubCount()
	return subCount > 0 && epVal <= float64(subCount)
}

func (s AnimeShow) HasDub(ep string) bool {
	if s.AvailableEpisodesDetail != nil {
		if eps, ok := s.AvailableEpisodesDetail["dub"]; ok {
			for _, e := range eps {
				if e == ep {
					return true
				}
			}
			return false
		}
	}
	epVal := parseEpisodeNumber(ep)
	dubCount := s.DubCount()
	return dubCount > 0 && epVal <= float64(dubCount)
}

type FlexInt int

func (fi *FlexInt) UnmarshalJSON(b []byte) error {
	if len(b) == 0 {
		return nil
	}
	var i int
	if err := json.Unmarshal(b, &i); err == nil {
		*fi = FlexInt(i)
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		var val int
		if _, err := fmt.Sscanf(s, "%d", &val); err == nil {
			*fi = FlexInt(val)
		}
		return nil
	}
	return nil
}

type SourceInfo struct {
	SourceName string
	SourceURL  string
}

type LinkEntry struct {
	Link          string `json:"link"`
	ResolutionStr string `json:"resolutionStr"`
	HLS           bool   `json:"hls"`
	Mp4           bool   `json:"mp4"`
}

type ClockResponse struct {
	Links []LinkEntry `json:"links"`
}

func decodeSourceURL(encoded string) string {
	if strings.HasPrefix(encoded, "http://") || strings.HasPrefix(encoded, "https://") {
		return encoded
	}

	// Strip legacy "--" prefix if present
	stripped := strings.TrimPrefix(encoded, "--")

	var result strings.Builder
	result.Grow(len(stripped))
	for i := 0; i+1 < len(stripped); i += 2 {
		pair := stripped[i : i+2]
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
	// If the decoded result looks like a valid URL, use it; otherwise return original
	if strings.HasPrefix(decoded, "http://") || strings.HasPrefix(decoded, "https://") || strings.HasPrefix(decoded, "/") {
		return decoded
	}
	return encoded
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

	// --- Try AES-256-CTR first (Current AllAnime mode) ---
	plaintext, errCTR := decryptCTR(data, nonce)
	if errCTR == nil {
		debugLog("[DECRYPT-CTR] Succeeded. Plaintext: %s", string(plaintext))
		if sources, parseErr := parseSourcePlaintext(plaintext); parseErr == nil {
			return sources, nil
		} else {
			debugLog("[DECRYPT-CTR] Failed parsing source mapping: %v", parseErr)
		}
	} else {
		debugLog("[DECRYPT-CTR] Decryption failed: %v", errCTR)
	}

	// --- Fallback: Try AES-256-GCM (Alternative mode) ---
	plaintext, errGCM := decryptGCM(data, nonce)
	if errGCM == nil {
		debugLog("[DECRYPT-GCM] Succeeded. Plaintext: %s", string(plaintext))
		if sources, parseErr := parseSourcePlaintext(plaintext); parseErr == nil {
			return sources, nil
		} else {
			debugLog("[DECRYPT-GCM] Failed parsing source mapping: %v", parseErr)
		}
	} else {
		debugLog("[DECRYPT-GCM] Decryption failed: %v", errGCM)
	}

	return nil, fmt.Errorf("decryption failed under both CTR and GCM modes")
}

func decryptCTR(data, nonce []byte) ([]byte, error) {
	ciphertext := data[13 : len(data)-16]
	ctrKey := sha256.Sum256([]byte(allAnimeKeyPhrase))

	block, err := aes.NewCipher(ctrKey[:])
	if err != nil {
		return nil, err
	}

	iv := make([]byte, 16)
	copy(iv[:12], nonce)
	iv[15] = 0x02

	stream := cipher.NewCTR(block, iv)
	plaintext := make([]byte, len(ciphertext))
	stream.XORKeyStream(plaintext, ciphertext)
	return plaintext, nil
}

func decryptGCM(data, nonce []byte) ([]byte, error) {
	block, err := aes.NewCipher(allAnimeKey)
	if err != nil {
		return nil, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	ciphertextWithTag := data[13:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertextWithTag, nil)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}

func parseSourcePlaintext(plaintext []byte) ([]SourceInfo, error) {
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

	// Also try top-level {"episode":{...}} structure (CTR decryption path)
	var topLevel struct {
		Episode struct {
			SourceUrls []struct {
				SourceURL  string `json:"sourceUrl"`
				SourceName string `json:"sourceName"`
			} `json:"sourceUrls"`
		} `json:"episode"`
	}
	if err := json.Unmarshal(plaintext, &topLevel); err == nil && len(topLevel.Episode.SourceUrls) > 0 {
		var sources []SourceInfo
		for _, su := range topLevel.Episode.SourceUrls {
			sources = append(sources, SourceInfo{
				SourceName: su.SourceName,
				SourceURL:  decodeSourceURL(su.SourceURL),
			})
		}
		debugLog("[PARSE-SOURCES] Parsed %d sources from top-level episode structure", len(sources))
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

func generateAAReq(qh string) (string, error) {
	epoch := 4128
	buildID := "11"
	ts := (time.Now().Unix() / 300) * 300 * 1000

	payload := fmt.Sprintf(`{"v":1,"ts":%d,"epoch":%d,"buildId":"%s","qh":"%s"}`, ts, epoch, buildID, qh)

	ivSeed := fmt.Sprintf("%d:%s:%s:%d", epoch, buildID, qh, ts)
	hash := sha256.Sum256([]byte(ivSeed))
	iv := hash[:12]

	block, err := aes.NewCipher(allAnimeKey)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	ciphertextWithTag := aesGCM.Seal(nil, iv, []byte(payload), nil)

	result := make([]byte, 1+len(iv)+len(ciphertextWithTag))
	result[0] = 1
	copy(result[1:13], iv)
	copy(result[13:], ciphertextWithTag)

	return base64.StdEncoding.EncodeToString(result), nil
}

func fetchEpisodeSources(showID, mode, episodeNo string) ([]SourceInfo, error) {
	queryVars := fmt.Sprintf(`{"showId":"%s","translationType":"%s","episodeString":"%s"}`, showID, mode, episodeNo)

	aareq, err := generateAAReq(allAnimeQueryHash)
	if err != nil {
		return nil, fmt.Errorf("failed to generate aaReq: %w", err)
	}

	queryExt := fmt.Sprintf(`{"persistedQuery":{"version":1,"sha256Hash":"%s"},"aaReq":"%s"}`, allAnimeQueryHash, aareq)
	reqURL := fmt.Sprintf("%s?variables=%s&extensions=%s", AllAnimeAPI, url.QueryEscape(queryVars), url.QueryEscape(queryExt))

	headers := map[string]string{
		"User-Agent": UserAgent,
		"Referer":    AllAnimeReferer,
		"Origin":     allAnimeQueryOrigin,
		"x-build-id": "11",
	}

	body, err := doHTTPReqWithRetry("GET", reqURL, nil, headers)
	if err != nil {
		return nil, err
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
	debugLog("[RESOLVE-CLOCK] Resolving clock payload on URL: %s", sourceURL)

	// 1. Intercept direct fast4speed redirector (no clock query needed)
	if strings.Contains(sourceURL, "fast4speed.rsvp") {
		debugLog("[RESOLVE-CLOCK] fast4speed.rsvp direct intercept matched.")
		return map[string]string{
			"best": sourceURL,
		}, nil
	}

	// 2. Intercept ok.ru embed (played natively via yt-dlp)
	if strings.Contains(sourceURL, "ok.ru/videoembed/") {
		debugLog("[RESOLVE-CLOCK] ok.ru direct intercept matched.")
		return map[string]string{
			"best": sourceURL,
		}, nil
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

	// extractLinks processes a slice of LinkEntry and populates the links map.
	// If an entry has mp4:true but no link URL, it signals a direct-play source;
	// return a special sentinel so the caller can use sourceURL directly.
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
				// mp4:true with no link = API signals use the sourceURL directly
				directPlay = true
			}
		}
		return directPlay
	}

	// Try unmarshaling as a raw JSON array of LinkEntry (Wixmp/HLS format)
	var linksArr []LinkEntry
	if err := json.Unmarshal(body, &linksArr); err == nil {
		debugLog("[RESOLVE-CLOCK] Successfully unmarshaled raw array []LinkEntry")
		if extractLinks(linksArr) && len(links) == 0 {
			debugLog("[RESOLVE-CLOCK] mp4:true with no link URL — skipping broken source")
			return nil, fmt.Errorf("mp4:true response with no link URL (server returned incomplete payload)")
		}
	} else {
		debugLog("[RESOLVE-CLOCK] Array []LinkEntry unmarshal failed: %v. Retrying as ClockResponse object...", err)
		// Fall back to the ClockResponse object format
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

	// Apply robust regex fallbacks if JSON unmarshaling fails to map links
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

	// Ultimate wildcard matcher in case key ordering is reversed or different
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
			nameLower := strings.ToLower(src.SourceName)
			urlLower := strings.ToLower(src.SourceURL)
			matched := false
			if prioDomain == "mp4" {
				matched = (nameLower == "mp4")
			} else {
				matched = strings.Contains(nameLower, prioDomain) || strings.Contains(urlLower, prioDomain)
			}
			if matched {
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

	cookieVal := os.Getenv("CLARE_COOKIE")
	if cookieVal == "" {
		cookieVal = os.Getenv("ALLANIME_COOKIE")
	}

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

		// Inside client.go -> doHTTPReqWithRetry()
		if strings.Contains(url, "allanime") || strings.Contains(url, "allmanga") || strings.Contains(url, "mkissa") || strings.Contains(url, "youtube-anime") || strings.Contains(url, "allanimenews") {
			// CRITICAL: x-build-id, Origin, and Cookie are ONLY sent to the GraphQL API endpoint (/api).
			// We check req.URL.Path strictly to prevent matching "/apivtwo/clock.json" which contains the substring "api".
			if req.URL.Path == "/api" || req.URL.Path == "/api/" {
				req.Header.Set("x-build-id", "11")
				if req.Header.Get("Origin") == "" {
					req.Header.Set("Origin", "https://youtu-chan.com")
				}
				if cookieVal != "" {
					req.Header.Set("Cookie", cookieVal)
				}
			}

			if req.Header.Get("Referer") == "" {
				req.Header.Set("Referer", "https://youtu-chan.com")
			}
		}

		// Log detailed request fingerprinting
		var headerLogs []string
		for k, v := range req.Header {
			headerLogs = append(headerLogs, fmt.Sprintf("%s: %s", k, strings.Join(v, ",")))
		}
		debugLog("[API-REQ] Attempt %d: %s %s | Headers: %s", attempt, method, url, strings.Join(headerLogs, "; "))

		var resp *http.Response
		resp, err = client.Do(req)
		if err == nil {
			defer resp.Body.Close()
			body, err = io.ReadAll(resp.Body)
			if err == nil {
				// Verbose error body capture
				if resp.StatusCode != http.StatusOK {
					bodySnippet := string(body)
					if len(bodySnippet) > 300 {
						bodySnippet = bodySnippet[:300] + "..."
					}
					debugLog("[API-RESP-ERR] Status %d on %s | Body payload: %s", resp.StatusCode, url, bodySnippet)
				} else {
					debugLog("[API-RESP-SUCCESS] Status 200 on %s", url)
				}

				isTransientError := resp.StatusCode == 500 || resp.StatusCode == 502 || resp.StatusCode == 503 || resp.StatusCode == 504 || resp.StatusCode == 429
				bodyStr := string(body)
				if !isTransientError && (strings.Contains(bodyStr, "error code: 502") || strings.Contains(bodyStr, "error code: 503") || strings.Contains(bodyStr, "error code: 500")) {
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
	var bodyBytes []byte
	if req.Body != nil {
		bodyBytes, _ = io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}

	reqLog := getHumanReadableRequestLog(req.Method, req.URL.String(), bodyBytes)
	debugLog("%s", reqLog)

	resp, err := l.next.RoundTrip(req)
	if err != nil {
		debugLog("[ERROR] %s -> %v", reqLog, err)
		return nil, err
	}

	respLog := getHumanReadableResponseLog(req.Method, req.URL.String(), resp.StatusCode)
	debugLog("%s", respLog)
	return resp, nil
}

func getHumanReadableRequestLog(method, urlStr string, body []byte) string {
	u, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Sprintf("HTTP Request: %s %s", method, urlStr)
	}

	if u.Host == "api.allanime.day" && u.Path == "/api" && method == "POST" {
		var payload struct {
			Query     string                 `json:"query"`
			Variables map[string]interface{} `json:"variables"`
		}
		if json.Unmarshal(body, &payload) == nil && payload.Variables != nil {
			if search, ok := payload.Variables["search"].(map[string]interface{}); ok {
				if q, _ := search["query"].(string); q != "" {
					translation := "sub"
					if t, _ := payload.Variables["translationType"].(string); t != "" {
						translation = t
					}
					return fmt.Sprintf("[API] Search Anime: %q (%s)", q, translation)
				}
			}
			if showID, _ := payload.Variables["showId"].(string); showID != "" {
				translation := "sub"
				if t, _ := payload.Variables["translationType"].(string); t != "" {
					translation = t
				}
				showName := showID
				if cached, _, found := loadShowCache(showID); found {
					showName = cached.Name
				}
				return fmt.Sprintf("[API] Fetch Episode List: %s (%s)", showName, translation)
			}
		}
	}

	if u.Host == "api.allanime.day" && u.Path == "/api" && method == "GET" {
		q := u.Query()
		if varsStr := q.Get("variables"); varsStr != "" {
			var vars map[string]interface{}
			if json.Unmarshal([]byte(varsStr), &vars) == nil {
				showID, _ := vars["showId"].(string)
				epNo, _ := vars["episodeString"].(string)
				translation, _ := vars["translationType"].(string)
				if translation == "" {
					translation = "sub"
				}
				showName := showID
				if cached, _, found := loadShowCache(showID); found {
					showName = cached.Name
				}
				return fmt.Sprintf("[API] Resolve Episode %s Link: %s (%s)", epNo, showName, translation)
			}
		}
	}

	if u.Host == "allanime.day" && u.Path == "/apivtwo/clock.json" {
		return "[API] Fetch Server Clock"
	}

	if u.Host == "api.jikan.moe" {
		parts := strings.Split(u.Path, "/")
		if len(parts) >= 4 && parts[1] == "v4" && parts[2] == "anime" {
			malID := parts[3]
			action := "Fetch Details"
			if len(parts) >= 5 {
				action = "Fetch " + strings.Title(parts[4])
			}
			showName := ""
			m := loadMalIDToNameMap()
			if name, ok := m[malID]; ok {
				showName = name + " - "
			}
			pageQuery := ""
			if page := u.Query().Get("page"); page != "" {
				pageQuery = fmt.Sprintf(" (Page %s)", page)
			}
			return fmt.Sprintf("[MAL] %s: %sMAL ID: %s%s", action, showName, malID, pageQuery)
		}
		return fmt.Sprintf("[MAL] Request: %s %s", method, u.Path)
	}

	if u.Host == "api.aniskip.com" {
		parts := strings.Split(u.Path, "/")
		if len(parts) >= 5 && parts[1] == "v1" && parts[2] == "skip-times" {
			malID := parts[3]
			epNo := parts[4]
			showName := ""
			m := loadMalIDToNameMap()
			if name, ok := m[malID]; ok {
				showName = name + " - "
			}
			return fmt.Sprintf("[AniSkip] Fetch Skip Times: %sMAL ID: %s, Ep: %s", showName, malID, epNo)
		}
		return fmt.Sprintf("[AniSkip] Request: %s %s", method, u.Path)
	}

	cleanPath := u.Path
	if cleanPath == "" {
		cleanPath = "/"
	}
	return fmt.Sprintf("HTTP Request: %s %s%s", method, u.Host, cleanPath)
}

func getHumanReadableResponseLog(method, urlStr string, statusCode int) string {
	u, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Sprintf("HTTP Response: Status %d", statusCode)
	}

	prefix := "[API]"
	if u.Host == "api.jikan.moe" {
		prefix = "[MAL]"
	} else if u.Host == "api.aniskip.com" {
		prefix = "[AniSkip]"
	}

	return fmt.Sprintf("%s Response: Status %d", prefix, statusCode)
}

func newLoggingHttpClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &loggingRoundTripper{
			next: http.DefaultTransport,
		},
	}
}

type ResolvedStream struct {
	SourceName string
	Quality    string
	URL        string
}

func fetchAllResolvedStreams(showID, mode, episodeNo string) ([]ResolvedStream, error) {
	sources, err := fetchEpisodeSources(showID, mode, episodeNo)
	if err != nil {
		return nil, err
	}

	var results []ResolvedStream
	for _, src := range sources {
		links, err := fetchProviderLinks(src.SourceURL)
		if err == nil && len(links) > 0 {
			for qual, urlVal := range links {
				results = append(results, ResolvedStream{
					SourceName: src.SourceName,
					Quality:    qual,
					URL:        urlVal,
				})
			}
		}
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no streams resolved for episode %s (%s)", episodeNo, mode)
	}

	return results, nil
}
