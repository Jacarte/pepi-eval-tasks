package inventory_test

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
	"shopcore/inventory"
)

const (
	m4Op    = "inventory.reserve"
	m4Event = "stock_item.reserved"
)

type m4Journal struct {
	mu    sync.Mutex
	steps []string
}

func (j *m4Journal) add(s string) {
	j.mu.Lock()
	j.steps = append(j.steps, s)
	j.mu.Unlock()
}

func (j *m4Journal) snapshot() []string {
	j.mu.Lock()
	defer j.mu.Unlock()
	return append([]string(nil), j.steps...)
}

type m4Repo struct {
	inventory.Repository
	j       *m4Journal
	failErr error
}

func (r *m4Repo) Put(ctx context.Context, it *inventory.StockItem) error {
	if r.failErr != nil {
		r.j.add("put-failed")
		return r.failErr
	}
	if err := r.Repository.Put(ctx, it); err != nil {
		return err
	}
	r.j.add("put")
	return nil
}

type m4Recorder struct {
	mu      sync.Mutex
	j       *m4Journal
	events  []audit.Event
	failErr error
}

func (r *m4Recorder) Record(ctx context.Context, ev audit.Event) error {
	r.j.add("record")
	r.mu.Lock()
	r.events = append(r.events, ev)
	r.mu.Unlock()
	return r.failErr
}

func (r *m4Recorder) all() []audit.Event {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]audit.Event(nil), r.events...)
}

type m4Env struct {
	svc  *inventory.Service
	base *inventory.MemoryRepo
	repo *m4Repo
	rec  *m4Recorder
	j    *m4Journal
}

func m4Setup() *m4Env {
	j := &m4Journal{}
	base := inventory.NewMemoryRepo()
	repo := &m4Repo{Repository: base, j: j}
	rec := &m4Recorder{j: j}
	svc := inventory.NewService(repo,
		inventory.WithAudit(rec),
		inventory.WithClock(clock.NewFixed(time.Date(2024, 5, 6, 7, 8, 9, 0, time.UTC))))
	return &m4Env{svc: svc, base: base, repo: repo, rec: rec, j: j}
}

func (e *m4Env) seed(t *testing.T, sku string, onHand, reserved int) {
	t.Helper()
	if err := e.base.Put(context.Background(), &inventory.StockItem{ID: sku, Warehouse: "ams", OnHand: onHand, Reserved: reserved}); err != nil {
		t.Fatal(err)
	}
}

func m4Ctx(t *testing.T, n int) context.Context {
	h := fnv.New32a()
	h.Write([]byte(t.Name()))
	ctx := ctxkeys.WithRequestID(context.Background(), fmt.Sprintf("req-%08x-%d", h.Sum32(), n))
	ctx = ctxkeys.WithTraceID(ctx, fmt.Sprintf("trace-%08x-%d", h.Sum32()^0x0f0f0f, n))
	return ctxkeys.WithActor(ctx, "u-m4")
}

func m4CheckWrapped(t *testing.T, label string, err, cause error, exact bool) {
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
	if e.Op != m4Op {
		t.Errorf("%s: op = %q, want %q", label, e.Op, m4Op)
	}
	n := 0
	for _, op := range errs.Ops(err) {
		if op == m4Op {
			n++
		}
	}
	if n > 1 {
		t.Errorf("%s: %q wrapped %d times: %q", label, m4Op, n, err)
	}
	msg := err.Error()
	if exact {
		if want := m4Op + ": " + cause.Error(); msg != want {
			t.Errorf("%s: err.Error() = %q, want %q", label, msg, want)
		}
	} else if !strings.HasPrefix(msg, m4Op+": ") || !strings.Contains(msg, cause.Error()) {
		t.Errorf("%s: err.Error() = %q, want %q + ...%q", label, msg, m4Op+": ", cause.Error())
	}
}

// Catches: wrong or inconsistent event names, missing or duplicate events.
func TestM4ReserveEventName(t *testing.T) {
	env := m4Setup()
	env.seed(t, "SKU-A", 10, 0)
	env.seed(t, "SKU-B", 3, 1)
	if _, err := env.svc.Reserve(m4Ctx(t, 1), "SKU-A", 4); err != nil {
		t.Fatal(err)
	}
	if _, err := env.svc.Reserve(m4Ctx(t, 2), "SKU-B", 2); err != nil {
		t.Fatal(err)
	}
	evs := env.rec.all()
	if len(evs) != 2 {
		t.Fatalf("want exactly one audit event per successful Reserve, got %d: %+v", len(evs), evs)
	}
	for i, ev := range evs {
		if ev.Name != m4Event {
			t.Errorf("event %d name = %q, want %q", i, ev.Name, m4Event)
		}
	}
}

