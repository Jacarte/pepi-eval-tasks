package reports

import (
	"errors"
)

var (
	// ErrNotFound is returned when a reports record does not exist.
	ErrNotFound = errors.New("reports: not found")
	// ErrInvalid marks validation failures. Use errors.Is to test for it.
	ErrInvalid = errors.New("reports: invalid")
)
