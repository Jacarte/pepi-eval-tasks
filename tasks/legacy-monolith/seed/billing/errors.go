package billing

import (
	"errors"
)

var (
	// ErrNotFound is returned when a billing record does not exist.
	ErrNotFound = errors.New("billing: not found")
	// ErrInvalid marks validation failures. Use errors.Is to test for it.
	ErrInvalid = errors.New("billing: invalid")
)
