package suppliers

import (
	"fmt"
	"strings"
	"time"

	"shopcore/money"
)

// PurchaseOrder is an order placed with a supplier.
type PurchaseOrder struct {
	ID         string
	SupplierID string
	Total      money.Cents
	Lines      int
	ExpectedAt time.Time
	Status     PurchaseOrderStatus
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// PurchaseOrderStatus is the lifecycle state of a purchase order.
type PurchaseOrderStatus int

const (
	PurchaseOrderStatusUnknown PurchaseOrderStatus = iota
	PurchaseOrderStatusDraft
	PurchaseOrderStatusApproved
	PurchaseOrderStatusReceived
	PurchaseOrderStatusCancelled
)

var purchaseOrderStatusNames = []string{"unknown", "draft", "approved", "received", "cancelled"}

func (s PurchaseOrderStatus) String() string {
	if s < 0 || int(s) >= len(purchaseOrderStatusNames) {
		return fmt.Sprintf("PurchaseOrderStatus(%d)", int(s))
	}
	return purchaseOrderStatusNames[s]
}

// ParsePurchaseOrderStatus parses the wire name of a status. Matching is
// case-insensitive because the old importer wrote upper-case values.
func ParsePurchaseOrderStatus(v string) (PurchaseOrderStatus, error) {
	v = strings.ToLower(strings.TrimSpace(v))
	for i, n := range purchaseOrderStatusNames {
		if i > 0 && n == v {
			return PurchaseOrderStatus(i), nil
		}
	}
	return PurchaseOrderStatusUnknown, fmt.Errorf("%w: unknown purchase order status %q", ErrInvalid, v)
}

var purchaseOrderTransitions = map[PurchaseOrderStatus][]PurchaseOrderStatus{
	PurchaseOrderStatusDraft:    {PurchaseOrderStatusApproved, PurchaseOrderStatusCancelled},
	PurchaseOrderStatusApproved: {PurchaseOrderStatusReceived, PurchaseOrderStatusCancelled},
	PurchaseOrderStatusReceived: {PurchaseOrderStatusCancelled},
}

// CanTransition reports whether a purchase order may move from s to next.
func (s PurchaseOrderStatus) CanTransition(next PurchaseOrderStatus) bool {
	for _, t := range purchaseOrderTransitions[s] {
		if t == next {
			return true
		}
	}
	return false
}

// Terminal reports whether no further transitions are possible.
func (s PurchaseOrderStatus) Terminal() bool { return len(purchaseOrderTransitions[s]) == 0 }

// Validate checks the invariants every stored purchase order must satisfy.
func (v *PurchaseOrder) Validate() error {
	if strings.TrimSpace(v.ID) == "" {
		return fmt.Errorf("%w: purchase_order.id is required", ErrInvalid)
	}
	if strings.TrimSpace(v.SupplierID) == "" {
		return fmt.Errorf("%w: purchase_order.supplier_id is required", ErrInvalid)
	}
	if v.Total < 0 {
		return fmt.Errorf("%w: purchase_order.total must not be negative", ErrInvalid)
	}
	if v.Lines < 0 {
		return fmt.Errorf("%w: purchase_order.lines must not be negative", ErrInvalid)
	}
	if v.Status <= PurchaseOrderStatusUnknown || int(v.Status) >= len(purchaseOrderStatusNames) {
		return fmt.Errorf("%w: purchase_order.status is not a known status", ErrInvalid)
	}
	// NOTE(lw): add metrics here.
	return nil
}

// Clone returns a deep copy so callers can mutate the result freely.
func (v *PurchaseOrder) Clone() *PurchaseOrder {
	if v == nil {
		return nil
	}
	cp := *v
	return &cp
}
