package main

import (
	"bufio"
	"bytes"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
)

//go:embed save-position.lua
var savePositionLua string


type AniSkipInterval struct {
	StartTime float64 `json:"startTime"`
	EndTime   float64 `json:"endTime"`

	// Fallback fields for v1 API (snake_case)
	StartTimeSnake float64 `json:"start_time"`
	EndTimeSnake   float64 `json:"end_time"`
}

type AniSkipResult struct {
	Interval      AniSkipInterval `json:"interval"`
	SkipType      string          `json:"skipType"`
	SkipTypeSnake string          `json:"skip_type"`
}

type AniSkipResponse struct {
	Found   bool            `json:"found"`
	Results []AniSkipResult `json:"results"`
}

func resolveMALIDByTitle(title string) string {
	if title == "" {
		return ""
	}
	cleanTitle := title
	if idx := strings.Index(cleanTitle, "("); idx > 0 {
		cleanTitle = strings.TrimSpace(cleanTitle[:idx])
	}
	query := `
	query ($search: String) {
		Media(search: $search, type: ANIME) {
			idMal
		}
	}`
	payload := map[string]interface{}{
		"query": query,
		"variables": map[string]interface{}{
			"search": cleanTitle,
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return ""
	}

	req, err := http.NewRequest("POST", "https://graphql.anilist.co", bytes.NewBuffer(body))
	if err != nil {
		return ""
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", UserAgent)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}

	var res struct {
		Data struct {
			Media struct {
				IDMal *int `json:"idMal"`
			} `json:"Media"`
		} `json:"data"`
	}
	respBody, err := io.ReadAll(resp.Body)
	if err == nil && json.Unmarshal(respBody, &res) == nil && res.Data.Media.IDMal != nil && *res.Data.Media.IDMal > 0 {
		return fmt.Sprintf("%d", *res.Data.Media.IDMal)
	}
	return ""
}

func fetchAniSkipTimes(malID string, epNo string, durationSeconds float64, showTitle ...string) []AniSkipResult {
	if epNo == "" {
		return nil
	}
	isAniDB := strings.HasPrefix(malID, "anidb:")
	cleanMALID := stripProviderPrefix(malID)
	_, parseErr := strconv.Atoi(cleanMALID)
	if isAniDB || parseErr != nil {
		if len(showTitle) > 0 && showTitle[0] != "" {
			tName := showTitle[0]
			if idx := strings.Index(tName, " - Ep "); idx > 0 {
				tName = tName[:idx]
			}
			if resolved := resolveMALIDByTitle(tName); resolved != "" {
				cleanMALID = resolved
			}
		}
	}
	if cleanMALID == "" || cleanMALID == "0" {
		return nil
	}
	cleanEp := ""
	for _, r := range epNo {
		if (r >= '0' && r <= '9') || r == '.' {
			cleanEp += string(r)
		} else if cleanEp != "" {
			break
		}
	}
	if cleanEp == "" {
		return nil
	}

	if cached := loadAniSkipCache(cleanMALID, cleanEp); cached != nil {
		debugLog("fetchAniSkipTimes: cache hit for MAL %s Ep %s", cleanMALID, cleanEp)
		return cached
	}

	client := newLoggingHttpClient(4 * time.Second)
	epLen := int(durationSeconds)
	if epLen <= 0 {
		epLen = 1440
	}

	// Request all supported skip types (op, ed, mixed-op, mixed-ed, recap, ncop, nced, preview)
	var body []byte
	endpoints := []struct {
		ver string
		url string
	}{
		{
			ver: "v2",
			url: fmt.Sprintf("https://api.aniskip.com/v2/skip-times/%s/%s?types=op&types=ed&types=mixed-op&types=mixed-ed&types=recap&types=ncop&types=nced&types=preview&episodeLength=%d", cleanMALID, cleanEp, epLen),
		},
		{
			ver: "v1",
			url: fmt.Sprintf("https://api.aniskip.com/v1/skip-times/%s/%s?types=op&types=ed", cleanMALID, cleanEp),
		},
	}

	for _, ep := range endpoints {
		debugLog("fetchAniSkipTimes: requesting %s (%s)", ep.ver, ep.url)
		req, err := http.NewRequest("GET", ep.url, nil)
		if err != nil {
			debugLog("fetchAniSkipTimes: error creating request: %v", err)
			continue
		}
		req.Header.Set("User-Agent", UserAgent)
		resp, err := client.Do(req)
		if err != nil {
			debugLog("fetchAniSkipTimes: HTTP error: %v", err)
			continue
		}
		body, err = io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			debugLog("fetchAniSkipTimes: error reading body: %v", err)
			continue
		}
		debugLog("fetchAniSkipTimes: %s status=%d, body=%s", ep.ver, resp.StatusCode, string(body))
		if resp.StatusCode == http.StatusOK {
			break // success
		}
		body = nil
	}
	if body == nil {
		return nil
	}
	var result AniSkipResponse
	if err := json.Unmarshal(body, &result); err != nil {
		debugLog("fetchAniSkipTimes: JSON unmarshal error: %v", err)
		return nil
	}
	if !result.Found {
		debugLog("fetchAniSkipTimes: API returned found=false")
		return nil
	}

	// Normalize v1/v2 field names (snake_case vs camelCase)
	for i := range result.Results {
		if result.Results[i].SkipType == "" {
			result.Results[i].SkipType = result.Results[i].SkipTypeSnake
		}
		if result.Results[i].Interval.StartTime == 0 && result.Results[i].Interval.StartTimeSnake > 0 {
			result.Results[i].Interval.StartTime = result.Results[i].Interval.StartTimeSnake
		}
		if result.Results[i].Interval.EndTime == 0 && result.Results[i].Interval.EndTimeSnake > 0 {
			result.Results[i].Interval.EndTime = result.Results[i].Interval.EndTimeSnake
		}
	}

	debugLog("fetchAniSkipTimes: found %d skip times", len(result.Results))
	saveAniSkipCache(malID, cleanEp, result.Results)

	return result.Results
}



type IntroDBResponse struct {
	Found   bool `json:"found"`
	Results []struct {
		Type     string `json:"type"`
		Interval struct {
			Start float64 `json:"start_time"`
			End   float64 `json:"end_time"`
		} `json:"interval"`
	} `json:"results"`
}

func fetchTVSkipTimes(tmdbID string, seasonStr string, epNo string) []AniSkipResult {
	if tmdbID == "" || epNo == "" {
		return nil
	}
	season := "1"
	episode := epNo

	if strings.HasPrefix(strings.ToUpper(epNo), "S") && strings.Contains(strings.ToUpper(epNo), "E") {
		parts := strings.Split(strings.ToUpper(epNo)[1:], "E")
		if len(parts) == 2 {
			season = parts[0]
			episode = parts[1]
		}
	} else if seasonStr != "" {
		season = seasonStr
	}

	cleanEp := ""
	for _, r := range episode {
		if r >= '0' && r <= '9' {
			cleanEp += string(r)
		} else if cleanEp != "" {
			break
		}
	}
	if cleanEp == "" {
		cleanEp = "1"
	}

	client := &http.Client{Timeout: 2 * time.Second}
	apiURL := fmt.Sprintf("https://api.introdb.app/v1/skip?tmdb_id=%s&season=%s&episode=%s", tmdbID, season, cleanEp)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", UserAgent)
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil
	}

	var res IntroDBResponse
	if err := json.Unmarshal(body, &res); err != nil || !res.Found {
		return nil
	}

	var results []AniSkipResult
	for _, r := range res.Results {
		sType := strings.ToLower(r.Type)
		if sType == "intro" {
			sType = "op"
		} else if sType == "outro" {
			sType = "ed"
		}
		results = append(results, AniSkipResult{
			SkipType: sType,
			Interval: AniSkipInterval{
				StartTime: r.Interval.Start,
				EndTime:   r.Interval.End,
			},
		})
	}
	return results
}

