package worker

import (
	"sync"
)

// Task represents a unit of work to be processed by the worker pool.
type Task func()

// Pool manages a pool of worker goroutines.
type Pool struct {
	tasks  chan Task
	wg     sync.WaitGroup
	quit   chan struct{}
	closed bool
}

// NewPool creates a new Pool with a specified number of workers.
func NewPool(numWorkers int) *Pool {
	// ARM fix: buffer the tasks channel so that Submit() never blocks the
	// caller waiting for a free worker. An unbuffered channel causes the
	// main goroutine (reading from groupChan) to stall between each task
	// submission, serialising what should be pipelined work. A buffer of
	// numWorkers*2 allows the caller to stay ahead of the workers without
	// holding more than a small number of pending tasks in memory.
	bufSize := numWorkers * 2
	if bufSize < 4 {
		bufSize = 4
	}
	pool := &Pool{
		tasks: make(chan Task, bufSize),
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
// It drains any buffered tasks that have not yet been picked up by a worker
// before signalling shutdown, ensuring no submitted work is silently dropped.
func (p *Pool) Stop() {
	if !p.closed {
		// Drain remaining buffered tasks before stopping workers.
		// With a buffered channel, tasks submitted just before Stop() may
		// sit in the buffer without a worker having picked them up yet.
		// We must let the workers consume them before sending quit.
		for len(p.tasks) > 0 {
			select {
			case task := <-p.tasks:
				task()
			default:
			}
		}
		close(p.quit)
		p.wg.Wait()
		close(p.tasks)
		p.closed = true
	}
}
