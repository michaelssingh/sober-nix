package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
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

	// 1. Pull down watch history from AniList first
	debugLog("[API] SyncAllHistory: pulling watch collection from AniList...")
	pulledChanged, err := pullFromAniList(anilistToken, positions)
	if err != nil {
		debugLog("[ERROR] SyncAllHistory: failed to pull from AniList: %v", err)
	} else if pulledChanged {
		changed = true
	}

	// 2. Push any local progress not yet on AniList
	for showID, state := range positions {
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

		// Look up the MAL ID from show cache
		show, _, found := loadShowCache(showID)
		if !found || show.MALID == "" || show.MALID == "0" {
			continue
		}

		malID, err := strconv.Atoi(show.MALID)
		if err != nil || malID == 0 {
			continue
		}

		epProgress := int(maxEp)
		if err := syncToAniList(anilistToken, malID, epProgress); err != nil {
			debugLog("[ERROR] SyncAllHistory: AniList sync failed for %s (MAL %d): %v", show.Name, malID, err)
			continue
		}

		debugLog("[INFO] SyncAllHistory: synced %s (MAL %d) up to ep %d", show.Name, malID, epProgress)
		state.LastSyncedEp = maxEp
		positions[showID] = state
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
func SyncProgress(malIDStr string, epNoStr string) {
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

	if anilistToken != "" {
		go func() {
			err := syncToAniList(anilistToken, malID, epProgress)
			if err != nil {
				debugLog("[ERROR] SyncProgress: AniList sync failed: %v", err)
			} else {
				debugLog("[INFO] SyncProgress: AniList sync successful for MAL ID %d, Ep %d", malID, epProgress)
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

func syncToAniList(token string, malID int, progress int) error {
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

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("AniList resolve query returned status %d", resp.StatusCode)
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

	anilistMediaID := resolveResp.Data.Media.ID
	if anilistMediaID == 0 {
		return fmt.Errorf("could not resolve AniList Media ID for MAL ID %d", malID)
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
			"status":   "CURRENT",
		},
	}
	mutBody, _ := json.Marshal(mutationPayload)

	debugLog("[API] AniList POST https://graphql.anilist.co (SaveMediaListEntry MediaId %d, Ep %d)", anilistMediaID, progress)
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
		return fmt.Errorf("AniList mutation returned status %d", respMut.StatusCode)
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
	query := `query {
		Viewer {
			mediaListCollection (type: ANIME, statusIn: [CURRENT, COMPLETED]) {
				lists {
					entries {
						media {
							idMal
							title { romaji english }
							episodes
						}
						progress
						status
					}
				}
			}
		}
	}`
	payload := map[string]interface{}{
		"query": query,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", "https://graphql.anilist.co", bytes.NewBuffer(body))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("AniList pull returned status %d", resp.StatusCode)
	}

	var pullResp struct {
		Data struct {
			Viewer struct {
				MediaListCollection struct {
					Lists []struct {
						Entries []struct {
							Media struct {
								IDMal    int `json:"idMal"`
								Episodes int `json:"episodes"`
							} `json:"media"`
							Progress int    `json:"progress"`
							Status   string `json:"status"`
						} `json:"entries"`
					} `json:"lists"`
				} `json:"mediaListCollection"`
			} `json:"Viewer"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&pullResp); err != nil {
		return false, err
	}

	changed := false
	for _, l := range pullResp.Data.Viewer.MediaListCollection.Lists {
		for _, entry := range l.Entries {
			malID := entry.Media.IDMal
			if malID <= 0 {
				continue
			}
			malIDStr := strconv.Itoa(malID)
			
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
		}
	}

	return changed, nil
}
