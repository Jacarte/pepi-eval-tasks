// Package errs is the error annotation helper used at service boundaries.
package errs

import (
	"errors"
	"strings"
)

// Error annotates an underlying error with the operation that failed.
type Error struct {
	Op  string
	Err error
}

func (e *Error) Error() string {
	if e.Err == nil {
		return e.Op
	}
	return e.Op + ": " + e.Err.Error()
}

func (e *Error) Unwrap() error { return e.Err }

// Wrap annotates err with op. It returns nil when err is nil so it can be
// used directly in return statements.
func Wrap(op string, err error) error {
	if err == nil {
		return nil
	}
	return &Error{Op: op, Err: err}
}

// New returns an error for op with a plain message.
func New(op, msg string) error {
	return &Error{Op: op, Err: errors.New(msg)}
}

// Op returns the outermost operation recorded on err, or "".
func Op(err error) string {
	var e *Error
	if errors.As(err, &e) {
		return e.Op
	}
	return ""
}

// Ops returns every operation in the chain, outermost first.
func Ops(err error) []string {
	var out []string
	for err != nil {
		if e, ok := err.(*Error); ok {
			out = append(out, e.Op)
		}
		err = errors.Unwrap(err)
	}
	return out
}

// Trace renders the op chain as "a > b > c" for log lines.
func Trace(err error) string {
	return strings.Join(Ops(err), " > ")
}
