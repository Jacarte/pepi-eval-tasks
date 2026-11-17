package reports

import (
	"fmt"
	"strconv"
	"strings"
)

// EncodeReportRun renders v in the key=value wire format used by the nightly
// export and the v1 admin screens. Field order is stable.
func EncodeReportRun(v *ReportRun) string {
	var b strings.Builder
	appendField(&b, "id", v.ID)
	appendField(&b, "kind", v.Kind)
	appendField(&b, "requested_by", v.RequestedBy)
	appendField(&b, "rows", strconv.Itoa(v.Rows))
	appendField(&b, "format", v.Format)
	appendField(&b, "status", v.Status.String())
	appendField(&b, "created_at", timeString(v.CreatedAt))
	appendField(&b, "updated_at", timeString(v.UpdatedAt))
	return b.String()
}

// DecodeReportRun parses the output of EncodeReportRun.
// Unknown keys are ignored so newer exports can be read by older jobs.
func DecodeReportRun(line string) (*ReportRun, error) {
	v := &ReportRun{}
	for _, part := range splitOn(line, ';') {
		if part == "" {
			continue
		}
		key, raw, ok := cutOn(part, '=')
		if !ok {
			return nil, fmt.Errorf("%w: malformed report run field %q", ErrInvalid, part)
		}
		val := unquoteValue(raw)
		switch unquoteValue(key) {
		case "id":
			v.ID = val
		case "kind":
			v.Kind = val
		case "requested_by":
			v.RequestedBy = val
		case "rows":
			n, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("%w: rows: %v", ErrInvalid, err)
			}
			v.Rows = n
		case "format":
			v.Format = val
		case "status":
			st, err := ParseReportRunStatus(val)
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
		return nil, fmt.Errorf("%w: report run record without id", ErrInvalid)
	}
	return v, nil
}
