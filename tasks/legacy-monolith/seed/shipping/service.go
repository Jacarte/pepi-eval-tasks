// Package shipping hands parcels to carriers and tracks them to delivery.
package shipping

import (
	"context"
	"errors"
	"strings"

	"shopcore/audit"
	"shopcore/clock"
	"shopcore/ctxkeys"
	"shopcore/errs"
	"shopcore/ids"
)

var (
	ErrMissingOrder  = errors.New("shipping: order id is required")
	ErrBadTransition = errors.New("shipping: shipment cannot move to that state")
	ErrNoCarrier     = errors.New("shipping: unknown carrier")
)

// Service coordinates shipment lifecycle changes.
type Service struct {
	store ShipmentStore
	audit audit.Recorder
	clock clock.Clock
	ids   ids.Generator
}

func NewService(store ShipmentStore, rec audit.Recorder, clk clock.Clock, gen ids.Generator) *Service {
	return &Service{store: store, audit: audit.OrDiscard(rec), clock: clock.OrSystem(clk), ids: ids.OrRandom(gen)}
}

// record emits an audit event about sh, correlated with the current request.
func (s *Service) record(ctx context.Context, name string, sh *Shipment, attrs map[string]string) error {
	return s.audit.Record(ctx, audit.Event{
		Name:      name,
		RequestID: ctxkeys.RequestIDFrom(ctx),
		Subject:   sh.ID,
		Attrs:     attrs,
	})
}

// Create registers a new shipment for an order.
func (s *Service) Create(ctx context.Context, orderID, carrier string, grams int) (*Shipment, error) {
	const op = "shipping.create"
	if strings.TrimSpace(orderID) == "" {
		return nil, errs.Wrap(op, ErrMissingOrder)
	}
	if !knownCarrier(carrier) {
		return nil, errs.Wrap(op, ErrNoCarrier)
	}
	now := s.clock.Now()
	sh := &Shipment{
		ID:          s.ids.Next("shp"),
		OrderID:     orderID,
		Carrier:     carrier,
		WeightGrams: grams,
		Service:     "standard",
		Status:      ShipmentStatusCreated,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.store.Put(ctx, sh); err != nil {
		return nil, errs.Wrap(op, err)
	}
	if err := s.record(ctx, "shipment.created", sh, map[string]string{"order_id": orderID, "carrier": carrier}); err != nil {
		return nil, errs.Wrap(op, err)
	}
	return sh, nil
}

// Dispatch marks a shipment as handed to the carrier.
func (s *Service) Dispatch(ctx context.Context, id, trackingNo string) (*Shipment, error) {
	const op = "shipping.dispatch"
	sh, err := s.store.Get(ctx, id)
	if err != nil {
		return nil, errs.Wrap(op, err)
	}
	if sh.Status != ShipmentStatusCreated {
		return nil, errs.Wrap(op, ErrBadTransition)
	}
	sh.Status = ShipmentStatusDispatched
	sh.TrackingNo = trackingNo
	sh.UpdatedAt = s.clock.Now()
	if err := s.store.Put(ctx, sh); err != nil {
		return nil, errs.Wrap(op, err)
	}
	if err := s.record(ctx, "shipment.dispatched", sh, map[string]string{"tracking_no": trackingNo}); err != nil {
		return nil, errs.Wrap(op, err)
	}
	return sh, nil
}

// Hold stops a shipment, e.g. on a fraud review or an address problem.
func (s *Service) Hold(ctx context.Context, id, reason string) (*Shipment, error) {
	const op = "shipping.hold"
	sh, err := s.store.Get(ctx, id)
	if err != nil {
		return nil, errs.Wrap(op, err)
	}
	if sh.Status == ShipmentStatusDelivered || sh.Status == ShipmentStatusHeld {
		return nil, errs.Wrap(op, ErrBadTransition)
	}
	sh.Status = ShipmentStatusHeld
	sh.HoldReason = reason
	sh.UpdatedAt = s.clock.Now()
	if err := s.store.Put(ctx, sh); err != nil {
		return nil, errs.Wrap(op, err)
	}
	if err := s.record(ctx, "shipment.held", sh, map[string]string{"reason": reason}); err != nil {
		return nil, errs.Wrap(op, err)
	}
	return sh, nil
}

// Release puts a held shipment back in the created state.
func (s *Service) Release(ctx context.Context, id string) (*Shipment, error) {
	const op = "shipping.release"
	sh, err := s.store.Get(ctx, id)
	if err != nil {
		return nil, errs.Wrap(op, err)
	}
	if sh.Status != ShipmentStatusHeld {
		return nil, errs.Wrap(op, ErrBadTransition)
	}
	sh.Status = ShipmentStatusCreated
	sh.HoldReason = ""
	sh.UpdatedAt = s.clock.Now()
	if err := s.store.Put(ctx, sh); err != nil {
		return nil, errs.Wrap(op, err)
	}
	if err := s.record(ctx, "shipment.released", sh, nil); err != nil {
		return nil, errs.Wrap(op, err)
	}
	return sh, nil
}

// Reroute moves a not-yet-delivered shipment to another carrier.
func (s *Service) Reroute(ctx context.Context, id, carrier string) (*Shipment, error) {
	const op = "shipping.reroute"
	if !knownCarrier(carrier) {
		return nil, errs.Wrap(op, ErrNoCarrier)
	}
	sh, err := s.store.Get(ctx, id)
	if err != nil {
		return nil, errs.Wrap(op, err)
	}
	if sh.Status == ShipmentStatusDelivered {
		return nil, errs.Wrap(op, ErrBadTransition)
	}
	from := sh.Carrier
	sh.Carrier = carrier
	sh.UpdatedAt = s.clock.Now()
	if err := s.store.Put(ctx, sh); err != nil {
		return nil, errs.Wrap(op, err)
	}
	if err := s.record(ctx, "shipment.rerouted", sh, map[string]string{"from": from, "to": carrier}); err != nil {
		return nil, errs.Wrap(op, err)
	}
	return sh, nil
}

// Deliver is called from the carrier webhook.
func (s *Service) Deliver(ctx context.Context, id string) (*Shipment, error) {
	const op = "shipping.deliver"
	sh, err := s.store.Get(ctx, id)
	if err != nil {
		return nil, errs.Wrap(op, err)
	}
	if sh.Status != ShipmentStatusDispatched {
		return nil, errs.Wrap(op, ErrBadTransition)
	}
	sh.Status = ShipmentStatusDelivered
	sh.UpdatedAt = s.clock.Now()
	if err := s.store.Put(ctx, sh); err != nil {
		return nil, errs.Wrap(op, err)
	}
	if err := s.record(ctx, "shipment.delivered", sh, nil); err != nil {
		return nil, errs.Wrap(op, err)
	}
	return sh, nil
}
