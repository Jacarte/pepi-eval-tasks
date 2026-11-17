package searchindex

import (
	"fmt"
	"strconv"
	"strings"
)

// EncodeDocument renders v in the key=value wire format used by the nightly
// export and the v1 admin screens. Field order is stable.
func EncodeDocument(v *Document) string {
	var b strings.Builder
	writeKV(&b, "id", v.ID)
	writeKV(&b, "kind", v.Kind)
	writeKV(&b, "title", v.Title)
	writeKV(&b, "body", v.Body)
	writeKV(&b, "boost", strconv.Itoa(v.Boost))
	parts := make([]string, 0, len(v.Tags))
	for _, tag := range v.Tags {
		parts = append(parts, escapeField(tag))
	}
	writeRawKV(&b, "tags", strings.Join(parts, ","))
	writeKV(&b, "status", v.Status.String())
	writeKV(&b, "created_at", formatTime(v.CreatedAt))
	writeKV(&b, "updated_at", formatTime(v.UpdatedAt))
	return b.String()
}

// DecodeDocument parses the output of EncodeDocument.
// Unknown keys are ignored so newer exports can be read by older jobs.
func DecodeDocument(line string) (*Document, error) {
	v := &Document{}
	for _, part := range splitEscaped(line, ';') {
		if part == "" {
			continue
		}
		key, raw, ok := cutEscaped(part, '=')
		if !ok {
			return nil, fmt.Errorf("%w: malformed document field %q", ErrInvalid, part)
		}
		val := unescapeField(raw)
		switch unescapeField(key) {
		case "id":
			v.ID = val
		case "kind":
			v.Kind = val
		case "title":
			v.Title = val
		case "body":
			v.Body = val
		case "boost":
			n, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("%w: boost: %v", ErrInvalid, err)
			}
			v.Boost = n
		case "tags":
			v.Tags = nil
			if raw != "" {
				for _, tag := range splitEscaped(raw, ',') {
					v.Tags = append(v.Tags, unescapeField(tag))
				}
			}
		case "status":
			st, err := ParseDocumentStatus(val)
			if err != nil {
				return nil, err
			}
			v.Status = st
		case "created_at":
			ts, err := parseTime(val)
			if err != nil {
				return nil, fmt.Errorf("%w: created_at: %v", ErrInvalid, err)
			}
			v.CreatedAt = ts
		case "updated_at":
			ts, err := parseTime(val)
			if err != nil {
				return nil, fmt.Errorf("%w: updated_at: %v", ErrInvalid, err)
			}
			v.UpdatedAt = ts
		}
	}
	if v.ID == "" {
		return nil, fmt.Errorf("%w: document record without id", ErrInvalid)
	}
	return v, nil
}
