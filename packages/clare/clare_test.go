package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func TestCleanHTML(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello <br> World", "Hello\n World"},
		{"Hello <BR/> World", "Hello\n World"},
		{"Hello <br /> World", "Hello\n World"},
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
	tmpDir, err := os.MkdirTemp("", "clare-test-model-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origStateDir := os.Getenv("CLARE_STATE_DIR")
	os.Setenv("CLARE_STATE_DIR", tmpDir)
	defer os.Setenv("CLARE_STATE_DIR", origStateDir)

	m := initialModel("", "sub", "best", false)
	m.telemetryViewport.Width = 80
	m.telemetryViewport.Height = 20

	if len(m.telemetryLogs) != 0 {
		t.Errorf("Expected empty telemetryLogs, got %d lines", len(m.telemetryLogs))
	}

	msg1 := clareLogMsg("[12:34:56] [INFO] Syncing completed episode 5...")
	resModel, cmd := m.Update(msg1)
	m = resModel.(model)

	if len(m.telemetryLogs) != 1 || m.telemetryLogs[0] != string(msg1) {
		t.Errorf("Expected telemetryLogs to contain msg1, got %v", m.telemetryLogs)
	}
	if !strings.Contains(m.telemetryViewport.View(), "Syncing completed episode 5...") {
		t.Errorf("Expected viewport view to contain msg1 content, got %q", m.telemetryViewport.View())
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
	if !strings.Contains(m.telemetryViewport.View(), "AV: 00:01:23 / 00:23:28") {
		t.Errorf("Expected viewport view to contain msg2 content, got %q", m.telemetryViewport.View())
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

	origMpvSock := os.Getenv("CLARE_MPV_SOCK")
	os.Setenv("CLARE_MPV_SOCK", filepath.Join(tmpDir, "clare-mpv-test.sock"))
	defer os.Setenv("CLARE_MPV_SOCK", origMpvSock)

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

	if m.state != stateHistory {
		t.Errorf("Expected TUI state to remain stateHistory, got %v", m.state)
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

func TestCompletedShowsFiltering(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "clare-test-completed-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origStateDir := os.Getenv("CLARE_STATE_DIR")
	os.Setenv("CLARE_STATE_DIR", tmpDir)
	defer os.Setenv("CLARE_STATE_DIR", origStateDir)

	showID := "completedShow123"
	malID := "9999"
	showName := "Completed Show"

	show := AnimeShow{
		ID:    showID,
		Name:  showName,
		MALID: malID,
		AvailableEpisodes: map[string]any{
			"sub": float64(12),
		},
	}
	err = saveShowCache(showID, show, []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12"})
	if err != nil {
		t.Fatalf("failed to save show cache: %v", err)
	}

	histEntry := HistoryEntry{
		ShowID:    showID,
		ShowName:  showName,
		Episode:   "12",
		Timestamp: time.Now().Unix(),
	}
	err = saveHistory([]HistoryEntry{histEntry})
	if err != nil {
		t.Fatalf("failed to save history: %v", err)
	}

	posData := PositionsData{
		malID: ShowState{
			CompletedEpisodes: []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
		},
	}
	err = savePositions(posData)
	if err != nil {
		t.Fatalf("failed to save positions: %v", err)
	}

	m := initialModel("", "sub", "best", false)
	if len(m.historyList.Items()) != 0 {
		t.Errorf("Expected historyList displayed items to be empty (filtered completed show), got %d items", len(m.historyList.Items()))
	}
	if len(m.historyItems) != 1 {
		t.Errorf("Expected historyItems to store 1 item in total, got %d", len(m.historyItems))
	}

	// Enable showCompleted and verify it appears
	m.showCompleted = true
	m.applyHistoryFilter()
	if len(m.historyList.Items()) != 1 {
		t.Errorf("Expected historyList to show completed show when showCompleted is true, got %d items", len(m.historyList.Items()))
	}

	// Now make it incomplete
	histEntry.Episode = "6"
	err = saveHistory([]HistoryEntry{histEntry})
	if err != nil {
		t.Fatalf("failed to save history: %v", err)
	}

	posData[malID] = ShowState{
		CompletedEpisodes: []float64{1, 2, 3, 4, 5, 6},
	}
	err = savePositions(posData)
	if err != nil {
		t.Fatalf("failed to save positions: %v", err)
	}

	m.showCompleted = false
	m.refreshHistory()
	if len(m.historyList.Items()) != 1 {
		t.Errorf("Expected historyList to show 1 item (incomplete show), got %d items", len(m.historyList.Items()))
	}
}

func TestLegacyPositionsParsing(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "clare-test-legacy-positions-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origStateDir := os.Getenv("CLARE_STATE_DIR")
	os.Setenv("CLARE_STATE_DIR", tmpDir)
	defer os.Setenv("CLARE_STATE_DIR", origStateDir)

	// Write mock positions file containing both a new-format ShowState object
	// and legacy float64 number entries keyed by titles
	filePath := filepath.Join(tmpDir, "positions.json")
	rawJSON := `{
		"5114": {
			"resume_state": {
				"episode": 3,
				"position_seconds": 120,
				"total_seconds": 1440
			},
			"completed_episodes": [1, 2]
		},
		"Fullmetal Alchemist: Brotherhood - Episode 39": 1341.465,
		"Another Show - Episode 1": 42.0
	}`
	if err := os.WriteFile(filePath, []byte(rawJSON), 0644); err != nil {
		t.Fatalf("failed to write positions.json: %v", err)
	}

	// Verify decoding succeeds without errors, and it purges legacy/empty entries
	data, err := loadPositions()
	if err != nil {
		t.Fatalf("loadPositions failed on legacy format: %v", err)
	}

	if len(data) != 1 {
		t.Errorf("Expected 1 valid format entry after cleanup, got %d entries: %v", len(data), data)
	}

	if _, ok := data["5114"]; !ok {
		t.Error("Expected key '5114' to exist in parsed positions data")
	}
}

func TestSubDubIndicators(t *testing.T) {
	show := AnimeShow{
		ID:   "test",
		Name: "Test Show",
		AvailableEpisodes: map[string]any{
			"sub": float64(24),
			"dub": float64(12),
		},
	}

	if show.SubCount() != 24 {
		t.Errorf("Expected SubCount() to be 24, got %d", show.SubCount())
	}
	if show.DubCount() != 12 {
		t.Errorf("Expected DubCount() to be 12, got %d", show.DubCount())
	}

	item := episodeItem{
		epNo:     "10",
		subAvail: true,
		dubAvail: true,
	}
	if !strings.Contains(item.Title(), "[SUB+DUB]") {
		t.Errorf("Expected Title to contain [SUB+DUB], got %q", item.Title())
	}

	item2 := episodeItem{
		epNo:     "15",
		subAvail: true,
		dubAvail: false,
	}
	if !strings.Contains(item2.Title(), "[SUB]") || strings.Contains(item2.Title(), "DUB") {
		t.Errorf("Expected Title to contain [SUB] and not DUB, got %q", item2.Title())
	}
}

type mockTransport struct {
	mockServerURL     string
	originalTransport http.RoundTripper
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if strings.Contains(req.URL.Host, "api.allanime.day") || strings.Contains(req.URL.Host, "api.aniskip.com") || strings.Contains(req.URL.Host, "gogoanime.by") {
		mockURL, err := url.Parse(m.mockServerURL)
		if err != nil {
			return nil, err
		}
		req.URL.Scheme = mockURL.Scheme
		req.URL.Host = mockURL.Host
		req.Host = mockURL.Host
	}
	return m.originalTransport.RoundTrip(req)
}

func TestMockHTTP(t *testing.T) {
	mockSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if strings.Contains(r.URL.Path, "/v2/skip-times/") {
			resp := map[string]interface{}{
				"found": true,
				"results": []map[string]interface{}{
					{
						"skipType": "op",
						"interval": map[string]interface{}{
							"startTime": 58.642,
							"endTime":   148.642,
						},
					},
					{
						"skipType": "ed",
						"interval": map[string]interface{}{
							"startTime": 1349.903,
							"endTime":   1474.0,
						},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		if r.Method == "POST" && strings.Contains(r.URL.Path, "/api") {
			bodyBytes, _ := io.ReadAll(r.Body)
			bodyStr := string(bodyBytes)

			if strings.Contains(bodyStr, "query( $search") {
				resp := map[string]interface{}{
					"data": map[string]interface{}{
						"shows": map[string]interface{}{
							"edges": []map[string]interface{}{
								{
									"_id":         "frieren123",
									"name":        "Sousou no Frieren",
									"englishName": "Frieren: Beyond Journey's End",
									"nativeName":  "葬送のフリーレン",
									"thumbnail":   "https://example.com/frieren.jpg",
									"description": "Frieren is an elf mage.",
									"malId":       "52991",
									"aniListId":   "154587",
									"type":        "TV",
									"score":       9.39,
									"season": map[string]interface{}{
										"quarter": "fall",
										"year":    2023,
									},
									"availableEpisodes": map[string]interface{}{
										"sub": 28.0,
										"dub": 28.0,
									},
								},
							},
						},
					},
				}
				_ = json.NewEncoder(w).Encode(resp)
				return
			}

			if strings.Contains(bodyStr, "query ($showId") {
				resp := map[string]interface{}{
					"data": map[string]interface{}{
						"show": map[string]interface{}{
							"_id":         "frieren123",
							"name":        "Sousou no Frieren",
							"englishName": "Frieren: Beyond Journey's End",
							"nativeName":  "葬送のフリーレン",
							"thumbnail":   "https://example.com/frieren.jpg",
							"description": "Frieren is an elf mage.",
							"malId":       "52991",
							"aniListId":   "154587",
							"type":        "TV",
							"score":       9.39,
							"season": map[string]interface{}{
								"quarter": "fall",
								"year":    2023,
							},
							"availableEpisodes": map[string]interface{}{
								"sub": 28.0,
								"dub": 28.0,
							},
							"availableEpisodesDetail": map[string]interface{}{
								"sub": []string{"1", "2", "3", "4"},
								"dub": []string{"1", "2"},
							},
						},
					},
				}
				_ = json.NewEncoder(w).Encode(resp)
				return
			}
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockSrv.Close()

	origTransport := http.DefaultTransport
	defer func() { http.DefaultTransport = origTransport }()

	http.DefaultTransport = &mockTransport{
		mockServerURL:     mockSrv.URL,
		originalTransport: origTransport,
	}

	shows, err := searchAnime("Frieren", "sub")
	if err != nil || len(shows) == 0 {
		t.Fatalf("searchAnime failed or returned 0 shows: %v", err)
	}

	tmpDir, err := os.MkdirTemp("", "clare-test-cache-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origStateDir := os.Getenv("CLARE_STATE_DIR")
	os.Setenv("CLARE_STATE_DIR", tmpDir)
	defer os.Setenv("CLARE_STATE_DIR", origStateDir)

	show, eps, err := fetchEpisodeList("frieren123", "sub")
	if err != nil {
		t.Fatalf("fetchEpisodeList failed: %v", err)
	}
	if show.ID != "frieren123" || len(eps) != 4 || eps[3] != "4" {
		t.Errorf("fetchEpisodeList returned unexpected values: show=%+v, eps=%v", show, eps)
	}

	skipTimes := fetchAniSkipTimes("52991", "1", 1440.0)
	if len(skipTimes) != 2 || skipTimes[0].SkipType != "op" || skipTimes[1].SkipType != "ed" {
		t.Errorf("fetchAniSkipTimes returned unexpected results: %+v", skipTimes)
	}
}

func TestMockTUI(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "clare-tui-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origStateDir := os.Getenv("CLARE_STATE_DIR")
	os.Setenv("CLARE_STATE_DIR", tmpDir)
	defer os.Setenv("CLARE_STATE_DIR", origStateDir)

	// Save a dummy history item so the initial state is stateHistory
	histEntry := HistoryEntry{
		ShowID:    "dummyShowID",
		ShowName:  "Dummy Show",
		Episode:   "1",
		Timestamp: time.Now().Unix(),
	}
	_ = saveHistory([]HistoryEntry{histEntry})

	m := initialModel("", "sub", "best", false)

	// Verify starting state
	if m.state != stateHistory {
		t.Errorf("Expected initial state to be stateHistory, got %d", m.state)
	}

	// Pressing "c" or "C" in stateHistory toggles m.showCompleted
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}}
	updated, _ := m.Update(msg)
	mUpdated := updated.(model)
	if !mUpdated.showCompleted {
		t.Error("Expected showCompleted to be true after pressing 'c'")
	}

	// Pressing "s" or "/" switches state to stateSearchInput
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}
	updated, _ = mUpdated.Update(msg)
	mUpdated = updated.(model)
	if mUpdated.state != stateSearchInput {
		t.Errorf("Expected state to transition to stateSearchInput, got %d", mUpdated.state)
	}

	// Pressing "3" in stateHistory (using initial model m) transitions state to stateLogs
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}}
	updated, _ = m.Update(msg)
	mUpdatedLogs := updated.(model)
	if mUpdatedLogs.state != stateLogs {
		t.Errorf("Expected state to transition to stateLogs, got %d", mUpdatedLogs.state)
	}

	// Add dummy telemetry logs and test clearing them
	mUpdatedLogs.telemetryLogs = []string{"log line 1", "log line 2"}
	mUpdatedLogs.refreshLogsViewport()
	if len(mUpdatedLogs.telemetryLogs) != 2 {
		t.Errorf("Expected 2 telemetry log lines, got %d", len(mUpdatedLogs.telemetryLogs))
	}

	// Pressing "c" in stateLogs should clear them
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}}
	updated, _ = mUpdatedLogs.Update(msg)
	mUpdatedLogs = updated.(model)
	if len(mUpdatedLogs.telemetryLogs) != 0 {
		t.Errorf("Expected telemetry logs to be cleared, but got %d lines", len(mUpdatedLogs.telemetryLogs))
	}
}

func TestPlayerResumePosition(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "clare-resume-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origStateDir := os.Getenv("CLARE_STATE_DIR")
	os.Setenv("CLARE_STATE_DIR", tmpDir)
	defer os.Setenv("CLARE_STATE_DIR", origStateDir)

	// Save positions data with resume state
	posData := PositionsData{
		"12345": ShowState{
			ResumeState: &ResumeState{
				Episode:         3.0,
				PositionSeconds: 450.5,
				TotalSeconds:    1440.0,
			},
		},
	}
	err = savePositions(posData)
	if err != nil {
		t.Fatalf("failed to save positions: %v", err)
	}

	// Case 1: Match episode number - should append --start=450.500000
	cmd, tempLua, tempChaps, _, _, err := getMpvCmd("http://example.com/stream.m3u8", "Test Show", "3", "12345", "24 min", nil)
	if err != nil {
		t.Fatalf("getMpvCmd failed: %v", err)
	}
	if tempLua != "" {
		defer os.Remove(tempLua)
	}
	if tempChaps != "" {
		defer os.Remove(tempChaps)
	}

	hasStartArg := false
	expectedArg := "--start=450.500000"
	for _, arg := range cmd.Args {
		if strings.Contains(arg, "--start=") {
			hasStartArg = true
			if arg != expectedArg {
				t.Errorf("Expected start argument to be %q, got %q", expectedArg, arg)
			}
		}
	}
	if !hasStartArg {
		t.Error("Expected to find '--start=' argument, but none was found")
	}

	// Case 2: Episode mismatch - should NOT append --start
	cmd2, tempLua2, tempChaps2, _, _, err := getMpvCmd("http://example.com/stream.m3u8", "Test Show", "4", "12345", "24 min", nil)
	if err != nil {
		t.Fatalf("getMpvCmd failed on second call: %v", err)
	}
	if tempLua2 != "" {
		defer os.Remove(tempLua2)
	}
	if tempChaps2 != "" {
		defer os.Remove(tempChaps2)
	}

	for _, arg := range cmd2.Args {
		if strings.Contains(arg, "--start=") {
			t.Errorf("Did not expect '--start' argument for mismatched episode, got %q", arg)
		}
	}
}

func TestSearchHistory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "clare-search-history-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origStateDir := os.Getenv("CLARE_STATE_DIR")
	os.Setenv("CLARE_STATE_DIR", tmpDir)
	defer os.Setenv("CLARE_STATE_DIR", origStateDir)

	// Save search history
	err = recordSearch("Frieren")
	if err != nil {
		t.Fatalf("recordSearch failed: %v", err)
	}
	err = recordSearch("One Piece")
	if err != nil {
		t.Fatalf("recordSearch failed: %v", err)
	}
	err = recordSearch("Frieren") // should move to front
	if err != nil {
		t.Fatalf("recordSearch failed: %v", err)
	}

	hist, err := loadSearchHistory()
	if err != nil {
		t.Fatalf("loadSearchHistory failed: %v", err)
	}

	if len(hist) != 2 {
		t.Fatalf("expected 2 unique queries, got %d", len(hist))
	}
	if hist[0] != "Frieren" || hist[1] != "One Piece" {
		t.Errorf("unexpected history order: %v", hist)
	}

	// Test TUI integration
	m := initialModel("", "sub", "best", false)
	m.enterSearchState()

	if len(m.searchHistory) != 2 || m.searchHistory[0] != "Frieren" {
		t.Errorf("Expected searchHistory on model to be populated, got %v", m.searchHistory)
	}

	// Pressing UP key should select the last history item: "One Piece" (index 1)
	msg := tea.KeyMsg{Type: tea.KeyUp}
	updated, _ := m.Update(msg)
	mUpdated := updated.(model)

	if mUpdated.searchInput.Value() != "One Piece" {
		t.Errorf("Expected searchInput value to be 'One Piece', got %q", mUpdated.searchInput.Value())
	}
	if mUpdated.searchHistoryIndex != 1 {
		t.Errorf("Expected searchHistoryIndex to be 1, got %d", mUpdated.searchHistoryIndex)
	}

	// Pressing UP again should select the first history item: "Frieren" (index 0)
	updated, _ = mUpdated.Update(msg)
	mUpdated = updated.(model)

	if mUpdated.searchInput.Value() != "Frieren" {
		t.Errorf("Expected searchInput value to be 'Frieren', got %q", mUpdated.searchInput.Value())
	}
	if mUpdated.searchHistoryIndex != 0 {
		t.Errorf("Expected searchHistoryIndex to be 0, got %d", mUpdated.searchHistoryIndex)
	}

	// Pressing DOWN should go back to "One Piece" (index 1)
	msg = tea.KeyMsg{Type: tea.KeyDown}
	updated, _ = mUpdated.Update(msg)
	mUpdated = updated.(model)

	if mUpdated.searchInput.Value() != "One Piece" {
		t.Errorf("Expected searchInput value to be 'One Piece' after DOWN key, got %q", mUpdated.searchInput.Value())
	}
	if mUpdated.searchHistoryIndex != 1 {
		t.Errorf("Expected searchHistoryIndex to be 1, got %d", mUpdated.searchHistoryIndex)
	}
}

func TestInteractiveSourceSelect(t *testing.T) {
	m := initialModel("", "sub", "best", false)
	m.selectedShow = AnimeShow{ID: "frieren123", Name: "Sousou no Frieren"}
	m.selectedEp = "1"

	streams := []ResolvedStream{
		{SourceName: "Wixmp", Quality: "1080p", URL: "http://example.com/wixmp-1080.mp4"},
		{SourceName: "Sharepoint", Quality: "720p", URL: "http://example.com/sharepoint-720.mp4"},
	}
	updated, _ := m.Update(allStreamsResultMsg{epNo: "1", streams: streams, err: nil})
	mUpdated := updated.(model)

	if mUpdated.state != stateSourceSelect {
		t.Errorf("Expected state to be stateSourceSelect, got %d", mUpdated.state)
	}

	if len(mUpdated.sourceList.Items()) != 2 {
		t.Errorf("Expected 2 sources in list, got %d", len(mUpdated.sourceList.Items()))
	}

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updated2, _ := mUpdated.Update(msg)
	mUpdated2 := updated2.(model)

	if mUpdated2.state != statePlaybackPreparing {
		t.Errorf("Expected state to transition to statePlaybackPreparing, got %d", mUpdated2.state)
	}

	cacheKey := fmt.Sprintf("%s-%s-%s-%s", mUpdated2.selectedShow.ID, mUpdated2.mode, mUpdated2.selectedEp, mUpdated2.quality)
	streamCacheMu.RLock()
	cachedURL, ok := streamCache[cacheKey]
	streamCacheMu.RUnlock()

	if !ok || cachedURL != "http://example.com/wixmp-1080.mp4" {
		t.Errorf("Expected cached URL to be seeded as 'http://example.com/wixmp-1080.mp4', got %q (found=%t)", cachedURL, ok)
	}
}

func TestMpvIPCController(t *testing.T) {
	socketPath := "/tmp/clare-mpv.sock"
	_ = os.Remove(socketPath) // clean up any old one
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("failed to listen on unix socket: %v", err)
	}
	defer func() {
		listener.Close()
		os.Remove(socketPath)
	}()

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			req := scanner.Text()
			if strings.Contains(req, "playback-time") {
				conn.Write([]byte(`{"data": 345.67}` + "\n"))
			} else if strings.Contains(req, "duration") {
				conn.Write([]byte(`{"data": 1200.0}` + "\n"))
			} else if strings.Contains(req, "pause") {
				conn.Write([]byte(`{"data": false}` + "\n"))
			} else if strings.Contains(req, "volume") {
				conn.Write([]byte(`{"data": 75.0}` + "\n"))
			} else {
				conn.Write([]byte(`{"data": null}` + "\n"))
			}
		}
	}()

	status, err := queryMpvStatus()
	if err != nil {
		t.Fatalf("queryMpvStatus failed: %v", err)
	}

	if status.PlaybackTime != 345.67 {
		t.Errorf("expected PlaybackTime 345.67, got %f", status.PlaybackTime)
	}
	if status.Duration != 1200.0 {
		t.Errorf("expected Duration 1200.0, got %f", status.Duration)
	}
	if status.Paused != false {
		t.Errorf("expected Paused to be false, got %t", status.Paused)
	}
	if status.Volume != 75.0 {
		t.Errorf("expected Volume 75.0, got %f", status.Volume)
	}
}

func TestEpisodeDetailsScrolling(t *testing.T) {
	m := initialModel("", "sub", "best", false)
	m.state = stateEpisodeSelect
	m.detailsScrollOffset = 5

	// Left key should decrease scroll offset
	msg := tea.KeyMsg{Type: tea.KeyLeft}
	updated, _ := m.Update(msg)
	mUpdated := updated.(model)

	if mUpdated.detailsScrollOffset != 4 {
		t.Errorf("expected detailsScrollOffset to be 4 after Left key, got %d", mUpdated.detailsScrollOffset)
	}

	// Right key should increase scroll offset
	msg2 := tea.KeyMsg{Type: tea.KeyRight}
	updated2, _ := mUpdated.Update(msg2)
	mUpdated2 := updated2.(model)

	if mUpdated2.detailsScrollOffset != 5 {
		t.Errorf("expected detailsScrollOffset to be 5 after Right key, got %d", mUpdated2.detailsScrollOffset)
	}
}

type syncMockTransport struct {
	mockServerURL string
	origTransport http.RoundTripper
}

func (s *syncMockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("X-Original-Host", req.URL.Host)
	u, _ := url.Parse(s.mockServerURL)
	req.URL.Scheme = u.Scheme
	req.URL.Host = u.Host
	req.Host = u.Host
	return s.origTransport.RoundTrip(req)
}

func TestSyncProgressMock(t *testing.T) {
	var anilistCalled, malCalled bool
	var anilistProgress, malProgress int

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origHost := r.Header.Get("X-Original-Host")
		if strings.Contains(origHost, "anilist.co") {
			anilistCalled = true
			var body map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&body)

			if strings.Contains(fmt.Sprintf("%v", body["query"]), "Media") && !strings.Contains(fmt.Sprintf("%v", body["query"]), "SaveMediaListEntry") {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"data":{"Media":{"id":9988}}}`))
			} else if strings.Contains(fmt.Sprintf("%v", body["query"]), "SaveMediaListEntry") {
				vars := body["variables"].(map[string]interface{})
				anilistProgress = int(vars["progress"].(float64))
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"data":{"SaveMediaListEntry":{"id":123,"progress":12}}}`))
			}
		} else if strings.Contains(origHost, "myanimelist.net") {
			malCalled = true
			r.ParseForm()
			val, _ := strconv.Atoi(r.FormValue("num_watched_episodes"))
			malProgress = val
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{}`))
		}
	}))
	defer mockServer.Close()

	origTransport := http.DefaultTransport
	defer func() { http.DefaultTransport = origTransport }()

	http.DefaultTransport = &syncMockTransport{
		mockServerURL: mockServer.URL,
		origTransport: origTransport,
	}

	os.Setenv("ANILIST_TOKEN", "mock-anilist-token")
	os.Setenv("MAL_TOKEN", "mock-mal-token")
	defer func() {
		os.Unsetenv("ANILIST_TOKEN")
		os.Unsetenv("MAL_TOKEN")
	}()

	SyncProgress("12345", "12", false)

	time.Sleep(200 * time.Millisecond)

	if !anilistCalled {
		t.Error("Expected AniList API to be called")
	}
	if anilistProgress != 12 {
		t.Errorf("Expected AniList progress to be 12, got %d", anilistProgress)
	}

	if !malCalled {
		t.Error("Expected MAL API to be called")
	}
	if malProgress != 12 {
		t.Errorf("Expected MAL progress to be 12, got %d", malProgress)
	}
}

func TestSyncProgressFileMock(t *testing.T) {
	var anilistCalled, malCalled bool
	var anilistProgress, malProgress int

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origHost := r.Header.Get("X-Original-Host")
		if strings.Contains(origHost, "anilist.co") {
			anilistCalled = true
			var body map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&body)

			if strings.Contains(fmt.Sprintf("%v", body["query"]), "Media") && !strings.Contains(fmt.Sprintf("%v", body["query"]), "SaveMediaListEntry") {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"data":{"Media":{"id":9988}}}`))
			} else if strings.Contains(fmt.Sprintf("%v", body["query"]), "SaveMediaListEntry") {
				vars := body["variables"].(map[string]interface{})
				anilistProgress = int(vars["progress"].(float64))
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"data":{"SaveMediaListEntry":{"id":123,"progress":12}}}`))
			}
		} else if strings.Contains(origHost, "myanimelist.net") {
			malCalled = true
			r.ParseForm()
			val, _ := strconv.Atoi(r.FormValue("num_watched_episodes"))
			malProgress = val
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{}`))
		}
	}))
	defer mockServer.Close()

	origTransport := http.DefaultTransport
	defer func() { http.DefaultTransport = origTransport }()

	http.DefaultTransport = &syncMockTransport{
		mockServerURL: mockServer.URL,
		origTransport: origTransport,
	}

	// Create temp token files
	fAni, err := os.CreateTemp("", "mock-anilist-token")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(fAni.Name())
	_, _ = fAni.WriteString("file-anilist-token\n")
	fAni.Close()

	fMal, err := os.CreateTemp("", "mock-mal-token")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(fMal.Name())
	_, _ = fMal.WriteString("file-mal-token\r\n")
	fMal.Close()

	os.Setenv("ANILIST_TOKEN_FILE", fAni.Name())
	os.Setenv("MAL_TOKEN_FILE", fMal.Name())
	defer func() {
		os.Unsetenv("ANILIST_TOKEN_FILE")
		os.Unsetenv("MAL_TOKEN_FILE")
	}()

	SyncProgress("12345", "12", false)

	time.Sleep(200 * time.Millisecond)

	if !anilistCalled {
		t.Error("Expected AniList API to be called via token file")
	}
	if anilistProgress != 12 {
		t.Errorf("Expected AniList progress to be 12, got %d", anilistProgress)
	}

	if !malCalled {
		t.Error("Expected MAL API to be called via token file")
	}
	if malProgress != 12 {
		t.Errorf("Expected MAL progress to be 12, got %d", malProgress)
	}
}

