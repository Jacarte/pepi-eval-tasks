package returns

import (
	"errors"
)

var (
	// ErrNotFound is returned when a returns record does not exist.
	ErrNotFound = errors.New("returns: not found")
	// ErrInvalid marks validation failures. Use errors.Is to test for it.
	ErrInvalid = errors.New("returns: invalid")
)
