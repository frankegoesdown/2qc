package easy_lru_cache

import (
	"container/list"
	"errors"
	"sync"
	//"time"
)

// Key is any value which is comparable.
// See http://golang.org/ref/spec#Comparison_operators for details.
type Key interface{}

// Value is any value.
type Value interface{}

type LRUCache interface {
	Get(key interface{}) (value Value, err error)
	Put(key, value interface{}) (err error)
	Contains(key interface{}) (ok bool)
	Peek(key interface{}) (value interface{}, ok bool)
	Remove(key interface{}) (ok bool)
	Len() int
	RemoveOldest() (err error)
	Keys() []interface{}
	Purge()
}

type lruItem struct {
	key   Key
	value Value

	//expiration *time.Time
}

// lru implements a non-thread safe fixed size lru cache
type lru struct {
	capacity int
	list     *list.List
	cache    map[interface{}]*list.Element
	lock     sync.RWMutex
}

func (l *lru) Get(key interface{}) (value Value, err error) {
	l.lock.RLock()
	defer l.lock.Unlock()
	element, has := l.cache[key]
	if !has {
		err = errors.New("not found in cache")
		return
	}
	l.list.MoveBefore(element, l.list.Front())
	value = element.Value.(*lruItem).value
	return
}

func (l *lru) Put(key, value interface{}) (err error) {
	var item *lruItem
	l.lock.Lock()
	defer l.lock.Unlock()
	if it, ok := l.cache[key]; ok {
		l.list.MoveToFront(it)
		item = it.Value.(*lruItem)
		item.value = value
		return
	}

	if l.list.Len() >= l.capacity {
		l.removeOldest()
	}
	item = &lruItem{
		key:   key,
		value: value,
	}
	l.cache[key] = l.list.PushFront(item)
	return
}

// Peek returns the key value (or undefined if not found) without updating
// the "recently used"-ness of the key.
func (l *lru) Peek(key interface{}) (value interface{}, ok bool) {
	if element, ok := l.cache[key]; ok {
		value = element.Value.(*lruItem).value
	}
	return
}

// Contains checks if a key is in the cache, without updating the recent-ness
// or deleting it for being stale.
func (l *lru) Contains(key interface{}) (ok bool) {
	_, ok = l.cache[key]
	return ok
}

// Remove removes the provided key from the cache, returning if the
// key was contained.
func (l *lru) Remove(key interface{}) (ok bool) {
	if element, ok := l.cache[key]; ok {
		l.removeElement(element)
	}
	return
}

// Len returns the number of items in the cache.
func (l *lru) Len() int {
	return l.list.Len()
}

// RemoveOldest removes the oldest item from the cache.
func (l *lru) RemoveOldest() (err error) {
	ent := l.list.Back()
	if ent != nil {
		l.removeElement(ent)
	}
	return
}

// Keys returns a slice of the keys in the cache, from oldest to newest.
func (l *lru) Keys() []interface{} {
	keys := make([]interface{}, len(l.cache))
	i := 0
	for ent := l.list.Back(); ent != nil; ent = ent.Prev() {
		keys[i] = ent.Value.(*lruItem).key
		i++
	}
	return keys
}

// Purge is used to completely clear the cache.
func (l *lru) Purge() {
	for k, _ := range l.cache {
		delete(l.cache, k)
	}
	l.list.Init()
}

// removeOldest removes the oldest item from the cache.
func (l *lru) removeOldest() {
	ent := l.list.Back()
	if ent != nil {
		l.removeElement(ent)
	}
}

func (l *lru) removeElement(e *list.Element) {
	l.list.Remove(e)
	entry := e.Value.(*lruItem)
	delete(l.cache, entry.key)
}

// NewLRU constructs an lru of the given size
func NewLRU(capacity int) (*lru, error) {
	if capacity <= 0 {
		return nil, errors.New("must provide a positive size")
	}
	c := &lru{
		capacity: capacity,
		list:     list.New(),
		cache:    make(map[interface{}]*list.Element),
	}
	return c, nil
}
