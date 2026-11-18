// Package udiff computes line diffs with Myers' algorithm, formats them as
// unified diffs and applies unified diffs. See SPEC.md section 11.
package udiff

import (
	"errors"
	"fmt"
)

var errNotImplemented = errors.New("udiff: not implemented")

// OpKind is the kind of an edit.
type OpKind int

const (
	OpEqual OpKind = iota
	OpDelete
	OpInsert
)

// Edit is one line of an edit script.
type Edit struct {
	Kind OpKind
	Line string // the line, including its "\n" if it has one
}

// SplitLines splits s after every "\n".
func SplitLines(s string) []string { return nil }

// Diff returns the shortest edit script from a to b chosen by Myers' algorithm.
func Diff(a, b []string) []Edit { return nil }

// Unified formats the diff of a and b as a unified diff with context lines.
func Unified(aName, bName, a, b string, context int) string { return "" }

// ApplyError reports why a patch could not be applied.
type ApplyError struct {
	Line int    // 1-based line number in the patch text
	Msg  string // exact messages listed in SPEC.md
}

func (e *ApplyError) Error() string { return fmt.Sprintf("patch line %d: %s", e.Line, e.Msg) }

// Apply applies a unified diff to original.
func Apply(original, patch string) (string, error) { return "", errNotImplemented }
