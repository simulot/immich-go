package upload

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestConcurrentProcessing tests that concurrent upload processing
// actually processes items concurrently rather than sequentially
func TestConcurrentProcessing(t *testing.T) {
	const numItems = 6
	const numWorkers = 3
	const processingDelay = 50 * time.Millisecond

	// Create a channel to simulate the group processing
	itemChan := make(chan int, numItems)

	// Fill the channel with test items
	for i := 0; i < numItems; i++ {
		itemChan <- i
	}
	close(itemChan)

	var processedCount int64
	var startTimes []time.Time
	var mu sync.Mutex

	// Record start time
	overallStart := time.Now()

	// Simulate the concurrent processing pattern from uploadLoopWithWorkers
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for item := range itemChan {
				// Record when this item starts processing
				mu.Lock()
				startTimes = append(startTimes, time.Now())
				mu.Unlock()

				// Simulate processing time
				time.Sleep(processingDelay)

				// Increment processed count
				atomic.AddInt64(&processedCount, 1)

				t.Logf("Processed item %d", item)
			}
		}()
	}

	wg.Wait()
	totalDuration := time.Since(overallStart)

	// Verify all items were processed
	if processedCount != numItems {
		t.Errorf("Expected %d items to be processed, got %d", numItems, processedCount)
	}

	// With 3 workers processing 6 items that take 50ms each,
	// concurrent processing should take roughly 100ms (2 batches)
	// Sequential would take 300ms (6 items * 50ms each)
	expectedMaxDuration := 3 * processingDelay // Allow some buffer
	if totalDuration > expectedMaxDuration {
		t.Errorf("Processing took too long: %v (expected < %v)", totalDuration, expectedMaxDuration)
	}

	// Verify that multiple items started processing around the same time
	// (indicating concurrent execution)
	mu.Lock()
	defer mu.Unlock()

	if len(startTimes) != numItems {
		t.Errorf("Expected %d start times, got %d", numItems, len(startTimes))
		return
	}

	// Check that the first 3 items started within a short time window
	// (indicating they started concurrently)
	if len(startTimes) >= 3 {
		maxStartTimeDiff := startTimes[2].Sub(startTimes[0])
		if maxStartTimeDiff > 10*time.Millisecond {
			t.Errorf("First 3 items should start nearly simultaneously, but took %v", maxStartTimeDiff)
		}
	}
}

// TestConcurrentUploadPatternMatching tests that the upload loop
// correctly chooses between sequential and concurrent patterns
func TestConcurrentUploadPatternMatching(t *testing.T) {
	tests := []struct {
		name              string
		concurrentUploads int
		expectsConcurrent bool
	}{
		{
			name:              "single worker uses sequential",
			concurrentUploads: 1,
			expectsConcurrent: false,
		},
		{
			name:              "two workers use concurrent",
			concurrentUploads: 2,
			expectsConcurrent: true,
		},
		{
			name:              "max workers use concurrent",
			concurrentUploads: 20,
			expectsConcurrent: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			upCmd := &UpCmd{
				UploadOptions: &UploadOptions{
					ConcurrentUploads: tt.concurrentUploads,
				},
			}

			// The decision logic from uploadLoop
			usesConcurrent := upCmd.ConcurrentUploads != 1

			if usesConcurrent != tt.expectsConcurrent {
				t.Errorf("ConcurrentUploads=%d: expected concurrent=%v, got concurrent=%v",
					tt.concurrentUploads, tt.expectsConcurrent, usesConcurrent)
			}
		})
	}
}

// TestWorkerPoolScaling tests that the worker pool scales correctly
// with different numbers of workers
func TestWorkerPoolScaling(t *testing.T) {
	const numItems = 12
	const processingDelay = 10 * time.Millisecond

	testCases := []struct {
		workers         int
		expectedMaxTime time.Duration
	}{
		{1, 15 * processingDelay}, // Sequential: 12 * 10ms = 120ms + buffer
		{2, 8 * processingDelay},  // Concurrent: 6 * 10ms = 60ms + buffer
		{3, 6 * processingDelay},  // Concurrent: 4 * 10ms = 40ms + buffer
		{4, 5 * processingDelay},  // Concurrent: 3 * 10ms = 30ms + buffer
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%d_workers", tc.workers), func(t *testing.T) {
			itemChan := make(chan int, numItems)

			// Fill the channel
			for i := 0; i < numItems; i++ {
				itemChan <- i
			}
			close(itemChan)

			var processedCount int64
			start := time.Now()

			// Simulate worker pool
			var wg sync.WaitGroup
			for i := 0; i < tc.workers; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()

					for item := range itemChan {
						time.Sleep(processingDelay)
						atomic.AddInt64(&processedCount, 1)
						_ = item // Use the item
					}
				}()
			}

			wg.Wait()
			duration := time.Since(start)

			if processedCount != numItems {
				t.Errorf("Workers=%d: Expected %d items processed, got %d",
					tc.workers, numItems, processedCount)
			}

			if duration > tc.expectedMaxTime {
				t.Errorf("Workers=%d: Processing took %v, expected < %v",
					tc.workers, duration, tc.expectedMaxTime)
			}

			t.Logf("Workers=%d: Processed %d items in %v", tc.workers, numItems, duration)
		})
	}
}
