package acceptance

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"testing"
)

func TestM3BeginCommitVisibility(t *testing.T) {
	st, e, a := newQuery(t)
	b := e.NewSession()
	defer b.Close()
	expect(t, a, "SET base 0", "OK")
	expect(t, a, "BEGIN", "OK")
	if !a.InTransaction() {
		t.Fatal("InTransaction false after BEGIN")
	}
	expect(t, a, "SET x 1", "OK")
	expect(t, a, "SET base 1", "OK")
	expect(t, a, "GET x", `"1"`)
	expect(t, a, "GET base", `"1"`)
	expect(t, b, "GET x", "(nil)")
	expect(t, b, "GET base", `"0"`)
	mustMissing(t, st, "x")
	mustGet(t, st, "base", "0")
	expect(t, a, "COMMIT", "OK")
	if a.InTransaction() {
		t.Fatal("InTransaction true after COMMIT")
	}
	expect(t, b, "GET x", `"1"`)
	expect(t, b, "GET base", `"1"`)
	mustGet(t, st, "x", "1")
}

func TestM3Rollback(t *testing.T) {
	st, e, a := newQuery(t)
	b := e.NewSession()
	defer b.Close()
	expect(t, a, "SET keep yes", "OK")
	expect(t, a, "BEGIN", "OK")
	expect(t, a, "SET temp 1", "OK")
	expect(t, a, "DEL keep", "(integer) 1")
	expect(t, a, "GET keep", "(nil)")
	expect(t, b, "GET keep", `"yes"`)
	expect(t, a, "ROLLBACK", "OK")
	if a.InTransaction() {
		t.Fatal("InTransaction true after ROLLBACK")
	}
	expect(t, a, "GET temp", "(nil)")
	expect(t, a, "GET keep", `"yes"`)
	mustMissing(t, st, "temp")
	mustGet(t, st, "keep", "yes")
}

func TestM3StateErrors(t *testing.T) {
	_, _, s := newQuery(t)
	expect(t, s, "COMMIT", "(error) ERR no transaction in progress")
	expect(t, s, "ROLLBACK", "(error) ERR no transaction in progress")
	expect(t, s, "begin", "OK")
	expect(t, s, "SET a 1", "OK")
	expect(t, s, "BEGIN", "(error) ERR transaction already in progress")
	if !s.InTransaction() {
		t.Fatal("nested BEGIN ended the transaction")
	}
	expect(t, s, "GET a", `"1"`)
	expect(t, s, "SET a", "(error) ERR wrong number of arguments for 'SET'")
	expect(t, s, `SET a "oops`, "(error) ERR syntax error")
	expect(t, s, "BEGIN now", "(error) ERR wrong number of arguments for 'BEGIN'")
	expect(t, s, "COMMIT now", "(error) ERR wrong number of arguments for 'COMMIT'")
	expect(t, s, "ROLLBACK now", "(error) ERR wrong number of arguments for 'ROLLBACK'")
	expect(t, s, "GET a", `"1"`)
	expect(t, s, "commit", "OK")
	expect(t, s, "GET a", `"1"`)
	expect(t, s, "COMMIT", "(error) ERR no transaction in progress")
}

func TestM3ReadYourWritesMerged(t *testing.T) {
	st, e, a := newQuery(t)
	other := e.NewSession()
	defer other.Close()
	for _, k := range []string{"p:1", "p:2", "p:3", "p:5", "q:1"} {
		expect(t, a, "SET "+k+" c"+k, "OK")
	}
	expect(t, a, "BEGIN", "OK")
	expect(t, a, "DEL p:2", "(integer) 1")
	expect(t, a, "DEL p:2", "(integer) 0")
	expect(t, a, "DEL p:9", "(integer) 0")
	expect(t, a, "SET p:4 t4", "OK")
	expect(t, a, "SET p:1 t1", "OK")
	expect(t, a, "SET p:0 t0", "OK")
	expect(t, a, "DEL p:0", "(integer) 1")
	expect(t, a, "SET p:6 t6", "OK")
	expect(t, a, "GET p:2", "(nil)")
	expect(t, a, "GET p:0", "(nil)")
	expect(t, a, "COUNT p:", "(integer) 5")
	expect(t, a, "SCAN p:", arrayResp(
		[2]string{"p:1", "t1"}, [2]string{"p:3", "cp:3"}, [2]string{"p:4", "t4"}, [2]string{"p:5", "cp:5"}, [2]string{"p:6", "t6"}))
	expect(t, a, "SCAN p: LIMIT 3", arrayResp([2]string{"p:1", "t1"}, [2]string{"p:3", "cp:3"}, [2]string{"p:4", "t4"}))
	expect(t, a, "SCAN p: LIMIT 2", arrayResp([2]string{"p:1", "t1"}, [2]string{"p:3", "cp:3"}))
	expect(t, a, "SCAN p:2 LIMIT 1", "(array) 0")
	// Committed changes by others show through for keys the transaction did not touch.
	expect(t, other, "SET p:3 o3", "OK")
	expect(t, other, "DEL p:5", "(integer) 1")
	expect(t, a, "SCAN p:", arrayResp(
		[2]string{"p:1", "t1"}, [2]string{"p:3", "o3"}, [2]string{"p:4", "t4"}, [2]string{"p:6", "t6"}))
	expect(t, other, "COUNT p:", "(integer) 3")
	expect(t, other, "SCAN p:", arrayResp([2]string{"p:1", "cp:1"}, [2]string{"p:2", "cp:2"}, [2]string{"p:3", "o3"}))
	expect(t, a, "SET p:2 back", "OK")
	expect(t, a, "GET p:2", `"back"`)
	expect(t, a, "COMMIT", "OK")
	want := map[string]string{"p:1": "t1", "p:2": "back", "p:3": "o3", "p:4": "t4", "p:6": "t6", "q:1": "cq:1"}
	sameMap(t, "store after commit", dumpStore(t, st), want)
}

