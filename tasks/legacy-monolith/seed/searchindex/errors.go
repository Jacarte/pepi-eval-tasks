package searchindex

import (
	"errors"
	"fmt"
)

var (
	// ErrNotFound is returned when a searchindex record does not exist.
	ErrNotFound = errors.New("searchindex: not found")
	// ErrInvalid marks validation failures. Use errors.Is to test for it.
	ErrInvalid = errors.New("searchindex: invalid")
)

// ValidationError describes a single invalid field. It unwraps to ErrInvalid.
type ValidationError struct {
	Entity string
	Field  string
	Reason string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("searchindex: invalid %s.%s: %s", e.Entity, e.Field, e.Reason)
}

func (e *ValidationError) Unwrap() error { return ErrInvalid }
