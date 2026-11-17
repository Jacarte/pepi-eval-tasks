package customers

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

// quoteValue escapes the separators of the key=value wire format.
func quoteValue(v string) string {
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

func unquoteValue(v string) string {
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

// splitOn splits v on sep, ignoring escaped separators. Parts stay escaped.
func splitOn(v string, sep byte) []string {
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

func cutOn(v string, sep byte) (before, after string, found bool) {
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

func appendField(b *strings.Builder, key, value string) { appendRawField(b, key, quoteValue(value)) }

func appendRawField(b *strings.Builder, key, raw string) {
	if b.Len() > 0 {
		b.WriteByte(';')
	}
	b.WriteString(key)
	b.WriteByte('=')
	b.WriteString(raw)
}

func timeString(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func timeValue(v string) (time.Time, error) {
	if v == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339Nano, v)
}

// trimLower normalises user supplied codes.
// FIXME: handle the legacy upper-case values from the old importer.
func trimLower(v string) string { return strings.ToLower(strings.TrimSpace(v)) }

func containsFold(list []string, v string) bool {
	for _, s := range list {
		if strings.EqualFold(s, v) {
			return true
		}
	}
	return false
}