func TestM3WriteConflict(t *testing.T) {
	st, e, a := newQuery(t)
	b := e.NewSession()
	defer b.Close()
	expect(t, b, "SET x 0", "OK")
	expect(t, a, "BEGIN", "OK")
	expect(t, a, "SET x 1", "OK")
	expect(t, a, "SET y 1", "OK")
	expect(t, b, "SET x 2", "OK")
	expect(t, a, "COMMIT", "(error) ERR transaction conflict")
	if a.InTransaction() {
		t.Fatal("transaction still open after conflict")
	}
	mustGet(t, st, "x", "2")
	mustMissing(t, st, "y")
	expect(t, a, "COMMIT", "(error) ERR no transaction in progress")

	// A DEL by another session conflicts as well, also with a buffered DEL.
	expect(t, a, "BEGIN", "OK")
	expect(t, a, "DEL x", "(integer) 1")
	expect(t, b, "DEL x", "(integer) 1")
	expect(t, a, "COMMIT", "(error) ERR transaction conflict")

	// Writes that happened before BEGIN do not conflict.
	expect(t, b, "SET x 3", "OK")
	expect(t, a, "BEGIN", "OK")
	expect(t, a, "SET x 4", "OK")
	expect(t, a, "COMMIT", "OK")
	mustGet(t, st, "x", "4")
}

func TestM3ConflictBetweenTransactions(t *testing.T) {
	st, e, a := newQuery(t)
	b := e.NewSession()
	defer b.Close()
	c := e.NewSession()
	defer c.Close()
	expect(t, a, "BEGIN", "OK")
	expect(t, b, "BEGIN", "OK")
	expect(t, c, "BEGIN", "OK")
	expect(t, a, "SET k a", "OK")
	expect(t, b, "SET k b", "OK")
	expect(t, c, "SET other c", "OK")
	expect(t, b, "COMMIT", "OK")
	expect(t, a, "COMMIT", "(error) ERR transaction conflict")
	expect(t, c, "COMMIT", "OK")
	mustGet(t, st, "k", "b")
	mustGet(t, st, "other", "c")

	// Keys that are only read never conflict.
	expect(t, a, "BEGIN", "OK")
	expect(t, a, "GET k", `"b"`)
	expect(t, a, "COUNT k", "(integer) 1")
	expect(t, a, "SET mine 1", "OK")
	expect(t, b, "SET k changed", "OK")
	expect(t, a, "COMMIT", "OK")
	mustGet(t, st, "mine", "1")

	// A DEL that found nothing still puts the key in the write set.
	expect(t, a, "BEGIN", "OK")
	expect(t, a, "DEL ghost", "(integer) 0")
	expect(t, b, "SET ghost boo", "OK")
	expect(t, a, "COMMIT", "(error) ERR transaction conflict")
	mustGet(t, st, "ghost", "boo")

	// Rolled back transactions do not count as modifications.
	expect(t, a, "BEGIN", "OK")
	expect(t, a, "SET r 1", "OK")
	expect(t, b, "BEGIN", "OK")
	expect(t, b, "SET r 2", "OK")
	expect(t, b, "ROLLBACK", "OK")
	expect(t, a, "COMMIT", "OK")
	mustGet(t, st, "r", "1")
}

func TestM3SessionCloseDiscards(t *testing.T) {
	st, e, _ := newQuery(t)
	s := e.NewSession()
	expect(t, s, "BEGIN", "OK")
	expect(t, s, "SET gone 1", "OK")
	s.Close()
	mustMissing(t, st, "gone")
	s2 := e.NewSession()
	defer s2.Close()
	expect(t, s2, "GET gone", "(nil)")
	if s2.InTransaction() {
		t.Fatal("new session starts in a transaction")
	}
}

// Concurrent read-modify-write increments with retry on conflict must not lose updates.
func TestM3ConcurrentIncrements(t *testing.T) {
	st, e, s := newQuery(t)
	expect(t, s, "SET counter 0", "OK")
	const workers, perWorker = 8, 60
	var wg sync.WaitGroup
	var mu sync.Mutex
	conflicts := 0
	errs := make(chan string, workers)
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			sess := e.NewSession()
			defer sess.Close()
			for i := 0; i < perWorker; {
				if r := sess.Exec("BEGIN"); r != "OK" {
					errs <- "BEGIN: " + r
					return
				}
				v, err := strconv.Unquote(sess.Exec("GET counter"))
				if err != nil {
					errs <- "GET counter: " + err.Error()
					return
				}
				n, _ := strconv.Atoi(v)
				sess.Exec(fmt.Sprintf("SET counter %d", n+1))
				sess.Exec(fmt.Sprintf("SET w%d:%d done", w, i))
				switch r := sess.Exec("COMMIT"); {
				case r == "OK":
					i++
				case r == "(error) ERR transaction conflict":
					mu.Lock()
					conflicts++
					mu.Unlock()
				default:
					errs <- "COMMIT: " + r
					return
				}
			}
		}(w)
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Fatal(e)
	}
	mustGet(t, st, "counter", strconv.Itoa(workers*perWorker))
	kvs, err := st.ScanPrefix("w", 0)
	if err != nil || len(kvs) != workers*perWorker {
		t.Fatalf("marker keys: %d, %v; want %d", len(kvs), err, workers*perWorker)
	}
	for _, kv := range kvs {
		if !strings.HasPrefix(kv.Key, "w") || string(kv.Value) != "done" {
			t.Fatalf("bad marker %q=%q", kv.Key, kv.Value)
		}
	}
	t.Logf("conflicts retried: %d", conflicts)
}
