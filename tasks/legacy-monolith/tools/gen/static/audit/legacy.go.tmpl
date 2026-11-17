package audit

import (
	"sync"
	"time"
)

// LegacyEntry is a row of the pre-2020 audit trail.
type LegacyEntry struct {
	Name   string
	Fields map[string]string
	At     time.Time
}

var legacy struct {
	sync.Mutex
	entries []LegacyEntry
}

// Log appends to the process-wide legacy trail. kv is a flat list of
// key/value pairs; an odd trailing key is dropped.
//
// Deprecated: Log ignores the request context and cannot report failures.
// Use a Recorder.
func Log(name string, kv ...string) {
	fields := make(map[string]string, len(kv)/2)
	for i := 0; i+1 < len(kv); i += 2 {
		fields[kv[i]] = kv[i+1]
	}
	legacy.Lock()
	legacy.entries = append(legacy.entries, LegacyEntry{Name: name, Fields: fields, At: time.Now().UTC()})
	legacy.Unlock()
}

// LegacyEntries returns a copy of the legacy trail.
//
// Deprecated: only the v1 admin export reads this.
func LegacyEntries() []LegacyEntry {
	legacy.Lock()
	defer legacy.Unlock()
	return append([]LegacyEntry(nil), legacy.entries...)
}