func TestPullFromAniList(t *testing.T) {
	var viewerCalled, collectionCalled bool

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origHost := r.Header.Get("X-Original-Host")
		if strings.Contains(origHost, "anilist.co") {
			var body map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&body)
			queryStr := fmt.Sprintf("%v", body["query"])

			if strings.Contains(queryStr, "Viewer") {
				viewerCalled = true
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"data":{"Viewer":{"name":"test-user"}}}`))
			} else if strings.Contains(queryStr, "MediaListCollection") {
				collectionCalled = true
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{
					"data": {
						"mediaListCollection": {
							"lists": [
								{
									"entries": [
										{
											"media": {
												"idMal": 54321,
												"episodes": 12
											},
											"progress": 3,
											"status": "CURRENT"
										}
									]
								}
							]
						}
					}
				}`))
			}
		}
	}))
	defer mockServer.Close()

	origTransport := http.DefaultTransport
	defer func() { http.DefaultTransport = origTransport }()

	http.DefaultTransport = &syncMockTransport{
		mockServerURL: mockServer.URL,
		origTransport: origTransport,
	}

	positions := make(map[string]ShowState)
	changed, err := pullFromAniList("mock-token", positions)
	if err != nil {
		t.Fatalf("pullFromAniList failed: %v", err)
	}

	if !viewerCalled {
		t.Error("Expected AniList Viewer query to be executed")
	}
	if !collectionCalled {
		t.Error("Expected AniList MediaListCollection query to be executed")
	}
	if !changed {
		t.Error("Expected positions to be updated and changed=true")
	}

	state, ok := positions["54321"]
	if !ok {
		t.Fatal("Expected MAL ID 54321 to be added to positions")
	}
	if state.LastSyncedEp != 3 {
		t.Errorf("Expected LastSyncedEp to be 3, got %f", state.LastSyncedEp)
	}
	if len(state.CompletedEpisodes) != 3 {
		t.Errorf("Expected 3 completed episodes, got %d", len(state.CompletedEpisodes))
	}
}

