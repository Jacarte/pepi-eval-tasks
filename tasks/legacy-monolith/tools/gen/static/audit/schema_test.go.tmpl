package audit

import (
	"errors"
	"testing"
)

func TestValidateName(t *testing.T) {
	for _, ok := range []string{"shipment.dispatched", "purchase_order.approved", "bin_slot.allocated", "shipment.held", "customer.registered", "product_listing.published"} {
		if err := ValidateName(ok); err != nil {
			t.Errorf("ValidateName(%q) = %v", ok, err)
		}
	}
	for _, bad := range []string{"", "shipment", "ShipmentDispatched", "shipment_dispatched", "Shipment.Dispatched", "shipping.dispatch", "warehouse.allocate", "suppliers.onboard", "shipment.dispatch", "shipment-dispatched", "invoice_created", ".created", "shipment."} {
		if err := ValidateName(bad); !errors.Is(err, ErrBadName) {
			t.Errorf("ValidateName(%q) = %v, want ErrBadName", bad, err)
		}
	}
}

func TestValidateEvent(t *testing.T) {
	good := Event{Name: "coupon.redeemed", RequestID: "req-1", Subject: "cpn-1", Attrs: map[string]string{"order_id": "ord-1"}}
	if err := Validate(good); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		ev   Event
		want error
	}{
		{Event{Name: "coupon.redeem", RequestID: "req-1", Subject: "cpn-1"}, ErrBadName},
		{Event{Name: "coupon.redeemed", Subject: "cpn-1"}, ErrMissingRequestID},
		{Event{Name: "coupon.redeemed", RequestID: "req-1"}, ErrMissingSubject},
		{Event{Name: "coupon.redeemed", RequestID: "req-1", Subject: "cpn-1", Attrs: map[string]string{"orderID": "x"}}, ErrBadAttrKey},
	}
	for _, tc := range cases {
		if err := Validate(tc.ev); !errors.Is(err, tc.want) {
			t.Errorf("Validate(%+v) = %v, want %v", tc.ev, err, tc.want)
		}
	}
}
