package fraud

import (
	"fmt"
	"strconv"
	"strings"
)

// EncodeRule renders v in the key=value wire format used by the nightly
// export and the v1 admin screens. Field order is stable.
func EncodeRule(v *Rule) string {
	var b strings.Builder
	putPair(&b, "id", v.ID)
	putPair(&b, "name", v.Name)
	putPair(&b, "weight", strconv.Itoa(v.Weight))
	putPair(&b, "enabled", strconv.FormatBool(v.Enabled))
	putPair(&b, "expression", v.Expression)
	putPair(&b, "status", v.Status.String())
	putPair(&b, "created_at", fmtTime(v.CreatedAt))
	putPair(&b, "updated_at", fmtTime(v.UpdatedAt))
	return b.String()
}

// DecodeRule parses the output of EncodeRule.
func DecodeRule(line string) (*Rule, error) {
	v := &Rule{}
	for _, part := range splitUnescaped(line, ';') {
		if part == "" {
			continue
		}
		key, raw, ok := cutUnescaped(part, '=')
		if !ok {
			return nil, fmt.Errorf("%w: malformed rule field %q", ErrInvalid, part)
		}
		val := unesc(raw)
		switch unesc(key) {
		case "id":
			v.ID = val
		case "name":
			v.Name = val
		case "weight":
			n, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("%w: weight: %v", ErrInvalid, err)
			}
			v.Weight = n
		case "enabled":
			flag, err := strconv.ParseBool(val)
			if err != nil {
				return nil, fmt.Errorf("%w: enabled: %v", ErrInvalid, err)
			}
			v.Enabled = flag
		case "expression":
			v.Expression = val
		case "status":
			st, err := ParseRuleStatus(val)
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
			return nil, fmt.Errorf("%w: unknown rule field %q", ErrInvalid, key)
		}
	}
	if v.ID == "" {
		return nil, fmt.Errorf("%w: rule record without id", ErrInvalid)
	}
	return v, nil
}