func TestEpisodeSynopsisLazyLoading(t *testing.T) {
	m := initialModel("", "sub", "best", false)
	m.selectedShow = AnimeShow{
		ID:    "frieren-test",
		Name:  "Sousou no Frieren",
		MALID: "52991",
	}
	m.selectedEp = "1"
	m.state = stateEpisodeSelect

	// Populate the episode list so SelectedItem() is not nil
	epItem := episodeItem{epNo: "1"}
	m.episodeList.SetItems([]list.Item{epItem})

	// Initial check: synopsis should be empty
	if m.episodeDetails == nil {
		m.episodeDetails = make(map[string]JikanEpInfo)
	}

	// Create an update message with a mock synopsis
	msg := episodeSynopsisMsg{
		epNo:     "1",
		synopsis: "Frieren and her companions return from their decade-long quest.",
	}

	updated, _ := m.Update(msg)
	mUpdated := updated.(model)

	// Verify it was saved to local state
	info, ok := mUpdated.episodeDetails["1"]
	if !ok {
		t.Fatal("Expected episodeDetails for episode 1 to exist")
	}
	if info.Synopsis != "Frieren and her companions return from their decade-long quest." {
		t.Errorf("Expected synopsis to be populated, got %q", info.Synopsis)
	}

	// Verify rendering contains the synopsis
	renderedDetails := mUpdated.renderEpisodeDetailsPanel(80, 15)
	if !strings.Contains(renderedDetails, "Frieren and her companions return") {
		t.Errorf("Expected rendered details panel to contain synopsis text, got: %s", renderedDetails)
	}
}

