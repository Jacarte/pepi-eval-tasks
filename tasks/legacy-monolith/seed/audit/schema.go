package audit

import (
	"errors"
	"fmt"
	"strings"
)

// Validation errors returned by Validate.
var (
	ErrBadName          = errors.New("audit: event name does not follow the schema")
	ErrMissingRequestID = errors.New("audit: request id is required")
	ErrMissingSubject   = errors.New("audit: subject is required")
	ErrBadAttrKey       = errors.New("audit: attribute keys must be snake_case")
)

// irregularPast lists the past-tense verbs accepted by ValidateName that do
// not end in "ed".
var irregularPast = map[string]bool{"held": true, "sent": true, "paid": true, "won": true, "lost": true, "written": true, "rebuilt": true}

// ValidateName checks an event name against the audit_events schema:
// "<subject type>.<past-tense verb>", both parts snake_case, for example
// "shipment.dispatched" or "purchase_order.approved". The importer rejects
// anything else.
func ValidateName(name string) error {
	noun, verb, ok := strings.Cut(name, ".")
	if !ok || !isSnake(noun) || !isSnake(verb) {
		return fmt.Errorf("%w: %q", ErrBadName, name)
	}
	last := verb[strings.LastIndex(verb, "_")+1:]
	if !strings.HasSuffix(last, "ed") && !irregularPast[last] {
		return fmt.Errorf("%w: %q: the verb must be in the past tense", ErrBadName, name)
	}
	return nil
}

// Validate checks everything the importer needs: a schema-conforming name,
// the request id the event is correlated with, the subject and snake_case
// attribute keys.
func Validate(ev Event) error {
	if err := ValidateName(ev.Name); err != nil {
		return err
	}
	if ev.RequestID == "" {
		return ErrMissingRequestID
	}
	if ev.Subject == "" {
		return ErrMissingSubject
	}
	for k := range ev.Attrs {
		if !isSnake(k) {
			return fmt.Errorf("%w: %q", ErrBadAttrKey, k)
		}
	}
	return nil
}

func isSnake(s string) bool {
	if s == "" || s[0] == '_' || s[len(s)-1] == '_' || strings.Contains(s, "__") {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '_' {
			return false
		}
	}
	return s[0] >= 'a' && s[0] <= 'z'
}
