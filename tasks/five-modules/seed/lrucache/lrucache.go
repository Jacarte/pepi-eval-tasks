// Package lrucache is a generic, concurrency-safe LRU cache with TTL and
// weight limits. See SPEC.md section 5.
package lrucache

import "time"

// Clock supplies the current time.
type Clock interface {
	Now() time.Time
}

// EvictionReason says why an entry left the cache.
type EvictionReason int

const (
	ReasonCapacity EvictionReason = iota + 1 // removed to satisfy MaxEntries/MaxBytes
	ReasonExpired                            // TTL elapsed
	ReasonDeleted                            // Delete or Purge
	ReasonReplaced                           // overwritten by Set
)

// Options configures a Cache.
type Options[K comparable, V any] struct {
	MaxEntries int                        // <= 0: unlimited
	MaxBytes   int64                      // <= 0: unlimited
	Weigher    func(key K, value V) int64 // nil: every entry weighs 1; negative results count as 0
	TTL        time.Duration              // default TTL for Set; <= 0: no expiry
	Clock      Clock                      // nil: the system clock
	OnEvict    func(key K, value V, reason EvictionReason)
}

// Stats are cumulative counters.
type Stats struct {
	Hits, Misses, Evictions, Expirations uint64
}

// Cache is a generic LRU cache; create it with New.
type Cache[K comparable, V any] struct{}

// New creates a cache.
func New[K comparable, V any](opts Options[K, V]) *Cache[K, V] { return &Cache[K, V]{} }

// Set stores key with the default TTL.
func (c *Cache[K, V]) Set(key K, value V) {}

// SetWithTTL stores key with the given TTL; ttl <= 0 means no expiry.
func (c *Cache[K, V]) SetWithTTL(key K, value V, ttl time.Duration) {}

// Get returns the value for key and marks it most recently used.
func (c *Cache[K, V]) Get(key K) (V, bool) {
	var zero V
	return zero, false
}

// Peek returns the value for key without changing recency, stats or contents.
func (c *Cache[K, V]) Peek(key K) (V, bool) {
	var zero V
	return zero, false
}

// Delete removes key and reports whether a live entry was removed.
func (c *Cache[K, V]) Delete(key K) bool { return false }

// Len returns the number of stored entries, including expired ones not yet removed.
func (c *Cache[K, V]) Len() int { return 0 }

// Bytes returns the total weight of stored entries.
func (c *Cache[K, V]) Bytes() int64 { return 0 }

// Keys returns unexpired keys from most to least recently used.
func (c *Cache[K, V]) Keys() []K { return nil }

// RemoveExpired removes all expired entries and returns how many were removed.
func (c *Cache[K, V]) RemoveExpired() int { return 0 }

// Purge removes every entry.
func (c *Cache[K, V]) Purge() {}

// Stats returns a snapshot of the counters.
func (c *Cache[K, V]) Stats() Stats { return Stats{} }
