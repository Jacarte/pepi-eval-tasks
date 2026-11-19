package jobq

import "time"

// BackoffKind selects how the wait grows between attempts.
type BackoffKind string

const (
	Constant    BackoffKind = "constant"
	Linear      BackoffKind = "linear"
	Exponential BackoffKind = "exponential"
)

// Backoff describes the waits between attempts. Kind Linear waits Base, 2*Base, 3*Base, ...;
// Exponential waits Base, 2*Base, 4*Base, ...; Constant waits Base every time. A non-zero Max
// caps every wait. The zero value is DefaultBackoff.
type Backoff struct {
	Kind BackoffKind
	Base time.Duration
	Max  time.Duration
}

// DefaultBackoff is used when Config.Backoff is the zero value.
var DefaultBackoff = Backoff{Kind: Constant, Base: time.Second}

// wait returns the wait after the n-th failed attempt, n starting at 1.
func (b Backoff) wait(n int) time.Duration {
	if b == (Backoff{}) {
		b = DefaultBackoff
	}
	d := b.Base
	switch b.Kind {
	case Linear:
		d = b.Base * time.Duration(n)
	case Exponential:
		d = b.Base << (n - 1)
	}
	if b.Max > 0 && d > b.Max {
		d = b.Max
	}
	return d
}
