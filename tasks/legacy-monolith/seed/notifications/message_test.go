package notifications

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func sampleMessage(n int) *Message {
	return &Message{
		ID:        fmt.Sprintf("message-%03d", n),
		Recipient: fmt.Sprintf("recipient %d;a=b,c\\d", n),
		Template:  fmt.Sprintf("template-%d", n),
		Channel:   fmt.Sprintf("channel %d;a=b,c\\d", n),
		Attempts:  n*24 + 75,
		Status:    MessageStatusQueued,
		CreatedAt: time.Date(2023, time.January, n, 8, 0, 0, 0, time.UTC),
	}
}

func TestMessageCodecRoundTrip(t *testing.T) {
	in := sampleMessage(4)
	line := EncodeMessage(in)
	out, err := DecodeMessage(line)
	if err != nil {
		t.Fatalf("DecodeMessage(%q): %v", line, err)
	}
	if got := EncodeMessage(out); got != line {
		t.Fatalf("round trip mismatch:\n got %s\nwant %s", got, line)
	}
	if out.Recipient != in.Recipient {
		t.Errorf("Recipient = %q, want %q", out.Recipient, in.Recipient)
	}
	if out.Template != in.Template {
		t.Errorf("Template = %q, want %q", out.Template, in.Template)
	}
	if out.Channel != in.Channel {
		t.Errorf("Channel = %q, want %q", out.Channel, in.Channel)
	}
	if _, err := DecodeMessage("id"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("malformed input: err = %v, want ErrInvalid", err)
	}
}

func TestMemoryMessageStore(t *testing.T) {
	ctx := context.Background()
	st := NewMemoryMessageStore()
	for n := 1; n <= 3; n++ {
		if err := st.Store(ctx, sampleMessage(n)); err != nil {
			t.Fatalf("Store(%d): %v", n, err)
		}
	}
	got, err := st.Load(ctx, "message-002")
	if err != nil {
		t.Fatal(err)
	}
	got.Status = MessageStatusSuppressed
	again, _ := st.Load(ctx, "message-002")
	if again.Status != MessageStatusQueued {
		t.Fatal("store handed out a shared pointer")
	}
	if err := st.Store(ctx, got); err != nil {
		t.Fatal(err)
	}
	list, err := st.Scan(ctx, MessageFilter{Status: MessageStatusQueued})
	if err != nil || len(list) != 2 || list[0].ID != "message-001" {
		t.Fatalf("Scan by status = %v, %v", list, err)
	}
	if err := st.Drop(ctx, "message-001"); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Load(ctx, "message-001"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("after delete: err = %v, want ErrNotFound", err)
	}
	if st.Len() != 2 {
		t.Fatalf("Len = %d, want 2", st.Len())
	}
}

func TestSummarizeMessages(t *testing.T) {
	items := []*Message{sampleMessage(1), nil, sampleMessage(2), sampleMessage(3)}
	items[2].Status = MessageStatusSent
	sum := SummarizeMessages(items)
	if sum.Count != 3 || sum.ByStatus[MessageStatusQueued] != 2 || sum.ByStatus[MessageStatusSent] != 1 {
		t.Fatalf("counts = %+v", sum)
	}
	if sum.TotalAttempts != 369 {
		t.Errorf("TotalAttempts = %v, want 369", sum.TotalAttempts)
	}
	if !sum.Earliest.Equal(items[0].CreatedAt) || !sum.Latest.Equal(items[3].CreatedAt) {
		t.Errorf("range = %v..%v", sum.Earliest, sum.Latest)
	}
	if top := TopMessagesByAttempts(items[2:], 1); len(top) != 1 || top[0].ID != "message-003" {
		t.Errorf("top = %v", top)
	}
}