func getMpvCmd(streamURL string, title string, epNo string, malID string, durationStr string, extraArgs []string) (*exec.Cmd, string, string, float64, string, error) {
	durationSeconds := parseJikanDuration(durationStr)
	epVal := parseEpisodeNumber(epNo)

	tempChaptersFile := ""
	var times []AniSkipResult
	isMovie := strings.EqualFold(epNo, "Movie") || (strings.EqualFold(epNo, "1") && strings.Contains(strings.ToLower(title), "movie"))
	isVidSrc := strings.HasPrefix(streamURL, "vidsrc:") || strings.HasPrefix(malID, "vidsrc:") || strings.Contains(streamURL, "influencerstrategy") || strings.Contains(streamURL, "nextgen") || strings.Contains(streamURL, "vaplayer") || strings.Contains(streamURL, ".site/")

	if isVidSrc && !isMovie {
		tmdbID := strings.TrimPrefix(malID, "vidsrc:tv:")
		tmdbID = strings.TrimPrefix(tmdbID, "vidsrc:movie:")
		tmdbID = strings.TrimPrefix(tmdbID, "vidsrc:")
		debugLog("getMpvCmd: fetching TV skip times for TMDB %s epNo=%s, title=%s", tmdbID, epNo, title)
		times = fetchTVSkipTimes(tmdbID, "", epNo)
	} else if !isMovie && !isVidSrc {
		debugLog("getMpvCmd: fetching AniSkip skip times for malID=%s, epNo=%s, title=%s", malID, epNo, title)
		times = fetchAniSkipTimes(malID, epNo, durationSeconds, title)
	}

	if len(times) > 0 {
		var ffmetadata strings.Builder
		ffmetadata.WriteString(";FFMETADATA1\n")

		opStart := -1.0
		opEnd := -1.0
		edStart := -1.0
		edEnd := -1.0
		for _, t := range times {
			if t.SkipType == "op" || t.SkipType == "mixed-op" || t.SkipType == "ncop" {
				opStart = t.Interval.StartTime
				opEnd = t.Interval.EndTime
			} else if t.SkipType == "ed" || t.SkipType == "mixed-ed" || t.SkipType == "nced" {
				edStart = t.Interval.StartTime
				edEnd = t.Interval.EndTime
			}
		}

			type chap struct {
				title string
				start int64
				end   int64
			}
			var chaps []chap

			if opStart > 0 {
				chaps = append(chaps, chap{title: "Prologue", start: 0, end: int64(opStart * 1000)})
				chaps = append(chaps, chap{title: "Opening", start: int64(opStart * 1000), end: int64(opEnd * 1000)})
				chaps = append(chaps, chap{title: "Episode Start", start: int64(opEnd * 1000), end: int64(durationSeconds * 1000)})
			} else if opStart == 0 {
				chaps = append(chaps, chap{title: "Opening", start: 0, end: int64(opEnd * 1000)})
				chaps = append(chaps, chap{title: "Episode Start", start: int64(opEnd * 1000), end: int64(durationSeconds * 1000)})
			} else {
				chaps = append(chaps, chap{title: "Episode Start", start: 0, end: int64(durationSeconds * 1000)})
			}

			if edStart > 0 {
				if len(chaps) > 0 && chaps[len(chaps)-1].start < int64(edStart*1000) {
					chaps[len(chaps)-1].end = int64(edStart * 1000)
				}
				chaps = append(chaps, chap{title: "Ending", start: int64(edStart * 1000), end: int64(edEnd * 1000)})
				if edEnd < durationSeconds {
					chaps = append(chaps, chap{title: "Outro", start: int64(edEnd * 1000), end: int64(durationSeconds * 1000)})
				}
			}

			// Sort chapters by start time
			sort.Slice(chaps, func(i, j int) bool {
				return chaps[i].start < chaps[j].start
			})

			for _, c := range chaps {
				if c.end > c.start {
					ffmetadata.WriteString("[CHAPTER]\n")
					ffmetadata.WriteString("TIMEBASE=1/1000\n")
					fmt.Fprintf(&ffmetadata, "START=%d\n", c.start)
					fmt.Fprintf(&ffmetadata, "END=%d\n", c.end)
					fmt.Fprintf(&ffmetadata, "title=%s\n\n", c.title)
				}
			}

			cf, err := os.CreateTemp("", "clare-chapters-*.txt")
			if err == nil {
				_, _ = cf.WriteString(ffmetadata.String())
				cf.Close()
				tempChaptersFile = cf.Name()
				debugLog("getMpvCmd: created chapters file %s with payload:\n%s", tempChaptersFile, ffmetadata.String())
			} else {
				debugLog("getMpvCmd: error creating temp chapters file: %v", err)
			}
	}

	cfg := loadConfig()
	var skipTimesJSON []byte
	if len(times) > 0 {
		skipTimesJSON, _ = json.Marshal(times)
	} else {
		skipTimesJSON = []byte("[]")
	}

	luaContent := fmt.Sprintf(`
local mal_id = %q
local ep_no = %f
local ep_str = %q
local jikan_duration = %f
local auto_skip = %t
local autoskip_delay = %f
local skip_times_json = %q
`, malID, epVal, epNo, durationSeconds, cfg.Autoskip, cfg.AutoskipDelay, string(skipTimesJSON)) + savePositionLua

	tmpFile, err := os.CreateTemp("", "clare-save-position-*.lua")
	if err != nil {
		if tempChaptersFile != "" {
			os.Remove(tempChaptersFile)
		}
		return nil, "", "", 0.0, "", err
	}
	if _, err := tmpFile.WriteString(luaContent); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		if tempChaptersFile != "" {
			os.Remove(tempChaptersFile)
		}
		return nil, "", "", 0.0, "", err
	}
	tmpFile.Close()

	referer := StreamReferer
	if strings.Contains(streamURL, "mp4upload.com") {
		referer = "https://www.mp4upload.com/"
	} else if strings.Contains(streamURL, "fast4speed") {
		referer = ""
	} else if strings.Contains(streamURL, "flikhub") || strings.Contains(streamURL, "kotocdn") || strings.Contains(streamURL, "megap") {
		referer = "https://megaplay.buzz/"
	} else if strings.Contains(streamURL, "allanime") || strings.Contains(streamURL, "alltropic") || strings.Contains(streamURL, "ok.ru") {
		referer = "https://youtu-chan.com/"
	} else if strings.Contains(streamURL, "cloudorchestranova") || strings.Contains(streamURL, "zealotsofzenith") || strings.Contains(streamURL, "vidsrc") {
		referer = "https://cloudorchestranova.com/"
	} else if strings.Contains(streamURL, "nextgen") || strings.Contains(streamURL, "quietmidnight") || strings.Contains(streamURL, "influencerstrategy") || strings.Contains(streamURL, "vaplayer") || strings.Contains(streamURL, ".site/") {
		referer = "https://nextgencloudfabric.com/"
	}

	headerFields := "User-Agent: " + UserAgent
	if referer != "" {
		headerFields = "Referer: " + referer + "\r\nUser-Agent: " + UserAgent
	}

	keepOpenFlag := "--keep-open=yes"
	for _, extra := range extraArgs {
		if strings.HasPrefix(extra, "--keep-open=") {
			keepOpenFlag = extra
			break
		}
	}

	mediaTitle := formatMediaTitle(title, epNo, "", malID)

	args := []string{
		"--tls-verify=no",
		"--no-resume-playback",
		"--force-media-title=" + mediaTitle,
		"--script=" + tmpFile.Name(),
		"--http-header-fields=" + headerFields,
		"--input-ipc-server=" + getMpvSocketPath(),
		"--osc=yes",
		"--sub-auto=fuzzy",
		"--slang=eng,en,enUS,en-US",
		"--alang=jpn,ja,eng,en",
		keepOpenFlag,
	}

	if isRunningInTest() {
		args = append(args, "--mute=yes")
	}

	isDirectHLS := strings.Contains(streamURL, "m3u8") ||
		strings.Contains(streamURL, "vidsrc") ||
		strings.Contains(streamURL, "cloudorchestranova") ||
		strings.Contains(streamURL, "zealotsofzenith") ||
		strings.Contains(streamURL, "nextgen") ||
		strings.Contains(streamURL, "vaplayer") ||
		strings.Contains(streamURL, "kotocdn") ||
		strings.Contains(streamURL, "flikhub") ||
		strings.Contains(streamURL, "megap")

	if isDirectHLS {
		args = append(args, "--no-ytdl")
		args = append(args, "--demuxer-lavf-o=allowed_segment_extensions=ALL,probesize=1000000,analyzeduration=1000000,fflags=+nobuffer")
	} else {
		args = append(args, "--ytdl-raw-options=user-agent="+UserAgent+",referer="+referer+",extractor-args=generic:impersonate")
	}

	if tempChaptersFile != "" {
		args = append(args, "--chapters-file="+tempChaptersFile)
	}

	startSeconds := 0.0
	positions, err := loadPositions()
	if err == nil && positions != nil && malID != "" {
		keysToTry := []string{malID, stripProviderPrefix(malID)}
		var showState ShowState
		foundState := false
		for _, k := range keysToTry {
			if k != "" {
				if st, ok := positions[k]; ok && st.ResumeState != nil {
					showState = st
					foundState = true
					break
				}
			}
		}
		if foundState && showState.ResumeState != nil {
			res := showState.ResumeState
			matches := false
			if res.EpisodeStr != "" && strings.EqualFold(res.EpisodeStr, epNo) {
				matches = true
			} else {
				reqEp := parseEpisodeNumber(epNo)
				if res.Episode == reqEp || (isMovie && (res.Episode == 0 || reqEp == 0)) {
					matches = true
				}
			}
			if matches && res.PositionSeconds > 0 {
				startSeconds = res.PositionSeconds
			}
		}
	}
	if startSeconds > 0 {
		args = append(args, fmt.Sprintf("--start=%f", startSeconds))
		debugLog("getMpvCmd: resuming episode %s at position %f seconds", epNo, startSeconds)
	} else {
		args = append(args, "--start=0")
		debugLog("getMpvCmd: starting episode %s from the beginning (--start=0)", epNo)
	}

	playURL := streamURL
	if strings.Contains(streamURL, "flikhub") || strings.Contains(streamURL, "kotocdn") {
		if sanitizedFile, err := SanitizeM3U8Playlist(streamURL, map[string]string{"Referer": referer, "User-Agent": UserAgent}); err == nil && sanitizedFile != "" {
			playURL = sanitizedFile
			debugLog("getMpvCmd: using sanitized local m3u8 playlist file/URL: %s", playURL)
		}
	}

	args = append(args, extraArgs...)
	args = append(args, playURL)

	cmd := exec.Command("mpv", args...)
	cmd.Env = append(os.Environ(), "FFMPEG_PROTOCOL_WHITELIST=file,http,https,tcp,tls,crypto,data,concat")
	return cmd, tmpFile.Name(), tempChaptersFile, durationSeconds, string(skipTimesJSON), nil
}

