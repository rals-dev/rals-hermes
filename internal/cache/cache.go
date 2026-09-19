// Package cache is a small TTL cache with request coalescing.
//
// It exists to absorb UI polling: many requests for the same upstream
// resource within the TTL cost Hermes exactly one call. Concurrent misses on
// the same key are merged through singleflight so a burst never fans out.
// Errors are never cached — a failing upstream is retried on the next
// request, which keeps the "profile unreachable" state fresh.
package cache

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/singleflight"
)

// Loader fetches a value on a cache miss.
type Loader[V any] func(ctx context.Context) (V, error)

// Cache stores values of one type under string keys for a fixed TTL.
type Cache[V any] struct {
	ttl time.Duration
	now func() time.Time

	mu      sync.RWMutex
	entries map[string]entry[V]
	sf      singleflight.Group

	hits   atomic.Uint64
	misses atomic.Uint64
}

type entry[V any] struct {
	value   V
	expires time.Time
}

// sweepThreshold is the entry count above which a store also evicts expired
// entries. Keys include session ids, so the set is unbounded without this.
const sweepThreshold = 256

// New returns an empty cache whose entries live for ttl.
func New[V any](ttl time.Duration) *Cache[V] {
	return &Cache[V]{ttl: ttl, now: time.Now, entries: make(map[string]entry[V])}
}

// Get returns the cached value for key, or loads it. hit reports whether the
// value came from the cache. Concurrent callers missing on the same key share
// one load; if that load fails, every waiter gets the error and nothing is
// stored.
func (c *Cache[V]) Get(ctx context.Context, key string, load Loader[V]) (v V, hit bool, err error) {
	if v, ok := c.lookup(key); ok {
		c.hits.Add(1)
		return v, true, nil
	}
	c.misses.Add(1)

	res, err, _ := c.sf.Do(key, func() (any, error) {
		// Re-check under singleflight: a previous flight may have just stored
		// the value while we were queued.
		if v, ok := c.lookup(key); ok {
			return v, nil
		}
		v, err := load(ctx)
		if err != nil {
			return nil, err
		}
		c.store(key, v)
		return v, nil
	})
	if err != nil {
		var zero V
		return zero, false, err
	}
	return res.(V), false, nil //nolint:forcetypeassert // always V: we stored it
}

// Len returns the number of stored entries, expired or not.
func (c *Cache[V]) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

func (c *Cache[V]) store(key string, v V) {
	now := c.now()
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.entries) >= sweepThreshold {
		for k, e := range c.entries {
			if !now.Before(e.expires) {
				delete(c.entries, k)
			}
		}
	}
	c.entries[key] = entry[V]{value: v, expires: now.Add(c.ttl)}
}

// Stats returns the hit and miss counters since start.
func (c *Cache[V]) Stats() (hits, misses uint64) {
	return c.hits.Load(), c.misses.Load()
}

func (c *Cache[V]) lookup(key string) (V, bool) {
	c.mu.RLock()
	e, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok || !c.now().Before(e.expires) {
		var zero V
		return zero, false
	}
	return e.value, true
}
