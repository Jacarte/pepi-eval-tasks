package inventory

import (
	"fmt"
	"strconv"
	"strings"
)

// EncodeAdjustment renders v in the key=value wire format used by the nightly
// export and the v1 admin screens. Field order is stable.
func EncodeAdjustment(v *Adjustment) string {
	var b strings.Builder
	writeKV(&b, "id", v.ID)
	writeKV(&b, "sku", v.SKU)
	writeKV(&b, "delta", strconv.Itoa(v.Delta))
	writeKV(&b, "reason", v.Reason)
	writeKV(&b, "operator", v.Operator)
	writeKV(&b, "status", v.Status.String())
	writeKV(&b, "created_at", formatTime(v.CreatedAt))
	writeKV(&b, "updated_at", formatTime(v.UpdatedAt))
	return b.String()
}

// DecodeAdjustment parses the output of EncodeAdjustment.
func DecodeAdjustment(line string) (*Adjustment, error) {
	v := &Adjustment{}
	for _, part := range splitEscaped(line, ';') {
		if part == "" {
			continue
		}
		key, raw, ok := cutEscaped(part, '=')
		if !ok {
			return nil, fmt.Errorf("%w: malformed adjustment field %q", ErrInvalid, part)
		}
		val := unescapeField(raw)
		switch unescapeField(key) {
		case "id":
			v.ID = val
		case "sku":
			v.SKU = val
		case "delta":
			n, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("%w: delta: %v", ErrInvalid, err)
			}
			v.Delta = n
		case "reason":
			v.Reason = val
		case "operator":
			v.Operator = val
		case "status":
			st, err := ParseAdjustmentStatus(val)
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
		default:
			return nil, fmt.Errorf("%w: unknown adjustment field %q", ErrInvalid, key)
		}
	}
	if v.ID == "" {
		return nil, fmt.Errorf("%w: adjustment record without id", ErrInvalid)
	}
	return v, nil
}
