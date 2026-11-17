// Package csvparse reads and writes RFC 4180 style CSV. See SPEC.md section 3.
package csvparse

import (
	"errors"
	"fmt"
	"io"
)

var (
	ErrBareQuote      = errors.New("bare quote in unquoted field")
	ErrQuote          = errors.New("extraneous or missing quote in quoted field")
	ErrFieldCount     = errors.New("wrong number of fields")
	ErrInvalidOptions = errors.New("invalid options")
)

var errNotImplemented = errors.New("csvparse: not implemented")

// ParseError reports the position of a syntax or field-count error.
type ParseError struct {
	Line   int   // 1-based line
	Column int   // 1-based column, counted in runes
	Err    error // ErrBareQuote, ErrQuote or ErrFieldCount
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("line %d, column %d: %v", e.Line, e.Column, e.Err)
}

func (e *ParseError) Unwrap() error { return e.Err }

// RaggedPolicy decides what happens to records whose field count differs from the first record.
type RaggedPolicy int

const (
	RaggedError RaggedPolicy = iota // default
	RaggedAllow
	RaggedPad
)

// Options configures a Reader.
type Options struct {
	Delimiter  rune // 0 means ','
	Quote      rune // 0 means '"'
	Comment    rune // 0 disables comment lines
	LazyQuotes bool
	Ragged     RaggedPolicy
}

// Reader reads records from an io.Reader.
type Reader struct{}

// NewReader returns a Reader reading from r.
func NewReader(r io.Reader, opts Options) *Reader { return &Reader{} }

// Read returns the next record; at end of input it returns nil, io.EOF.
func (r *Reader) Read() ([]string, error) { return nil, errNotImplemented }

// ReadAll reads all remaining records.
func (r *Reader) ReadAll() ([][]string, error) { return nil, errNotImplemented }

// WriterOptions configures a Writer.
type WriterOptions struct {
	Delimiter rune // 0 means ','
	Quote     rune // 0 means '"'
	UseCRLF   bool // terminate records with "\r\n" instead of "\n"
	QuoteAll  bool // quote every field
}

// Writer writes CSV records.
type Writer struct{}

// NewWriter returns a Writer writing to w.
func NewWriter(w io.Writer, opts WriterOptions) *Writer { return &Writer{} }

// Write writes one record; output may be buffered until Flush.
func (w *Writer) Write(record []string) error { return errNotImplemented }

// Flush writes buffered data to the underlying writer.
func (w *Writer) Flush() error { return errNotImplemented }

// WriteAll writes all records, then flushes.
func (w *Writer) WriteAll(records [][]string) error { return errNotImplemented }
