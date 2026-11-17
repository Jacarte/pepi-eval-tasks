package lrucache_test

import (
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"fivemods/lrucache"
)

type m5Clock struct {
	mu  sync.Mutex
	now time.Time
}

func m5NewClock() *m5Clock {
	return &m5Clock{now: time.Date(2001, 2, 3, 4, 5, 6, 0, time.UTC)}
}

func (c *m5Clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *m5Clock) Advance(d time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(d)
	c.mu.Unlock()
}

type m5Ev struct {
	Key    string
	Val    string
	Reason lrucache.EvictionReason
}

type m5Recorder struct {
	mu  sync.Mutex
	evs []m5Ev
}

func (r *m5Recorder) on(k string, v string, reason lrucache.EvictionReason) {
	r.mu.Lock()
	r.evs = append(r.evs, m5Ev{k, v, reason})
	r.mu.Unlock()
}

func (r *m5Recorder) take() []m5Ev {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := r.evs
	r.evs = nil
	return out
}

func m5NewCache(opts lrucache.Options[string, string]) (*lrucache.Cache[string, string], *m5Clock, *m5Recorder) {
	clk := m5NewClock()
	rec := &m5Recorder{}
	if opts.Clock == nil {
		opts.Clock = clk
	}
	opts.OnEvict = rec.on
	return lrucache.New(opts), clk, rec
}

func m5ExpectEvs(t *testing.T, label string, got []m5Ev, want ...m5Ev) {
	t.Helper()
	if len(got) == 0 && len(want) == 0 {
		return
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s: evictions %+v, want %+v", label, got, want)
	}
}

func m5ExpectKeys(t *testing.T, label string, c *lrucache.Cache[string, string], want ...string) {
	t.Helper()
	got := c.Keys()
	if len(got) == 0 && len(want) == 0 {
		return
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s: Keys = %q, want %q", label, got, want)
	}
}

func m5ExpectStats(t *testing.T, label string, c *lrucache.Cache[string, string], want lrucache.Stats) {
	t.Helper()
	if got := c.Stats(); got != want {
		t.Errorf("%s: Stats = %+v, want %+v", label, got, want)
	}
}

func TestM5LRUOrder(t *testing.T) {
	c, _, rec := m5NewCache(lrucache.Options[string, string]{MaxEntries: 3})
	c.Set("a", "1")
	c.Set("b", "2")
	c.Set("c", "3")
	m5ExpectKeys(t, "after inserts", c, "c", "b", "a")
	if v, ok := c.Get("a"); !ok || v != "1" {
		t.Fatalf("Get(a) = %q, %v", v, ok)
	}
	m5ExpectKeys(t, "after Get(a)", c, "a", "c", "b")
	c.Set("d", "4")
	m5ExpectEvs(t, "insert d", rec.take(), m5Ev{"b", "2", lrucache.ReasonCapacity})
	m5ExpectKeys(t, "after d", c, "d", "a", "c")
	if _, ok := c.Get("b"); ok {
		t.Errorf("b still present")
	}
	if v, ok := c.Peek("c"); !ok || v != "3" {
		t.Errorf("Peek(c) = %q, %v", v, ok)
	}
	m5ExpectKeys(t, "Peek does not touch", c, "d", "a", "c")
	c.Set("e", "5")
	m5ExpectEvs(t, "insert e evicts c despite Peek", rec.take(), m5Ev{"c", "3", lrucache.ReasonCapacity})
	c.Set("a", "1b")
	m5ExpectEvs(t, "replace a", rec.take(), m5Ev{"a", "1", lrucache.ReasonReplaced})
	m5ExpectKeys(t, "replace moves to front", c, "a", "e", "d")
	if c.Len() != 3 {
		t.Errorf("Len = %d", c.Len())
	}
	if v, _ := c.Get("a"); v != "1b" {
		t.Errorf("Get(a) after replace = %q", v)
	}
	if _, ok := c.Peek("zz"); ok {
		t.Errorf("Peek(zz) ok")
	}
	m5ExpectStats(t, "end", c, lrucache.Stats{Hits: 2, Misses: 1, Evictions: 2})
}

