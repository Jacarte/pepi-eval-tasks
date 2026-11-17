// Package notifications renders templates and hands messages to the
// delivery vendor.
package notifications

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"

	"shopcore/clock"
	"shopcore/errs"
	"shopcore/ids"
)

type tmpl struct {
	ID      string
	Channel string
	Subject string
	Body    string
}

var (
	ErrUnknownTemplate = errors.New("notifications: unknown template")
	ErrSuppressed      = errors.New("notifications: recipient is suppressed")
	ErrMissingVar      = errors.New("notifications: template variable not provided")
)

var (
	byIDOnce sync.Once
	byID     map[string]tmpl
)

func lookup(id string) (tmpl, bool) {
	byIDOnce.Do(func() {
		byID = make(map[string]tmpl, len(templates))
		for _, t := range templates {
			byID[t.ID] = t
		}
	})
	t, ok := byID[id]
	return t, ok
}

// TemplateIDs lists the known template ids, sorted.
func TemplateIDs() []string {
	out := make([]string, 0, len(templates))
	for _, t := range templates {
		out = append(out, t.ID)
	}
	sort.Strings(out)
	return out
}

// Render substitutes {{name}} placeholders. Every placeholder must be bound.
func Render(id string, vars map[string]string) (channel, subject, body string, err error) {
	t, ok := lookup(id)
	if !ok {
		return "", "", "", ErrUnknownTemplate
	}
	subject, err = expand(t.Subject, vars)
	if err != nil {
		return "", "", "", err
	}
	body, err = expand(t.Body, vars)
	if err != nil {
		return "", "", "", err
	}
	return t.Channel, subject, body, nil
}

func expand(s string, vars map[string]string) (string, error) {
	var b strings.Builder
	for {
		i := strings.Index(s, "{{")
		if i < 0 {
			b.WriteString(s)
			return b.String(), nil
		}
		j := strings.Index(s[i:], "}}")
		if j < 0 {
			b.WriteString(s)
			return b.String(), nil
		}
		name := s[i+2 : i+j]
		v, ok := vars[name]
		if !ok {
			return "", errors.Join(ErrMissingVar, errors.New(name))
		}
		b.WriteString(s[:i])
		b.WriteString(v)
		s = s[i+j+2:]
	}
}

// Sender delivers a rendered message.
type Sender interface {
	Send(ctx context.Context, channel, recipient, subject, body string) error
}

// Dispatcher renders, stores and sends notifications.
type Dispatcher struct {
	store      MessageStore
	sender     Sender
	clock      clock.Clock
	ids        ids.Generator
	mu         sync.Mutex
	suppressed map[string]bool
}

func NewDispatcher(store MessageStore, sender Sender, clk clock.Clock, gen ids.Generator) *Dispatcher {
	return &Dispatcher{store: store, sender: sender, clock: clock.OrSystem(clk), ids: ids.OrRandom(gen), suppressed: map[string]bool{}}
}

// Suppress stops all future messages to recipient (unsubscribe, bounce).
func (d *Dispatcher) Suppress(recipient string) {
	d.mu.Lock()
	d.suppressed[strings.ToLower(recipient)] = true
	d.mu.Unlock()
}

// Notify renders templateID for recipient and sends it. Delivery failures are
// stored on the message and returned.
func (d *Dispatcher) Notify(ctx context.Context, recipient, templateID string, vars map[string]string) (*Message, error) {
	const op = "notifications.notify"
	channel, subject, body, err := Render(templateID, vars)
	if err != nil {
		return nil, errs.Wrap(op, err)
	}
	now := d.clock.Now()
	m := &Message{ID: d.ids.Next("msg"), Recipient: recipient, Template: templateID, Channel: channel, Status: MessageStatusQueued, CreatedAt: now, UpdatedAt: now}
	d.mu.Lock()
	sup := d.suppressed[strings.ToLower(recipient)]
	d.mu.Unlock()
	if sup {
		m.Status = MessageStatusSuppressed
		if err := d.store.Store(ctx, m); err != nil {
			return nil, errs.Wrap(op, err)
		}
		return m, errs.Wrap(op, ErrSuppressed)
	}
	m.Attempts++
	sendErr := d.sender.Send(ctx, channel, recipient, subject, body)
	if sendErr != nil {
		m.Status = MessageStatusFailed
	} else {
		m.Status = MessageStatusSent
	}
	if err := d.store.Store(ctx, m); err != nil {
		return nil, errs.Wrap(op, err)
	}
	return m, errs.Wrap(op, sendErr)
}
