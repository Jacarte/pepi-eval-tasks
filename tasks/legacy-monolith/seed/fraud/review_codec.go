package fraud

import (
	"fmt"
	"strconv"
	"strings"
)

// EncodeReview renders v in the key=value wire format used by the nightly
// export and the v1 admin screens. Field order is stable.
func EncodeReview(v *Review) string {
	var b strings.Builder
	putPair(&b, "id", v.ID)
	putPair(&b, "order_id", v.OrderID)
	putPair(&b, "score", strconv.Itoa(v.Score))
	putPair(&b, "reason", v.Reason)
	putPair(&b, "analyst", v.Analyst)
	putPair(&b, "status", v.Status.String())
	putPair(&b, "created_at", fmtTime(v.CreatedAt))
	putPair(&b, "updated_at", fmtTime(v.UpdatedAt))
	return b.String()
}

// DecodeReview parses the output of EncodeReview.
func DecodeReview(line string) (*Review, error) {
	v := &Review{}
	for _, part := range splitUnescaped(line, ';') {
		if part == "" {
			continue
		}
		key, raw, ok := cutUnescaped(part, '=')
		if !ok {
			return nil, fmt.Errorf("%w: malformed review field %q", ErrInvalid, part)
		}
		val := unesc(raw)
		switch unesc(key) {
		case "id":
			v.ID = val
		case "order_id":
			v.OrderID = val
		case "score":
			n, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("%w: score: %v", ErrInvalid, err)
			}
			v.Score = n
		case "reason":
			v.Reason = val
		case "analyst":
			v.Analyst = val
		case "status":
			st, err := ParseReviewStatus(val)
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
			return nil, fmt.Errorf("%w: unknown review field %q", ErrInvalid, key)
		}
	}
	if v.ID == "" {
		return nil, fmt.Errorf("%w: review record without id", ErrInvalid)
	}
	return v, nil
}