func TestM5DeleteAndPurge(t *testing.T) {
	c, _, rec := m5NewCache(lrucache.Options[string, string]{})
	for i := 0; i < 5; i++ {
		c.Set(fmt.Sprint("k", i), fmt.Sprint("v", i))
	}
	if !c.Delete("k2") {
		t.Errorf("Delete(k2) = false")
	}
	if c.Delete("k2") {
		t.Errorf("second Delete(k2) = true")
	}
	if c.Delete("nope") {
		t.Errorf("Delete(nope) = true")
	}
	m5ExpectEvs(t, "delete", rec.take(), m5Ev{"k2", "v2", lrucache.ReasonDeleted})
	c.Get("k0")
	c.Purge()
	m5ExpectEvs(t, "purge LRU to MRU", rec.take(),
		m5Ev{"k1", "v1", lrucache.ReasonDeleted}, m5Ev{"k3", "v3", lrucache.ReasonDeleted},
		m5Ev{"k4", "v4", lrucache.ReasonDeleted}, m5Ev{"k0", "v0", lrucache.ReasonDeleted})
	if c.Len() != 0 || c.Bytes() != 0 || len(c.Keys()) != 0 {
		t.Errorf("after Purge: Len %d Bytes %d Keys %q", c.Len(), c.Bytes(), c.Keys())
	}
	m5ExpectStats(t, "purge keeps stats", c, lrucache.Stats{Hits: 1})
	c.Set("x", "y")
	if v, ok := c.Get("x"); !ok || v != "y" {
		t.Errorf("usable after Purge: %q %v", v, ok)
	}
}

func TestM5TTL(t *testing.T) {
	c, clk, rec := m5NewCache(lrucache.Options[string, string]{TTL: 10 * time.Second})
	c.Set("a", "1")
	clk.Advance(10*time.Second - time.Nanosecond)
	if v, ok := c.Get("a"); !ok || v != "1" {
		t.Fatalf("Get(a) just before expiry = %q, %v", v, ok)
	}
	clk.Advance(time.Nanosecond)
	if v, ok := c.Peek("a"); ok {
		t.Errorf("Peek(a) at expiry = %q, true", v)
	}
	if c.Len() != 1 {
		t.Errorf("Peek removed the expired entry: Len %d", c.Len())
	}
	m5ExpectKeys(t, "Keys hides expired", c)
	m5ExpectEvs(t, "Peek fires nothing", rec.take())
	if _, ok := c.Get("a"); ok {
		t.Errorf("Get(a) at exactly TTL succeeded")
	}
	m5ExpectEvs(t, "lazy expiry", rec.take(), m5Ev{"a", "1", lrucache.ReasonExpired})
	if c.Len() != 0 {
		t.Errorf("Len after expiry = %d", c.Len())
	}
	m5ExpectStats(t, "ttl stats", c, lrucache.Stats{Hits: 1, Misses: 1, Expirations: 1})

	c.SetWithTTL("forever", "f", 0)
	c.SetWithTTL("neg", "n", -time.Second)
	c.SetWithTTL("short", "s", 2*time.Second)
	c.Set("dflt", "d")
	clk.Advance(5 * time.Second)
	m5ExpectKeys(t, "mixed ttl at +5s", c, "dflt", "neg", "forever")
	clk.Advance(5 * time.Second)
	m5ExpectKeys(t, "mixed ttl at +10s", c, "neg", "forever")
	clk.Advance(1000 * time.Hour)
	if v, ok := c.Get("forever"); !ok || v != "f" {
		t.Errorf("ttl 0 entry expired")
	}
	if _, ok := c.Get("neg"); !ok {
		t.Errorf("negative ttl entry expired")
	}
	rec.take()

	c.Set("r", "1")
	clk.Advance(8 * time.Second)
	c.Set("r", "2")
	m5ExpectEvs(t, "replace live", rec.take(), m5Ev{"r", "1", lrucache.ReasonReplaced})
	clk.Advance(8 * time.Second)
	if v, ok := c.Get("r"); !ok || v != "2" {
		t.Errorf("Set did not reset TTL: %q %v", v, ok)
	}
	clk.Advance(2 * time.Second)
	c.Set("r", "3")
	m5ExpectEvs(t, "replace expired", rec.take(), m5Ev{"r", "2", lrucache.ReasonExpired})
}

func TestM5RemoveExpiredAndDelete(t *testing.T) {
	c, clk, rec := m5NewCache(lrucache.Options[string, string]{})
	c.SetWithTTL("a", "1", time.Second)
	c.SetWithTTL("b", "2", time.Hour)
	c.SetWithTTL("c", "3", time.Second)
	c.Set("d", "4")
	c.SetWithTTL("e", "5", 2*time.Second)
	c.Get("a")
	clk.Advance(1500 * time.Millisecond)
	if c.Len() != 5 {
		t.Errorf("Len counts expired entries: got %d want 5", c.Len())
	}
	if n := c.RemoveExpired(); n != 2 {
		t.Errorf("RemoveExpired = %d, want 2", n)
	}
	m5ExpectEvs(t, "RemoveExpired LRU to MRU", rec.take(),
		m5Ev{"c", "3", lrucache.ReasonExpired}, m5Ev{"a", "1", lrucache.ReasonExpired})
	if c.RemoveExpired() != 0 {
		t.Errorf("second RemoveExpired removed entries")
	}
	clk.Advance(time.Second)
	if c.Delete("e") {
		t.Errorf("Delete of expired entry returned true")
	}
	m5ExpectEvs(t, "delete expired", rec.take(), m5Ev{"e", "5", lrucache.ReasonExpired})
	if c.Len() != 2 {
		t.Errorf("Len = %d, want 2", c.Len())
	}
	m5ExpectStats(t, "stats", c, lrucache.Stats{Hits: 1, Expirations: 3})
}

