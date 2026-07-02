package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
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

	client := &http.Client{Timeout: 4 * time.Second}
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

	// Prepend injected configuration variables to the savePositionLua content
	luaContent := fmt.Sprintf(`
local mal_id = %q
local ep_no = %f
local jikan_duration = %f
`, malID, epVal, durationSeconds) + savePositionLua

	tmpFile, err := os.CreateTemp("", "clare-save-position-*.lua")
	if err != nil {
		return nil, "", "", err
	}
	if _, err := tmpFile.WriteString(luaContent); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
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
	}

	// Fetch AniSkip times synchronously and generate FFmpeg metadata chapters file
	tempChaptersFile := ""
	if malID != "" && malID != "0" {
		debugLog("getMpvCmd: fetching AniSkip skip times for malID=%s, epNo=%s", malID, epNo)
		times := fetchAniSkipTimes(malID, epNo, durationSeconds)
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
