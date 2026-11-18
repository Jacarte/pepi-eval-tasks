// Package tmpl is a small text template engine. See SPEC.md section 12.
package tmpl

import (
	"errors"
	"fmt"
)

var errNotImplemented = errors.New("tmpl: not implemented")

// Error reports a parse or execution error at the "{{" of an action.
type Error struct {
	Line, Column int // 1-based, Column in runes
	Msg          string
}

func (e *Error) Error() string { return fmt.Sprintf("%d:%d: %s", e.Line, e.Column, e.Msg) }

// Options configures a template.
type Options struct {
	EscapeHTML bool // escape the output of value actions
}

// Template is a parsed template.
type Template struct{}

// Parse parses src.
func Parse(src string, opts Options) (*Template, error) { return nil, errNotImplemented }

// Execute runs the template with data as both "$" and the initial dot.
func (t *Template) Execute(data any) (string, error) { return "", errNotImplemented }
