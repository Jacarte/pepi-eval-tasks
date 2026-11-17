package billing

import (
	"fmt"
	"strconv"
	"strings"

	"shopcore/money"
)

// EncodeInvoice renders v in the key=value wire format used by the nightly
// export and the v1 admin screens. Field order is stable.
func EncodeInvoice(v *Invoice) string {
	var b strings.Builder
	appendField(&b, "id", v.ID)
	appendField(&b, "customer_id", v.CustomerID)
	appendField(&b, "number", v.Number)
	appendField(&b, "amount_due", strconv.FormatInt(int64(v.AmountDue), 10))
	appendField(&b, "currency", v.Currency)
	appendField(&b, "due_at", timeString(v.DueAt))
	appendField(&b, "lines", strconv.Itoa(v.Lines))
	appendField(&b, "status", v.Status.String())
	appendField(&b, "created_at", timeString(v.CreatedAt))
	appendField(&b, "updated_at", timeString(v.UpdatedAt))
	return b.String()
}

// DecodeInvoice parses the output of EncodeInvoice.
// Unknown keys are ignored so newer exports can be read by older jobs.
func DecodeInvoice(line string) (*Invoice, error) {
	v := &Invoice{}
	for _, part := range splitOn(line, ';') {
		if part == "" {
			continue
		}
		key, raw, ok := cutOn(part, '=')
		if !ok {
			return nil, fmt.Errorf("%w: malformed invoice field %q", ErrInvalid, part)
		}
		val := unquoteValue(raw)
		switch unquoteValue(key) {
		case "id":
			v.ID = val
		case "customer_id":
			v.CustomerID = val
		case "number":
			v.Number = val
		case "amount_due":
			n, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("%w: amount_due: %v", ErrInvalid, err)
			}
			v.AmountDue = money.Cents(n)
		case "currency":
			v.Currency = val
		case "due_at":
			ts, err := timeValue(val)
			if err != nil {
				return nil, fmt.Errorf("%w: due_at: %v", ErrInvalid, err)
			}
			v.DueAt = ts
		case "lines":
			n, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("%w: lines: %v", ErrInvalid, err)
			}
			v.Lines = n
		case "status":
			st, err := ParseInvoiceStatus(val)
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
		return nil, fmt.Errorf("%w: invoice record without id", ErrInvalid)
	}
	return v, nil
}
