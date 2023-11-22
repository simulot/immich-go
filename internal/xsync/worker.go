package xsync

import "sync"

// Worker is a goroutine pool which allows to load-balance
// the work over n workers.
type Worker[T any] struct {
	// queue holds the enqueued tasks.
	queue  chan T
	wg     sync.WaitGroup
	worker func(T)
}

// NewWorker returns a fully initialized [Worker] pool.
// It starts maxWorker goroutines.
// The max. number of elements which can be enqueued is limited by buffer.
// For each task enqueued, the worker function will be called.
func NewWorker[T any](maxWorker uint, worker func(T)) *Worker[T] {
	wk := &Worker[T]{
		queue:  make(chan T),
		worker: worker,
	}

	wk.startWorker(maxWorker)

	return wk
}

func (w *Worker[T]) startWorker(n uint) {
	for i := uint(0); i < n; i++ {
		w.wg.Add(1)
		go w.rawWorker()
	}
}

// rawWorker is a single worker routine waiting for work.
func (w *Worker[T]) rawWorker() {
	defer w.wg.Done()

	for tt := range w.queue {
		w.worker(tt)
	}
}

// Wait waits until all work has been done.
func (w *Worker[T]) Wait() {
	w.wg.Wait()
}

// Finish informs w that no further work will be enqueued.
// After calling this method, no work shall be enqueued!
func (w *Worker[T]) Finish() {
	close(w.queue)
}

// Enqueue enqueues task to be processed by one of the workers.
// The method blocks until one of the workers is able to pick up the
// task.
func (w *Worker[T]) Enqueue(task T) {
	w.queue <- task
}
