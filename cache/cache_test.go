package cache

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestCachePut(t *testing.T) {
	var testDefaultExpiry time.Duration = -1

	// test without expiry, cleanup, or backup save
	tcache := New(testDefaultExpiry, -1, -1)

	// test put/set key value successful
	testKeyVal := []keyvaluePair{
		{Key: "aa", Value: 10},
		{Key: "bb", Value: int32(32)},
		{Key: "cc", Value: float64(64.6464)},
		{Key: "dd", Value: []byte("{\"HelloWorld\": {\"array\":[\"abc\", \"abc\", \"abc\"]}}")},
		{Key: "ee", Value: "HelloWorld"},
	}

	for _, tt := range testKeyVal {
		name := fmt.Sprintf("PutShouldSuccess: %v", tt)
		t.Run(name, func(t *testing.T) {
			tcache.Put(tt.Key, tt.Value)
		})
	}
}

func TestCacheGet(t *testing.T) {
	var testDefaultExpiry time.Duration = -1

	// test without expiry, cleanup, or backup save
	tcache := New(testDefaultExpiry, -1, -1)

	// test cases for getting key-values
	testKeyVal := []keyvaluePair{
		{Key: "aa", Value: 10},
		{Key: "bb", Value: int32(32)},
		{Key: "cc", Value: float64(64.6464)},
		{Key: "dd", Value: []byte("{\"HelloWorld\": {\"array\":[\"abc\", \"abc\", \"abc\"]}}")},
		{Key: "ee", Value: "HelloWorld"},
	}

	for _, tt := range testKeyVal {
		tcache.Put(tt.Key, tt.Value)
	}

	// test getting key-values that should be found
	for _, tt := range testKeyVal {
		name := "GetShouldSuccess: " + tt.Key
		t.Run(name, func(t *testing.T) {
			x, found := tcache.Get(tt.Key)

			if !found || x == nil {
				t.Errorf("test cache.Get() wrong result, \"%s\" key-val not found", tt.Key)
			} else if reflect.TypeOf(x) != reflect.TypeOf(tt.Value) {
				t.Errorf("test cache.Get() wrong value type for key \"%s\". Expected: %T, Get: %T", tt.Key, tt.Value, x)
			} else if !cmp.Equal(x, tt.Value) {
				t.Errorf("test cache.Get() wrong value type for key \"%s\". Expected: %v, Get: %v", tt.Key, tt.Value, x)
			}
		})
	}

	// test getting key-values that never been put, should not found
	testKeysNotPut := []string{"zz", "yy", "ww", "gg"}
	for _, key := range testKeysNotPut {
		name := "GetShouldNotFound: " + key
		t.Run(name, func(t *testing.T) {
			x, found := tcache.Get(key)
			if found {
				t.Errorf("test cache.Get() key %s has never been put but found: %v", key, x)
			}
		})
	}
}

func TestItemExpiry(t *testing.T) {
	expiry := 100 * time.Millisecond
	timer := time.NewTimer(expiry)

	item := Item{
		Object:     1,
		ExpiryNano: time.Now().Add(expiry).UnixNano(),
	}

	<-timer.C

	if !item.Expired() {
		t.Errorf("test item.Expiry() fail: item should be expired by now")
	}
}

func TestCacheDelete(t *testing.T) {
	var testDefaultExpiry time.Duration = -1

	// test without expiry, cleanup, or backup save
	tcache := New(testDefaultExpiry, -1, -1)

	// test cases for deleting entry
	testKeyVal := []keyvaluePair{
		{Key: "aa", Value: 10},
		{Key: "bb", Value: int32(32)},
		{Key: "cc", Value: float64(64.6464)},
		{Key: "dd", Value: []byte("{\"HelloWorld\": {\"array\":[\"abc\", \"abc\", \"abc\"]}}")},
		{Key: "ee", Value: "HelloWorld"},
	}

	for _, tt := range testKeyVal {
		tcache.Put(tt.Key, tt.Value)
	}

	for _, tt := range testKeyVal {
		name := "DeleteShouldEvict: " + tt.Key
		t.Run(name, func(t *testing.T) {
			found := tcache.Delete(tt.Key)
			if !found {
				t.Errorf("test cache.Delete() wrong result, key \"%s\" not found", tt.Key)
			}
		})
	}
}

func TestCachePutExpiry(t *testing.T) {
	var testDefaultExpiry time.Duration = -1

	// test without cleanup, or backup save
	tcache := New(testDefaultExpiry, -1, -1)

	// test put/set key value successful
	testKeyVal := []keyvaluePair{
		{Key: "aa", Value: 10},
		{Key: "bb", Value: 12},
	}

	expiry := 80 * time.Millisecond
	expiredTime := time.Now().Add(expiry)

	for _, tt := range testKeyVal {
		tcache.PutExpiry(tt.Key, tt.Value, time.Until(expiredTime))

		if _, found := tcache.Get(tt.Key); !found {
			t.Errorf("test cache.PutExpiry() at setup test, fail to perform PUT")
			t.FailNow()
		}
	}

	// blocks until timer of 20 ms awake
	<-time.After(20 * time.Millisecond)

	t.Run("PutShouldNotExpiredYet", func(t *testing.T) {
		for _, tt := range testKeyVal {
			_, found := tcache.Get(tt.Key)

			if !found {
				t.Errorf(
					"test cache.PutExpiry(). Item should not expired yet. "+
						"Key: %s, Timer: %s, elapsed: %d sec",
					tt.Key, expiry.String(), time.Until(expiredTime),
				)
			}
		}
	})

	// blocks until timer of 60 ms awake
	<-time.After(60 * time.Millisecond)

	t.Run("PutShouldExpired", func(t *testing.T) {
		for _, tt := range testKeyVal {
			_, found := tcache.Get(tt.Key)

			if found {
				t.Errorf(
					"test cache.PutExpiry(). Item should be expired but still found. "+
						"Key: %s, Timer: %s, elapsed: %d sec",
					tt.Key, expiry.String(), time.Until(expiredTime),
				)
			}
		}
	})
}

