package refunds_test

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
	"shopcore/refunds"
)

const (
	m3Op    = "refunds.issue"
	m3Event = "refund.issued"
)

type m3Journal struct {
	mu    sync.Mutex
	steps []string
}

func (j *m3Journal) add(s string) {
	j.mu.Lock()
	j.steps = append(j.steps, s)
	j.mu.Unlock()
}

func (j *m3Journal) snapshot() []string {
	j.mu.Lock()
	defer j.mu.Unlock()
	return append([]string(nil), j.steps...)
}

type m3Repo struct {
	refunds.Repository
	j       *m3Journal
	failErr error
}

func (r *m3Repo) Insert(ctx context.Context, rf *refunds.Refund) error {
	if r.failErr != nil {
		r.j.add("insert-failed")
		return r.failErr
	}
	if err := r.Repository.Insert(ctx, rf); err != nil {
		return err
	}
	r.j.add("insert")
	return nil
}

type m3Recorder struct {
	mu      sync.Mutex
	j       *m3Journal
	events  []audit.Event
	failErr error
}

func (r *m3Recorder) Record(ctx context.Context, ev audit.Event) error {
	r.j.add("record")
	r.mu.Lock()
	r.events = append(r.events, ev)
	r.mu.Unlock()
	return r.failErr
}

func (r *m3Recorder) all() []audit.Event {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]audit.Event(nil), r.events...)
}

type m3Env struct {
	svc  *refunds.Service
	base *refunds.MemoryRepo
	repo *m3Repo
	rec  *m3Recorder
	j    *m3Journal
}

func m3Setup() *m3Env {
	j := &m3Journal{}
	base := refunds.NewMemoryRepo()
	repo := &m3Repo{Repository: base, j: j}
	rec := &m3Recorder{j: j}
	svc := refunds.New(refunds.Deps{
		Repo:      repo,
		Audit:     rec,
		Clock:     clock.NewFixed(time.Date(2024, 5, 6, 7, 8, 9, 0, time.UTC)),
		IDs:       ids.NewSequence(),
		MaxAmount: 50000,
	})
	return &m3Env{svc: svc, base: base, repo: repo, rec: rec, j: j}
}

func m3Ctx(t *testing.T, n int) context.Context {
	h := fnv.New32a()
	h.Write([]byte(t.Name()))
	ctx := ctxkeys.WithRequestID(context.Background(), fmt.Sprintf("req-%08x-%d", h.Sum32(), n))
	ctx = ctxkeys.WithTraceID(ctx, fmt.Sprintf("trace-%08x-%d", h.Sum32()^0x33cc33, n))
	return ctxkeys.WithActor(ctx, "u-m3")
}

func m3Valid(payment string) refunds.IssueRequest {
	return refunds.IssueRequest{PaymentID: payment, OrderID: "ord-" + payment, Amount: 1250, Reason: refunds.ReasonDamaged, Note: "box crushed"}
}

func m3CheckWrapped(t *testing.T, label string, err, cause error, exact bool) {
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
	if e.Op != m3Op {
		t.Errorf("%s: op = %q, want %q", label, e.Op, m3Op)
	}
	n := 0
	for _, op := range errs.Ops(err) {
		if op == m3Op {
			n++
		}
	}
	if n > 1 {
		t.Errorf("%s: %q wrapped %d times: %q", label, m3Op, n, err)
	}
	msg := err.Error()
	if exact {
		if want := m3Op + ": " + cause.Error(); msg != want {
			t.Errorf("%s: err.Error() = %q, want %q", label, msg, want)
		}
	} else if !strings.HasPrefix(msg, m3Op+": ") || !strings.Contains(msg, cause.Error()) {
		t.Errorf("%s: err.Error() = %q, want %q + ...%q", label, msg, m3Op+": ", cause.Error())
	}
}

// Catches: wrong or inconsistent event names, missing or duplicate events.
func TestM3IssueEventName(t *testing.T) {
	env := m3Setup()
	for n := 1; n <= 2; n++ {
		if _, err := env.svc.Issue(m3Ctx(t, n), m3Valid(fmt.Sprintf("pay-%d", n))); err != nil {
			t.Fatalf("Issue #%d: %v", n, err)
		}
	}
	evs := env.rec.all()
	if len(evs) != 2 {
		t.Fatalf("want exactly one audit event per successful Issue, got %d: %+v", len(evs), evs)
	}
	for i, ev := range evs {
		if ev.Name != m3Event {
			t.Errorf("event %d name = %q, want %q", i, ev.Name, m3Event)
		}
	}
}

