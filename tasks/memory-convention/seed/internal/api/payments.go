package api

import (
	"errors"
	"net/http"

	"payments/internal/apperr"
	"payments/internal/store"
	"payments/internal/xlog"
)

var paymentsLog = xlog.For("payments")

var currencies = map[string]bool{"USD": true, "EUR": true, "GBP": true}

func (s *Server) CreatePayment(w http.ResponseWriter, r *http.Request) {
	var in struct {
		CustomerID  string `json:"customer_id"`
		AmountCents int64  `json:"amount_cents"`
		Currency    string `json:"currency"`
	}
	if !decodeBody(w, r, &in) {
		return
	}
	switch {
	case in.CustomerID == "" || in.Currency == "":
		paymentsLog.Warn("payment rejected", "reason", "missing field")
		writeError(w, http.StatusBadRequest, apperr.ErrMissingField, "customer_id and currency are required")
		return
	case in.AmountCents <= 0:
		paymentsLog.Warn("payment rejected", "reason", "amount not positive")
		writeError(w, http.StatusBadRequest, apperr.ErrAmountNotPositive, "amount_cents must be positive")
		return
	case !currencies[in.Currency]:
		paymentsLog.Warn("payment rejected", "reason", "unsupported currency")
		writeError(w, http.StatusBadRequest, apperr.ErrUnsupportedCurrency, "currency is not supported")
		return
	}
	if _, ok := s.store.Customer(in.CustomerID); !ok {
		paymentsLog.Warn("payment rejected", "reason", "customer not found")
		writeError(w, http.StatusNotFound, apperr.ErrCustomerNotFound, "customer not found")
		return
	}
	p := s.store.AddPayment(store.Payment{CustomerID: in.CustomerID, AmountCents: in.AmountCents, Currency: in.Currency})
	paymentsLog.Info("payment authorized", "id", p.ID, "amount_cents", p.AmountCents)
	writeJSON(w, http.StatusCreated, p)
}

func (s *Server) GetPayment(w http.ResponseWriter, r *http.Request) {
	p, ok := s.store.Payment(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, apperr.ErrPaymentNotFound, "payment not found")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) CapturePayment(w http.ResponseWriter, r *http.Request) {
	s.transition(w, r, "captured")
}

func (s *Server) VoidPayment(w http.ResponseWriter, r *http.Request) {
	s.transition(w, r, "voided")
}

var errNotAuthorized = errors.New("not authorized")

// transition moves an authorized payment to the given status.
func (s *Server) transition(w http.ResponseWriter, r *http.Request, to string) {
	id := r.PathValue("id")
	p, found, err := s.store.UpdatePayment(id, func(p *store.Payment) error {
		if p.Status != "authorized" {
			return errNotAuthorized
		}
		p.Status = to
		return nil
	})
	switch {
	case !found:
		paymentsLog.Warn("transition rejected", "id", id, "reason", "not found")
		writeError(w, http.StatusNotFound, apperr.ErrPaymentNotFound, "payment not found")
	case err != nil:
		paymentsLog.Warn("transition rejected", "id", id, "reason", "not authorized")
		writeError(w, http.StatusConflict, apperr.ErrPaymentNotAuthorized, "payment is not authorized")
	default:
		paymentsLog.Info("payment "+to, "id", p.ID)
		writeJSON(w, http.StatusOK, p)
	}
}
