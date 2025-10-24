// Package parallel provides advanced parallel execution capabilities
// for miniKanren goals. This package contains internal utilities
// for managing concurrent goal evaluation with proper resource
// management and backpressure control.
package parallel

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"
)

// WorkerPool manages a pool of goroutines for parallel goal evaluation.
// It provides controlled concurrency with backpressure handling to
// prevent resource exhaustion during large search spaces.
type WorkerPool struct {
	maxWorkers   int
	taskChan     chan func()
	workerWg     sync.WaitGroup
	shutdownChan chan struct{}
	once         sync.Once
}

// NewWorkerPool creates a new worker pool with the specified number of workers.
// If maxWorkers is 0 or negative, it defaults to the number of CPU cores.
func NewWorkerPool(maxWorkers int) *WorkerPool {
	if maxWorkers <= 0 {
		maxWorkers = runtime.NumCPU()
	}

	pool := &WorkerPool{
		maxWorkers:   maxWorkers,
		taskChan:     make(chan func(), maxWorkers*2), // Buffered channel for backpressure
		shutdownChan: make(chan struct{}),
	}

	// Start worker goroutines
	for i := 0; i < maxWorkers; i++ {
		pool.workerWg.Add(1)
		go pool.worker()
	}

	return pool
}

// worker is the main worker loop that processes tasks from the channel.
func (wp *WorkerPool) worker() {
	defer wp.workerWg.Done()

	for {
		select {
		case task := <-wp.taskChan:
			if task != nil {
				task()
			}
		case <-wp.shutdownChan:
			return
		}
	}
}

// Submit submits a task to the worker pool for execution.
// If the pool is full, this call will block until a worker becomes available.
func (wp *WorkerPool) Submit(ctx context.Context, task func()) error {
	select {
	case wp.taskChan <- task:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-wp.shutdownChan:
		return ErrPoolShutdown
	}
}

// Shutdown gracefully shuts down the worker pool, waiting for all
// currently executing tasks to complete.
func (wp *WorkerPool) Shutdown() {
	wp.once.Do(func() {
		close(wp.shutdownChan)
		close(wp.taskChan)
		wp.workerWg.Wait()
	})
}

// ErrPoolShutdown is returned when trying to submit tasks to a shutdown pool.
var ErrPoolShutdown = fmt.Errorf("worker pool has been shutdown")

// StreamMerger combines multiple streams into a single output stream
// while maintaining fairness and preventing any single stream from
// dominating the output.
type StreamMerger struct {
	outputChan chan interface{}
	doneChan   chan struct{}
	wg         sync.WaitGroup
	mu         sync.Mutex
	closed     bool
}

// NewStreamMerger creates a new stream merger.
func NewStreamMerger() *StreamMerger {
	return &StreamMerger{
		outputChan: make(chan interface{}),
		doneChan:   make(chan struct{}),
	}
}

// AddStream adds an input stream to the merger.
// The merger will read from this stream and forward items to the output.
func (sm *StreamMerger) AddStream(inputChan <-chan interface{}) {
	sm.wg.Add(1)
	go func() {
		defer sm.wg.Done()

		for {
			select {
			case item, ok := <-inputChan:
				if !ok {
					return // Input stream closed
				}

				select {
				case sm.outputChan <- item:
				case <-sm.doneChan:
					return // Merger is shutting down
				}

			case <-sm.doneChan:
				return // Merger is shutting down
			}
		}
	}()
}

// Output returns the output channel for reading merged items.
func (sm *StreamMerger) Output() <-chan interface{} {
	return sm.outputChan
}

// Close closes the merger and all associated resources.
// After calling Close, no more items will be output.
func (sm *StreamMerger) Close() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.closed {
		return
	}

	sm.closed = true
	close(sm.doneChan)

	// Wait for all input streams to finish
	go func() {
		sm.wg.Wait()
		close(sm.outputChan)
	}()
}

// RateLimiter provides rate limiting functionality to prevent
// overwhelming downstream consumers during intensive goal evaluation.
type RateLimiter struct {
	ticker   *time.Ticker
	tokens   chan struct{}
	shutdown chan struct{}
	once     sync.Once
}

// NewRateLimiter creates a new rate limiter that allows up to
// tokensPerSecond operations per second.
func NewRateLimiter(tokensPerSecond int) *RateLimiter {
	if tokensPerSecond <= 0 {
		tokensPerSecond = 1000 // Default rate
	}

	interval := time.Second / time.Duration(tokensPerSecond)
	rl := &RateLimiter{
		ticker:   time.NewTicker(interval),
		tokens:   make(chan struct{}, tokensPerSecond),
		shutdown: make(chan struct{}),
	}

	// Fill initial token bucket
	for i := 0; i < tokensPerSecond; i++ {
		rl.tokens <- struct{}{}
	}

	// Start token refill goroutine
	go rl.refillTokens()

	return rl
}