// Catches: request id not taken from ctxkeys.RequestID, wrong subject.
func TestM4ReserveRequestIDAndSubject(t *testing.T) {
	env := m4Setup()
	env.seed(t, "SKU-X1", 5, 0)
	env.seed(t, "SKU-X2", 5, 0)
	ctxA, ctxB := m4Ctx(t, 1), m4Ctx(t, 2)
	if _, err := env.svc.Reserve(ctxA, "SKU-X1", 1); err != nil {
		t.Fatal(err)
	}
	if _, err := env.svc.Reserve(ctxB, "SKU-X2", 5); err != nil {
		t.Fatal(err)
	}
	evs := env.rec.all()
	if len(evs) != 2 {
		t.Fatalf("want 2 audit events, got %d: %+v", len(evs), evs)
	}
	for i, tc := range []struct {
		ctx context.Context
		id  string
	}{{ctxA, "SKU-X1"}, {ctxB, "SKU-X2"}} {
		if want := ctxkeys.RequestIDFrom(tc.ctx); evs[i].RequestID != want {
			t.Errorf("event %d RequestID = %q, want %q (from ctxkeys.RequestID)", i, evs[i].RequestID, want)
		}
		if evs[i].Subject != tc.id {
			t.Errorf("event %d Subject = %q, want the stock item id %q", i, evs[i].Subject, tc.id)
		}
	}
}

// Catches: recording before the repository write, recording asynchronously,
// recording more than once.
func TestM4ReserveRecordsAfterWrite(t *testing.T) {
	env := m4Setup()
	env.seed(t, "SKU-1", 8, 2)
	ctx := m4Ctx(t, 1)
	if _, err := env.svc.Reserve(ctx, "SKU-1", 3); err != nil {
		t.Fatal(err)
	}
	steps := env.j.snapshot()
	if len(steps) != 2 || steps[0] != "put" || steps[1] != "record" {
		t.Fatalf("calls during Reserve = %v, want [put record]: the event must be recorded after the stock item is saved and before Reserve returns", steps)
	}
	it, err := env.base.Get(ctx, "SKU-1")
	if err != nil || it.Reserved != 5 {
		t.Fatalf("stored item = %+v, %v", it, err)
	}
}

// Catches: audit events emitted on validation or repository failures.
func TestM4ReserveNoAuditOnFailure(t *testing.T) {
	env := m4Setup()
	env.seed(t, "SKU-1", 4, 1)
	ctx := m4Ctx(t, 1)
	for i, tc := range []struct {
		sku string
		qty int
	}{{"SKU-1", 0}, {"SKU-1", -2}, {"SKU-1", 4}, {"SKU-404", 1}} {
		if _, err := env.svc.Reserve(ctx, tc.sku, tc.qty); err == nil {
			t.Fatalf("bad reservation %d accepted", i)
		}
	}
	env.repo.failErr = errors.New("m4: repository unavailable")
	if _, err := env.svc.Reserve(ctx, "SKU-1", 1); err == nil {
		t.Fatal("Reserve succeeded although the repository failed")
	}
	if evs := env.rec.all(); len(evs) != 0 {
		t.Fatalf("failed Reserve calls must not be audited, got %d events: %+v", len(evs), evs)
	}
	env.repo.failErr = nil
	if _, err := env.svc.Reserve(ctx, "SKU-1", 1); err != nil {
		t.Fatal(err)
	}
	if evs := env.rec.all(); len(evs) != 1 {
		t.Fatalf("successful Reserve after failures: want 1 audit event, got %d: %+v", len(evs), evs)
	}
}

// Catches: errors not wrapped with errs.Wrap("inventory.reserve", err).
func TestM4ReserveErrorWrapping(t *testing.T) {
	env := m4Setup()
	env.seed(t, "SKU-1", 4, 0)
	env.seed(t, "SKU-2", 4, 0)
	ctx := m4Ctx(t, 1)

	_, err := env.svc.Reserve(ctx, "SKU-1", 0)
	m4CheckWrapped(t, "zero quantity", err, inventory.ErrInvalidQuantity, true)
	_, err = env.svc.Reserve(ctx, "SKU-1", 5)
	m4CheckWrapped(t, "insufficient stock", err, inventory.ErrInsufficientStock, true)
	_, err = env.svc.Reserve(ctx, "SKU-404", 1)
	m4CheckWrapped(t, "unknown sku", err, inventory.ErrNotFound, false)

	saveErr := errors.New("m4: write timeout")
	env.repo.failErr = saveErr
	_, err = env.svc.Reserve(ctx, "SKU-1", 1)
	m4CheckWrapped(t, "repository failure", err, saveErr, true)
	env.repo.failErr = nil

	recErr := errors.New("m4: audit sink unavailable")
	env.rec.failErr = recErr
	_, err = env.svc.Reserve(ctx, "SKU-2", 2)
	m4CheckWrapped(t, "audit failure", err, recErr, true)
	if it, _ := env.base.Get(ctx, "SKU-2"); it == nil || it.Reserved != 2 {
		t.Fatalf("audit failure: the reservation must already be stored, got %+v", it)
	}
}
