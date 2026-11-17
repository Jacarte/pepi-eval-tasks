package acceptance

import (
	"bytes"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"kvdb/query"
	"kvdb/store"
)

var noCompact = store.Options{Sync: store.SyncNever, CompactThreshold: -1}

func walPath(dir string) string  { return filepath.Join(dir, store.WALFileName) }
func snapPath(dir string) string { return filepath.Join(dir, store.SnapshotFileName) }

func TestM4FileNamesAndDefaults(t *testing.T) {
	if store.WALFileName != "wal.log" || store.SnapshotFileName != "snapshot.db" || store.DefaultCompactThreshold != 4<<20 {
		t.Fatal("file name / default constants changed")
	}
	dir := filepath.Join(t.TempDir(), "nested", "data")
	s := openStore(t, dir, store.Options{})
	mustPut(t, s, "a", "1")
	if fileSize(t, walPath(dir)) == 0 {
		t.Fatal("wal.log is empty after Put")
	}
}

func TestM4ReopenAfterClose(t *testing.T) {
	dir := t.TempDir()
	r := rand.New(rand.NewSource(4))
	want := map[string]string{}
	s, err := store.Open(dir, store.Options{Sync: store.SyncAlways, CompactThreshold: -1})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 200; i++ {
		k := fmt.Sprintf("k%02d", r.Intn(40))
		switch r.Intn(5) {
		case 0:
			err := s.Delete(k)
			if _, ok := want[k]; ok != (err == nil) {
				t.Fatalf("Delete(%q): %v", k, err)
			}
			delete(want, k)
		case 1:
			k2 := fmt.Sprintf("k%02d", r.Intn(40))
			v := fmt.Sprint("batch", i)
			if err := s.Apply([]store.Op{{Key: k, Value: []byte(v)}, {Key: k2, Delete: true}}); err != nil {
				t.Fatal(err)
			}
			want[k] = v
			delete(want, k2)
		default:
			v := string(bytes.Repeat([]byte{byte(r.Intn(256))}, r.Intn(30)))
			mustPut(t, s, k, v)
			want[k] = v
		}
	}
	sameMap(t, "before close", dumpStore(t, s), want)
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s2 := openStore(t, dir, noCompact)
	sameMap(t, "after reopen", dumpStore(t, s2), want)
	mustPut(t, s2, "after", "reopen")
	s2.Close()
	s3 := openStore(t, dir, noCompact)
	want["after"] = "reopen"
	sameMap(t, "after second reopen", dumpStore(t, s3), want)
}

// The WAL must contain exactly the spec'd records, one per successful write.
func TestM4WALRecordFormat(t *testing.T) {
	dir := t.TempDir()
	s := openStore(t, dir, noCompact)
	mustPut(t, s, "alpha", "one")
	mustPut(t, s, "beta", "")
	if err := s.Delete("alpha"); err != nil {
		t.Fatal(err)
	}
	s.Delete("missing")            // ErrNotFound: no record
	s.Put("", []byte("x"))         // ErrEmptyKey: no record
	s.Apply([]store.Op{{Key: ""}}) // ErrEmptyKey: no record
	s.Apply(nil)                   // no-op: no record
	bin := string([]byte{0, 0xff, '\n'})
	if err := s.Apply([]store.Op{{Key: "g", Value: []byte(bin)}, {Key: "beta", Delete: true}, {Key: "h", Value: []byte("2")}}); err != nil {
		t.Fatal(err)
	}
	want := [][]recOp{
		{put("alpha", "one")},
		{put("beta", "")},
		{del("alpha")},
		{put("g", bin), del("beta"), put("h", "2")},
	}
	var wantBytes []byte
	for _, r := range want {
		wantBytes = append(wantBytes, encodeRec(r...)...)
	}
	// Read the WAL while the store is still open: records must already be on disk.
	got := readFile(t, walPath(dir))
	if !bytes.Equal(got, wantBytes) {
		recs, _, derr := decodeRecsPrefix(got)
		t.Fatalf("wal.log bytes differ from spec encoding\n got %d bytes: %x\nwant %d bytes: %x\n(decoded %d records, err %v)", len(got), got, len(wantBytes), wantBytes, len(recs), derr)
	}
}

func TestM4RecoverHandWrittenWAL(t *testing.T) {
	dir := t.TempDir()
	big := string(bytes.Repeat([]byte("xyz\x00"), 5000))
	var wal []byte
	wal = append(wal, encodeRec(put("a", "1"), put("b", "2"), put("c", "3"))...)
	wal = append(wal, encodeRec(del("b"))...)
	wal = append(wal, encodeRec()...) // zero ops is valid
	wal = append(wal, encodeRec(put("a", "1b"), put("big", big))...)
	wal = append(wal, encodeRec(del("nothing-here"), put("d", ""))...)
	if err := os.WriteFile(walPath(dir), wal, 0o644); err != nil {
		t.Fatal(err)
	}
	s := openStore(t, dir, noCompact)
	sameMap(t, "recovered", dumpStore(t, s), map[string]string{"a": "1b", "c": "3", "big": big, "d": ""})
}

