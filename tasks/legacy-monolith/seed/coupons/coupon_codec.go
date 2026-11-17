package coupons

import (
	"fmt"
	"strconv"
	"strings"
)

// EncodeCoupon renders v in the key=value wire format used by the nightly
// export and the v1 admin screens. Field order is stable.
func EncodeCoupon(v *Coupon) string {
	var b strings.Builder
	putPair(&b, "id", v.ID)
	putPair(&b, "code", v.Code)
	putPair(&b, "percent_off", strconv.Itoa(v.PercentOff))
	putPair(&b, "max_uses", strconv.Itoa(v.MaxUses))
	putPair(&b, "uses", strconv.Itoa(v.Uses))
	putPair(&b, "expires_at", fmtTime(v.ExpiresAt))
	putPair(&b, "status", v.Status.String())
	putPair(&b, "created_at", fmtTime(v.CreatedAt))
	putPair(&b, "updated_at", fmtTime(v.UpdatedAt))
	return b.String()
}

// DecodeCoupon parses the output of EncodeCoupon.
// Unknown keys are ignored so newer exports can be read by older jobs.
func DecodeCoupon(line string) (*Coupon, error) {
	v := &Coupon{}
	for _, part := range splitUnescaped(line, ';') {
		if part == "" {
			continue
		}
		key, raw, ok := cutUnescaped(part, '=')
		if !ok {
			return nil, fmt.Errorf("%w: malformed coupon field %q", ErrInvalid, part)
		}
		val := unesc(raw)
		switch unesc(key) {
		case "id":
			v.ID = val
		case "code":
			v.Code = val
		case "percent_off":
			n, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("%w: percent_off: %v", ErrInvalid, err)
			}
			v.PercentOff = n
		case "max_uses":
			n, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("%w: max_uses: %v", ErrInvalid, err)
			}
			v.MaxUses = n
		case "uses":
			n, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("%w: uses: %v", ErrInvalid, err)
			}
			v.Uses = n
		case "expires_at":
			ts, err := parseTimeField(val)
			if err != nil {
				return nil, fmt.Errorf("%w: expires_at: %v", ErrInvalid, err)
			}
			v.ExpiresAt = ts
		case "status":
			st, err := ParseCouponStatus(val)
			if err != nil {
				return nil, err
			}
			v.Status = st
		case "created_at":
			ts, err := parseTimeField(val)
			if err != nil {
				return nil, fmt.Errorf("%w: created_at: %v", ErrInvalid, err)
			}
			v.CreatedAt = ts
		case "updated_at":
			ts, err := parseTimeField(val)
			if err != nil {
				return nil, fmt.Errorf("%w: updated_at: %v", ErrInvalid, err)
			}
			v.UpdatedAt = ts
		}
	}
	if v.ID == "" {
		return nil, fmt.Errorf("%w: coupon record without id", ErrInvalid)
	}
	return v, nil
}
