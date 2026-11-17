// Package inireader parses a small INI/TOML-like configuration format. See SPEC.md section 4.
package inireader

import (
	"errors"
	"fmt"
	"io"
)

var errNotImplemented = errors.New("inireader: not implemented")

// Kind is the type of a Value.
type Kind int

const (
	KindString Kind = iota + 1
	KindInt
	KindFloat
	KindBool
	KindArray
)

// Value is a parsed value. Only the field matching Kind is meaningful.
type Value struct {
	Kind  Kind
	Str   string  // KindString
	Int   int64   // KindInt
	Float float64 // KindFloat
	Bool  bool    // KindBool
	Array []Value // KindArray (non-nil, possibly empty)
}

// ParseError is returned for any syntax error.
type ParseError struct {
	Line int    // 1-based physical line number
	Msg  string // exact message from SPEC.md
}

func (e *ParseError) Error() string { return fmt.Sprintf("line %d: %s", e.Line, e.Msg) }

// File is a parsed configuration.
type File struct{}

// Parse reads all of r and parses it.
func Parse(r io.Reader) (*File, error) { return nil, errNotImplemented }

// ParseString parses s.
func ParseString(s string) (*File, error) { return nil, errNotImplemented }

// Sections returns declared section names in order of appearance.
func (f *File) Sections() []string { return nil }

// Subsections returns the distinct immediate child names below parent.
func (f *File) Subsections(parent string) []string { return nil }

// Keys returns the keys of section in definition order.
func (f *File) Keys(section string) []string { return nil }

// Lookup returns the raw value for section/key.
func (f *File) Lookup(section, key string) (Value, bool) { return Value{}, false }

// GetString returns a string value or def.
func (f *File) GetString(section, key, def string) string { return def }

// GetInt returns an int value or def.
func (f *File) GetInt(section, key string, def int64) int64 { return def }

// GetFloat returns a float (or int converted to float) value or def.
func (f *File) GetFloat(section, key string, def float64) float64 { return def }

// GetBool returns a bool value or def.
func (f *File) GetBool(section, key string, def bool) bool { return def }

// GetStrings returns an array of strings or def.
func (f *File) GetStrings(section, key string, def []string) []string { return def }

// GetInts returns an array of ints or def.
func (f *File) GetInts(section, key string, def []int64) []int64 { return def }
