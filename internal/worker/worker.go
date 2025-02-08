package worker

import (
	"sync"
)

// Task represents a unit of work to be processed by the worker pool.
type Task func()

// Pool manages a pool of worker goroutines.
type Pool struct {
	tasks chan Task
	wg    sync.WaitGroup
	quit  chan struct{}
}

// NewPool creates a new Pool with a specified number of workers.
func NewPool(numWorkers int) *Pool {
	pool := &Pool{
		tasks: make(chan Task),
		quit:  make(chan struct{}),
	}

	for i := 0; i < numWorkers; i++ {
		pool.wg.Add(1)
		go pool.worker()
	}

	return pool
}

// worker is the function that each worker goroutine runs.
func (p *Pool) worker() {
	defer p.wg.Done()
	for {
		select {
		case task := <-p.tasks:
			task()
		case <-p.quit:
			return
		}
	}
}

// Submit adds a task to the worker pool.
func (p *Pool) Submit(task Task) {
	p.tasks <- task
}

// Stop stops all the workers and waits for them to finish.
func (p *Pool) Stop() {
	close(p.quit)
	p.wg.Wait()
	close(p.tasks)
}
