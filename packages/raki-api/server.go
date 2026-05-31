package adminapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strings"
)

type Server struct {
	adminSocketPath string
	mux             *http.ServeMux
	apiKeys         map[string]struct{}
	logger          *slog.Logger
}

type UserRequest struct {
	Username    string `json:"username,omitempty"`
	Password    string `json:"password,omitempty"`
	Admin       bool   `json:"admin"`
}

func NewServer(adminSocketPath string, apiKeys []string) *Server {
	keys := make(map[string]struct{})
	for _, k := range apiKeys {
		keys[k] = struct{}{}
	}
	s := &Server{
		adminSocketPath: adminSocketPath,
		mux:             http.NewServeMux(),
		apiKeys:         keys,
		logger:          slog.New(slog.NewJSONHandler(os.Stdout, nil)),
	}
	s.setupRoutes()
	return s
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("X-API-Key")
		if _, ok := s.apiKeys[key]; !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) setupRoutes() {
	s.mux.HandleFunc("/health", s.healthCheck)
	s.mux.Handle("/api/users", s.authMiddleware(http.HandlerFunc(s.usersHandler)))
	s.mux.Handle("/api/users/", s.authMiddleware(http.HandlerFunc(s.userHandler)))
	s.mux.Handle("/api/networks", s.authMiddleware(http.HandlerFunc(s.networksHandler)))
	s.mux.Handle("/api/channels", s.authMiddleware(http.HandlerFunc(s.channelsHandler)))
}

func (s *Server) healthCheck(w http.ResponseWriter, r *http.Request) {
	_, err := s.dialAdmin(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) usersHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		resp, _ := s.sendAdminCommand(r.Context(), []string{"user", "status"})
		w.Write([]byte(resp))
	case "POST":
		var req UserRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		cmd := []string{"user", "create", "-username", req.Username, "-password", req.Password}
		if req.Admin {
			cmd = append(cmd, "-admin", "true")
		}
		resp, err := s.sendAdminCommand(r.Context(), cmd)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write([]byte(resp))
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) userHandler(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimPrefix(r.URL.Path, "/api/users/")
	switch r.Method {
	case "GET":
		resp, _ := s.sendAdminCommand(r.Context(), []string{"user", "status", username})
		w.Write([]byte(resp))
	case "DELETE":
		resp, _ := s.sendAdminCommand(r.Context(), []string{"user", "delete", username})
		w.Write([]byte(resp))
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) networksHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		resp, _ := s.sendAdminCommand(r.Context(), []string{"network", "status"})
		w.Write([]byte(resp))
	case "POST":
		var req struct {
			User string `json:"user"`
			Addr string `json:"addr"`
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		cmd := []string{"user", "run", req.User, "network", "create", "-addr", req.Addr}
		if req.Name != "" {
			cmd = append(cmd, "-name", req.Name)
		}
		resp, err := s.sendAdminCommand(r.Context(), cmd)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write([]byte(resp))
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) channelsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		resp, _ := s.sendAdminCommand(r.Context(), []string{"channel", "status"})
		w.Write([]byte(resp))
	case "POST":
		var req struct {
			User    string `json:"user"`
			Network string `json:"network"`
			Name    string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		// user run <user> channel join <net> <channel>
		cmd := []string{"user", "run", req.User, "channel", "join", req.Network, req.Name}
		resp, err := s.sendAdminCommand(r.Context(), cmd)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write([]byte(resp))
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}
