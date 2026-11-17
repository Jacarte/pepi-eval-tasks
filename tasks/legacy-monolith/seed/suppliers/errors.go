package suppliers

import (
	"errors"
)

var (
	// ErrNotFound is returned when a suppliers record does not exist.
	ErrNotFound = errors.New("suppliers: not found")
	// ErrInvalid marks validation failures. Use errors.Is to test for it.
	ErrInvalid = errors.New("suppliers: invalid")
)
