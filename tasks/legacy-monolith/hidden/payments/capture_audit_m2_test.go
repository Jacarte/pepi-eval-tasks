package payments_test

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
	"shopcore/money"
	"shopcore/payments"
)

const (
	m2Op    = "payments.capture"
	m2Event = "payment.captured"
)

type m2Journal struct {
	mu    sync.Mutex
	steps []string
}

func (j *m2Journal) add(s string) {
	j.mu.Lock()
	j.steps = append(j.steps, s)
	j.mu.Unlock()
}

func (j *m2Journal) snapshot() []string {
	j.mu.Lock()
	defer j.mu.Unlock()
	return append([]string(nil), j.steps...)
}

type m2Repo struct {
	payments.Repository
	j       *m2Journal
	failErr error
}

func (r *m2Repo) Save(ctx context.Context, p *payments.Payment) error {
	if r.failErr != nil {
		r.j.add("save-failed")
		return r.failErr
	}
	if err := r.Repository.Save(ctx, p); err != nil {
		return err
	}
	r.j.add("save")
	return nil
}

type m2Recorder struct {
	mu      sync.Mutex
	j       *m2Journal
	events  []audit.Event
	failErr error
}

func (r *m2Recorder) Record(ctx context.Context, ev audit.Event) error {
	r.j.add("record")
	r.mu.Lock()
	r.events = append(r.events, ev)
	r.mu.Unlock()
	return r.failErr
}

func (r *m2Recorder) all() []audit.Event {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]audit.Event(nil), r.events...)
}

type m2Env struct {
	svc  *payments.Service
	base *payments.MemoryRepo
	repo *m2Repo
	rec  *m2Recorder
	j    *m2Journal
}

func m2Setup(t *testing.T) *m2Env {
	j := &m2Journal{}
	base := payments.NewMemoryRepo()
	repo := &m2Repo{Repository: base, j: j}
	rec := &m2Recorder{j: j}
	svc := payments.NewService(repo, rec, clock.NewFixed(time.Date(2024, 5, 6, 7, 8, 9, 0, time.UTC)))
	return &m2Env{svc: svc, base: base, repo: repo, rec: rec, j: j}
}

// m2Seed stores an authorization directly in the underlying repository.
func (e *m2Env) seed(t *testing.T, id string, amount money.Cents, st payments.Status) {
	t.Helper()
	p := &payments.Payment{ID: id, OrderID: "ord-" + id, Processor: "stripe", Amount: amount, Status: st, AuthorizedAt: time.Date(2024, 5, 6, 7, 0, 0, 0, time.UTC)}
	if err := e.base.Save(context.Background(), p); err != nil {
		t.Fatal(err)
	}
}

func m2Ctx(t *testing.T, n int) context.Context {
	h := fnv.New32a()
	h.Write([]byte(t.Name()))
	ctx := ctxkeys.WithRequestID(context.Background(), fmt.Sprintf("req-%08x-%d", h.Sum32(), n))
	ctx = ctxkeys.WithTraceID(ctx, fmt.Sprintf("trace-%08x-%d", h.Sum32()^0x5a5a5a, n))
	return ctxkeys.WithActor(ctx, "u-m2")
}

func m2CheckWrapped(t *testing.T, label string, err, cause error, exact bool) {
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
	if e.Op != m2Op {
		t.Errorf("%s: op = %q, want %q", label, e.Op, m2Op)
	}
	n := 0
	for _, op := range errs.Ops(err) {
		if op == m2Op {
			n++
		}
	}
	if n > 1 {
		t.Errorf("%s: %q wrapped %d times: %q", label, m2Op, n, err)
	}
	msg := err.Error()
	if exact {
		if want := m2Op + ": " + cause.Error(); msg != want {
			t.Errorf("%s: err.Error() = %q, want %q", label, msg, want)
		}
	} else if !strings.HasPrefix(msg, m2Op+": ") || !strings.Contains(msg, cause.Error()) {
		t.Errorf("%s: err.Error() = %q, want %q + ...%q", label, msg, m2Op+": ", cause.Error())
	}
}

// Catches: wrong or inconsistent event names, missing or duplicate events.
func TestM2CaptureEventName(t *testing.T) {
	env := m2Setup(t)
	env.seed(t, "pay-a", 5000, payments.StatusAuthorized)
	env.seed(t, "pay-b", 900, payments.StatusAuthorized)
	if _, err := env.svc.Capture(m2Ctx(t, 1), "pay-a", 4000); err != nil {
		t.Fatal(err)
	}
	if _, err := env.svc.Capture(m2Ctx(t, 2), "pay-b", 900); err != nil {
		t.Fatal(err)
	}
	evs := env.rec.all()
	if len(evs) != 2 {
		t.Fatalf("want exactly one audit event per successful Capture, got %d: %+v", len(evs), evs)
	}
	for i, ev := range evs {
		if ev.Name != m2Event {
			t.Errorf("event %d name = %q, want %q", i, ev.Name, m2Event)
		}
	}
}

