package tax

import (
	"fmt"
	"strings"
)

// EncodeExemption renders v in the key=value wire format used by the nightly
// export and the v1 admin screens. Field order is stable.
func EncodeExemption(v *Exemption) string {
	var b strings.Builder
	putPair(&b, "id", v.ID)
	putPair(&b, "customer_id", v.CustomerID)
	putPair(&b, "certificate", v.Certificate)
	putPair(&b, "region", v.Region)
	putPair(&b, "expires_at", fmtTime(v.ExpiresAt))
	putPair(&b, "status", v.Status.String())
	putPair(&b, "created_at", fmtTime(v.CreatedAt))
	putPair(&b, "updated_at", fmtTime(v.UpdatedAt))
	return b.String()
}

// DecodeExemption parses the output of EncodeExemption.
func DecodeExemption(line string) (*Exemption, error) {
	v := &Exemption{}
	for _, part := range splitUnescaped(line, ';') {
		if part == "" {
			continue
		}
		key, raw, ok := cutUnescaped(part, '=')
		if !ok {
			return nil, fmt.Errorf("%w: malformed exemption field %q", ErrInvalid, part)
		}
		val := unesc(raw)
		switch unesc(key) {
		case "id":
			v.ID = val
		case "customer_id":
			v.CustomerID = val
		case "certificate":
			v.Certificate = val
		case "region":
			v.Region = val
		case "expires_at":
			ts, err := parseTimeField(val)
			if err != nil {
				return nil, fmt.Errorf("%w: expires_at: %v", ErrInvalid, err)
			}
			v.ExpiresAt = ts
		case "status":
			st, err := ParseExemptionStatus(val)
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
			return nil, fmt.Errorf("%w: unknown exemption field %q", ErrInvalid, key)
		}
	}
	if v.ID == "" {
		return nil, fmt.Errorf("%w: exemption record without id", ErrInvalid)
	}
	return v, nil
}
