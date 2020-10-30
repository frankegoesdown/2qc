package lru

import (
	"container/list"
	"errors"
	"sync"
	"time"
)

// Key is any value which is comparable.
// See http://golang.org/ref/spec#Comparison_operators for details.
type Key interface{}

// Value is any value.
type Value interface{}

type lruItem struct {
	key        Key
	value      Value
	expiration *time.Time
}

// LRU implements a non-thread safe fixed size LRU cache
type LRU struct {
	capacity int
	list     *list.List
	cache    map[interface{}]*list.Element
	mu       sync.RWMutex
}

func (l *LRU) Get(key interface{}) (value Value, err error) {
	element, has := l.cache[key]
	if !has {
		err = errors.New("not found in cache")
		return
	}
	l.list.MoveBefore(element, l.list.Front())
	value = element.Value.(*lruItem).value
	return
}

func (l *LRU) Put(key, value interface{}) (err error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	var item *lruItem
	if it, ok := l.cache[key]; ok {
		l.list.MoveToFront(it)
		item = it.Value.(*lruItem)
		item.value = value
	} else {
		// Verify size not exceeded
		if l.list.Len() >= l.capacity {
			l.removeOldest()
		}
		item = &lruItem{
			key:   key,
			value: value,
		}
		l.cache[key] = l.list.PushFront(item)
	}

	return
	//_, err = l.set(key, value)
	//return err
}

// removeOldest removes the oldest item from the cache.
func (l *LRU) removeOldest() {
	ent := l.list.Back()
	if ent != nil {
		l.removeElement(ent)
	}
}

func (l *LRU) removeElement(e *list.Element) {
	l.list.Remove(e)
	entry := e.Value.(*lruItem)
	delete(l.cache, entry.key)
}

// NewLRU constructs an LRU of the given size
func NewLRU(capacity int) (*LRU, error) {
	if capacity <= 0 {
		return nil, errors.New("must provide a positive size")
	}
	c := &LRU{
		capacity: capacity,
		list:     list.New(),
		cache:    make(map[interface{}]*list.Element),
	}
	return c, nil
}
