package main

import (
	"testing"
)

func TestMovieToggleCompleted(t *testing.T) {
	movieShowID := "vidsrc:movie:12345"

	m := initialModel("", "sub", "best", false)

	mockShow := AnimeShow{
		ID:    movieShowID,
		Name:  "Blade Runner 2049",
		Type:  "MOVIE",
		MALID: "",
	}
	_ = saveShowCache(movieShowID, mockShow, []string{"1"})

	// 1. Initially toggle movie completed
	m.toggleShowCompleted(movieShowID)

	pos, err := loadPositions()
	if err != nil {
		t.Fatalf("loadPositions failed: %v", err)
	}

	st, ok := pos[movieShowID]
	if !ok || len(st.CompletedEpisodes) == 0 {
		t.Fatalf("expected movie %s to be marked completed in positions.json, got %+v", movieShowID, st)
	}

	// 2. Toggle movie back to uncompleted
	m.toggleShowCompleted(movieShowID)

	pos, _ = loadPositions()
	st = pos[movieShowID]
	if len(st.CompletedEpisodes) > 0 {
		t.Fatalf("expected movie %s to be unmarked, got completed_episodes: %v", movieShowID, st.CompletedEpisodes)
	}
}

func TestMovieEpisodeTitleFormatting(t *testing.T) {
	movieShowID := "vidsrc:movie:blade-runner-2049"

	m := initialModel("", "sub", "best", false)
	m.selectedShow = AnimeShow{
		ID:    movieShowID,
		Name:  "Blade Runner 2049",
		Type:  "MOVIE",
		MALID: "",
	}
	m.episodes = []string{"1"}
	m.episodeDetails = map[string]JikanEpInfo{
		"1": {Title: "Time to Die"},
	}

	m.refreshEpisodeListItems()

	if len(m.episodeItems) == 0 {
		t.Fatalf("expected episode items to be populated")
	}

	epItem, ok := m.episodeItems[0].(episodeItem)
	if !ok {
		t.Fatalf("expected item to be episodeItem")
	}

	if epItem.title != "Full Movie" {
		t.Fatalf("expected movie title to be 'Full Movie', got %q", epItem.title)
	}
}
