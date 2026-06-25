package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"strings"
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

func getMpvCmd(streamURL string, title string, epNo string, extraArgs []string) (*exec.Cmd, string, error) {
	// Write the embedded Lua script to a temporary file
	tmpFile, err := os.CreateTemp("", "clare-save-position-*.lua")
	if err != nil {
		return nil, "", err
	}
	if _, err := tmpFile.WriteString(savePositionLua); err != nil {
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

func playSingleCmd(streamURL, title, epNo string) (*exec.Cmd, string, error) {
	return getMpvCmd(streamURL, title, epNo, nil)
}

func playDualCmd(subStream, dubStream string, title, epNo string) (*exec.Cmd, string, error) {
	extraArgs := []string{
		"--audio-file=" + dubStream,
		"--aid=last",
	}
	return getMpvCmd(subStream, title, epNo, extraArgs)
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
