// Package cron parses five-field cron expressions and computes the times they
// match. See SPEC.md section 8.
package cron

import (
	"errors"
	"time"
)

var errNotImplemented = errors.New("cron: not implemented")

// ParseError reports an invalid cron expression.
type ParseError struct {
	Field string // "minute", "hour", "day-of-month", "month", "day-of-week", or ""
	Msg   string
}

func (e *ParseError) Error() string {
	if e.Field == "" {
		return "cron: " + e.Msg
	}
	return "cron: " + e.Field + " field: " + e.Msg
}

// Schedule is a parsed cron expression.
type Schedule struct{}

// Parse parses a cron expression.
func Parse(expr string) (*Schedule, error) { return nil, errNotImplemented }

// Matches reports whether the minute containing t (in UTC) matches.
func (s *Schedule) Matches(t time.Time) bool { return false }

// Next returns the first matching minute strictly after after.
func (s *Schedule) Next(after time.Time) (time.Time, bool) { return time.Time{}, false }

// NextN returns up to n successive matching times after after.
func (s *Schedule) NextN(after time.Time, n int) []time.Time { return nil }
