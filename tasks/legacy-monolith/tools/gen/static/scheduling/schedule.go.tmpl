// Package scheduling runs recurring background jobs and manages delivery
// booking slots.
package scheduling

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"shopcore/clock"
	"shopcore/ctxkeys"
	"shopcore/errs"
)

var (
	ErrBadSpec    = errors.New("scheduling: bad schedule spec")
	ErrUnknownJob = errors.New("scheduling: no function registered for job")
	ErrSlotFull   = errors.New("scheduling: slot is full")
)

// Spec is a parsed schedule: either a fixed interval or a daily wall time.
type Spec struct {
	Every  time.Duration
	Daily  bool
	Hour   int
	Minute int
}

// ParseSpec accepts "every <duration>", "hourly" and "daily HH:MM" (UTC).
func ParseSpec(s string) (Spec, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	switch {
	case s == "hourly":
		return Spec{Every: time.Hour}, nil
	case strings.HasPrefix(s, "every "):
		d, err := time.ParseDuration(strings.TrimPrefix(s, "every "))
		if err != nil || d < time.Minute {
			return Spec{}, fmt.Errorf("%w: %q", ErrBadSpec, s)
		}
		return Spec{Every: d}, nil
	case strings.HasPrefix(s, "daily "):
		hh, mm, ok := strings.Cut(strings.TrimPrefix(s, "daily "), ":")
		h, err1 := strconv.Atoi(hh)
		m, err2 := strconv.Atoi(mm)
		if !ok || err1 != nil || err2 != nil || h < 0 || h > 23 || m < 0 || m > 59 {
			return Spec{}, fmt.Errorf("%w: %q", ErrBadSpec, s)
		}
		return Spec{Daily: true, Hour: h, Minute: m}, nil
	}
	return Spec{}, fmt.Errorf("%w: %q", ErrBadSpec, s)
}

// Next returns the first run time strictly after last. A zero last means the
// job has never run and is due immediately.
func (sp Spec) Next(last time.Time) time.Time {
	if last.IsZero() {
		return last
	}
	if !sp.Daily {
		return last.Add(sp.Every)
	}
	last = last.UTC()
	y, mo, d := last.Date()
	t := time.Date(y, mo, d, sp.Hour, sp.Minute, 0, 0, time.UTC)
	if !t.After(last) {
		t = t.AddDate(0, 0, 1)
	}
	return t
}

// JobFunc is the body of a job.
type JobFunc func(ctx context.Context) error

// Runner executes due jobs. It is driven by an external ticker so tests can
// call RunDue directly.
type Runner struct {
	jobs  JobStore
	clock clock.Clock
	mu    sync.Mutex
	funcs map[string]JobFunc
	seq   int
}

func NewRunner(jobs JobStore, clk clock.Clock) *Runner {
	return &Runner{jobs: jobs, clock: clock.OrSystem(clk), funcs: map[string]JobFunc{}}
}

// Register binds a job name to its function.
func (r *Runner) Register(name string, fn JobFunc) {
	r.mu.Lock()
	r.funcs[name] = fn
	r.mu.Unlock()
}

// RunDue runs every idle job whose next run time has passed and returns the
// names it ran, in name order. Each execution gets its own request id and the
// "scheduler" actor, like an inbound HTTP request would.
func (r *Runner) RunDue(ctx context.Context) ([]string, error) {
	const op = "scheduling.rundue"
	jobs, err := r.jobs.Scan(ctx, JobFilter{Status: JobStatusIdle})
	if err != nil {
		return nil, errs.Wrap(op, err)
	}
	sort.Slice(jobs, func(i, j int) bool { return jobs[i].Name < jobs[j].Name })
	now := r.clock.Now()
	var ran []string
	for _, j := range jobs {
		sp, err := ParseSpec(j.Spec)
		if err != nil {
			return ran, errs.Wrap(op, err)
		}
		if sp.Next(j.LastRunAt).After(now) {
			continue
		}
		r.mu.Lock()
		fn := r.funcs[j.Name]
		r.seq++
		reqID := fmt.Sprintf("job-%s-%d", j.Name, r.seq)
		r.mu.Unlock()
		if fn == nil {
			return ran, errs.Wrap(op, fmt.Errorf("%w: %s", ErrUnknownJob, j.Name))
		}
		jctx := ctxkeys.WithActor(ctxkeys.WithRequestID(ctx, reqID), "scheduler")
		if err := fn(jctx); err != nil {
			j.Failures++
			if j.Failures >= 5 {
				j.Status = JobStatusPaused // someone has to look at it
			}
		} else {
			j.Failures = 0
		}
		j.LastRunAt = now
		j.UpdatedAt = now
		if err := r.jobs.Store(ctx, j); err != nil {
			return ran, errs.Wrap(op, err)
		}
		ran = append(ran, j.Name)
	}
	return ran, nil
}

// Book reserves one unit of capacity in a delivery slot.
func Book(ctx context.Context, slots SlotStore, id string) (*Slot, error) {
	const op = "scheduling.book"
	sl, err := slots.Load(ctx, id)
	if err != nil {
		return nil, errs.Wrap(op, err)
	}
	if sl.Status != SlotStatusOpen || sl.Booked >= sl.Capacity {
		return nil, errs.Wrap(op, ErrSlotFull)
	}
	sl.Booked++
	if sl.Booked == sl.Capacity {
		sl.Status = SlotStatusFull
	}
	if err := slots.Store(ctx, sl); err != nil {
		return nil, errs.Wrap(op, err)
	}
	return sl, nil
}
