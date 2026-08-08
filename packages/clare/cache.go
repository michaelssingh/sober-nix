package main

import (
	"encoding/json"
	"os"
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

func loadEpisodeMetadataCache(showID, malID string) (map[string]JikanEpInfo, error) {
	if showID != "" {
		cleanID := strings.ReplaceAll(showID, ":", "_")
		path := filepath.Join(getCacheDir(), "episodes", cleanID+".json")
		if f, err := os.Open(path); err == nil {
			var data map[string]JikanEpInfo
			if json.NewDecoder(f).Decode(&data) == nil && len(data) > 0 {
				f.Close()
				return data, nil
			}
			f.Close()
		}
	}
	if malID != "" && malID != "0" {
		return loadJikanCache(malID)
	}
	return nil, nil
}

func saveEpisodeMetadataCache(showID, malID string, data map[string]JikanEpInfo) error {
	if len(data) == 0 {
		return nil
	}
	if showID != "" {
		cleanID := strings.ReplaceAll(showID, ":", "_")
		path := filepath.Join(getCacheDir(), "episodes", cleanID+".json")
		_ = os.MkdirAll(filepath.Dir(path), 0o755)
		if f, err := os.Create(path); err == nil {
			encoder := json.NewEncoder(f)
			encoder.SetIndent("", "  ")
			_ = encoder.Encode(data)
			f.Close()
		}
	}
	if malID != "" && malID != "0" {
		_ = saveJikanCache(malID, data)
	}
	return nil
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

// AniSkip Cache

func getAniSkipCachePath(malID, epNo string) string {
	return filepath.Join(getCacheDir(), "aniskip", malID, epNo+".json")
}

func loadAniSkipCache(malID, epNo string) []AniSkipResult {
	path := getAniSkipCachePath(malID, epNo)
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	var results []AniSkipResult
	if json.NewDecoder(f).Decode(&results) == nil {
		return results
	}
	return nil
}

func saveAniSkipCache(malID, epNo string, results []AniSkipResult) {
	if len(results) == 0 {
		return
	}
	path := getAniSkipCachePath(malID, epNo)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	if f, err := os.Create(path); err == nil {
		_ = json.NewEncoder(f).Encode(results)
		f.Close()
	}
}
