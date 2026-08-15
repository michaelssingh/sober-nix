package main

import (
	"testing"
)

func TestMovieToggleCompleted(t *testing.T) {
	movieShowID := "flikhub:movie:12345"

	m := initialModel("", "sub", "best", false)

	mockShow := AnimeShow{
		ID:    movieShowID,
		Name:  "Test Sci-Fi Movie",
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
