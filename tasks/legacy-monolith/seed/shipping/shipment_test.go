package shipping

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func sampleShipment(n int) *Shipment {
	return &Shipment{
		ID:          fmt.Sprintf("shipment-%03d", n),
		OrderID:     fmt.Sprintf("orderid %d;a=b,c\\d", n),
		Carrier:     fmt.Sprintf("carrier-%d", n),
		TrackingNo:  fmt.Sprintf("trackingno %d;a=b,c\\d", n),
		WeightGrams: n*26 + 409,
		Service:     fmt.Sprintf("service %d;a=b,c\\d", n),
		HoldReason:  fmt.Sprintf("holdreason-%d", n),
		Status:      ShipmentStatusCreated,
		CreatedAt:   time.Date(2023, time.January, n, 8, 0, 0, 0, time.UTC),
	}
}

func TestShipmentCodecRoundTrip(t *testing.T) {
	in := sampleShipment(4)
	line := EncodeShipment(in)
	out, err := DecodeShipment(line)
	if err != nil {
		t.Fatalf("DecodeShipment(%q): %v", line, err)
	}
	if got := EncodeShipment(out); got != line {
		t.Fatalf("round trip mismatch:\n got %s\nwant %s", got, line)
	}
	if out.OrderID != in.OrderID {
		t.Errorf("OrderID = %q, want %q", out.OrderID, in.OrderID)
	}
	if out.Carrier != in.Carrier {
		t.Errorf("Carrier = %q, want %q", out.Carrier, in.Carrier)
	}
	if out.TrackingNo != in.TrackingNo {
		t.Errorf("TrackingNo = %q, want %q", out.TrackingNo, in.TrackingNo)
	}
	if out.Service != in.Service {
		t.Errorf("Service = %q, want %q", out.Service, in.Service)
	}
	if out.HoldReason != in.HoldReason {
		t.Errorf("HoldReason = %q, want %q", out.HoldReason, in.HoldReason)
	}
	if _, err := DecodeShipment("id"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("malformed input: err = %v, want ErrInvalid", err)
	}
}

func TestMemoryShipmentStore(t *testing.T) {
	ctx := context.Background()
	st := NewMemoryShipmentStore()
	for n := 1; n <= 3; n++ {
		if err := st.Put(ctx, sampleShipment(n)); err != nil {
			t.Fatalf("Put(%d): %v", n, err)
		}
	}
	got, err := st.Get(ctx, "shipment-002")
	if err != nil {
		t.Fatal(err)
	}
	got.Status = ShipmentStatusDelivered
	again, _ := st.Get(ctx, "shipment-002")
	if again.Status != ShipmentStatusCreated {
		t.Fatal("store handed out a shared pointer")
	}
	if err := st.Put(ctx, got); err != nil {
		t.Fatal(err)
	}
	list, err := st.List(ctx, ShipmentFilter{Status: ShipmentStatusCreated})
	if err != nil || len(list) != 2 || list[0].ID != "shipment-001" {
		t.Fatalf("List by status = %v, %v", list, err)
	}
	if err := st.Delete(ctx, "shipment-001"); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Get(ctx, "shipment-001"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("after delete: err = %v, want ErrNotFound", err)
	}
	if st.Len() != 2 {
		t.Fatalf("Len = %d, want 2", st.Len())
	}
}

func TestShipmentStatusTransitions(t *testing.T) {
	for _, s := range []ShipmentStatus{ShipmentStatusCreated, ShipmentStatusDelivered} {
		got, err := ParseShipmentStatus(s.String())
		if err != nil || got != s {
			t.Fatalf("Parse(%q) = %v, %v", s.String(), got, err)
		}
	}
	if !ShipmentStatusCreated.CanTransition(ShipmentStatusDispatched) {
		t.Fatal("first transition should be allowed")
	}
	if ShipmentStatusDispatched.CanTransition(ShipmentStatusCreated) {
		t.Fatal("backwards transition allowed")
	}
	if !ShipmentStatusDelivered.Terminal() {
		t.Fatal("last status should be terminal")
	}
}

func TestSummarizeShipments(t *testing.T) {
	items := []*Shipment{sampleShipment(1), nil, sampleShipment(2), sampleShipment(3)}
	items[2].Status = ShipmentStatusDispatched
	sum := SummarizeShipments(items)
	if sum.Count != 3 || sum.ByStatus[ShipmentStatusCreated] != 2 || sum.ByStatus[ShipmentStatusDispatched] != 1 {
		t.Fatalf("counts = %+v", sum)
	}
	if sum.TotalWeightGrams != 1383 {
		t.Errorf("TotalWeightGrams = %v, want 1383", sum.TotalWeightGrams)
	}
	if !sum.Earliest.Equal(items[0].CreatedAt) || !sum.Latest.Equal(items[3].CreatedAt) {
		t.Errorf("range = %v..%v", sum.Earliest, sum.Latest)
	}
	if top := TopShipmentsByWeightGrams(items[2:], 1); len(top) != 1 || top[0].ID != "shipment-003" {
		t.Errorf("top = %v", top)
	}
}
