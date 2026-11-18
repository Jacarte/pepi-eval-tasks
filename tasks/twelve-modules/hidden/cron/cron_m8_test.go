package cron_test

import (
	"errors"
	"testing"
	"time"

	"twelvemods/cron"
)

func m8T(t *testing.T, s string) time.Time {
	t.Helper()
	layout := "2006-01-02 15:04"
	if len(s) > len(layout) {
		layout = "2006-01-02 15:04:05.999999999"
	}
	v, err := time.ParseInLocation(layout, s, time.UTC)
	if err != nil {
		t.Fatalf("bad test time %q: %v", s, err)
	}
	return v
}

func m8Parse(t *testing.T, expr string) *cron.Schedule {
	t.Helper()
	s, err := cron.Parse(expr)
	if err != nil {
		t.Fatalf("Parse(%q): %v", expr, err)
	}
	if s == nil {
		t.Fatalf("Parse(%q) returned nil and no error", expr)
	}
	return s
}

func TestM8Next(t *testing.T) {
	cases := []struct {
		expr, after, want string // want "" means no match within the horizon
	}{
		{"* * * * *", "2024-01-01 10:15:30", "2024-01-01 10:16"},
		{"* * * * *", "2024-01-01 10:15", "2024-01-01 10:16"},
		{"* * * * *", "2024-01-01 10:15:59.999", "2024-01-01 10:16"},
		{"15 10 * * *", "2024-01-01 10:15:30", "2024-01-02 10:15"},
		{"15 10 * * *", "2024-01-01 10:14:59", "2024-01-01 10:15"},
		{"0 0 1 1 *", "2024-06-01 00:00", "2025-01-01 00:00"},
		{"@yearly", "2024-06-01 00:00", "2025-01-01 00:00"},
		{"@annually", "2024-01-01 00:00", "2025-01-01 00:00"},
		{"@hourly", "2024-01-01 10:15", "2024-01-01 11:00"},
		{"@daily", "2024-12-31 23:59", "2025-01-01 00:00"},
		{"@midnight", "2024-02-28 00:00", "2024-02-29 00:00"},
		{"@weekly", "2024-01-01 00:00", "2024-01-07 00:00"},
		{"@monthly", "2024-01-31 12:00", "2024-02-01 00:00"},
		{"  @daily\t", "2024-01-01 00:00", "2024-01-02 00:00"},
		{"*/15 * * * *", "2024-01-01 10:16", "2024-01-01 10:30"},
		{"5/20 * * * *", "2024-01-01 10:46", "2024-01-01 11:05"},
		{"0 9-17/4 * * *", "2024-01-01 13:00", "2024-01-01 17:00"},
		{"0 9-17/4 * * *", "2024-01-01 17:00", "2024-01-02 09:00"},
		{"0,30 * * * *", "2024-01-01 10:29", "2024-01-01 10:30"},
		{"0,30 * * * *", "2024-01-01 10:30", "2024-01-01 11:00"},
		{"1-3,10 0 * * *", "2024-01-01 00:03", "2024-01-01 00:10"},
		{"00 05 * * *", "2024-01-01 00:00", "2024-01-01 05:00"},
		{"0 0 29 2 *", "2024-03-01 00:00", "2028-02-29 00:00"},
		{"0 0 29 2 *", "2096-03-01 00:00", "2104-02-29 00:00"},
		{"0 0 31 * *", "2024-04-01 00:00", "2024-05-31 00:00"},
		{"0 0 30 2 *", "2024-01-01 00:00", ""},
		{"* * 31 4,6,9,11 *", "2024-01-01 00:00", ""},
		{"59 23 31 12 *", "2024-12-31 23:59", "2025-12-31 23:59"},
		{"0 0 1 */3 *", "2024-02-15 00:00", "2024-04-01 00:00"},
		{"0 0 1 FEB/5 *", "2024-03-01 00:00", "2024-07-01 00:00"},
		{"0 0 1 jan-mar *", "2024-03-01 00:00", "2025-01-01 00:00"},
		// 2024-01-01 is a Monday.
		{"0 12 * * MON-FRI", "2024-01-05 12:00", "2024-01-08 12:00"},
		{"0 12 * * 6,0", "2024-01-05 13:00", "2024-01-06 12:00"},
		{"0 12 * * 7", "2024-01-01 00:00", "2024-01-07 12:00"},
		{"0 0 * * sat", "2024-01-01 00:00", "2024-01-06 00:00"},
		{"0 0 * * 5-7", "2024-01-01 00:00", "2024-01-05 00:00"},
		{"0 0 * * 5-7", "2024-01-07 00:00", "2024-01-12 00:00"},
		{"0 0 * * 0-7/2", "2024-01-01 00:00", "2024-01-02 00:00"},
		{"0 0 * * */2", "2024-01-02 00:00", "2024-01-04 00:00"},
		{"0 0 * * */7", "2024-01-01 00:00", "2024-01-07 00:00"},
		{"0 0 * * 1/3", "2024-01-01 00:00", "2024-01-04 00:00"},
		{"0 0 * * 1/3", "2024-01-04 00:00", "2024-01-07 00:00"},
		{"30 2 * jan,Jul sun", "2024-01-28 03:00", "2024-07-07 02:30"},
		// day-of-month / day-of-week OR semantics
		{"0 0 13 * FRI", "2024-01-01 00:00", "2024-01-05 00:00"},
		{"0 0 13 * 5", "2024-01-06 00:00", "2024-01-12 00:00"},
		{"0 0 13 * 5", "2024-01-12 00:00", "2024-01-13 00:00"},
		{"0 0 13 * ?", "2024-01-01 00:00", "2024-01-13 00:00"},
		{"0 0 ? * 5", "2024-01-01 00:00", "2024-01-05 00:00"},
		{"0 0 ? * ?", "2024-01-01 00:00", "2024-01-02 00:00"},
		{"0 0 */1 * MON", "2024-01-02 05:00", "2024-01-03 00:00"},
		{"0 0 * * */1", "2024-01-02 05:00", "2024-01-03 00:00"},
		{"0 0 1-7 * 1", "2024-01-07 01:00", "2024-01-08 00:00"},
		{"0 0 1-31 * 1", "2024-01-07 01:00", "2024-01-08 00:00"},
		{"0 0 30 2 MON", "2024-01-01 00:00", "2024-02-05 00:00"},
		{"0 0 31 4 SUN", "2024-04-01 00:00", "2024-04-07 00:00"},
	}
	for _, c := range cases {
		s, err := cron.Parse(c.expr)
		if err != nil {
			t.Errorf("Parse(%q): %v", c.expr, err)
			continue
		}
		got, ok := s.Next(m8T(t, c.after))
		if c.want == "" {
			if ok || !got.IsZero() {
				t.Errorf("%q.Next(%s) = %v, %v; want zero time, false", c.expr, c.after, got, ok)
			}
			continue
		}
		want := m8T(t, c.want)
		if !ok || !got.Equal(want) || got.Location() != time.UTC {
			t.Errorf("%q.Next(%s) = %v (%v), %v; want %v in UTC", c.expr, c.after, got, got.Location(), ok, want)
		}
	}
}

