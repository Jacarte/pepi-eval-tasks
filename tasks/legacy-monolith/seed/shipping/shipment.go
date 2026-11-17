package shipping

import (
	"fmt"
	"strings"
	"time"
)

// Shipment is a parcel handed to a carrier.
type Shipment struct {
	ID          string
	OrderID     string
	Carrier     string
	TrackingNo  string
	WeightGrams int
	Service     string // carrier service level
	HoldReason  string
	Status      ShipmentStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ShipmentStatus is the lifecycle state of a shipment.
type ShipmentStatus int

const (
	ShipmentStatusUnknown ShipmentStatus = iota
	ShipmentStatusCreated
	ShipmentStatusDispatched
	ShipmentStatusHeld
	ShipmentStatusDelivered
)

var shipmentStatusNames = []string{"unknown", "created", "dispatched", "held", "delivered"}

func (s ShipmentStatus) String() string {
	if s < 0 || int(s) >= len(shipmentStatusNames) {
		return fmt.Sprintf("ShipmentStatus(%d)", int(s))
	}
	return shipmentStatusNames[s]
}

// ParseShipmentStatus parses the wire name of a status. Matching is
// case-insensitive because the old importer wrote upper-case values.
func ParseShipmentStatus(v string) (ShipmentStatus, error) {
	v = strings.ToLower(strings.TrimSpace(v))
	for i, n := range shipmentStatusNames {
		if i > 0 && n == v {
			return ShipmentStatus(i), nil
		}
	}
	return ShipmentStatusUnknown, fmt.Errorf("%w: unknown shipment status %q", ErrInvalid, v)
}

var shipmentTransitions = map[ShipmentStatus][]ShipmentStatus{
	ShipmentStatusCreated:    {ShipmentStatusDispatched, ShipmentStatusDelivered},
	ShipmentStatusDispatched: {ShipmentStatusHeld, ShipmentStatusDelivered},
	ShipmentStatusHeld:       {ShipmentStatusDelivered},
}

// CanTransition reports whether a shipment may move from s to next.
func (s ShipmentStatus) CanTransition(next ShipmentStatus) bool {
	for _, t := range shipmentTransitions[s] {
		if t == next {
			return true
		}
	}
	return false
}

// Terminal reports whether no further transitions are possible.
func (s ShipmentStatus) Terminal() bool { return len(shipmentTransitions[s]) == 0 }

// Validate checks the invariants every stored shipment must satisfy.
func (v *Shipment) Validate() error {
	if strings.TrimSpace(v.ID) == "" {
		return &ValidationError{Entity: "shipment", Field: "id", Reason: "is required"}
	}
	if strings.TrimSpace(v.OrderID) == "" {
		return &ValidationError{Entity: "shipment", Field: "order_id", Reason: "is required"}
	}
	if strings.TrimSpace(v.Carrier) == "" {
		return &ValidationError{Entity: "shipment", Field: "carrier", Reason: "is required"}
	}
	if v.WeightGrams < 0 {
		return &ValidationError{Entity: "shipment", Field: "weight_grams", Reason: "must not be negative"}
	}
	if v.Status <= ShipmentStatusUnknown || int(v.Status) >= len(shipmentStatusNames) {
		return &ValidationError{Entity: "shipment", Field: "status", Reason: "is not a known status"}
	}
	return nil
}

// Clone returns a deep copy so callers can mutate the result freely.
func (v *Shipment) Clone() *Shipment {
	if v == nil {
		return nil
	}
	cp := *v
	return &cp
}
