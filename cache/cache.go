package cache

import (
	"encoding/gob"
	"fmt"
	"io"
	"sync"
	"time"
)

const (
	// NoExpiry ...
	NoExpiry time.Duration = -1
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
	if item.ExpiryNano == 0 { // no expiry
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
func (cache *Cache) Put(k string, v interface{}) {
	var expiry int64

	// Using default expiry if cache.defaultExpiry is set > 0
	// otherwise item will never be expired
	if cache.defaultExpiry > 0 {
		expiry = time.Now().Add(cache.defaultExpiry).UnixNano()
	}

	item := Item{
		Object: v, ExpiryNano: expiry,
	}

	cache.mutex.Lock()
	cache.itemsMap[k] = item

	cache.mutex.Unlock()
}

// PutExpiry ...
func (cache *Cache) PutExpiry(k string, v interface{}, expiryDuration time.Duration) {
	var expiry int64

	// Set expiryDuration to -1 for no expiry time
	// Set expiryDuration as positive int (sec) for setting expiry time
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
		return nil, false
	}

	cache.mutex.RUnlock()
	return item.Object, true
}

// Delete ...
func (cache *Cache) Delete(k string) bool {
	cache.mutex.Lock()
	item, found := cache.itemsMap[k]
	if found {
		delete(cache.itemsMap, k)
		if cache.onEviction != nil {
			cache.onEviction(k, item)
		}
	}
	cache.mutex.Unlock()
	return found
}

type keyvaluePair struct {
	Key   string
	Value interface{}
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
		_, exist := cache.itemsMap[k]
		if !exist {
			// We don't ignore expired items here
			// to respect no cleanup option.
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

// New returns a new cache with a given default expiry duration and cleanup
// interval. If itemExpiry duration is less than 1 (e.g. -1) it will never
// expire.
//
// The cache will also have a clean-up mechanism that polls on fixed interval
// and run cache cleanup to find and delete expired items.
// If the cleanup interval is less than one, expired items are not deleted
// from cache before calling cache.DeleteExpired().
func New(defaultExpiry, saveInterval, cleanupInterval time.Duration) *Cache {
	itemsMap := make(map[string]Item)
	cache := &Cache{
		defaultExpiry: defaultExpiry,
		itemsMap:      itemsMap,
		mutex:         sync.RWMutex{},
	}
	return cache
}

// NewFrom loads the previously saved cache and returns a cache that uses
// previously saved key-value data.
func NewFrom(defaultExpiry, cleanupInterval time.Duration, itemsMap map[string]Item) *Cache {
	cache := &Cache{
		defaultExpiry: defaultExpiry,
		itemsMap:      itemsMap,
		mutex:         sync.RWMutex{},
	}
	return cache
}
