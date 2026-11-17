package returns

import (
	"fmt"
	"strings"
	"time"
)

// ReturnRequest is a customer request to send items back.
type ReturnRequest struct {
	ID        string
	OrderID   string
	SKU       string
	Qty       int
	Reason    string
	Inspector string // warehouse inspector
	Status    ReturnRequestStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ReturnRequestStatus is the lifecycle state of a return request.
type ReturnRequestStatus int

const (
	ReturnRequestStatusUnknown ReturnRequestStatus = iota
	ReturnRequestStatusRequested
	ReturnRequestStatusAuthorized
	ReturnRequestStatusReceived
	ReturnRequestStatusRejected
)

var returnRequestStatusNames = []string{"unknown", "requested", "authorized", "received", "rejected"}

func (s ReturnRequestStatus) String() string {
	if s < 0 || int(s) >= len(returnRequestStatusNames) {
		return fmt.Sprintf("ReturnRequestStatus(%d)", int(s))
	}
	return returnRequestStatusNames[s]
}

// ParseReturnRequestStatus parses the wire name of a status. Matching is
// case-insensitive because the old importer wrote upper-case values.
func ParseReturnRequestStatus(v string) (ReturnRequestStatus, error) {
	v = strings.ToLower(strings.TrimSpace(v))
	for i, n := range returnRequestStatusNames {
		if i > 0 && n == v {
			return ReturnRequestStatus(i), nil
		}
	}
	return ReturnRequestStatusUnknown, fmt.Errorf("%w: unknown return request status %q", ErrInvalid, v)
}

var returnRequestTransitions = map[ReturnRequestStatus][]ReturnRequestStatus{
	ReturnRequestStatusRequested:  {ReturnRequestStatusAuthorized, ReturnRequestStatusRejected},
	ReturnRequestStatusAuthorized: {ReturnRequestStatusReceived, ReturnRequestStatusRejected},
	ReturnRequestStatusReceived:   {ReturnRequestStatusRejected},
}

// CanTransition reports whether a return request may move from s to next.
func (s ReturnRequestStatus) CanTransition(next ReturnRequestStatus) bool {
	for _, t := range returnRequestTransitions[s] {
		if t == next {
			return true
		}
	}
	return false
}

// Terminal reports whether no further transitions are possible.
func (s ReturnRequestStatus) Terminal() bool { return len(returnRequestTransitions[s]) == 0 }

// Validate checks the invariants every stored return request must satisfy.
func (v *ReturnRequest) Validate() error {
	if strings.TrimSpace(v.ID) == "" {
		return fmt.Errorf("%w: return_request.id is required", ErrInvalid)
	}
	if strings.TrimSpace(v.OrderID) == "" {
		return fmt.Errorf("%w: return_request.order_id is required", ErrInvalid)
	}
	if strings.TrimSpace(v.SKU) == "" {
		return fmt.Errorf("%w: return_request.sku is required", ErrInvalid)
	}
	if v.Qty < 0 {
		return fmt.Errorf("%w: return_request.qty must not be negative", ErrInvalid)
	}
	if v.Status <= ReturnRequestStatusUnknown || int(v.Status) >= len(returnRequestStatusNames) {
		return fmt.Errorf("%w: return_request.status is not a known status", ErrInvalid)
	}
	return nil
}

// Clone returns a deep copy so callers can mutate the result freely.
func (v *ReturnRequest) Clone() *ReturnRequest {
	if v == nil {
		return nil
	}
	cp := *v
	return &cp
}
