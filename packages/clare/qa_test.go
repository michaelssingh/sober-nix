package main

import (
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
}
