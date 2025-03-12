package cache

import (
	"sync"

	"github.com/simulot/immich-go/internal/gen/syncmap"
	"github.com/simulot/immich-go/internal/gen/syncset"
)

type saveFn[T any] func(coll T, ids []string) (T, error)

type CollectionCache[T comparable] struct {
	maxCacheSize      int
	collections       *syncmap.SyncMap[string, *Collection[T]] // collection of Collections
	chanNewCollection chan func()
	saveFn            saveFn[T]
}

// NewCollectionCache creates a new collection cache with the given max cache size.
func NewCollectionCache[T comparable](maxCacheSize int, saveFn saveFn[T]) *CollectionCache[T] {
	cc := &CollectionCache[T]{
		maxCacheSize:      maxCacheSize,
		collections:       syncmap.New[string, *Collection[T]](),
		chanNewCollection: make(chan func(), 100),
		saveFn:            saveFn,
	}
	go func() {
		for f := range cc.chanNewCollection {
			f()
		}
	}()

	return cc
}

// NewCollection manage the cache for the given collection and the given list of initial ids.
// If the collection already exists, the ids are added to the existing collection.
func (cc *CollectionCache[T]) NewCollection(key string, coll T, ids []string) {
	wg := sync.WaitGroup{}
	wg.Add(1)
	cc.chanNewCollection <- func() {
		defer wg.Done()
		c, ok := cc.collections.Load(key)
		if !ok {
			c := newCollection(coll, cc.maxCacheSize, cc.saveFn, ids, nil)
			cc.collections.Store(key, c)
		} else {
			for _, id := range ids {
				c.addID(id)
			}
		}
	}
	wg.Wait()
}

func (cc *CollectionCache[T]) AddIDToCollection(key string, collObj T, id string) bool {
	added := false
	wg := sync.WaitGroup{}
	wg.Add(1)
	cc.chanNewCollection <- func() {
		defer wg.Done()
		c, ok := cc.collections.Load(key)
		if !ok {
			c = newCollection(collObj, cc.maxCacheSize, cc.saveFn, nil, nil)
			cc.collections.Store(key, c)
		}
		added = c.addID(id)
	}
	wg.Wait()
	return added
}

func (cc *CollectionCache[T]) GetCollections() *syncmap.SyncMap[string, *Collection[T]] {
	return cc.collections
}

func (cc *CollectionCache[T]) GetCollection(key string) (T, []string, bool) {
	c, ok := cc.collections.Load(key)
	if !ok {
		var zero T
		return zero, nil, false
	}
	return c.collection, c.Items(), true
}

func (cc *CollectionCache[T]) Close() {
	cc.collections.Range(func(key string, c *Collection[T]) bool {
		c.close()
		return true
	})
	close(cc.chanNewCollection)
}

type Collection[T comparable] struct {
	collection   T
	items        *syncset.Set[string]
	newItems     *syncset.Set[string]
	saveFn       saveFn[T]
	maxCacheSize int
}

func newCollection[T comparable](collection T, maxCacheSize int, saveFn saveFn[T], initialIDs []string, newIDs []string) *Collection[T] {
	c := &Collection[T]{
		collection:   collection,
		items:        syncset.New(initialIDs...),
		newItems:     syncset.New(newIDs...),
		saveFn:       saveFn,
		maxCacheSize: maxCacheSize,
	}

	return c
}

func (c *Collection[T]) close() {
	_, _ = c.saveFn(c.collection, c.newItems.Items())
}

func (c *Collection[T]) Items() []string {
	return c.items.Items()
}

func (c *Collection[T]) NewItems() []string {
	return c.newItems.Items()
}

func (c *Collection[T]) addID(id string) bool {
	added := false
	if c.items.Add(id) {
		added = true
		c.newItems.Add(id)
		if c.newItems.Len() >= c.maxCacheSize {
			// err is ignored because it's logged in the saveFn
			c.collection, _ = c.saveFn(c.collection, c.newItems.Items())

			// a fresh set of assets, even if the save failed, to avoid retrying the same assets
			c.newItems = syncset.New[string]()
		}
	}
	return added
}
