package customers

import (
	"fmt"
	"strconv"
	"strings"

	"shopcore/money"
)

// EncodeCustomer renders v in the key=value wire format used by the nightly
// export and the v1 admin screens. Field order is stable.
func EncodeCustomer(v *Customer) string {
	var b strings.Builder
	appendField(&b, "id", v.ID)
	appendField(&b, "email", v.Email)
	appendField(&b, "display_name", v.DisplayName)
	appendField(&b, "segment", v.Segment)
	appendField(&b, "lifetime_value", strconv.FormatInt(int64(v.LifetimeValue), 10))
	appendField(&b, "marketing", strconv.FormatBool(v.Marketing))
	appendField(&b, "status", v.Status.String())
	appendField(&b, "created_at", timeString(v.CreatedAt))
	appendField(&b, "updated_at", timeString(v.UpdatedAt))
	return b.String()
}

// DecodeCustomer parses the output of EncodeCustomer.
// Unknown keys are ignored so newer exports can be read by older jobs.
func DecodeCustomer(line string) (*Customer, error) {
	v := &Customer{}
	for _, part := range splitOn(line, ';') {
		if part == "" {
			continue
		}
		key, raw, ok := cutOn(part, '=')
		if !ok {
			return nil, fmt.Errorf("%w: malformed customer field %q", ErrInvalid, part)
		}
		val := unquoteValue(raw)
		switch unquoteValue(key) {
		case "id":
			v.ID = val
		case "email":
			v.Email = val
		case "display_name":
			v.DisplayName = val
		case "segment":
			v.Segment = val
		case "lifetime_value":
			n, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("%w: lifetime_value: %v", ErrInvalid, err)
			}
			v.LifetimeValue = money.Cents(n)
		case "marketing":
			flag, err := strconv.ParseBool(val)
			if err != nil {
				return nil, fmt.Errorf("%w: marketing: %v", ErrInvalid, err)
			}
			v.Marketing = flag
		case "status":
			st, err := ParseCustomerStatus(val)
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
		return nil, fmt.Errorf("%w: customer record without id", ErrInvalid)
	}
	return v, nil
}