func SanitizeM3U8Playlist(streamURL string, headers map[string]string) (string, error) {
	if !strings.Contains(streamURL, "kotocdn") && !strings.Contains(streamURL, "flikhub") {
		return "", fmt.Errorf("sanitization not needed for non-ad streams")
	}

	body, err := doHTTPReqWithRetry("GET", streamURL, nil, headers)
	if err != nil {
		return "", err
	}

	content := string(body)
	if !strings.Contains(content, "#EXTM3U") {
		return "", fmt.Errorf("invalid m3u8 playlist body")
	}

	lines := strings.Split(content, "\n")

	// If master playlist, resolve and fetch variant playlist
	if strings.Contains(content, "#EXT-X-STREAM-INF") {
		var variantURL string
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
				variantURL = trimmed
				break
			}
		}
		if variantURL != "" {
			if !strings.HasPrefix(variantURL, "http") {
				variantURL = resolveRelativeM3U8URL(streamURL, variantURL)
			}
			variantBody, err := doHTTPReqWithRetry("GET", variantURL, nil, headers)
			if err == nil && strings.Contains(string(variantBody), "#EXTM3U") {
				content = string(variantBody)
				lines = strings.Split(content, "\n")
				streamURL = variantURL
			}
		}
	}

	var sanitizedLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.Contains(trimmed, "ibyteimg.com") ||
			strings.Contains(trimmed, "ad-site") ||
			strings.Contains(trimmed, ".png") ||
			strings.Contains(trimmed, "doubleclick") ||
			strings.Contains(trimmed, "googleadservices") {
			// Strip preceding #EXTINF tag if present in sanitizedLines
			if len(sanitizedLines) > 0 && strings.HasPrefix(sanitizedLines[len(sanitizedLines)-1], "#EXTINF:") {
				sanitizedLines = sanitizedLines[:len(sanitizedLines)-1]
			}
			continue
		}

		// Ensure relative segment URLs are resolved to full HTTPS URLs for local file playback
		if !strings.HasPrefix(trimmed, "#") && trimmed != "" && !strings.HasPrefix(trimmed, "http") {
			line = resolveRelativeM3U8URL(streamURL, trimmed)
		}

		sanitizedLines = append(sanitizedLines, line)
	}

	tmpFile, err := os.CreateTemp("", "clare-sanitized-*.m3u8")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	if _, err := tmpFile.WriteString(strings.Join(sanitizedLines, "\n")); err != nil {
		os.Remove(tmpFile.Name())
		return "", err
	}

	return tmpFile.Name(), nil
}

