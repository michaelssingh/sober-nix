package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	AllAnimeReferer     = "https://mkissa.to"
	StreamReferer       = "https://mkissa.to/" // Required for Wixmp/HLS CDN and Clock Handshakes
	AllAnimeBase        = "mkissa.net"
	AllAnimeAPI         = "https://api.mkissa.net/api"
	UserAgent           = "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:150.0) Gecko/20100101 Firefox/150.0"
	allAnimeKeyPhrase   = "Xot36i3lK3:v1"
	allAnimeQueryHash   = "f4662f4b7510b26795dd53ef824a0bf1740fbbc5d1273fab18222ac831bca8d0"
	allAnimeQueryOrigin = "https://mkissa.to"
)

var (
	allAnimeClientMaskHex = "948f4e192f9b462ec946efc1996bbc8e66c7ef768c184f4aabf512b3505c9247"
	allAnimeKey           = func() []byte {
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

type MediaType string

const (
	MediaTypeAnime  MediaType = "anime"
	MediaTypeMovie  MediaType = "movie"
	MediaTypeTV     MediaType = "tv"
	MediaTypeManga  MediaType = "manga"
	MediaTypeSports MediaType = "sports"
)

type MediaItem struct {
	ID                      string              `json:"_id"`
	Provider                string              `json:"provider"`
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

	// Unified Media extension fields
	MediaType MediaType `json:"media_type"`
	TMDBID    string    `json:"tmdb_id,omitempty"`
	Genres    []string  `json:"genres,omitempty"`
	Rating    string    `json:"rating,omitempty"`
}

type AnimeShow = MediaItem

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
	if strings.HasPrefix(decoded, "//") {
		decoded = "https:" + decoded
	} else if strings.HasPrefix(decoded, "/") {
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
	versionByte := data[0]

	// Key 1: Versioned Legacy Key
	keySeed := fmt.Sprintf("Xot36i3lK3:v%d", versionByte)
	legacyKeyBytes := sha256.Sum256([]byte(keySeed))

	// Try Legacy Key first
	block, err := aes.NewCipher(legacyKeyBytes[:])
	if err == nil {
		aesGCM, errGCM := cipher.NewGCM(block)
		if errGCM == nil {
			ciphertextWithTag := data[13:]
			plaintext, errOpen := aesGCM.Open(nil, nonce, ciphertextWithTag, nil)
			if errOpen == nil {
				return plaintext, nil
			}
		}
	}

	// Key 2: Derived Key from Key Manager (fallback)
	_, derivedKey, _, errDK := getDerivedKey()
	if errDK == nil {
		block, err = aes.NewCipher(derivedKey)
		if err == nil {
			aesGCM, errGCM := cipher.NewGCM(block)
			if errGCM == nil {
				ciphertextWithTag := data[13:]
				plaintext, errOpen := aesGCM.Open(nil, nonce, ciphertextWithTag, nil)
				if errOpen == nil {
					return plaintext, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("GCM decryption failed for all keys")
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

var (
	cachedEpoch      int64
	cachedDerivedKey []byte
	cachedBuildID    string
	cachedFetchedAt  time.Time
	cachedMutex      sync.Mutex
)

func getDerivedKey() (int64, []byte, string, error) {
	cachedMutex.Lock()
	defer cachedMutex.Unlock()

	if cachedEpoch > 0 && time.Since(cachedFetchedAt) < 30*time.Minute {
		return cachedEpoch, cachedDerivedKey, cachedBuildID, nil
	}

	epoch, key, buildID, err := fetchAllAnimeCryptoMaterial()
	if err != nil {
		return 0, nil, "", err
	}

	cachedEpoch = epoch
	cachedDerivedKey = key
	cachedBuildID = buildID
	cachedFetchedAt = time.Now()
	return epoch, key, buildID, nil
}

func invalidateDerivedKeyCache() {
	cachedMutex.Lock()
	cachedEpoch = 0
	cachedFetchedAt = time.Time{}
	cachedMutex.Unlock()
}

func fetchAllAnimeCryptoMaterial() (int64, []byte, string, error) {
	headers := map[string]string{
		"User-Agent": UserAgent,
		"Referer":    "https://mkissa.to/",
	}

	// 1. Fetch homepage
	htmlBytes, err := doHTTPReqWithRetry("GET", "https://mkissa.to/", nil, headers)
	if err != nil {
		return 0, nil, "", fmt.Errorf("failed to fetch mkissa homepage: %w", err)
	}
	html := string(htmlBytes)

	// 2. Parse __aaCrypto JSON (with fallback if site removed it)
	reCrypto := regexp.MustCompile(`(?:window\.)?__aaCrypto\s*=\s*(\{[^{}]*\})`)
	matchCrypto := reCrypto.FindStringSubmatch(html)
	if len(matchCrypto) < 2 {
		debugLog("fetchAllAnimeCryptoMaterial: __aaCrypto not found in homepage, using fallback key")
		return time.Now().Unix(), allAnimeKey, "64", nil
	}

	var bootstrap struct {
		Epoch int64  `json:"epoch"`
		PartB string `json:"partB"`
	}
	if err := json.Unmarshal([]byte(matchCrypto[1]), &bootstrap); err != nil {
		return 0, nil, "", fmt.Errorf("failed to parse __aaCrypto JSON: %w", err)
	}

	partB, err := base64.StdEncoding.DecodeString(bootstrap.PartB)
	if err != nil {
		return 0, nil, "", fmt.Errorf("failed to decode partB: %w", err)
	}
	if len(partB) < 32 {
		return 0, nil, "", fmt.Errorf("partB too short: %d bytes", len(partB))
	}

	// 3. Find app entry JS
	reApp := regexp.MustCompile(`import\("([^"]*/entry/app\.[^"]*\.js)"\)`)
	matchApp := reApp.FindStringSubmatch(html)
	if len(matchApp) < 2 {
		return 0, nil, "", fmt.Errorf("unable to find SvelteKit app entry JS")
	}
	appURL := matchApp[1]

	// 4. Fetch app entry JS
	appJSBytes, err := doHTTPReqWithRetry("GET", appURL, nil, headers)
	if err != nil {
		return 0, nil, "", fmt.Errorf("failed to fetch app entry JS: %w", err)
	}
	appJS := string(appJSBytes)

	// 5. Extract chunk names
	reChunks := regexp.MustCompile(`\.\./chunks/([A-Za-z0-9_-]+\.js)`)
	matchesChunks := reChunks.FindAllStringSubmatch(appJS, -1)
	if len(matchesChunks) == 0 {
		return 0, nil, "", fmt.Errorf("no chunk references found in app entry JS")
	}

	// Get base URL for chunks
	lastEntryIdx := strings.LastIndex(appURL, "/entry/")
	if lastEntryIdx == -1 {
		return 0, nil, "", fmt.Errorf("invalid app entry URL structure: %q", appURL)
	}
	chunkBaseURL := appURL[:lastEntryIdx] + "/chunks/"

	// 6. Find the crypto chunk containing "aaReq"
	var maskHex string
	var buildID string
	maxChunks := 40
	if len(matchesChunks) < maxChunks {
		maxChunks = len(matchesChunks)
	}

	// Extract unique chunk names to avoid duplicate fetches
	seen := make(map[string]bool)
	var chunkNames []string
	for _, m := range matchesChunks {
		name := m[1]
		if !seen[name] {
			seen[name] = true
			chunkNames = append(chunkNames, name)
		}
	}

	for i, name := range chunkNames {
		if i >= maxChunks {
			break
		}
		chunkURL := chunkBaseURL + name
		chunkBytes, err := doHTTPReqWithRetry("GET", chunkURL, nil, headers)
		if err != nil {
			continue
		}
		chunkContent := string(chunkBytes)

		if strings.Contains(chunkContent, "aaReq") {
			reHex := regexp.MustCompile(`\b[0-9a-fA-F]{64}\b`)
			matchesHex := reHex.FindAllString(chunkContent, -1)
			for _, hexStr := range matchesHex {
				if !strings.EqualFold(hexStr, allAnimeQueryHash) {
					maskHex = hexStr
					break
				}
			}

			// Extract buildId dynamically
			reBuildVar := regexp.MustCompile(`buildId=\\?"\+encodeURIComponent\(([a-zA-Z0-9_]+)\)`)
			matchBuildVar := reBuildVar.FindStringSubmatch(chunkContent)
			if len(matchBuildVar) >= 2 {
				varName := matchBuildVar[1]
				reBuildVal := regexp.MustCompile(fmt.Sprintf(`\b%s\s*=\s*"([^"]+)"`, varName))
				matchBuildVal := reBuildVal.FindStringSubmatch(chunkContent)
				if len(matchBuildVal) >= 2 {
					buildID = matchBuildVal[1]
				}
			}

			if maskHex != "" {
				break
			}
		}
	}

	if maskHex == "" {
		return 0, nil, "", fmt.Errorf("unable to find client mask in SvelteKit chunks")
	}

	if buildID == "" {
		buildID = "64" // fallback to current known buildID
	}

	mask, err := hex.DecodeString(maskHex)
	if err != nil {
		return 0, nil, "", fmt.Errorf("failed to decode client mask hex: %w", err)
	}

	derivedKey := make([]byte, 32)
	for i := 0; i < 32; i++ {
		derivedKey[i] = partB[i] ^ mask[i%len(mask)]
	}

	return bootstrap.Epoch, derivedKey, buildID, nil
}

func generateAAReq(qh string) (string, error) {
	epoch, key, buildID, err := getDerivedKey()
	if err != nil {
		// Fallback to legacy static key and epoch if fetching fails
		epoch = 4128
		key = allAnimeKey
		buildID = "64"
	}
	if buildID == "" {
		buildID = "64"
	}

	ts := (time.Now().Unix() / 300) * 300 * 1000

	payload := fmt.Sprintf(`{"v":1,"ts":%d,"epoch":%d,"buildId":"%s","qh":"%s"}`, ts, epoch, buildID, qh)

	ivSeed := fmt.Sprintf("%d:%s:%s:%d", epoch, buildID, qh, ts)
	hash := sha256.Sum256([]byte(ivSeed))
	iv := hash[:12]

	block, err := aes.NewCipher(key)
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
	_, _, buildID, err := getDerivedKey()
	if err != nil {
		buildID = "64"
	}
	if buildID == "" {
		buildID = "64"
	}

	queryVars := fmt.Sprintf(`{"showId":"%s","translationType":"%s","episodeString":"%s"}`, showID, mode, episodeNo)

	aareq, err := generateAAReq(allAnimeQueryHash)
	if err != nil {
		return nil, fmt.Errorf("failed to generate aaReq: %w", err)
	}

	queryExt := fmt.Sprintf(`{"aaReq":"%s"}`, aareq)
	reqURL := fmt.Sprintf("%s?variables=%s&extensions=%s", AllAnimeAPI, url.QueryEscape(queryVars), url.QueryEscape(queryExt))

	headers := map[string]string{
		"User-Agent": UserAgent,
		"Referer":    AllAnimeReferer,
		"Origin":     allAnimeQueryOrigin,
		"x-build-id": buildID,
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

var (
	streamCache        = make(map[string]string)
	streamCacheMu      sync.RWMutex
	activePrefetches   = make(map[string]bool)
	activePrefetchesMu sync.Mutex
)

var providers = []Provider{
	&AllAnimeProvider{},
	&AniDBProvider{},
	&VidSrcProvider{},
	&FlikhubProvider{},
	&GogoanimeProvider{},
}

func getProvider(name string) Provider {
	for _, p := range providers {
		if p.Name() == name {
			return p
		}
	}
	return &AllAnimeProvider{}
}

func rankTitle(name, query string) int {
	if name == "" {
		return 3
	}
	nameLower := strings.ToLower(name)
	queryLower := strings.ToLower(query)
	if nameLower == queryLower {
		return 0
	}
	if strings.HasPrefix(nameLower, queryLower) {
		return 1
	}
	if strings.Contains(nameLower, queryLower) {
		return 2
	}
	return 3
}

func rankSearchRelevance(show AnimeShow, query string) int {
	best := rankTitle(show.Name, query)
	if r := rankTitle(show.EnglishName, query); r < best {
		best = r
	}
	if r := rankTitle(show.NativeName, query); r < best {
		best = r
	}
	return best
}

func searchAnime(query, mode string) ([]AnimeShow, error) {
	var results []AnimeShow
	var mu sync.Mutex
	var wg sync.WaitGroup

	cfg := loadConfig()
	for _, p := range providers {
		if !cfg.IsProviderEnabled(p.Name()) {
			continue
		}
		wg.Add(1)
		go func(prov Provider) {
			defer wg.Done()
			shows, err := prov.Search(query, mode)
			if err == nil {
				mu.Lock()
				for i := range shows {
					shows[i].Provider = prov.Name()
					if !strings.HasPrefix(shows[i].ID, prov.Name()+":") {
						shows[i].ID = prov.Name() + ":" + shows[i].ID
					}
					_ = saveShowCache(shows[i].ID, shows[i], nil)
				}
				results = append(results, shows...)
				mu.Unlock()
			}
		}(p)
	}
	wg.Wait()

	var filtered []AnimeShow
	queryWords := strings.Fields(strings.ToLower(query))
	for _, show := range results {
		rank := rankSearchRelevance(show, query)
		if rank < 3 {
			filtered = append(filtered, show)
		} else {
			// Check if at least one query word matches
			titleLower := strings.ToLower(show.Name + " " + show.EnglishName + " " + show.NativeName)
			matched := false
			for _, qw := range queryWords {
				if len(qw) > 2 && strings.Contains(titleLower, qw) {
					matched = true
					break
				}
			}
			if matched {
				filtered = append(filtered, show)
			}
		}
	}
	results = filtered

	sort.SliceStable(results, func(i, j int) bool {
		rankI := rankSearchRelevance(results[i], query)
		rankJ := rankSearchRelevance(results[j], query)
		if rankI != rankJ {
			return rankI < rankJ
		}
		// Provider priority: anime-native providers first, then movie/TV
		provOrder := map[string]int{"allanime": 0, "anidb": 1, "flikhub": 2, "gogoanime": 3, "vidsrc": 4}
		pi, piOk := provOrder[strings.ToLower(results[i].Provider)]
		pj, pjOk := provOrder[strings.ToLower(results[j].Provider)]
		if !piOk {
			pi = 99
		}
		if !pjOk {
			pj = 99
		}
		if pi != pj {
			return pi < pj
		}
		return strings.ToLower(results[i].Name) < strings.ToLower(results[j].Name)
	})

	// Deduplicate: keep one entry per (normalised-title, type) pair.
	// After sorting the preferred provider is already first, so we keep the
	// first occurrence and discard later duplicates with the same title+type.
	seen := make(map[string]struct{})
	var deduped []AnimeShow
	for _, show := range results {
		key := strings.ToLower(strings.TrimSpace(show.Name)) + "|" + strings.ToLower(show.Type)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		deduped = append(deduped, show)
	}
	results = deduped

	return results, nil
}

func fetchEpisodeList(showID, mode string) (AnimeShow, []string, error) {
	cfg := loadConfig()
	provider := ""
	if cached, _, found := loadShowCache(showID); found && cached.Provider != "" {
		provider = cached.Provider
	}
	if provider == "" {
		// Infer provider by showID format
		if strings.HasSuffix(showID, "-online") || strings.Contains(showID, "-1") {
			provider = "gogoanime"
		} else if strings.Contains(showID, "-") {
			provider = "flikhub"
		} else {
			provider = "allanime"
		}
	}
	if !cfg.IsProviderEnabled(provider) {
		return AnimeShow{}, nil, fmt.Errorf("provider %q is disabled", provider)
	}
	p := getProvider(provider)
	return p.FetchEpisodes(showID, mode)
}

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

	streams, err := fetchAllResolvedStreams(showID, mode, episodeNo, "")
	if err != nil {
		return "", err
	}

	best := selectBestLinkFromResolved(streams, quality)
	if best != "" {
		streamCacheMu.Lock()
		streamCache[cacheKey] = best
		streamCacheMu.Unlock()
		return best, nil
	}

	return "", fmt.Errorf("no stream links resolved for episode %s (%s)", episodeNo, mode)
}

func selectBestLinkFromResolved(streams []ResolvedStream, requested string) string {
	if len(streams) == 0 {
		return ""
	}
	priorities := []string{"1080p", "720p", "480p", "360p", "hls", "best"}
	if requested == "worst" {
		priorities = []string{"360p", "480p", "720p", "1080p", "hls", "best"}
	} else if requested != "best" && requested != "" {
		for _, s := range streams {
			if strings.EqualFold(s.Quality, requested) {
				return s.URL
			}
		}
	}
	for _, p := range priorities {
		for _, s := range streams {
			if strings.EqualFold(s.Quality, p) {
				return s.URL
			}
		}
	}
	return streams[0].URL
}

func doHTTPReqWithRetry(method, url string, payload []byte, headers map[string]string) ([]byte, error) {
	var body []byte
	var err error
	client := newLoggingHttpClient(10 * time.Second)

	cookieVal := os.Getenv("CLARE_COOKIE")
	if cookieVal == "" {
		cookieVal = os.Getenv("ALLANIME_COOKIE")
	}
	maxAttempts := 5
	for attempt := 1; attempt <= maxAttempts; attempt++ {
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
				buildID := "64"
				if cachedBuildID != "" {
					buildID = cachedBuildID
				} else {
					if !strings.Contains(url, "chunks") && !strings.Contains(url, "entry") && !strings.Contains(url, "mkissa.to") {
						_, _, bid, err := getDerivedKey()
						if err == nil && bid != "" {
							buildID = bid
						}
					}
				}
				req.Header.Set("x-build-id", buildID)
				if req.Header.Get("Origin") == "" {
					req.Header.Set("Origin", "https://mkissa.to")
				}
				if cookieVal != "" {
					req.Header.Set("Cookie", cookieVal)
				}
			}

			if req.Header.Get("Referer") == "" {
				req.Header.Set("Referer", "https://mkissa.to")
			}
		}

		// Log detailed request fingerprinting
		var headerLogs []string
		for k, v := range req.Header {
			headerLogs = append(headerLogs, fmt.Sprintf("%s: %s", k, strings.Join(v, ",")))
		}
		debugLog("[API-REQ] Attempt %d/%d: %s %s | Headers: %s", attempt, maxAttempts, method, url, strings.Join(headerLogs, "; "))

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

				isTransientError := (resp.StatusCode >= 500 && resp.StatusCode < 600) || resp.StatusCode == 429
				bodyStr := string(body)
				if !isTransientError && (strings.Contains(bodyStr, "error code: 5") || strings.Contains(bodyStr, "Too many requests")) {
					isTransientError = true
					sleepSec := 5
					reSec := regexp.MustCompile(`try again in (\d+) second`)
					if m := reSec.FindStringSubmatch(bodyStr); len(m) >= 2 {
						if parsedSec, err := strconv.Atoi(m[1]); err == nil && parsedSec > 0 {
							sleepSec = parsedSec + 1
						}
					}
					debugLog("[API-RATE-LIMIT] Rate limit on %s. Sleeping %d seconds before retry (attempt %d/%d)...", url, sleepSec, attempt, maxAttempts)
					time.Sleep(time.Duration(sleepSec) * time.Second)
				}

				if !isTransientError {
					return body, nil
				}
				err = fmt.Errorf("transient HTTP error (status %d): %s", resp.StatusCode, strings.TrimSpace(bodyStr))
			}
		}

		debugLog("HTTP request attempt %d/%d failed for %s: %v", attempt, maxAttempts, url, err)
		if attempt < maxAttempts {
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

	if (strings.Contains(u.Host, "allanime") || strings.Contains(u.Host, "mkissa")) && strings.HasSuffix(u.Path, "/api") && method == "POST" {
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

	if (strings.Contains(u.Host, "allanime") || strings.Contains(u.Host, "mkissa")) && strings.HasSuffix(u.Path, "/api") && method == "GET" {
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

	if (strings.Contains(u.Host, "allanime") || strings.Contains(u.Host, "mkissa")) && strings.Contains(u.Path, "clock") {
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

type SubtitleTrack struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

type ResolvedStream struct {
	Provider   string
	SourceName string
	Quality    string
	URL        string
	Subtitles  []SubtitleTrack
}

func stripProviderPrefix(id string) string {
	if idx := strings.Index(id, ":"); idx != -1 {
		return id[idx+1:]
	}
	return id
}

func fetchAllResolvedStreams(showID, mode, episodeNo, providerName string) ([]ResolvedStream, error) {
	var results []ResolvedStream
	var mu sync.Mutex
	var wg sync.WaitGroup

	cfg := loadConfig()
	if providerName == "" && strings.Contains(showID, ":") {
		providerName = strings.SplitN(showID, ":", 2)[0]
	}

	targetProviders := providers
	if providerName != "" {
		for _, p := range providers {
			if strings.EqualFold(p.Name(), providerName) {
				targetProviders = []Provider{p}
				break
			}
		}
	}

	cleanID := stripProviderPrefix(showID)

	for _, p := range targetProviders {
		if !cfg.IsProviderEnabled(p.Name()) {
			debugLog("fetchAllResolvedStreams: skipping disabled provider %s", p.Name())
			continue
		}
		wg.Add(1)
		go func(prov Provider) {
			defer wg.Done()
			streams, err := prov.ResolveStreams(cleanID, mode, episodeNo, "best")
			if err == nil && len(streams) > 0 {
				mu.Lock()
				for _, s := range streams {
					if s.Provider == "" {
						s.Provider = prov.Name()
					}
					results = append(results, s)
				}
				mu.Unlock()
			}
		}(p)
	}
	wg.Wait()

	if len(results) == 0 {
		return nil, fmt.Errorf("no streams resolved for episode %s (%s)", episodeNo, mode)
	}

	return results, nil
}

type AiringShow struct {
	ID           int    `json:"id"`
	IDMal        int    `json:"idMal"`
	TitleRomaji  string `json:"titleRomaji"`
	TitleEnglish string `json:"titleEnglish"`
	TitleNative  string `json:"titleNative"`
	Description  string `json:"description"`
	Episodes     int    `json:"episodes"`
	CoverImage   string `json:"coverImage"`
	NextEpisode  int    `json:"nextEpisode"`
	TimeUntil    int    `json:"timeUntil"`
}

func fetchAiringAnime() ([]AiringShow, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	query := `
	query {
	  Page(page: 1, perPage: 20) {
	    media(status: RELEASING, sort: POPULARITY_DESC, type: ANIME) {
	      id
	      idMal
	      title {
	        romaji
	        english
	        native
	      }
	      coverImage {
	        large
	      }
	      description
	      episodes
	      nextAiringEpisode {
	        episode
	        timeUntilAiring
	      }
	    }
	  }
	}`

	payload := map[string]interface{}{
		"query": query,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", "https://graphql.anilist.co", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("AniList airing request returned status %d: %s", resp.StatusCode, string(respBytes))
	}

	var aniResp struct {
		Data struct {
			Page struct {
				Media []struct {
					ID    int   `json:"id"`
					IDMal *int  `json:"idMal"`
					Title struct {
						Romaji  string `json:"romaji"`
						English string `json:"english"`
						Native  string `json:"native"`
					} `json:"title"`
					CoverImage struct {
						Large string `json:"large"`
					} `json:"coverImage"`
					Description       string `json:"description"`
					Episodes          *int   `json:"episodes"`
					NextAiringEpisode *struct {
						Episode          int `json:"episode"`
						TimeUntilAiring int `json:"timeUntilAiring"`
					} `json:"nextAiringEpisode"`
				} `json:"media"`
			} `json:"Page"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&aniResp); err != nil {
		return nil, err
	}

	var shows []AiringShow
	for _, m := range aniResp.Data.Page.Media {
		malID := 0
		if m.IDMal != nil {
			malID = *m.IDMal
		}
		eps := 0
		if m.Episodes != nil {
			eps = *m.Episodes
		}
		nextEp := 0
		timeUntil := 0
		if m.NextAiringEpisode != nil {
			nextEp = m.NextAiringEpisode.Episode
			timeUntil = m.NextAiringEpisode.TimeUntilAiring
		}

		shows = append(shows, AiringShow{
			ID:           m.ID,
			IDMal:        malID,
			TitleRomaji:  m.Title.Romaji,
			TitleEnglish: m.Title.English,
			TitleNative:  m.Title.Native,
			Description:  m.Description,
			Episodes:     eps,
			CoverImage:   m.CoverImage.Large,
			NextEpisode:  nextEp,
			TimeUntil:    timeUntil,
		})
	}
	return shows, nil
}

func checkStreamMpvDryRun(urlVal string, headers map[string]string) (bool, error) {
	useYtdl := false
	lowerURL := strings.ToLower(urlVal)
	if strings.Contains(lowerURL, "ok.ru") ||
		strings.Contains(lowerURL, "filemoon") ||
		strings.Contains(lowerURL, "bysekoze") ||
		strings.Contains(lowerURL, "streamwish") ||
		strings.Contains(lowerURL, "listeamed") ||
		strings.Contains(lowerURL, "vidguard") {
		useYtdl = true
	}

	args := []string{
		"--vo=null",
		"--ao=null",
		"--length=5",
		"--msg-level=all=status",
		"--network-timeout=5",
	}
	if !useYtdl {
		args = append(args, "--ytdl=no")
	}
	var headerFields []string
	for k, v := range headers {
		headerFields = append(headerFields, fmt.Sprintf("%s: %s", k, v))
	}
	if len(headerFields) > 0 {
		args = append(args, fmt.Sprintf("--http-header-fields=%s", strings.Join(headerFields, ",")))
	}
	args = append(args, urlVal)

	cmd := exec.Command("mpv", args...)
	err := cmd.Run()
	if err != nil {
		return false, err
	}
	return true, nil
}
