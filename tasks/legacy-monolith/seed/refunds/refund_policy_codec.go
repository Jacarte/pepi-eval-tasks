package refunds

import (
	"fmt"
	"strconv"
	"strings"
)

// EncodeRefundPolicy renders v in the key=value wire format used by the nightly
// export and the v1 admin screens. Field order is stable.
func EncodeRefundPolicy(v *RefundPolicy) string {
	var b strings.Builder
	appendField(&b, "id", v.ID)
	appendField(&b, "name", v.Name)
	appendField(&b, "window_days", strconv.Itoa(v.WindowDays))
	appendField(&b, "restock_fee_pct", strconv.Itoa(v.RestockFeePct))
	appendField(&b, "requires_receipt", strconv.FormatBool(v.RequiresReceipt))
	appendField(&b, "status", v.Status.String())
	appendField(&b, "created_at", timeString(v.CreatedAt))
	appendField(&b, "updated_at", timeString(v.UpdatedAt))
	return b.String()
}

// DecodeRefundPolicy parses the output of EncodeRefundPolicy.
func DecodeRefundPolicy(line string) (*RefundPolicy, error) {
	v := &RefundPolicy{}
	for _, part := range splitOn(line, ';') {
		if part == "" {
			continue
		}
		key, raw, ok := cutOn(part, '=')
		if !ok {
			return nil, fmt.Errorf("%w: malformed refund policy field %q", ErrInvalid, part)
		}
		val := unquoteValue(raw)
		switch unquoteValue(key) {
		case "id":
			v.ID = val
		case "name":
			v.Name = val
		case "window_days":
			n, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("%w: window_days: %v", ErrInvalid, err)
			}
			v.WindowDays = n
		case "restock_fee_pct":
			n, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("%w: restock_fee_pct: %v", ErrInvalid, err)
			}
			v.RestockFeePct = n
		case "requires_receipt":
			flag, err := strconv.ParseBool(val)
			if err != nil {
				return nil, fmt.Errorf("%w: requires_receipt: %v", ErrInvalid, err)
			}
			v.RequiresReceipt = flag
		case "status":
			st, err := ParseRefundPolicyStatus(val)
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
			return nil, fmt.Errorf("%w: unknown refund policy field %q", ErrInvalid, key)
		}
	}
	if v.ID == "" {
		return nil, fmt.Errorf("%w: refund policy record without id", ErrInvalid)
	}
	return v, nil
}
