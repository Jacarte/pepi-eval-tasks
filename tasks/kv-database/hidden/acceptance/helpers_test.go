package acceptance

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"kvdb/store"
)

// ---- record format (SPEC.md, M4) implemented independently of the store ----

type recOp struct {
	del   bool
	key   string
	value []byte
}

func put(k, v string) recOp { return recOp{key: k, value: []byte(v)} }
func del(k string) recOp    { return recOp{del: true, key: k} }

func encodeRec(ops ...recOp) []byte {
	var p []byte
	p = binary.LittleEndian.AppendUint32(p, uint32(len(ops)))
	for _, op := range ops {
		if op.del {
			p = append(p, 2)
		} else {
			p = append(p, 1)
		}
		p = binary.LittleEndian.AppendUint32(p, uint32(len(op.key)))
		p = append(p, op.key...)
		if !op.del {
			p = binary.LittleEndian.AppendUint32(p, uint32(len(op.value)))
			p = append(p, op.value...)
		}
	}
	var r []byte
	r = binary.LittleEndian.AppendUint32(r, uint32(len(p)))
	r = binary.LittleEndian.AppendUint32(r, crc32.ChecksumIEEE(p))
	return append(r, p...)
}

// rawRecord frames an arbitrary payload with a correct length and CRC.
func rawRecord(payload []byte) []byte {
	var r []byte
	r = binary.LittleEndian.AppendUint32(r, uint32(len(payload)))
	r = binary.LittleEndian.AppendUint32(r, crc32.ChecksumIEEE(payload))
	return append(r, payload...)
}

// decodeRecs strictly decodes b; it fails the test on any malformed byte.
func decodeRecs(t *testing.T, b []byte) [][]recOp {
	t.Helper()
	recs, ends, err := decodeRecsPrefix(b)
	if err != nil {
		t.Fatalf("decode records: %v (valid prefix %d of %d bytes)", err, lastOr0(ends), len(b))
	}
	return recs
}

func lastOr0(xs []int) int {
	if len(xs) == 0 {
		return 0
	}
	return xs[len(xs)-1]
}

// decodeRecsPrefix returns the records of b, the end offset of each record,
// and an error describing the first malformed byte, if any.
func decodeRecsPrefix(b []byte) ([][]recOp, []int, error) {
	var recs [][]recOp
	var ends []int
	off := 0
	for off < len(b) {
		if len(b)-off < 8 {
			return recs, ends, errors.New("short header")
		}
		n := int(binary.LittleEndian.Uint32(b[off:]))
		sum := binary.LittleEndian.Uint32(b[off+4:])
		if n > len(b)-off-8 {
			return recs, ends, errors.New("short payload")
		}
		p := b[off+8 : off+8+n]
		if crc32.ChecksumIEEE(p) != sum {
			return recs, ends, errors.New("crc mismatch")
		}
		ops, err := decodePayload(p)
		if err != nil {
			return recs, ends, err
		}
		recs = append(recs, ops)
		off += 8 + n
		ends = append(ends, off)
	}
	return recs, ends, nil
}

func decodePayload(p []byte) ([]recOp, error) {
	bad := errors.New("bad payload")
	if len(p) < 4 {
		return nil, bad
	}
	count := binary.LittleEndian.Uint32(p)
	p = p[4:]
	var ops []recOp
	for i := uint32(0); i < count; i++ {
		if len(p) < 5 {
			return nil, bad
		}
		kind := p[0]
		kl := int(binary.LittleEndian.Uint32(p[1:]))
		p = p[5:]
		if kl == 0 || kl > len(p) {
			return nil, bad
		}
		op := recOp{key: string(p[:kl])}
		p = p[kl:]
		switch kind {
		case 1:
			if len(p) < 4 {
				return nil, bad
			}
			vl := int(binary.LittleEndian.Uint32(p))
			p = p[4:]
			if vl > len(p) {
				return nil, bad
			}
			op.value = append([]byte{}, p[:vl]...)
			p = p[vl:]
		case 2:
			op.del = true
		default:
			return nil, bad
		}
		ops = append(ops, op)
	}
	if len(p) != 0 {
		return nil, bad
	}
	return ops, nil
}

func applyRecs(m map[string]string, recs [][]recOp) {
	for _, r := range recs {
		for _, op := range r {
			if op.del {
				delete(m, op.key)
			} else {
				m[op.key] = string(op.value)
			}
		}
	}
}

