package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
)

type CreateUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type CreateUserResponse struct {
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

func main() {
	http.HandleFunc("/api/v1/users", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req CreateUserRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		err := createSojuUser(req.Username, req.Password)
		w.Header().Set("Content-Type", "application/json")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(CreateUserResponse{Error: err.Error()})
			return
		}

		json.NewEncoder(w).Encode(CreateUserResponse{Message: "User created successfully"})
	})

	log.Println("Starting Soju API server on :8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}

func createSojuUser(username, password string) error {
	adminSock := os.Getenv("SOJU_ADMIN_SOCK")
	if adminSock == "" {
		adminSock = "/var/lib/soju/admin.sock"
	}

	conn, err := net.Dial("unix", adminSock)
	if err != nil {
		return fmt.Errorf("failed to connect to soju admin socket at %s: %w", adminSock, err)
	}
	defer conn.Close()

	// The admin socket provides automatic authentication, no NICK/USER needed.
	fmt.Fprintf(conn, "PRIVMSG BouncerServ :user create -username %s -password %s\r\n", username, password)
	log.Printf("Sent: PRIVMSG BouncerServ :user create -username %s -password [REDACTED]", username)

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := scanner.Text()
		log.Printf("Received: %s", line)
		
		// BouncerServ response format usually includes the sender prefix and the message.
		if strings.Contains(line, "NOTICE") {
			if strings.Contains(line, "created user") {
				return nil
			}
			if strings.Contains(line, "already exists") || strings.Contains(line, "invalid") || strings.Contains(line, "error") {
				return fmt.Errorf("BouncerServ error: %s", line)
			}
		}
	}
	
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading from soju: %w", err)
	}

	return fmt.Errorf("did not receive a successful confirmation from BouncerServ")
}
