// Package recfmt reads .rec record files (see README.md).
package recfmt

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
)

// HeaderSize is the size of the fixed header before the payload.
const HeaderSize = 12

// Record is one decoded .rec file.
type Record struct {
	Kind   uint8
	Fields []Field
}

// Field is one key=value line of the payload.
type Field struct {
	Key   string
	Value string
}

// Parse decodes the layout of a .rec file. It does not verify the check value.
func Parse(b []byte) (Record, error) {
	if len(b) < HeaderSize {
		return Record{}, errors.New("recfmt: short header")
	}
	if string(b[0:4]) != "RECF" {
		return Record{}, errors.New("recfmt: bad magic")
	}
	if b[4] != 2 {
		return Record{}, fmt.Errorf("recfmt: unsupported version %d", b[4])
	}
	n := int(binary.BigEndian.Uint16(b[6:8]))
	if len(b) != HeaderSize+n {
		return Record{}, fmt.Errorf("recfmt: payload length %d, file holds %d bytes", n, len(b)-HeaderSize)
	}
	rec := Record{Kind: b[5]}
	payload := string(b[HeaderSize:])
	if payload != "" && !strings.HasSuffix(payload, "\n") {
		return Record{}, errors.New("recfmt: payload does not end in a newline")
	}
	for _, line := range strings.SplitAfter(payload, "\n") {
		if line == "" {
			continue
		}
		key, value, ok := strings.Cut(strings.TrimSuffix(line, "\n"), "=")
		if !ok || key == "" {
			return Record{}, fmt.Errorf("recfmt: bad payload line %q", line)
		}
		rec.Fields = append(rec.Fields, Field{Key: key, Value: value})
	}
	return rec, nil
}
