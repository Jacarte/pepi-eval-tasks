package suppliers

import (
	"fmt"
	"strconv"
	"strings"

	"shopcore/money"
)

// EncodePurchaseOrder renders v in the key=value wire format used by the nightly
// export and the v1 admin screens. Field order is stable.
func EncodePurchaseOrder(v *PurchaseOrder) string {
	var b strings.Builder
	writeKV(&b, "id", v.ID)
	writeKV(&b, "supplier_id", v.SupplierID)
	writeKV(&b, "total", strconv.FormatInt(int64(v.Total), 10))
	writeKV(&b, "lines", strconv.Itoa(v.Lines))
	writeKV(&b, "expected_at", formatTime(v.ExpectedAt))
	writeKV(&b, "status", v.Status.String())
	writeKV(&b, "created_at", formatTime(v.CreatedAt))
	writeKV(&b, "updated_at", formatTime(v.UpdatedAt))
	return b.String()
}

// DecodePurchaseOrder parses the output of EncodePurchaseOrder.
func DecodePurchaseOrder(line string) (*PurchaseOrder, error) {
	v := &PurchaseOrder{}
	for _, part := range splitEscaped(line, ';') {
		if part == "" {
			continue
		}
		key, raw, ok := cutEscaped(part, '=')
		if !ok {
			return nil, fmt.Errorf("%w: malformed purchase order field %q", ErrInvalid, part)
		}
		val := unescapeField(raw)
		switch unescapeField(key) {
		case "id":
			v.ID = val
		case "supplier_id":
			v.SupplierID = val
		case "total":
			n, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("%w: total: %v", ErrInvalid, err)
			}
			v.Total = money.Cents(n)
		case "lines":
			n, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("%w: lines: %v", ErrInvalid, err)
			}
			v.Lines = n
		case "expected_at":
			ts, err := parseTime(val)
			if err != nil {
				return nil, fmt.Errorf("%w: expected_at: %v", ErrInvalid, err)
			}
			v.ExpectedAt = ts
		case "status":
			st, err := ParsePurchaseOrderStatus(val)
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
			return nil, fmt.Errorf("%w: unknown purchase order field %q", ErrInvalid, key)
		}
	}
	if v.ID == "" {
		return nil, fmt.Errorf("%w: purchase order record without id", ErrInvalid)
	}
	return v, nil
}
