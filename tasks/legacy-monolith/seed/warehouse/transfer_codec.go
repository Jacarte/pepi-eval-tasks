package warehouse

import (
	"fmt"
	"strconv"
	"strings"
)

// EncodeTransfer renders v in the key=value wire format used by the nightly
// export and the v1 admin screens. Field order is stable.
func EncodeTransfer(v *Transfer) string {
	var b strings.Builder
	appendField(&b, "id", v.ID)
	appendField(&b, "from_bin", v.FromBin)
	appendField(&b, "to_bin", v.ToBin)
	appendField(&b, "sku", v.SKU)
	appendField(&b, "qty", strconv.Itoa(v.Qty))
	appendField(&b, "status", v.Status.String())
	appendField(&b, "created_at", timeString(v.CreatedAt))
	appendField(&b, "updated_at", timeString(v.UpdatedAt))
	return b.String()
}

// DecodeTransfer parses the output of EncodeTransfer.
// Unknown keys are ignored so newer exports can be read by older jobs.
func DecodeTransfer(line string) (*Transfer, error) {
	v := &Transfer{}
	for _, part := range splitOn(line, ';') {
		if part == "" {
			continue
		}
		key, raw, ok := cutOn(part, '=')
		if !ok {
			return nil, fmt.Errorf("%w: malformed transfer field %q", ErrInvalid, part)
		}
		val := unquoteValue(raw)
		switch unquoteValue(key) {
		case "id":
			v.ID = val
		case "from_bin":
			v.FromBin = val
		case "to_bin":
			v.ToBin = val
		case "sku":
			v.SKU = val
		case "qty":
			n, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("%w: qty: %v", ErrInvalid, err)
			}
			v.Qty = n
		case "status":
			st, err := ParseTransferStatus(val)
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
		return nil, fmt.Errorf("%w: transfer record without id", ErrInvalid)
	}
	return v, nil
}
