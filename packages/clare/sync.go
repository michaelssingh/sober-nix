package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// SyncAllHistory silently syncs all completed episodes from positions.json
// to AniList on startup, skipping entries already synced (tracked via LastSyncedEp).
func SyncAllHistory() {
	cfg := loadConfig()
	anilistToken := cfg.AnilistToken
	if anilistToken == "" {
		anilistToken = os.Getenv("ANILIST_TOKEN")
		if anilistToken == "" {
			if path := os.Getenv("ANILIST_TOKEN_FILE"); path != "" {
				if data, err := os.ReadFile(path); err == nil {
					anilistToken = strings.TrimSpace(string(data))
				}
			}
		}
	}
	if anilistToken == "" {
		return
	}

	positions, err := loadPositions()
	if err != nil {
		positions = make(map[string]ShowState)
	}

	changed := false

	// 2. Push any local progress not yet on AniList
	malToName := loadMalIDToNameMap()
	for malIDStr, state := range positions {
		if len(state.CompletedEpisodes) == 0 {
			continue
		}

		// Find the highest completed episode
		var maxEp float64
		for _, ep := range state.CompletedEpisodes {
			if ep > maxEp {
				maxEp = ep
			}
		}

		// Skip if already synced up to this episode
		if maxEp <= state.LastSyncedEp {
			continue
		}

		malID, err := strconv.Atoi(malIDStr)
		if err != nil || malID == 0 {
			continue
		}

		showName := malToName[malIDStr]
		if showName == "" {
			showName = fmt.Sprintf("MAL ID %d", malID)
		}

		epProgress := int(maxEp)
		if err := syncToAniList(anilistToken, malID, epProgress, false, 0); err != nil {
			debugLog("[ERROR] SyncAllHistory: AniList sync failed for %s: %v", showName, err)
			continue
		}

		debugLog("[INFO] SyncAllHistory: synced %s up to ep %d", showName, epProgress)
		state.LastSyncedEp = maxEp
		positions[malIDStr] = state
		changed = true
	}

	if changed {
		if err := savePositions(positions); err != nil {
			debugLog("[ERROR] SyncAllHistory: failed to save positions after sync: %v", err)
		}
		if globalProgram != nil {
			globalProgram.Send(syncRefreshMsg{})
		}
	}
}

// SyncProgress syncs the anime progress to AniList and/or MyAnimeList in the background.
// completedLocally should be true only when Clare itself determines the show is finished
// (i.e. the user watched the final episode in the AllAnime arc). Clare — not a remote
// tracker's episode count — is the source of truth for completion.
func SyncProgress(malIDStr string, epNoStr string, completedLocally bool) {
	if malIDStr == "" || malIDStr == "0" {
		return
	}
	malID, err := strconv.Atoi(malIDStr)
	if err != nil {
		debugLog("[ERROR] SyncProgress: invalid MAL ID %q: %v", malIDStr, err)
		return
	}
	var epNo float64
	_, err = fmt.Sscanf(epNoStr, "%f", &epNo)
	if err != nil {
		debugLog("[ERROR] SyncProgress: invalid episode number %q: %v", epNoStr, err)
		return
	}
	epProgress := int(epNo)

	cfg := loadConfig()
	anilistToken := cfg.AnilistToken
	if anilistToken == "" {
		anilistToken = os.Getenv("ANILIST_TOKEN")
		if anilistToken == "" {
			if path := os.Getenv("ANILIST_TOKEN_FILE"); path != "" {
				if data, err := os.ReadFile(path); err == nil {
					anilistToken = strings.TrimSpace(string(data))
				}
			}
		}
	}

	malToken := cfg.MalToken
	if malToken == "" {
		malToken = os.Getenv("MAL_TOKEN")
		if malToken == "" {
			if path := os.Getenv("MAL_TOKEN_FILE"); path != "" {
				if data, err := os.ReadFile(path); err == nil {
					malToken = strings.TrimSpace(string(data))
				}
			}
		}
	}

	if anilistToken == "" && malToken == "" {
		debugLog("[WARN] SyncProgress: no AniList or MAL tokens found. Skipping sync.")
		return
	}

	// Look up cached AnilistID from history to skip the MAL→AniList resolve call.
	cachedAnilistID := 0
	if history, err := loadHistory(); err == nil {
		for _, h := range history {
			if cached, _, found := loadShowCache(h.ShowID); found && cached.MALID == malIDStr {
				cachedAnilistID = h.AnilistID
				break
			}
		}
	}

	if anilistToken != "" {
		go func() {
			err := syncToAniList(anilistToken, malID, epProgress, completedLocally, cachedAnilistID)
			if err != nil {
				debugLog("[ERROR] SyncProgress: AniList sync failed: %v", err)
			} else {
				debugLog("[INFO] SyncProgress: AniList sync successful for MAL ID %d, Ep %d (completed=%v)", malID, epProgress, completedLocally)
			}
		}()
	}

	if malToken != "" {
		go func() {
			err := syncToMAL(malToken, malID, epProgress)
			if err != nil {
				debugLog("[ERROR] SyncProgress: MAL sync failed: %v", err)
			} else {
				debugLog("[INFO] SyncProgress: MAL sync successful for MAL ID %d, Ep %d", malID, epProgress)
			}
		}()
	}
}