// Catches: request id not taken from ctxkeys.RequestID, wrong subject.
func TestM2CaptureRequestIDAndSubject(t *testing.T) {
	env := m2Setup(t)
	env.seed(t, "pay-x1", 700, payments.StatusAuthorized)
	env.seed(t, "pay-x2", 800, payments.StatusAuthorized)
	ctxA, ctxB := m2Ctx(t, 1), m2Ctx(t, 2)
	if _, err := env.svc.Capture(ctxA, "pay-x1", 700); err != nil {
		t.Fatal(err)
	}
	if _, err := env.svc.Capture(ctxB, "pay-x2", 100); err != nil {
		t.Fatal(err)
	}
	evs := env.rec.all()
	if len(evs) != 2 {
		t.Fatalf("want 2 audit events, got %d: %+v", len(evs), evs)
	}
	for i, tc := range []struct {
		ctx context.Context
		id  string
	}{{ctxA, "pay-x1"}, {ctxB, "pay-x2"}} {
		if want := ctxkeys.RequestIDFrom(tc.ctx); evs[i].RequestID != want {
			t.Errorf("event %d RequestID = %q, want %q (from ctxkeys.RequestID)", i, evs[i].RequestID, want)
		}
		if evs[i].Subject != tc.id {
			t.Errorf("event %d Subject = %q, want the payment id %q", i, evs[i].Subject, tc.id)
		}
	}
}

// Catches: recording before the repository write, recording asynchronously,
// recording more than once.
func TestM2CaptureRecordsAfterWrite(t *testing.T) {
	env := m2Setup(t)
	env.seed(t, "pay-1", 5000, payments.StatusAuthorized)
	ctx := m2Ctx(t, 1)
	if _, err := env.svc.Capture(ctx, "pay-1", 2500); err != nil {
		t.Fatal(err)
	}
	steps := env.j.snapshot()
	if len(steps) != 2 || steps[0] != "save" || steps[1] != "record" {
		t.Fatalf("calls during Capture = %v, want [save record]: the event must be recorded after the payment is saved and before Capture returns", steps)
	}
	p, err := env.base.Get(ctx, "pay-1")
	if err != nil || p.Status != payments.StatusCaptured || p.Captured != 2500 {
		t.Fatalf("stored payment = %+v, %v", p, err)
	}
}

// Catches: audit events emitted on validation or repository failures.
func TestM2CaptureNoAuditOnFailure(t *testing.T) {
	env := m2Setup(t)
	env.seed(t, "pay-ok", 1000, payments.StatusAuthorized)
	env.seed(t, "pay-void", 1000, payments.StatusVoided)
	ctx := m2Ctx(t, 1)
	for i, tc := range []struct {
		id     string
		amount money.Cents
	}{{"pay-ok", 0}, {"pay-ok", 1001}, {"pay-void", 10}, {"pay-missing", 10}} {
		if _, err := env.svc.Capture(ctx, tc.id, tc.amount); err == nil {
			t.Fatalf("bad capture %d accepted", i)
		}
	}
	env.repo.failErr = errors.New("m2: repository unavailable")
	if _, err := env.svc.Capture(ctx, "pay-ok", 500); err == nil {
		t.Fatal("Capture succeeded although the repository failed")
	}
	if evs := env.rec.all(); len(evs) != 0 {
		t.Fatalf("failed Capture calls must not be audited, got %d events: %+v", len(evs), evs)
	}
	env.repo.failErr = nil
	if _, err := env.svc.Capture(ctx, "pay-ok", 500); err != nil {
		t.Fatal(err)
	}
	if evs := env.rec.all(); len(evs) != 1 {
		t.Fatalf("successful Capture after failures: want 1 audit event, got %d: %+v", len(evs), evs)
	}
}

// Catches: errors not wrapped with errs.Wrap("payments.capture", err).
func TestM2CaptureErrorWrapping(t *testing.T) {
	env := m2Setup(t)
	env.seed(t, "pay-1", 1000, payments.StatusAuthorized)
	env.seed(t, "pay-2", 1000, payments.StatusAuthorized)
	env.seed(t, "pay-v", 1000, payments.StatusVoided)
	ctx := m2Ctx(t, 1)

	_, err := env.svc.Capture(ctx, "pay-1", 0)
	m2CheckWrapped(t, "zero amount", err, payments.ErrInvalidAmount, true)
	_, err = env.svc.Capture(ctx, "pay-1", 1001)
	m2CheckWrapped(t, "over authorization", err, payments.ErrExceedsAuthorization, true)
	_, err = env.svc.Capture(ctx, "pay-v", 1)
	m2CheckWrapped(t, "voided payment", err, payments.ErrInvalidState, true)
	_, err = env.svc.Capture(ctx, "pay-missing", 1)
	m2CheckWrapped(t, "missing payment", err, payments.ErrNotFound, false)

	saveErr := errors.New("m2: write timeout")
	env.repo.failErr = saveErr
	_, err = env.svc.Capture(ctx, "pay-1", 10)
	m2CheckWrapped(t, "repository failure", err, saveErr, true)
	env.repo.failErr = nil

	recErr := errors.New("m2: audit sink unavailable")
	env.rec.failErr = recErr
	_, err = env.svc.Capture(ctx, "pay-2", 10)
	m2CheckWrapped(t, "audit failure", err, recErr, true)
	if p, _ := env.base.Get(ctx, "pay-2"); p == nil || p.Status != payments.StatusCaptured {
		t.Fatalf("audit failure: the capture must already be stored, got %+v", p)
	}
}