func TestM5CapacityWithExpired(t *testing.T) {
	c, clk, rec := m5NewCache(lrucache.Options[string, string]{MaxEntries: 3})
	c.SetWithTTL("a", "1", time.Second)
	c.SetWithTTL("b", "2", time.Second)
	c.Set("c", "3")
	clk.Advance(time.Minute)
	c.Set("d", "4")
	m5ExpectEvs(t, "expired LRU victim", rec.take(), m5Ev{"a", "1", lrucache.ReasonExpired})
	if c.Len() != 3 {
		t.Errorf("Set removed more than needed: Len %d", c.Len())
	}
	c.Get("c")
	c.Set("e", "5")
	m5ExpectEvs(t, "second victim still expired", rec.take(), m5Ev{"b", "2", lrucache.ReasonExpired})
	c.Set("f", "6")
	m5ExpectEvs(t, "live victim", rec.take(), m5Ev{"d", "4", lrucache.ReasonCapacity})
	m5ExpectKeys(t, "final", c, "f", "e", "c")
	m5ExpectStats(t, "stats", c, lrucache.Stats{Hits: 1, Evictions: 1, Expirations: 2})
}

func TestM5Weights(t *testing.T) {
	calls := 0
	var mu sync.Mutex
	weigh := func(k, v string) int64 {
		mu.Lock()
		calls++
		mu.Unlock()
		return int64(len(v))
	}
	c, _, rec := m5NewCache(lrucache.Options[string, string]{MaxBytes: 10, Weigher: weigh})
	c.Set("a", "xxxx")
	c.Set("b", "xxxx")
	if c.Bytes() != 8 {
		t.Errorf("Bytes = %d, want 8", c.Bytes())
	}
	c.Set("c", "xxx")
	m5ExpectEvs(t, "byte limit", rec.take(), m5Ev{"a", "xxxx", lrucache.ReasonCapacity})
	if c.Bytes() != 7 || c.Len() != 2 {
		t.Errorf("Bytes %d Len %d, want 7, 2", c.Bytes(), c.Len())
	}
	c.Set("c", "xxxxxxxxx")
	m5ExpectEvs(t, "grow on replace", rec.take(),
		m5Ev{"c", "xxx", lrucache.ReasonReplaced}, m5Ev{"b", "xxxx", lrucache.ReasonCapacity})
	if c.Bytes() != 9 {
		t.Errorf("Bytes = %d, want 9", c.Bytes())
	}
	c.Set("big", "01234567890")
	m5ExpectEvs(t, "oversize rejected", rec.take(), m5Ev{"big", "01234567890", lrucache.ReasonCapacity})
	if _, ok := c.Peek("big"); ok {
		t.Errorf("oversize entry stored")
	}
	c.Set("c", "0123456789abc")
	m5ExpectEvs(t, "oversize replace", rec.take(),
		m5Ev{"c", "xxxxxxxxx", lrucache.ReasonReplaced}, m5Ev{"c", "0123456789abc", lrucache.ReasonCapacity})
	if c.Len() != 0 || c.Bytes() != 0 {
		t.Errorf("Len %d Bytes %d after oversize replace", c.Len(), c.Bytes())
	}
	c.Set("exact", "0123456789")
	if v, ok := c.Get("exact"); !ok || v != "0123456789" {
		t.Errorf("entry weighing exactly MaxBytes not stored")
	}
	m5ExpectStats(t, "stats", c, lrucache.Stats{Hits: 1, Evictions: 4})
	mu.Lock()
	if calls != 7 {
		t.Errorf("Weigher called %d times for 7 Sets", calls)
	}
	mu.Unlock()

	both, _, rec2 := m5NewCache(lrucache.Options[string, string]{MaxBytes: 100, MaxEntries: 2, Weigher: weigh})
	both.Set("a", "1")
	both.Set("b", "2")
	both.Set("c", "3")
	m5ExpectEvs(t, "entries limit with bytes", rec2.take(), m5Ev{"a", "1", lrucache.ReasonCapacity})

	neg, _, _ := m5NewCache(lrucache.Options[string, string]{MaxBytes: 5, Weigher: func(k, v string) int64 { return -3 }})
	for i := 0; i < 20; i++ {
		neg.Set(fmt.Sprint(i), "v")
	}
	if neg.Len() != 20 || neg.Bytes() != 0 {
		t.Errorf("negative weights: Len %d Bytes %d, want 20, 0", neg.Len(), neg.Bytes())
	}

	unit, _, rec3 := m5NewCache(lrucache.Options[string, string]{MaxBytes: 2})
	unit.Set("a", "long value")
	unit.Set("b", "long value")
	if unit.Bytes() != 2 {
		t.Errorf("nil Weigher: Bytes %d, want 2", unit.Bytes())
	}
	unit.Set("c", "x")
	m5ExpectEvs(t, "nil weigher weighs 1", rec3.take(), m5Ev{"a", "long value", lrucache.ReasonCapacity})
}

