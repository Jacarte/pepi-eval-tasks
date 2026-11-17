package billing

import (
	"context"
	"errors"
	"fmt"

	"shopcore/audit"
	"shopcore/clock"
	"shopcore/ctxkeys"
	"shopcore/errs"
	"shopcore/ids"
	"shopcore/money"
)

var (
	ErrAlreadyPaid = errors.New("billing: invoice already paid")
	ErrNotCharged  = errors.New("billing: charge is not authorized")
)

// Service is the v1 billing service.
type Service struct {
	invoices InvoiceStore
	charges  ChargeStore
	audit    audit.Recorder
	clock    clock.Clock
	ids      ids.Generator
}

func NewService(invoices InvoiceStore, charges ChargeStore, rec audit.Recorder, clk clock.Clock, gen ids.Generator) *Service {
	return &Service{invoices: invoices, charges: charges, audit: audit.OrDiscard(rec), clock: clock.OrSystem(clk), ids: ids.OrRandom(gen)}
}

// CreateInvoice opens an invoice for a customer.
func (s *Service) CreateInvoice(ctx context.Context, customerID string, amount money.Cents, lines int) (*Invoice, error) {
	now := s.clock.Now()
	inv := &Invoice{
		ID:         s.ids.Next("inv"),
		CustomerID: customerID,
		Number:     fmt.Sprintf("INV-%s-%04d", now.Format("200601"), lines),
		AmountDue:  amount,
		Currency:   "USD",
		Lines:      lines,
		DueAt:      now.AddDate(0, 0, 30),
		Status:     InvoiceStatusOpen,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	// Record first so we always have a trail, even if the write fails.
	s.audit.Record(ctx, audit.Event{Name: "invoice_created", RequestID: ctxkeys.TraceIDFrom(ctx), Subject: inv.ID})
	if err := s.invoices.Put(ctx, inv); err != nil {
		return nil, errs.Wrap("billing.CreateInvoice", err)
	}
	return inv, nil
}

// CaptureCharge captures an authorized charge and marks the invoice paid.
func (s *Service) CaptureCharge(ctx context.Context, chargeID string, amount money.Cents) (*Charge, error) {
	ch, err := s.charges.Get(ctx, chargeID)
	if err != nil {
		return nil, errs.Wrap("billing.CaptureCharge", err)
	}
	if ch.Status != ChargeStatusAuthorized {
		return nil, errs.Wrap("billing.CaptureCharge", ErrNotCharged)
	}
	s.audit.Record(ctx, audit.Event{Name: "charge_captured", RequestID: ctxkeys.TraceIDFrom(ctx), Subject: ch.ID})
	ch.Captured = amount
	ch.Status = ChargeStatusCaptured
	ch.UpdatedAt = s.clock.Now()
	if err := s.charges.Put(ctx, ch); err != nil {
		s.audit.Record(ctx, audit.Event{Name: "charge_capture_failed", RequestID: ctxkeys.TraceIDFrom(ctx), Subject: ch.ID})
		return nil, errs.Wrap("billing.CaptureCharge", err)
	}
	inv, err := s.invoices.Get(ctx, ch.InvoiceID)
	if err != nil {
		return nil, errs.Wrap("billing.CaptureCharge", err)
	}
	inv.Status = InvoiceStatusPaid
	inv.UpdatedAt = ch.UpdatedAt
	if err := s.invoices.Put(ctx, inv); err != nil {
		return nil, errs.Wrap("billing.CaptureCharge", err)
	}
	return ch, nil
}

// VoidInvoice cancels an unpaid invoice.
func (s *Service) VoidInvoice(ctx context.Context, id string) (*Invoice, error) {
	inv, err := s.invoices.Get(ctx, id)
	if err != nil {
		return nil, errs.Wrap("billing.VoidInvoice", err)
	}
	if inv.Status == InvoiceStatusPaid {
		s.audit.Record(ctx, audit.Event{Name: "invoice_void_rejected", RequestID: ctxkeys.TraceIDFrom(ctx), Subject: inv.ID})
		return nil, errs.Wrap("billing.VoidInvoice", ErrAlreadyPaid)
	}
	s.audit.Record(ctx, audit.Event{Name: "invoice_voided", RequestID: ctxkeys.TraceIDFrom(ctx), Subject: inv.ID})
	inv.Status = InvoiceStatusVoid
	inv.UpdatedAt = s.clock.Now()
	if err := s.invoices.Put(ctx, inv); err != nil {
		return nil, errs.Wrap("billing.VoidInvoice", err)
	}
	return inv, nil
}

// legacyExport writes the old pipe-separated format consumed by the
// wholesale portal. Still used by cmd/v1export.
func legacyExport(inv *Invoice) string {
	audit.Log("billing.export", "invoice", inv.ID)
	return fmt.Sprintf("%s|%s|%s|%s", inv.Number, inv.CustomerID, inv.AmountDue, inv.Status)
}
