package payments

import (
	"fmt"
	"strconv"
	"strings"

	"shopcore/money"
)

// EncodeSettlement renders v in the key=value wire format used by the nightly
// export and the v1 admin screens. Field order is stable.
func EncodeSettlement(v *Settlement) string {
	var b strings.Builder
	writeKV(&b, "id", v.ID)
	writeKV(&b, "batch", v.Batch)
	writeKV(&b, "amount", strconv.FormatInt(int64(v.Amount), 10))
	writeKV(&b, "processor", v.Processor)
	writeKV(&b, "items", strconv.Itoa(v.Items))
	writeKV(&b, "status", v.Status.String())
	writeKV(&b, "created_at", formatTime(v.CreatedAt))
	writeKV(&b, "updated_at", formatTime(v.UpdatedAt))
	return b.String()
}

// DecodeSettlement parses the output of EncodeSettlement.
// Unknown keys are ignored so newer exports can be read by older jobs.
func DecodeSettlement(line string) (*Settlement, error) {
	v := &Settlement{}
	for _, part := range splitEscaped(line, ';') {
		if part == "" {
			continue
		}
		key, raw, ok := cutEscaped(part, '=')
		if !ok {
			return nil, fmt.Errorf("%w: malformed settlement field %q", ErrInvalid, part)
		}
		val := unescapeField(raw)
		switch unescapeField(key) {
		case "id":
			v.ID = val
		case "batch":
			v.Batch = val
		case "amount":
			n, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("%w: amount: %v", ErrInvalid, err)
			}
			v.Amount = money.Cents(n)
		case "processor":
			v.Processor = val
		case "items":
			n, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("%w: items: %v", ErrInvalid, err)
			}
			v.Items = n
		case "status":
			st, err := ParseSettlementStatus(val)
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
		}
	}
	if v.ID == "" {
		return nil, fmt.Errorf("%w: settlement record without id", ErrInvalid)
	}
	return v, nil
}
