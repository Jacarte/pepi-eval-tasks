package api

import "net/http"

func (s *Server) Healthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) ListRoutes(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, Manifest)
}
