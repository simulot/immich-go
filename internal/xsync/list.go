package xsync

import "sync"

// List is a concurrent safe slice of T.
type List[T any] struct {
	lo   sync.RWMutex
	data []T
}

// Push pushes the given elems onto l.
func (l *List[T]) Push(elems ...T) {
	l.lo.Lock()
	defer l.lo.Unlock()

	l.data = append(l.data, elems...)
}

// All safely iterates over l, executing yield for every item.
// The iterator stops once yield returns false and in turn, All
// returns false too.
//
// NOTE: Once https://github.com/golang/go/issues/61405 has been merged and released
// ouside of GOEXPERIMENTAL, all references to this method can be rewritten to a simple
// for-loop.
func (l *List[T]) All(yield func(T) bool) bool {
	l.lo.RLock()
	defer l.lo.RUnlock()

	for _, elem := range l.data {
		// make a explicit copy – with go 1.22 (https://go.dev/blog/loopvar-preview)
		// this will not be necessary anymore.
		elem := elem

		if !yield(elem) {
			return false
		}
	}

	return true
}

// Len returns the number of elements in l.
func (l *List[T]) Len() int {
	l.lo.RLock()
	defer l.lo.RUnlock()

	return len(l.data)
}
