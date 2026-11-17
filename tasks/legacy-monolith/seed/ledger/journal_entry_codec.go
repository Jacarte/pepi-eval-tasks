package ledger

import (
	"fmt"
	"strconv"
	"strings"

	"shopcore/money"
)

// EncodeJournalEntry renders v in the key=value wire format used by the nightly
// export and the v1 admin screens. Field order is stable.
func EncodeJournalEntry(v *JournalEntry) string {
	var b strings.Builder
	appendField(&b, "id", v.ID)
	appendField(&b, "account", v.Account)
	appendField(&b, "debit", strconv.FormatInt(int64(v.Debit), 10))
	appendField(&b, "credit", strconv.FormatInt(int64(v.Credit), 10))
	appendField(&b, "memo", v.Memo)
	appendField(&b, "reference", v.Reference)
	appendField(&b, "status", v.Status.String())
	appendField(&b, "created_at", timeString(v.CreatedAt))
	appendField(&b, "updated_at", timeString(v.UpdatedAt))
	return b.String()
}

// DecodeJournalEntry parses the output of EncodeJournalEntry.
func DecodeJournalEntry(line string) (*JournalEntry, error) {
	v := &JournalEntry{}
	for _, part := range splitOn(line, ';') {
		if part == "" {
			continue
		}
		key, raw, ok := cutOn(part, '=')
		if !ok {
			return nil, fmt.Errorf("%w: malformed journal entry field %q", ErrInvalid, part)
		}
		val := unquoteValue(raw)
		switch unquoteValue(key) {
		case "id":
			v.ID = val
		case "account":
			v.Account = val
		case "debit":
			n, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("%w: debit: %v", ErrInvalid, err)
			}
			v.Debit = money.Cents(n)
		case "credit":
			n, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("%w: credit: %v", ErrInvalid, err)
			}
			v.Credit = money.Cents(n)
		case "memo":
			v.Memo = val
		case "reference":
			v.Reference = val
		case "status":
			st, err := ParseJournalEntryStatus(val)
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
			return nil, fmt.Errorf("%w: unknown journal entry field %q", ErrInvalid, key)
		}
	}
	if v.ID == "" {
		return nil, fmt.Errorf("%w: journal entry record without id", ErrInvalid)
	}
	return v, nil
}
