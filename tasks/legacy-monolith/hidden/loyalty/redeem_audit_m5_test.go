package loyalty_test

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
	"shopcore/loyalty"
)

const (
	m5Op    = "loyalty.redeem"
	m5Event = "account.redeemed"
)

type m5Journal struct {
	mu    sync.Mutex
	steps []string
}

func (j *m5Journal) add(s string) {
	j.mu.Lock()
	j.steps = append(j.steps, s)
	j.mu.Unlock()
}

func (j *m5Journal) snapshot() []string {
	j.mu.Lock()
	defer j.mu.Unlock()
	return append([]string(nil), j.steps...)
}

type m5Repo struct {
	loyalty.Repository
	j       *m5Journal
	failErr error
}

func (r *m5Repo) Store(ctx context.Context, a *loyalty.Account) error {
	if r.failErr != nil {
		r.j.add("store-failed")
		return r.failErr
	}
	if err := r.Repository.Store(ctx, a); err != nil {
		return err
	}
	r.j.add("store")
	return nil
}

type m5Recorder struct {
	mu      sync.Mutex
	j       *m5Journal
	events  []audit.Event
	failErr error
}

func (r *m5Recorder) Record(ctx context.Context, ev audit.Event) error {
	r.j.add("record")
	r.mu.Lock()
	r.events = append(r.events, ev)
	r.mu.Unlock()
	return r.failErr
}

func (r *m5Recorder) all() []audit.Event {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]audit.Event(nil), r.events...)
}

type m5Env struct {
	svc  *loyalty.Service
	base *loyalty.MemoryRepo
	repo *m5Repo
	rec  *m5Recorder
	j    *m5Journal
}

func m5Setup() *m5Env {
	j := &m5Journal{}
	base := loyalty.NewMemoryRepo()
	repo := &m5Repo{Repository: base, j: j}
	rec := &m5Recorder{j: j}
	svc := loyalty.NewService(repo, rec, clock.NewFixed(time.Date(2024, 5, 6, 7, 8, 9, 0, time.UTC)))
	return &m5Env{svc: svc, base: base, repo: repo, rec: rec, j: j}
}

func (e *m5Env) seed(t *testing.T, id string, balance int64) {
	t.Helper()
	a := &loyalty.Account{ID: id, CustomerID: "cus-" + id, Balance: balance, Lifetime: balance, TierName: "bronze"}
	if err := e.base.Store(context.Background(), a); err != nil {
		t.Fatal(err)
	}
}

func m5Ctx(t *testing.T, n int) context.Context {
	h := fnv.New32a()
	h.Write([]byte(t.Name()))
	ctx := ctxkeys.WithRequestID(context.Background(), fmt.Sprintf("req-%08x-%d", h.Sum32(), n))
	ctx = ctxkeys.WithTraceID(ctx, fmt.Sprintf("trace-%08x-%d", h.Sum32()^0x777777, n))
	return ctxkeys.WithActor(ctx, "u-m5")
}

func m5CheckWrapped(t *testing.T, label string, err, cause error, exact bool) {
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
	if e.Op != m5Op {
		t.Errorf("%s: op = %q, want %q", label, e.Op, m5Op)
	}
	n := 0
	for _, op := range errs.Ops(err) {
		if op == m5Op || op == "Redeem" {
			n++
		}
	}
	if n > 1 {
		t.Errorf("%s: redeem op wrapped %d times: %q", label, n, err)
	}
	msg := err.Error()
	if exact {
		if want := m5Op + ": " + cause.Error(); msg != want {
			t.Errorf("%s: err.Error() = %q, want %q", label, msg, want)
		}
	} else if !strings.HasPrefix(msg, m5Op+": ") || !strings.Contains(msg, cause.Error()) {
		t.Errorf("%s: err.Error() = %q, want %q + ...%q", label, msg, m5Op+": ", cause.Error())
	}
}

// Catches: wrong or inconsistent event names, missing or duplicate events.
func TestM5RedeemEventName(t *testing.T) {
	env := m5Setup()
	env.seed(t, "loy-a", 1000)
	env.seed(t, "loy-b", 50)
	if _, err := env.svc.Redeem(m5Ctx(t, 1), "loy-a", 400); err != nil {
		t.Fatal(err)
	}
	if _, err := env.svc.Redeem(m5Ctx(t, 2), "loy-b", 50); err != nil {
		t.Fatal(err)
	}
	evs := env.rec.all()
	if len(evs) != 2 {
		t.Fatalf("want exactly one audit event per successful Redeem, got %d: %+v", len(evs), evs)
	}
	for i, ev := range evs {
		if ev.Name != m5Event {
			t.Errorf("event %d name = %q, want %q", i, ev.Name, m5Event)
		}
	}
}

