package orders

import (
	"fmt"
	"strconv"
	"strings"

	"shopcore/money"
)

// EncodeDraft renders v in the key=value wire format used by the nightly
// export and the v1 admin screens. Field order is stable.
func EncodeDraft(v *Draft) string {
	var b strings.Builder
	putPair(&b, "id", v.ID)
	putPair(&b, "customer_id", v.CustomerID)
	putPair(&b, "items", strconv.Itoa(v.Items))
	putPair(&b, "note", v.Note)
	putPair(&b, "total", strconv.FormatInt(int64(v.Total), 10))
	putPair(&b, "status", v.Status.String())
	putPair(&b, "created_at", fmtTime(v.CreatedAt))
	putPair(&b, "updated_at", fmtTime(v.UpdatedAt))
	return b.String()
}

// DecodeDraft parses the output of EncodeDraft.
func DecodeDraft(line string) (*Draft, error) {
	v := &Draft{}
	for _, part := range splitUnescaped(line, ';') {
		if part == "" {
			continue
		}
		key, raw, ok := cutUnescaped(part, '=')
		if !ok {
			return nil, fmt.Errorf("%w: malformed draft field %q", ErrInvalid, part)
		}
		val := unesc(raw)
		switch unesc(key) {
		case "id":
			v.ID = val
		case "customer_id":
			v.CustomerID = val
		case "items":
			n, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("%w: items: %v", ErrInvalid, err)
			}
			v.Items = n
		case "note":
			v.Note = val
		case "total":
			n, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("%w: total: %v", ErrInvalid, err)
			}
			v.Total = money.Cents(n)
		case "status":
			st, err := ParseDraftStatus(val)
			if err != nil {
				return nil, err
			}
			v.Status = st
		case "created_at":
			ts, err := parseTimeField(val)
			if err != nil {
				return nil, fmt.Errorf("%w: created_at: %v", ErrInvalid, err)
			}
			v.CreatedAt = ts
		case "updated_at":
			ts, err := parseTimeField(val)
			if err != nil {
				return nil, fmt.Errorf("%w: updated_at: %v", ErrInvalid, err)
			}
			v.UpdatedAt = ts
		default:
			return nil, fmt.Errorf("%w: unknown draft field %q", ErrInvalid, key)
		}
	}
	if v.ID == "" {
		return nil, fmt.Errorf("%w: draft record without id", ErrInvalid)
	}
	return v, nil
}
