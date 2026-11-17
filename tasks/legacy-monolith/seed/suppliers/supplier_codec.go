package suppliers

import (
	"fmt"
	"strconv"
	"strings"
)

// EncodeSupplier renders v in the key=value wire format used by the nightly
// export and the v1 admin screens. Field order is stable.
func EncodeSupplier(v *Supplier) string {
	var b strings.Builder
	writeKV(&b, "id", v.ID)
	writeKV(&b, "name", v.Name)
	writeKV(&b, "country", v.Country)
	writeKV(&b, "contact_email", v.ContactEmail)
	writeKV(&b, "lead_time_days", strconv.Itoa(v.LeadTimeDays))
	writeKV(&b, "preferred", strconv.FormatBool(v.Preferred))
	writeKV(&b, "status", v.Status.String())
	writeKV(&b, "created_at", formatTime(v.CreatedAt))
	writeKV(&b, "updated_at", formatTime(v.UpdatedAt))
	return b.String()
}

// DecodeSupplier parses the output of EncodeSupplier.
func DecodeSupplier(line string) (*Supplier, error) {
	v := &Supplier{}
	for _, part := range splitEscaped(line, ';') {
		if part == "" {
			continue
		}
		key, raw, ok := cutEscaped(part, '=')
		if !ok {
			return nil, fmt.Errorf("%w: malformed supplier field %q", ErrInvalid, part)
		}
		val := unescapeField(raw)
		switch unescapeField(key) {
		case "id":
			v.ID = val
		case "name":
			v.Name = val
		case "country":
			v.Country = val
		case "contact_email":
			v.ContactEmail = val
		case "lead_time_days":
			n, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("%w: lead_time_days: %v", ErrInvalid, err)
			}
			v.LeadTimeDays = n
		case "preferred":
			flag, err := strconv.ParseBool(val)
			if err != nil {
				return nil, fmt.Errorf("%w: preferred: %v", ErrInvalid, err)
			}
			v.Preferred = flag
		case "status":
			st, err := ParseSupplierStatus(val)
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
			return nil, fmt.Errorf("%w: unknown supplier field %q", ErrInvalid, key)
		}
	}
	if v.ID == "" {
		return nil, fmt.Errorf("%w: supplier record without id", ErrInvalid)
	}
	return v, nil
}
