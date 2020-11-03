package easy_lru_cache

import (
	"fmt"
	"sync"
	"time"
)

const (
	// Default2QRecentRatio is the ratio of the 2Q cache dedicated
	// to recently added entries that have only been accessed once.
	Default2QRecentRatio = 0.20

	// Default2QGhostEntries is the default ratio of ghost
	// entries kept to track entries recently evicted
	Default2QGhostEntries = 0.80
)

// twoQueueCache is a thread-safe 2Q cache.
// https://medium.com/@koushikmohan/an-analysis-of-2q-cache-replacement-algorithms-21acceae672a
type twoQueueCache struct {
	size       int
	recentSize int

	recent      LRUCache
	frequent    LRUCache
	recentEvict LRUCache
	lock        sync.RWMutex
	expiration  *time.Time
}

// New2Q creates a new twoQueueCache using the default
// values for the parameters.
func New2Q(size int, recentRation, ghostEntries float64) (*twoQueueCache, error) {
	if recentRation == 0.0 {
		recentRation = Default2QRecentRatio
	}
	if ghostEntries == 0.0 {
		ghostEntries = Default2QGhostEntries
	}
	return New2QParams(size, recentRation, ghostEntries)
}

// New2QParams creates a new twoQueueCache using the provided
// parameter values.
func New2QParams(size int, recentRatio, ghostRatio float64) (cache *twoQueueCache, err error) {
	if size <= 0 {
		err = fmt.Errorf("invalid size")
		return
	}
	if recentRatio < 0.0 || recentRatio > 1.0 {
		err = fmt.Errorf("invalid recent ratio")
		return
	}
	if ghostRatio < 0.0 || ghostRatio > 1.0 {
		err = fmt.Errorf("invalid ghost ratio")
		return
	}

	// Determine the sub-sizes
	recentSize := int(float64(size) * recentRatio)
	evictSize := int(float64(size) * ghostRatio)

	// Allocate the LRUs
	recent, err := NewLRU(size)
	if err != nil {
		return
	}
	frequent, err := NewLRU(size)
	if err != nil {
		return
	}
	recentEvict, err := NewLRU(evictSize)
	if err != nil {
		return
	}

	// Initialize the cache
	cache = &twoQueueCache{
		size:        size,
		recentSize:  recentSize,
		recent:      recent,
		frequent:    frequent,
		recentEvict: recentEvict,
	}
	return
}

// Get looks up a key's value from the cache.
func (t *twoQueueCache) Get(key interface{}) (value interface{}, err error) {
	if value, err = t.frequent.Get(key); err == nil {
		return
	}

	// If the value is contained in recent, then we
	// promote it to frequent
	if value, ok := t.recent.Peek(key); ok {
		t.recent.Remove(key)
		err = t.frequent.Put(key, value)
	}
	return
}

// Add adds a value to the cache.
func (t *twoQueueCache) Add(key, value interface{}) (err error) {
	t.lock.Lock()
	defer t.lock.Unlock()

	// Check if the value is frequently used already,
	// and just update the value
	if t.frequent.Contains(key) {
		err = t.frequent.Put(key, value)
		return
	}

	// Check if the value is recently used, and promote
	// the value into the frequent list
	if t.recent.Contains(key) {
		t.recent.Remove(key)
		err = t.frequent.Put(key, value)
		return
	}

	// If the value was recently evicted, add it to the
	// frequently used list
	if t.recentEvict.Contains(key) {
		err = t.ensureSpace(true)
		if err != nil {
			return
		}
		t.recentEvict.Remove(key)
		err = t.frequent.Put(key, value)
		return
	}

	// Add to the recently seen list
	err = t.ensureSpace(false)
	if err != nil {
		return
	}
	err = t.recent.Put(key, value)
	return
}

// ensureSpace is used to ensure we have space in the cache
func (t *twoQueueCache) ensureSpace(recentEvict bool) (err error) {
	// If we have space, nothing to do
	recentLen := t.recent.Len()
	freqLen := t.frequent.Len()
	if recentLen+freqLen < t.size {
		return
	}

	// If the recent buffer is larger than
	// the target, evict from there
	if recentLen > 0 && (recentLen > t.recentSize || (recentLen == t.recentSize && !recentEvict)) {
		err = t.recent.RemoveOldest()
		return
	}

	// Remove from the frequent list otherwise
	err = t.frequent.RemoveOldest()
	return
}

// Len returns the number of items in the cache.
func (t *twoQueueCache) Len() int {
	t.lock.RLock()
	defer t.lock.RUnlock()
	return t.recent.Len() + t.frequent.Len()
}

// Keys returns a slice of the keys in the cache.
// The frequently used keys are first in the returned slice.
func (t *twoQueueCache) Keys() []interface{} {
	t.lock.RLock()
	defer t.lock.RUnlock()
	k1 := t.frequent.Keys()
	k2 := t.recent.Keys()
	return append(k1, k2...)
}

// Remove removes the provided key from the cache.
func (t *twoQueueCache) Remove(key interface{}) {
	t.lock.Lock()
	defer t.lock.Unlock()
	if t.frequent.Remove(key) {
		return
	}
	if t.recent.Remove(key) {
		return
	}
	if t.recentEvict.Remove(key) {
		return
	}
}

// Purge is used to completely clear the cache.
func (t *twoQueueCache) Purge() {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.recent.Purge()
	t.frequent.Purge()
	t.recentEvict.Purge()
}

// Contains is used to check if the cache contains a key
// without updating recency or frequency.
func (t *twoQueueCache) Contains(key interface{}) bool {
	t.lock.RLock()
	defer t.lock.RUnlock()
	return t.frequent.Contains(key) || t.recent.Contains(key)
}

// Peek is used to inspect the cache value of a key
// without updating recency or frequency.
func (t *twoQueueCache) Peek(key interface{}) (value interface{}, ok bool) {
	t.lock.RLock()
	defer t.lock.RUnlock()
	if val, ok := t.frequent.Peek(key); ok {
		return val, ok
	}
	return t.recent.Peek(key)
}
