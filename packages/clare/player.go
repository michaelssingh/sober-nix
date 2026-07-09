package main

import (
	"bufio"
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

//go:embed save-position.lua
var savePositionLua string

func countAudioStreams(streamURL string) int {
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-referer", AllAnimeReferer,
		"-user_agent", UserAgent,
		"-select_streams", "a",
		"-show_entries", "stream=index",
		"-of", "csv=p=0",
		streamURL,
	)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		debugLog("countAudioStreams: ffprobe run failed for URL %s: %v", streamURL, err)
		return 0
	}
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	count := 0
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			count++
		}
	}
	debugLog("countAudioStreams: ffprobe detected %d audio stream(s) for URL %s", count, streamURL)
	return count
}

type AniSkipInterval struct {
	StartTime float64 `json:"startTime"`
	EndTime   float64 `json:"endTime"`
}

type AniSkipResult struct {
	Interval AniSkipInterval `json:"interval"`
	SkipType string          `json:"skipType"`
}

type AniSkipResponse struct {
	Found   bool            `json:"found"`
	Results []AniSkipResult `json:"results"`
}

func fetchAniSkipTimes(malID string, epNo string, durationSeconds float64) []AniSkipResult {
	if malID == "" || malID == "0" || epNo == "" {
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

	// 1. Try to load from local cache first
	if cached := loadAniSkipCache(malID, cleanEp); cached != nil {
		debugLog("fetchAniSkipTimes: cache hit for MAL %s Ep %s", malID, cleanEp)
		return cached
	}

	client := newLoggingHttpClient(4 * time.Second)
	url := fmt.Sprintf("https://api.aniskip.com/v2/skip-times/%s/%s?types[]=op&types[]=ed&types[]=mixed-op&types[]=mixed-ed&types[]=recap&episodeLength=0", malID, cleanEp)
	debugLog("fetchAniSkipTimes: requesting %s", url)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		debugLog("fetchAniSkipTimes: error creating request: %v", err)
		return nil
	}
	req.Header.Set("User-Agent", UserAgent)
	resp, err := client.Do(req)
	if err != nil {
		debugLog("fetchAniSkipTimes: HTTP error: %v", err)
		return nil
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		debugLog("fetchAniSkipTimes: error reading body: %v", err)
		return nil
	}
	debugLog("fetchAniSkipTimes: status=%d, body=%s", resp.StatusCode, string(body))
	if resp.StatusCode != http.StatusOK {
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
	debugLog("fetchAniSkipTimes: found %d skip times", len(result.Results))

	// 2. Save results to local cache
	saveAniSkipCache(malID, cleanEp, result.Results)

	return result.Results
}

func parseJikanDuration(d string) float64 {
	d = strings.ToLower(d)
	total := 0.0
	r := regexp.MustCompile(`(\d+)\s*(hr|min|sec|s|m|h)`)
	matches := r.FindAllStringSubmatch(d, -1)
	if len(matches) > 0 {
		for _, m := range matches {
			var val float64
			fmt.Sscanf(m[1], "%f", &val)
			unit := m[2]
			if strings.HasPrefix(unit, "h") {
				total += val * 3600
			} else if strings.HasPrefix(unit, "m") {
				total += val * 60
			} else if strings.HasPrefix(unit, "s") {
				total += val
			}
		}
		return total
	}
	rDigits := regexp.MustCompile(`\d+`)
	if m := rDigits.FindString(d); m != "" {
		var val float64
		fmt.Sscanf(m, "%f", &val)
		return val * 60
	}
	return 1440.0 // Default 24 mins
}

func getMpvCmd(streamURL string, title string, epNo string, malID string, durationStr string, extraArgs []string) (*exec.Cmd, string, string, error) {
	durationSeconds := parseJikanDuration(durationStr)
	epVal := parseEpisodeNumber(epNo)

	// 1. Fetch AniSkip times synchronously and generate FFmpeg metadata chapters file first
	tempChaptersFile := ""
	var times []AniSkipResult
	if malID != "" && malID != "0" {
		debugLog("getMpvCmd: fetching AniSkip skip times for malID=%s, epNo=%s", malID, epNo)
		times = fetchAniSkipTimes(malID, epNo, durationSeconds)
		if len(times) > 0 {
			var ffmetadata strings.Builder
			ffmetadata.WriteString(";FFMETADATA1\n")

			opStart := -1.0
			opEnd := -1.0
			edStart := -1.0
			edEnd := -1.0
			for _, t := range times {
				if t.SkipType == "op" || t.SkipType == "mixed-op" {
					opStart = t.Interval.StartTime
					opEnd = t.Interval.EndTime
				} else if t.SkipType == "ed" || t.SkipType == "mixed-ed" {
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
				if len(chaps) > 0 {
					chaps[len(chaps)-1].end = int64(edStart * 1000)
				}
				chaps = append(chaps, chap{title: "Ending", start: int64(edStart * 1000), end: int64(edEnd * 1000)})
				if edEnd < durationSeconds {
					chaps = append(chaps, chap{title: "Outro", start: int64(edEnd * 1000), end: int64(durationSeconds * 1000)})
				}
			}

			for _, c := range chaps {
				ffmetadata.WriteString("[CHAPTER]\n")
				ffmetadata.WriteString("TIMEBASE=1/1000\n")
				fmt.Fprintf(&ffmetadata, "START=%d\n", c.start)
				fmt.Fprintf(&ffmetadata, "END=%d\n", c.end)
				fmt.Fprintf(&ffmetadata, "title=%s\n\n", c.title)
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
		} else {
			debugLog("getMpvCmd: AniSkip returned no skip times for malID=%s, epNo=%s", malID, epNo)
		}
	}

	// 2. Prepend injected configuration variables (including auto_skip and skip_times_json) to the savePositionLua content
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
local jikan_duration = %f
local auto_skip = %t
local autoskip_delay = %f
local skip_times_json = %q
`, malID, epVal, durationSeconds, cfg.Autoskip, cfg.AutoskipDelay, string(skipTimesJSON)) + savePositionLua

	tmpFile, err := os.CreateTemp("", "clare-save-position-*.lua")
	if err != nil {
		if tempChaptersFile != "" {
			os.Remove(tempChaptersFile)
		}
		return nil, "", "", err
	}
	if _, err := tmpFile.WriteString(luaContent); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		if tempChaptersFile != "" {
			os.Remove(tempChaptersFile)
		}
		return nil, "", "", err
	}
	tmpFile.Close()

	args := []string{
		"--tls-verify=no",
		"--force-media-title=" + title + " - Episode " + epNo,
		"--script=" + tmpFile.Name(),
		"--http-header-fields=Referer: " + AllAnimeReferer + ",User-Agent: " + UserAgent,
		"--input-ipc-server=/tmp/clare-mpv.sock",
		"--osc=yes",
		"--keep-open=yes",
	}

	if tempChaptersFile != "" {
		args = append(args, "--chapters-file="+tempChaptersFile)
	}

	// Retrieve resume position from positions.json and append --start if present
	startSeconds := 0.0
	positions, err := loadPositions()
	if err == nil && positions != nil && malID != "" {
		if showState, ok := positions[malID]; ok && showState.ResumeState != nil {
			reqEp := parseEpisodeNumber(epNo)
			if showState.ResumeState.Episode == reqEp {
				startSeconds = showState.ResumeState.PositionSeconds
			}
		}
	}
	if startSeconds > 0 {
		args = append(args, fmt.Sprintf("--start=%f", startSeconds))
		debugLog("getMpvCmd: resuming episode %s at position %f seconds", epNo, startSeconds)
	}

	args = append(args, extraArgs...)
	args = append(args, streamURL)

	cmd := exec.Command("mpv", args...)
	return cmd, tmpFile.Name(), tempChaptersFile, nil
}

func playSingleCmd(streamURL, title, epNo, malID, durationStr string) (*exec.Cmd, string, string, error) {
	return getMpvCmd(streamURL, title, epNo, malID, durationStr, nil)
}

func playDualCmd(subStream, dubStream string, title, epNo, malID, durationStr string) (*exec.Cmd, string, string, error) {
	subTracks := countAudioStreams(subStream)
	if subTracks <= 0 {
		debugLog("playDualCmd: subTracks is %d, falling back to 1", subTracks)
		subTracks = 1
	}
	aid := fmt.Sprintf("%d", subTracks+1)
	extraArgs := []string{
		"--audio-file=" + dubStream,
		"--aid=" + aid,
	}
	return getMpvCmd(subStream, title, epNo, malID, durationStr, extraArgs)
}

func downloadCmd(streamURL, title, epNo string) *exec.Cmd {
	outputName := fmt.Sprintf("%s - Episode %s", title, epNo)
	outputName = strings.ReplaceAll(outputName, "/", "-")

	var cmd *exec.Cmd
	if _, err := exec.LookPath("yt-dlp"); err == nil {
		cmd = exec.Command("yt-dlp",
			"--referer", AllAnimeReferer,
			streamURL,
			"-o", outputName+".mp4",
		)
	} else {
		cmd = exec.Command("ffmpeg",
			"-extension_picky", "0",
			"-referer", AllAnimeReferer,
			"-i", streamURL,
			"-c", "copy",
			outputName+".mp4",
		)
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
	conn, err := net.DialTimeout("unix", "/tmp/clare-mpv.sock", 100*time.Millisecond)
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
	conn, err := net.DialTimeout("unix", "/tmp/clare-mpv.sock", 100*time.Millisecond)
	if err != nil {
		return err
	}
	defer conn.Close()
	_, err = sendMpvCommand(conn, cmd)
	return err
}

func queryMediaTitle() (string, error) {
	conn, err := net.DialTimeout("unix", "/tmp/clare-mpv.sock", 100*time.Millisecond)
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

func loadFileInMpv(streamURL, title, epNo, malID string, extraArgs []string) error {
	conn, err := net.DialTimeout("unix", "/tmp/clare-mpv.sock", 100*time.Millisecond)
	if err != nil {
		return err
	}
	defer conn.Close()

	// Stop current and load new stream
	_, err = sendMpvCommand(conn, []interface{}{"loadfile", streamURL, "replace"})
	if err != nil {
		return err
	}

	// Update player window title
	fullTitle := fmt.Sprintf("%s - Episode %s", title, epNo)
	_, _ = sendMpvCommand(conn, []interface{}{"set_property", "force-media-title", fullTitle})

	// Add external dub stream if dual-audio mapping is active
	for _, arg := range extraArgs {
		if strings.HasPrefix(arg, "--audio-file=") {
			audioPath := strings.TrimPrefix(arg, "--audio-file=")
			_, _ = sendMpvCommand(conn, []interface{}{"audio-add", audioPath, "select"})
		}
	}

	return nil
}