// Cutting the last record anywhere must keep every earlier record and drop the last one.
func TestM4TornTailIsDiscarded(t *testing.T) {
	src := t.TempDir()
	s := openStore(t, src, noCompact)
	const n = 30
	want := map[string]string{}
	for i := 0; i < n; i++ {
		k, v := fmt.Sprintf("key%02d", i), fmt.Sprintf("value-%d-%s", i, bytes.Repeat([]byte{'v'}, i))
		mustPut(t, s, k, v)
		want[k] = v
	}
	if err := s.Apply([]store.Op{{Key: "key00", Value: []byte("overwritten")}, {Key: "last", Value: []byte("batch")}, {Key: "key01", Delete: true}}); err != nil {
		t.Fatal(err)
	}
	s.Close()
	full := readFile(t, walPath(src))
	recs, ends, err := decodeRecsPrefix(full)
	if err != nil || len(recs) != n+1 {
		t.Fatalf("expected %d valid records, got %d (%v)", n+1, len(recs), err)
	}
	lastStart := ends[n-1]
	lastLen := len(full) - lastStart
	cuts := []int{1, 4, 7, 8, 9, lastLen / 2, lastLen - 1}
	for _, keep := range cuts {
		t.Run(fmt.Sprintf("keep%d", keep), func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(walPath(dir), full[:lastStart+keep], 0o644); err != nil {
				t.Fatal(err)
			}
			s := openStore(t, dir, noCompact)
			sameMap(t, "after torn tail", dumpStore(t, s), want)
			// New writes must land after the valid prefix and survive another reopen.
			mustPut(t, s, "fresh", "write")
			s.Close()
			s2 := openStore(t, dir, noCompact)
			w2 := map[string]string{"fresh": "write"}
			for k, v := range want {
				w2[k] = v
			}
			sameMap(t, "after append following torn tail", dumpStore(t, s2), w2)
			if _, _, err := decodeRecsPrefix(readFile(t, walPath(dir))); err != nil {
				t.Fatalf("wal.log still contains garbage after recovery and append: %v", err)
			}
		})
	}
	corrupt := func(name string, mutate func(b []byte) []byte) {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			b := mutate(append([]byte{}, full...))
			if err := os.WriteFile(walPath(dir), b, 0o644); err != nil {
				t.Fatal(err)
			}
			s := openStore(t, dir, noCompact)
			sameMap(t, name, dumpStore(t, s), want)
		})
	}
	corrupt("bad-crc", func(b []byte) []byte { b[len(b)-1] ^= 0x55; return b })
	corrupt("bad-length", func(b []byte) []byte { b[lastStart] ^= 0x10; return b })
	corrupt("zero-tail", func(b []byte) []byte {
		return append(b[:lastStart], make([]byte, 64)...)
	})
	corrupt("bad-kind", func(b []byte) []byte {
		// Re-encode the last record with an invalid op kind but a valid CRC.
		p := append([]byte{}, b[lastStart+8:]...)
		p[4] = 9
		return append(b[:lastStart], rawRecord(p)...)
	})
}

// A torn transaction commit leaves no trace of any of its writes.
func TestM4TornCommitIsAllOrNothing(t *testing.T) {
	src := t.TempDir()
	st := openStore(t, src, noCompact)
	e := query.NewEngine(st)
	s := e.NewSession()
	expect(t, s, "SET before 1", "OK")
	sizeBefore := fileSize(t, walPath(src))
	expect(t, s, "BEGIN", "OK")
	expect(t, s, "SET t1 a", "OK")
	expect(t, s, "SET t2 b", "OK")
	expect(t, s, "DEL before", "(integer) 1")
	if fileSize(t, walPath(src)) != sizeBefore {
		t.Fatal("buffered transaction writes reached the WAL before COMMIT")
	}
	expect(t, s, "COMMIT", "OK")
	full := readFile(t, walPath(src))
	recs := decodeRecs(t, full[sizeBefore:])
	if len(recs) != 1 || len(recs[0]) < 3 {
		t.Fatalf("COMMIT wrote %d records, want exactly 1 holding all writes: %v", len(recs), recs)
	}
	m := map[string]string{"before": "1"}
	applyRecs(m, recs)
	sameMap(t, "commit record", m, map[string]string{"t1": "a", "t2": "b"})

	dir := t.TempDir()
	if err := os.WriteFile(walPath(dir), full[:len(full)-3], 0o644); err != nil {
		t.Fatal(err)
	}
	st2 := openStore(t, dir, noCompact)
	sameMap(t, "torn commit", dumpStore(t, st2), map[string]string{"before": "1"})
}

