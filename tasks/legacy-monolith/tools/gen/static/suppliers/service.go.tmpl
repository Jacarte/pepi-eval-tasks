// Package suppliers manages vendors and the purchase orders we place with
// them.
package suppliers

import (
	"context"
	"errors"

	"shopcore/audit"
	"shopcore/clock"
	"shopcore/ctxkeys"
	"shopcore/errs"
	"shopcore/money"
)

var (
	ErrNotProspect  = errors.New("suppliers: supplier is not a prospect")
	ErrNotOnboarded = errors.New("suppliers: supplier is not onboarded")
	ErrNotDraft     = errors.New("suppliers: purchase order is not a draft")
	ErrNotApproved  = errors.New("suppliers: purchase order is not approved")
	ErrOverBudget   = errors.New("suppliers: purchase order exceeds approval limit")
)

// Service is the supplier back office.
type Service struct {
	suppliers SupplierStore
	pos       PurchaseOrderStore
	audit     audit.Recorder
	clock     clock.Clock
	// ApprovalLimit is the largest PO a buyer can approve without finance.
	ApprovalLimit money.Cents
}

func NewService(suppliers SupplierStore, pos PurchaseOrderStore, rec audit.Recorder, clk clock.Clock) *Service {
	return &Service{suppliers: suppliers, pos: pos, audit: audit.OrDiscard(rec), clock: clock.OrSystem(clk), ApprovalLimit: 5_000_000}
}

func requestID(ctx context.Context) string {
	id, _ := ctx.Value(ctxkeys.RequestID).(string)
	return id
}

// Onboard activates a prospect supplier after the compliance checklist.
func (s *Service) Onboard(ctx context.Context, id string) (*Supplier, error) {
	sup, err := s.suppliers.Find(ctx, id)
	if err != nil {
		return nil, errs.Wrap("suppliers.onboard", err)
	}
	if sup.Status != SupplierStatusProspect {
		return nil, errs.Wrap("suppliers.onboard", ErrNotProspect)
	}
	sup.Status = SupplierStatusOnboarded
	sup.UpdatedAt = s.clock.Now()
	if err := s.suppliers.Save(ctx, sup); err != nil {
		return nil, errs.Wrap("suppliers.onboard", err)
	}
	err = s.audit.Record(ctx, audit.Event{
		Name:      "supplier.onboarded",
		RequestID: requestID(ctx),
		Subject:   sup.ID,
		Attrs:     map[string]string{"country": sup.Country},
	})
	if err != nil {
		return nil, errs.Wrap("suppliers.onboard", err)
	}
	return sup, nil
}

// Suspend blocks new purchase orders for a supplier.
func (s *Service) Suspend(ctx context.Context, id, reason string) (*Supplier, error) {
	sup, err := s.suppliers.Find(ctx, id)
	if err != nil {
		return nil, errs.Wrap("suppliers.suspend", err)
	}
	if sup.Status != SupplierStatusOnboarded {
		return nil, errs.Wrap("suppliers.suspend", ErrNotOnboarded)
	}
	sup.Status = SupplierStatusSuspended
	sup.UpdatedAt = s.clock.Now()
	if err := s.suppliers.Save(ctx, sup); err != nil {
		return nil, errs.Wrap("suppliers.suspend", err)
	}
	err = s.audit.Record(ctx, audit.Event{
		Name:      "supplier.suspended",
		RequestID: requestID(ctx),
		Subject:   sup.ID,
		Attrs:     map[string]string{"reason": reason},
	})
	if err != nil {
		return nil, errs.Wrap("suppliers.suspend", err)
	}
	return sup, nil
}

// Approve approves a draft purchase order.
func (s *Service) Approve(ctx context.Context, poID string) (*PurchaseOrder, error) {
	po, err := s.pos.Find(ctx, poID)
	if err != nil {
		return nil, errs.Wrap("suppliers.approve", err)
	}
	if po.Status != PurchaseOrderStatusDraft {
		return nil, errs.Wrap("suppliers.approve", ErrNotDraft)
	}
	if po.Total > s.ApprovalLimit {
		return nil, errs.Wrap("suppliers.approve", ErrOverBudget)
	}
	sup, err := s.suppliers.Find(ctx, po.SupplierID)
	if err != nil {
		return nil, errs.Wrap("suppliers.approve", err)
	}
	if sup.Status != SupplierStatusOnboarded {
		return nil, errs.Wrap("suppliers.approve", ErrNotOnboarded)
	}
	po.Status = PurchaseOrderStatusApproved
	po.UpdatedAt = s.clock.Now()
	if err := s.pos.Save(ctx, po); err != nil {
		return nil, errs.Wrap("suppliers.approve", err)
	}
	err = s.audit.Record(ctx, audit.Event{
		Name:      "purchase_order.approved",
		RequestID: requestID(ctx),
		Subject:   po.ID,
		Attrs:     map[string]string{"supplier_id": po.SupplierID, "total": po.Total.String()},
	})
	if err != nil {
		return nil, errs.Wrap("suppliers.approve", err)
	}
	return po, nil
}

// Receive marks an approved purchase order as delivered to the warehouse.
func (s *Service) Receive(ctx context.Context, poID string) (*PurchaseOrder, error) {
	po, err := s.pos.Find(ctx, poID)
	if err != nil {
		return nil, errs.Wrap("suppliers.receive", err)
	}
	if po.Status != PurchaseOrderStatusApproved {
		return nil, errs.Wrap("suppliers.receive", ErrNotApproved)
	}
	po.Status = PurchaseOrderStatusReceived
	po.UpdatedAt = s.clock.Now()
	if err := s.pos.Save(ctx, po); err != nil {
		return nil, errs.Wrap("suppliers.receive", err)
	}
	err = s.audit.Record(ctx, audit.Event{
		Name:      "purchase_order.received",
		RequestID: requestID(ctx),
		Subject:   po.ID,
	})
	if err != nil {
		return nil, errs.Wrap("suppliers.receive", err)
	}
	return po, nil
}

// Scorecard summarises a supplier's purchase orders.
func (s *Service) Scorecard(ctx context.Context, supplierID string) (PurchaseOrderSummary, error) {
	list, err := s.pos.All(ctx, PurchaseOrderFilter{SupplierID: supplierID})
	if err != nil {
		return PurchaseOrderSummary{}, errs.Wrap("suppliers.scorecard", err)
	}
	return SummarizePurchaseOrders(list), nil
}
