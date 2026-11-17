package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"shopcore/audit"
	"shopcore/clock"
	"shopcore/ids"
	"shopcore/inventory"
	"shopcore/loyalty"
	"shopcore/orders"
	"shopcore/payments"
	"shopcore/refunds"
	"shopcore/shipping"
)

type fixture struct {
	srv      *Server
	rec      *audit.Memory
	payments *payments.Service
	stock    *inventory.Service
	loyalty  *loyalty.Service
	shipping *shipping.Service
}

func newFixture() *fixture {
	clk := clock.NewFixed(time.Date(2024, 3, 1, 9, 0, 0, 0, time.UTC))
	seq := ids.NewSequence()
	rec := audit.NewMemory(clk)
	f := &fixture{rec: rec}
	f.payments = payments.NewService(payments.NewMemoryRepo(), rec, clk).WithIDs(seq)
	f.stock = inventory.NewService(inventory.NewMemoryRepo(), inventory.WithAudit(rec), inventory.WithClock(clk))
	f.loyalty = loyalty.NewService(loyalty.NewMemoryRepo(), rec, clk)
	f.shipping = shipping.NewService(shipping.NewMemoryShipmentStore(), rec, clk, seq)
	f.srv = New(Services{
		Orders:    orders.NewService(orders.Config{Repo: orders.NewMemoryRepo(), Audit: rec, Clock: clk, IDs: seq}),
		Payments:  f.payments,
		Refunds:   refunds.New(refunds.Deps{Repo: refunds.NewMemoryRepo(), Audit: rec, Clock: clk, IDs: seq}),
		Inventory: f.stock,
		Loyalty:   f.loyalty,
		Shipping:  f.shipping,
	}, seq)
	return f
}

func (f *fixture) do(t *testing.T, method, path, body string, hdr map[string]string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	rr := httptest.NewRecorder()
	f.srv.ServeHTTP(rr, req)
	var out map[string]any
	if rr.Body.Len() > 0 && strings.HasPrefix(rr.Header().Get("Content-Type"), "application/json") {
		if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
			t.Fatalf("bad json %q: %v", rr.Body.String(), err)
		}
	}
	return rr, out
}

func TestRequestIDHeader(t *testing.T) {
	f := newFixture()
	rr, _ := f.do(t, "GET", "/healthz", "", map[string]string{"X-Request-ID": "abc-123"})
	if rr.Code != http.StatusNoContent || rr.Header().Get("X-Request-ID") != "abc-123" {
		t.Fatalf("code=%d rid=%q", rr.Code, rr.Header().Get("X-Request-ID"))
	}
	rr, _ = f.do(t, "GET", "/healthz", "", nil)
	if rid := rr.Header().Get("X-Request-ID"); !ids.HasPrefix(rid, "req") {
		t.Fatalf("generated rid = %q", rid)
	}
}

func TestPlaceAndGetOrder(t *testing.T) {
	f := newFixture()
	rr, body := f.do(t, "POST", "/v2/orders", `{"customer_id":"cus-1","region":"US-NY","lines":[{"sku":"A","qty":2,"unit_price":1250}]}`, nil)
	if rr.Code != http.StatusCreated || body["total"].(float64) != 2600 {
		t.Fatalf("code=%d body=%v", rr.Code, body)
	}
	rr, body = f.do(t, "GET", "/v2/orders/"+body["id"].(string), "", nil)
	if rr.Code != http.StatusOK || body["status"] != "placed" {
		t.Fatalf("code=%d body=%v", rr.Code, body)
	}
	rr, body = f.do(t, "POST", "/v2/orders", `{"customer_id":"cus-1","region":"US-NY","lines":[]}`, map[string]string{"X-Request-ID": "rid-9"})
	if rr.Code != http.StatusUnprocessableEntity || body["request_id"] != "rid-9" {
		t.Fatalf("code=%d body=%v", rr.Code, body)
	}
	rr, _ = f.do(t, "GET", "/v2/orders/ord-404", "", nil)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("code=%d", rr.Code)
	}
}

func TestPaymentStockLoyaltyShipping(t *testing.T) {
	f := newFixture()
	ctx := context.Background()
	p, _ := f.payments.Authorize(ctx, "ord-1", 1000, "stripe")
	rr, body := f.do(t, "POST", "/v2/payments/"+p.ID+"/capture", `{"amount":1500}`, nil)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("over-capture code=%d body=%v", rr.Code, body)
	}
	rr, body = f.do(t, "POST", "/v2/payments/"+p.ID+"/capture", `{"amount":1000}`, nil)
	if rr.Code != http.StatusOK || body["status"] != "captured" {
		t.Fatalf("capture code=%d body=%v", rr.Code, body)
	}

	_, _ = f.stock.Receive(ctx, "SKU-1", "ams", 3)
	rr, body = f.do(t, "POST", "/v2/inventory/SKU-1/reserve", `{"qty":2}`, nil)
	if rr.Code != http.StatusOK || body["available"].(float64) != 1 {
		t.Fatalf("reserve code=%d body=%v", rr.Code, body)
	}
	rr, _ = f.do(t, "POST", "/v2/inventory/SKU-1/reserve", `{"qty":2}`, nil)
	if rr.Code != http.StatusConflict {
		t.Fatalf("over-reserve code=%d", rr.Code)
	}

	a, _ := f.loyalty.Open(ctx, "cus-1")
	_, _ = f.loyalty.Accrue(ctx, a.ID, 500)
	rr, body = f.do(t, "POST", "/v2/loyalty/"+a.ID+"/redeem", `{"points":200}`, nil)
	if rr.Code != http.StatusOK || body["balance"].(float64) != 300 {
		t.Fatalf("redeem code=%d body=%v", rr.Code, body)
	}

	sh, _ := f.shipping.Create(ctx, "ord-1", "dhl", 800)
	rr, body = f.do(t, "POST", "/v2/shipments/"+sh.ID+"/dispatch", `{"tracking_no":"JD01"}`, map[string]string{"X-Request-ID": "rid-ship"})
	if rr.Code != http.StatusOK || body["status"] != "dispatched" {
		t.Fatalf("dispatch code=%d body=%v", rr.Code, body)
	}
	evs := f.rec.Named("shipment.dispatched")
	if len(evs) != 1 || evs[0].RequestID != "rid-ship" {
		t.Fatalf("dispatch audit = %v", evs)
	}
}

func TestRefundEndpoint(t *testing.T) {
	f := newFixture()
	rr, body := f.do(t, "POST", "/v2/refunds", `{"payment_id":"pay-1","amount":300,"reason":"damaged"}`, nil)
	if rr.Code != http.StatusCreated || body["status"] != "issued" {
		t.Fatalf("code=%d body=%v", rr.Code, body)
	}
	rr, _ = f.do(t, "POST", "/v2/refunds", `{"payment_id":"pay-1","amount":300,"reason":"meh"}`, nil)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("code=%d", rr.Code)
	}
}
