// Package jsonpatch implements JSON Pointer (RFC 6901) and JSON Patch
// (RFC 6902) on decoded JSON values. See SPEC.md section 7.
package jsonpatch

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidPointer   = errors.New("invalid JSON pointer")
	ErrPathNotFound     = errors.New("path not found")
	ErrInvalidIndex     = errors.New("invalid array index")
	ErrTestFailed       = errors.New("test operation failed")
	ErrInvalidOperation = errors.New("invalid operation")
	ErrInvalidDocument  = errors.New("invalid document")
)

var errNotImplemented = errors.New("jsonpatch: not implemented")

// PatchError reports the operation that made a patch fail.
type PatchError struct {
	Index int    // 0-based index of the failing operation; -1 when the patch as a whole is malformed
	Op    string // the operation's "op" ("" if absent or not a string)
	Path  string // the operation's "path" ("" if absent or not a string)
	Err   error  // one of the sentinel errors
}

func (e *PatchError) Error() string {
	return fmt.Sprintf("operation %d (%s %s): %v", e.Index, e.Op, e.Path, e.Err)
}

func (e *PatchError) Unwrap() error { return e.Err }

// ParsePointer splits a JSON pointer into unescaped reference tokens.
func ParsePointer(s string) ([]string, error) { return nil, errNotImplemented }

// FormatPointer joins tokens into a JSON pointer, escaping "~" and "/".
func FormatPointer(tokens []string) string { return "" }

// Get returns the value that pointer refers to in doc.
func Get(doc any, pointer string) (any, error) { return nil, errNotImplemented }

// Equal reports whether a and b are equal JSON values.
func Equal(a, b any) bool { return false }

// Operation is one JSON Patch operation.
type Operation struct {
	Op    string // "add", "remove", "replace", "move", "copy" or "test"
	Path  string
	From  string // move and copy only
	Value any    // add, replace and test only
}

// DecodePatch decodes a JSON Patch document.
func DecodePatch(data []byte) ([]Operation, error) { return nil, errNotImplemented }

// Apply returns doc with patch applied; doc itself is never modified.
func Apply(doc any, patch []Operation) (any, error) { return nil, errNotImplemented }

// ApplyJSON decodes doc and patch, applies the patch and encodes the result.
func ApplyJSON(doc, patch []byte) ([]byte, error) { return nil, errNotImplemented }
