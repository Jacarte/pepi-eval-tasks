package payments

import (
	"strings"
	"time"
)

// FieldChange is one differing field between two versions of a record.
// The admin UI renders these in the history tab.
type FieldChange struct {
	Field string
	From  string
	To    string
}

// escapeField escapes the separators of the key=value wire format.
func escapeField(v string) string {
	if !strings.ContainsAny(v, "\\;=,") {
		return v
	}
	var b strings.Builder
	for i := 0; i < len(v); i++ {
		switch v[i] {
		case '\\', ';', '=', ',':
			b.WriteByte('\\')
		}
		b.WriteByte(v[i])
	}
	return b.String()
}

func unescapeField(v string) string {
	if !strings.Contains(v, "\\") {
		return v
	}
	var b strings.Builder
	for i := 0; i < len(v); i++ {
		if v[i] == '\\' && i+1 < len(v) {
			i++
		}
		b.WriteByte(v[i])
	}
	return b.String()
}

// splitEscaped splits v on sep, ignoring escaped separators. Parts stay escaped.
func splitEscaped(v string, sep byte) []string {
	var out []string
	start := 0
	for i := 0; i < len(v); i++ {
		if v[i] == '\\' {
			i++
			continue
		}
		if v[i] == sep {
			out = append(out, v[start:i])
			start = i + 1
		}
	}
	return append(out, v[start:])
}

func cutEscaped(v string, sep byte) (before, after string, found bool) {
	for i := 0; i < len(v); i++ {
		if v[i] == '\\' {
			i++
			continue
		}
		if v[i] == sep {
			return v[:i], v[i+1:], true
		}
	}
	return v, "", false
}

func writeKV(b *strings.Builder, key, value string) { writeRawKV(b, key, escapeField(value)) }

func writeRawKV(b *strings.Builder, key, raw string) {
	if b.Len() > 0 {
		b.WriteByte(';')
	}
	b.WriteString(key)
	b.WriteByte('=')
	b.WriteString(raw)
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func parseTime(v string) (time.Time, error) {
	if v == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339Nano, v)
}

// trimLower normalises user supplied codes.
// NOTE(pc): handle the legacy upper-case values from the old importer.
func trimLower(v string) string { return strings.ToLower(strings.TrimSpace(v)) }

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
