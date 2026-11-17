package lrucache_test

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"fivemods/lrucache"
)

var m5ReasonNames = map[lrucache.EvictionReason]string{
	lrucache.ReasonCapacity: "capacity",
	lrucache.ReasonExpired:  "expired",
	lrucache.ReasonDeleted:  "deleted",
	lrucache.ReasonReplaced: "replaced",
}

// Each script line is one operation:
//
//	set K V | ttl K V DUR | get K V | get K - | peek K V | peek K - | del K BOOL | adv DUR
//	keys K... | len N | bytes N | evs K:V:REASON... | rmexp N | purge | stats HITS MISSES EVICTIONS EXPIRATIONS
func TestM5Scripts(t *testing.T) {
	cases := []struct {
		name     string
		max      int
		maxBytes int64 // > 0 also installs a Weigher returning len(value)
		unitW    bool  // MaxBytes without a Weigher
		ttl      time.Duration
		script   string
	}{
		{"hit and miss", 0, 0, false, 0, "set a 1; get a 1; get b -; stats 1 1 0 0"},
		{"lru eviction", 2, 0, false, 0, "set a 1; set b 2; get a 1; set c 3; evs b:2:capacity; keys c a"},
		{"insert order eviction", 2, 0, false, 0, "set a 1; set b 2; set c 3; set d 4; evs a:1:capacity b:2:capacity; keys d c"},
		{"peek keeps order", 2, 0, false, 0, "set a 1; set b 2; peek a 1; set c 3; evs a:1:capacity; peek a -; stats 0 0 1 0"},
		{"replace", 0, 0, false, 0, "set a 1; set a 2; evs a:1:replaced; get a 2; len 1; stats 1 0 0 0"},
		{"replace moves to front", 2, 0, false, 0, "set a 1; set b 2; set a 3; set c 4; evs a:1:replaced b:2:capacity; keys c a"},
		{"delete live", 0, 0, false, 0, "set a 1; del a true; evs a:1:deleted; get a -; len 0"},
		{"delete absent", 0, 0, false, 0, "del a false; evs; stats 0 0 0 0"},
		{"delete expired", 0, 0, false, time.Second, "set a 1; adv 1s; del a false; evs a:1:expired; stats 0 0 0 1"},
		{"ttl exact boundary", 0, 0, false, 10 * time.Second, "set a 1; adv 10s; get a -; evs a:1:expired; stats 0 1 0 1"},
		{"ttl just before", 0, 0, false, 10 * time.Second, "set a 1; adv 9999ms; get a 1; stats 1 0 0 0"},
		{"explicit ttl overrides default", 0, 0, false, time.Hour, "ttl a 1 2s; adv 2s; get a -; evs a:1:expired"},
		{"ttl zero never expires", 0, 0, false, time.Second, "ttl a 1 0s; adv 1000h; get a 1"},
		{"set resets ttl", 0, 0, false, 10 * time.Second, "set a 1; adv 9s; set a 2; adv 9s; get a 2; adv 1s; get a -"},
		{"replace expired", 0, 0, false, time.Second, "set a 1; adv 5s; set a 2; evs a:1:expired; get a 2; stats 1 0 0 1"},
		{"expired victim", 2, 0, false, 0, "ttl a 1 1s; set b 2; adv 1s; set c 3; evs a:1:expired; stats 0 0 0 1; len 2"},
		{"expiry alone removes nothing", 3, 0, false, time.Second, "set a 1; set b 2; adv 1h; set c 3; evs; len 3; keys c"},
		{"remove expired", 0, 0, false, 0, "ttl a 1 1s; set b 2; ttl c 3 1s; get a 1; adv 1s; rmexp 2; evs c:3:expired a:1:expired; keys b; rmexp 0"},
		{"keys skip expired", 0, 0, false, 0, "ttl a 1 1s; set b 2; ttl c 3 3s; adv 2s; keys c b; len 3"},
		{"bytes multiple victims", 0, 10, false, 0, "set a 111; set b 222; set c 333; set d 4444444; evs a:111:capacity b:222:capacity; bytes 10; keys d c"},
		{"bytes oversize rejected", 0, 5, false, 0, "set a 12; set big 123456; evs big:123456:capacity; keys a; bytes 2; stats 0 0 1 0"},
		{"bytes oversize replace", 0, 5, false, 0, "set a 12; set a 123456; evs a:12:replaced a:123456:capacity; len 0; bytes 0"},
		{"bytes exact fit", 0, 5, false, 0, "set a 1; set b 12345; evs a:1:capacity; get b 12345; bytes 5"},
		{"bytes grow on replace", 0, 6, false, 0, "set a 11; set b 22; set c 33; set a 1111; evs a:11:replaced b:22:capacity; bytes 6; keys a c"},
		{"unit weights", 0, 2, true, 0, "set a 1111; set b 2222; bytes 2; set c 3; evs a:1111:capacity; bytes 2"},
		{"purge", 0, 0, false, 0, "set a 1; set b 2; set c 3; get a 1; purge; evs b:2:deleted c:3:deleted a:1:deleted; len 0; stats 1 0 0 0"},
		{"max entries one", 1, 0, false, 0, "set a 1; set b 2; evs a:1:capacity; get a -; get b 2; set b 3; evs b:2:replaced; len 1"},
		{"peek expired stays stored", 0, 0, false, time.Second, "set a 1; adv 1s; peek a -; len 1; evs; get a -; len 0"},
		{"bytes and entries", 2, 100, false, 0, "set a 1; set b 2; set c 3; evs a:1:capacity; bytes 2"},
	}
	for _, tc := range cases {
		t.Run(strings.ReplaceAll(tc.name, " ", "_"), func(t *testing.T) {
			clk := m5NewClock()
			rec := &m5Recorder{}
			opts := lrucache.Options[string, string]{MaxEntries: tc.max, TTL: tc.ttl, Clock: clk, OnEvict: rec.on}
			if tc.maxBytes > 0 {
				opts.MaxBytes = tc.maxBytes
				if !tc.unitW {
					opts.Weigher = func(k, v string) int64 { return int64(len(v)) }
				}
			}
			c := lrucache.New(opts)
			for _, op := range strings.Split(tc.script, ";") {
				f := strings.Fields(op)
				if len(f) == 0 {
					continue
				}
				fail := func(format string, args ...any) {
					t.Helper()
					t.Errorf("op %q: %s", strings.TrimSpace(op), fmt.Sprintf(format, args...))
				}
				switch f[0] {
				case "set":
					c.Set(f[1], f[2])
				case "ttl":
					d, _ := time.ParseDuration(f[3])
					c.SetWithTTL(f[1], f[2], d)
				case "get", "peek":
					var v string
					var ok bool
					if f[0] == "get" {
						v, ok = c.Get(f[1])
					} else {
						v, ok = c.Peek(f[1])
					}
					if f[2] == "-" && ok {
						fail("got %q, want miss", v)
					} else if f[2] != "-" && (!ok || v != f[2]) {
						fail("got %q,%v want %q", v, ok, f[2])
					}
				case "del":
					want := f[2] == "true"
					if got := c.Delete(f[1]); got != want {
						fail("got %v", got)
					}
				case "adv":
					d, _ := time.ParseDuration(f[1])
					clk.Advance(d)
				case "keys":
					got := c.Keys()
					if strings.Join(got, " ") != strings.Join(f[1:], " ") {
						fail("got %q", got)
					}
				case "len":
					n, _ := strconv.Atoi(f[1])
					if c.Len() != n {
						fail("got %d", c.Len())
					}
				case "bytes":
					n, _ := strconv.ParseInt(f[1], 10, 64)
					if c.Bytes() != n {
						fail("got %d", c.Bytes())
					}
				case "evs":
					var got []string
					for _, e := range rec.take() {
						got = append(got, e.Key+":"+e.Val+":"+m5ReasonNames[e.Reason])
					}
					if strings.Join(got, " ") != strings.Join(f[1:], " ") {
						fail("got %q", got)
					}
				case "rmexp":
					n, _ := strconv.Atoi(f[1])
					if got := c.RemoveExpired(); got != n {
						fail("got %d", got)
					}
				case "purge":
					c.Purge()
				case "stats":
					var want [4]uint64
					for i := range want {
						want[i], _ = strconv.ParseUint(f[1+i], 10, 64)
					}
					st := c.Stats()
					if got := [4]uint64{st.Hits, st.Misses, st.Evictions, st.Expirations}; got != want {
						fail("got %v", got)
					}
				default:
					t.Fatalf("bad op %q", op)
				}
			}
		})
	}
}