// refillTokens continuously refills the token bucket at the specified rate.
func (rl *RateLimiter) refillTokens() {
	for {
		select {
		case <-rl.ticker.C:
			select {
			case rl.tokens <- struct{}{}:
			default:
				// Token bucket is full, drop the token
			}
		case <-rl.shutdown:
			rl.ticker.Stop()
			return
		}
	}
}

// Wait blocks until a token is available or the context is cancelled.
func (rl *RateLimiter) Wait(ctx context.Context) error {
	select {
	case <-rl.tokens:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-rl.shutdown:
		return ErrLimiterShutdown
	}
}

// Close shuts down the rate limiter and releases all resources.
func (rl *RateLimiter) Close() {
	rl.once.Do(func() {
		close(rl.shutdown)
	})
}

// ErrLimiterShutdown is returned when trying to wait on a shutdown limiter.
var ErrLimiterShutdown = fmt.Errorf("rate limiter has been shutdown")

// LoadBalancer distributes work across multiple workers using
// a round-robin strategy to ensure fair distribution.
type LoadBalancer struct {
	workers []Worker
	current int64
	mu      sync.Mutex
}

// Worker represents a worker that can process tasks.
type Worker interface {
	Process(ctx context.Context, task interface{}) error
	ID() string
}

// NewLoadBalancer creates a new load balancer with the given workers.
func NewLoadBalancer(workers []Worker) *LoadBalancer {
	return &LoadBalancer{
		workers: workers,
	}
}

// Submit submits a task to the next available worker using round-robin.
func (lb *LoadBalancer) Submit(ctx context.Context, task interface{}) error {
	if len(lb.workers) == 0 {
		return fmt.Errorf("no workers available")
	}

	lb.mu.Lock()
	worker := lb.workers[lb.current%int64(len(lb.workers))]
	lb.current++
	lb.mu.Unlock()

	return worker.Process(ctx, task)
}

// BackpressureController manages backpressure in the goal evaluation pipeline
// to prevent memory exhaustion during large or infinite search spaces.
type BackpressureController struct {
	maxQueueSize  int
	currentLoad   int64
	highWaterMark int
	lowWaterMark  int
	paused        bool
	pauseChan     chan struct{}
	resumeChan    chan struct{}
	mu            sync.RWMutex
}

// NewBackpressureController creates a new backpressure controller.
func NewBackpressureController(maxQueueSize int) *BackpressureController {
	if maxQueueSize <= 0 {
		maxQueueSize = 1000
	}

	return &BackpressureController{
		maxQueueSize:  maxQueueSize,
		highWaterMark: int(float64(maxQueueSize) * 0.8), // Pause at 80%
		lowWaterMark:  int(float64(maxQueueSize) * 0.2), // Resume at 20%
		pauseChan:     make(chan struct{}),
		resumeChan:    make(chan struct{}),
	}
}

// CheckBackpressure checks if backpressure should be applied.
// Returns true if the caller should pause, false otherwise.
func (bc *BackpressureController) CheckBackpressure(ctx context.Context) error {
	bc.mu.RLock()
	shouldPause := bc.paused
	bc.mu.RUnlock()

	if shouldPause {
		select {
		case <-bc.resumeChan:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}

// AddLoad adds to the current load and checks if backpressure should be applied.
func (bc *BackpressureController) AddLoad(amount int) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	bc.currentLoad += int64(amount)

	if !bc.paused && int(bc.currentLoad) >= bc.highWaterMark {
		bc.paused = true
		close(bc.pauseChan)
		bc.pauseChan = make(chan struct{})
	}
}

// RemoveLoad removes from the current load and checks if backpressure should be released.
func (bc *BackpressureController) RemoveLoad(amount int) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	bc.currentLoad -= int64(amount)
	if bc.currentLoad < 0 {
		bc.currentLoad = 0
	}

	if bc.paused && int(bc.currentLoad) <= bc.lowWaterMark {
		bc.paused = false
		close(bc.resumeChan)
		bc.resumeChan = make(chan struct{})
	}
}

// CurrentLoad returns the current load level.
func (bc *BackpressureController) CurrentLoad() int64 {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.currentLoad
}