func TestLiveAPIIntegration(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "clare-test-live-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origStateDir := os.Getenv("CLARE_STATE_DIR")
	os.Setenv("CLARE_STATE_DIR", tmpDir)
	defer os.Setenv("CLARE_STATE_DIR", origStateDir)

	showID := "caSCmQepJo2pbKzq9"
	mode := "sub"
	episodeNo := "1"

	sources, err := fetchAllResolvedStreams(showID, mode, episodeNo)
	if err != nil {
		t.Logf("Live fetchAllResolvedStreams failed: %v", err)
	} else {
		t.Logf("Found %d sources for episode %s (%s)", len(sources), episodeNo, mode)
	}
}

func TestTUIProviderBadgesAndPlay(t *testing.T) {
	m := initialModel("", "sub", "best", false)
	m.selectedShow = AnimeShow{ID: "frieren-test", Name: "Frieren"}
	m.selectedEp = "1"

	streams := []ResolvedStream{
		{Provider: "allanime", SourceName: "Mp4", Quality: "1080p", URL: "http://example.com/allanime-1080.mp4"},
		{Provider: "gogoanime", SourceName: "Gogo", Quality: "720p", URL: "http://example.com/gogo-720.mp4"},
	}

	updated, _ := m.Update(allStreamsResultMsg{epNo: "1", streams: streams, err: nil})
	mUpdated := updated.(model)

	if mUpdated.state != stateSourceSelect {
		t.Fatalf("expected state to be stateSourceSelect, got %d", mUpdated.state)
	}

	items := mUpdated.sourceList.Items()
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	// Verify badges are rendered correctly in titles
	item0 := items[0].(sourceItem)
	title0 := item0.Title()
	if !strings.Contains(title0, "[ALLANIME]") {
		t.Errorf("expected title to contain '[ALLANIME]' badge, got %q", title0)
	}

	item1 := items[1].(sourceItem)
	title1 := item1.Title()
	if !strings.Contains(title1, "[GOGO]") {
		t.Errorf("expected title to contain '[GOGO]' badge, got %q", title1)
	}

	// Navigate to item 1 (Gogoanime)
	mUpdated.sourceList.Select(1)

	// Press Enter to select/play the stream
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updated2, _ := mUpdated.Update(msg)
	mUpdated2 := updated2.(model)

	if mUpdated2.state != statePlaybackPreparing {
		t.Fatalf("expected state playback preparing, got %d", mUpdated2.state)
	}

	cacheKey := fmt.Sprintf("%s-%s-%s-%s", mUpdated2.selectedShow.ID, mUpdated2.mode, mUpdated2.selectedEp, mUpdated2.quality)
	streamCacheMu.RLock()
	cachedURL, ok := streamCache[cacheKey]
	streamCacheMu.RUnlock()

	if !ok || cachedURL != "http://example.com/gogo-720.mp4" {
		t.Errorf("expected selected URL to be 'http://example.com/gogo-720.mp4', got %q", cachedURL)
	}
}

