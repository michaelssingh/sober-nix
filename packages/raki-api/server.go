package adminapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
)

type Server struct {
	adminSocketPath string
	mux             *http.ServeMux
	apiKeys         map[string]struct{}
	logger          *slog.Logger
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
	// Single generic endpoint for integration testing
	s.mux.Handle("/api/exec", s.authMiddleware(http.HandlerFunc(s.execHandler)))
}

func (s *Server) healthCheck(w http.ResponseWriter, r *http.Request) {
	_, err := s.dialAdmin(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) execHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Args []string `json:"args"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp, err := s.sendAdminCommand(r.Context(), req.Args)
	if err != nil {
		s.logger.Error("command failed", "args", req.Args, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write([]byte(resp))
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}
