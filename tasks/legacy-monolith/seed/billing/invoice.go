package billing

import (
	"fmt"
	"strings"
	"time"

	"shopcore/money"
)

// Invoice is a customer invoice issued through the v1 API.
type Invoice struct {
	ID         string
	CustomerID string // owning customer
	Number     string // human readable invoice number
	AmountDue  money.Cents
	Currency   string // ISO 4217; empty means USD
	DueAt      time.Time
	Lines      int // number of invoice lines
	Status     InvoiceStatus
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// InvoiceStatus is the lifecycle state of a invoice.
type InvoiceStatus int

const (
	InvoiceStatusUnknown InvoiceStatus = iota
	InvoiceStatusDraft
	InvoiceStatusOpen
	InvoiceStatusPaid
	InvoiceStatusVoid
)

var invoiceStatusNames = []string{"unknown", "draft", "open", "paid", "void"}

func (s InvoiceStatus) String() string {
	if s < 0 || int(s) >= len(invoiceStatusNames) {
		return fmt.Sprintf("InvoiceStatus(%d)", int(s))
	}
	return invoiceStatusNames[s]
}

// ParseInvoiceStatus parses the wire name of a status. Matching is
// case-insensitive because the old importer wrote upper-case values.
func ParseInvoiceStatus(v string) (InvoiceStatus, error) {
	v = strings.ToLower(strings.TrimSpace(v))
	for i, n := range invoiceStatusNames {
		if i > 0 && n == v {
			return InvoiceStatus(i), nil
		}
	}
	return InvoiceStatusUnknown, fmt.Errorf("%w: unknown invoice status %q", ErrInvalid, v)
}

var invoiceTransitions = map[InvoiceStatus][]InvoiceStatus{
	InvoiceStatusDraft: {InvoiceStatusOpen, InvoiceStatusVoid},
	InvoiceStatusOpen:  {InvoiceStatusPaid, InvoiceStatusVoid},
	InvoiceStatusPaid:  {InvoiceStatusVoid},
}

// CanTransition reports whether a invoice may move from s to next.
func (s InvoiceStatus) CanTransition(next InvoiceStatus) bool {
	for _, t := range invoiceTransitions[s] {
		if t == next {
			return true
		}
	}
	return false
}

// Terminal reports whether no further transitions are possible.
func (s InvoiceStatus) Terminal() bool { return len(invoiceTransitions[s]) == 0 }

// Validate checks the invariants every stored invoice must satisfy.
func (v *Invoice) Validate() error {
	if strings.TrimSpace(v.ID) == "" {
		return fmt.Errorf("%w: invoice.id is required", ErrInvalid)
	}
	if strings.TrimSpace(v.CustomerID) == "" {
		return fmt.Errorf("%w: invoice.customer_id is required", ErrInvalid)
	}
	if strings.TrimSpace(v.Number) == "" {
		return fmt.Errorf("%w: invoice.number is required", ErrInvalid)
	}
	if v.AmountDue < 0 {
		return fmt.Errorf("%w: invoice.amount_due must not be negative", ErrInvalid)
	}
	if v.Lines < 0 {
		return fmt.Errorf("%w: invoice.lines must not be negative", ErrInvalid)
	}
	if v.Status <= InvoiceStatusUnknown || int(v.Status) >= len(invoiceStatusNames) {
		return fmt.Errorf("%w: invoice.status is not a known status", ErrInvalid)
	}
	return nil
}

// Clone returns a deep copy so callers can mutate the result freely.
func (v *Invoice) Clone() *Invoice {
	if v == nil {
		return nil
	}
	cp := *v
	return &cp
}
