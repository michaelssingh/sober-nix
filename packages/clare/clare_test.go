package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCleanHTML(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello <br> World", "Hello \n World"},
		{"Hello <BR/> World", "Hello \n World"},
		{"Hello <br /> World", "Hello \n World"},
		{"<b>Bold</b> and <i>Italic</i>", "Bold and Italic"},
		{"&quot;Quotes&quot; &amp; Ampersand", "\"Quotes\" & Ampersand"},
		{"&#039;Single quote&#039; and &rsquo;apostrophe&rsquo;", "'Single quote' and 'apostrophe'"},
		{"   Spaces and tabs\t", "Spaces and tabs"},
	}

	for _, tc := range tests {
		actual := cleanHTML(tc.input)
		if actual != tc.expected {
			t.Errorf("cleanHTML(%q) = %q; expected %q", tc.input, actual, tc.expected)
		}
	}
}

func TestParseJikanDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"24 min", 24.0 * 60.0},
		{"24 min per ep", 24.0 * 60.0},
		{"1 hr 20 min", 1.0*3600.0 + 20.0*60.0},
		{"45 s", 45.0},
		{"2 hours", 2.0 * 3600.0},
		{"unknown", 1440.0}, // Default fallback
	}

	for _, tc := range tests {
		actual := parseJikanDuration(tc.input)
		if actual != tc.expected {
			t.Errorf("parseJikanDuration(%q) = %f; expected %f", tc.input, actual, tc.expected)
		}
	}
}

func TestParseEpisodeNumber(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"1", 1.0},
		{"12.5", 12.5},
		{"Episode 24", 24.0},
		{"Special 3.5", 3.5},
		{"NoNumber", 0.0},
	}

	for _, tc := range tests {
		actual := parseEpisodeNumber(tc.input)
		if actual != tc.expected {
			t.Errorf("parseEpisodeNumber(%q) = %f; expected %f", tc.input, actual, tc.expected)
		}
	}
}

func TestPositionsFile(t *testing.T) {
	// Set up temporary CLARE_STATE_DIR
	tmpDir, err := os.MkdirTemp("", "clare-test-state-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origStateDir := os.Getenv("CLARE_STATE_DIR")
	os.Setenv("CLARE_STATE_DIR", tmpDir)
	defer os.Setenv("CLARE_STATE_DIR", origStateDir)

	// Load empty positions
	data, err := loadPositions()
	if err != nil {
		t.Fatalf("loadPositions failed on empty: %v", err)
	}
	if len(data) != 0 {
		t.Errorf("expected empty PositionsData, got %d items", len(data))
	}

	// Save some position data
	mockData := PositionsData{
		"12345": ShowState{
			ResumeState: &ResumeState{
				Episode:         5.0,
				PositionSeconds: 350.5,
				TotalSeconds:    1440.0,
			},
			CompletedEpisodes: []float64{1.0, 2.0, 3.0},
		},
	}

	err = savePositions(mockData)
	if err != nil {
		t.Fatalf("savePositions failed: %v", err)
	}

	// Verify file was created
	filePath := filepath.Join(tmpDir, "positions.json")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Errorf("expected positions.json to be created at %s", filePath)
	}

	// Reload and verify
	reloaded, err := loadPositions()
	if err != nil {
		t.Fatalf("loadPositions failed on reload: %v", err)
	}
	showState, ok := reloaded["12345"]
	if !ok {
		t.Fatalf("reloaded data does not contain key '12345'")
	}
	if showState.ResumeState == nil {
		t.Fatalf("expected ResumeState to be non-nil")
	}
	if showState.ResumeState.Episode != 5.0 || showState.ResumeState.PositionSeconds != 350.5 {
		t.Errorf("ResumeState values incorrect, got episode %f, position %f", showState.ResumeState.Episode, showState.ResumeState.PositionSeconds)
	}
	if len(showState.CompletedEpisodes) != 3 || showState.CompletedEpisodes[2] != 3.0 {
		t.Errorf("CompletedEpisodes values incorrect, got %v", showState.CompletedEpisodes)
	}
}
