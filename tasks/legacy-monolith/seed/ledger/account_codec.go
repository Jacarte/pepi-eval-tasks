package ledger

import (
	"fmt"
	"strconv"
	"strings"

	"shopcore/money"
)

// EncodeAccount renders v in the key=value wire format used by the nightly
// export and the v1 admin screens. Field order is stable.
func EncodeAccount(v *Account) string {
	var b strings.Builder
	appendField(&b, "id", v.ID)
	appendField(&b, "code", v.Code)
	appendField(&b, "name", v.Name)
	appendField(&b, "kind", v.Kind)
	appendField(&b, "balance", strconv.FormatInt(int64(v.Balance), 10))
	appendField(&b, "status", v.Status.String())
	appendField(&b, "created_at", timeString(v.CreatedAt))
	appendField(&b, "updated_at", timeString(v.UpdatedAt))
	return b.String()
}

// DecodeAccount parses the output of EncodeAccount.
func DecodeAccount(line string) (*Account, error) {
	v := &Account{}
	for _, part := range splitOn(line, ';') {
		if part == "" {
			continue
		}
		key, raw, ok := cutOn(part, '=')
		if !ok {
			return nil, fmt.Errorf("%w: malformed account field %q", ErrInvalid, part)
		}
		val := unquoteValue(raw)
		switch unquoteValue(key) {
		case "id":
			v.ID = val
		case "code":
			v.Code = val
		case "name":
			v.Name = val
		case "kind":
			v.Kind = val
		case "balance":
			n, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("%w: balance: %v", ErrInvalid, err)
			}
			v.Balance = money.Cents(n)
		case "status":
			st, err := ParseAccountStatus(val)
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
			return nil, fmt.Errorf("%w: unknown account field %q", ErrInvalid, key)
		}
	}
	if v.ID == "" {
		return nil, fmt.Errorf("%w: account record without id", ErrInvalid)
	}
	return v, nil
}
