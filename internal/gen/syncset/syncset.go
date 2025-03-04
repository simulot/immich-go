package syncset

import "sync"

// Set is a thread-safe set of items.
type Set[T comparable] struct {
	m sync.Map
}

// NewSet creates a new set.
func NewSet[T comparable]() *Set[T] {
	return &Set[T]{
		m: sync.Map{},
	}
}

// Add adds an item to the set. Returns true if the item was added, false if it was already present.
func (s *Set[T]) Add(item T) bool {
	_, loaded := s.m.LoadOrStore(item, struct{}{})
	return loaded
}

// Remove removes an item from the set.
func (s *Set[T]) Remove(item T) {
	s.m.Delete(item)
}

// Contains returns true if the set contains the item.
func (s *Set[T]) Contains(item T) bool {
	_, ok := s.m.Load(item)
	return ok
}

// Items returns a slice of all items in the set.
func (s *Set[T]) Items() []T {
	var items []T
	s.m.Range(func(key, _ any) bool {
		items = append(items, key.(T))
		return true
	})
	return items
}

// Range calls the given function for each item in the set.
func (s *Set[T]) Range(fn func(T)) {
	s.m.Range(func(key, _ any) bool {
		fn(key.(T))
		return true
	})
}
