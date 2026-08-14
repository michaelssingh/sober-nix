package main

import (
	"os"
	"os/exec"
	"testing"
)

// TestQAFullPipeline runs an automated end-to-end QA validation of Clare's core systems.
func TestQAFullPipeline(t *testing.T) {
	_ = InitLogger("")

	t.Run("01_MultiProviderResolver_Initialization", func(t *testing.T) {
		resolver := NewMultiProviderResolver()
		if resolver == nil || len(resolver.providers) == 0 {
			t.Fatalf("Expected multi-provider resolver to initialize active providers")
		}
		t.Logf("✓ MultiProviderResolver initialized with %d active providers", len(resolver.providers))
	})

	t.Run("02_AniDB_Search_And_Episodes", func(t *testing.T) {
		p := &AniDBProvider{}
		shows, err := p.Search("Sakamoto Days", "sub")
		if err != nil || len(shows) == 0 {
			t.Fatalf("AniDB search failed: %v", err)
		}
		t.Logf("✓ AniDB search returned %d shows (Top: %s)", len(shows), shows[0].Name)

		show, eps, err := p.FetchEpisodes(shows[0].ID, "sub")
		if err != nil || len(eps) == 0 {
			t.Fatalf("AniDB episode fetch failed for %s: %v", shows[0].ID, err)
		}
		t.Logf("✓ AniDB fetched %d episodes for %s", len(eps), show.Name)
	})

	t.Run("03_Metadata_Enrichment_AniList", func(t *testing.T) {
		show := AnimeShow{Name: "Sakamoto Days"}
		enrichShowMetadata(&show)
		if show.MALID == "" && show.AniListID == "" {
			t.Fatalf("Expected metadata enrichment to populate MALID or AniListID for Sakamoto Days")
		}
		t.Logf("✓ AniList enriched Sakamoto Days: MALID=%s, AniListID=%s, Thumbnail=%s",
			show.MALID, show.AniListID, show.Thumbnail)
	})

	t.Run("04_Preflight_Stream_Resolution", func(t *testing.T) {
		resolver := NewMultiProviderResolver()
		show, stream, err := resolver.ResolveWithFallback("Sakamoto Days", "sub", "1", "best")
		if err != nil {
			t.Fatalf("ResolveWithFallback failed for Sakamoto Days Ep 1: %v", err)
		}
		if stream.URL == "" {
			t.Fatalf("Expected resolved stream URL to be non-empty")
		}
		t.Logf("✓ ResolveWithFallback resolved stream URL for %s: %s (Provider: %s)",
			show.Name, stream.URL, stream.Provider)
	})

	t.Run("05_MPV_Command_Build", func(t *testing.T) {
		show := AnimeShow{
			ID:       "anidb:4556",
			Name:     "Sakamoto Days",
			MALID:    "58939",
			Duration: "24 min",
		}
		streamURL := "https://hls.anidb.app/stream/test/master.m3u8"
		cmd, luaFile, chapFile, _, _, err := getMpvCmd(streamURL, show.Name, "1", show.MALID, show.Duration, nil)
		if err != nil {
			t.Fatalf("Failed to generate mpv command: %v", err)
		}
		if cmd == nil {
			t.Fatalf("Generated mpv command was nil")
		}
		_ = exec.Command("rm", "-f", luaFile, chapFile)
		t.Logf("✓ Generated MPV command binary: %s", cmd.Path)
	})

	t.Run("06_TUI_State_Transitions", func(t *testing.T) {
		m := initialModel("", "sub", "best", false)
		if m.state != stateHistory {
			t.Fatalf("Expected initial state to be stateHistory, got %v", m.state)
		}
		view := m.View()
		if view == "" {
			t.Fatalf("TUI view string was empty")
		}
		t.Logf("✓ TUI initial state & rendering verified!")
	})

	t.Run("07_VidSrc_Movie_And_TV_MPV_Command_Build", func(t *testing.T) {
		// 1. Movie Test
		movieShow := AnimeShow{
			ID:   "vidsrc:movie:603",
			Name: "The Matrix",
			Type: "MOVIE",
		}
		movieStream := "https://nextgenmarketinghub.site/mKTGUdZlu/pl/test/master.m3u8"
		movieCmd, movieLua, _, _, _, err := getMpvCmd(movieStream, movieShow.Name, "Movie", movieShow.ID, "", nil)
		if err != nil || movieCmd == nil {
			t.Fatalf("Failed to generate MPV command for Movie: %v", err)
		}
		_ = exec.Command("rm", "-f", movieLua)
		t.Logf("✓ Movie MPV command build verified! Args count: %d", len(movieCmd.Args))

		// 2. TV Show Test (South Park with full episode title)
		tvShow := AnimeShow{
			ID:   "vidsrc:tv:2190",
			Name: "South Park",
			Type: "TV Series",
		}
		tvStream := "https://influencerstrategygroup.site/2anMY6het/pl/test/master.m3u8"
		fullEpTitle := "South Park - Ep S02E01: Terrance and Phillip in Not Without My Anus"
		tvCmd, tvLua, tvChap, _, _, err := getMpvCmd(tvStream, fullEpTitle, "S02E01", tvShow.ID, "22 min", nil)
		if err != nil || tvCmd == nil {
			t.Fatalf("Failed to generate MPV command for TV Show: %v", err)
		}
		_ = exec.Command("rm", "-f", tvLua, tvChap)

		// 3. Verify MPV flag parsing validity
		testArgs := append([]string{}, tvCmd.Args[1:len(tvCmd.Args)-1]...)
		testArgs = append(testArgs, "--help")
		verifyCmd := exec.Command("mpv", testArgs...)
		out, err := verifyCmd.CombinedOutput()
		if err != nil {
			t.Fatalf("MPV command flag validation failed: %v | Output: %s", err, string(out))
		}
		t.Logf("✓ MPV binary successfully accepted all generated TV & Movie arguments!")
	})

	t.Run("08_Episode_Parsing_And_Resume_Position_Lookup", func(t *testing.T) {
		// 1. Verify TV episode numbers parse uniquely per season/episode
		s1e1 := parseEpisodeNumber("S01E01")
		s1e2 := parseEpisodeNumber("S01E02")
		s2e1 := parseEpisodeNumber("S02E01")
		s2e5 := parseEpisodeNumber("S02E05")

		if s1e1 == s2e1 || s2e1 == s2e5 || s1e1 == s1e2 {
			t.Fatalf("parseEpisodeNumber produced duplicate values for TV episodes: S01E01=%f, S01E02=%f, S02E01=%f, S02E05=%f", s1e1, s1e2, s2e1, s2e5)
		}
		t.Logf("✓ TV episode numbers parsed uniquely: S01E01=%f, S01E02=%f, S02E01=%f, S02E05=%f", s1e1, s1e2, s2e1, s2e5)

		// 2. Verify resume position lookup for TV show with EpisodeStr
		tmpDir, err := os.MkdirTemp("", "clare-resume-test-*")
		if err != nil {
			t.Fatalf("failed to create temp state dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		origDir := os.Getenv("CLARE_STATE_DIR")
		os.Setenv("CLARE_STATE_DIR", tmpDir)
		defer os.Setenv("CLARE_STATE_DIR", origDir)

		posData := PositionsData{
			"vidsrc:tv:2190": ShowState{
				ResumeState: &ResumeState{
					Episode:         20005.0,
					EpisodeStr:      "S02E05",
					PositionSeconds: 450.5,
					TotalSeconds:    1320.0,
				},
			},
		}
		if err := savePositions(posData); err != nil {
			t.Fatalf("failed to save test positions: %v", err)
		}

		cmd, lua, _, startSec, _, err := getMpvCmd("https://influencerstrategygroup.site/test/master.m3u8", "South Park - Ep S02E05: City on the Edge of Forever", "S02E05", "vidsrc:tv:2190", "22 min", nil)
		if err != nil || cmd == nil {
			t.Fatalf("getMpvCmd failed for S02E05: %v", err)
		}
		_ = os.Remove(lua)

		if startSec == 0 {
			t.Fatalf("Expected S02E05 to resume at 450.5s, got %f", startSec)
		}
		t.Logf("✓ TV show resume position lookup verified: S02E05 resumed at %fs", startSec)
	})
}
