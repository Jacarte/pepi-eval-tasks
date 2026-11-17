package acceptance

import (
	"bytes"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"testing"

	"kvdb/query"
	"kvdb/store"
)

func newQuery(t *testing.T) (*store.Store, *query.Engine, *query.Session) {
	t.Helper()
	st := store.New()
	e := query.NewEngine(st)
	s := e.NewSession()
	t.Cleanup(s.Close)
	return st, e, s
}

func expect(t *testing.T, s *query.Session, line, want string) {
	t.Helper()
	if got := s.Exec(line); got != want {
		t.Fatalf("Exec(%q)\n got: %q\nwant: %q", line, got, want)
	}
}

func TestM2SetGetDel(t *testing.T) {
	st, _, s := newQuery(t)
	expect(t, s, "GET name", "(nil)")
	expect(t, s, "SET name alice", "OK")
	expect(t, s, "GET name", `"alice"`)
	expect(t, s, "SET name bob", "OK")
	expect(t, s, "GET name", `"bob"`)
	mustGet(t, st, "name", "bob")
	expect(t, s, "DEL name", "(integer) 1")
	expect(t, s, "DEL name", "(integer) 0")
	expect(t, s, "GET name", "(nil)")
	mustMissing(t, st, "name")
	mustPut(t, st, "direct", "from-store")
	expect(t, s, "GET direct", `"from-store"`)
}

func TestM2KeywordsCaseInsensitive(t *testing.T) {
	_, _, s := newQuery(t)
	expect(t, s, "set K V", "OK")
	expect(t, s, "Set k v", "OK")
	expect(t, s, "gEt K", `"V"`)
	expect(t, s, "get k", `"v"`)
	expect(t, s, "count k", "(integer) 1")
	expect(t, s, "scan \"\" limit 1", arrayResp([2]string{"K", "V"}))
	expect(t, s, "del K", "(integer) 1")
}

func TestM2Whitespace(t *testing.T) {
	_, _, s := newQuery(t)
	expect(t, s, "  SET \t a   b  ", "OK")
	expect(t, s, "\tGET a\t", `"b"`)
	expect(t, s, `SET back a\b`, "OK")
	expect(t, s, "GET back", `"a\\b"`)
}

