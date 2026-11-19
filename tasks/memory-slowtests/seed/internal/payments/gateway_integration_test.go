package payments

import (
	"fmt"
	"testing"
	"time"

	"shopbill/internal/billing"
	"shopbill/internal/testlock"
)

// fakeGateway settles each charge after a delay.
type fakeGateway struct {
	settle  time.Duration
	charged []int64
}

func (g *fakeGateway) Charge(cents int64) (string, error) {
	time.Sleep(g.settle)
	g.charged = append(g.charged, cents)
	return fmt.Sprintf("ch_%d", len(g.charged)), nil
}

func TestChargeInvoiceRejectsRefunds(t *testing.T) {
	g := &fakeGateway{}
	inv := billing.Invoice{ID: "R1", Lines: []billing.Line{{UnitCents: 500, Quantity: -1}}}
	if _, err := ChargeInvoice(g, inv); err == nil {
		t.Fatal("expected an error for a refund-only invoice")
	}
}

// TestGatewayIntegration runs the gateway scenarios one after another.
func TestGatewayIntegration(t *testing.T) {
	testlock.Acquire(t)
	for i := 1; i <= 6; i++ {
		i := i
		t.Run(fmt.Sprintf("scenario-%d", i), func(t *testing.T) {
			g := &fakeGateway{settle: testlock.Window}
			inv := billing.Invoice{ID: fmt.Sprintf("P%d", i), Lines: []billing.Line{
				{UnitCents: 1000, Quantity: int64(i), TaxBP: 1000},
			}}
			id, err := ChargeInvoice(g, inv)
			if err != nil {
				t.Fatal(err)
			}
			if id != "ch_1" || len(g.charged) != 1 || g.charged[0] != int64(1100*i) {
				t.Fatalf("charge %q %v, want ch_1 [%d]", id, g.charged, 1100*i)
			}
		})
	}
}