func TestM8NextConvertsToUTC(t *testing.T) {
	zone := time.FixedZone("UTC+2", 2*60*60)
	after := time.Date(2024, 1, 1, 10, 15, 0, 0, zone) // 08:15 UTC
	got, ok := m8Parse(t, "0 9 * * *").Next(after)
	want := time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC)
	if !ok || !got.Equal(want) || got.Location() != time.UTC {
		t.Errorf("Next(%v) = %v, %v; want %v", after, got, ok, want)
	}
	// 2024-01-01 01:30 at UTC+2 is 2023-12-31 23:30 UTC (a Sunday).
	after = time.Date(2024, 1, 1, 1, 30, 0, 0, zone)
	got, ok = m8Parse(t, "45 23 * * SUN").Next(after)
	want = time.Date(2023, 12, 31, 23, 45, 0, 0, time.UTC)
	if !ok || !got.Equal(want) {
		t.Errorf("Next(%v) = %v, %v; want %v", after, got, ok, want)
	}
}

func TestM8NextIsEarliest(t *testing.T) {
	exprs := []string{"*/7 3-5 * * *", "0 0 13 * 5", "15 */6 1,15 * MON", "0 12 * 2 0-2", "@hourly", "1-59/29 22 * * 6"}
	start := m8T(t, "2024-02-26 21:58")
	for _, e := range exprs {
		s := m8Parse(t, e)
		got, ok := s.Next(start)
		if !ok {
			t.Errorf("%q: no next time", e)
			continue
		}
		if !s.Matches(got) {
			t.Errorf("%q: Next = %v does not match", e, got)
		}
		for m := start.Add(time.Minute); m.Before(got); m = m.Add(time.Minute) {
			if s.Matches(m) {
				t.Errorf("%q: %v matches but Next returned %v", e, m, got)
				break
			}
		}
	}
}

