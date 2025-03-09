package cache

import (
	"sync"

	"github.com/simulot/immich-go/internal/gen/syncmap"
	"github.com/simulot/immich-go/internal/gen/syncset"
)

type saveFn[T any] func(coll T, ids []string) (T, error)

type CollectionCache[T comparable] struct {
	maxCacheSize int
	collections  *syncmap.SyncMap[string, *Collection[T]] // collection of Collections
}

type Collection[T comparable] struct {
	lock       sync.RWMutex
	collection T
	items      *syncset.Set[string]
	newItems   *syncset.Set[string]
}

func (c *Collection[T]) Items() []string {
	return c.items.Items()
}

func (c *Collection[T]) NewItems() []string {
	return c.newItems.Items()
}

// NewCollectionCache creates a new collection cache with the given max cache size.
func NewCollectionCache[T comparable](maxCacheSize int) *CollectionCache[T] {
	return &CollectionCache[T]{
		maxCacheSize: maxCacheSize,
		collections:  syncmap.New[string, *Collection[T]](),
	}
}

// NewCollection manage the cache for the given collection and the given list of initial ids.
// If the collection already exists, the ids are added to the existing collection.
func (cc *CollectionCache[T]) NewCollection(key string, coll T, ids []string) {
	c, ok := cc.collections.Load(key)
	if !ok {
		c = &Collection[T]{
			collection: coll,
			items:      syncset.New(ids...),
			newItems:   syncset.New[string](),
		}
		cc.collections.Store(key, c)
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	for _, id := range ids {
		c.items.Add(id)
	}
}

func (cc *CollectionCache[T]) GetCollections() *syncmap.SyncMap[string, *Collection[T]] {
	return cc.collections
}

func (cc *CollectionCache[T]) GetCollection(key string) (T, bool) {
	c, ok := cc.collections.Load(key)
	if !ok {
		var zero T
		return zero, false
	}
	return c.collection, true
}

func (cc *CollectionCache[T]) AddAssetsToCollection(key string, coll T, id string, saveFn saveFn[T]) bool {
	c, ok := cc.collections.Load(key)
	if !ok {
		c = &Collection[T]{
			collection: coll,
			items:      syncset.New(id),
			newItems:   syncset.New(id),
		}
		cc.collections.Store(key, c)
		return true
	}
	if c.items.Add(id) { // We have seen id
		c.newItems.Add(id) // id is new, added to newItems
		if c.newItems.Len() >= cc.maxCacheSize {
			c.lock.Lock()
			defer c.lock.Unlock()
			var err error
			c.collection, err = saveFn(c.collection, c.newItems.Items())
			if err == nil {
				c.newItems = syncset.New[string]()
			}
		}
		return true
	}
	return false
}

func (cc *CollectionCache[T]) Flush(saveFn saveFn[T]) {
	cc.collections.Range(func(key string, c *Collection[T]) bool {
		if c.newItems.Len() > 0 {
			c.lock.Lock()
			defer c.lock.Unlock()
			_, _ = saveFn(c.collection, c.newItems.Items()) // potential errors are logged in the saveFn
		}
		return true
	})
}
