// Package returns handles return merchandise authorizations (RMAs).
package returns

import (
	"context"
	"errors"

	"shopcore/audit"
	"shopcore/clock"
	"shopcore/ctxkeys"
	"shopcore/errs"
	"shopcore/ids"
)

var (
	ErrNotRequested = errors.New("returns: return is not awaiting authorization")
	ErrBadQuantity  = errors.New("returns: quantity must be positive")
)

type Service struct {
	store ReturnRequestStore
	audit audit.Recorder
	clock clock.Clock
	ids   ids.Generator
}

func NewService(store ReturnRequestStore, rec audit.Recorder, clk clock.Clock, gen ids.Generator) *Service {
	return &Service{store: store, audit: audit.OrDiscard(rec), clock: clock.OrSystem(clk), ids: ids.OrRandom(gen)}
}

// Authorize approves a return so the customer can ship it back.
func (s *Service) Authorize(ctx context.Context, id, inspector string) (*ReturnRequest, error) {
	const op = "returns.authorize"
	rr, err := s.store.Find(ctx, id)
	if err != nil {
		return nil, errs.Wrap(op, err)
	}
	if rr.Status != ReturnRequestStatusRequested {
		return nil, errs.Wrap(op, ErrNotRequested)
	}
	rr.Status = ReturnRequestStatusAuthorized
	rr.Inspector = inspector
	rr.UpdatedAt = s.clock.Now()
	if err := s.store.Save(ctx, rr); err != nil {
		return nil, errs.Wrap(op, err)
	}
	if err := s.audit.Record(ctx, audit.Event{
		Name:      "return_request.authorized",
		RequestID: ctxkeys.RequestIDFrom(ctx),
		Subject:   rr.ID,
		Attrs:     map[string]string{"order_id": rr.OrderID, "inspector": inspector},
	}); err != nil {
		return nil, errs.Wrap(op, err)
	}
	return rr, nil
}

// Reject declines a return request.
func (s *Service) Reject(ctx context.Context, id, why string) (*ReturnRequest, error) {
	const op = "returns.reject"
	rr, err := s.store.Find(ctx, id)
	if err != nil {
		return nil, errs.Wrap(op, err)
	}
	if rr.Status != ReturnRequestStatusRequested {
		return nil, errs.Wrap(op, ErrNotRequested)
	}
	rr.Status = ReturnRequestStatusRejected
	rr.UpdatedAt = s.clock.Now()
	if err := s.store.Save(ctx, rr); err != nil {
		return nil, errs.Wrap(op, err)
	}
	if err := s.audit.Record(ctx, audit.Event{
		Name:      "return_request.rejected",
		RequestID: ctxkeys.RequestIDFrom(ctx),
		Subject:   rr.ID,
		Attrs:     map[string]string{"why": why},
	}); err != nil {
		return nil, errs.Wrap(op, err)
	}
	return rr, nil
}
