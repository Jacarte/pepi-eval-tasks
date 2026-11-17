// Package clock abstracts time so services can be tested without sleeping.
package clock

import (
	"sync"
	"time"
)

// Clock reports the current time.
type Clock interface {
	Now() time.Time
}

// System is the wall clock, in UTC.
type System struct{}

func (System) Now() time.Time { return time.Now().UTC() }

// Func adapts a plain function to Clock.
type Func func() time.Time

func (f Func) Now() time.Time { return f() }

// Fixed is a manually advanced clock for tests and replays.
type Fixed struct {
	mu sync.Mutex
	t  time.Time
}

// NewFixed returns a clock stopped at t.
func NewFixed(t time.Time) *Fixed { return &Fixed{t: t} }

func (f *Fixed) Now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.t
}

// Advance moves the clock forward by d.
func (f *Fixed) Advance(d time.Duration) {
	f.mu.Lock()
	f.t = f.t.Add(d)
	f.mu.Unlock()
}

// Set jumps the clock to t.
func (f *Fixed) Set(t time.Time) {
	f.mu.Lock()
	f.t = t
	f.mu.Unlock()
}

// OrSystem returns c, or the system clock when c is nil.
func OrSystem(c Clock) Clock {
	if c == nil {
		return System{}
	}
	return c
}

// StartOfDay truncates t to midnight in its location.
func StartOfDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}
