package worker

// Semaphore limits concurrent execution using a buffered channel.
// Unlike a worker pool, it allows unlimited goroutines but only N run concurrently.
// This prevents deadlock in recursive directory scanning scenarios.
type Semaphore struct {
	sem chan struct{}
}

// NewSemaphore creates a new Semaphore with the specified concurrency limit.
func NewSemaphore(limit int) *Semaphore {
	return &Semaphore{
		sem: make(chan struct{}, limit),
	}
}

// Acquire acquires a semaphore slot, blocking if limit is reached.
func (s *Semaphore) Acquire() {
	s.sem <- struct{}{}
}

// Release releases a semaphore slot, allowing another goroutine to proceed.
func (s *Semaphore) Release() {
	<-s.sem
}

// Pool is kept for backward compatibility but now uses Semaphore internally.
type Pool struct {
	semaphore *Semaphore
}

// NewPool creates a new Pool that uses a Semaphore for concurrency control.
func NewPool(numWorkers int) *Pool {
	return &Pool{
		semaphore: NewSemaphore(numWorkers),
	}
}

// Submit executes a task with semaphore-based concurrency control.
// The task runs in its own goroutine but is limited by the semaphore.
func (p *Pool) Submit(task func()) {
	go func() {
		p.semaphore.Acquire()
		defer p.semaphore.Release()
		task()
	}()
}

// Stop is a no-op for backward compatibility.
// With semaphore approach, there are no workers to stop.
func (p *Pool) Stop() {
	// No-op: semaphore doesn't need cleanup
}