func TestCachePutDefaultExpiry(t *testing.T) {
	var testDefaultExpiry time.Duration = 160 * time.Millisecond

	// test without cleanup, or backup save
	tcache := New(testDefaultExpiry, -1, -1)

	// test put/set key value successful
	kvOne := keyvaluePair{Key: "aa", Value: 10}
	kvTwo := keyvaluePair{Key: "bb", Value: 12}
	kvThree := keyvaluePair{Key: "cc", Value: 13}

	tcache.PutExpiry(kvOne.Key, kvOne.Value, 80*time.Millisecond) // using set expiry to 40ms
	tcache.Put(kvTwo.Key, kvTwo.Value)                            // using default expiry
	tcache.PutExpiry(kvThree.Key, kvThree.Value, -1)              // using set expiry to never

	// blocks until timer of 80ms awaken
	<-time.After(80 * time.Millisecond)

	// Regardless of initialising cache with default expiry or not,
	// PutExpiry() should set the item's expiry and should be respected.
	// Hence, the first item (kvOne) should be expired and the rest stay.
	t.Run("PutExpiryShouldBeRespectedOn80ms", func(t *testing.T) {
		// kvOne should have been expired
		_, found := tcache.Get(kvOne.Key)
		if found {
			t.Errorf("test cache.PutExpiry(kvOne) is not respected when default expiry is set. Elapsed: 80ms")
		}

		_, found = tcache.Get(kvTwo.Key)
		if !found {
			t.Errorf("test cache.Put(kvTwo) with default expiry fail." +
				"Item should NOT be expired yet. Elapsed: 80ms")
		}

		_, found = tcache.Get(kvThree.Key)
		if !found {
			t.Errorf("test cache.PutExpiry(kvThree) with default expiry fail." +
				"Item should Never be expired. Elapsed: 80ms")
		}

	})

	// blocks until timer of 80ms awaken
	<-time.After(80 * time.Millisecond)

	// When caling Put(), item will be expired using default expiry
	// that has been configured during cache initialisation.
	// Hence, the second item should be expired now.
	t.Run("PutWithDefaultExpiryShouldExpireOn160ms", func(t *testing.T) {
		_, found := tcache.Get(kvTwo.Key)
		if found {
			t.Errorf("test cache.Put(kvTwo) with default expiry fail." +
				"Item should be expired by now. Elapsed: 160ms (default expiry)")
		}

		_, found = tcache.Get(kvThree.Key)
		if !found {
			t.Errorf("test cache.Put(kvTwo) with default expiry fail." +
				"Item should Bever expired ever. Elapsed: 160ms (default expiry)")
		}
	})
}

type TestStruct struct {
	Value    int
	Children []*TestStruct
}

func TestCacheSaveAndLoad(t *testing.T) {
	var testDefaultExpiry time.Duration = -1

	// test without expiry, cleanup, or backup save
	tcache := New(testDefaultExpiry, -1, -1)

	testStruct := TestStruct{
		Value: 42,
		Children: []*TestStruct{
			{Value: 19872, Children: []*TestStruct{{Value: 3894}}},
			{Value: 19873, Children: []*TestStruct{{Value: 3891}}},
		},
	}

	// populate cache with test key-value store
	tcache.Put("aa", 1)
	tcache.Put("bb", []byte("HelloWorld this is testing. #test"))
	tcache.Put("structNested", testStruct)

	buf := &bytes.Buffer{}
	err := tcache.Save(buf)
	if err != nil {
		t.Fatalf("test cache.Save() fail, couldn't save to buffer, err: %s", err.Error())
	}

	lcache := New(testDefaultExpiry, -1, -1)
	lcache.Load(buf)
	if err != nil {
		t.Fatalf("test cache.Load() fail, couldn't load from buffer, err: %s", err.Error())
	}

	aa, found := lcache.Get("aa")
	if !found {
		t.Errorf("test cache.Load() then cache.Get() fail, item 'aa' not found from backup cache")
	} else if aa == nil || aa.(int) != 1 {
		t.Errorf(
			"test cache.Load() then cache.Get() fail, wrong value of 'aa'. Expected: %v, Get: %v",
			aa, 1,
		)
	}

	bb, found := lcache.Get("bb")
	if !found {
		t.Errorf("test cache.Load() then cache.Get() fail, item 'bb' not found from backup cache")
	} else if bbyte, ok := bb.([]byte); !ok && !cmp.Equal(bbyte, []byte("HelloWorld this is testing. #test")) {
		t.Errorf(
			"test cache.Load() then cache.Get() fail, wrong value of 'bb'. Expected: %v, Get: %v",
			bb, []byte("HelloWorld this is testing. #test"),
		)
	}

	ss, found := lcache.Get("structNested")
	if !found {
		t.Errorf("test cache.Load() then cache.Get() fail, item 'cc' not found from backup cache")
	} else if ccstruct, ok := ss.(TestStruct); !ok || !cmp.Equal(ccstruct, testStruct) {
		t.Errorf(
			"test cache.Load() then cache.Get() fail, wrong value of 'cc'. Expected: %v, Get: %v",
			ss, testStruct,
		)
	}

}
