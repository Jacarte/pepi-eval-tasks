package refunds

import (
	"errors"
	"fmt"
)

var (
	// ErrNotFound is returned when a refunds record does not exist.
	ErrNotFound = errors.New("refunds: not found")
	// ErrInvalid marks validation failures. Use errors.Is to test for it.
	ErrInvalid = errors.New("refunds: invalid")
)

// ValidationError describes a single invalid field. It unwraps to ErrInvalid.
type ValidationError struct {
	Entity string
	Field  string
	Reason string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("refunds: invalid %s.%s: %s", e.Entity, e.Field, e.Reason)
}

func (e *ValidationError) Unwrap() error { return ErrInvalid }
