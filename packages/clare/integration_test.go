package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestAiringStreamResolution20(t *testing.T) {
	t.Log("Fetching currently airing anime from AniList...")
	shows, err := fetchAiringAnime()
	if err != nil {
		t.Fatalf("Failed to fetch airing anime: %v", err)
	}

	if len(shows) == 0 {
		t.Fatalf("No airing shows returned")
	}

	t.Logf("Successfully fetched %d airing anime titles. Testing end-to-end resolution for each...", len(shows))

	client := &http.Client{Timeout: 15 * time.Second}
	successCount := 0

	for idx, show := range shows {
		title := show.TitleRomaji
		if title == "" {
			title = show.TitleEnglish
		}
		t.Logf("[%d/20] Testing title: %q", idx+1, title)

		// 1. Search show on AllAnime
		searchResults, err := searchAnime(title, "sub")
		if err != nil {
			t.Logf("  [SKIP] AllAnime search query failed for %q: %v", title, err)
			continue
		}
		if len(searchResults) == 0 {
			t.Logf("  [SKIP] No AllAnime search results for %q", title)
			continue
		}

		matchedShow := searchResults[0]
		t.Logf("  Resolved to AllAnime show: %q (ID: %s)", matchedShow.Name, matchedShow.ID)

		// 2. Fetch episodes list
		resolvedShowDetails, _, err := fetchEpisodeList(matchedShow.ID, "sub")
		if err != nil {
			t.Logf("  [SKIP] Failed to fetch episodes for show %s: %v", matchedShow.ID, err)
			continue
		}
		
		epCount := resolvedShowDetails.EpCount()
		if epCount == 0 {
			t.Logf("  [SKIP] Show has 0 episodes in AllAnime")
			continue
		}

		// Try resolving stream for episode 1
		epNo := "1"
		t.Logf("  Resolving stream URL for Episode %s...", epNo)
		
		// 3. Resolve all sources/streams for episode 1
		streams, err := fetchAllResolvedStreams(matchedShow.ID, "sub", epNo)
		if err != nil {
			t.Logf("  [SKIP] Failed to resolve streams for Episode %s: %v", epNo, err)
			continue
		}
		if len(streams) == 0 {
			t.Logf("  [SKIP] 0 streams resolved for Episode %s", epNo)
			continue
		}

		// Pick the first resolved stream and check it
		stream := streams[0]
		t.Logf("  Selected stream: %s (%s) -> URL: %s", stream.SourceName, stream.Quality, stream.URL)

		// 4. Download first 5MB and validate headers directly on this VM
		req, err := http.NewRequest("GET", stream.URL, nil)
		if err != nil {
			t.Logf("  [FAIL] Failed to create HTTP request for stream: %v", err)
			t.Fail()
			continue
		}

		// Set required referer and user-agent headers
		referer := StreamReferer
		if strings.Contains(stream.URL, "mp4upload.com") {
			referer = "https://www.mp4upload.com/"
		} else if strings.Contains(stream.URL, "fast4speed") {
			referer = ""
		}
		headers := map[string]string{
			"User-Agent": UserAgent,
		}
		if referer != "" {
			headers["Referer"] = referer
		}

		t.Logf("  Running MPV dry-run check...")
		mpvSuccess, mpvErr := checkStreamMpvDryRun(stream.URL, headers)
		if mpvSuccess {
			t.Logf("  [SUCCESS] MPV dry-run succeeded for %s!", stream.SourceName)
			successCount++
			continue
		} else {
			t.Logf("  [INFO] MPV dry-run failed or warning occurred: %v. Falling back to HTTP chunk check...", mpvErr)
		}

		resp, err := client.Do(req)
		if err != nil {
			t.Logf("  [SKIP] Connection failed for stream URL: %v", err)
			continue
		}

		// Read first 5MB
		limitReader := io.LimitReader(resp.Body, 5*1024*1024)
		chunk, err := io.ReadAll(limitReader)
		resp.Body.Close()

		if err != nil && err != io.EOF {
			t.Logf("  [FAIL] Failed to download chunk from stream URL: %v", err)
			t.Fail()
			continue
		}

		if len(chunk) < 100 {
			t.Logf("  [FAIL] Downloaded chunk too small: %d bytes (expected >= 100). Status: %s, Headers: %v", len(chunk), resp.Status, resp.Header)
			t.Fail()
			continue
		}

		// Inspect headers to ensure it is a valid video format
		isValidVideo := false
		if chunk[0] == 0x47 {
			isValidVideo = true
			t.Logf("  [SUCCESS] Validated MPEG-TS sync byte 0x47! Chunk size: %d bytes", len(chunk))
		} else {
			// Look for ftyp or standard video containers in the first 256 bytes
			inspectArea := chunk
			if len(inspectArea) > 256 {
				inspectArea = inspectArea[:256]
			}
			inspectStr := string(inspectArea)
			if strings.Contains(inspectStr, "ftyp") || strings.Contains(inspectStr, "FLV") || strings.Contains(inspectStr, "matroska") || strings.Contains(inspectStr, "RIFF") || strings.Contains(inspectStr, "#EXTM3U") {
				isValidVideo = true
				t.Logf("  [SUCCESS] Validated container header! Chunk size: %d bytes", len(chunk))
			} else {
				// Print first 20 bytes as hex safely
				hexBytes := ""
				for i := 0; i < 20 && i < len(chunk); i++ {
					hexBytes += fmt.Sprintf("%02x ", chunk[i])
				}
				t.Logf("  [FAIL] Unrecognized media container prefix: %q (first 20 bytes: %s)", inspectStr[:20], hexBytes)
				t.Fail()
				continue
			}
		}

		if isValidVideo {
			successCount++
		}
	}

	t.Logf("Integration testing complete. Successfully validated %d / %d shows.", successCount, len(shows))
	if successCount == 0 {
		t.Fatalf("Zero shows validated successfully")
	}
}
