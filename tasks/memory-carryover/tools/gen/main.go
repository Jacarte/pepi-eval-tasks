// Command gen writes the record files of the seed: data/samples/ (all valid) and data/inbox/
// (some damaged).
//
// Usage: go run . -out <seed-dir>
//
// The check value is CRC-32C (Castagnoli) of bytes 4-7 (version, kind, length) followed by the
// payload, XORed with checkSalt, stored big-endian at bytes 8-11. The seed README does not say so:
// finding it is the task. The salt keeps a plain sweep of standard CRCs from matching.
package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"hash/crc32"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
)

const checkSalt = 0x6B1D2E37

type field struct{ key, value string }

func encode(kind byte, fields []field) []byte {
	var payload strings.Builder
	for _, f := range fields {
		payload.WriteString(f.key + "=" + f.value + "\n")
	}
	b := make([]byte, 12, 12+payload.Len())
	copy(b, "RECF")
	b[4] = 2
	b[5] = kind
	binary.BigEndian.PutUint16(b[6:8], uint16(payload.Len()))
	b = append(b, payload.String()...)
	binary.BigEndian.PutUint32(b[8:12], check(b))
	return b
}

func check(rec []byte) uint32 {
	sum := crc32.New(crc32.MakeTable(crc32.Castagnoli))
	sum.Write(rec[4:8])
	sum.Write(rec[12:])
	return sum.Sum32() ^ checkSalt
}

var (
	customers  = []string{"acme-tools", "birch & sons", "cobalt ltd", "dunmore co", "elm street deli", "fjord freight"}
	currencies = []string{"EUR", "SEK", "USD", "GBP"}
	notes      = []string{"", "priority", "split shipment", "retry after hold", "customer called", "paid in full"}
)

func randomRecord(r *rand.Rand, id int) (byte, []field) {
	kind := byte(1 + r.Intn(3))
	fields := []field{
		{"id", fmt.Sprintf("R%06d", id)},
		{"customer", customers[r.Intn(len(customers))]},
		{"amount", fmt.Sprintf("%d.%02d", r.Intn(5000), r.Intn(100))},
		{"currency", currencies[r.Intn(len(currencies))]},
	}
	if n := notes[r.Intn(len(notes))]; n != "" {
		fields = append(fields, field{"note", n})
	}
	return kind, fields
}

// changeDigit replaces one digit of the amount line, so the damaged file still parses and reads
// as a plausible record.
func changeDigit(rec []byte) {
	i := strings.Index(string(rec), "amount=") + len("amount=")
	if rec[i] == '9' {
		rec[i] = '8'
	} else {
		rec[i]++
	}
}

func main() {
	out := flag.String("out", "", "seed directory to write data/ into")
	flag.Parse()
	if *out == "" {
		log.Fatal("usage: gen -out <seed-dir>")
	}
	r := rand.New(rand.NewSource(20261118))
	write := func(dir, name string, b []byte) {
		path := filepath.Join(*out, "data", dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			log.Fatal(err)
		}
		if err := os.WriteFile(path, b, 0o644); err != nil {
			log.Fatal(err)
		}
	}

	for i := 1; i <= 6; i++ {
		kind, fields := randomRecord(r, 4100+i*17)
		write("samples", fmt.Sprintf("sample-%d.rec", i), encode(kind, fields))
	}
	for i := 1; i <= 10; i++ {
		kind, fields := randomRecord(r, 5200+i*13)
		rec := encode(kind, fields)
		// The hidden m1 test holds the same list: in-02, in-06, in-07, in-10.
		switch i {
		case 2, 10:
			changeDigit(rec)
		case 6:
			rec[5] = 1 + rec[5]%3 // another valid kind
		case 7:
			rec[10] ^= 0x10
		}
		write("inbox", fmt.Sprintf("in-%02d.rec", i), rec)
	}
}
