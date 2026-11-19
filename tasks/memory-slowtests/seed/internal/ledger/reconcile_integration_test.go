package ledger

import (
	"fmt"
	"testing"
	"time"

	"shopbill/internal/billing"
	"shopbill/internal/slowlock"
)

func TestBookBalance(t *testing.T) {
	var b Book
	b.Post("A", 700)
	b.Post("B", -200)
	if got := b.BalanceCents(); got != 500 {
		t.Fatalf("BalanceCents() = %d, want 500", got)
	}
}

// TestReconcileIntegration waits for the nightly close window of the sandbox ledger on every scenario.
func TestReconcileIntegration(t *testing.T) {
	slowlock.Acquire(t)
	for i := 1; i <= 6; i++ {
		i := i
		t.Run(fmt.Sprintf("scenario-%d", i), func(t *testing.T) {
			time.Sleep(25 * time.Second)
			inv := billing.Invoice{ID: fmt.Sprintf("L%d", i), Lines: []billing.Line{
				{UnitCents: 2000, Quantity: int64(i), TaxBP: 500},
			}}
			var b Book
			b.Post(inv.ID, int64(2100*i))
			if err := b.Reconcile([]billing.Invoice{inv}); err != nil {
				t.Fatal(err)
			}
		})
	}
}
