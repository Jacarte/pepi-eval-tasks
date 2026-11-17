package shipping

import (
	"fmt"
	"strconv"
	"strings"
)

// EncodeShipment renders v in the key=value wire format used by the nightly
// export and the v1 admin screens. Field order is stable.
func EncodeShipment(v *Shipment) string {
	var b strings.Builder
	writeKV(&b, "id", v.ID)
	writeKV(&b, "order_id", v.OrderID)
	writeKV(&b, "carrier", v.Carrier)
	writeKV(&b, "tracking_no", v.TrackingNo)
	writeKV(&b, "weight_grams", strconv.Itoa(v.WeightGrams))
	writeKV(&b, "service", v.Service)
	writeKV(&b, "hold_reason", v.HoldReason)
	writeKV(&b, "status", v.Status.String())
	writeKV(&b, "created_at", formatTime(v.CreatedAt))
	writeKV(&b, "updated_at", formatTime(v.UpdatedAt))
	return b.String()
}

// DecodeShipment parses the output of EncodeShipment.
func DecodeShipment(line string) (*Shipment, error) {
	v := &Shipment{}
	for _, part := range splitEscaped(line, ';') {
		if part == "" {
			continue
		}
		key, raw, ok := cutEscaped(part, '=')
		if !ok {
			return nil, fmt.Errorf("%w: malformed shipment field %q", ErrInvalid, part)
		}
		val := unescapeField(raw)
		switch unescapeField(key) {
		case "id":
			v.ID = val
		case "order_id":
			v.OrderID = val
		case "carrier":
			v.Carrier = val
		case "tracking_no":
			v.TrackingNo = val
		case "weight_grams":
			n, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("%w: weight_grams: %v", ErrInvalid, err)
			}
			v.WeightGrams = n
		case "service":
			v.Service = val
		case "hold_reason":
			v.HoldReason = val
		case "status":
			st, err := ParseShipmentStatus(val)
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
			return nil, fmt.Errorf("%w: unknown shipment field %q", ErrInvalid, key)
		}
	}
	if v.ID == "" {
		return nil, fmt.Errorf("%w: shipment record without id", ErrInvalid)
	}
	return v, nil
}
