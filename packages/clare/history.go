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
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	return encoder.Encode(history)
}

func getUniqueHistory(history []HistoryEntry) []HistoryEntry {
	seen := make(map[string]bool)
	var unique []HistoryEntry
	for _, entry := range history {
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

	// Filter out older entries of this exact show/episode if we want to keep history size reasonable,
	// but keeping the full log is fine. Let's prepend the new one.
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

