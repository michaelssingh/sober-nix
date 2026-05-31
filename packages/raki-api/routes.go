package adminapi

import "net/http"

func (s *Server) setupRoutes() {
    s.mux.HandleFunc("/health", s.healthCheck)
    s.mux.Handle("/api/users", s.authMiddleware(http.HandlerFunc(s.listUsers)))
    // ... etc
}
