package cache

import (
	"sync"
	"time"
)

const (
	// NoExpiryDuration ...
	NoExpiryDuration time.Duration = -1
	// DefaultExpiryDuration ...
	DefaultExpiryDuration time.Duration = 0
)

// OnEvictionFunc ...
type OnEvictionFunc func(k string, v interface{})

// Item ...
type Item struct {
	Object     interface{}
	ExpiryNano int64
}

// Expired ...
func (item Item) Expired() bool {
	if item.ExpiryNano == 0 {
		return false
	}
	return time.Now().UnixNano() > item.ExpiryNano
}

// Cache ...
type Cache struct {
	defaultExpiry time.Duration
	itemsMap      map[string]Item
	mutex         sync.RWMutex
	onEviction    OnEvictionFunc
}

// Set ...
func (cache *Cache) Set(k string, v interface{}, expiryDuration time.Duration) {
	var expiry int64

	if expiryDuration == DefaultExpiryDuration {
		expiryDuration = cache.defaultExpiry
	}

	if expiryDuration > 0 {
		expiry = time.Now().Add(expiryDuration).UnixNano()
	}

	item := Item{
		Object: v, ExpiryNano: expiry,
	}

	cache.mutex.Lock()
	cache.itemsMap[k] = item

	cache.mutex.Unlock()
}

// Get ...
func (cache *Cache) Get(k string) (interface{}, bool) {
	cache.mutex.RLock()
	item, found := cache.itemsMap[k]

	// Return not found when item isn't there or
	// expired already
	if !found || item.Expired() {
		cache.mutex.RUnlock()
		return nil, found
	}

	cache.mutex.RUnlock()
	return item, found
}

// Delete ...
func (cache *Cache) Delete(k string) {
	cache.mutex.Lock()
	if item, found := cache.itemsMap[k]; found {
		delete(cache.itemsMap, k)
		if cache.onEviction != nil {
			cache.onEviction(k, item)
		}
	}
	cache.mutex.Unlock()
}


