package ledger

import (
	"errors"
)

var (
	// ErrNotFound is returned when a ledger record does not exist.
	ErrNotFound = errors.New("ledger: not found")
	// ErrInvalid marks validation failures. Use errors.Is to test for it.
	ErrInvalid = errors.New("ledger: invalid")
)
