// Package audit records business events for compliance and support tooling.
//
// Production wires a Recorder that ships events to the audit_events table via
// the auditd sidecar (not part of this module). Tests and the dev server use
// Memory.
package audit

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"shopcore/clock"
)

// Event is a single audited business fact.
type Event struct {
	Name      string
	RequestID string
	Subject   string            // id of the record the event is about
	Attrs     map[string]string // optional, small; values are not indexed
	At        time.Time         // set by the recorder when zero
}

// String renders the event for log lines with attributes in key order.
func (e Event) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s subject=%s request=%s", e.Name, e.Subject, e.RequestID)
	keys := make([]string, 0, len(e.Attrs))
	for k := range e.Attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(&b, " %s=%s", k, e.Attrs[k])
	}
	return b.String()
}

// Recorder persists audit events.
type Recorder interface {
	Record(ctx context.Context, ev Event) error
}

// RecorderFunc adapts a function to Recorder.
type RecorderFunc func(ctx context.Context, ev Event) error

func (f RecorderFunc) Record(ctx context.Context, ev Event) error { return f(ctx, ev) }

// Discard drops every event.
type Discard struct{}

func (Discard) Record(context.Context, Event) error { return nil }

// OrDiscard returns r, or Discard when r is nil.
func OrDiscard(r Recorder) Recorder {
	if r == nil {
		return Discard{}
	}
	return r
}

// ErrMissingName is returned by Memory for events without a name.
var ErrMissingName = errors.New("audit: event name is required")

// Memory keeps events in memory, in recording order.
type Memory struct {
	mu     sync.Mutex
	clock  clock.Clock
	events []Event
}

// NewMemory returns an empty recorder stamping events with clk (or the system
// clock when clk is nil).
func NewMemory(clk clock.Clock) *Memory {
	return &Memory{clock: clock.OrSystem(clk)}
}

func (m *Memory) Record(ctx context.Context, ev Event) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if ev.Name == "" {
		return ErrMissingName
	}
	if ev.At.IsZero() {
		ev.At = m.clock.Now()
	}
	if ev.Attrs != nil {
		cp := make(map[string]string, len(ev.Attrs))
		for k, v := range ev.Attrs {
			cp[k] = v
		}
		ev.Attrs = cp
	}
	m.mu.Lock()
	m.events = append(m.events, ev)
	m.mu.Unlock()
	return nil
}

// Events returns a copy of everything recorded so far.
func (m *Memory) Events() []Event {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]Event(nil), m.events...)
}

// Named returns the recorded events with the given name.
func (m *Memory) Named(name string) []Event {
	var out []Event
	for _, ev := range m.Events() {
		if ev.Name == name {
			out = append(out, ev)
		}
	}
	return out
}

// ForSubject returns the recorded events about one record.
func (m *Memory) ForSubject(subject string) []Event {
	var out []Event
	for _, ev := range m.Events() {
		if ev.Subject == subject {
			out = append(out, ev)
		}
	}
	return out
}

// Reset forgets all events.
func (m *Memory) Reset() {
	m.mu.Lock()
	m.events = nil
	m.mu.Unlock()
}

// Multi records to every recorder in order and stops at the first error.
type Multi []Recorder

func (mr Multi) Record(ctx context.Context, ev Event) error {
	for _, r := range mr {
		if err := r.Record(ctx, ev); err != nil {
			return err
		}
	}
	return nil
}
