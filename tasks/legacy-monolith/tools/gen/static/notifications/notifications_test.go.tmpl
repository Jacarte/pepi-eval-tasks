package notifications

import (
	"context"
	"errors"
	"strings"
	"testing"

	"shopcore/ids"
)

type fakeSender struct {
	sent []string
	err  error
}

func (f *fakeSender) Send(_ context.Context, channel, recipient, subject, body string) error {
	f.sent = append(f.sent, channel+":"+recipient+":"+body)
	return f.err
}

func TestRender(t *testing.T) {
	_, subj, body, err := Render(TemplateRefundIssued, map[string]string{"name": "Ada", "ref": "rfd-1"})
	if err != nil {
		t.Fatal(err)
	}
	if subj == "" || !strings.Contains(body, "Ada") || !strings.Contains(body, "rfd-1") {
		t.Fatalf("subject=%q body=%q", subj, body)
	}
	if _, _, _, err := Render(TemplateRefundIssued, map[string]string{"name": "Ada"}); !errors.Is(err, ErrMissingVar) {
		t.Fatalf("err = %v", err)
	}
	if _, _, _, err := Render("nope", nil); !errors.Is(err, ErrUnknownTemplate) {
		t.Fatalf("err = %v", err)
	}
	if len(TemplateIDs()) != len(templates) {
		t.Fatal("duplicate template ids")
	}
}

func TestDispatcher(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryMessageStore()
	snd := &fakeSender{}
	d := NewDispatcher(store, snd, nil, ids.NewSequence())
	vars := map[string]string{"name": "Bo", "ref": "ord-1"}
	m, err := d.Notify(ctx, "bo@example.com", TemplateOrderShipped, vars)
	if err != nil || m.Status != MessageStatusSent || len(snd.sent) != 1 {
		t.Fatalf("notify = %+v, %v", m, err)
	}
	d.Suppress("BO@example.com")
	if _, err := d.Notify(ctx, "bo@example.com", TemplateOrderShipped, vars); !errors.Is(err, ErrSuppressed) {
		t.Fatalf("err = %v", err)
	}
	snd.err = errors.New("vendor down")
	m, err = d.Notify(ctx, "cy@example.com", TemplateOrderShipped, vars)
	if err == nil || m.Status != MessageStatusFailed {
		t.Fatalf("failed send = %+v, %v", m, err)
	}
	if store.Len() != 3 {
		t.Fatalf("stored = %d", store.Len())
	}
}
