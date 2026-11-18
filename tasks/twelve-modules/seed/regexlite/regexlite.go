// Package regexlite is a backtracking regular expression engine for a small
// Perl-like subset. See SPEC.md section 6.
package regexlite

import (
	"errors"
	"fmt"
)

var errNotImplemented = errors.New("regexlite: not implemented")

// SyntaxError reports an invalid pattern.
type SyntaxError struct {
	Pos int    // 0-based byte offset in the pattern
	Msg string // one of the messages listed in SPEC.md
}

func (e *SyntaxError) Error() string {
	return fmt.Sprintf("regexlite: %s at position %d", e.Msg, e.Pos)
}

// Regexp is a compiled pattern.
type Regexp struct{}

// Compile parses pattern.
func Compile(pattern string) (*Regexp, error) { return nil, errNotImplemented }

// MustCompile is like Compile but panics on error.
func MustCompile(pattern string) *Regexp {
	re, err := Compile(pattern)
	if err != nil {
		panic(err)
	}
	return re
}

// NumGroups returns the number of capturing groups.
func (re *Regexp) NumGroups() int { return 0 }

// MatchString reports whether s contains a match.
func (re *Regexp) MatchString(s string) bool { return false }

// FindStringSubmatchIndex returns the offsets of the leftmost match and its groups.
func (re *Regexp) FindStringSubmatchIndex(s string) []int { return nil }

// FindStringSubmatch returns the text of the leftmost match and its groups.
func (re *Regexp) FindStringSubmatch(s string) []string { return nil }

// FindAllString returns successive non-overlapping matches.
func (re *Regexp) FindAllString(s string, n int) []string { return nil }

// ReplaceAllString replaces every match with the expansion of repl.
func (re *Regexp) ReplaceAllString(src, repl string) string { return "" }