// Catches: request id not taken from ctxkeys.RequestID, wrong subject.
func TestM3IssueRequestIDAndSubject(t *testing.T) {
	env := m3Setup()
	ctxA, ctxB := m3Ctx(t, 1), m3Ctx(t, 2)
	a, err := env.svc.Issue(ctxA, m3Valid("pay-a"))
	if err != nil {
		t.Fatal(err)
	}
	b, err := env.svc.Issue(ctxB, m3Valid("pay-b"))
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
			t.Errorf("event %d Subject = %q, want the refund id %q", i, evs[i].Subject, tc.id)
		}
	}
}

// Catches: recording before the repository write, recording asynchronously,
// recording more than once.
func TestM3IssueRecordsAfterWrite(t *testing.T) {
	env := m3Setup()
	ctx := m3Ctx(t, 1)
	r, err := env.svc.Issue(ctx, m3Valid("pay-1"))
	if err != nil {
		t.Fatal(err)
	}
	steps := env.j.snapshot()
	if len(steps) != 2 || steps[0] != "insert" || steps[1] != "record" {
		t.Fatalf("calls during Issue = %v, want [insert record]: the event must be recorded after the refund is stored and before Issue returns", steps)
	}
	if _, err := env.base.Get(ctx, r.ID); err != nil {
		t.Fatalf("refund not stored: %v", err)
	}
}

// Catches: audit events emitted on validation or repository failures.
func TestM3IssueNoAuditOnFailure(t *testing.T) {
	env := m3Setup()
	ctx := m3Ctx(t, 1)
	bad := []refunds.IssueRequest{
		{Amount: 10, Reason: refunds.ReasonGoodwill},
		{PaymentID: "pay-1", Amount: 0, Reason: refunds.ReasonGoodwill},
		{PaymentID: "pay-1", Amount: 10, Reason: "because"},
		{PaymentID: "pay-1", Amount: 50001, Reason: refunds.ReasonGoodwill},
	}
	for i, req := range bad {
		if _, err := env.svc.Issue(ctx, req); err == nil {
			t.Fatalf("bad request %d accepted", i)
		}
	}
	env.repo.failErr = errors.New("m3: repository unavailable")
	if _, err := env.svc.Issue(ctx, m3Valid("pay-1")); err == nil {
		t.Fatal("Issue succeeded although the repository failed")
	}
	if evs := env.rec.all(); len(evs) != 0 {
		t.Fatalf("failed Issue calls must not be audited, got %d events: %+v", len(evs), evs)
	}
	env.repo.failErr = nil
	if _, err := env.svc.Issue(ctx, m3Valid("pay-1")); err != nil {
		t.Fatal(err)
	}
	if evs := env.rec.all(); len(evs) != 1 {
		t.Fatalf("successful Issue after failures: want 1 audit event, got %d: %+v", len(evs), evs)
	}
}

// Catches: errors not wrapped with errs.Wrap("refunds.issue", err).
func TestM3IssueErrorWrapping(t *testing.T) {
	env := m3Setup()
	ctx := m3Ctx(t, 1)

	_, err := env.svc.Issue(ctx, refunds.IssueRequest{Amount: 10, Reason: refunds.ReasonGoodwill})
	m3CheckWrapped(t, "missing payment", err, refunds.ErrMissingPayment, true)
	_, err = env.svc.Issue(ctx, refunds.IssueRequest{PaymentID: "pay-1", Amount: -5, Reason: refunds.ReasonGoodwill})
	m3CheckWrapped(t, "negative amount", err, refunds.ErrInvalidAmount, true)
	_, err = env.svc.Issue(ctx, refunds.IssueRequest{PaymentID: "pay-1", Amount: 50001, Reason: refunds.ReasonGoodwill})
	m3CheckWrapped(t, "over limit", err, refunds.ErrOverLimit, true)
	_, err = env.svc.Issue(ctx, refunds.IssueRequest{PaymentID: "pay-1", Amount: 5, Reason: "because"})
	m3CheckWrapped(t, "unknown reason", err, refunds.ErrUnknownReason, false)

	saveErr := errors.New("m3: write timeout")
	env.repo.failErr = saveErr
	_, err = env.svc.Issue(ctx, m3Valid("pay-1"))
	m3CheckWrapped(t, "repository failure", err, saveErr, true)
	env.repo.failErr = nil

	recErr := errors.New("m3: audit sink unavailable")
	env.rec.failErr = recErr
	_, err = env.svc.Issue(ctx, m3Valid("pay-wrap"))
	m3CheckWrapped(t, "audit failure", err, recErr, true)
	if list, _ := env.base.ForPayment(ctx, "pay-wrap"); len(list) != 1 {
		t.Fatalf("audit failure: the refund must already be stored, found %d", len(list))
	}
}