func TestM2Errors(t *testing.T) {
	_, _, s := newQuery(t)
	cases := []struct{ in, want string }{
		{"", "(error) ERR empty command"},
		{"   \t ", "(error) ERR empty command"},
		{"fetch x", "(error) ERR unknown command 'fetch'"},
		{"Fetch", "(error) ERR unknown command 'Fetch'"},
		{`"my cmd" x`, "(error) ERR unknown command 'my cmd'"},
		{"SET a", "(error) ERR wrong number of arguments for 'SET'"},
		{"set a b c", "(error) ERR wrong number of arguments for 'SET'"},
		{"get", "(error) ERR wrong number of arguments for 'GET'"},
		{"GET a b", "(error) ERR wrong number of arguments for 'GET'"},
		{"del", "(error) ERR wrong number of arguments for 'DEL'"},
		{"count", "(error) ERR wrong number of arguments for 'COUNT'"},
		{"COUNT a b", "(error) ERR wrong number of arguments for 'COUNT'"},
		{"scan", "(error) ERR wrong number of arguments for 'SCAN'"},
		{"SCAN a LIMIT", "(error) ERR wrong number of arguments for 'SCAN'"},
		{"SCAN a LIMIT 1 2", "(error) ERR wrong number of arguments for 'SCAN'"},
		{`SET "" v`, "(error) ERR empty key"},
		{`GET ""`, "(error) ERR empty key"},
		{`del ""`, "(error) ERR empty key"},
		{"SCAN a TOP 3", "(error) ERR syntax error"},
		{"SCAN a LIMIT 0", "(error) ERR invalid limit"},
		{"SCAN a LIMIT -1", "(error) ERR invalid limit"},
		{"SCAN a LIMIT +2", "(error) ERR invalid limit"},
		{"SCAN a LIMIT 2x", "(error) ERR invalid limit"},
		{"SCAN a LIMIT 99999999999999999999999", "(error) ERR invalid limit"},
		{`SET a "unterminated`, "(error) ERR syntax error"},
		{`SET a "bad\q"`, "(error) ERR syntax error"},
		{`SET a "bad\x4"`, "(error) ERR syntax error"},
		{`SET a "bad\xzz"`, "(error) ERR syntax error"},
		{`SET a b"c`, "(error) ERR syntax error"},
		{`SET a "b"c`, "(error) ERR syntax error"},
		{`SET a "trailing\`, "(error) ERR syntax error"},
		{`fetch "x`, "(error) ERR syntax error"},
		{`SET "" "x`, "(error) ERR syntax error"},
		{`"`, "(error) ERR syntax error"},
	}
	for _, c := range cases {
		if got := s.Exec(c.in); got != c.want {
			t.Errorf("Exec(%q)\n got: %q\nwant: %q", c.in, got, c.want)
		}
	}
	expect(t, s, "COUNT \"\"", "(integer) 0")
}

func TestM2QuoteAllBytes(t *testing.T) {
	all := make([]byte, 256)
	for i := range all {
		all[i] = byte(i)
	}
	inputs := [][]byte{nil, []byte(""), []byte("plain"), []byte(`a"b\c`), []byte("tab\there\nnl\rcr"), []byte("héllo ✓"), all}
	for i := 0; i < 256; i++ {
		inputs = append(inputs, []byte{byte(i)})
	}
	for _, in := range inputs {
		want := specQuote(in)
		if got := query.Quote(in); got != want {
			t.Fatalf("Quote(%v)\n got: %s\nwant: %s", in, got, want)
		}
		back, err := query.Unquote(want)
		if err != nil || !bytes.Equal(back, in) {
			t.Fatalf("Unquote(%s) = %v, %v; want %v", want, back, err, in)
		}
	}
}

func TestM2Unquote(t *testing.T) {
	good := map[string]string{
		`""`:             "",
		`"abc"`:          "abc",
		`"a b\tc"`:       "a b\tc",
		`"\x41\x4a\x4A"`: "AJJ",
		`"\xFF\x00"`:     "\xff\x00",
		`"\\\"\n\r\t"`:   "\\\"\n\r\t",
		"\"raw\ttab\"":   "raw\ttab",
	}
	for in, want := range good {
		got, err := query.Unquote(in)
		if err != nil || string(got) != want {
			t.Errorf("Unquote(%s) = %q, %v; want %q", in, got, err, want)
		}
	}
	bad := []string{``, `"`, `abc`, `"abc`, `abc"`, `"a"b"`, `"a" `, ` "a"`, `"\q"`, `"\x1"`, `"\xg0"`, `"a\"`, `"a""b"`}
	for _, in := range bad {
		if got, err := query.Unquote(in); !errors.Is(err, query.ErrSyntax) {
			t.Errorf("Unquote(%q) = %q, %v; want ErrSyntax", in, got, err)
		}
	}
}

func TestM2QuotedTokensReachStore(t *testing.T) {
	st, _, s := newQuery(t)
	r := rand.New(rand.NewSource(7))
	for i := 0; i < 300; i++ {
		key := make([]byte, 1+r.Intn(12))
		val := make([]byte, r.Intn(40))
		r.Read(key)
		r.Read(val)
		// Mix the upper-case hex form into the request to check both cases are accepted.
		line := "SET " + specQuote(key) + " " + strings.ReplaceAll(specQuote(val), `\xff`, `\xFF`)
		expect(t, s, line, "OK")
		got, err := st.Get(string(key))
		if err != nil || !bytes.Equal(got, val) {
			t.Fatalf("store has %v, %v for key %v; want %v (line %q)", got, err, key, val, line)
		}
		expect(t, s, "GET "+specQuote(key), specQuote(val))
	}
	expect(t, s, `SET "with space" "a \"quoted\" value"`, "OK")
	mustGet(t, st, "with space", `a "quoted" value`)
	expect(t, s, `GET "with space"`, `"a \"quoted\" value"`)
	expect(t, s, `SET e ""`, "OK")
	expect(t, s, `GET e`, `""`)
}

func TestM2ScanAndCount(t *testing.T) {
	_, _, s := newQuery(t)
	for _, kv := range [][2]string{{"user:2", "bob smith"}, {"user:1", "alice"}, {"user:10", "joe"}, {"usex", "no"}, {"admin", "root"}, {"user:\xff", "hi\n"}} {
		expect(t, s, "SET "+q(kv[0])+" "+q(kv[1]), "OK")
	}
	expect(t, s, "SCAN user:", arrayResp(
		[2]string{"user:1", "alice"}, [2]string{"user:10", "joe"}, [2]string{"user:2", "bob smith"}, [2]string{"user:\xff", "hi\n"}))
	expect(t, s, "SCAN user: LIMIT 2", arrayResp([2]string{"user:1", "alice"}, [2]string{"user:10", "joe"}))
	expect(t, s, "SCAN user: LIMIT 100", arrayResp(
		[2]string{"user:1", "alice"}, [2]string{"user:10", "joe"}, [2]string{"user:2", "bob smith"}, [2]string{"user:\xff", "hi\n"}))
	expect(t, s, "SCAN nobody", "(array) 0")
	expect(t, s, `SCAN ""`, arrayResp(
		[2]string{"admin", "root"}, [2]string{"user:1", "alice"}, [2]string{"user:10", "joe"},
		[2]string{"user:2", "bob smith"}, [2]string{"user:\xff", "hi\n"}, [2]string{"usex", "no"}))
	expect(t, s, `SCAN "" LIMIT 1`, arrayResp([2]string{"admin", "root"}))
	expect(t, s, "COUNT user:", "(integer) 4")
	expect(t, s, "COUNT us", "(integer) 5")
	expect(t, s, `COUNT ""`, "(integer) 6")
	expect(t, s, "COUNT zzz", "(integer) 0")
}

// A random workload compared response-by-response with a simple model.
func TestM2RandomWorkloadMatchesModel(t *testing.T) {
	_, _, s := newQuery(t)
	r := rand.New(rand.NewSource(99))
	m := map[string]string{}
	keyOf := func() string {
		return fmt.Sprintf("%c%c:%d", 'a'+r.Intn(3), 'a'+r.Intn(3), r.Intn(8))
	}
	for i := 0; i < 3000; i++ {
		var line, want string
		switch op := r.Intn(10); {
		case op < 4:
			k, v := keyOf(), fmt.Sprintf("val %d \"x\"", r.Intn(1000))
			line, want = "SET "+k+" "+q(v), "OK"
			m[k] = v
		case op < 6:
			k := keyOf()
			line = "GET " + k
			if v, ok := m[k]; ok {
				want = q(v)
			} else {
				want = "(nil)"
			}
		case op < 7:
			k := keyOf()
			line = "DEL " + k
			if _, ok := m[k]; ok {
				want = "(integer) 1"
			} else {
				want = "(integer) 0"
			}
			delete(m, k)
		case op < 8:
			p := keyOf()[:r.Intn(3)]
			n := 0
			for k := range m {
				if strings.HasPrefix(k, p) {
					n++
				}
			}
			line, want = "COUNT "+q(p), fmt.Sprintf("(integer) %d", n)
		default:
			p := keyOf()[:r.Intn(3)]
			limit := r.Intn(5)
			line = "SCAN " + q(p)
			if limit > 0 {
				line += fmt.Sprintf(" LIMIT %d", limit)
			}
			var keys []string
			for k := range m {
				if strings.HasPrefix(k, p) {
					keys = append(keys, k)
				}
			}
			sort.Strings(keys)
			if limit > 0 && len(keys) > limit {
				keys = keys[:limit]
			}
			var pairs [][2]string
			for _, k := range keys {
				pairs = append(pairs, [2]string{k, m[k]})
			}
			want = arrayResp(pairs...)
		}
		if got := s.Exec(line); got != want {
			t.Fatalf("step %d: Exec(%q)\n got: %q\nwant: %q", i, line, got, want)
		}
	}
}

func TestM2SessionsShareEngine(t *testing.T) {
	st, e, a := newQuery(t)
	b := e.NewSession()
	defer b.Close()
	expect(t, a, "SET shared 1", "OK")
	expect(t, b, "GET shared", `"1"`)
	expect(t, b, "DEL shared", "(integer) 1")
	expect(t, a, "GET shared", "(nil)")
	mustMissing(t, st, "shared")
	if a.InTransaction() || b.InTransaction() {
		t.Fatal("InTransaction true without BEGIN")
	}
}