// Every acknowledged write is recoverable without Close, under both sync policies.
func TestM4CrashWithoutClose(t *testing.T) {
	for _, pol := range []store.SyncPolicy{store.SyncAlways, store.SyncNever} {
		t.Run(fmt.Sprint("policy", int(pol)), func(t *testing.T) {
			dir := t.TempDir()
			s := openStore(t, dir, store.Options{Sync: pol, CompactThreshold: -1})
			want := map[string]string{}
			for i := 0; i < 40; i++ {
				k := fmt.Sprintf("c%d", i%13)
				if i%7 == 6 {
					if s.Delete(k) == nil {
						delete(want, k)
					}
					continue
				}
				v := fmt.Sprint(i)
				mustPut(t, s, k, v)
				want[k] = v
				if i%10 == 9 {
					s2 := openStore(t, crashCopy(t, dir), noCompact)
					sameMap(t, fmt.Sprintf("crash after write %d", i), dumpStore(t, s2), want)
				}
			}
		})
	}
}

func TestM4CompactWritesSnapshot(t *testing.T) {
	dir := t.TempDir()
	s := openStore(t, dir, noCompact)
	want := map[string]string{}
	for i := 0; i < 500; i++ {
		k, v := fmt.Sprintf("s%03d", i%120), fmt.Sprintf("v%d", i)
		mustPut(t, s, k, v)
		want[k] = v
	}
	for i := 0; i < 120; i += 3 {
		k := fmt.Sprintf("s%03d", i)
		if err := s.Delete(k); err != nil {
			t.Fatal(err)
		}
		delete(want, k)
	}
	if err := s.Compact(); err != nil {
		t.Fatalf("Compact: %v", err)
	}
	if sz := fileSize(t, walPath(dir)); sz != 0 {
		t.Fatalf("wal.log is %d bytes after Compact, want 0", sz)
	}
	recs := decodeRecs(t, readFile(t, snapPath(dir)))
	snap := map[string]string{}
	for _, r := range recs {
		for _, op := range r {
			if op.del {
				t.Fatalf("snapshot contains a delete op for %q", op.key)
			}
		}
	}
	applyRecs(snap, recs)
	sameMap(t, "snapshot content", snap, want)
	if _, err := os.Stat(snapPath(dir) + ".tmp"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("snapshot.db.tmp left behind: %v", err)
	}

	mustPut(t, s, "post", "compact")
	if err := s.Delete("s001"); err != nil {
		t.Fatal(err)
	}
	want["post"] = "compact"
	delete(want, "s001")
	s2 := openStore(t, crashCopy(t, dir), noCompact)
	sameMap(t, "snapshot + wal after crash", dumpStore(t, s2), want)

	if err := s.Compact(); err != nil {
		t.Fatal(err)
	}
	s.Close()
	s3 := openStore(t, dir, noCompact)
	sameMap(t, "after second compact", dumpStore(t, s3), want)
}

func TestM4RecoverHandWrittenSnapshotAndWAL(t *testing.T) {
	dir := t.TempDir()
	var snap []byte
	snap = append(snap, encodeRec(put("a", "snap-a"), put("b", "snap-b"))...)
	snap = append(snap, encodeRec(put("c", "snap-c"))...)
	snap = append(snap, encodeRec(put("d", "snap-d"), put("e", "snap-e"), put("f", "snap-f"))...)
	var wal []byte
	wal = append(wal, encodeRec(put("b", "wal-b"))...)
	wal = append(wal, encodeRec(del("c"), put("g", "wal-g"))...)
	wal = append(wal, encodeRec(put("x", "torn"))[:10]...)
	if err := os.WriteFile(snapPath(dir), snap, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(walPath(dir), wal, 0o644); err != nil {
		t.Fatal(err)
	}
	// A leftover temp snapshot from a crash must be ignored.
	if err := os.WriteFile(snapPath(dir)+".tmp", encodeRec(put("a", "tmp"), put("zzz", "tmp")), 0o644); err != nil {
		t.Fatal(err)
	}
	s := openStore(t, dir, noCompact)
	want := map[string]string{"a": "snap-a", "b": "wal-b", "d": "snap-d", "e": "snap-e", "f": "snap-f", "g": "wal-g"}
	sameMap(t, "snapshot + wal", dumpStore(t, s), want)
	if _, err := os.Stat(snapPath(dir) + ".tmp"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("snapshot.db.tmp not removed by Open: %v", err)
	}
	mustPut(t, s, "h", "new")
	s.Close()
	s2 := openStore(t, dir, noCompact)
	want["h"] = "new"
	sameMap(t, "reopen", dumpStore(t, s2), want)
}

func TestM4CorruptSnapshotIsAnError(t *testing.T) {
	for name, data := range map[string][]byte{
		"truncated": encodeRec(put("a", "1"), put("b", "2"))[:12],
		"bad-crc": func() []byte {
			b := encodeRec(put("a", "1"))
			b[len(b)-1] ^= 1
			return b
		}(),
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(snapPath(dir), data, 0o644); err != nil {
				t.Fatal(err)
			}
			s, err := store.Open(dir, noCompact)
			if err == nil {
				s.Close()
				t.Fatal("Open succeeded with a corrupt snapshot.db")
			}
		})
	}
}