func resolveRelativeM3U8URL(baseURLStr, relativeURLStr string) string {
	baseURL, err := url.Parse(baseURLStr)
	if err != nil {
		return relativeURLStr
	}
	relURL, err := url.Parse(relativeURLStr)
	if err != nil {
		return relativeURLStr
	}
	return baseURL.ResolveReference(relURL).String()
}

func playSingleCmd(streamURL, title, epNo, malID, durationStr string) (*exec.Cmd, string, string, float64, string, error) {
	return getMpvCmd(streamURL, title, epNo, malID, durationStr, nil)
}

func playSingleCmdWithExtra(streamURL, title, epNo, malID, durationStr string, extraArgs []string) (*exec.Cmd, string, string, float64, string, error) {
	return getMpvCmd(streamURL, title, epNo, malID, durationStr, extraArgs)
}


func downloadCmd(streamURL, title, epNo string) *exec.Cmd {
	outputName := fmt.Sprintf("%s - Episode %s", title, epNo)
	outputName = strings.ReplaceAll(outputName, "/", "-")

	referer := StreamReferer
	if strings.Contains(streamURL, "mp4upload.com") {
		referer = "https://www.mp4upload.com/"
	} else if strings.Contains(streamURL, "fast4speed") {
		referer = ""
	} else if strings.Contains(streamURL, "flikhub") || strings.Contains(streamURL, "kotocdn") || strings.Contains(streamURL, "megap") {
		referer = "https://megaplay.buzz/"
	} else if strings.Contains(streamURL, "allanime") || strings.Contains(streamURL, "alltropic") || strings.Contains(streamURL, "ok.ru") {
		referer = "https://youtu-chan.com/"
	} else if strings.Contains(streamURL, "cloudorchestranova") || strings.Contains(streamURL, "zealotsofzenith") || strings.Contains(streamURL, "vidsrc") {
		referer = "https://cloudorchestranova.com/"
	} else if strings.Contains(streamURL, "nextgen") || strings.Contains(streamURL, "quietmidnight") || strings.Contains(streamURL, "influencerstrategy") || strings.Contains(streamURL, "vaplayer") || strings.Contains(streamURL, ".site/") {
		referer = "https://nextgencloudfabric.com/"
	}

	var cmd *exec.Cmd
	if _, err := exec.LookPath("yt-dlp"); err == nil {
		args := []string{
			"--extractor-args", "generic:impersonate",
			"--user-agent", UserAgent,
		}
		if referer != "" {
			args = append(args, "--referer", referer)
		}
		args = append(args, streamURL, "-o", outputName+".mp4")
		cmd = exec.Command("yt-dlp", args...)
	} else {
		args := []string{"-extension_picky", "0"}
		if referer != "" {
			args = append(args, "-referer", referer)
		}
		args = append(args, "-i", streamURL, "-c", "copy", outputName+".mp4")
		cmd = exec.Command("ffmpeg", args...)
	}
	return cmd
}

