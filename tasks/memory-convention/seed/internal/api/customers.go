package api

import (
	"net/http"

	"payments/internal/apperr"
	"payments/internal/store"
	"payments/internal/xlog"
)

var customersLog = xlog.For("customers")

func (s *Server) CreateCustomer(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if !decodeBody(w, r, &in) {
		return
	}
	if in.Name == "" || in.Email == "" {
		customersLog.Warn("customer rejected", "reason", "missing field")
		writeError(w, http.StatusBadRequest, apperr.ErrMissingField, "name and email are required")
		return
	}
	c, ok := s.store.AddCustomer(store.Customer{Name: in.Name, Email: in.Email})
	if !ok {
		customersLog.Warn("customer rejected", "reason", "email taken")
		writeError(w, http.StatusConflict, apperr.ErrCustomerEmailTaken, "email is already registered")
		return
	}
	customersLog.Info("customer created", "id", c.ID)
	writeJSON(w, http.StatusCreated, c)
}

func (s *Server) GetCustomer(w http.ResponseWriter, r *http.Request) {
	c, ok := s.store.Customer(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, apperr.ErrCustomerNotFound, "customer not found")
		return
	}
	writeJSON(w, http.StatusOK, c)
}
