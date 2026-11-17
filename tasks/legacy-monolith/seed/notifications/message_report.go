package notifications

import (
	"encoding/csv"
	"fmt"
	"io"
	"sort"
	"strconv"
	"time"
)

// MessageSummary aggregates a batch of messages for the reports package.
type MessageSummary struct {
	Count         int
	ByStatus      map[MessageStatus]int
	TotalAttempts int64
	Earliest      time.Time
	Latest        time.Time
}

// SummarizeMessages folds items into a MessageSummary. Nil entries are skipped.
func SummarizeMessages(items []*Message) MessageSummary {
	sum := MessageSummary{ByStatus: make(map[MessageStatus]int)}
	for _, v := range items {
		if v == nil {
			continue
		}
		sum.Count++
		sum.ByStatus[v.Status]++
		sum.TotalAttempts += int64(v.Attempts)
		if sum.Earliest.IsZero() || v.CreatedAt.Before(sum.Earliest) {
			sum.Earliest = v.CreatedAt
		}
		if v.CreatedAt.After(sum.Latest) {
			sum.Latest = v.CreatedAt
		}
	}
	return sum
}

// TopMessagesByAttempts returns the n items with the largest attempts, largest first.
// Ties keep their input order.
func TopMessagesByAttempts(items []*Message, n int) []*Message {
	out := append([]*Message(nil), items...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Attempts > out[j].Attempts })
	if n >= 0 && len(out) > n {
		out = out[:n]
	}
	return out
}

// GroupMessagesByRecipient buckets items by recipient.
func GroupMessagesByRecipient(items []*Message) map[string][]*Message {
	out := make(map[string][]*Message)
	for _, v := range items {
		out[v.Recipient] = append(out[v.Recipient], v)
	}
	return out
}

// MessagesCreatedBetween keeps items created in [from, to).
func MessagesCreatedBetween(items []*Message, from, to time.Time) []*Message {
	var out []*Message
	for _, v := range items {
		if !v.CreatedAt.Before(from) && v.CreatedAt.Before(to) {
			out = append(out, v)
		}
	}
	return out
}

// WriteMessagesCSV writes items as CSV with a header row.
func WriteMessagesCSV(out io.Writer, items []*Message) error {
	cw := csv.NewWriter(out)
	if err := cw.Write([]string{"id", "recipient", "template", "channel", "attempts", "status", "created_at"}); err != nil {
		return err
	}
	for _, v := range items {
		rec := []string{
			v.ID,
			v.Recipient,
			v.Template,
			v.Channel,
			strconv.Itoa(v.Attempts),
			v.Status.String(),
			fmtTime(v.CreatedAt),
		}
		if err := cw.Write(rec); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// DiffMessage lists the fields that differ between a and b.
func DiffMessage(a, b *Message) []FieldChange {
	var out []FieldChange
	if a.Recipient != b.Recipient {
		out = append(out, FieldChange{"recipient", fmt.Sprint(a.Recipient), fmt.Sprint(b.Recipient)})
	}
	if a.Template != b.Template {
		out = append(out, FieldChange{"template", fmt.Sprint(a.Template), fmt.Sprint(b.Template)})
	}
	if a.Channel != b.Channel {
		out = append(out, FieldChange{"channel", fmt.Sprint(a.Channel), fmt.Sprint(b.Channel)})
	}
	if a.Attempts != b.Attempts {
		out = append(out, FieldChange{"attempts", fmt.Sprint(a.Attempts), fmt.Sprint(b.Attempts)})
	}
	if a.Status != b.Status {
		out = append(out, FieldChange{"status", a.Status.String(), b.Status.String()})
	}
	return out
}
