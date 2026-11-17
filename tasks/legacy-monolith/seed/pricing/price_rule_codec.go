package pricing

import (
	"fmt"
	"strconv"
	"strings"
)

// EncodePriceRule renders v in the key=value wire format used by the nightly
// export and the v1 admin screens. Field order is stable.
func EncodePriceRule(v *PriceRule) string {
	var b strings.Builder
	putPair(&b, "id", v.ID)
	putPair(&b, "sku", v.SKU)
	putPair(&b, "kind", v.Kind)
	putPair(&b, "value", strconv.FormatInt(v.Value, 10))
	putPair(&b, "priority", strconv.Itoa(v.Priority))
	putPair(&b, "stackable", strconv.FormatBool(v.Stackable))
	putPair(&b, "status", v.Status.String())
	putPair(&b, "created_at", fmtTime(v.CreatedAt))
	putPair(&b, "updated_at", fmtTime(v.UpdatedAt))
	return b.String()
}

// DecodePriceRule parses the output of EncodePriceRule.
func DecodePriceRule(line string) (*PriceRule, error) {
	v := &PriceRule{}
	for _, part := range splitUnescaped(line, ';') {
		if part == "" {
			continue
		}
		key, raw, ok := cutUnescaped(part, '=')
		if !ok {
			return nil, fmt.Errorf("%w: malformed price rule field %q", ErrInvalid, part)
		}
		val := unesc(raw)
		switch unesc(key) {
		case "id":
			v.ID = val
		case "sku":
			v.SKU = val
		case "kind":
			v.Kind = val
		case "value":
			n, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("%w: value: %v", ErrInvalid, err)
			}
			v.Value = n
		case "priority":
			n, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("%w: priority: %v", ErrInvalid, err)
			}
			v.Priority = n
		case "stackable":
			flag, err := strconv.ParseBool(val)
			if err != nil {
				return nil, fmt.Errorf("%w: stackable: %v", ErrInvalid, err)
			}
			v.Stackable = flag
		case "status":
			st, err := ParsePriceRuleStatus(val)
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
			return nil, fmt.Errorf("%w: unknown price rule field %q", ErrInvalid, key)
		}
	}
	if v.ID == "" {
		return nil, fmt.Errorf("%w: price rule record without id", ErrInvalid)
	}
	return v, nil
}
