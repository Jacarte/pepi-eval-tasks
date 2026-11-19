package jobq

import "errors"

// ErrPermanent marks a failure that must not be retried.
var ErrPermanent = errors.New("permanent failure")

type permanentError struct{ err error }

func (e permanentError) Error() string   { return e.err.Error() }
func (e permanentError) Unwrap() []error { return []error{ErrPermanent, e.err} }

// Permanent wraps err so that Run does not retry it.
func Permanent(err error) error { return permanentError{err} }
