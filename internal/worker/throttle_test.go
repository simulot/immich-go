package worker

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestThrottle_BasicConcurrency(t *testing.T) {
	th := NewThrottle(3)
	defer th.Close()

	var running atomic.Int32
	var maxSeen atomic.Int32
	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			th.Acquire()
			cur := running.Add(1)
			for {
				old := maxSeen.Load()
				if cur <= old || maxSeen.CompareAndSwap(old, cur) {
					break
				}
			}
			time.Sleep(10 * time.Millisecond)
			running.Add(-1)
			th.Release()
		}()
	}

	wg.Wait()
	if maxSeen.Load() > 3 {
		t.Errorf("max concurrent = %d, want <= 3", maxSeen.Load())
	}
}

func TestThrottle_SetConcurrency(t *testing.T) {
	th := NewThrottle(10)
	defer th.Close()

	if th.Current() != 10 {
		t.Errorf("initial concurrency = %d, want 10", th.Current())
	}

	th.SetConcurrency(5)
	if th.Current() != 5 {
		t.Errorf("after SetConcurrency(5) = %d, want 5", th.Current())
	}

	th.SetConcurrency(0) // should clamp to 1
	if th.Current() != 1 {
		t.Errorf("after SetConcurrency(0) = %d, want 1", th.Current())
	}

	th.SetConcurrency(20)
	if th.Current() != 20 {
		t.Errorf("after SetConcurrency(20) = %d, want 20", th.Current())
	}
}

func TestThrottle_Max(t *testing.T) {
	th := NewThrottle(8)
	if th.Max() != 8 {
		t.Errorf("Max() = %d, want 8", th.Max())
	}
	th.SetConcurrency(3)
	if th.Max() != 8 {
		t.Errorf("Max() after SetConcurrency = %d, want 8", th.Max())
	}
}
