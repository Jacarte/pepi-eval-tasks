package catalog

import (
	"fmt"
	"strconv"
	"strings"

	"shopcore/money"
)

// EncodeProductListing renders v in the key=value wire format used by the nightly
// export and the v1 admin screens. Field order is stable.
func EncodeProductListing(v *ProductListing) string {
	var b strings.Builder
	appendField(&b, "id", v.ID)
	appendField(&b, "sku", v.SKU)
	appendField(&b, "title", v.Title)
	appendField(&b, "brand", v.Brand)
	appendField(&b, "price", strconv.FormatInt(int64(v.Price), 10))
	parts := make([]string, 0, len(v.Tags))
	for _, tag := range v.Tags {
		parts = append(parts, quoteValue(tag))
	}
	appendRawField(&b, "tags", strings.Join(parts, ","))
	appendField(&b, "featured", strconv.FormatBool(v.Featured))
	appendField(&b, "status", v.Status.String())
	appendField(&b, "created_at", timeString(v.CreatedAt))
	appendField(&b, "updated_at", timeString(v.UpdatedAt))
	return b.String()
}

// DecodeProductListing parses the output of EncodeProductListing.
// Unknown keys are ignored so newer exports can be read by older jobs.
func DecodeProductListing(line string) (*ProductListing, error) {
	v := &ProductListing{}
	for _, part := range splitOn(line, ';') {
		if part == "" {
			continue
		}
		key, raw, ok := cutOn(part, '=')
		if !ok {
			return nil, fmt.Errorf("%w: malformed product listing field %q", ErrInvalid, part)
		}
		val := unquoteValue(raw)
		switch unquoteValue(key) {
		case "id":
			v.ID = val
		case "sku":
			v.SKU = val
		case "title":
			v.Title = val
		case "brand":
			v.Brand = val
		case "price":
			n, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("%w: price: %v", ErrInvalid, err)
			}
			v.Price = money.Cents(n)
		case "tags":
			v.Tags = nil
			if raw != "" {
				for _, tag := range splitOn(raw, ',') {
					v.Tags = append(v.Tags, unquoteValue(tag))
				}
			}
		case "featured":
			flag, err := strconv.ParseBool(val)
			if err != nil {
				return nil, fmt.Errorf("%w: featured: %v", ErrInvalid, err)
			}
			v.Featured = flag
		case "status":
			st, err := ParseProductListingStatus(val)
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
		return nil, fmt.Errorf("%w: product listing record without id", ErrInvalid)
	}
	return v, nil
}
