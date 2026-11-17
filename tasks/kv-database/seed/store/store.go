// Package store is the ordered key-value store (SPEC.md, milestones M1 and M4).
package store

import "errors"

var (
	ErrNotFound = errors.New("store: key not found")
	ErrEmptyKey = errors.New("store: empty key")
	ErrClosed   = errors.New("store: closed")
)

var errNotImplemented = errors.New("store: not implemented")

// KV is one key/value pair returned by a scan.
type KV struct {
	Key   string
	Value []byte
}

// Op is one write in a batch passed to Apply.
// Delete == true means delete Key (Value is ignored); otherwise put Key=Value.
type Op struct {
	Key    string
	Value  []byte
	Delete bool
}

// SyncPolicy selects when the WAL is fsynced.
type SyncPolicy int

const (
	SyncAlways SyncPolicy = iota // fsync the WAL before every write returns
	SyncNever                    // never fsync the WAL; the OS flushes eventually
)

// Options configures a durable store opened with Open.
type Options struct {
	Sync SyncPolicy
	// CompactThreshold: when the WAL file grows beyond this many bytes after
	// a write, the store compacts automatically before the write returns.
	// 0 means DefaultCompactThreshold; a negative value disables
	// automatic compaction (Compact can still be called).
	CompactThreshold int64
}

const (
	WALFileName             = "wal.log"
	SnapshotFileName        = "snapshot.db"
	DefaultCompactThreshold = 4 << 20
)

// Store is an ordered key-value store.
type Store struct{}

// New returns an empty in-memory store.
func New() *Store { return &Store{} }

// Open opens (or creates) a durable store in dir.
func Open(dir string, opts Options) (*Store, error) { return nil, errNotImplemented }

func (s *Store) Put(key string, value []byte) error { return errNotImplemented }

func (s *Store) Get(key string) ([]byte, error) { return nil, errNotImplemented }

func (s *Store) Delete(key string) error { return errNotImplemented }

func (s *Store) Apply(ops []Op) error { return errNotImplemented }

func (s *Store) Scan(start, end string, limit int) ([]KV, error) { return nil, errNotImplemented }

func (s *Store) ScanPrefix(prefix string, limit int) ([]KV, error) {
	return nil, errNotImplemented
}

func (s *Store) Len() int { return 0 }

func (s *Store) Compact() error { return errNotImplemented }

func (s *Store) Close() error { return errNotImplemented }
