package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func getCacheDir() string {
	dir := os.Getenv("CLARE_STATE_DIR")
	if dir == "" {
		stateHome := os.Getenv("XDG_STATE_HOME")
		if stateHome == "" {
			home, _ := os.UserHomeDir()
			stateHome = filepath.Join(home, ".local", "state")
		}
		dir = filepath.Join(stateHome, "clare")
	}
	return filepath.Join(dir, "cache")
}

// Jikan Cache

func getJikanCachePath(malID string) string {
	return filepath.Join(getCacheDir(), "jikan", malID+".json")
}

func loadJikanCache(malID string) (map[string]JikanEpInfo, error) {
	if malID == "" || malID == "0" {
		return nil, nil
	}
	path := getJikanCachePath(malID)
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]JikanEpInfo), nil
		}
		return nil, err
	}
	defer f.Close()

	var data map[string]JikanEpInfo
	if err := json.NewDecoder(f).Decode(&data); err != nil {
		return nil, err
	}
	return data, nil
}

func saveJikanCache(malID string, data map[string]JikanEpInfo) error {
	if malID == "" || malID == "0" || len(data) == 0 {
		return nil
	}
	path := getJikanCachePath(malID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// Show Cache (AllAnime)

type CachedShowEntry struct {
	Show      AnimeShow `json:"show"`
	Episodes  []string  `json:"episodes"`
	Timestamp int64     `json:"timestamp"`
}

func getShowCachePath(showID string) string {
	return filepath.Join(getCacheDir(), "shows", showID+".json")
}

func loadShowCache(showID string) (AnimeShow, []string, bool) {
	if showID == "" {
		return AnimeShow{}, nil, false
	}
	path := getShowCachePath(showID)
	f, err := os.Open(path)
	if err != nil {
		return AnimeShow{}, nil, false
	}
	defer f.Close()

	var entry CachedShowEntry
	if err := json.NewDecoder(f).Decode(&entry); err != nil {
		return AnimeShow{}, nil, false
	}

	// Invalidate show cache if it is older than 24 hours
	if time.Now().Unix()-entry.Timestamp > 24*60*60 {
		return AnimeShow{}, nil, false
	}

	return entry.Show, entry.Episodes, true
}

func saveShowCache(showID string, show AnimeShow, episodes []string) error {
	if showID == "" {
		return nil
	}
	path := getShowCachePath(showID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	entry := CachedShowEntry{
		Show:      show,
		Episodes:  episodes,
		Timestamp: time.Now().Unix(),
	}

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	return encoder.Encode(entry)
}

func downloadThumbnail(showID, urlStr string) (string, error) {
	if showID == "" || urlStr == "" {
		return "", fmt.Errorf("invalid arguments")
	}

	ext := ".jpg"
	if strings.Contains(urlStr, ".png") {
		ext = ".png"
	} else if strings.Contains(urlStr, ".webp") {
		ext = ".webp"
	}

	dir := filepath.Join(getCacheDir(), "thumbnails")
	path := filepath.Join(dir, showID+ext)

	if _, err := os.Stat(path); err == nil {
		return path, nil
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	client := newLoggingHttpClient(10 * time.Second)
	resp, err := client.Get(urlStr)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}

	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return "", err
	}

	return path, nil
}

func renderImageANSI(imgPath string, width, height int) string {
	if imgPath == "" {
		return ""
	}
	widthStr := fmt.Sprintf("%dx%d", width, height)
	debugLog("renderImageANSI: rendering image %s to dimensions %s with symbols", imgPath, widthStr)

	cmd := exec.Command("chafa", "-f", "symbols", "-s", widthStr, "--symbols", "block", "--colors", "256", imgPath)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		debugLog("renderImageANSI error running chafa: %v", err)
		return ""
	}
	return out.String()
}
