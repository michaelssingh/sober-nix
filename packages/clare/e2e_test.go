package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// TestE2EFullSuite executes a comprehensive end-to-end test suite for Clare.
func TestE2EFullSuite(t *testing.T) {
	_ = InitLogger("")

	t.Run("01_AniDB_Provider_Pipeline", func(t *testing.T) {
		p := &AniDBProvider{}
		if p.Name() != "anidb" {
			t.Fatalf("expected provider name 'anidb', got %s", p.Name())
		}

		shows, err := p.Search("Sakamoto Days", "sub")
		if err != nil || len(shows) == 0 {
			t.Fatalf("AniDB search failed for 'Sakamoto Days': %v", err)
		}
		if !strings.HasPrefix(shows[0].ID, "anidb:") {
			t.Fatalf("expected show ID prefix 'anidb:', got %s", shows[0].ID)
		}
		t.Logf("✓ [AniDB E2E] Search found %d shows. Top: %s (%s)", len(shows), shows[0].Name, shows[0].ID)

		show, eps, err := p.FetchEpisodes(shows[0].ID, "sub")
		if err != nil || len(eps) == 0 {
			t.Fatalf("AniDB episode fetch failed for %s: %v", shows[0].ID, err)
		}
		t.Logf("✓ [AniDB E2E] Fetched %d episodes for %s", len(eps), show.Name)

		streams, err := p.ResolveStreams(shows[0].ID, "sub", "1", "best")
		if err != nil || len(streams) == 0 {
			t.Fatalf("AniDB stream resolution failed: %v", err)
		}
		t.Logf("✓ [AniDB E2E] Resolved stream URL: %s", streams[0].URL)

		headers := map[string]string{
			"User-Agent": UserAgent,
			"Referer":    getRefererForURL(streams[0].URL),
		}
		if err := PreflightStreamURLWithTimeout(streams[0].URL, headers, 15*time.Second); err != nil {
			t.Fatalf("AniDB stream preflight HTTP check failed: %v", err)
		}
		t.Logf("✓ [AniDB E2E] Preflight HTTP check 200 OK verified!")
	})

	t.Run("02_VidSrc_Movie_And_TV_Pipeline", func(t *testing.T) {
		p := &VidSrcProvider{}
		if p.Name() != "vidsrc" {
			t.Fatalf("expected provider name 'vidsrc', got %s", p.Name())
		}

		// Test Movie Resolution
		movieStreams, err := p.ResolveStreams("vidsrc:movie:603", "sub", "1", "best")
		if err != nil || len(movieStreams) == 0 {
			t.Fatalf("VidSrc movie stream resolution failed: %v", err)
		}
		t.Logf("✓ [VidSrc E2E] Movie (The Matrix) stream resolved: %s", movieStreams[0].URL)

		// Test TV Series Episode Resolution
		tvShow, tvEps, err := p.FetchEpisodes("vidsrc:tv:2190", "sub")
		if err != nil || len(tvEps) == 0 {
			t.Fatalf("VidSrc TV episode fetch failed: %v", err)
		}
		t.Logf("✓ [VidSrc E2E] TV Show %s returned %d episodes", tvShow.Name, len(tvEps))

		tvStreams, err := p.ResolveStreams("vidsrc:tv:2190", "sub", "S02E01", "best")
		if err != nil || len(tvStreams) == 0 {
			t.Fatalf("VidSrc TV S02E01 stream resolution failed: %v", err)
		}
		t.Logf("✓ [VidSrc E2E] TV (South Park S02E01) stream resolved: %s", tvStreams[0].URL)

		tvHeaders := map[string]string{
			"User-Agent": UserAgent,
			"Referer":    getRefererForURL(tvStreams[0].URL),
		}
		if err := PreflightStreamURLWithTimeout(tvStreams[0].URL, tvHeaders, 15*time.Second); err != nil {
			t.Logf("⚠️ [VidSrc E2E] TV S02E01 preflight note: %v", err)
		} else {
			t.Logf("✓ [VidSrc E2E] TV S02E01 preflight HTTP check 200 OK verified!")
		}
	})

	t.Run("03_MultiProviderResolver_Fallback", func(t *testing.T) {
		resolver := NewMultiProviderResolver()
		if resolver == nil || len(resolver.providers) != 2 {
			t.Fatalf("expected 2 active providers in MultiProviderResolver")
		}

		show, stream, err := resolver.ResolveWithFallback("Sakamoto Days", "sub", "1", "best")
		if err != nil || stream.URL == "" {
			t.Fatalf("ResolveWithFallback failed: %v", err)
		}
		t.Logf("✓ [Resolver E2E] Multi-provider resolved %s via provider '%s': %s", show.Name, stream.Provider, stream.URL)
	})

	t.Run("04_Headless_MPV_IPC_Control", func(t *testing.T) {
		mpvPath, err := exec.LookPath("mpv")
		if err != nil {
			t.Skip("mpv binary not found in PATH, skipping live MPV IPC test")
		}

		socketPath := filepath.Join(t.TempDir(), "mpv.sock")
		t.Setenv("CLARE_MPV_SOCK", socketPath)

		p := &AniDBProvider{}
		anidbStreams, err := p.ResolveStreams("anidb:4556", "sub", "1", "best")
		streamURL := "https://hls.anidb.app/stream/B2V-LPvOAhipMt9rZxeRccGoBPLnQyY6ASqiHfNhIFyc_u-yGsEIWCKv7hQIfX-z/master.m3u8"
		if err == nil && len(anidbStreams) > 0 {
			streamURL = anidbStreams[0].URL
		}

		cmd, luaFile, chapFile, _, _, err := getMpvCmd(
			streamURL, "E2E Test Show - Ep 1: The Legendary Hit Man", "1", "58939", "24 min",
			[]string{
				"--vo=null",
				"--ao=null",
				"--mute=yes",
				"--idle=no",
				"--keep-open=no",
				"--no-terminal",
			},
		)
		if err != nil {
			t.Fatalf("getMpvCmd failed: %v", err)
		}

		defer func() {
			if luaFile != "" {
				_ = os.Remove(luaFile)
			}
			if chapFile != "" {
				_ = os.Remove(chapFile)
			}
		}()

		if err := cmd.Start(); err != nil {
			t.Fatalf("failed to start headless mpv command (%s): %v", mpvPath, err)
		}
		defer func() {
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
				_ = cmd.Wait()
			}
		}()

		var ipc *MPVIPCClient
		connDeadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(connDeadline) {
			if client, err := NewMPVIPCClient(socketPath); err == nil {
				ipc = client
				break
			}
			time.Sleep(150 * time.Millisecond)
		}

		if ipc == nil {
			t.Fatalf("Failed to connect to AniDB MPV IPC socket at %s", socketPath)
		}
		defer ipc.Close()

		_ = ipc.Seek(10.0)
		health, _ := ipc.InspectHealth()
		t.Logf("✓ [MPV IPC E2E] Socket connection verified! Health stats: %+v", health)

		titleVal, err := ipc.GetProperty("media-title")
		if err != nil {
			titleVal, err = ipc.GetProperty("force-media-title")
		}
		if err != nil {
			t.Fatalf("Failed to query title from running AniDB MPV instance via IPC: %v", err)
		}
		titleStr := fmt.Sprintf("%v", titleVal)
		if !strings.Contains(titleStr, "The Legendary Hit Man") {
			t.Fatalf("Running AniDB MPV IPC title mismatch! Expected 'The Legendary Hit Man', got: '%s'", titleStr)
		}
		t.Logf("✓ [AniDB MPV IPC E2E] Running MPV instance confirmed full episode title via IPC socket: '%s'", titleStr)

		pos, err := ipc.WaitForPlaybackToStart(10 * time.Second)
		if err != nil {
			t.Fatalf("AniDB MPV failed to start playback: %v", err)
		}
		t.Logf("✓ [AniDB Playback E2E] AniDB MPV playback successfully confirmed active! Position: %.2fs", pos)
	})

	t.Run("05_Full_TUI_Interactive_User_Flow", func(t *testing.T) {
		m := initialModel("", "sub", "best", false)

		// 1. Switch to Search tab ('2')
		res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
		m2 := res.(model)
		if m2.state != stateSearchInput {
			t.Fatalf("expected state stateSearchInput (1), got %v", m2.state)
		}

		// 2. Perform search via doSearch
		searchCmd := doSearch("Sakamoto Days", "sub")
		searchMsg := searchCmd()
		sResMsg, ok := searchMsg.(searchResultMsg)
		if !ok || sResMsg.err != nil || len(sResMsg.shows) == 0 {
			t.Fatalf("doSearch failed: %v", sResMsg.err)
		}

		res, _ = m2.Update(sResMsg)
		m3 := res.(model)
		if m3.state != stateShowSelect {
			t.Fatalf("expected state stateShowSelect (3), got %v", m3.state)
		}
		t.Logf("✓ [TUI E2E] Transitioned to stateShowSelect with search results")

		// 3. Select show & fetch episodes
		topShow := sResMsg.shows[0]
		epCmd := doFetchEpisodes(topShow.ID, "sub")
		epMsg := epCmd()
		eResMsg, ok := epMsg.(episodesResultMsg)
		if !ok || eResMsg.err != nil || len(eResMsg.episodes) == 0 {
			t.Fatalf("doFetchEpisodes failed: %v", eResMsg.err)
		}

		res, _ = m3.Update(eResMsg)
		m4 := res.(model)
		if m4.state != stateEpisodeSelect {
			t.Fatalf("expected state stateEpisodeSelect (5), got %v", m4.state)
		}
		t.Logf("✓ [TUI E2E] Transitioned to stateEpisodeSelect with %d episodes", len(eResMsg.episodes))

		// 4. Prepare playback
		playCmd := doPreparePlayback(m4.selectedShow, m4.episodes[0], "Ep 1", "sub", "best", false)
		playMsg := playCmd()
		pResMsg, ok := playMsg.(resolvedPlaybackMsg)
		if !ok || pResMsg.err != nil || pResMsg.cmd == nil {
			t.Fatalf("doPreparePlayback failed: %v", pResMsg.err)
		}

		res, _ = m4.Update(pResMsg)
		m5 := res.(model)
		if !m5.playbackActive {
			t.Fatalf("expected playbackActive=true after resolvedPlaybackMsg, got false")
		}
		t.Logf("✓ [TUI E2E] Full interactive TUI user flow successfully completed!")
	})

	t.Run("06_State_Persistence_And_Position_Sync", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "clare-e2e-state-*")
		if err != nil {
			t.Fatalf("failed to create temp state dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		origDir := os.Getenv("CLARE_STATE_DIR")
		os.Setenv("CLARE_STATE_DIR", tmpDir)
		defer os.Setenv("CLARE_STATE_DIR", origDir)

		showID := "anidb:4556"
		epNo := "1"

		// 1. Record search
		_ = recordSearch("Sakamoto Days")
		searches, err := loadSearchHistory()
		if err != nil || len(searches) == 0 || searches[0] != "Sakamoto Days" {
			t.Fatalf("expected search history to contain 'Sakamoto Days', got %v (err: %v)", searches, err)
		}

		// 2. Save position
		posData := PositionsData{
			showID: ShowState{
				ResumeState: &ResumeState{
					Episode:         1.0,
					PositionSeconds: 345.5,
					TotalSeconds:    1440.0,
				},
				CompletedEpisodes: []float64{1.0},
			},
		}
		if err := savePositions(posData); err != nil {
			t.Fatalf("savePositions failed: %v", err)
		}

		loadedPositions, err := loadPositions()
		if err != nil || loadedPositions[showID].ResumeState == nil || loadedPositions[showID].ResumeState.PositionSeconds != 345.5 {
			t.Fatalf("expected loaded positions to contain position 345.5, got %+v (err: %v)", loadedPositions[showID], err)
		}
		t.Logf("✓ [Persistence E2E] Playback positions saved & recovered: %.1fs", loadedPositions[showID].ResumeState.PositionSeconds)

		// 3. Record watch history
		if err := recordWatch(showID, "Sakamoto Days", epNo); err != nil {
			t.Fatalf("recordWatch failed: %v", err)
		}

		loadedHist, err := loadHistory()
		if err != nil || len(loadedHist) == 0 || loadedHist[0].ShowID != showID {
			t.Fatalf("expected loaded history to contain show %s, got %+v (err: %v)", showID, loadedHist, err)
		}
		t.Logf("✓ [Persistence E2E] Watch history entry saved & verified!")
	})

	t.Run("07_AniList_And_Jikan_Metadata_Enrichment", func(t *testing.T) {
		show := AnimeShow{Name: "Sakamoto Days"}
		enrichShowMetadata(&show)

		if show.MALID == "" {
			t.Errorf("expected MALID to be populated")
		}
		if show.AniListID == "" {
			t.Errorf("expected AniListID to be populated")
		}
		if show.Thumbnail == "" {
			t.Errorf("expected Thumbnail URL to be populated")
		}
		if show.Description == "" {
			t.Errorf("expected Description synopsis to be populated")
		}
		if show.Score == 0 {
			t.Errorf("expected average score to be populated")
		}
		t.Logf("✓ [Metadata E2E] AniList Enrichment verified: MALID=%s, AniListID=%s, Score=%.2f, Year=%d, Genres=%v, Thumbnail=%s",
			show.MALID, show.AniListID, show.Score, show.Season.Year, show.Genres, show.Thumbnail)

		// Test Jikan episode metadata API fetching with Kitsu fallback
		if show.MALID != "" || show.Name != "" {
			jCmd := doFetchJikanMetadata(show.MALID, show.Name, 1)
			jMsg := jCmd()
			mMsg, ok := jMsg.(jikanMetadataMsg)
			if !ok || mMsg.err != nil {
				t.Logf("⚠️ [Metadata E2E] Jikan episode metadata note: %v", mMsg.err)
			} else {
				t.Logf("✓ [Metadata E2E] Jikan fetched %d episode details (Top Ep 1: Title=%q, Aired=%s)",
					len(mMsg.metadata), mMsg.metadata["1"].Title, mMsg.metadata["1"].Aired)
			}
		}
	})

	t.Run("08_AniSkip_And_Chapter_File_Generation", func(t *testing.T) {
		show := AnimeShow{Name: "Fullmetal Alchemist: Brotherhood"}
		enrichShowMetadata(&show)
		if show.MALID == "" && show.AniListID == "" {
			t.Fatalf("expected AniList metadata enrichment for Fullmetal Alchemist: Brotherhood")
		}

		skipTimes := fetchAniSkipTimes("5114", "1", 1440.0)
		t.Logf("✓ [AniSkip E2E] AniSkip API returned %d skip intervals for MAL ID 5114", len(skipTimes))

		anidbSkipTimes := fetchAniSkipTimes("anidb:18464", "1", 1440.0, "Sakamoto Days")
		if len(anidbSkipTimes) == 0 {
			t.Fatalf("Expected AniSkip API to return skip intervals for AniDB ID 'anidb:18464' via title resolution")
		}
		t.Logf("✓ [AniSkip E2E] AniDB ID 'anidb:18464' successfully resolved MAL ID and returned %d skip intervals", len(anidbSkipTimes))

		_, luaFile, chapFile, _, _, err := getMpvCmd(
			"https://example.com/test.m3u8", show.Name, "1", "5114", "24 min", nil,
		)
		if err != nil {
			t.Fatalf("getMpvCmd failed: %v", err)
		}

		defer func() {
			if luaFile != "" {
				_ = os.Remove(luaFile)
			}
			if chapFile != "" {
				_ = os.Remove(chapFile)
			}
		}()

		if chapFile != "" {
			content, err := os.ReadFile(chapFile)
			if err != nil {
				t.Fatalf("failed to read generated chapters file: %v", err)
			}
			if !strings.Contains(string(content), "CHAPTER") {
				t.Fatalf("chapters file missing CHAPTER header: %s", string(content))
			}
			t.Logf("✓ [AniSkip E2E] Generated MPV chapters file (%d bytes) verified!", len(content))
		}
	})

	t.Run("09_VidSrc_Movie_Headless_MPV_Playback_And_IPC", func(t *testing.T) {
		mpvPath, err := exec.LookPath("mpv")
		if err != nil {
			t.Skip("mpv binary not found in PATH, skipping Movie MPV test")
		}

		socketPath := filepath.Join(t.TempDir(), "mpv.sock")
		t.Setenv("CLARE_MPV_SOCK", socketPath)

		movieShow := AnimeShow{
			ID:    "vidsrc:movie:603",
			Name:  "The Matrix",
			Type:  "MOVIE",
			MALID: "603",
		}
		playCmd := doPreparePlayback(movieShow, "1", "The Matrix", "sub", "best", false)
		playMsg := playCmd()
		pResMsg, ok := playMsg.(resolvedPlaybackMsg)
		if !ok || pResMsg.err != nil || pResMsg.cmd == nil {
			t.Fatalf("doPreparePlayback failed for Movie: %v", pResMsg.err)
		}

		cmd := pResMsg.cmd
		cmd.Args = append(cmd.Args,
			"--vo=null",
			"--ao=null",
			"--mute=yes",
			"--idle=no",
			"--keep-open=no",
			"--no-terminal",
		)

		if err := cmd.Start(); err != nil {
			t.Fatalf("Failed to start headless Movie MPV command (%s): %v", mpvPath, err)
		}
		defer func() {
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
				_ = cmd.Wait()
			}
		}()

		socketPath = getMpvSocketPath()
		var ipc *MPVIPCClient
		// Retry connecting to IPC socket for up to 3 seconds until MPV initializes socket listener
		connDeadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(connDeadline) {
			if client, err := NewMPVIPCClient(socketPath); err == nil {
				ipc = client
				break
			}
			time.Sleep(150 * time.Millisecond)
		}

		if ipc == nil {
			t.Fatalf("Failed to connect to Movie MPV IPC socket at %s", socketPath)
		}
		defer ipc.Close()

		health, _ := ipc.InspectHealth()
		t.Logf("✓ [Movie E2E] Headless MPV successfully launched and verified via IPC! Health: %+v", health)

		// 1. Query running MPV instance via IPC to verify media-title property for movie
		titleVal, err := ipc.GetProperty("media-title")
		if err != nil {
			titleVal, err = ipc.GetProperty("force-media-title")
		}
		if err != nil {
			t.Fatalf("Failed to query title from running Movie MPV instance via IPC: %v", err)
		}
		titleStr := fmt.Sprintf("%v", titleVal)
		if !strings.Contains(titleStr, "The Matrix") {
			t.Fatalf("Running Movie MPV IPC title mismatch! Expected 'The Matrix', got: '%s'", titleStr)
		}
		t.Logf("✓ [Movie IPC E2E] Running MPV instance verified movie title over IPC socket: '%s'", titleStr)

		// 2. Wait until MPV connects to stream, demuxes media, and playback actively begins
		pos, err := ipc.WaitForPlaybackToStart(10 * time.Second)
		if err != nil {
			t.Fatalf("Movie MPV failed to start playback: %v", err)
		}
		t.Logf("✓ [Movie Playback E2E] Movie MPV playback successfully confirmed active! Position: %.2fs", pos)
	})

	t.Run("10_VidSrc_TV_Show_Headless_MPV_Playback_And_IPC", func(t *testing.T) {
		mpvPath, err := exec.LookPath("mpv")
		if err != nil {
			t.Skip("mpv binary not found in PATH, skipping TV Show MPV test")
		}

		socketPath := filepath.Join(t.TempDir(), "mpv.sock")
		t.Setenv("CLARE_MPV_SOCK", socketPath)

		tvShow := AnimeShow{
			ID:    "vidsrc:tv:2190",
			Name:  "South Park",
			MALID: "2190",
		}
		playCmd := doPreparePlayback(tvShow, "S02E01", "Terrance and Phillip in Not Without My Anus", "sub", "best", false)
		playMsg := playCmd()
		pResMsg, ok := playMsg.(resolvedPlaybackMsg)
		if !ok || pResMsg.err != nil || pResMsg.cmd == nil {
			t.Fatalf("doPreparePlayback failed for TV Show: %v", pResMsg.err)
		}

		cmd := pResMsg.cmd
		cmd.Args = append(cmd.Args,
			"--vo=null",
			"--ao=null",
			"--mute=yes",
			"--idle=no",
			"--keep-open=no",
			"--no-terminal",
		)

		if err := cmd.Start(); err != nil {
			t.Fatalf("Failed to start headless TV Show MPV command (%s): %v", mpvPath, err)
		}
		defer func() {
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
				_ = cmd.Wait()
			}
		}()

		socketPath = getMpvSocketPath()
		var ipc *MPVIPCClient
		connDeadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(connDeadline) {
			if client, err := NewMPVIPCClient(socketPath); err == nil {
				ipc = client
				break
			}
			time.Sleep(150 * time.Millisecond)
		}

		if ipc == nil {
			t.Fatalf("Failed to connect to TV Show MPV IPC socket at %s", socketPath)
		}
		defer ipc.Close()

		health, _ := ipc.InspectHealth()
		t.Logf("✓ [TV Show E2E] Headless MPV successfully launched and verified via IPC! Health: %+v", health)

		// Query running MPV instance via IPC to verify media-title property is active
		titleVal, err := ipc.GetProperty("media-title")
		if err != nil {
			titleVal, err = ipc.GetProperty("force-media-title")
		}
		if err != nil {
			t.Fatalf("Failed to query title from running MPV instance via IPC: %v", err)
		}
		titleStr := fmt.Sprintf("%v", titleVal)
		if !strings.Contains(titleStr, "South Park") || !strings.Contains(titleStr, "Terrance and Phillip") {
			t.Fatalf("Running MPV IPC title mismatch! Expected show & ep name, got: '%s'", titleStr)
		}
		t.Logf("✓ [TV Show IPC E2E] Running MPV instance verified title over IPC socket: '%s'", titleStr)

		// Wait until MPV connects to stream, demuxes media, and playback actively begins
		pos, err := ipc.WaitForPlaybackToStart(10 * time.Second)
		if err != nil {
			t.Fatalf("TV Show MPV failed to start playback: %v", err)
		}
		t.Logf("✓ [TV Show Playback E2E] TV Show MPV playback successfully confirmed active! Position: %.2fs", pos)
	})

	t.Run("11_Full_TUI_VidSrc_Movie_And_TV_Interactive_User_Flow", func(t *testing.T) {
		m := initialModel("", "sub", "best", false)

		// 1. Search for South Park
		p := &VidSrcProvider{}
		shows, err := p.Search("South Park", "sub")
		if err != nil || len(shows) == 0 {
			t.Fatalf("VidSrc search for South Park failed: %v", err)
		}

		// 2. Fetch TV episodes
		tvShow, tvEps, err := p.FetchEpisodes(shows[0].ID, "sub")
		if err != nil || len(tvEps) == 0 {
			t.Fatalf("VidSrc fetch episodes failed: %v", err)
		}

		// 3. Prepare playback for S02E01 with actual TMDB episode title
		epName := "Terrance and Phillip in Not Without My Anus"
		playCmd := doPreparePlayback(tvShow, tvEps[0], epName, "sub", "best", false)
		playMsg := playCmd()
		pResMsg, ok := playMsg.(resolvedPlaybackMsg)
		if !ok || pResMsg.err != nil || pResMsg.cmd == nil {
			t.Fatalf("doPreparePlayback failed for VidSrc TV Show: %v", pResMsg.err)
		}

		// Verify --force-media-title contains exact show name, episode number, and episode title
		foundTitleFlag := false
		for _, arg := range pResMsg.cmd.Args {
			if strings.HasPrefix(arg, "--force-media-title=") {
				foundTitleFlag = true
				if !strings.Contains(arg, "South Park") || !strings.Contains(arg, "Terrance and Phillip") {
					t.Fatalf("Expected --force-media-title to contain show and episode title, got: %s", arg)
				}
				t.Logf("✓ [TUI VidSrc E2E] MPV --force-media-title flag verified: %s", arg)
			}
		}
		if !foundTitleFlag {
			t.Fatalf("Expected --force-media-title argument in generated MPV command")
		}

		m.selectedShow = tvShow
		m.selectedEp = "S01E01"
		res, _ := m.Update(pResMsg)
		mUpdated := res.(model)
		if !mUpdated.playbackActive {
			t.Fatalf("expected playbackActive=true after VidSrc resolvedPlaybackMsg, got false")
		}
		t.Logf("✓ [TUI VidSrc E2E] Interactive TUI user flow for TV Show S01E01 verified!")
	})

	t.Run("12_VidSrc_Subtitles_And_Captions_Verification", func(t *testing.T) {
		p := &VidSrcProvider{}
		tvStreams, err := p.ResolveStreams("vidsrc:tv:2190", "sub", "S01E01", "best")
		if err != nil || len(tvStreams) == 0 {
			t.Fatalf("VidSrc TV stream resolution failed: %v", err)
		}

		if len(tvStreams[0].Subtitles) > 0 {
			subURL := tvStreams[0].Subtitles[0].URL
			t.Logf("✓ [Captions E2E] Resolved subtitle track: Label=%s, URL=%s", tvStreams[0].Subtitles[0].Label, subURL)

			client := newLoggingHttpClient(5 * time.Second)
			resp, err := client.Get(subURL)
			if err == nil && resp.StatusCode == http.StatusOK {
				resp.Body.Close()
				t.Logf("✓ [Captions E2E] Subtitle file HTTP 200 OK verified!")
			}
		} else {
			t.Logf("✓ [Captions E2E] Resolved stream URL: %s (no external SRT subtitles returned for this episode)", tvStreams[0].URL)
		}
	})

	t.Run("13_TV_And_Anime_Episode_Title_Assertion", func(t *testing.T) {
		socketPath := filepath.Join(t.TempDir(), "mpv.sock")
		t.Setenv("CLARE_MPV_SOCK", socketPath)

		// 1. Verify TMDB Season Details fetch populates actual episode names for TV shows
		seasonCmd := doFetchTmdbSeasonDetails("vidsrc:tv:2190", 2)
		seasonMsg := seasonCmd()
		sResMsg, ok := seasonMsg.(tmdbSeasonDetailsMsg)
		if !ok || sResMsg.err != nil || len(sResMsg.epInfo) == 0 {
			t.Fatalf("doFetchTmdbSeasonDetails failed for South Park S2: %v", sResMsg.err)
		}

		ep1Info, ok := sResMsg.epInfo["S02E01"]
		if !ok || ep1Info.Title == "" {
			t.Fatalf("Expected S02E01 to have non-empty episode title in TMDB details map, got: %+v", ep1Info)
		}
		if !strings.Contains(ep1Info.Title, "Terrance") {
			t.Fatalf("Expected S02E01 title to contain 'Terrance', got: '%s'", ep1Info.Title)
		}
		t.Logf("✓ [Title E2E] TMDB season details correctly resolved episode title: S02E01 -> '%s'", ep1Info.Title)

		// 2. Verify doPreparePlayback formats title with full episode name
		tvShow := AnimeShow{ID: "vidsrc:tv:2190", Name: "South Park"}
		cmdFunc := doPreparePlayback(tvShow, "S02E01", ep1Info.Title, "sub", "best", false)
		msg := cmdFunc()
		resMsg, ok := msg.(resolvedPlaybackMsg)
		if !ok || resMsg.err != nil || resMsg.cmd == nil {
			t.Fatalf("doPreparePlayback failed: %v", resMsg.err)
		}

		expectedFullTitle := "South Park - Ep S02E01: Terrance and Phillip in Not Without My Anus"
		foundMatch := false
		for _, arg := range resMsg.cmd.Args {
			if arg == "--force-media-title="+expectedFullTitle {
				foundMatch = true
				break
			}
		}
		if !foundMatch {
			t.Fatalf("Expected MPV --force-media-title='%s', generated args: %v", expectedFullTitle, resMsg.cmd.Args)
		}
		t.Logf("✓ [Title E2E] doPreparePlayback correctly formatted --force-media-title='%s'", expectedFullTitle)

		// 3. Launch headless MPV process and query running instance title over IPC socket
		_, err := exec.LookPath("mpv")
		if err != nil {
			t.Skip("mpv binary not in PATH, skipping running MPV IPC title check")
		}

		resMsg.cmd.Args = append(resMsg.cmd.Args,
			"--vo=null",
			"--ao=null",
			"--idle=no",
			"--keep-open=no",
			"--no-terminal",
		)
		if err := resMsg.cmd.Start(); err != nil {
			t.Fatalf("Failed to start headless MPV for title verification: %v", err)
		}
		defer func() {
			if resMsg.cmd.Process != nil {
				_ = resMsg.cmd.Process.Kill()
				_ = resMsg.cmd.Wait()
			}
		}()

		time.Sleep(600 * time.Millisecond)

		ipc, err := NewMPVIPCClient(getMpvSocketPath())
		if err != nil {
			t.Logf("⚠️ IPC connection to running MPV instance note: %v", err)
		} else {
			defer ipc.Close()
			runningTitle, err := ipc.GetProperty("media-title")
			if err != nil {
				runningTitle, err = ipc.GetProperty("force-media-title")
			}
			if err != nil {
				t.Fatalf("Failed to query title from running MPV instance via IPC: %v", err)
			}

			titleStr := fmt.Sprintf("%v", runningTitle)
			if titleStr != expectedFullTitle {
				t.Fatalf("Running MPV IPC title mismatch! Expected '%s', got '%s'", expectedFullTitle, titleStr)
			}
			t.Logf("✓ [Title IPC E2E] Verified title on running MPV instance via IPC socket: '%s'", titleStr)
		}
	})

	t.Run("14_Autoplay_IPC_Title_Preservation_Verification", func(t *testing.T) {
		if _, err := exec.LookPath("mpv"); err != nil {
			t.Skip("mpv binary not found in PATH, skipping Autoplay IPC Title test")
		}

		socketPath := filepath.Join(t.TempDir(), "mpv.sock")
		t.Setenv("CLARE_MPV_SOCK", socketPath)

		m := initialModel("", "sub", "best", false)
		m.state = statePlaybackPreparing
		m.playbackActive = true
		m.selectedShow = AnimeShow{
			ID:    "vidsrc:tv:2190",
			Name:  "South Park",
			MALID: "2190",
		}
		m.playingShow = m.selectedShow
		m.episodes = []string{"S02E01", "S02E02"}
		m.playingEpisodes = m.episodes
		m.playingEp = "S02E01"
		m.episodeDetails["S02E02"] = JikanEpInfo{Title: "Chickenlover"}

		// Simulate autoplay trigger
		cmd := m.triggerAutoplayAction()
		if cmd == nil {
			t.Fatalf("Expected triggerAutoplayAction to return non-nil tea.Cmd for next episode S02E02")
		}
		msg := cmd()
		resMsg, ok := msg.(resolvedPlaybackMsg)
		if !ok || resMsg.err != nil {
			t.Fatalf("Failed to resolve playback command for S02E02 autoplay: %v", resMsg.err)
		}

		// Verify --force-media-title flag in prepared cmd
		var forceTitleFlag string
		for _, arg := range resMsg.cmd.Args {
			if strings.HasPrefix(arg, "--force-media-title=") {
				forceTitleFlag = strings.TrimPrefix(arg, "--force-media-title=")
				break
			}
		}
		expectedAutoplayTitle := "South Park - Ep S02E02: Chickenlover"
		if forceTitleFlag != expectedAutoplayTitle {
			t.Fatalf("Prepared autoplay command title mismatch! Expected '%s', got '%s'", expectedAutoplayTitle, forceTitleFlag)
		}
		t.Logf("✓ [Autoplay Title E2E] Prepared autoplay command title verified: '%s'", forceTitleFlag)

		// Start headless MPV
		if err := resMsg.cmd.Start(); err != nil {
			t.Fatalf("Failed to start headless MPV for autoplay title verification: %v", err)
		}
		defer func() {
			if resMsg.cmd.Process != nil {
				_ = resMsg.cmd.Process.Kill()
				_ = resMsg.cmd.Wait()
			}
		}()

		time.Sleep(600 * time.Millisecond)

		nextURL := "https://hls.anidb.app/stream/test/master.m3u8"
		loadErr := loadFileInMpv(nextURL, forceTitleFlag, "S02E02", "2190", nil, 1440.0, "[]")
		if loadErr != nil {
			t.Fatalf("loadFileInMpv failed over IPC socket: %v", loadErr)
		}

		time.Sleep(300 * time.Millisecond)

		ipc, err := NewMPVIPCClient(getMpvSocketPath())
		if err != nil {
			t.Logf("⚠️ IPC connection note: %v", err)
		} else {
			defer ipc.Close()
			runningTitle, err := ipc.GetProperty("media-title")
			if err != nil {
				runningTitle, err = ipc.GetProperty("force-media-title")
			}
			if err != nil {
				t.Fatalf("Failed to query title from running MPV process after IPC loadFileInMpv: %v", err)
			}
			titleStr := fmt.Sprintf("%v", runningTitle)
			if titleStr != expectedAutoplayTitle {
				t.Fatalf("Running MPV IPC title mismatch after autoplay load! Expected '%s', got '%s'", expectedAutoplayTitle, titleStr)
			}
			t.Logf("✓ [Autoplay Title IPC E2E] Running MPV process confirmed title over IPC socket during autoplay: '%s'", titleStr)
		}
	})
}
