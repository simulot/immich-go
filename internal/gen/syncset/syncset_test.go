package syncset

import (
	"sync"
	"testing"
)

func TestSet(t *testing.T) {
	t.Run("New set is empty", func(t *testing.T) {
		s := New[int]()
		if s.Len() != 0 {
			t.Errorf("Expected len 0, got %d", s.Len())
		}
	})

	t.Run("Add increases size and returns false if not present", func(t *testing.T) {
		s := New[int]()
		if !s.Add(1) {
			t.Errorf("Expected true on first add, got false")
		}
		if s.Len() != 1 {
			t.Errorf("Expected len 1, got %d", s.Len())
		}
		if s.Add(1) {
			t.Errorf("Expected false on adding duplicate, got true")
		}
		if s.Len() != 1 {
			t.Errorf("Expected len 1, got %d", s.Len())
		}
	})

	t.Run("Remove decreases size if item was present", func(t *testing.T) {
		s := New[int](1, 2)
		s.Remove(1)
		if s.Len() != 1 {
			t.Errorf("Expected len 1, got %d", s.Len())
		}
		s.Remove(1)
		if s.Len() != 1 {
			t.Errorf("Expected len to remain 1, got %d", s.Len())
		}
	})

	t.Run("Contains checks presence of item", func(t *testing.T) {
		s := New[int](1, 2, 3)
		if !s.Contains(1) {
			t.Errorf("Expected set to contain 1")
		}
		if s.Contains(4) {
			t.Errorf("Expected set not to contain 4")
		}
	})

	t.Run("Items returns all items", func(t *testing.T) {
		s := New[int](1, 2, 3)
		items := s.Items()
		if len(items) != 3 {
			t.Errorf("Expected 3 items, got %d", len(items))
		}
	})

	t.Run("Range visits all items", func(t *testing.T) {
		s := New[int](1, 2, 3)
		visited := 0
		s.Range(func(item int) {
			visited++
		})
		if visited != 3 {
			t.Errorf("Expected to visit 3 items, got %d", visited)
		}
	})

	t.Run("Concurrent access", func(t *testing.T) {
		s := New[int]()
		var wg sync.WaitGroup
		const total = 100
		for i := 0; i < total; i++ {
			wg.Add(1)
			go func(val int) {
				defer wg.Done()
				s.Add(val)
			}(i)
		}
		wg.Wait()

		// Verify all items were added
		if s.Len() != total {
			t.Errorf("expected %d items, got %d", total, s.Len())
		}

		// Concurrent remove
		for i := 0; i < total; i++ {
			wg.Add(1)
			go func(val int) {
				defer wg.Done()
				s.Remove(val)
			}(i)
		}
		wg.Wait()

		// Ensure all items were removed
		if s.Len() != 0 {
			t.Errorf("expected 0 items, got %d", s.Len())
		}
	})
}