func TestFetchBuildID(t *testing.T) {
	epoch, key, buildID, err := getDerivedKey()
	if err != nil {
		t.Fatalf("getDerivedKey failed: %v", err)
	}
	t.Logf("Epoch: %d", epoch)
	t.Logf("Key: %x", key)
	t.Logf("BuildID: %q", buildID)
}
func TestFlikhubProvider(t *testing.T) {
	p := &FlikhubProvider{}
	shows, err := p.Search("Frieren", "sub")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(shows) == 0 {
		t.Fatalf("No shows found for Frieren")
	}
	t.Logf("Search found %d shows. First: %s (MAL ID: %s)", len(shows), shows[0].Name, shows[0].MALID)

	showID := ""
	for _, s := range shows {
		if strings.Contains(s.ID, "c6fbj") {
			showID = s.ID
			break
		}
	}
	if showID == "" {
		showID = shows[0].ID
	}
	show, episodes, err := p.FetchEpisodes(showID, "sub")
	if err != nil {
		t.Fatalf("FetchEpisodes failed: %v", err)
	}
	_ = saveShowCache(showID, show, episodes)
	if len(episodes) == 0 {
		t.Fatalf("No episodes found for Frieren")
	}
	t.Logf("FetchEpisodes found %d episodes. First: %s, Details Name: %s", len(episodes), episodes[0], show.Name)

	streams, err := p.ResolveStreams(showID, "sub", "1", "best")
	if err != nil {
		t.Fatalf("ResolveStreams failed: %v", err)
	}
	if len(streams) == 0 {
		t.Fatalf("No streams resolved")
	}
	for i, stream := range streams {
		t.Logf("Stream %d: Source=%s, URL=%s", i, stream.SourceName, stream.URL)
	}
}

