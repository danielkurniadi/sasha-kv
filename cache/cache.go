package cache

import (
	"encoding/gob"
	"fmt"
	"io"
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
	// expiryJanitor
}

// Put ...
func (cache *Cache) Put(k string, v interface{}, expiryDuration time.Duration) {
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

// OnEvicted sets an (optional) function that will be called with the key, value input
// when a delete operation is called and item is evicted successfully from the cache.
// However, OnEvicted() will not be call for overriding values with same key.
// Set to nil to disable, otherwise pass a function for on-evicted callback.
func (cache *Cache) OnEvicted(f OnEvictionFunc) {
	cache.mutex.Lock()
	cache.onEviction = f
	cache.mutex.Unlock()
}

// Len gives the number of items in the cache
// similar to len([]items)
func (cache *Cache) Len() int {
	cache.mutex.RLock()
	defer cache.mutex.RUnlock()
	return len(cache.itemsMap)
}

// Save ...
func (cache *Cache) Save(w io.Writer) (err error) {
	encoder := gob.NewEncoder(w)
	defer func() {
		if x := recover(); x != nil {
			err = fmt.Errorf("error cache.Save() registering item types to disk")
		}
	}()
	cache.mutex.Lock()

	// TODO: if item types are predefined, e.g. string or int only
	// then this could hurt performance quite a bit
	for _, v := range cache.itemsMap {
		if !v.Expired() {
			gob.Register(v.Object)
		}
	}
	err = encoder.Encode(&cache.itemsMap)
	return
}

// Load ...
func (cache *Cache) Load(r io.Reader) error {
	decoder := gob.NewDecoder(r)
	items := make(map[string]Item)

	err := decoder.Decode(&items)
	if err != nil {
		return err
	}

	cache.mutex.Lock()
	defer cache.mutex.Unlock()

	for k, v := range items {
		object, exist := cache.itemsMap[k]
		if !exist || object.Expired() {
			cache.itemsMap[k] = v
		}
	}

	return nil
}

// Flush delete all items from the cache regardless
// expiration deadline. It startover a with a new empty cache.
func (cache *Cache) Flush() {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	cache.itemsMap = make(map[string]Item)
}
