package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func TestAniSkipAPI(t *testing.T) {
	if os.Getenv("NIX_BUILD_TOP") != "" {
		t.Skip("Skipping live network test inside Nix build sandbox")
	}

	// Enable debug logging for this test
	os.Setenv("CLARE_DEBUG", "1")
	defer os.Unsetenv("CLARE_DEBUG")

	results := fetchAniSkipTimes("5114", "1", 1440.0)
	if len(results) == 0 {
		t.Fatal("fetchAniSkipTimes returned no results for FMAB ep 1 (MAL ID 5114) - API may be down or broken")
	}

	foundOp := false
	foundEd := false
	for _, r := range results {
		t.Logf("AniSkip result: type=%s, start=%.3f, end=%.3f", r.SkipType, r.Interval.StartTime, r.Interval.EndTime)
		if r.SkipType == "op" {
			foundOp = true
			if r.Interval.StartTime < 0 || r.Interval.EndTime <= r.Interval.StartTime {
				t.Errorf("Invalid OP interval: start=%.3f, end=%.3f", r.Interval.StartTime, r.Interval.EndTime)
			}
		}
		if r.SkipType == "ed" {
			foundEd = true
			if r.Interval.StartTime < 0 || r.Interval.EndTime <= r.Interval.StartTime {
				t.Errorf("Invalid ED interval: start=%.3f, end=%.3f", r.Interval.StartTime, r.Interval.EndTime)
			}
		}
	}

	if !foundOp {
		t.Error("Expected to find OP skip time for FMAB ep 1")
	}
	if !foundEd {
		t.Error("Expected to find ED skip time for FMAB ep 1")
	}

	// Edge cases: invalid inputs should return nil
	if results := fetchAniSkipTimes("", "1", 1440.0); results != nil {
		t.Error("Expected nil for empty malID")
	}
	if results := fetchAniSkipTimes("0", "1", 1440.0); results != nil {
		t.Error("Expected nil for malID '0'")
	}
	if results := fetchAniSkipTimes("5114", "", 1440.0); results != nil {
		t.Error("Expected nil for empty epNo")
	}
}

func TestChaptersFileGeneration(t *testing.T) {
	if os.Getenv("NIX_BUILD_TOP") != "" {
		t.Skip("Skipping live network test inside Nix build sandbox")
	}

	os.Setenv("CLARE_DEBUG", "1")
	defer os.Unsetenv("CLARE_DEBUG")

	// Simulate what getMpvCmd does with AniSkip results
	results := fetchAniSkipTimes("5114", "1", 1440.0)
	if len(results) == 0 {
		t.Skip("Skipping chapters test - no AniSkip data available")
	}

	opStart := -1.0
	opEnd := -1.0
	edStart := -1.0
	edEnd := -1.0
	for _, r := range results {
		if r.SkipType == "op" {
			opStart = r.Interval.StartTime
			opEnd = r.Interval.EndTime
		} else if r.SkipType == "ed" {
			edStart = r.Interval.StartTime
			edEnd = r.Interval.EndTime
		}
	}

	t.Logf("OP: start=%.3f, end=%.3f", opStart, opEnd)
	t.Logf("ED: start=%.3f, end=%.3f", edStart, edEnd)

	if opStart < 0 {
		t.Error("Expected OP start time >= 0")
	}
	if edStart < 0 {
		t.Error("Expected ED start time >= 0")
	}
	if opEnd <= opStart {
		t.Error("OP end should be > OP start")
	}
	if edEnd <= edStart {
		t.Error("ED end should be > ED start")
	}
	if edStart <= opEnd {
		t.Error("ED should start after OP ends")
	}
}

