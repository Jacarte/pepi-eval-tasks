package notifications

import (
	"fmt"
	"strings"
	"time"
)

// Message is an outbound customer notification.
type Message struct {
	ID        string
	Recipient string
	Template  string
	Channel   string // email, sms or push
	Attempts  int
	Status    MessageStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

// MessageStatus is the lifecycle state of a message.
type MessageStatus int

const (
	MessageStatusUnknown MessageStatus = iota
	MessageStatusQueued
	MessageStatusSent
	MessageStatusFailed
	MessageStatusSuppressed
)

var messageStatusNames = []string{"unknown", "queued", "sent", "failed", "suppressed"}

func (s MessageStatus) String() string {
	if s < 0 || int(s) >= len(messageStatusNames) {
		return fmt.Sprintf("MessageStatus(%d)", int(s))
	}
	return messageStatusNames[s]
}

// ParseMessageStatus parses the wire name of a status. Matching is
// case-insensitive because the old importer wrote upper-case values.
func ParseMessageStatus(v string) (MessageStatus, error) {
	v = strings.ToLower(strings.TrimSpace(v))
	for i, n := range messageStatusNames {
		if i > 0 && n == v {
			return MessageStatus(i), nil
		}
	}
	return MessageStatusUnknown, fmt.Errorf("%w: unknown message status %q", ErrInvalid, v)
}

var messageTransitions = map[MessageStatus][]MessageStatus{
	MessageStatusQueued: {MessageStatusSent, MessageStatusSuppressed},
	MessageStatusSent:   {MessageStatusFailed, MessageStatusSuppressed},
	MessageStatusFailed: {MessageStatusSuppressed},
}

// CanTransition reports whether a message may move from s to next.
func (s MessageStatus) CanTransition(next MessageStatus) bool {
	for _, t := range messageTransitions[s] {
		if t == next {
			return true
		}
	}
	return false
}

// Terminal reports whether no further transitions are possible.
func (s MessageStatus) Terminal() bool { return len(messageTransitions[s]) == 0 }

// Validate checks the invariants every stored message must satisfy.
func (v *Message) Validate() error {
	if strings.TrimSpace(v.ID) == "" {
		return fmt.Errorf("%w: message.id is required", ErrInvalid)
	}
	if strings.TrimSpace(v.Recipient) == "" {
		return fmt.Errorf("%w: message.recipient is required", ErrInvalid)
	}
	if strings.TrimSpace(v.Template) == "" {
		return fmt.Errorf("%w: message.template is required", ErrInvalid)
	}
	if v.Attempts < 0 {
		return fmt.Errorf("%w: message.attempts must not be negative", ErrInvalid)
	}
	if v.Status <= MessageStatusUnknown || int(v.Status) >= len(messageStatusNames) {
		return fmt.Errorf("%w: message.status is not a known status", ErrInvalid)
	}
	return nil
}

// Clone returns a deep copy so callers can mutate the result freely.
func (v *Message) Clone() *Message {
	if v == nil {
		return nil
	}
	cp := *v
	return &cp
}
