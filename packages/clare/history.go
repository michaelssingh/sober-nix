package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type HistoryEntry struct {
	ShowID    string `json:"show_id"`
	ShowName  string `json:"show_name"`
	Episode   string `json:"episode"`
	Timestamp int64  `json:"timestamp"`
	// Completed is set to true when AniList confirms COMPLETED status on pull,
	// or when the user watches the final episode locally. This is the canonical
	// completion signal — Clare never infers completion from episode count math alone.
	Completed bool `json:"completed,omitempty"`
	// AnilistID caches the AniList media ID so syncToAniList can skip the
	// MAL-to-AniList ID resolution API call on every episode advance.
	AnilistID int `json:"anilist_id,omitempty"`
}

func getHistoryPath() string {
	dir := os.Getenv("CLARE_STATE_DIR")
	if dir == "" {
		stateHome := os.Getenv("XDG_STATE_HOME")
		if stateHome == "" {
			home, _ := os.UserHomeDir()
			stateHome = filepath.Join(home, ".local", "state")
		}
		dir = filepath.Join(stateHome, "clare")
	}
	return filepath.Join(dir, "history.json")
}

func loadHistory() ([]HistoryEntry, error) {
	path := getHistoryPath()
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []HistoryEntry{}, nil
		}
		return nil, err
	}
	defer f.Close()

	var history []HistoryEntry
	if err := json.NewDecoder(f).Decode(&history); err != nil {
		return nil, err
	}
	return history, nil
}

func saveHistory(history []HistoryEntry) error {
	path := getHistoryPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	// Write to a temp file first, then rename atomically to avoid corruption.
	tmpPath := path + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(history); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return err
	}
	f.Close()

	// Keep a one-step-behind backup before overwriting.
	backupPath := path + ".backup"
	if existing, err := os.ReadFile(path); err == nil {
		_ = os.WriteFile(backupPath, existing, 0644)
	}

	return os.Rename(tmpPath, path)
}

func getUniqueHistory(history []HistoryEntry) []HistoryEntry {
	seen := make(map[string]bool)
	var unique []HistoryEntry
	for _, entry := range history {
		if strings.HasPrefix(entry.ShowName, "AniDB Show") {
			if cached, _, found := loadShowCache(entry.ShowID); found && cached.Name != "" && !strings.HasPrefix(cached.Name, "AniDB Show") {
				entry.ShowName = cached.Name
			}
		}
		if !seen[entry.ShowID] {
			seen[entry.ShowID] = true
			unique = append(unique, entry)
		}
	}
	return unique
}

func recordWatch(showID, showName, episode string) error {
	history, err := loadHistory()
	if err != nil {
		history = []HistoryEntry{}
	}

	if strings.HasPrefix(showName, "AniDB Show") {
		if cached, _, found := loadShowCache(showID); found && cached.Name != "" && !strings.HasPrefix(cached.Name, "AniDB Show") {
			showName = cached.Name
		}
	}

	// Update any existing history entries for this showID if showName was enriched
	for i := range history {
		if history[i].ShowID == showID && strings.HasPrefix(history[i].ShowName, "AniDB Show") && !strings.HasPrefix(showName, "AniDB Show") {
			history[i].ShowName = showName
		}
	}

	newEntry := HistoryEntry{
		ShowID:    showID,
		ShowName:  showName,
		Episode:   episode,
		Timestamp: time.Now().Unix(),
	}

	// Prepend
	history = append([]HistoryEntry{newEntry}, history...)

	return saveHistory(history)
}

type ResumeState struct {
	Episode         float64 `json:"episode"`
	PositionSeconds float64 `json:"position_seconds"`
	TotalSeconds    float64 `json:"total_seconds"`
}

type ShowState struct {
	ResumeState       *ResumeState `json:"resume_state"`
	CompletedEpisodes []float64    `json:"completed_episodes"`
	LastSyncedEp      float64      `json:"last_synced_ep,omitempty"`
}

func (s *ShowState) UnmarshalJSON(b []byte) error {
	if len(b) == 0 {
		return nil
	}
	// If it's a JSON number (starts with a digit, minus sign, dot, etc.)
	first := b[0]
	if (first >= '0' && first <= '9') || first == '-' || first == '.' {
		// Legacy numeric format (represented seconds watched keyed by title) - return empty ShowState
		return nil
	}

	// Try standard object unmarshaling
	type Alias ShowState
	var aux Alias
	if err := json.Unmarshal(b, &aux); err != nil {
		// Handle gracefully rather than failing the whole file load
		return nil
	}
	*s = ShowState(aux)
	return nil
}

type PositionsData map[string]ShowState

func getPositionsPath() string {
	dir := os.Getenv("CLARE_STATE_DIR")
	if dir == "" {
		stateHome := os.Getenv("XDG_STATE_HOME")
		if stateHome == "" {
			home, _ := os.UserHomeDir()
			stateHome = filepath.Join(home, ".local", "state")
		}
		dir = filepath.Join(stateHome, "clare")
	}
	return filepath.Join(dir, "positions.json")
}

func loadPositions() (PositionsData, error) {
	path := getPositionsPath()
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(PositionsData), nil
		}
		return nil, err
	}
	defer f.Close()

	var data PositionsData
	if err := json.NewDecoder(f).Decode(&data); err != nil {
		// If decoding fails completely (e.g. syntax error or invalid top-level JSON), return empty map
		return make(PositionsData), nil
	}

	// Auto-cleanup legacy keys / empty entries
	for k, v := range data {
		if v.ResumeState == nil && len(v.CompletedEpisodes) == 0 {
			delete(data, k)
		}
	}

	return data, nil
}

func savePositions(data PositionsData) error {
	path := getPositionsPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
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

func getSearchHistoryPath() string {
	dir := os.Getenv("CLARE_STATE_DIR")
	if dir == "" {
		stateHome := os.Getenv("XDG_STATE_HOME")
		if stateHome == "" {
			home, _ := os.UserHomeDir()
			stateHome = filepath.Join(home, ".local", "state")
		}
		dir = filepath.Join(stateHome, "clare")
	}
	return filepath.Join(dir, "search_history.json")
}

func loadSearchHistory() ([]string, error) {
	path := getSearchHistoryPath()
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	defer f.Close()

	var history []string
	if err := json.NewDecoder(f).Decode(&history); err != nil {
		return nil, err
	}
	return history, nil
}

func saveSearchHistory(history []string) error {
	path := getSearchHistoryPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	return encoder.Encode(history)
}

func recordSearch(query string) error {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}

	history, err := loadSearchHistory()
	if err != nil {
		history = []string{}
	}

	var filtered []string
	for _, q := range history {
		if q != query {
			filtered = append(filtered, q)
		}
	}

	history = append([]string{query}, filtered...)
	if len(history) > 20 {
		history = history[:20]
	}

	return saveSearchHistory(history)
}
