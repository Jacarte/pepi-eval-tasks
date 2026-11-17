package orders_test

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"strings"
	"sync"
	"testing"
	"time"

	"shopcore/audit"
	"shopcore/clock"
	"shopcore/ctxkeys"
	"shopcore/errs"
	"shopcore/ids"
	"shopcore/orders"
)

const (
	m1Op    = "orders.place"
	m1Event = "order.placed"
)

type m1Journal struct {
	mu    sync.Mutex
	steps []string
}

func (j *m1Journal) add(s string) {
	j.mu.Lock()
	j.steps = append(j.steps, s)
	j.mu.Unlock()
}

func (j *m1Journal) snapshot() []string {
	j.mu.Lock()
	defer j.mu.Unlock()
	return append([]string(nil), j.steps...)
}

// m1Repo journals successful writes and can be told to fail them.
type m1Repo struct {
	orders.Repository
	j       *m1Journal
	failErr error
}

func (r *m1Repo) Save(ctx context.Context, o *orders.Order) error {
	if r.failErr != nil {
		r.j.add("save-failed")
		return r.failErr
	}
	if err := r.Repository.Save(ctx, o); err != nil {
		return err
	}
	r.j.add("save")
	return nil
}

type m1Recorder struct {
	mu      sync.Mutex
	j       *m1Journal
	events  []audit.Event
	failErr error
}

func (r *m1Recorder) Record(ctx context.Context, ev audit.Event) error {
	r.j.add("record")
	r.mu.Lock()
	r.events = append(r.events, ev)
	r.mu.Unlock()
	return r.failErr
}

func (r *m1Recorder) all() []audit.Event {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]audit.Event(nil), r.events...)
}

type m1Env struct {
	svc  *orders.Service
	base *orders.MemoryRepo
	repo *m1Repo
	rec  *m1Recorder
	j    *m1Journal
}

func m1Setup() *m1Env {
	j := &m1Journal{}
	base := orders.NewMemoryRepo()
	repo := &m1Repo{Repository: base, j: j}
	rec := &m1Recorder{j: j}
	svc := orders.NewService(orders.Config{
		Repo:  repo,
		Audit: rec,
		Clock: clock.NewFixed(time.Date(2024, 5, 6, 7, 8, 9, 0, time.UTC)),
		IDs:   ids.NewSequence(),
	})
	return &m1Env{svc: svc, base: base, repo: repo, rec: rec, j: j}
}

// m1Ctx builds a request context like httpapi does, with ids unique to the
// test and call so they cannot be hard-coded.
func m1Ctx(t *testing.T, n int) context.Context {
	h := fnv.New32a()
	h.Write([]byte(t.Name()))
	ctx := ctxkeys.WithRequestID(context.Background(), fmt.Sprintf("req-%08x-%d", h.Sum32(), n))
	ctx = ctxkeys.WithTraceID(ctx, fmt.Sprintf("trace-%08x-%d", h.Sum32()^0xabcdef, n))
	return ctxkeys.WithActor(ctx, "u-m1")
}

func m1Valid(customer string) orders.PlaceRequest {
	return orders.PlaceRequest{
		CustomerID: customer,
		Region:     "US-CA",
		Lines:      []orders.Line{{SKU: "SKU-1", Qty: 2, UnitPrice: 1999}, {SKU: "SKU-2", Qty: 1, UnitPrice: 500}},
	}
}

func m1CheckWrapped(t *testing.T, label string, err, cause error, exact bool) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s: expected an error", label)
	}
	if !errors.Is(err, cause) {
		t.Fatalf("%s: error %q does not wrap %q", label, err, cause)
	}
	var e *errs.Error
	if !errors.As(err, &e) {
		t.Fatalf("%s: error %q is not annotated with errs.Wrap", label, err)
	}
	if e.Op != m1Op {
		t.Errorf("%s: op = %q, want %q", label, e.Op, m1Op)
	}
	n := 0
	for _, op := range errs.Ops(err) {
		if op == m1Op {
			n++
		}
	}
	if n > 1 {
		t.Errorf("%s: %q wrapped %d times: %q", label, m1Op, n, err)
	}
	msg := err.Error()
	if exact {
		if want := m1Op + ": " + cause.Error(); msg != want {
			t.Errorf("%s: err.Error() = %q, want %q", label, msg, want)
		}
	} else if !strings.HasPrefix(msg, m1Op+": ") || !strings.Contains(msg, cause.Error()) {
		t.Errorf("%s: err.Error() = %q, want %q + ...%q", label, msg, m1Op+": ", cause.Error())
	}
}

// Catches: wrong or inconsistent event names, missing or duplicate events.
func TestM1PlaceEventName(t *testing.T) {
	env := m1Setup()
	for n := 1; n <= 2; n++ {
		if _, err := env.svc.Place(m1Ctx(t, n), m1Valid(fmt.Sprintf("cus-%d", n))); err != nil {
			t.Fatalf("Place #%d: %v", n, err)
		}
	}
	evs := env.rec.all()
	if len(evs) != 2 {
		t.Fatalf("want exactly one audit event per successful Place, got %d: %+v", len(evs), evs)
	}
	for i, ev := range evs {
		if ev.Name != m1Event {
			t.Errorf("event %d name = %q, want %q", i, ev.Name, m1Event)
		}
	}
}