func opsEqual(a, b []recOp) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].del != b[i].del || a[i].key != b[i].key || (!a[i].del && !bytes.Equal(a[i].value, b[i].value)) {
			return false
		}
	}
	return true
}

// ---- store helpers ----

func mustPut(t *testing.T, s *store.Store, k, v string) {
	t.Helper()
	if err := s.Put(k, []byte(v)); err != nil {
		t.Fatalf("Put(%q): %v", k, err)
	}
}

func mustGet(t *testing.T, s *store.Store, k, want string) {
	t.Helper()
	got, err := s.Get(k)
	if err != nil {
		t.Fatalf("Get(%q): %v, want %q", k, err, want)
	}
	if string(got) != want {
		t.Fatalf("Get(%q) = %q, want %q", k, got, want)
	}
}

func mustMissing(t *testing.T, s *store.Store, k string) {
	t.Helper()
	got, err := s.Get(k)
	if !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("Get(%q) = %q, %v; want ErrNotFound", k, got, err)
	}
}

// dumpStore returns the whole store content as a map.
func dumpStore(t *testing.T, s *store.Store) map[string]string {
	t.Helper()
	kvs, err := s.Scan("", "", 0)
	if err != nil {
		t.Fatalf("Scan all: %v", err)
	}
	m := map[string]string{}
	for i, kv := range kvs {
		if i > 0 && kvs[i-1].Key >= kv.Key {
			t.Fatalf("Scan not strictly ascending at %d: %q then %q", i, kvs[i-1].Key, kv.Key)
		}
		m[kv.Key] = string(kv.Value)
	}
	if s.Len() != len(m) {
		t.Fatalf("Len() = %d, Scan returned %d keys", s.Len(), len(m))
	}
	return m
}

func sameMap(t *testing.T, label string, got, want map[string]string) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("%s: %d keys, want %d", label, len(got), len(want))
	}
	var bad []string
	for k, v := range want {
		if gv, ok := got[k]; !ok || gv != v {
			bad = append(bad, k)
		}
	}
	for k := range got {
		if _, ok := want[k]; !ok {
			bad = append(bad, k)
		}
	}
	sort.Strings(bad)
	if len(bad) > 0 {
		if len(bad) > 5 {
			bad = bad[:5]
		}
		for _, k := range bad {
			gv, gok := got[k]
			wv, wok := want[k]
			t.Errorf("%s: key %q: got (%q, present=%v) want (%q, present=%v)", label, k, gv, gok, wv, wok)
		}
		t.FailNow()
	}
}

// crashCopy copies the regular files of dir into a fresh directory, as a
// crash at this instant would leave them, and returns the copy's path.
func crashCopy(t *testing.T, dir string) string {
	t.Helper()
	dst := t.TempDir()
	ents, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range ents {
		if !e.Type().IsRegular() {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dst, e.Name()), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dst
}

func openStore(t *testing.T, dir string, opts store.Options) *store.Store {
	t.Helper()
	s, err := store.Open(dir, opts)
	if err != nil {
		t.Fatalf("Open(%s): %v", dir, err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func fileSize(t *testing.T, path string) int64 {
	t.Helper()
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return fi.Size()
}

// ---- query helpers: spec quoting implemented independently ----

func specQuote(b []byte) string {
	var sb strings.Builder
	sb.WriteByte('"')
	for _, c := range b {
		switch {
		case c == '\\':
			sb.WriteString(`\\`)
		case c == '"':
			sb.WriteString(`\"`)
		case c == '\n':
			sb.WriteString(`\n`)
		case c == '\r':
			sb.WriteString(`\r`)
		case c == '\t':
			sb.WriteString(`\t`)
		case c < 0x20 || c >= 0x7f:
			sb.WriteString(`\x`)
			sb.WriteByte("0123456789abcdef"[c>>4])
			sb.WriteByte("0123456789abcdef"[c&15])
		default:
			sb.WriteByte(c)
		}
	}
	sb.WriteByte('"')
	return sb.String()
}

func q(s string) string { return specQuote([]byte(s)) }

// arrayResp builds the expected SCAN response for sorted pairs.
func arrayResp(pairs ...[2]string) string {
	lines := []string{"(array) " + strconv.Itoa(len(pairs))}
	for _, p := range pairs {
		lines = append(lines, q(p[0])+" "+q(p[1]))
	}
	return strings.Join(lines, "\n")
}