// syncToAniList pushes episode progress and status to AniList.
// cachedAnilistID — if non-zero — skips the MAL-to-AniList ID resolution API call.
// completedLocally — set by Clare when the user finishes the final AllAnime episode;
// Clare is the authority here, not AniList's episode count.
func syncToAniList(token string, malID int, progress int, completedLocally bool, cachedAnilistID int) error {
	client := &http.Client{Timeout: 10 * time.Second}

	anilistMediaID := cachedAnilistID
	if anilistMediaID == 0 {
		// No cached ID — resolve via MAL ID.
		var resolveQuery = `query ($idMal: Int) { Media (idMal: $idMal, type: ANIME) { id } }`
		payload := map[string]interface{}{
			"query": resolveQuery,
			"variables": map[string]interface{}{
				"idMal": malID,
			},
		}
		body, _ := json.Marshal(payload)

		debugLog("[API] AniList POST https://graphql.anilist.co (Resolve MAL ID %d)", malID)
		req, err := http.NewRequest("POST", "https://graphql.anilist.co", bytes.NewBuffer(body))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("AniList resolve query returned status %d: %s", resp.StatusCode, string(bodyBytes))
		}

		var resolveResp struct {
			Data struct {
				Media struct {
					ID int `json:"id"`
				} `json:"Media"`
			} `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&resolveResp); err != nil {
			return err
		}
		anilistMediaID = resolveResp.Data.Media.ID
		if anilistMediaID == 0 {
			return fmt.Errorf("could not resolve AniList Media ID for MAL ID %d", malID)
		}
	} else {
		debugLog("[API] AniList: using cached Media ID %d for MAL ID %d (skipping resolve)", anilistMediaID, malID)
	}

	status := "CURRENT"
	if completedLocally {
		status = "COMPLETED"
	}

	var mutation = `mutation ($mediaId: Int, $progress: Int, $status: MediaListStatus) {
		SaveMediaListEntry (mediaId: $mediaId, progress: $progress, status: $status) {
			id
			progress
		}
	}`
	mutationPayload := map[string]interface{}{
		"query": mutation,
		"variables": map[string]interface{}{
			"mediaId":  anilistMediaID,
			"progress": progress,
			"status":   status,
		},
	}
	mutBody, _ := json.Marshal(mutationPayload)

	debugLog("[API] AniList POST https://graphql.anilist.co (SaveMediaListEntry MediaId %d, Ep %d, Status %s)", anilistMediaID, progress, status)
	reqMut, err := http.NewRequest("POST", "https://graphql.anilist.co", bytes.NewBuffer(mutBody))
	if err != nil {
		return err
	}
	reqMut.Header.Set("Content-Type", "application/json")
	reqMut.Header.Set("Authorization", "Bearer "+token)

	respMut, err := client.Do(reqMut)
	if err != nil {
		return err
	}
	defer respMut.Body.Close()

	if respMut.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(respMut.Body)
		return fmt.Errorf("AniList mutation returned status %d: %s", respMut.StatusCode, string(bodyBytes))
	}

	return nil
}

func syncToMAL(token string, malID int, progress int) error {
	apiURL := fmt.Sprintf("https://api.myanimelist.net/v2/anime/%d/my_list_status", malID)
	data := fmt.Sprintf("num_watched_episodes=%d&status=watching", progress)

	debugLog("[API] MyAnimeList PUT %s (data: %s)", apiURL, data)
	req, err := http.NewRequest("PUT", apiURL, bytes.NewBufferString(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("MAL API returned status %d", resp.StatusCode)
	}

	return nil
}

type syncRefreshMsg struct{}

func pullFromAniList(token string, positions map[string]ShowState) (bool, error) {
	// 1. Fetch current viewer's username
	viewerQuery := `query { Viewer { name } }`
	viewerPayload := map[string]interface{}{
		"query": viewerQuery,
	}
	viewerBody, _ := json.Marshal(viewerPayload)

	reqViewer, err := http.NewRequest("POST", "https://graphql.anilist.co", bytes.NewBuffer(viewerBody))
	if err != nil {
		return false, err
	}
	reqViewer.Header.Set("Content-Type", "application/json")
	reqViewer.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 15 * time.Second}
	respViewer, err := client.Do(reqViewer)
	if err != nil {
		return false, err
	}
	defer respViewer.Body.Close()

	if respViewer.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(respViewer.Body)
		return false, fmt.Errorf("AniList viewer query returned status %d: %s", respViewer.StatusCode, string(bodyBytes))
	}

	var viewerResp struct {
		Data struct {
			Viewer struct {
				Name string `json:"name"`
			} `json:"Viewer"`
		} `json:"data"`
	}
	if err := json.NewDecoder(respViewer.Body).Decode(&viewerResp); err != nil {
		return false, err
	}

	username := viewerResp.Data.Viewer.Name
	if username == "" {
		return false, fmt.Errorf("could not retrieve authenticated AniList username")
	}

	// 2. Fetch the MediaListCollection using the username — pull both CURRENT and
	// COMPLETED so we can restore full watch history on a fresh install and
	// honour AniList COMPLETED status as the canonical completion signal.
	collectionQuery := `query ($userName: String!) {
		MediaListCollection (userName: $userName, type: ANIME, status_in: [CURRENT, COMPLETED]) {
			lists {
				entries {
					media {
						id
						idMal
						title {
							english
							romaji
						}
					}
					progress
					status
				}
			}
		}
	}`
	payload := map[string]interface{}{
		"query": collectionQuery,
		"variables": map[string]interface{}{
			"userName": username,
		},
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", "https://graphql.anilist.co", bytes.NewBuffer(body))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("AniList pull returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var pullResp struct {
		Data struct {
			MediaListCollection struct {
				Lists []struct {
					Entries []struct {
						Media struct {
							ID    int `json:"id"`
							IDMal int `json:"idMal"`
							Title struct {
								English string `json:"english"`
								Romaji  string `json:"romaji"`
							} `json:"title"`
						} `json:"media"`
						Progress int    `json:"progress"`
						Status   string `json:"status"`
					} `json:"entries"`
				} `json:"lists"`
			} `json:"mediaListCollection"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&pullResp); err != nil {
		return false, err
	}

	history, _ := loadHistory()
	changed := false
	for _, l := range pullResp.Data.MediaListCollection.Lists {
		for _, entry := range l.Entries {
			malID := entry.Media.IDMal
			if malID <= 0 {
				continue
			}
			malIDStr := strconv.Itoa(malID)
			anilistMediaID := entry.Media.ID
			isCompletedOnAniList := entry.Status == "COMPLETED"

			state, exists := positions[malIDStr]
			if !exists {
				state = ShowState{
					CompletedEpisodes: []float64{},
				}
			}

			localMax := 0.0
			for _, ep := range state.CompletedEpisodes {
				if ep > localMax {
					localMax = ep
				}
			}

			// Bidirectional sync: sync to local history.json so it populates Continue Watching.
			// We check if we already have this show resolved and recorded in history.
			showID, showName, found := resolveShowByMALID(malIDStr)

			inHistory := false
			var existingEntry *HistoryEntry
			for i := range history {
				if (showID != "" && history[i].ShowID == showID) || history[i].ShowName == showName {
					inHistory = true
					existingEntry = &history[i]
					break
				}
			}

			// If AniList says COMPLETED but we don't know it locally yet, mark it.
			if isCompletedOnAniList && existingEntry != nil && !existingEntry.Completed {
				existingEntry.Completed = true
				if anilistMediaID != 0 {
					existingEntry.AnilistID = anilistMediaID
				}
				if err := saveHistory(history); err == nil {
					changed = true
					debugLog("[INFO] PullSync: marked %s as COMPLETED from AniList", showName)
				}
				continue
			}

			needLocalUpdate := float64(entry.Progress) > localMax || !inHistory

			if needLocalUpdate {
				if float64(entry.Progress) > localMax {
					newCompleted := make(map[float64]bool)
					for _, ep := range state.CompletedEpisodes {
						newCompleted[ep] = true
					}
					for ep := 1; ep <= entry.Progress; ep++ {
						newCompleted[float64(ep)] = true
					}

					state.CompletedEpisodes = []float64{}
					for ep := range newCompleted {
						state.CompletedEpisodes = append(state.CompletedEpisodes, ep)
					}
					sort.Float64s(state.CompletedEpisodes)

					state.LastSyncedEp = float64(entry.Progress)
					positions[malIDStr] = state
					changed = true
					debugLog("[INFO] PullSync: updated local progress for MAL ID %d to Ep %d from AniList", malID, entry.Progress)
				}

				if !found {
					// Proactively try to resolve it by searching AllAnime using its title
					titleToSearch := entry.Media.Title.English
					if titleToSearch == "" {
						titleToSearch = entry.Media.Title.Romaji
					}
					if titleToSearch != "" {
						debugLog("[INFO] PullSync: show MAL ID %s (%q) not cached locally. Searching AniDB/VidSrc...", malIDStr, titleToSearch)
						shows, err := searchAnime(titleToSearch, "sub")
						if err == nil {
							for _, s := range shows {
								if s.MALID == malIDStr {
									debugLog("[INFO] PullSync: found match for %q: %s (%s). Fetching episodes to cache it...", titleToSearch, s.Name, s.ID)
									// Fetch episode list, which automatically caches the show details
									_, _, err := fetchEpisodeList(s.ID, "sub")
									if err == nil {
										showID = s.ID
										showName = s.Name
										found = true
										break
									}
								}
							}
						}
					}
				}

				if found {
					restoredEp := entry.Progress
					if cached, _, ok := loadShowCache(showID); ok {
						if arcCount := cached.EpCount(); arcCount > 0 && restoredEp > arcCount {
							debugLog("[INFO] PullSync: AniList progress %d exceeds show count %d for %s — capping at count for resume", restoredEp, arcCount, showName)
							restoredEp = arcCount
						}
					}
					epStr := fmt.Sprintf("%d", restoredEp)
					if err := recordWatch(showID, showName, epStr); err != nil {
						debugLog("[ERROR] PullSync: failed to update local history for %s: %v", showName, err)
					} else {
						// Persist AnilistID into the history entry for efficient future syncs.
						if anilistMediaID != 0 {
							if h, err := loadHistory(); err == nil {
								for i := range h {
									if h[i].ShowID == showID {
										h[i].AnilistID = anilistMediaID
										// Trust AniList COMPLETED flag even for freshly restored entries.
										if isCompletedOnAniList {
											h[i].Completed = true
										}
										_ = saveHistory(h)
										break
									}
								}
							}
						}
						debugLog("[INFO] PullSync: successfully added/updated %s Ep %s in history from AniList", showName, epStr)
						changed = true
					}
				}
			}
		}
	}

	return changed, nil
}

