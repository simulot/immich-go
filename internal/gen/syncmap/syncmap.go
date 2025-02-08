package syncmap

// SyncMap is a thread-safe and type-safe map based on sync.Map.

import "sync"

type SyncMap[K comparable, V any] struct {
	m sync.Map
}

// New creates a new SyncMap.
// ex:
// var m *syncmap.SyncMap[string, int]
// m := New[string, int]()
func New[K comparable, V any]() *SyncMap[K, V] {
	return &SyncMap[K, V]{
		m: sync.Map{},
	}
}

// Clear deletes all the entries, resulting in an empty Map.
func (m *SyncMap[K, V]) Clear() {
	m.m.Clear()
}

// CompareAndDelete deletes the entry for key if its value is equal to old.
//
// If there is no current value for key in the map, CompareAndDelete returns false .
func (m *SyncMap[K, V]) CompareAndDelete(key K, old V) bool {
	return m.m.CompareAndDelete(key, old)
}

// CompareAndSwap swaps the old and new values for key
// if the value stored in the map is equal to old.
func (m *SyncMap[K, V]) CompareAndSwap(key K, old V, new V) (swapped bool) { // nolint: predeclared
	return m.m.CompareAndSwap(key, old, new)
}

// Delete deletes the value for a key.
func (m *SyncMap[K, V]) Delete(key K) {
	m.m.Delete(key)
}

// Load returns the value stored in the map for a key, or a zero value if no value is present.
// The ok result indicates whether value was found in the map.
func (m *SyncMap[K, V]) Load(key K) (V, bool) {
	v, ok := m.m.Load(key)
	if !ok {
		var zero V
		return zero, false
	}
	return v.(V), true
}

// LoadAndDelete deletes the value for a key, returning the previous value if any.
// The loaded result reports whether the key was present.

func (m *SyncMap[K, V]) LoadAndDelete(key K) (value V, loaded bool) {
	v, ok := m.m.LoadAndDelete(key)
	if !ok {
		var zero V
		return zero, false
	}
	return v.(V), true
}

// LoadOrStore returns the existing value for the key if present.
// Otherwise, it stores and returns the given value.
// The loaded result is true if the value was loaded, false if stored.

func (m *SyncMap[K, V]) LoadOrStore(key K, value V) (V, bool) {
	v, ok := m.m.LoadOrStore(key, value)
	if !ok {
		return value, false
	}
	return v.(V), true
}

// Range calls f sequentially for each key and value present in the map.
// If f returns false, range stops the iteration.
//
// Range does not necessarily correspond to any consistent snapshot of the Map's
// contents: no key will be visited more than once, but if the value for any key
// is stored or deleted concurrently (including by f), Range may reflect any
// mapping for that key from any point during the Range call. Range does not
// block other methods on the receiver; even f itself may call any method on m.
//
// Range may be O(N) with the number of elements in the map even if f returns
// false after a constant number of calls.
func (m *SyncMap[K, V]) Range(f func(key K, value V) bool) {
	m.m.Range(func(key, value interface{}) bool {
		return f(key.(K), value.(V))
	})
}

// Store sets the value for a key.
func (m *SyncMap[K, V]) Store(key K, value V) {
	m.m.Store(key, value)
}

// Swap swaps the value for a key and returns the previous value if any.
// The loaded result reports whether the key was present.
func (m *SyncMap[K, V]) Swap(key K, value V) (previous V, loaded bool) {
	v, ok := m.m.Swap(key, value)
	if !ok {
		var zero V
		return zero, false
	}
	return v.(V), true
}

// Keys returns all the keys in the map.
func (m *SyncMap[K, V]) Keys() []K {
	all := make([]K, 0)
	m.Range(func(key K, value V) bool {
		all = append(all, key)
		return true
	})
	return all
}
