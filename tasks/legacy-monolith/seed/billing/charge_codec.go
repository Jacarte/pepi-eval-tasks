package billing

import (
	"fmt"
	"strconv"
	"strings"

	"shopcore/money"
)

// EncodeCharge renders v in the key=value wire format used by the nightly
// export and the v1 admin screens. Field order is stable.
func EncodeCharge(v *Charge) string {
	var b strings.Builder
	appendField(&b, "id", v.ID)
	appendField(&b, "invoice_id", v.InvoiceID)
	appendField(&b, "amount", strconv.FormatInt(int64(v.Amount), 10))
	appendField(&b, "captured", strconv.FormatInt(int64(v.Captured), 10))
	appendField(&b, "processor", v.Processor)
	appendField(&b, "status", v.Status.String())
	appendField(&b, "created_at", timeString(v.CreatedAt))
	appendField(&b, "updated_at", timeString(v.UpdatedAt))
	return b.String()
}

// DecodeCharge parses the output of EncodeCharge.
// Unknown keys are ignored so newer exports can be read by older jobs.
func DecodeCharge(line string) (*Charge, error) {
	v := &Charge{}
	for _, part := range splitOn(line, ';') {
		if part == "" {
			continue
		}
		key, raw, ok := cutOn(part, '=')
		if !ok {
			return nil, fmt.Errorf("%w: malformed charge field %q", ErrInvalid, part)
		}
		val := unquoteValue(raw)
		switch unquoteValue(key) {
		case "id":
			v.ID = val
		case "invoice_id":
			v.InvoiceID = val
		case "amount":
			n, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("%w: amount: %v", ErrInvalid, err)
			}
			v.Amount = money.Cents(n)
		case "captured":
			n, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("%w: captured: %v", ErrInvalid, err)
			}
			v.Captured = money.Cents(n)
		case "processor":
			v.Processor = val
		case "status":
			st, err := ParseChargeStatus(val)
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
		return nil, fmt.Errorf("%w: charge record without id", ErrInvalid)
	}
	return v, nil
}
