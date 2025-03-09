package cache

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
)

func mockSaveFn(coll string, ids []string) (string, error) {
	if len(ids) == 0 {
		return coll, errors.New("no new items")
	}
	return coll + "|" + strings.Join(ids, ","), nil
}

func TestNewCollection(t *testing.T) {
	cc := NewCollectionCache[string](5, mockSaveFn)
	cc.NewCollection("key1", "coll1", []string{"id1", "id2"})

	if c, ok := cc.collections.Load("key1"); !ok || c.collection != "coll1" {
		t.Error("expected collection coll1 for key1")
	} else {
		if c.items.Len() != 2 {
			t.Errorf("expected 2 items in collection, got %d", c.items.Len())
		}
		cc.Close()
	}
}

func TestAddItemsByRepeatedNewCollectionCalls(t *testing.T) {
	wasCalled := false
	wrappedMockSaveFn := func(coll string, ids []string) (string, error) {
		wasCalled = true
		return mockSaveFn(coll, ids)
	}

	cc := NewCollectionCache[string](2, wrappedMockSaveFn)
	cc.NewCollection("key1", "coll1", nil)

	// Adding items by calling NewCollection multiple times
	cc.AddIDToCollection("key1", "coll1", "id1")
	cc.AddIDToCollection("key1", "coll1", "id2")
	cc.AddIDToCollection("key1", "coll1", "id3")

	cc.Close()
	if !wasCalled {
		t.Error("expected wrappedMockSaveFn to be called at least once")
	}
}

func TestMultipleCollectionsExceedingCacheSize(t *testing.T) {
	wasCalledCount := make(map[string]int)

	wrappedMockSaveFn := func(coll string, ids []string) (string, error) {
		wasCalledCount[coll]++
		return coll, nil
	}

	cc := NewCollectionCache[string](2, wrappedMockSaveFn)

	// Initial collections
	cc.NewCollection("key1", "coll1", []string{"id1", "id2"})
	cc.NewCollection("key2", "coll2", []string{"id3", "id4"})

	// Exceed cache size by adding more IDs
	cc.AddIDToCollection("key1", "coll1", "id5")
	if wasCalledCount["coll1"] != 0 {
		t.Errorf("expected saveFn called %d times for coll1, got %d", 0, wasCalledCount["coll1"])
	}
	cc.AddIDToCollection("key1", "coll1", "id6")
	if wasCalledCount["coll1"] != 1 {
		t.Errorf("expected saveFn called %d times for coll1, got %d", 1, wasCalledCount["coll1"])
	}

	cc.AddIDToCollection("key1", "coll1", "id7")
	if wasCalledCount["coll1"] != 1 {
		t.Errorf("expected saveFn called %d times for coll1, got %d", 1, wasCalledCount["coll1"])
	}

	cc.AddIDToCollection("key2", "coll2", "id8")
	if wasCalledCount["coll2"] != 0 {
		t.Errorf("expected saveFn called %d times for coll2, got %d", 0, wasCalledCount["coll2"])
	}
	cc.Close()

	// Checking collections
	coll1, ID1s, ok := cc.GetCollection("key1")
	if !ok {
		t.Error("expected collection key1 to be present")
		return
	}
	if coll1 != "coll1" {
		t.Errorf("expected collection coll1, got %s", coll1)
	}
	if len(ID1s) != 5 {
		t.Errorf("expected 2 IDs in collection coll1, got %d", len(ID1s))
	}

	coll2, ID2s, ok := cc.GetCollection("key2")
	if !ok {
		t.Error("expected collection key2 to be present")
		return
	}
	if coll2 != "coll2" {
		t.Errorf("expected collection coll2, got %s", coll2)
	}
	if len(ID2s) != 3 {
		t.Errorf("expected 3 IDs in collection coll2, got %d", len(ID2s))
	}

	// Checking calls
	if wasCalledCount["coll1"] < 2 {
		t.Errorf("expected multiple saveFn calls for coll1, got %d", wasCalledCount["coll1"])
	}
	if wasCalledCount["coll2"] < 1 {
		t.Errorf("expected saveFn call for coll2, got %d", wasCalledCount["coll2"])
	}
}

func TestCollectionCacheConcurrentAccess(t *testing.T) {
	cc := NewCollectionCache[string](10, func(coll string, ids []string) (string, error) {
		return coll, nil
	})

	cc.NewCollection("testKey", "testColl", nil)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			id := fmt.Sprintf("asset%d", n)
			cc.AddIDToCollection("testKey", "testColl", id)
		}(i)
	}
	wg.Wait()

	items, ids, ok := cc.GetCollection("testKey")
	if !ok {
		t.Fatal("expected testKey to exist")
	}
	if len(ids) != 50 {
		t.Errorf("expected 50 items, got %d", len(items))
	}
	cc.Close()
}