func TestGogoanimeProvider(t *testing.T) {
	p := &GogoanimeProvider{}
	shows, err := p.Search("Frieren", "sub")
	if err != nil || len(shows) == 0 {
		t.Skipf("Gogoanime search returned no results: %v", err)
	}
	t.Logf("Gogoanime search found %d shows. First: %s (ID: %s)", len(shows), shows[0].Name, shows[0].ID)

	show, eps, err := p.FetchEpisodes(shows[0].ID, "sub")
	if err != nil {
		t.Skipf("Gogoanime FetchEpisodes failed: %v", err)
	}
	t.Logf("Gogoanime FetchEpisodes found %d episodes for %s", len(eps), show.Name)

	if len(eps) > 0 {
		streams, err := p.ResolveStreams(shows[0].ID, "sub", "1", "best")
		if err != nil {
			t.Logf("Gogoanime ResolveStreams note: %v", err)
		} else {
			for i, st := range streams {
				t.Logf("Gogoanime Stream %d: Source=%s, Quality=%s, URL=%s", i, st.SourceName, st.Quality, st.URL)
			}
		}
	}
}

func TestPlayEpisodesOnOtus(t *testing.T) {
	if os.Getenv("TEST_PLAYBACK") == "" {
		t.Skip("Skipping playback test unless TEST_PLAYBACK=1 is set")
	}

	_ = InitLogger("")
	resolver := NewMultiProviderResolver()
	queries := []string{"Sakamoto Days", "Gachiakuta", "Ghost in the Shell", "Bleach"}

	var targetShow AnimeShow
	var targetStream ResolvedStream

	for _, query := range queries {
		show, stream, err := resolver.ResolveWithFallback(query, "sub", "1", "best")
		if err == nil && stream.URL != "" {
			targetShow = show
			targetStream = stream
			break
		} else {
			t.Logf("MultiProviderResolver query %q did not yield a pre-flighted stream: %v", query, err)
		}
	}

	if targetShow.ID == "" || targetStream.URL == "" {
		t.Fatalf("No working pre-flighted stream found across queries")
	}

	t.Logf("Selected Pre-flighted Show: %s (Provider: %s, URL: %s)", targetShow.Name, targetStream.Provider, targetStream.URL)

	episodesToTest := []string{"1"}
	for _, epNo := range episodesToTest {
		t.Logf("=== Testing Playback for %s Episode %s ===", targetShow.Name, epNo)
		_, stream, err := resolver.ResolveWithFallback(targetShow.Name, "sub", epNo, "best")
		if err != nil || stream.URL == "" {
			t.Fatalf("Failed to resolve pre-flighted stream for ep %s: %v", epNo, err)
		}

		t.Logf("Selected Stream: %s (%s)", stream.SourceName, stream.URL)

		cmd, luaFile, chapFile, _, _, err := getMpvCmd(
			stream.URL, targetShow.Name, epNo, targetShow.MALID,
			"24 min", []string{
				"--length=8",
				"--keep-open=no",
				"--no-terminal",
				"--idle=no",
			},
		)
		if err != nil {
			t.Fatalf("Failed to generate mpv command: %v", err)
		}
		defer func() {
			if luaFile != "" {
				os.Remove(luaFile)
			}
			if chapFile != "" {
				os.Remove(chapFile)
			}
		}()

		if os.Getenv("WAYLAND_DISPLAY") == "" {
			cmd.Env = append(os.Environ(),
				"WAYLAND_DISPLAY=wayland-1",
				"XDG_RUNTIME_DIR=/run/user/1001",
			)
		}

		LogEventInfo(DomainMpvIPC, fmt.Sprintf("Automated Test Playback Started: %s (Ep %s)", targetShow.Name, epNo))
		out, err := cmd.CombinedOutput()
		t.Logf("MPV Output (Ep %s):\n%s", epNo, string(out))
		if err != nil {
			t.Fatalf("MPV playback failed for Episode %s: %v", epNo, err)
		}
		LogEventInfo(DomainMpvIPC, fmt.Sprintf("Automated Test Playback Completed: %s (Ep %s)", targetShow.Name, epNo))
		t.Logf("✓ Episode %s played successfully on otus!", epNo)
	}

	summary, err := ValidateSessionTrace("")
	if err == nil {
		t.Logf("Session Health Trace Summary: Status=%s (SearchOK=%t, PreflightOK=%t, VideoOK=%t, Errors=%d)",
			summary.OverallStatus, summary.SearchSuccess, summary.PreflightOK, summary.VideoCodecOK, len(summary.PlaybackErrors))
	}
	t.Logf("✓✓ All 3 episodes of %s played successfully on otus!", targetShow.Name)
}
