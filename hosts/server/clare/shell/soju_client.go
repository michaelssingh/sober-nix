package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type SojuAdminClient struct {
	apiURL string
}

func NewSojuAdminClient(apiURL string) *SojuAdminClient {
	return &SojuAdminClient{apiURL: apiURL}
}

type CreateUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type CreateUserResponse struct {
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

func (c *SojuAdminClient) CreateUser(username, password string) error {
	reqBody := CreateUserRequest{
		Username: username,
		Password: password,
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(c.apiURL+"/api/v1/users", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to call API: %w", err)
	}
	defer resp.Body.Close()

	var res CreateUserResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		if res.Error != "" {
			return fmt.Errorf("API error: %s", res.Error)
		}
		return fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	return nil
}
