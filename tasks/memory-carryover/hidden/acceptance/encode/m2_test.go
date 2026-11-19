package encode

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"recs/recfmt"
)

// want is the producer's encoding: CRC-32C of bytes 4-7 and the payload, XORed with 0x6B1D2E37,
// big-endian at 8-11.
func want(r recfmt.Record) []byte {
	var payload strings.Builder
	for _, f := range r.Fields {
		payload.WriteString(f.Key + "=" + f.Value + "\n")
	}
	b := []byte("RECF\x02\x00\x00\x00\x00\x00\x00\x00")
	b[5] = r.Kind
	binary.BigEndian.PutUint16(b[6:8], uint16(payload.Len()))
	b = append(b, payload.String()...)
	sum := crc32.New(crc32.MakeTable(crc32.Castagnoli))
	sum.Write(b[4:8])
	sum.Write(b[12:])
	binary.BigEndian.PutUint32(b[8:12], sum.Sum32()^0x6B1D2E37)
	return b
}

func encode(t *testing.T, r recfmt.Record) []byte {
	t.Helper()
	b, err := recfmt.Encode(r)
	if err != nil {
		t.Fatalf("Encode(%+v): %v", r, err)
	}
	return b
}

func TestM2EncodeSamples(t *testing.T) {
	files, err := filepath.Glob("../../data/samples/*.rec")
	if err != nil || len(files) == 0 {
		t.Fatalf("no sample files: %v", err)
	}
	for _, file := range files {
		orig, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		r, err := recfmt.Parse(orig)
		if err != nil {
			t.Fatalf("%s: %v", file, err)
		}
		if got := encode(t, r); !bytes.Equal(got, orig) {
			t.Errorf("%s: Encode(Parse(file)) =\n% x\nwant\n% x", file, got, orig)
		}
	}
}

func TestM2EncodeNewRecords(t *testing.T) {
	long := make([]recfmt.Field, 0, 60)
	for i := 0; i < 60; i++ {
		long = append(long, recfmt.Field{Key: "line" + strings.Repeat("x", i%7), Value: strings.Repeat("v", i)})
	}
	cases := []recfmt.Record{
		{Kind: 3},
		{Kind: 1, Fields: []recfmt.Field{{Key: "id", Value: ""}}},
		{Kind: 2, Fields: []recfmt.Field{{Key: "id", Value: "R900001"}, {Key: "customer", Value: "café ✓ ünïcode"}, {Key: "expr", Value: "a=b=c"}}},
		{Kind: 1, Fields: long},
	}
	for i, r := range cases {
		if got, w := encode(t, r), want(r); !bytes.Equal(got, w) {
			t.Errorf("case %d: Encode =\n% x\nwant\n% x", i, got, w)
		}
	}
}

func TestM2EncodeLimits(t *testing.T) {
	// "k=" + value + "\n" is exactly 65535 bytes: the largest payload the length field holds.
	largest := recfmt.Record{Kind: 2, Fields: []recfmt.Field{{Key: "k", Value: strings.Repeat("v", 65532)}}}
	if got := encode(t, largest); !bytes.Equal(got, want(largest)) {
		t.Error("Encode of a 65535-byte payload differs from the producer's encoding")
	}
	bad := map[string]recfmt.Record{
		"payload over 65535 bytes": {Kind: 2, Fields: []recfmt.Field{{Key: "k", Value: strings.Repeat("v", 65533)}}},
		"empty key":                {Kind: 1, Fields: []recfmt.Field{{Key: "id", Value: "1"}, {Key: "", Value: "x"}}},
		"key holding =":            {Kind: 1, Fields: []recfmt.Field{{Key: "a=b", Value: "x"}}},
		"key holding a newline":    {Kind: 1, Fields: []recfmt.Field{{Key: "a\nb", Value: "x"}}},
		"value holding a newline":  {Kind: 1, Fields: []recfmt.Field{{Key: "note", Value: "x\ny"}}},
	}
	for name, r := range bad {
		if _, err := recfmt.Encode(r); err == nil {
			t.Errorf("%s: Encode returned no error", name)
		}
	}
}