func resolveShowByMALID(malIDStr string) (string, string, bool) {
	cacheDir := filepath.Join(getCacheDir(), "shows")
	files, err := os.ReadDir(cacheDir)
	if err != nil {
		return "", "", false
	}
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") {
			path := filepath.Join(cacheDir, file.Name())
			if f, err := os.Open(path); err == nil {
				var entry CachedShowEntry
				if json.NewDecoder(f).Decode(&entry) == nil {
					if entry.Show.MALID == malIDStr {
						f.Close()
						return entry.Show.ID, entry.Show.Name, true
					}
				}
				f.Close()
			}
		}
	}
	return "", "", false
}

func loadMalIDToNameMap() map[string]string {
	m := make(map[string]string)
	cacheDir := filepath.Join(getCacheDir(), "shows")
	files, err := os.ReadDir(cacheDir)
	if err != nil {
		return m
	}
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") {
			path := filepath.Join(cacheDir, file.Name())
			if f, err := os.Open(path); err == nil {
				var entry CachedShowEntry
				if json.NewDecoder(f).Decode(&entry) == nil {
					if entry.Show.MALID != "" && entry.Show.MALID != "0" {
						m[entry.Show.MALID] = entry.Show.Name
					}
				}
				f.Close()
			}
		}
	}
	return m
}
