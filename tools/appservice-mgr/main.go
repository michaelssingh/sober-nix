package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Config struct {
	HomeserverURL string
	AccessToken   string
}

type JoinedRoomsResponse struct {
	JoinedRooms []string `json:"joined_rooms"`
}

type RoomNameResponse struct {
	Name string `json:"name"`
}

type MessagesResponse struct {
	Chunk []MessageEvent `json:"chunk"`
}

type MessageEvent struct {
	Sender  string `json:"sender"`
	Content struct {
		Body    string `json:"body"`
		MsgType string `json:"msgtype"`
	} `json:"content"`
}

func init() {
	handler := slog.NewJSONHandler(os.Stdout, nil)
	slog.SetDefault(slog.New(handler))
}

func main() {
	if len(os.Args) < 2 {
		slog.Error("Usage: appservice-mgr <data-directory>")
		os.Exit(1)
	}

	dataDir := os.Args[1]
	config := Config{
		HomeserverURL: "http://sober-athene.flycast:6167",
		AccessToken:   os.Getenv("MATRIX_ADMIN_TOKEN"),
	}

	if config.AccessToken == "" {
		slog.Error("MATRIX_ADMIN_TOKEN environment variable not set")
		os.Exit(1)
	}

	slog.Info("Starting appservice registration orchestrator", "data_dir", dataDir)

	var adminRoom string
	var err error
	maxAttempts := 30
	for i := 1; i <= maxAttempts; i++ {
		adminRoom, err = discoverAdminRoom(config)
		if err == nil {
			break
		}
		if i < maxAttempts {
			slog.Warn("Conduit not ready yet, retrying...", "attempt", i, "error", err)
			time.Sleep(5 * time.Second)
		} else {
			slog.Error("Failed to discover admin room", "error", err)
			os.Exit(1)
		}
	}
	slog.Info("Discovered Admin Room", "room_id", adminRoom)

	pattern := filepath.Join(dataDir, "*", "registration.yaml")
	files, _ := filepath.Glob(pattern)

	if len(files) == 0 {
		slog.Warn("No registration files found", "pattern", pattern)
		return
	}

	for _, file := range files {
		slog.Info("Processing registration", "file", file)
		registrationData, err := os.ReadFile(file)
		if err != nil {
			slog.Error("Failed to read registration file", "file", file, "error", err)
			continue
		}

		command := fmt.Sprintf("@conduit:sober.fyi: register-appservice\n```\n%s\n```", string(registrationData))
		if err := sendRegistrationAndVerify(config, adminRoom, command); err != nil {
			slog.Error("Registration failed or unverified", "file", file, "error", err)
			os.Exit(1) // Fail hard if registration doesn't stick
		}
		slog.Info("Appservice registered and verified successfully", "file", file)
	}
}

func sendRegistrationAndVerify(config Config, roomID string, command string) error {
	client := &http.Client{Timeout: 30 * time.Second}
	txnID := fmt.Sprintf("reg-%d", time.Now().UnixNano())
	url := fmt.Sprintf("%s/_matrix/client/v3/rooms/%s/send/m.room.message/%s", config.HomeserverURL, roomID, txnID)

	payload := map[string]interface{}{
		"msgtype": "m.text",
		"body":    command,
	}
	payloadBytes, _ := json.Marshal(payload)

	req, _ := http.NewRequest("PUT", url, strings.NewReader(string(payloadBytes)))
	req.Header.Add("Authorization", "Bearer "+config.AccessToken)
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("send failed with status %d", resp.StatusCode)
	}

	slog.Info("Command sent, waiting for bot response...")
	
	// Wait and poll for response from @conduit:sober.fyi
	for i := 0; i < 10; i++ {
		time.Sleep(2 * time.Second)
		messages, err := getRecentMessages(client, config, roomID)
		if err != nil {
			slog.Warn("Failed to fetch messages while waiting for response", "error", err)
			continue
		}

		for _, msg := range messages {
			if strings.Contains(msg.Sender, "conduit") && (strings.Contains(msg.Content.Body, "registered") || strings.Contains(msg.Content.Body, "Appservice")) {
				slog.Info("Received confirmation from bot", "response", msg.Content.Body)
				return nil
			}
			if strings.Contains(msg.Sender, "conduit") && strings.Contains(msg.Content.Body, "error") {
				return fmt.Errorf("bot reported error: %s", msg.Content.Body)
			}
		}
	}

	return fmt.Errorf("timeout waiting for bot response")
}

func getRecentMessages(client *http.Client, config Config, roomID string) ([]MessageEvent, error) {
	url := fmt.Sprintf("%s/_matrix/client/v3/rooms/%s/messages?limit=5&dir=b", config.HomeserverURL, roomID)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Authorization", "Bearer "+config.AccessToken)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var mr MessagesResponse
	if err := json.NewDecoder(resp.Body).Decode(&mr); err != nil {
		return nil, err
	}
	return mr.Chunk, nil
}

func discoverAdminRoom(config Config) (string, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest("GET", config.HomeserverURL+"/_matrix/client/v3/joined_rooms", nil)
	req.Header.Add("Authorization", "Bearer "+config.AccessToken)
	
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var jr JoinedRoomsResponse
	if err := json.NewDecoder(resp.Body).Decode(&jr); err != nil {
		return "", err
	}

	for _, roomID := range jr.JoinedRooms {
		req, _ := http.NewRequest("GET", config.HomeserverURL+"/_matrix/client/v3/rooms/"+roomID+"/state/m.room.name/", nil)
		req.Header.Add("Authorization", "Bearer "+config.AccessToken)
		
		r, err := client.Do(req)
		if err != nil {
			continue
		}
		
		var rn RoomNameResponse
		if err := json.NewDecoder(r.Body).Decode(&rn); err == nil {
			if strings.Contains(rn.Name, "Admin Room") {
				r.Body.Close()
				return roomID, nil
			}
		}
		r.Body.Close()
	}

	return "", fmt.Errorf("admin room not found")
}
