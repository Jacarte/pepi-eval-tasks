package returns

import (
	"fmt"
	"strconv"
	"strings"
)

// EncodeReturnRequest renders v in the key=value wire format used by the nightly
// export and the v1 admin screens. Field order is stable.
func EncodeReturnRequest(v *ReturnRequest) string {
	var b strings.Builder
	appendField(&b, "id", v.ID)
	appendField(&b, "order_id", v.OrderID)
	appendField(&b, "sku", v.SKU)
	appendField(&b, "qty", strconv.Itoa(v.Qty))
	appendField(&b, "reason", v.Reason)
	appendField(&b, "inspector", v.Inspector)
	appendField(&b, "status", v.Status.String())
	appendField(&b, "created_at", timeString(v.CreatedAt))
	appendField(&b, "updated_at", timeString(v.UpdatedAt))
	return b.String()
}

// DecodeReturnRequest parses the output of EncodeReturnRequest.
func DecodeReturnRequest(line string) (*ReturnRequest, error) {
	v := &ReturnRequest{}
	for _, part := range splitOn(line, ';') {
		if part == "" {
			continue
		}
		key, raw, ok := cutOn(part, '=')
		if !ok {
			return nil, fmt.Errorf("%w: malformed return request field %q", ErrInvalid, part)
		}
		val := unquoteValue(raw)
		switch unquoteValue(key) {
		case "id":
			v.ID = val
		case "order_id":
			v.OrderID = val
		case "sku":
			v.SKU = val
		case "qty":
			n, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("%w: qty: %v", ErrInvalid, err)
			}
			v.Qty = n
		case "reason":
			v.Reason = val
		case "inspector":
			v.Inspector = val
		case "status":
			st, err := ParseReturnRequestStatus(val)
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
			return nil, fmt.Errorf("%w: unknown return request field %q", ErrInvalid, key)
		}
	}
	if v.ID == "" {
		return nil, fmt.Errorf("%w: return request record without id", ErrInvalid)
	}
	return v, nil
}
