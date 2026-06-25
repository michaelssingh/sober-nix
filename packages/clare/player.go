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

func fetchAniSkipTimes(malID string, epNo string) []AniSkipResult {
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
	url := fmt.Sprintf("https://api.aniskip.com/v1/skip-times/%s/%s?types[]=op&types[]=ed&types[]=recap&types[]=mixed-op&types[]=mixed-ed", malID, cleanEp)
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

func getMpvCmd(streamURL string, title string, epNo string, malID string, extraArgs []string) (*exec.Cmd, string, error) {
	skipTimes := fetchAniSkipTimes(malID, epNo)
	luaContent := savePositionLua
	if len(skipTimes) > 0 {
		var intervals []string
		for _, t := range skipTimes {
			label := "Opening"
			if t.SkipType == "ed" || t.SkipType == "mixed-ed" {
				label = "Ending"
			} else if t.SkipType == "recap" {
				label = "Recap"
			}
			intervals = append(intervals, fmt.Sprintf("{ start = %f, [\"end\"] = %f, label = %q }", t.Interval.StartTime, t.Interval.EndTime, label))
		}
		luaContent += fmt.Sprintf(`
-- Auto-generated AniSkip auto-skipper
local skip_intervals = {
    %s
}

mp.observe_property("time-pos", "number", function(name, val)
    if not val then return end
    for _, interval in ipairs(skip_intervals) do
        if val >= interval.start and val < interval["end"] - 0.5 then
            mp.commandv("seek", interval["end"], "absolute")
            mp.osd_message("Auto-skipped " .. interval.label, 3)
            break
        end
    end
end)
`, strings.Join(intervals, ",\n    "))
	}

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
		"--tls-verify=no",
		"--force-media-title=" + title + " - Episode " + epNo,
		"--script=" + tmpFile.Name(),
		"--http-header-fields=Referer: " + AllAnimeReferer + ",User-Agent: " + UserAgent,
	}
	args = append(args, extraArgs...)
	args = append(args, streamURL)

	cmd := exec.Command("mpv", args...)
	return cmd, tmpFile.Name(), nil
}

func playSingleCmd(streamURL, title, epNo, malID string) (*exec.Cmd, string, error) {
	return getMpvCmd(streamURL, title, epNo, malID, nil)
}

func playDualCmd(subStream, dubStream string, title, epNo, malID string) (*exec.Cmd, string, error) {
	extraArgs := []string{
		"--audio-file=" + dubStream,
		"--aid=last",
	}
	return getMpvCmd(subStream, title, epNo, malID, extraArgs)
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
