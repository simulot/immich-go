package syncset

import (
	"sync"
	"sync/atomic"
)

// Set is a thread-safe set of items.
type Set[T comparable] struct {
	lock sync.Mutex
	m    sync.Map
	len  int64
}

// NewSet creates a new set.
func New[T comparable](items ...T) *Set[T] {
	s := &Set[T]{
		m: sync.Map{},
	}
	for _, item := range items {
		s.Add(item)
	}
	return s
}

// Add adds an item to the set. Returns true if the item was added, false if it was already present.
func (s *Set[T]) Add(item T) bool {
	s.lock.Lock()
	defer s.lock.Unlock()
	_, loaded := s.m.LoadOrStore(item, struct{}{})
	if !loaded {
		atomic.AddInt64(&s.len, 1)
	}
	return !loaded
}

// Remove removes an item from the set.
func (s *Set[T]) Remove(item T) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if _, loaded := s.m.LoadAndDelete(item); loaded {
		atomic.AddInt64(&s.len, -1)
	}
}

// Contains returns true if the set contains the item.
func (s *Set[T]) Contains(item T) bool {
	_, ok := s.m.Load(item)
	return ok
}

// Len returns the number of items in the set.
func (s *Set[T]) Len() int {
	return int(atomic.LoadInt64(&s.len))
}

// Items returns a slice of all items in the set.
func (s *Set[T]) Items() []T {
	s.lock.Lock()
	defer s.lock.Unlock()
	var items []T
	s.m.Range(func(key, _ any) bool {
		items = append(items, key.(T))
		return true
	})
	return items
}

// Range calls the given function for each item in the set.
func (s *Set[T]) Range(fn func(T)) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.m.Range(func(key, _ any) bool {
		fn(key.(T))
		return true
	})
}
