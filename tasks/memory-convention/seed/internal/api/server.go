// Package api serves the HTTP API.
package api

import (
	"net/http"

	"payments/internal/store"
)

// Server routes requests to the handlers below.
type Server struct {
	store *store.Memory
	mux   *http.ServeMux
}

func NewServer(s *store.Memory) *Server {
	srv := &Server{store: s, mux: http.NewServeMux()}
	srv.mux.HandleFunc("GET /healthz", srv.Healthz)
	srv.mux.HandleFunc("GET /debug/routes", srv.ListRoutes)
	srv.mux.HandleFunc("POST /customers", srv.CreateCustomer)
	srv.mux.HandleFunc("GET /customers/{id}", srv.GetCustomer)
	srv.mux.HandleFunc("POST /payments", srv.CreatePayment)
	srv.mux.HandleFunc("GET /payments/{id}", srv.GetPayment)
	srv.mux.HandleFunc("POST /payments/{id}/capture", srv.CapturePayment)
	srv.mux.HandleFunc("POST /payments/{id}/void", srv.VoidPayment)
	return srv
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }
