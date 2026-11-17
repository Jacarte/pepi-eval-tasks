package acceptance

import (
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"testing"

	"kvdb/store"
)

func TestM1PutGetOverwrite(t *testing.T) {
	s := store.New()
	mustMissing(t, s, "a")
	mustPut(t, s, "a", "1")
	mustPut(t, s, "b", "2")
	mustGet(t, s, "a", "1")
	mustGet(t, s, "b", "2")
	mustPut(t, s, "a", "one")
	mustGet(t, s, "a", "one")
	if s.Len() != 2 {
		t.Fatalf("Len = %d, want 2", s.Len())
	}
	if err := s.Put("nil", nil); err != nil {
		t.Fatal(err)
	}
	if v, err := s.Get("nil"); err != nil || len(v) != 0 {
		t.Fatalf("Get(nil value) = %q, %v", v, err)
	}
	bin := string([]byte{0, 1, 2, 0xff, '\n', ' '})
	mustPut(t, s, bin, bin)
	mustGet(t, s, bin, bin)
}

func TestM1EmptyKey(t *testing.T) {
	s := store.New()
	if err := s.Put("", []byte("x")); !errors.Is(err, store.ErrEmptyKey) {
		t.Fatalf("Put empty key: %v", err)
	}
	if _, err := s.Get(""); !errors.Is(err, store.ErrEmptyKey) {
		t.Fatalf("Get empty key: %v", err)
	}
	if err := s.Delete(""); !errors.Is(err, store.ErrEmptyKey) {
		t.Fatalf("Delete empty key: %v", err)
	}
	if s.Len() != 0 {
		t.Fatalf("Len = %d", s.Len())
	}
}

func TestM1Delete(t *testing.T) {
	s := store.New()
	mustPut(t, s, "a", "1")
	mustPut(t, s, "b", "2")
	if err := s.Delete("a"); err != nil {
		t.Fatalf("Delete existing: %v", err)
	}
	mustMissing(t, s, "a")
	if err := s.Delete("a"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("Delete missing: %v, want ErrNotFound", err)
	}
	if s.Len() != 1 {
		t.Fatalf("Len = %d, want 1", s.Len())
	}
	kvs, err := s.Scan("", "", 0)
	if err != nil || len(kvs) != 1 || kvs[0].Key != "b" {
		t.Fatalf("Scan after delete = %v, %v", kvs, err)
	}
	mustPut(t, s, "a", "again")
	mustGet(t, s, "a", "again")
}

func TestM1ValuesAreCopied(t *testing.T) {
	s := store.New()
	in := []byte("hello")
	mustPut(t, s, "k", string(in))
	if err := s.Put("k2", in); err != nil {
		t.Fatal(err)
	}
	in[0] = 'J'
	mustGet(t, s, "k2", "hello")
	got, _ := s.Get("k2")
	got[0] = 'X'
	mustGet(t, s, "k2", "hello")
	kvs, _ := s.ScanPrefix("k", 0)
	for _, kv := range kvs {
		kv.Value[0] = 'Z'
	}
	kvs, _ = s.Scan("", "", 0)
	for _, kv := range kvs {
		kv.Value[0] = 'Z'
	}
	mustGet(t, s, "k", "hello")
	mustGet(t, s, "k2", "hello")
	batch := []store.Op{{Key: "k3", Value: []byte("batch")}}
	if err := s.Apply(batch); err != nil {
		t.Fatal(err)
	}
	batch[0].Value[0] = 'X'
	mustGet(t, s, "k3", "batch")
}

// model is a trivially correct reference: a map plus sorting on demand.
type model map[string]string

func (m model) sorted() []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (m model) scan(start, end string, limit int) []string {
	var out []string
	for _, k := range m.sorted() {
		if k < start || (end != "" && k >= end) {
			continue
		}
		if limit > 0 && len(out) >= limit {
			break
		}
		out = append(out, k)
	}
	return out
}

func (m model) prefix(p string, limit int) []string {
	var out []string
	for _, k := range m.sorted() {
		if !strings.HasPrefix(k, p) {
			continue
		}
		if limit > 0 && len(out) >= limit {
			break
		}
		out = append(out, k)
	}
	return out
}

func checkKVs(t *testing.T, label string, m model, got []store.KV, wantKeys []string) {
	t.Helper()
	if len(got) != len(wantKeys) {
		t.Fatalf("%s: %d results, want %d (got %v, want %q)", label, len(got), len(wantKeys), kvKeys(got), wantKeys)
	}
	for i, k := range wantKeys {
		if got[i].Key != k || string(got[i].Value) != m[k] {
			t.Fatalf("%s: result %d = (%q, %q), want (%q, %q)", label, i, got[i].Key, got[i].Value, k, m[k])
		}
	}
}

func kvKeys(kvs []store.KV) []string {
	var out []string
	for _, kv := range kvs {
		out = append(out, kv.Key)
	}
	return out
}

func randKey(r *rand.Rand) string {
	alphabet := []byte{'a', 'b', 'c', 0x00, 0x7f, 0xff, '/'}
	n := 1 + r.Intn(5)
	b := make([]byte, n)
	for i := range b {
		b[i] = alphabet[r.Intn(len(alphabet))]
	}
	return string(b)
}

func TestM1ScanMatchesModel(t *testing.T) {
	r := rand.New(rand.NewSource(20240611))
	s := store.New()
	m := model{}
	for i := 0; i < 3000; i++ {
		k := randKey(r)
		switch r.Intn(4) {
		case 0:
			err := s.Delete(k)
			_, had := m[k]
			if had != (err == nil) || (!had && !errors.Is(err, store.ErrNotFound)) {
				t.Fatalf("Delete(%q) = %v, present in model = %v", k, err, had)
			}
			delete(m, k)
		default:
			v := fmt.Sprintf("v%d", i)
			mustPut(t, s, k, v)
			m[k] = v
		}
		if i%10 != 0 {
			continue
		}
		start, end := randKey(r), randKey(r)
		if r.Intn(4) == 0 {
			start = ""
		}
		if r.Intn(4) == 0 {
			end = ""
		}
		limit := r.Intn(6) - 1
		got, err := s.Scan(start, end, limit)
		if err != nil {
			t.Fatal(err)
		}
		checkKVs(t, fmt.Sprintf("Scan(%q,%q,%d)", start, end, limit), m, got, m.scan(start, end, limit))
		p := randKey(r)
		p = p[:r.Intn(len(p)+1)]
		got, err = s.ScanPrefix(p, limit)
		if err != nil {
			t.Fatal(err)
		}
		checkKVs(t, fmt.Sprintf("ScanPrefix(%q,%d)", p, limit), m, got, m.prefix(p, limit))
	}
	if s.Len() != len(m) {
		t.Fatalf("Len = %d, want %d", s.Len(), len(m))
	}
	got, _ := s.Scan("", "", 0)
	checkKVs(t, "full scan", m, got, m.sorted())
}

func TestM1ScanBounds(t *testing.T) {
	s := store.New()
	for _, k := range []string{"b", "a", "ab", "a\x00", "abc", "b0", "c", "\xff"} {
		mustPut(t, s, k, "v"+k)
	}
	cases := []struct {
		start, end string
		limit      int
		want       []string
	}{
		{"", "", 0, []string{"a", "a\x00", "ab", "abc", "b", "b0", "c", "\xff"}},
		{"a", "b", 0, []string{"a", "a\x00", "ab", "abc"}},
		{"a\x00", "abc", 0, []string{"a\x00", "ab"}},
		{"aa", "", 3, []string{"ab", "abc", "b"}},
		{"", "a", 0, nil},
		{"c", "b", 0, nil},
		{"b", "b", 0, nil},
		{"zz", "", 0, []string{"\xff"}},
		{"", "", 2, []string{"a", "a\x00"}},
		{"", "", -1, []string{"a", "a\x00", "ab", "abc", "b", "b0", "c", "\xff"}},
	}
	m := model{}
	for _, kv := range []string{"b", "a", "ab", "a\x00", "abc", "b0", "c", "\xff"} {
		m[kv] = "v" + kv
	}
	for _, c := range cases {
		got, err := s.Scan(c.start, c.end, c.limit)
		if err != nil {
			t.Fatal(err)
		}
		checkKVs(t, fmt.Sprintf("Scan(%q,%q,%d)", c.start, c.end, c.limit), m, got, c.want)
	}
	prefixes := []struct {
		p     string
		limit int
		want  []string
	}{
		{"a", 0, []string{"a", "a\x00", "ab", "abc"}},
		{"ab", 0, []string{"ab", "abc"}},
		{"ab", 1, []string{"ab"}},
		{"b", 0, []string{"b", "b0"}},
		{"", 3, []string{"a", "a\x00", "ab"}},
		{"d", 0, nil},
		{"\xff", 0, []string{"\xff"}},
	}
	for _, c := range prefixes {
		got, err := s.ScanPrefix(c.p, c.limit)
		if err != nil {
			t.Fatal(err)
		}
		checkKVs(t, fmt.Sprintf("ScanPrefix(%q,%d)", c.p, c.limit), m, got, c.want)
	}
}