func TestM4AutoCompactionBoundsWAL(t *testing.T) {
	dir := t.TempDir()
	const threshold = 4096
	s := openStore(t, dir, store.Options{Sync: store.SyncNever, CompactThreshold: threshold})
	want := map[string]string{}
	maxRec := 0
	r := rand.New(rand.NewSource(11))
	for i := 0; i < 3000; i++ {
		k := fmt.Sprintf("auto%02d", r.Intn(25))
		v := string(bytes.Repeat([]byte{byte('a' + i%26)}, 20+r.Intn(80)))
		if rl := len(encodeRec(put(k, v))); rl > maxRec {
			maxRec = rl
		}
		mustPut(t, s, k, v)
		want[k] = v
		if i%97 == 0 {
			if sz := fileSize(t, walPath(dir)); sz > threshold+int64(maxRec) {
				t.Fatalf("after %d writes wal.log is %d bytes, threshold %d", i+1, sz, threshold)
			}
		}
		if i%500 == 499 {
			s2 := openStore(t, crashCopy(t, dir), noCompact)
			sameMap(t, fmt.Sprintf("crash after %d writes", i+1), dumpStore(t, s2), want)
			s2.Close()
		}
	}
	if _, err := os.Stat(snapPath(dir)); err != nil {
		t.Fatalf("no snapshot.db after automatic compaction: %v", err)
	}
	if sz := fileSize(t, walPath(dir)); sz > threshold+int64(maxRec) {
		t.Fatalf("wal.log is %d bytes, threshold %d", sz, threshold)
	}
	s.Close()
	s3 := openStore(t, dir, noCompact)
	sameMap(t, "reopen after auto compaction", dumpStore(t, s3), want)
}

// The WAL order must match the order writes took effect, even with concurrent writers.
func TestM4ConcurrentWritersReplayIdentically(t *testing.T) {
	dir := t.TempDir()
	s := openStore(t, dir, store.Options{Sync: store.SyncNever, CompactThreshold: 16 << 10})
	var wg sync.WaitGroup
	for w := 0; w < 8; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < 300; i++ {
				k := fmt.Sprintf("hot%d", (i+w)%6)
				var err error
				switch i % 5 {
				case 4:
					err = s.Delete(k)
					if errors.Is(err, store.ErrNotFound) {
						err = nil
					}
				case 3:
					err = s.Apply([]store.Op{{Key: k, Value: []byte(fmt.Sprint(w, "/", i))}, {Key: "hot-batch", Value: []byte(fmt.Sprint(w))}})
				default:
					err = s.Put(k, []byte(fmt.Sprint(w, ":", i)))
				}
				if err != nil {
					t.Error(err)
					return
				}
			}
		}(w)
	}
	wg.Wait()
	want := dumpStore(t, s)
	s2 := openStore(t, crashCopy(t, dir), noCompact)
	sameMap(t, "replayed state", dumpStore(t, s2), want)
	s.Close()
	s3 := openStore(t, dir, noCompact)
	sameMap(t, "reopened state", dumpStore(t, s3), want)
}

func TestM4ClosedDurableStore(t *testing.T) {
	dir := t.TempDir()
	s, err := store.Open(dir, store.Options{})
	if err != nil {
		t.Fatal(err)
	}
	mustPut(t, s, "a", "1")
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	size := fileSize(t, walPath(dir))
	if err := s.Put("b", []byte("2")); !errors.Is(err, store.ErrClosed) {
		t.Fatalf("Put after Close: %v", err)
	}
	if err := s.Compact(); !errors.Is(err, store.ErrClosed) {
		t.Fatalf("Compact after Close: %v", err)
	}
	if fileSize(t, walPath(dir)) != size {
		t.Fatal("WAL changed after Close")
	}
	if err := store.New().Compact(); err != nil {
		t.Fatalf("Compact on in-memory store: %v", err)
	}
}