func TestLogStreamingAndFormatting(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "clare-test-logs-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origStateDir := os.Getenv("CLARE_STATE_DIR")
	os.Setenv("CLARE_STATE_DIR", tmpDir)
	defer os.Setenv("CLARE_STATE_DIR", origStateDir)

	logChan := make(chan string, 10)

	logFile := filepath.Join(tmpDir, "debug.log")
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		t.Fatalf("failed to open log file: %v", err)
	}
	_, _ = f.WriteString("[00:00:00] Initial setup log\n")
	_ = f.Close()

	go tailLogFile(logChan)

	time.Sleep(600 * time.Millisecond)

	f, err = os.OpenFile(logFile, os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		t.Fatalf("failed to open log file for appending: %v", err)
	}
	
	testMsg := "HTTP Request: GET https://api.jikan.moe/v4/anime/889"
	_, _ = f.WriteString("[12:34:56] " + testMsg + "\n")
	_ = f.Close()

	select {
	case line := <-logChan:
		t.Logf("Read tailed log line: %q", line)
		if !strings.HasPrefix(line, "[12:34:56]") {
			t.Errorf("Expected line to start with timestamp '[12:34:56]', got %q", line)
		}
		if !strings.Contains(line, testMsg) {
			t.Errorf("Expected line to contain test message %q, got %q", testMsg, line)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for tailed log line from logChan")
	}

	mpvLogChan := make(chan string, 10)
	go func() {
		mpvLogChan <- "[MPV] AV: 00:01:23 / 00:23:28"
	}()

	select {
	case mpvLine := <-mpvLogChan:
		t.Logf("Read MPV log line: %q", mpvLine)
		if !strings.HasPrefix(mpvLine, "[MPV] ") {
			t.Errorf("Expected MPV log line to start with '[MPV] ', got %q", mpvLine)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for MPV log line")
	}
}

func TestModelLoggingIntegration(t *testing.T) {
	m := initialModel("", "sub", "best", false)
	m.telemetryViewport.Width = 80
	m.telemetryViewport.Height = 20

	if len(m.telemetryLogs) != 0 {
		t.Errorf("Expected empty telemetryLogs, got %d lines", len(m.telemetryLogs))
	}

	msg1 := clareLogMsg("[12:34:56] HTTP Request: GET https://api.jikan.moe")
	resModel, cmd := m.Update(msg1)
	m = resModel.(model)

	if len(m.telemetryLogs) != 1 || m.telemetryLogs[0] != string(msg1) {
		t.Errorf("Expected telemetryLogs to contain msg1, got %v", m.telemetryLogs)
	}
	if !strings.Contains(m.telemetryViewport.View(), string(msg1)) {
		t.Errorf("Expected viewport view to contain msg1, got %q", m.telemetryViewport.View())
	}
	if cmd == nil {
		t.Error("Expected next readClareLogsCmd to be returned/scheduled")
	}

	msg2 := clareLogMsg("[12:34:57] [MPV] AV: 00:01:23 / 00:23:28")
	resModel, cmd = m.Update(msg2)
	m = resModel.(model)

	if len(m.telemetryLogs) != 2 || m.telemetryLogs[1] != string(msg2) {
		t.Errorf("Expected telemetryLogs to contain msg2 at index 1, got %v", m.telemetryLogs)
	}
	if !strings.Contains(m.telemetryViewport.View(), string(msg2)) {
		t.Errorf("Expected viewport view to contain msg2, got %q", m.telemetryViewport.View())
	}
	if cmd == nil {
		t.Error("Expected next readClareLogsCmd to be returned/scheduled")
	}
}

func TestMpvProcessLoggingIntegration(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "clare-test-mpv-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origStateDir := os.Getenv("CLARE_STATE_DIR")
	os.Setenv("CLARE_STATE_DIR", tmpDir)
	defer os.Setenv("CLARE_STATE_DIR", origStateDir)

	m := initialModel("", "sub", "best", false)

	// Sleep to ensure tailLogFile has finished its startup delay and seeked to EOF
	time.Sleep(600 * time.Millisecond)

	cmd := exec.Command("bash", "-c", `echo "AV: 00:01:00 / 00:23:00"; sleep 0.2; echo "AV: 00:02:00 / 00:23:00"`)

	msg := resolvedPlaybackMsg{
		cmd:              cmd,
		tempLuaFile:      "dummy.lua",
		tempChaptersFile: "dummy.txt",
	}

	resModel, _ := m.Update(msg)
	m = resModel.(model)

	if m.state != stateEpisodeSelect {
		t.Errorf("Expected TUI state to remain stateEpisodeSelect, got %v", m.state)
	}

	if m.activeCmd != cmd {
		t.Errorf("Expected activeCmd to be cmd, got %v", m.activeCmd)
	}

	if m.clareLogChan == nil {
		t.Fatal("Expected clareLogChan to be initialized")
	}

	foundMpvLog := false
	timeout := time.After(3 * time.Second)
	for !foundMpvLog {
		select {
		case line := <-m.clareLogChan:
			t.Logf("Read clare log line: %q", line)
			if strings.Contains(line, "[MPV] ") {
				foundMpvLog = true
			}
		case <-timeout:
			t.Fatal("Timeout waiting for MPV log output in clareLogChan")
		}
	}

	_ = cmd.Process.Kill()
	_ = cmd.Wait()
}

func TestStreamCacheAndPrefetch(t *testing.T) {
	showID := "testShow123"
	mode := "sub"
	epNo := "5"
	quality := "1080p"
	expectedURL := "https://example.com/stream.m3u8"

	cacheKey := fmt.Sprintf("%s-%s-%s-%s", showID, mode, epNo, quality)
	
	streamCacheMu.Lock()
	streamCache[cacheKey] = expectedURL
	streamCacheMu.Unlock()

	resolved, err := resolveStreamURL(showID, mode, epNo, quality)
	if err != nil {
		t.Fatalf("resolveStreamURL returned error: %v", err)
	}
	if resolved != expectedURL {
		t.Errorf("Expected resolved URL to be %q, got %q", expectedURL, resolved)
	}

	streamCacheMu.Lock()
	delete(streamCache, cacheKey)
	streamCacheMu.Unlock()

	prefetchEpisodeStream("", "", "", "")
	prefetchEpisodeStream(showID, mode, epNo, quality)
}
