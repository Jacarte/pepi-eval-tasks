// Package server is the TCP line-protocol server (SPEC.md, milestone M5).
package server

import (
	"context"
	"errors"

	"kvdb/query"
)

const MaxLineLength = 1 << 20

var errNotImplemented = errors.New("server: not implemented")

type Server struct {
	e *query.Engine
}

func New(e *query.Engine) *Server { return &Server{e: e} }

func (s *Server) Start(addr string) error { return errNotImplemented }

func (s *Server) Addr() string { return "" }

func (s *Server) Shutdown(ctx context.Context) error { return errNotImplemented }