// Catches: request id not taken from ctxkeys.RequestID (missing, trace id,
// actor, constant), wrong subject.
func TestM1PlaceRequestIDAndSubject(t *testing.T) {
	env := m1Setup()
	ctxA, ctxB := m1Ctx(t, 1), m1Ctx(t, 2)
	a, err := env.svc.Place(ctxA, m1Valid("cus-a"))
	if err != nil {
		t.Fatal(err)
	}
	b, err := env.svc.Place(ctxB, m1Valid("cus-b"))
	if err != nil {
		t.Fatal(err)
	}
	evs := env.rec.all()
	if len(evs) != 2 {
		t.Fatalf("want 2 audit events, got %d: %+v", len(evs), evs)
	}
	for i, tc := range []struct {
		ctx context.Context
		id  string
	}{{ctxA, a.ID}, {ctxB, b.ID}} {
		if want := ctxkeys.RequestIDFrom(tc.ctx); evs[i].RequestID != want {
			t.Errorf("event %d RequestID = %q, want %q (from ctxkeys.RequestID)", i, evs[i].RequestID, want)
		}
		if evs[i].Subject != tc.id {
			t.Errorf("event %d Subject = %q, want the order id %q", i, evs[i].Subject, tc.id)
		}
	}
}

// Catches: recording before the repository write, recording asynchronously
// (after Place returns), or recording more than once.
func TestM1PlaceRecordsAfterWrite(t *testing.T) {
	env := m1Setup()
	ctx := m1Ctx(t, 1)
	o, err := env.svc.Place(ctx, m1Valid("cus-1"))
	if err != nil {
		t.Fatal(err)
	}
	steps := env.j.snapshot()
	if len(steps) != 2 || steps[0] != "save" || steps[1] != "record" {
		t.Fatalf("calls during Place = %v, want [save record]: the event must be recorded after the order is saved and before Place returns", steps)
	}
	if _, err := env.base.Get(ctx, o.ID); err != nil {
		t.Fatalf("order not stored: %v", err)
	}
}

// Catches: audit events emitted on validation or repository failures.
func TestM1PlaceNoAuditOnFailure(t *testing.T) {
	env := m1Setup()
	ctx := m1Ctx(t, 1)
	bad := []orders.PlaceRequest{
		{Region: "US-CA", Lines: m1Valid("").Lines},
		{CustomerID: "cus-1", Region: "US-CA"},
		{CustomerID: "cus-1", Region: "US-CA", Lines: []orders.Line{{SKU: "SKU-1", Qty: 0, UnitPrice: 10}}},
		{CustomerID: "cus-1", Region: "XX-NOWHERE", Lines: m1Valid("").Lines},
	}
	for i, req := range bad {
		if _, err := env.svc.Place(ctx, req); err == nil {
			t.Fatalf("bad request %d accepted", i)
		}
	}
	env.repo.failErr = errors.New("m1: repository unavailable")
	if _, err := env.svc.Place(ctx, m1Valid("cus-1")); err == nil {
		t.Fatal("Place succeeded although the repository failed")
	}
	if evs := env.rec.all(); len(evs) != 0 {
		t.Fatalf("failed Place calls must not be audited, got %d events: %+v", len(evs), evs)
	}
	env.repo.failErr = nil
	if _, err := env.svc.Place(ctx, m1Valid("cus-1")); err != nil {
		t.Fatal(err)
	}
	if evs := env.rec.all(); len(evs) != 1 {
		t.Fatalf("successful Place after failures: want 1 audit event, got %d: %+v", len(evs), evs)
	}
}

// Catches: errors not wrapped with errs.Wrap("orders.place", err), wrong op
// spelling, extra prefixes, double wrapping.
func TestM1PlaceErrorWrapping(t *testing.T) {
	env := m1Setup()
	ctx := m1Ctx(t, 1)

	_, err := env.svc.Place(ctx, orders.PlaceRequest{CustomerID: "cus-1", Region: "US-CA"})
	m1CheckWrapped(t, "empty order", err, orders.ErrEmptyOrder, true)
	_, err = env.svc.Place(ctx, orders.PlaceRequest{Region: "US-CA", Lines: m1Valid("").Lines})
	m1CheckWrapped(t, "missing customer", err, orders.ErrMissingCustomer, true)
	_, err = env.svc.Place(ctx, orders.PlaceRequest{CustomerID: "cus-1", Region: "US-CA", Lines: []orders.Line{{SKU: "S", Qty: -1}}})
	m1CheckWrapped(t, "invalid quantity", err, orders.ErrInvalidQuantity, false)

	saveErr := errors.New("m1: write timeout")
	env.repo.failErr = saveErr
	_, err = env.svc.Place(ctx, m1Valid("cus-1"))
	m1CheckWrapped(t, "repository failure", err, saveErr, true)
	env.repo.failErr = nil

	recErr := errors.New("m1: audit sink unavailable")
	env.rec.failErr = recErr
	_, err = env.svc.Place(ctx, m1Valid("cus-wrap"))
	m1CheckWrapped(t, "audit failure", err, recErr, true)
	if list, _ := env.base.ListByCustomer(ctx, "cus-wrap"); len(list) != 1 {
		t.Fatalf("audit failure: the order must already be stored, found %d", len(list))
	}
}
