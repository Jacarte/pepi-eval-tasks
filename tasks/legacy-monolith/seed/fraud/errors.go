package fraud

import (
	"errors"
)

var (
	// ErrNotFound is returned when a fraud record does not exist.
	ErrNotFound = errors.New("fraud: not found")
	// ErrInvalid marks validation failures. Use errors.Is to test for it.
	ErrInvalid = errors.New("fraud: invalid")
)
