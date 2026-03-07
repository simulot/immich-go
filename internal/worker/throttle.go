package worker

import "sync"

// Throttle controls the level of concurrency using a channel-based semaphore.
// It can be dynamically adjusted at runtime via SetConcurrency.
type Throttle struct {
	sem     chan struct{}
	current int
	max     int
	mu      sync.Mutex
}

// NewThrottle creates a Throttle with the given initial concurrency level.
func NewThrottle(n int) *Throttle {
	if n < 1 {
		n = 1
	}
	t := &Throttle{
		sem:     make(chan struct{}, n),
		current: n,
		max:     n,
	}
	return t
}

// Acquire blocks until a slot is available.
func (t *Throttle) Acquire() {
	t.mu.Lock()
	sem := t.sem
	t.mu.Unlock()
	sem <- struct{}{}
}

// Release frees a slot.
func (t *Throttle) Release() {
	t.mu.Lock()
	sem := t.sem
	t.mu.Unlock()
	<-sem
}

// SetConcurrency changes the concurrency level by replacing the semaphore channel.
// In-flight tasks continue to hold slots on the old channel; new tasks use the new one.
// Minimum concurrency is 1.
func (t *Throttle) SetConcurrency(n int) {
	if n < 1 {
		n = 1
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if n == t.current {
		return
	}
	t.sem = make(chan struct{}, n)
	t.current = n
}

// Current returns the current concurrency level.
func (t *Throttle) Current() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.current
}

// Max returns the original (maximum) concurrency level.
func (t *Throttle) Max() int {
	return t.max
}

// Close is a no-op for compatibility; the throttle doesn't need explicit cleanup.
func (t *Throttle) Close() {}
