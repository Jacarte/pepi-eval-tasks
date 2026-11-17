// Package ids generates record identifiers.
package ids

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
)

// Generator returns a fresh id with the given prefix, e.g. "ord-1f2e...".
type Generator interface {
	Next(prefix string) string
}

// Random generates prefix-<16 hex chars> ids from crypto/rand.
type Random struct{}

func (Random) Next(prefix string) string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic("ids: crypto/rand failed: " + err.Error())
	}
	return prefix + "-" + hex.EncodeToString(b[:])
}

// Sequence generates prefix-000001, prefix-000002, ... per prefix.
// Tests use it to get predictable ids.
type Sequence struct {
	mu sync.Mutex
	n  map[string]int
}

func NewSequence() *Sequence { return &Sequence{n: make(map[string]int)} }

func (s *Sequence) Next(prefix string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.n[prefix]++
	return fmt.Sprintf("%s-%06d", prefix, s.n[prefix])
}

// OrRandom returns g, or Random when g is nil.
func OrRandom(g Generator) Generator {
	if g == nil {
		return Random{}
	}
	return g
}

// HasPrefix reports whether id was minted with prefix.
func HasPrefix(id, prefix string) bool {
	return strings.HasPrefix(id, prefix+"-") && len(id) > len(prefix)+1
}
