package acceptance

import "testing"

func TestM1RefundFlow(t *testing.T) {
	e := newEnv(t)
	id := e.payment(10000, true)

	r := e.mustCall(201, "POST", "/refunds", refundBody(id, 4000))
	if r["id"] == nil || r["id"] == "" {
		t.Fatalf("refund has no id: %v", r)
	}
	if r["payment_id"] != id || r["amount_cents"] != float64(4000) || r["reason"] != "damaged" || r["status"] != "succeeded" {
		t.Fatalf("unexpected refund body: %v", r)
	}
	if got := e.refunded(id); got != 4000 {
		t.Fatalf("refunded_cents = %v, want 4000", got)
	}

	// reason is optional
	e.mustCall(201, "POST", "/refunds", `{"payment_id":"`+id+`","amount_cents":1000}`)
	if got := e.refunded(id); got != 5000 {
		t.Fatalf("refunded_cents = %v, want 5000", got)
	}

	// the exact remainder is allowed, one cent more is not
	e.mustCall(201, "POST", "/refunds", refundBody(id, 5000))
	if got := e.refunded(id); got != 10000 {
		t.Fatalf("refunded_cents = %v, want 10000", got)
	}
	e.mustCall(422, "POST", "/refunds", refundBody(id, 1))
}

func TestM1RefundRejections(t *testing.T) {
	e := newEnv(t)
	captured := e.payment(10000, true)
	authorized := e.payment(500, false)
	voided := e.payment(500, false)
	e.mustCall(200, "POST", "/payments/"+voided+"/void", "")
	e.mustCall(201, "POST", "/refunds", refundBody(captured, 9000))

	cases := []struct {
		name   string
		body   string
		status int
	}{
		{"not json", `{"payment_id":`, 400},
		{"missing payment_id", `{"amount_cents":100}`, 400},
		{"zero amount", refundBody(captured, 0), 400},
		{"negative amount", refundBody(captured, -5), 400},
		{"unknown payment", refundBody("pay_999", 100), 404},
		{"authorized payment", refundBody(authorized, 100), 409},
		{"voided payment", refundBody(voided, 100), 409},
		{"more than remaining", refundBody(captured, 1001), 422},
	}
	for _, c := range cases {
		status, out := e.call("POST", "/refunds", c.body)
		if status != c.status {
			t.Errorf("%s: status %d, want %d (%v)", c.name, status, c.status, out)
			continue
		}
		if errorCode(t, out) == "" {
			t.Errorf("%s: empty error code", c.name)
		}
	}

	if got := e.refunded(captured); got != 9000 {
		t.Fatalf("failed refunds changed refunded_cents to %v, want 9000", got)
	}
	if got := e.refunded(authorized); got != 0 {
		t.Fatalf("authorized payment refunded_cents = %v, want 0", got)
	}
}
