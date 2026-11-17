package warehouse

import (
	"fmt"
	"strconv"
	"strings"
)

// EncodeBinSlot renders v in the key=value wire format used by the nightly
// export and the v1 admin screens. Field order is stable.
func EncodeBinSlot(v *BinSlot) string {
	var b strings.Builder
	appendField(&b, "id", v.ID)
	appendField(&b, "zone", v.Zone)
	appendField(&b, "aisle", strconv.Itoa(v.Aisle))
	appendField(&b, "level", strconv.Itoa(v.Level))
	appendField(&b, "sku", v.SKU)
	appendField(&b, "capacity", strconv.Itoa(v.Capacity))
	appendField(&b, "status", v.Status.String())
	appendField(&b, "created_at", timeString(v.CreatedAt))
	appendField(&b, "updated_at", timeString(v.UpdatedAt))
	return b.String()
}

// DecodeBinSlot parses the output of EncodeBinSlot.
// Unknown keys are ignored so newer exports can be read by older jobs.
func DecodeBinSlot(line string) (*BinSlot, error) {
	v := &BinSlot{}
	for _, part := range splitOn(line, ';') {
		if part == "" {
			continue
		}
		key, raw, ok := cutOn(part, '=')
		if !ok {
			return nil, fmt.Errorf("%w: malformed bin slot field %q", ErrInvalid, part)
		}
		val := unquoteValue(raw)
		switch unquoteValue(key) {
		case "id":
			v.ID = val
		case "zone":
			v.Zone = val
		case "aisle":
			n, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("%w: aisle: %v", ErrInvalid, err)
			}
			v.Aisle = n
		case "level":
			n, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("%w: level: %v", ErrInvalid, err)
			}
			v.Level = n
		case "sku":
			v.SKU = val
		case "capacity":
			n, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("%w: capacity: %v", ErrInvalid, err)
			}
			v.Capacity = n
		case "status":
			st, err := ParseBinSlotStatus(val)
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
		return nil, fmt.Errorf("%w: bin slot record without id", ErrInvalid)
	}
	return v, nil
}