type MpvStatus struct {
	PlaybackTime float64
	Duration     float64
	Paused       bool
	Volume       float64
}

func sendMpvCommand(conn net.Conn, cmd []interface{}) ([]byte, error) {
	_ = conn.SetDeadline(time.Now().Add(250 * time.Millisecond))
	payload := map[string]interface{}{
		"command": cmd,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	_, err = conn.Write(append(data, '\n'))
	if err != nil {
		return nil, err
	}

	reader := bufio.NewReader(conn)
	line, err := reader.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	return line, nil
}

func queryFloatProperty(conn net.Conn, prop string) (float64, error) {
	resp, err := sendMpvCommand(conn, []interface{}{"get_property", prop})
	if err != nil {
		return 0, err
	}
	var result struct {
		Data float64 `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return 0, err
	}
	return result.Data, nil
}

func queryBoolProperty(conn net.Conn, prop string) (bool, error) {
	resp, err := sendMpvCommand(conn, []interface{}{"get_property", prop})
	if err != nil {
		return false, err
	}
	var result struct {
		Data bool `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return false, err
	}
	return result.Data, nil
}

func queryMpvStatus() (MpvStatus, error) {
	conn, err := net.DialTimeout("unix", getMpvSocketPath(), 300*time.Millisecond)
	if err != nil {
		return MpvStatus{}, err
	}
	defer conn.Close()

	var status MpvStatus

	playbackTime, err := queryFloatProperty(conn, "playback-time")
	if err == nil {
		status.PlaybackTime = playbackTime
	}

	duration, err := queryFloatProperty(conn, "duration")
	if err == nil {
		status.Duration = duration
	}

	paused, err := queryBoolProperty(conn, "pause")
	if err == nil {
		status.Paused = paused
	}

	volume, err := queryFloatProperty(conn, "volume")
	if err == nil {
		status.Volume = volume
	}

	return status, nil
}

func executeMpvAction(cmd []interface{}) error {
	conn, err := net.DialTimeout("unix", getMpvSocketPath(), 300*time.Millisecond)
	if err != nil {
		return err
	}
	defer conn.Close()
	_, err = sendMpvCommand(conn, cmd)
	return err
}

func queryMediaTitle() (string, error) {
	conn, err := net.DialTimeout("unix", getMpvSocketPath(), 300*time.Millisecond)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	resp, err := sendMpvCommand(conn, []interface{}{"get_property", "media-title"})
	if err != nil {
		return "", err
	}
	var result struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return "", err
	}
	return result.Data, nil
}

