package acceptance

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"payments/internal/api"
	"payments/internal/store"
)

type env struct {
	t          *testing.T
	srv        http.Handler
	customerID string
}

func newEnv(t *testing.T) *env {
	t.Helper()
	e := &env{t: t, srv: api.NewServer(store.NewMemory())}
	e.customerID = e.mustCall(201, "POST", "/customers", `{"name":"Ada","email":"ada@example.com"}`)["id"].(string)
	return e
}

// call returns the status and the decoded JSON object of the response.
func (e *env) call(method, path, body string) (int, map[string]any) {
	e.t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	e.srv.ServeHTTP(rec, req)
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, out
}

func (e *env) mustCall(want int, method, path, body string) map[string]any {
	e.t.Helper()
	status, out := e.call(method, path, body)
	if status != want {
		e.t.Fatalf("%s %s: status %d, want %d (%v)", method, path, status, want, out)
	}
	return out
}

// payment creates a payment of amount cents; captured also captures it.
func (e *env) payment(amount int, captured bool) string {
	e.t.Helper()
	body := fmt.Sprintf(`{"customer_id":%q,"amount_cents":%d,"currency":"USD"}`, e.customerID, amount)
	id := e.mustCall(201, "POST", "/payments", body)["id"].(string)
	if captured {
		e.mustCall(200, "POST", "/payments/"+id+"/capture", "")
	}
	return id
}

func (e *env) refunded(paymentID string) float64 {
	e.t.Helper()
	return e.mustCall(200, "GET", "/payments/"+paymentID, "")["refunded_cents"].(float64)
}

func refundBody(paymentID string, amount int) string {
	return fmt.Sprintf(`{"payment_id":%q,"amount_cents":%d,"reason":"damaged"}`, paymentID, amount)
}

// errorCode returns error.code of a response, requiring a message too.
func errorCode(t *testing.T, out map[string]any) string {
	t.Helper()
	obj, ok := out["error"].(map[string]any)
	if !ok {
		t.Fatalf("response has no error object: %v", out)
	}
	if msg, _ := obj["message"].(string); msg == "" {
		t.Fatalf("error has no message: %v", out)
	}
	code, _ := obj["code"].(string)
	return code
}
