package cache

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func mockSaveFn(coll string, ids []string) (string, error) {
	if len(ids) == 0 {
		return coll, errors.New("no new items")
	}
	return coll + "|" + strings.Join(ids, ","), nil
}

func TestNewCollectionCache(t *testing.T) {
	cc := NewCollectionCache[string](5)
	if cc.maxCacheSize != 5 {
		t.Errorf("expected maxCacheSize = 5, got %d", cc.maxCacheSize)
	}
}

func TestNewCollection(t *testing.T) {
	cc := NewCollectionCache[string](5)
	cc.NewCollection("key1", "coll1", []string{"id1", "id2"})

	if c, ok := cc.collections.Load("key1"); !ok || c.collection != "coll1" {
		t.Error("expected collection coll1 for key1")
	}
}

func TestAddAssetsToCollection(t *testing.T) {
	wasCalled := false
	wrappedMockSaveFn := func(coll string, ids []string) (string, error) {
		wasCalled = true
		return mockSaveFn(coll, ids)
	}

	cc := NewCollectionCache[string](2)
	added := cc.AddAssetsToCollection("key1", "coll1", "id1", wrappedMockSaveFn)
	if !added {
		t.Error("expected id1 to be added")
	}
	added = cc.AddAssetsToCollection("key1", "coll1", "id2", wrappedMockSaveFn)
	if !added {
		t.Error("expected id2 to be added")
	}
	added = cc.AddAssetsToCollection("key1", "coll1", "id3", wrappedMockSaveFn)
	if !added {
		t.Error("expected id3 to be added")
	}

	// Confirm saveFn was called at least once
	if !wasCalled {
		t.Error("wrappedMockSaveFn (mockSaveFn) was never called")
	}
}

func TestFlush(t *testing.T) {
	wasCalled := false
	wrappedMockSaveFn := func(coll string, ids []string) (string, error) {
		wasCalled = true
		return mockSaveFn(coll, ids)
	}

	cc := NewCollectionCache[string](5)
	cc.NewCollection("key1", "coll1", []string{})
	cc.AddAssetsToCollection("key1", "coll1", "id1", wrappedMockSaveFn)
	cc.Flush(wrappedMockSaveFn)

	if !wasCalled {
		t.Error("wrappedMockSaveFn (mockSaveFn) was never called during flush")
	}
}

func TestSaveFnCalledCorrectNumberOfTimesWhenExceedingCacheSize(t *testing.T) {
	wasCalledCount := 0
	wrappedMockSaveFn := func(coll string, ids []string) (string, error) {
		wasCalledCount++
		return mockSaveFn(coll, ids)
	}

	// Given a cache size of 2
	cc := NewCollectionCache[string](2)

	// Adding 5 items
	for i := 1; i <= 5; i++ {
		id := fmt.Sprintf("id%d", i)
		cc.AddAssetsToCollection("key1", "coll1", id, wrappedMockSaveFn)
	}

	cc.Flush(wrappedMockSaveFn)

	// Expected calls: 3 times (for saving items : 1&2, 3&4, then 5)
	expectedCalls := 3
	if wasCalledCount != expectedCalls {
		t.Errorf("expected saveFn to be called %d times, got %d", expectedCalls, wasCalledCount)
	}
}

func TestMultipleCollectionsExceedingCacheSize(t *testing.T) {
	wasCalledCount := make(map[string]int)

	wrappedMockSaveFn := func(coll string, ids []string) (string, error) {
		wasCalledCount[coll]++
		return coll, nil
	}

	cc := NewCollectionCache[string](2)

	cc.NewCollection("key1", "coll1", []string{"id1", "id2"})
	cc.NewCollection("key2", "coll2", []string{"id3", "id4"})

	// Add enough items to exceed the cache size for each collection.
	// This should trigger saving (wrappedMockSaveFn) multiple times.
	cc.AddAssetsToCollection("key1", "coll1", "id5", wrappedMockSaveFn)
	cc.AddAssetsToCollection("key1", "coll1", "id6", wrappedMockSaveFn)
	cc.AddAssetsToCollection("key1", "coll1", "id9", wrappedMockSaveFn)
	cc.AddAssetsToCollection("key2", "coll2", "id7", wrappedMockSaveFn)
	cc.AddAssetsToCollection("key2", "coll2", "id8", wrappedMockSaveFn)

	// check if coll2 is saved once for id7, id8
	if wasCalledCount["coll2"] != 1 {
		t.Errorf("expected 2 saveFn calls for coll2, got %d", wasCalledCount["coll2"])
	}

	cc.Flush(wrappedMockSaveFn)

	// check if coll1 is saved twice: once for id5, id6 and once for id9
	if wasCalledCount["coll1"] != 2 {
		t.Errorf("expected 2 saveFn calls for coll1, got %d", wasCalledCount["coll1"])
	}
	// no change, only once
	if wasCalledCount["coll2"] != 1 {
		t.Errorf("expected 2 saveFn calls for coll2, got %d", wasCalledCount["coll2"])
	}
}