func loadFileInMpv(streamURL, title, epNo, malID string, extraArgs []string, durationSeconds float64, skipTimesJSON string, isAutoplay ...bool) error {
	conn, err := net.DialTimeout("unix", getMpvSocketPath(), 200*time.Millisecond)
	if err != nil {
		return err
	}
	defer conn.Close()

	referer := StreamReferer
	if strings.Contains(streamURL, "mp4upload.com") {
		referer = "https://www.mp4upload.com/"
	} else if strings.Contains(streamURL, "cloudorchestranova") || strings.Contains(streamURL, "zealotsofzenith") || strings.Contains(streamURL, "vidsrc") {
		referer = "https://cloudorchestranova.com/"
	} else if strings.Contains(streamURL, "nextgen") || strings.Contains(streamURL, "quietmidnight") || strings.Contains(streamURL, "influencerstrategy") || strings.Contains(streamURL, "vaplayer") || strings.Contains(streamURL, ".site/") {
		referer = "https://nextgencloudfabric.com/"
	}

	if referer != "" {
		_, _ = sendMpvCommand(conn, []interface{}{"set_property", "referrer", referer})
	}
	_, _ = sendMpvCommand(conn, []interface{}{"set_property", "user-agent", UserAgent})

	startSeconds := 0.0
	autoplayFlag := len(isAutoplay) > 0 && isAutoplay[0]
	if !autoplayFlag {
		positions, posErr := loadPositions()
		if posErr == nil && positions != nil && malID != "" {
			if showState, ok := positions[malID]; ok && showState.ResumeState != nil {
				reqEp := parseEpisodeNumber(epNo)
				if showState.ResumeState.Episode == reqEp {
					startSeconds = showState.ResumeState.PositionSeconds
				}
			}
		}
	}

	_, _ = sendMpvCommand(conn, []interface{}{"script-message", "update-episode-info", malID, epNo, fmt.Sprintf("%f", durationSeconds), skipTimesJSON})

	_, err = sendMpvCommand(conn, []interface{}{"loadfile", streamURL, "replace"})
	if err != nil {
		return err
	}

	time.Sleep(150 * time.Millisecond)
	_, _ = sendMpvCommand(conn, []interface{}{"seek", startSeconds, "absolute"})
	if startSeconds > 0 {
		debugLog("loadFileInMpv: loaded next episode %s and seeked to position %f seconds", epNo, startSeconds)
	} else {
		debugLog("loadFileInMpv: loaded next episode %s from the beginning (seek 0.0s)", epNo)
	}

	fullTitle := formatMediaTitle(title, epNo, "", malID)
	_, _ = sendMpvCommand(conn, []interface{}{"set_property", "force-media-title", fullTitle})

	for _, arg := range extraArgs {
		if strings.HasPrefix(arg, "--audio-file=") {
			audioPath := strings.TrimPrefix(arg, "--audio-file=")
			_, _ = sendMpvCommand(conn, []interface{}{"audio-add", audioPath, "select"})
		} else if strings.HasPrefix(arg, "--sub-file=") {
			subPath := strings.TrimPrefix(arg, "--sub-file=")
			_, _ = sendMpvCommand(conn, []interface{}{"sub-add", subPath, "select"})
		}
	}

	_, _ = sendMpvCommand(conn, []interface{}{"set_property", "pause", false})
	if isRunningInTest() {
		_, _ = sendMpvCommand(conn, []interface{}{"set_property", "mute", true})
	}

	return nil
}

func isRunningInTest() bool {
	if strings.HasSuffix(os.Args[0], ".test") || os.Getenv("CLARE_TESTING") == "1" || flag.Lookup("test.v") != nil {
		return true
	}
	for _, arg := range os.Args {
		if strings.HasPrefix(arg, "-test.") {
			return true
		}
	}
	return false
}



func getMpvSocketPath() string {
	if path := os.Getenv("CLARE_MPV_SOCK"); path != "" {
		return path
	}
	return "/tmp/clare-mpv.sock"
}
