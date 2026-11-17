package payments

import (
	"errors"
	"fmt"
)

var (
	// ErrNotFound is returned when a payments record does not exist.
	ErrNotFound = errors.New("payments: not found")
	// ErrInvalid marks validation failures. Use errors.Is to test for it.
	ErrInvalid = errors.New("payments: invalid")
)

// ValidationError describes a single invalid field. It unwraps to ErrInvalid.
type ValidationError struct {
	Entity string
	Field  string
	Reason string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("payments: invalid %s.%s: %s", e.Entity, e.Field, e.Reason)
}

func (e *ValidationError) Unwrap() error { return ErrInvalid }
