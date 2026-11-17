package loyalty

import (
	"fmt"
	"strconv"
	"strings"
)

// EncodeTier renders v in the key=value wire format used by the nightly
// export and the v1 admin screens. Field order is stable.
func EncodeTier(v *Tier) string {
	var b strings.Builder
	appendField(&b, "id", v.ID)
	appendField(&b, "name", v.Name)
	appendField(&b, "threshold", strconv.FormatInt(v.Threshold, 10))
	appendField(&b, "multiplier", strconv.Itoa(v.Multiplier))
	appendField(&b, "status", v.Status.String())
	appendField(&b, "created_at", timeString(v.CreatedAt))
	appendField(&b, "updated_at", timeString(v.UpdatedAt))
	return b.String()
}

// DecodeTier parses the output of EncodeTier.
// Unknown keys are ignored so newer exports can be read by older jobs.
func DecodeTier(line string) (*Tier, error) {
	v := &Tier{}
	for _, part := range splitOn(line, ';') {
		if part == "" {
			continue
		}
		key, raw, ok := cutOn(part, '=')
		if !ok {
			return nil, fmt.Errorf("%w: malformed tier field %q", ErrInvalid, part)
		}
		val := unquoteValue(raw)
		switch unquoteValue(key) {
		case "id":
			v.ID = val
		case "name":
			v.Name = val
		case "threshold":
			n, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("%w: threshold: %v", ErrInvalid, err)
			}
			v.Threshold = n
		case "multiplier":
			n, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("%w: multiplier: %v", ErrInvalid, err)
			}
			v.Multiplier = n
		case "status":
			st, err := ParseTierStatus(val)
			if err != nil {
				return nil, err
			}
			v.Status = st
		case "created_at":
			ts, err := timeValue(val)
			if err != nil {
				return nil, fmt.Errorf("%w: created_at: %v", ErrInvalid, err)
			}
			v.CreatedAt = ts
		case "updated_at":
			ts, err := timeValue(val)
			if err != nil {
				return nil, fmt.Errorf("%w: updated_at: %v", ErrInvalid, err)
			}
			v.UpdatedAt = ts
		}
	}
	if v.ID == "" {
		return nil, fmt.Errorf("%w: tier record without id", ErrInvalid)
	}
	return v, nil
}
