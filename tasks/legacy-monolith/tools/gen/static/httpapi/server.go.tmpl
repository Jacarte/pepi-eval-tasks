// Package httpapi exposes the v2 JSON API. Handlers decode the request, call
// exactly one domain service method with the request context, and encode the
// result. Business rules do not belong here.
package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"shopcore/ctxkeys"
	"shopcore/ids"
	"shopcore/inventory"
	"shopcore/loyalty"
	"shopcore/orders"
	"shopcore/payments"
	"shopcore/refunds"
	"shopcore/shipping"
)

// Services are the domain services the API calls into.
type Services struct {
	Orders    *orders.Service
	Payments  *payments.Service
	Refunds   *refunds.Service
	Inventory *inventory.Service
	Loyalty   *loyalty.Service
	Shipping  *shipping.Service
}

// Server is the v2 API handler.
type Server struct {
	svc Services
	ids ids.Generator
	mux *http.ServeMux
}

// New builds the API. gen mints request ids for requests that arrive without
// an X-Request-ID header.
func New(svc Services, gen ids.Generator) *Server {
	s := &Server{svc: svc, ids: ids.OrRandom(gen), mux: http.NewServeMux()}
	s.routes()
	return s
}

func (s *Server) routes() {
	s.mux.HandleFunc("POST /v2/orders", s.placeOrder)
	s.mux.HandleFunc("GET /v2/orders/{id}", s.getOrder)
	s.mux.HandleFunc("POST /v2/payments/{id}/capture", s.capturePayment)
	s.mux.HandleFunc("POST /v2/refunds", s.issueRefund)
	s.mux.HandleFunc("POST /v2/inventory/{sku}/reserve", s.reserveStock)
	s.mux.HandleFunc("POST /v2/loyalty/{id}/redeem", s.redeemPoints)
	s.mux.HandleFunc("POST /v2/shipments/{id}/dispatch", s.dispatchShipment)
	s.mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })
}

// ServeHTTP attaches the request context every service expects and then
// routes the request.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rid := strings.TrimSpace(r.Header.Get("X-Request-ID"))
	if rid == "" || len(rid) > 64 {
		rid = s.ids.Next("req")
	}
	ctx := ctxkeys.WithRequestID(r.Context(), rid)
	if tr := r.Header.Get("X-Trace-ID"); tr != "" {
		ctx = ctxkeys.WithTraceID(ctx, tr)
	}
	if actor := r.Header.Get("X-Actor"); actor != "" {
		ctx = ctxkeys.WithActor(ctx, actor)
	}
	w.Header().Set("X-Request-ID", rid)
	s.mux.ServeHTTP(w, r.WithContext(ctx))
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func decode(r *http.Request, v any) error {
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}
