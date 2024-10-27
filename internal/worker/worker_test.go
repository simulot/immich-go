package worker

import (
	"sync"
	"testing"
)

func TestPool(t *testing.T) {
	var mu sync.Mutex
	results := make([]int, 0)

	// Create a worker pool with 3 workers.
	pool := NewPool(3)

	// Submit some tasks to the pool.
	for i := 0; i < 10; i++ {
		taskNum := i
		pool.Submit(func() {
			mu.Lock()
			results = append(results, taskNum)
			mu.Unlock()
		})
	}

	// Stop the worker pool and wait for all workers to finish.
	pool.Stop()

	// Check if all tasks were processed.
	if len(results) != 10 {
		t.Errorf("Expected 10 tasks to be processed, but got %d", len(results))
	}
}
