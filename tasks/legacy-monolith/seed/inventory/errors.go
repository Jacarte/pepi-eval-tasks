package inventory

import (
	"errors"
	"fmt"
)

var (
	// ErrNotFound is returned when a inventory record does not exist.
	ErrNotFound = errors.New("inventory: not found")
	// ErrInvalid marks validation failures. Use errors.Is to test for it.
	ErrInvalid = errors.New("inventory: invalid")
)

// ValidationError describes a single invalid field. It unwraps to ErrInvalid.
type ValidationError struct {
	Entity string
	Field  string
	Reason string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("inventory: invalid %s.%s: %s", e.Entity, e.Field, e.Reason)
}

func (e *ValidationError) Unwrap() error { return ErrInvalid }