func TestM5Unlimited(t *testing.T) {
	c := lrucache.New(lrucache.Options[int, []int]{Clock: m5NewClock()})
	for i := 0; i < 1000; i++ {
		c.Set(i, []int{i, i * i})
	}
	if c.Len() != 1000 {
		t.Fatalf("Len = %d", c.Len())
	}
	if v, ok := c.Get(31); !ok || v[1] != 961 {
		t.Errorf("Get(31) = %v %v", v, ok)
	}
	if k := c.Keys(); len(k) != 1000 || k[0] != 31 || k[1] != 999 || k[999] != 0 {
		t.Errorf("Keys order wrong: first %v", k[:3])
	}
	one := lrucache.New(lrucache.Options[int, int]{MaxEntries: 1})
	one.Set(1, 1)
	one.Set(2, 2)
	if _, ok := one.Get(1); ok || one.Len() != 1 {
		t.Errorf("MaxEntries 1 broken")
	}
	if v, ok := one.Get(2); !ok || v != 2 {
		t.Errorf("MaxEntries 1 lost newest entry")
	}
}

func TestM5CallbackReentrancy(t *testing.T) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		var c *lrucache.Cache[string, int]
		var seen []string
		c = lrucache.New(lrucache.Options[string, int]{
			MaxEntries: 2,
			Clock:      m5NewClock(),
			OnEvict: func(k string, v int, r lrucache.EvictionReason) {
				seen = append(seen, fmt.Sprintf("%s:%d:%d", k, c.Len(), len(c.Keys())))
				c.Peek(k)
				c.Stats()
			},
		})
		c.Set("a", 1)
		c.Set("b", 2)
		c.Set("c", 3)
		c.Set("c", 4)
		c.Delete("b")
		c.Purge()
		want := []string{"a:2:2", "c:2:2", "b:1:1", "c:0:0"}
		if !reflect.DeepEqual(seen, want) {
			t.Errorf("callback observations %q, want %q", seen, want)
		}
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("deadlock: OnEvict called while holding the cache lock")
	}
}

func TestM5Concurrent(t *testing.T) {
	var mu sync.Mutex
	counts := map[lrucache.EvictionReason]int{}
	c := lrucache.New(lrucache.Options[int, int]{
		MaxEntries: 50,
		Clock:      m5NewClock(),
		OnEvict: func(k, v int, r lrucache.EvictionReason) {
			mu.Lock()
			counts[r]++
			mu.Unlock()
		},
	})
	const workers, ops = 8, 2000
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < ops; i++ {
				k := (i*7 + w*13) % 120
				if i%2 == 0 {
					c.Set(k, i)
				} else {
					c.Get(k)
				}
				if c.Len() > 50 {
					t.Errorf("Len exceeded MaxEntries")
					return
				}
			}
		}(w)
	}
	wg.Wait()
	st := c.Stats()
	if st.Hits+st.Misses != workers*ops/2 {
		t.Errorf("Hits+Misses = %d, want %d", st.Hits+st.Misses, workers*ops/2)
	}
	sets := workers * ops / 2
	mu.Lock()
	defer mu.Unlock()
	if uint64(counts[lrucache.ReasonCapacity]) != st.Evictions {
		t.Errorf("capacity callbacks %d != Evictions %d", counts[lrucache.ReasonCapacity], st.Evictions)
	}
	if got := sets - counts[lrucache.ReasonReplaced] - counts[lrucache.ReasonCapacity]; got != c.Len() {
		t.Errorf("sets - replaced - evicted = %d, Len = %d", got, c.Len())
	}
	if c.Len() != 50 {
		t.Errorf("Len = %d, want 50", c.Len())
	}
}
