// Package query implements the text query language (SPEC.md, milestones M2 and M3).
package query

import (
	"errors"

	"kvdb/store"
)

var ErrSyntax = errors.New("query: syntax error")

// Engine executes commands against one store. Safe for concurrent use.
type Engine struct {
	st *store.Store
}

// Session is one client's state, including its open transaction.
type Session struct {
	e *Engine
}

func NewEngine(st *store.Store) *Engine { return &Engine{st: st} }

func (e *Engine) NewSession() *Session { return &Session{e: e} }

// Exec runs one command line and returns the response without a trailing newline.
func (s *Session) Exec(line string) string { return "(error) ERR not implemented" }

func (s *Session) InTransaction() bool { return false }

func (s *Session) Close() {}

func Quote(b []byte) string { return "" }

func Unquote(s string) ([]byte, error) { return nil, ErrSyntax }