// Catches: request id not taken from ctxkeys.RequestID, wrong subject.
func TestM5RedeemRequestIDAndSubject(t *testing.T) {
	env := m5Setup()
	env.seed(t, "loy-x1", 300)
	env.seed(t, "loy-x2", 300)
	ctxA, ctxB := m5Ctx(t, 1), m5Ctx(t, 2)
	if _, err := env.svc.Redeem(ctxA, "loy-x1", 100); err != nil {
		t.Fatal(err)
	}
	if _, err := env.svc.Redeem(ctxB, "loy-x2", 300); err != nil {
		t.Fatal(err)
	}
	evs := env.rec.all()
	if len(evs) != 2 {
		t.Fatalf("want 2 audit events, got %d: %+v", len(evs), evs)
	}
	for i, tc := range []struct {
		ctx context.Context
		id  string
	}{{ctxA, "loy-x1"}, {ctxB, "loy-x2"}} {
		if want := ctxkeys.RequestIDFrom(tc.ctx); evs[i].RequestID != want {
			t.Errorf("event %d RequestID = %q, want %q (from ctxkeys.RequestID)", i, evs[i].RequestID, want)
		}
		if evs[i].Subject != tc.id {
			t.Errorf("event %d Subject = %q, want the account id %q", i, evs[i].Subject, tc.id)
		}
	}
}

// Catches: recording before the repository write, recording asynchronously,
// recording more than once.
func TestM5RedeemRecordsAfterWrite(t *testing.T) {
	env := m5Setup()
	env.seed(t, "loy-1", 900)
	ctx := m5Ctx(t, 1)
	if _, err := env.svc.Redeem(ctx, "loy-1", 250); err != nil {
		t.Fatal(err)
	}
	steps := env.j.snapshot()
	if len(steps) != 2 || steps[0] != "store" || steps[1] != "record" {
		t.Fatalf("calls during Redeem = %v, want [store record]: the event must be recorded after the account is saved and before Redeem returns", steps)
	}
	a, err := env.base.Load(ctx, "loy-1")
	if err != nil || a.Balance != 650 {
		t.Fatalf("stored account = %+v, %v", a, err)
	}
}

// Catches: audit events emitted on validation or repository failures.
func TestM5RedeemNoAuditOnFailure(t *testing.T) {
	env := m5Setup()
	env.seed(t, "loy-1", 100)
	ctx := m5Ctx(t, 1)
	for i, tc := range []struct {
		id     string
		points int64
	}{{"loy-1", 0}, {"loy-1", -5}, {"loy-1", 101}, {"loy-404", 1}} {
		if _, err := env.svc.Redeem(ctx, tc.id, tc.points); err == nil {
			t.Fatalf("bad redemption %d accepted", i)
		}
	}
	env.repo.failErr = errors.New("m5: repository unavailable")
	if _, err := env.svc.Redeem(ctx, "loy-1", 10); err == nil {
		t.Fatal("Redeem succeeded although the repository failed")
	}
	if evs := env.rec.all(); len(evs) != 0 {
		t.Fatalf("failed Redeem calls must not be audited, got %d events: %+v", len(evs), evs)
	}
	env.repo.failErr = nil
	if _, err := env.svc.Redeem(ctx, "loy-1", 10); err != nil {
		t.Fatal(err)
	}
	if evs := env.rec.all(); len(evs) != 1 {
		t.Fatalf("successful Redeem after failures: want 1 audit event, got %d: %+v", len(evs), evs)
	}
}

// Catches: errors not wrapped with errs.Wrap("loyalty.redeem", err), including
// the old "Redeem" op and double wrapping.
func TestM5RedeemErrorWrapping(t *testing.T) {
	env := m5Setup()
	env.seed(t, "loy-1", 100)
	env.seed(t, "loy-2", 100)
	ctx := m5Ctx(t, 1)

	_, err := env.svc.Redeem(ctx, "loy-1", 0)
	m5CheckWrapped(t, "zero points", err, loyalty.ErrInvalidPoints, true)
	_, err = env.svc.Redeem(ctx, "loy-1", 101)
	m5CheckWrapped(t, "insufficient points", err, loyalty.ErrInsufficientPoints, true)
	_, err = env.svc.Redeem(ctx, "loy-404", 1)
	m5CheckWrapped(t, "unknown account", err, loyalty.ErrNotFound, false)

	saveErr := errors.New("m5: write timeout")
	env.repo.failErr = saveErr
	_, err = env.svc.Redeem(ctx, "loy-1", 1)
	m5CheckWrapped(t, "repository failure", err, saveErr, true)
	env.repo.failErr = nil

	recErr := errors.New("m5: audit sink unavailable")
	env.rec.failErr = recErr
	_, err = env.svc.Redeem(ctx, "loy-2", 30)
	m5CheckWrapped(t, "audit failure", err, recErr, true)
	if a, _ := env.base.Load(ctx, "loy-2"); a == nil || a.Balance != 70 {
		t.Fatalf("audit failure: the redemption must already be stored, got %+v", a)
	}
}
