// Package expr parses and evaluates arithmetic, comparison, boolean and string
// expressions. See SPEC.md section 10.
package expr

import (
	"errors"
	"fmt"
)

var errNotImplemented = errors.New("expr: not implemented")

// Error reports a lexical, syntax or evaluation error with its position.
type Error struct {
	Line, Column int // 1-based; Column counts runes within the line
	Msg          string
}

func (e *Error) Error() string { return fmt.Sprintf("%d:%d: %s", e.Line, e.Column, e.Msg) }

// Expr is a parsed expression.
type Expr struct{}

// Parse parses src.
func Parse(src string) (*Expr, error) { return nil, errNotImplemented }

// String returns the fully parenthesized form of e.
func (e *Expr) String() string { return "" }

// Eval evaluates e with the variables in env.
func (e *Expr) Eval(env map[string]any) (any, error) { return nil, errNotImplemented }

// Eval parses and evaluates src.
func Eval(src string, env map[string]any) (any, error) {
	e, err := Parse(src)
	if err != nil {
		return nil, err
	}
	return e.Eval(env)
}
