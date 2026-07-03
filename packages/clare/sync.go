package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// SyncProgress syncs the anime progress to AniList and/or MyAnimeList in the background.
func SyncProgress(malIDStr string, epNoStr string) {
	if malIDStr == "" || malIDStr == "0" {
		return
	}
	malID, err := strconv.Atoi(malIDStr)
	if err != nil {
		debugLog("SyncProgress: invalid MAL ID %q: %v", malIDStr, err)
		return
	}
	var epNo float64
	_, err = fmt.Sscanf(epNoStr, "%f", &epNo)
	if err != nil {
		debugLog("SyncProgress: invalid episode number %q: %v", epNoStr, err)
		return
	}
	epProgress := int(epNo)

	anilistToken := os.Getenv("ANILIST_TOKEN")
	if anilistToken == "" {
		if path := os.Getenv("ANILIST_TOKEN_FILE"); path != "" {
			if data, err := os.ReadFile(path); err == nil {
				anilistToken = strings.TrimSpace(string(data))
			}
		}
	}

	malToken := os.Getenv("MAL_TOKEN")
	if malToken == "" {
		if path := os.Getenv("MAL_TOKEN_FILE"); path != "" {
			if data, err := os.ReadFile(path); err == nil {
				malToken = strings.TrimSpace(string(data))
			}
		}
	}

	if anilistToken == "" && malToken == "" {
		debugLog("SyncProgress: no ANILIST_TOKEN or MAL_TOKEN env vars found. Skipping sync.")
		return
	}

	if anilistToken != "" {
		go func() {
			err := syncToAniList(anilistToken, malID, epProgress)
			if err != nil {
				debugLog("SyncProgress: AniList sync failed: %v", err)
			} else {
				debugLog("SyncProgress: AniList sync successful for MAL ID %d, Ep %d", malID, epProgress)
			}
		}()
	}

	if malToken != "" {
		go func() {
			err := syncToMAL(malToken, malID, epProgress)
			if err != nil {
				debugLog("SyncProgress: MAL sync failed: %v", err)
			} else {
				debugLog("SyncProgress: MAL sync successful for MAL ID %d, Ep %d", malID, epProgress)
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
