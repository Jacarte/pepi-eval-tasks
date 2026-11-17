package notifications

import (
	"errors"
)

var (
	// ErrNotFound is returned when a notifications record does not exist.
	ErrNotFound = errors.New("notifications: not found")
	// ErrInvalid marks validation failures. Use errors.Is to test for it.
	ErrInvalid = errors.New("notifications: invalid")
)
