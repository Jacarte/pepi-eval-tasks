package notifications

import (
	"fmt"
	"strconv"
	"strings"
)

// EncodeMessage renders v in the key=value wire format used by the nightly
// export and the v1 admin screens. Field order is stable.
func EncodeMessage(v *Message) string {
	var b strings.Builder
	putPair(&b, "id", v.ID)
	putPair(&b, "recipient", v.Recipient)
	putPair(&b, "template", v.Template)
	putPair(&b, "channel", v.Channel)
	putPair(&b, "attempts", strconv.Itoa(v.Attempts))
	putPair(&b, "status", v.Status.String())
	putPair(&b, "created_at", fmtTime(v.CreatedAt))
	putPair(&b, "updated_at", fmtTime(v.UpdatedAt))
	return b.String()
}

// DecodeMessage parses the output of EncodeMessage.
// Unknown keys are ignored so newer exports can be read by older jobs.
func DecodeMessage(line string) (*Message, error) {
	v := &Message{}
	for _, part := range splitUnescaped(line, ';') {
		if part == "" {
			continue
		}
		key, raw, ok := cutUnescaped(part, '=')
		if !ok {
			return nil, fmt.Errorf("%w: malformed message field %q", ErrInvalid, part)
		}
		val := unesc(raw)
		switch unesc(key) {
		case "id":
			v.ID = val
		case "recipient":
			v.Recipient = val
		case "template":
			v.Template = val
		case "channel":
			v.Channel = val
		case "attempts":
			n, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("%w: attempts: %v", ErrInvalid, err)
			}
			v.Attempts = n
		case "status":
			st, err := ParseMessageStatus(val)
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
		return nil, fmt.Errorf("%w: message record without id", ErrInvalid)
	}
	return v, nil
}
