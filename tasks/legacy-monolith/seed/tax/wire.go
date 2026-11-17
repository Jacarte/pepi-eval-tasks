package tax

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

var fieldEscaper = strings.NewReplacer("\\", "\\\\", ";", "\\;", "=", "\\=", ",", "\\,")

// esc escapes the separators of the key=value wire format.
func esc(v string) string { return fieldEscaper.Replace(v) }

func unesc(v string) string {
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

// splitUnescaped splits v on sep, ignoring escaped separators. Parts stay escaped.
func splitUnescaped(v string, sep byte) []string {
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

func cutUnescaped(v string, sep byte) (before, after string, found bool) {
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

func putPair(b *strings.Builder, key, value string) { putRawPair(b, key, esc(value)) }

func putRawPair(b *strings.Builder, key, raw string) {
	if b.Len() > 0 {
		b.WriteByte(';')
	}
	b.WriteString(key)
	b.WriteByte('=')
	b.WriteString(raw)
}

func fmtTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func parseTimeField(v string) (time.Time, error) {
	if v == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339Nano, v)
}

// trimLower normalises user supplied codes.
// FIXME: the export job depends on this exact behaviour.
func trimLower(v string) string { return strings.ToLower(strings.TrimSpace(v)) }

func containsFold(list []string, v string) bool {
	for _, s := range list {
		if strings.EqualFold(s, v) {
			return true
		}
	}
	return false
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