func TestM8Matches(t *testing.T) {
	cases := []struct {
		expr string
		at   string
		want bool
	}{
		{"30 10 * * *", "2024-01-01 10:30:59", true},
		{"30 10 * * *", "2024-01-01 10:31:00", false},
		{"* * 29 2 *", "2024-02-29 12:00", true},
		{"0 0 13 * 5", "2024-09-13 00:00", true},
		{"0 0 13 * 5", "2024-09-20 00:00", true},
		{"0 0 13 * 5", "2024-09-14 00:00", false},
		{"0 0 13 * *", "2024-09-20 00:00", false},
		{"0 0 * * 0", "2024-01-07 00:00", true},
		{"0 0 * * 7", "2024-01-07 00:00", true},
		{"0 0 * * 1-5", "2024-01-06 00:00", false},
		{"* * * DEC *", "2024-12-31 23:59", true},
		{"* * * DEC *", "2024-11-30 23:59", false},
	}
	for _, c := range cases {
		if got := m8Parse(t, c.expr).Matches(m8T(t, c.at)); got != c.want {
			t.Errorf("%q.Matches(%s) = %v, want %v", c.expr, c.at, got, c.want)
		}
	}
	zone := time.FixedZone("UTC-5", -5*60*60)
	if !m8Parse(t, "0 5 * * *").Matches(time.Date(2024, 1, 1, 0, 0, 0, 0, zone)) {
		t.Errorf("Matches must use UTC: 00:00 at UTC-5 is 05:00 UTC")
	}
}

func TestM8NextN(t *testing.T) {
	got := m8Parse(t, "*/20 * * * *").NextN(m8T(t, "2024-01-01 10:15"), 4)
	want := []string{"2024-01-01 10:20", "2024-01-01 10:40", "2024-01-01 11:00", "2024-01-01 11:20"}
	if len(got) != len(want) {
		t.Fatalf("NextN = %v, want %v", got, want)
	}
	for i := range want {
		if !got[i].Equal(m8T(t, want[i])) {
			t.Errorf("NextN[%d] = %v, want %s", i, got[i], want[i])
		}
	}
	leap := m8Parse(t, "0 0 29 2 *").NextN(m8T(t, "2024-01-01 00:00"), 3)
	if len(leap) != 3 || leap[0].Year() != 2024 || leap[1].Year() != 2028 || leap[2].Year() != 2032 {
		t.Errorf("leap NextN = %v", leap)
	}
	if n := m8Parse(t, "0 0 30 2 *").NextN(m8T(t, "2024-01-01 00:00"), 3); len(n) != 0 {
		t.Errorf("NextN of a never-matching schedule = %v", n)
	}
	if n := m8Parse(t, "* * * * *").NextN(m8T(t, "2024-01-01 00:00"), 0); len(n) != 0 {
		t.Errorf("NextN(0) = %v", n)
	}
}

