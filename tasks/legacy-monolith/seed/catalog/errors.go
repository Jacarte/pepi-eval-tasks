package catalog

import (
	"errors"
)

var (
	// ErrNotFound is returned when a catalog record does not exist.
	ErrNotFound = errors.New("catalog: not found")
	// ErrInvalid marks validation failures. Use errors.Is to test for it.
	ErrInvalid = errors.New("catalog: invalid")
)
