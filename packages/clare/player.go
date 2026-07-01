package main

import (
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
	StartTime float64 `json:"start_time"`
	EndTime   float64 `json:"end_time"`
}

type AniSkipResult struct {
	Interval AniSkipInterval `json:"interval"`
	SkipType string          `json:"skip_type"`
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

	client := &http.Client{Timeout: 4 * time.Second}
	url := fmt.Sprintf("https://api.aniskip.com/v1/skip-times/%s/%s?types[]=op&types[]=ed&types[]=recap&types[]=mixed-op&types[]=mixed-ed&episodeLength=%f", malID, cleanEp, durationSeconds)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", UserAgent)
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}
	var result AniSkipResponse
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil
	}
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

func getMpvCmd(streamURL string, title string, epNo string, malID string, durationStr string, extraArgs []string) (*exec.Cmd, string, error) {
	durationSeconds := parseJikanDuration(durationStr)
	epVal := parseEpisodeNumber(epNo)

	// Prepend injected configuration variables to the savePositionLua content
	luaContent := fmt.Sprintf(`
local mal_id = %q
local ep_no = %f
local jikan_duration = %f
`, malID, epVal, durationSeconds) + savePositionLua

	tmpFile, err := os.CreateTemp("", "clare-save-position-*.lua")
	if err != nil {
		return nil, "", err
	}
	if _, err := tmpFile.WriteString(luaContent); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return nil, "", err
	}
	tmpFile.Close()

	args := []string{
		"--really-quiet",
		"--tls-verify=no",
		"--force-media-title=" + title + " - Episode " + epNo,
		"--script=" + tmpFile.Name(),
		"--http-header-fields=Referer: " + AllAnimeReferer + ",User-Agent: " + UserAgent,
		"--input-ipc-server=/tmp/clare-mpv.sock",
		"--osc=yes",
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
	}

	args = append(args, extraArgs...)
	args = append(args, streamURL)

	cmd := exec.Command("mpv", args...)
	return cmd, tmpFile.Name(), nil
}

func playSingleCmd(streamURL, title, epNo, malID, durationStr string) (*exec.Cmd, string, error) {
	return getMpvCmd(streamURL, title, epNo, malID, durationStr, nil)
}

func playDualCmd(subStream, dubStream string, title, epNo, malID, durationStr string) (*exec.Cmd, string, error) {
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

func monitorAndInjectChapters(malID, epNo string, duration float64) {
	if malID == "" || malID == "0" {
		return
	}
	skipTimes := fetchAniSkipTimes(malID, epNo, duration)
	if len(skipTimes) == 0 {
		return
	}

	var chapters []map[string]any
	opStart := -1.0
	opEnd := -1.0
	edStart := -1.0
	edEnd := -1.0

	for _, t := range skipTimes {
		if t.SkipType == "op" || t.SkipType == "mixed-op" {
			opStart = t.Interval.StartTime
			opEnd = t.Interval.EndTime
		} else if t.SkipType == "ed" || t.SkipType == "mixed-ed" {
			edStart = t.Interval.StartTime
			edEnd = t.Interval.EndTime
		}
	}

	if opStart > 0 {
		chapters = append(chapters, map[string]any{"title": "Prologue", "time": 0.0})
		chapters = append(chapters, map[string]any{"title": "Opening", "time": opStart})
		chapters = append(chapters, map[string]any{"title": "Episode Start", "time": opEnd})
	} else if opStart == 0 {
		chapters = append(chapters, map[string]any{"title": "Opening", "time": 0.0})
		chapters = append(chapters, map[string]any{"title": "Episode Start", "time": opEnd})
	} else {
		chapters = append(chapters, map[string]any{"title": "Episode Start", "time": 0.0})
	}

	if edStart > 0 {
		chapters = append(chapters, map[string]any{"title": "Ending", "time": edStart})
		if edEnd < duration {
			chapters = append(chapters, map[string]any{"title": "Outro", "time": edEnd})
		}
	}

	socketPath := "/tmp/clare-mpv.sock"
	for i := 0; i < 30; i++ {
		conn, err := net.Dial("unix", socketPath)
		if err == nil {
			defer conn.Close()
			payload := map[string]any{
				"command": []any{
					"set_property",
					"chapter-list",
					chapters,
				},
			}
			data, err := json.Marshal(payload)
			if err == nil {
				conn.Write(append(data, '\n'))
				debugLog("monitorAndInjectChapters: successfully injected chapters payload: %s", string(data))
			}
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	debugLog("monitorAndInjectChapters: failed to connect to mpv IPC socket after 6 seconds")
}
