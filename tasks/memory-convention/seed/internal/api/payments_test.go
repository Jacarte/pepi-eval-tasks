package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"payments/internal/store"
)

func do(t *testing.T, srv http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	return rec
}

func TestCaptureLifecycle(t *testing.T) {
	srv := NewServer(store.NewMemory())
	if rec := do(t, srv, "POST", "/customers", `{"name":"Ada","email":"ada@example.com"}`); rec.Code != 201 {
		t.Fatalf("customer: %d %s", rec.Code, rec.Body)
	}
	if rec := do(t, srv, "POST", "/payments", `{"customer_id":"cus_1","amount_cents":500,"currency":"USD"}`); rec.Code != 201 {
		t.Fatalf("payment: %d %s", rec.Code, rec.Body)
	}
	if rec := do(t, srv, "POST", "/payments/pay_2/capture", ``); rec.Code != 200 {
		t.Fatalf("capture: %d %s", rec.Code, rec.Body)
	}
	if rec := do(t, srv, "POST", "/payments/pay_2/void", ``); rec.Code != 409 {
		t.Fatalf("void after capture: %d", rec.Code)
	}
}
