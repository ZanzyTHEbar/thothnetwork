package worker

import (
	"context"
	"sync"
	"sync/atomic"

	errbuilder "github.com/ZanzyTHEbar/errbuilder-go"
)

// Pool represents a worker pool for executing tasks concurrently
type Pool struct {
	tasks       chan func()
	numWorkers  atomic.Int32
	maxWorkers  int32
	activeCount atomic.Int32
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
	started     atomic.Bool
	mu          sync.Mutex
}

// NewPool creates a new worker pool with the specified number of workers
func NewPool(maxWorkers int) *Pool {
	ctx, cancel := context.WithCancel(context.Background())
	p := &Pool{
		tasks:      make(chan func(), 100), // Buffer size of 100
		maxWorkers: int32(maxWorkers),
		ctx:        ctx,
		cancel:     cancel,
	}
	p.numWorkers.Store(0)
	p.activeCount.Store(0)
	return p
}

// Start starts the worker pool with the specified number of workers
func (p *Pool) Start(numWorkers int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.started.Load() {
		return errbuilder.GenericErr("Worker pool already started", nil)
	}

	if numWorkers <= 0 {
		return errbuilder.GenericErr("Number of workers must be greater than zero", nil)
	}

	if int32(numWorkers) > p.maxWorkers {
		numWorkers = int(p.maxWorkers)
	}

	// Start workers
	for range make([]struct{}, numWorkers) {
		p.startWorker()
	}

	p.started.Store(true)
	return nil
}

// Stop stops the worker pool
func (p *Pool) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.started.Load() {
		return errbuilder.GenericErr("Worker pool not started", nil)
	}

	// Cancel context to signal workers to stop
	p.cancel()

	// Wait for all workers to finish
	p.wg.Wait()

	// Create new context for future use
	p.ctx, p.cancel = context.WithCancel(context.Background())

	p.started.Store(false)
	p.numWorkers.Store(0)
	p.activeCount.Store(0)

	return nil
}

// Submit submits a task to the worker pool
func (p *Pool) Submit(task func()) error {
	if !p.started.Load() {
		return errbuilder.GenericErr("Worker pool not started", nil)
	}

	select {
	case p.tasks <- task:
		return nil
	case <-p.ctx.Done():
		return errbuilder.GenericErr("Worker pool stopped", nil)
	}
}

// Resize resizes the worker pool to the specified number of workers
func (p *Pool) Resize(numWorkers int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.started.Load() {
		return errbuilder.GenericErr("Worker pool not started", nil)
	}

	if numWorkers <= 0 {
		return errbuilder.GenericErr("Number of workers must be greater than zero", nil)
	}

	if int32(numWorkers) > p.maxWorkers {
		numWorkers = int(p.maxWorkers)
	}

	current := p.numWorkers.Load()
	target := int32(numWorkers)

	// Add workers if needed
	for i := current; i < target; i++ {
		p.startWorker()
	}

	// Remove workers if needed (they will exit when context is cancelled)
	if current > target {
		diff := current - target
		for i := int32(0); i < diff; i++ {
			p.tasks <- func() {
				// This is a special task that signals the worker to exit
				panic("worker exit signal")
			}
		}
	}

	return nil
}

// ActiveCount returns the number of active workers
func (p *Pool) ActiveCount() int {
	return int(p.activeCount.Load())
}

// WorkerCount returns the total number of workers
func (p *Pool) WorkerCount() int {
	return int(p.numWorkers.Load())
}

// startWorker starts a new worker
func (p *Pool) startWorker() {
	p.wg.Add(1)
	p.numWorkers.Add(1)

	go func() {
		defer func() {
			// Recover from panics
			if r := recover(); r != nil {
				// If this is our special exit signal, decrement the worker count
				if r == "worker exit signal" {
					p.numWorkers.Add(-1)
				}
				// Otherwise, it's a real panic, but we still need to decrement
				p.numWorkers.Add(-1)
			}
			p.wg.Done()
		}()

		for {
			select {
			case task := <-p.tasks:
				// Increment active count
				p.activeCount.Add(1)

				// Execute task
				func() {
					defer func() {
						// Recover from task panics
						if r := recover(); r != nil {
							// If this is our special exit signal, propagate it
							if r == "worker exit signal" {
								panic(r)
							}
							// Otherwise, just continue
						}
						// Decrement active count
						p.activeCount.Add(-1)
					}()
					task()
				}()

			case <-p.ctx.Done():
				// Context cancelled, exit
				p.numWorkers.Add(-1)
				return
			}
		}
	}()
}