func TestM8ParseErrors(t *testing.T) {
	cases := []struct {
		expr, field, msg string
	}{
		{"", "", "expected 5 fields, got 0"},
		{"   ", "", "expected 5 fields, got 0"},
		{"* * * *", "", "expected 5 fields, got 4"},
		{"* * * * * *", "", "expected 5 fields, got 6"},
		{"@every 5m", "", `unknown macro "@every 5m"`},
		{" @Daily ", "", `unknown macro "@Daily"`},
		{"60 * * * *", "minute", `value "60" out of range 0-59`},
		{"* 24 * * *", "hour", `value "24" out of range 0-23`},
		{"* * 0 * *", "day-of-month", `value "0" out of range 1-31`},
		{"* * 32 * *", "day-of-month", `value "32" out of range 1-31`},
		{"* * 00 * *", "day-of-month", `value "00" out of range 1-31`},
		{"* * * 13 *", "month", `value "13" out of range 1-12`},
		{"* * * 0 *", "month", `value "0" out of range 1-12`},
		{"* * * * 8", "day-of-week", `value "8" out of range 0-7`},
		{"99999999999999999999 * * * *", "minute", `value "99999999999999999999" out of range 0-59`},
		{"1,,2 * * * *", "minute", "empty list item"},
		{"* * * * 1,", "day-of-week", "empty list item"},
		{"* ,1 * * *", "hour", "empty list item"},
		{"x * * * *", "minute", `invalid value "x"`},
		{"? * * * *", "minute", `invalid value "?"`},
		{"* * * ? *", "month", `invalid value "?"`},
		{"* * * JANUARY *", "month", `invalid value "JANUARY"`},
		{"* * * MON *", "month", `invalid value "MON"`},
		{"MON * * * *", "minute", `invalid value "MON"`},
		{"* * * * MON-FRI/x", "day-of-week", `invalid step "x"`},
		{"*/0 * * * *", "minute", `invalid step "0"`},
		{"*/ * * * *", "minute", `invalid step ""`},
		{"1/2/3 * * * *", "minute", `invalid step "2/3"`},
		{"* * * * */MON", "day-of-week", `invalid step "MON"`},
		{"* */-1 * * *", "hour", `invalid step "-1"`},
		{"5-3 * * * *", "minute", `invalid range "5-3"`},
		{"* * * * SAT-SUN", "day-of-week", `invalid range "SAT-SUN"`},
		{"* * * DEC-JAN *", "month", `invalid range "DEC-JAN"`},
		{"-5 * * * *", "minute", `invalid value ""`},
		{"5- * * * *", "minute", `invalid value ""`},
		{"/5 * * * *", "minute", `invalid value ""`},
		{"1-2-3 * * * *", "minute", `invalid value "2-3"`},
		{"*-5 * * * *", "minute", `invalid value "*"`},
		{"1-* * * * *", "minute", `invalid value "*"`},
		{"60 x * * *", "minute", `value "60" out of range 0-59`},
		{"0 x 99 * *", "hour", `invalid value "x"`},
		{"1,60,x * * * *", "minute", `value "60" out of range 0-59`},
		{"5-3/0 * * * *", "minute", `invalid range "5-3"`},
		{"70-80 * * * *", "minute", `value "70" out of range 0-59`},
		{"1-70 * * * *", "minute", `value "70" out of range 0-59`},
		{"x-70/0 * * * *", "minute", `invalid value "x"`},
	}
	for _, c := range cases {
		s, err := cron.Parse(c.expr)
		var pe *cron.ParseError
		if !errors.As(err, &pe) {
			t.Errorf("Parse(%q) error = %v, want *ParseError", c.expr, err)
			continue
		}
		if s != nil {
			t.Errorf("Parse(%q) returned a schedule with an error", c.expr)
		}
		if pe.Field != c.field || pe.Msg != c.msg {
			t.Errorf("Parse(%q) = {Field %q, Msg %q}, want {Field %q, Msg %q}", c.expr, pe.Field, pe.Msg, c.field, c.msg)
		}
	}
	_, err := cron.Parse("61 * * * *")
	if err == nil || err.Error() != `cron: minute field: value "61" out of range 0-59` {
		t.Errorf("Error() = %v", err)
	}
	_, err = cron.Parse("* *")
	if err == nil || err.Error() != "cron: expected 5 fields, got 2" {
		t.Errorf("Error() = %v", err)
	}
	for _, ok := range []string{"  0  0\t*  *  *  ", "0-59 0-23 1-31 1-12 0-7", "0 0 * * Sun,mon,TUE", "0 0 1 jan-DEC/2 *", "0 0 ? * 7/1", "07 08 09 10 06"} {
		if _, err := cron.Parse(ok); err != nil {
			t.Errorf("Parse(%q) = %v, want no error", ok, err)
		}
	}
}
