package scheduling

import (
	"fmt"
	"strconv"
	"strings"
)

// EncodeSlot renders v in the key=value wire format used by the nightly
// export and the v1 admin screens. Field order is stable.
func EncodeSlot(v *Slot) string {
	var b strings.Builder
	appendField(&b, "id", v.ID)
	appendField(&b, "warehouse", v.Warehouse)
	appendField(&b, "window", v.Window)
	appendField(&b, "capacity", strconv.Itoa(v.Capacity))
	appendField(&b, "booked", strconv.Itoa(v.Booked))
	appendField(&b, "status", v.Status.String())
	appendField(&b, "created_at", timeString(v.CreatedAt))
	appendField(&b, "updated_at", timeString(v.UpdatedAt))
	return b.String()
}

// DecodeSlot parses the output of EncodeSlot.
func DecodeSlot(line string) (*Slot, error) {
	v := &Slot{}
	for _, part := range splitOn(line, ';') {
		if part == "" {
			continue
		}
		key, raw, ok := cutOn(part, '=')
		if !ok {
			return nil, fmt.Errorf("%w: malformed slot field %q", ErrInvalid, part)
		}
		val := unquoteValue(raw)
		switch unquoteValue(key) {
		case "id":
			v.ID = val
		case "warehouse":
			v.Warehouse = val
		case "window":
			v.Window = val
		case "capacity":
			n, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("%w: capacity: %v", ErrInvalid, err)
			}
			v.Capacity = n
		case "booked":
			n, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("%w: booked: %v", ErrInvalid, err)
			}
			v.Booked = n
		case "status":
			st, err := ParseSlotStatus(val)
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
		default:
			return nil, fmt.Errorf("%w: unknown slot field %q", ErrInvalid, key)
		}
	}
	if v.ID == "" {
		return nil, fmt.Errorf("%w: slot record without id", ErrInvalid)
	}
	return v, nil
}
