package scheduling

import (
	"fmt"
	"strconv"
	"strings"
)

// EncodeJob renders v in the key=value wire format used by the nightly
// export and the v1 admin screens. Field order is stable.
func EncodeJob(v *Job) string {
	var b strings.Builder
	appendField(&b, "id", v.ID)
	appendField(&b, "name", v.Name)
	appendField(&b, "spec", v.Spec)
	appendField(&b, "owner", v.Owner)
	appendField(&b, "last_run_at", timeString(v.LastRunAt))
	appendField(&b, "failures", strconv.Itoa(v.Failures))
	appendField(&b, "status", v.Status.String())
	appendField(&b, "created_at", timeString(v.CreatedAt))
	appendField(&b, "updated_at", timeString(v.UpdatedAt))
	return b.String()
}

// DecodeJob parses the output of EncodeJob.
func DecodeJob(line string) (*Job, error) {
	v := &Job{}
	for _, part := range splitOn(line, ';') {
		if part == "" {
			continue
		}
		key, raw, ok := cutOn(part, '=')
		if !ok {
			return nil, fmt.Errorf("%w: malformed job field %q", ErrInvalid, part)
		}
		val := unquoteValue(raw)
		switch unquoteValue(key) {
		case "id":
			v.ID = val
		case "name":
			v.Name = val
		case "spec":
			v.Spec = val
		case "owner":
			v.Owner = val
		case "last_run_at":
			ts, err := timeValue(val)
			if err != nil {
				return nil, fmt.Errorf("%w: last_run_at: %v", ErrInvalid, err)
			}
			v.LastRunAt = ts
		case "failures":
			n, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("%w: failures: %v", ErrInvalid, err)
			}
			v.Failures = n
		case "status":
			st, err := ParseJobStatus(val)
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
			return nil, fmt.Errorf("%w: unknown job field %q", ErrInvalid, key)
		}
	}
	if v.ID == "" {
		return nil, fmt.Errorf("%w: job record without id", ErrInvalid)
	}
	return v, nil
}