func TestM1Apply(t *testing.T) {
	s := store.New()
	mustPut(t, s, "keep", "k")
	mustPut(t, s, "gone", "g")
	err := s.Apply([]store.Op{
		{Key: "x", Value: []byte("1")},
		{Key: "gone", Delete: true},
		{Key: "x", Value: []byte("2")},
		{Key: "never-existed", Delete: true},
		{Key: "y", Value: []byte("3")},
	})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	mustGet(t, s, "x", "2")
	mustGet(t, s, "y", "3")
	mustGet(t, s, "keep", "k")
	mustMissing(t, s, "gone")
	if s.Len() != 3 {
		t.Fatalf("Len = %d, want 3", s.Len())
	}
	err = s.Apply([]store.Op{{Key: "z", Value: []byte("z")}, {Key: "", Value: []byte("bad")}})
	if !errors.Is(err, store.ErrEmptyKey) {
		t.Fatalf("Apply with empty key: %v", err)
	}
	mustMissing(t, s, "z")
	if err := s.Apply(nil); err != nil {
		t.Fatalf("Apply(nil): %v", err)
	}
	if s.Len() != 3 {
		t.Fatalf("Len = %d after failed Apply, want 3", s.Len())
	}
}

func TestM1ConcurrentWriters(t *testing.T) {
	s := store.New()
	const writers, perWriter = 8, 400
	var wg sync.WaitGroup
	stop := make(chan struct{})
	errs := make(chan error, 64)
	for w := 0; w < writers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < perWriter; i++ {
				k := fmt.Sprintf("w%d/%04d", w, i)
				if err := s.Put(k, []byte(k)); err != nil {
					errs <- err
					return
				}
				if i%3 == 0 {
					if err := s.Delete(k); err != nil {
						errs <- err
						return
					}
				}
			}
		}(w)
	}
	var rwg sync.WaitGroup
	for rd := 0; rd < 4; rd++ {
		rwg.Add(1)
		go func(rd int) {
			defer rwg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				kvs, err := s.ScanPrefix(fmt.Sprintf("w%d/", rd), 0)
				if err != nil {
					errs <- err
					return
				}
				for i, kv := range kvs {
					if string(kv.Value) != kv.Key || (i > 0 && kvs[i-1].Key >= kv.Key) {
						errs <- fmt.Errorf("inconsistent scan result at %d: %q=%q", i, kv.Key, kv.Value)
						return
					}
				}
				s.Get(fmt.Sprintf("w%d/0001", rd))
				s.Len()
			}
		}(rd)
	}
	wg.Wait()
	close(stop)
	rwg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	want := 0
	for w := 0; w < writers; w++ {
		for i := 0; i < perWriter; i++ {
			k := fmt.Sprintf("w%d/%04d", w, i)
			if i%3 == 0 {
				mustMissing(t, s, k)
			} else {
				mustGet(t, s, k, k)
				want++
			}
		}
	}
	if s.Len() != want {
		t.Fatalf("Len = %d, want %d", s.Len(), want)
	}
}

// Readers must never observe half of a batch.
func TestM1ApplyAtomicForReaders(t *testing.T) {
	s := store.New()
	if err := s.Apply([]store.Op{{Key: "pair/a", Value: []byte("0")}, {Key: "pair/b", Value: []byte("0")}}); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	errs := make(chan error, 8)
	go func() {
		defer close(done)
		for i := 1; i <= 3000; i++ {
			v := []byte(fmt.Sprint(i))
			if err := s.Apply([]store.Op{{Key: "pair/a", Value: v}, {Key: "pair/b", Value: v}, {Key: "pair/c", Delete: i%2 == 0, Value: v}}); err != nil {
				errs <- err
				return
			}
		}
	}()
	var wg sync.WaitGroup
	for r := 0; r < 4; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
				}
				kvs, err := s.ScanPrefix("pair/", 0)
				if err != nil {
					errs <- err
					return
				}
				if len(kvs) < 2 || string(kvs[0].Value) != string(kvs[1].Value) {
					errs <- fmt.Errorf("torn batch observed: %v", kvs)
					return
				}
				if len(kvs) == 3 && string(kvs[2].Value) != string(kvs[0].Value) {
					errs <- fmt.Errorf("torn batch observed: %v", kvs)
					return
				}
			}
		}()
	}
	wg.Wait()
	<-done
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	mustGet(t, s, "pair/a", "3000")
	mustMissing(t, s, "pair/c")
}

func TestM1Closed(t *testing.T) {
	s := store.New()
	mustPut(t, s, "a", "1")
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := s.Put("a", []byte("2")); !errors.Is(err, store.ErrClosed) {
		t.Fatalf("Put after Close: %v", err)
	}
	if _, err := s.Get("a"); !errors.Is(err, store.ErrClosed) {
		t.Fatalf("Get after Close: %v", err)
	}
	if err := s.Delete("a"); !errors.Is(err, store.ErrClosed) {
		t.Fatalf("Delete after Close: %v", err)
	}
	if _, err := s.Scan("", "", 0); !errors.Is(err, store.ErrClosed) {
		t.Fatalf("Scan after Close: %v", err)
	}
	if _, err := s.ScanPrefix("", 0); !errors.Is(err, store.ErrClosed) {
		t.Fatalf("ScanPrefix after Close: %v", err)
	}
	if err := s.Apply([]store.Op{{Key: "b", Value: nil}}); !errors.Is(err, store.ErrClosed) {
		t.Fatalf("Apply after Close: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}
