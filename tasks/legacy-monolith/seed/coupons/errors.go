package coupons

import (
	"errors"
)

var (
	// ErrNotFound is returned when a coupons record does not exist.
	ErrNotFound = errors.New("coupons: not found")
	// ErrInvalid marks validation failures. Use errors.Is to test for it.
	ErrInvalid = errors.New("coupons: invalid")
)
