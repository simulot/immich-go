package main

import "sync"

type task func()

type Worker struct {
	threads int
	limiter chan task
	stop    chan any
}

func NewWorker(threads int) *Worker {
	w := Worker{
		threads: threads,
		limiter: make(chan task, threads),
		stop:    make(chan any),
	}

	return &w
}

func (w *Worker) Run() func() {
	if w.threads > 1 {
		wg := sync.WaitGroup{}
		go func() {
			for t := range w.limiter {
				select {
				default:
					wg.Add(1)
					go func(t task) {
						t()
						wg.Done()
					}(t)
				case <-w.stop:
					return
				}
			}
		}()
		return func() {
			close(w.stop)
			wg.Wait()
		}
	}
	return func() {}
}

func (w *Worker) Push(t task) {
	if w.threads > 1 {
		w.limiter <- t
		return
	}
	t()
}
