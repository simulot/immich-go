package gen

import (
	"sort"
	"sync"

	"golang.org/x/exp/constraints"
)

func MapKeys[K comparable, T any](m map[K]T) []K {
	r := make([]K, len(m))
	i := 0
	for k := range m {
		r[i] = k
		i++
	}
	return r
}

func MapKeysSorted[K constraints.Ordered, T any](m map[K]T) []K {
	r := make([]K, len(m))
	i := 0
	for k := range m {
		r[i] = k
		i++
	}
	sort.Slice(r, func(i, j int) bool {
		return r[i] < r[j]
	})
	return r
}

func MapFilterKeys[K comparable, T any](m map[K]T, f func(i T) bool) []K {
	r := make([]K, 0, len(m))
	for k, v := range m {
		if f(v) {
			r = append(r, k)
		}
	}
	return r
}

type SyncMap[K comparable, V any] struct {
	mu sync.RWMutex
	m  map[K]V
}

func NewSyncMap[K comparable, V any]() *SyncMap[K, V] {
	return &SyncMap[K, V]{m: make(map[K]V)}
}

func (m *SyncMap[K, V]) Load(k K) (V, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.m[k]
	return v, ok
}

func (m *SyncMap[K, V]) Store(k K, v V) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.m[k] = v
}

func (m *SyncMap[K, V]) Delete(k K) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.m, k)
}

func (m *SyncMap[K, V]) Keys() []K {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r := make([]K, len(m.m))
	i := 0
	for k := range m.m {
		r[i] = k
		i++
	}
	return r
}
