package customers

import (
	"fmt"
	"strconv"
	"strings"
)

// EncodeAddress renders v in the key=value wire format used by the nightly
// export and the v1 admin screens. Field order is stable.
func EncodeAddress(v *Address) string {
	var b strings.Builder
	appendField(&b, "id", v.ID)
	appendField(&b, "customer_id", v.CustomerID)
	appendField(&b, "line1", v.Line1)
	appendField(&b, "city", v.City)
	appendField(&b, "postcode", v.Postcode)
	appendField(&b, "country", v.Country)
	appendField(&b, "primary", strconv.FormatBool(v.Primary))
	appendField(&b, "status", v.Status.String())
	appendField(&b, "created_at", timeString(v.CreatedAt))
	appendField(&b, "updated_at", timeString(v.UpdatedAt))
	return b.String()
}

// DecodeAddress parses the output of EncodeAddress.
// Unknown keys are ignored so newer exports can be read by older jobs.
func DecodeAddress(line string) (*Address, error) {
	v := &Address{}
	for _, part := range splitOn(line, ';') {
		if part == "" {
			continue
		}
		key, raw, ok := cutOn(part, '=')
		if !ok {
			return nil, fmt.Errorf("%w: malformed address field %q", ErrInvalid, part)
		}
		val := unquoteValue(raw)
		switch unquoteValue(key) {
		case "id":
			v.ID = val
		case "customer_id":
			v.CustomerID = val
		case "line1":
			v.Line1 = val
		case "city":
			v.City = val
		case "postcode":
			v.Postcode = val
		case "country":
			v.Country = val
		case "primary":
			flag, err := strconv.ParseBool(val)
			if err != nil {
				return nil, fmt.Errorf("%w: primary: %v", ErrInvalid, err)
			}
			v.Primary = flag
		case "status":
			st, err := ParseAddressStatus(val)
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
		return nil, fmt.Errorf("%w: address record without id", ErrInvalid)
	}
	return v, nil
}
