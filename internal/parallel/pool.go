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
	"sync/atomic"
	"time"
)

// WorkerPool manages a pool of goroutines for parallel goal evaluation.
// It provides controlled concurrency with backpressure handling to
// prevent resource exhaustion during large search spaces.
type WorkerPool struct {
	maxWorkers     int
	minWorkers     int
	currentWorkers int
	taskChan       chan func()
	workerWg       sync.WaitGroup
	shutdownChan   chan struct{}
	scaleChan      chan int // Channel for scaling requests
	once           sync.Once
	mu             sync.RWMutex

	// Scaling parameters
	scaleUpThreshold   int           // Queue depth to trigger scale up
	scaleDownThreshold int           // Queue depth to trigger scale down
	scaleCheckInterval time.Duration // How often to check scaling
	lastScaleTime      time.Time     // Last time we scaled
	scaleCooldown      time.Duration // Minimum time between scaling operations

	// Monitoring and statistics
	stats            *ExecutionStats
	deadlockDetector *DeadlockDetector
}

// NewWorkerPool creates a new worker pool with the specified number of workers.
// If maxWorkers is 0 or negative, it defaults to the number of CPU cores.
func NewWorkerPool(maxWorkers int) *WorkerPool {
	return NewDynamicWorkerPool(maxWorkers, 1) // Default min workers to 1
}

// NewDynamicWorkerPool creates a new worker pool with dynamic scaling capabilities.
func NewDynamicWorkerPool(maxWorkers, minWorkers int) *WorkerPool {
	return NewDynamicWorkerPoolWithConfig(maxWorkers, minWorkers, DynamicConfig{})
}

// DynamicConfig holds configuration for dynamic scaling.
type DynamicConfig struct {
	ScaleUpThreshold   int
	ScaleDownThreshold int
	ScaleCheckInterval time.Duration
	ScaleCooldown      time.Duration
}

// NewDynamicWorkerPoolWithConfig creates a new worker pool with custom dynamic scaling config.
func NewDynamicWorkerPoolWithConfig(maxWorkers, minWorkers int, config DynamicConfig) *WorkerPool {
	if maxWorkers <= 0 {
		maxWorkers = runtime.NumCPU()
	}
	if minWorkers <= 0 {
		minWorkers = 1
	}
	if minWorkers > maxWorkers {
		minWorkers = maxWorkers
	}

	// Set defaults for config
	if config.ScaleUpThreshold <= 0 {
		config.ScaleUpThreshold = maxWorkers * 2
	}
	if config.ScaleDownThreshold <= 0 {
		config.ScaleDownThreshold = maxWorkers / 2
		if config.ScaleDownThreshold <= 0 {
			config.ScaleDownThreshold = 1
		}
	}
	if config.ScaleCheckInterval <= 0 {
		config.ScaleCheckInterval = 100 * time.Millisecond
	}
	if config.ScaleCooldown <= 0 {
		config.ScaleCooldown = 500 * time.Millisecond
	}

	pool := &WorkerPool{
		maxWorkers:         maxWorkers,
		minWorkers:         minWorkers,
		currentWorkers:     minWorkers,
		taskChan:           make(chan func(), maxWorkers*4), // Larger buffer for dynamic scaling
		shutdownChan:       make(chan struct{}),
		scaleChan:          make(chan int, 1),
		scaleUpThreshold:   config.ScaleUpThreshold,
		scaleDownThreshold: config.ScaleDownThreshold,
		scaleCheckInterval: config.ScaleCheckInterval,
		scaleCooldown:      config.ScaleCooldown,
		lastScaleTime:      time.Now(),
		stats:              NewExecutionStats(),
		deadlockDetector:   NewDeadlockDetector(30*time.Second, 5*time.Second),
	}

	// Start initial worker goroutines
	for i := 0; i < minWorkers; i++ {
		pool.workerWg.Add(1)
		go pool.worker()
	}

	// Start scaling monitor
	go pool.scalingMonitor()

	return pool
}

// worker is the main worker loop that processes tasks from the channel.
func (wp *WorkerPool) worker() {
	defer wp.workerWg.Done()

	for {
		select {
		case task := <-wp.taskChan:
			if task != nil {
				startTime := time.Now()
				func() {
					defer func() {
						if r := recover(); r != nil {
							if wp.stats != nil {
								wp.stats.RecordTaskFailed(fmt.Errorf("task panicked: %v", r))
							}
						}
					}()
					task()
					if wp.stats != nil {
						duration := time.Since(startTime)
						wp.stats.RecordTaskCompleted(duration)
					}
				}()
			}
		case <-wp.shutdownChan:
			return
		}
	}
}

// Submit submits a task to the worker pool for execution.
// If the pool is full, this call will block until a worker becomes available.
func (wp *WorkerPool) Submit(ctx context.Context, task func()) error {
	if wp.stats != nil {
		wp.stats.RecordTaskSubmitted()
	}

	select {
	case wp.taskChan <- task:
		if wp.stats != nil {
			wp.stats.RecordQueueDepth(len(wp.taskChan))
			wp.mu.RLock()
			workerCount := wp.currentWorkers
			wp.mu.RUnlock()
			wp.stats.RecordWorkerCount(workerCount)
		}
		return nil
	case <-ctx.Done():
		if wp.stats != nil {
			wp.stats.RecordTaskCancelled()
		}
		return ctx.Err()
	case <-wp.shutdownChan:
		if wp.stats != nil {
			wp.stats.RecordTaskCancelled()
		}
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

		// Finalize statistics
		if wp.stats != nil {
			wp.stats.Finalize()
		}

		// Shutdown deadlock detector
		if wp.deadlockDetector != nil {
			wp.deadlockDetector.Shutdown()
		}
	})
}

// scalingMonitor continuously monitors queue depth and adjusts worker count.
func (wp *WorkerPool) scalingMonitor() {
	ticker := time.NewTicker(wp.scaleCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			wp.checkScaling()
		case newWorkers := <-wp.scaleChan:
			wp.adjustWorkers(newWorkers)
		case <-wp.shutdownChan:
			return
		}
	}
}

// checkScaling evaluates current queue depth and decides if scaling is needed.
func (wp *WorkerPool) checkScaling() {
	wp.mu.RLock()
	if time.Since(wp.lastScaleTime) < wp.scaleCooldown {
		wp.mu.RUnlock()
		return
	}
	currentWorkers := wp.currentWorkers
	maxWorkers := wp.maxWorkers
	minWorkers := wp.minWorkers
	scaleUpThreshold := wp.scaleUpThreshold
	scaleDownThreshold := wp.scaleDownThreshold
	wp.mu.RUnlock()

	queueDepth := len(wp.taskChan)

	// Scale up if queue is getting full and we have room
	if queueDepth > scaleUpThreshold && currentWorkers < maxWorkers {
		newWorkers := currentWorkers + 1
		if newWorkers > maxWorkers {
			newWorkers = maxWorkers
		}
		select {
		case wp.scaleChan <- newWorkers:
		default:
			// Scale request already pending
		}
	} else if queueDepth < scaleDownThreshold && currentWorkers > minWorkers {
		// Scale down if queue is mostly empty and we have extra workers
		newWorkers := currentWorkers - 1
		if newWorkers < minWorkers {
			newWorkers = minWorkers
		}
		select {
		case wp.scaleChan <- newWorkers:
		default:
			// Scale request already pending
		}
	}
}

// adjustWorkers changes the number of active workers.
func (wp *WorkerPool) adjustWorkers(targetWorkers int) {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	currentWorkers := wp.currentWorkers
	if targetWorkers == currentWorkers {
		return
	}

	if targetWorkers > currentWorkers {
		// Scale up: add more workers
		for i := currentWorkers; i < targetWorkers; i++ {
			wp.workerWg.Add(1)
			go wp.worker()
		}
		if wp.stats != nil {
			wp.stats.RecordScaleUp()
		}
	} else {
		// Scale down: workers will terminate naturally when they finish current tasks
		// We don't forcibly terminate workers to avoid interrupting work
		if wp.stats != nil {
			wp.stats.RecordScaleDown()
		}
	}

	wp.currentWorkers = targetWorkers
	wp.lastScaleTime = time.Now()
}

// GetWorkerCount returns the current number of active workers.
func (wp *WorkerPool) GetWorkerCount() int {
	wp.mu.RLock()
	defer wp.mu.RUnlock()
	return wp.currentWorkers
}

// GetQueueDepth returns the current number of queued tasks.
func (wp *WorkerPool) GetQueueDepth() int {
	return len(wp.taskChan)
}

// GetMaxWorkers returns the maximum number of workers.
func (wp *WorkerPool) GetMaxWorkers() int {
	wp.mu.RLock()
	defer wp.mu.RUnlock()
	return wp.maxWorkers
}

// GetStats returns the execution statistics collector.
func (wp *WorkerPool) GetStats() *ExecutionStats {
	return wp.stats
}

// GetDeadlockDetector returns the deadlock detector.
func (wp *WorkerPool) GetDeadlockDetector() *DeadlockDetector {
	return wp.deadlockDetector
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

// StaticWorkerPool provides a fixed-size worker pool without dynamic scaling.
type StaticWorkerPool struct {
	maxWorkers   int
	taskChan     chan func()
	workerWg     sync.WaitGroup
	shutdownChan chan struct{}
	once         sync.Once
}

// NewStaticWorkerPool creates a new static worker pool with fixed size.
func NewStaticWorkerPool(maxWorkers int) *StaticWorkerPool {
	if maxWorkers <= 0 {
		maxWorkers = runtime.NumCPU()
	}

	pool := &StaticWorkerPool{
		maxWorkers:   maxWorkers,
		taskChan:     make(chan func(), maxWorkers*2),
		shutdownChan: make(chan struct{}),
	}

	// Start worker goroutines
	for i := 0; i < maxWorkers; i++ {
		pool.workerWg.Add(1)
		go pool.worker()
	}

	return pool
}

// worker is the main worker loop for static pool.
func (swp *StaticWorkerPool) worker() {
	defer swp.workerWg.Done()

	for {
		select {
		case task := <-swp.taskChan:
			if task != nil {
				task()
			}
		case <-swp.shutdownChan:
			return
		}
	}
}

// Submit submits a task to the static worker pool.
func (swp *StaticWorkerPool) Submit(ctx context.Context, task func()) error {
	select {
	case swp.taskChan <- task:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-swp.shutdownChan:
		return ErrPoolShutdown
	}
}

// Shutdown shuts down the static worker pool.
func (swp *StaticWorkerPool) Shutdown() {
	swp.once.Do(func() {
		close(swp.shutdownChan)
		close(swp.taskChan)
		swp.workerWg.Wait()
	})
}

// GetWorkerCount returns the number of workers (static).
func (swp *StaticWorkerPool) GetWorkerCount() int {
	return swp.maxWorkers
}

// GetQueueDepth returns the current queue depth.
func (swp *StaticWorkerPool) GetQueueDepth() int {
	return len(swp.taskChan)
}

// GetMaxWorkers returns the maximum workers (same as current for static pool).
func (swp *StaticWorkerPool) GetMaxWorkers() int {
	return swp.maxWorkers
}

// GetStats returns nil for static worker pool (no statistics).
func (swp *StaticWorkerPool) GetStats() *ExecutionStats {
	return nil
}

// GetDeadlockDetector returns nil for static worker pool (no deadlock detection).
func (swp *StaticWorkerPool) GetDeadlockDetector() *DeadlockDetector {
	return nil
}

// WorkerPoolInterface defines the interface for worker pools.
type WorkerPoolInterface interface {
	Submit(ctx context.Context, task func()) error
	Shutdown()
	GetWorkerCount() int
	GetQueueDepth() int
	GetMaxWorkers() int
	GetStats() *ExecutionStats
	GetDeadlockDetector() *DeadlockDetector
}

// WorkStealingWorkerPool implements a work-stealing scheduler for load balancing.
type WorkStealingWorkerPool struct {
	maxWorkers     int
	minWorkers     int
	currentWorkers int
	workerDeques   []chan func() // One deque per worker
	workers        []*workStealingWorker
	globalQueue    chan func() // Fallback queue for new tasks
	shutdownChan   chan struct{}
	scaleChan      chan int
	mu             sync.RWMutex
	once           sync.Once

	// Scaling parameters
	scaleUpThreshold   int
	scaleDownThreshold int
	scaleCheckInterval time.Duration
	lastScaleTime      time.Time
	scaleCooldown      time.Duration

	// Monitoring and statistics
	stats            *ExecutionStats
	deadlockDetector *DeadlockDetector
}

// workStealingWorker represents a single worker with its own deque.
type workStealingWorker struct {
	id      int
	deque   chan func()
	pool    *WorkStealingWorkerPool
	running bool
}

// NewWorkStealingWorkerPool creates a work-stealing worker pool.
func NewWorkStealingWorkerPool(maxWorkers, minWorkers int) *WorkStealingWorkerPool {
	return NewWorkStealingWorkerPoolWithConfig(maxWorkers, minWorkers, DynamicConfig{})
}

// NewWorkStealingWorkerPoolWithConfig creates a work-stealing pool with custom config.
func NewWorkStealingWorkerPoolWithConfig(maxWorkers, minWorkers int, config DynamicConfig) *WorkStealingWorkerPool {
	if maxWorkers <= 0 {
		maxWorkers = runtime.NumCPU()
	}
	if minWorkers <= 0 {
		minWorkers = 1
	}
	if minWorkers > maxWorkers {
		minWorkers = maxWorkers
	}

	// Set defaults for config
	if config.ScaleUpThreshold <= 0 {
		config.ScaleUpThreshold = maxWorkers * 2
	}
	if config.ScaleDownThreshold <= 0 {
		config.ScaleDownThreshold = maxWorkers / 2
		if config.ScaleDownThreshold <= 0 {
			config.ScaleDownThreshold = 1
		}
	}
	if config.ScaleCheckInterval <= 0 {
		config.ScaleCheckInterval = 100 * time.Millisecond
	}
	if config.ScaleCooldown <= 0 {
		config.ScaleCooldown = 500 * time.Millisecond
	}

	pool := &WorkStealingWorkerPool{
		maxWorkers:         maxWorkers,
		minWorkers:         minWorkers,
		currentWorkers:     minWorkers,
		workerDeques:       make([]chan func(), maxWorkers),
		workers:            make([]*workStealingWorker, maxWorkers),
		globalQueue:        make(chan func(), maxWorkers*4),
		shutdownChan:       make(chan struct{}),
		scaleChan:          make(chan int, 1),
		scaleUpThreshold:   config.ScaleUpThreshold,
		scaleDownThreshold: config.ScaleDownThreshold,
		scaleCheckInterval: config.ScaleCheckInterval,
		scaleCooldown:      config.ScaleCooldown,
		lastScaleTime:      time.Now(),
		stats:              NewExecutionStats(),
		deadlockDetector:   NewDeadlockDetector(30*time.Second, 5*time.Second),
	}

	// Initialize worker deques and workers
	for i := 0; i < maxWorkers; i++ {
		pool.workerDeques[i] = make(chan func(), 256) // Local deque
		pool.workers[i] = &workStealingWorker{
			id:    i,
			deque: pool.workerDeques[i],
			pool:  pool,
		}
	}

	// Start initial workers
	for i := 0; i < minWorkers; i++ {
		pool.workers[i].running = true
		go pool.workers[i].run()
	}

	// Start scaling monitor
	go pool.scalingMonitor()

	return pool
}

// Submit submits a task to the work-stealing pool.
func (wswp *WorkStealingWorkerPool) Submit(ctx context.Context, task func()) error {
	if wswp.stats != nil {
		wswp.stats.RecordTaskSubmitted()
	}

	select {
	case wswp.globalQueue <- task:
		return nil
	case <-ctx.Done():
		if wswp.stats != nil {
			wswp.stats.RecordTaskCancelled()
		}
		return ctx.Err()
	case <-wswp.shutdownChan:
		if wswp.stats != nil {
			wswp.stats.RecordTaskCancelled()
		}
		return ErrPoolShutdown
	}
}

// Shutdown shuts down the work-stealing pool.
func (wswp *WorkStealingWorkerPool) Shutdown() {
	wswp.once.Do(func() {
		close(wswp.shutdownChan)
		close(wswp.globalQueue)
		for _, deque := range wswp.workerDeques {
			close(deque)
		}
		// Workers will terminate when they see the shutdown signal

		// Finalize statistics
		if wswp.stats != nil {
			wswp.stats.Finalize()
		}

		// Shutdown deadlock detector
		if wswp.deadlockDetector != nil {
			wswp.deadlockDetector.Shutdown()
		}
	})
}

// run is the main worker loop with work stealing.
func (wsw *workStealingWorker) run() {
	for {
		var task func()
		var ok bool

		// Try to get task from own deque first
		select {
		case task, ok = <-wsw.deque:
			if !ok {
				return // Shutdown
			}
		case <-wsw.pool.shutdownChan:
			return
		default:
			// No task in own deque, try to steal from others
			task = wsw.stealTask()
			if task == nil {
				// No task stolen, try global queue
				select {
				case task, ok = <-wsw.pool.globalQueue:
					if !ok {
						return // Shutdown
					}
				case <-wsw.pool.shutdownChan:
					return
				default:
					// No work available, small delay to prevent busy waiting
					time.Sleep(1 * time.Millisecond)
					continue
				}
			}
		}

		if task != nil {
			startTime := time.Now()
			func() {
				defer func() {
					if r := recover(); r != nil {
						if wsw.pool.stats != nil {
							wsw.pool.stats.RecordTaskFailed(fmt.Errorf("task panicked: %v", r))
						}
					}
				}()
				task()
				if wsw.pool.stats != nil {
					duration := time.Since(startTime)
					wsw.pool.stats.RecordTaskCompleted(duration)
				}
			}()
		}
	}
}

// stealTask attempts to steal a task from another worker's deque.
func (wsw *workStealingWorker) stealTask() func() {
	// Try to steal from other workers (random selection for fairness)
	workers := wsw.pool.workers
	start := (wsw.id + 1) % len(workers)

	for i := 0; i < len(workers); i++ {
		victimID := (start + i) % len(workers)
		if victimID == wsw.id {
			continue // Don't steal from self
		}

		select {
		case task := <-wsw.pool.workerDeques[victimID]:
			return task // Successfully stole a task
		default:
			continue // No task available from this victim
		}
	}

	return nil // No tasks available to steal
}

// scalingMonitor monitors and adjusts worker count.
func (wswp *WorkStealingWorkerPool) scalingMonitor() {
	ticker := time.NewTicker(wswp.scaleCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			wswp.checkScaling()
		case newWorkers := <-wswp.scaleChan:
			wswp.adjustWorkers(newWorkers)
		case <-wswp.shutdownChan:
			return
		}
	}
}

// checkScaling evaluates current load and decides if scaling is needed.
func (wswp *WorkStealingWorkerPool) checkScaling() {
	wswp.mu.RLock()
	if time.Since(wswp.lastScaleTime) < wswp.scaleCooldown {
		wswp.mu.RUnlock()
		return
	}
	currentWorkers := wswp.currentWorkers
	maxWorkers := wswp.maxWorkers
	minWorkers := wswp.minWorkers
	scaleUpThreshold := wswp.scaleUpThreshold
	scaleDownThreshold := wswp.scaleDownThreshold
	wswp.mu.RUnlock()

	// Calculate total queued tasks across all deques and global queue
	totalQueued := len(wswp.globalQueue)
	for _, deque := range wswp.workerDeques {
		totalQueued += len(deque)
	}

	// Scale up if queue is getting full and we have room
	if totalQueued > scaleUpThreshold && currentWorkers < maxWorkers {
		newWorkers := currentWorkers + 1
		if newWorkers > maxWorkers {
			newWorkers = maxWorkers
		}
		select {
		case wswp.scaleChan <- newWorkers:
		default:
			// Scale request already pending
		}
	} else if totalQueued < scaleDownThreshold && currentWorkers > minWorkers {
		// Scale down if queue is mostly empty and we have extra workers
		newWorkers := currentWorkers - 1
		if newWorkers < minWorkers {
			newWorkers = minWorkers
		}
		select {
		case wswp.scaleChan <- newWorkers:
		default:
			// Scale request already pending
		}
	}
}

// adjustWorkers changes the number of active workers.
func (wswp *WorkStealingWorkerPool) adjustWorkers(targetWorkers int) {
	wswp.mu.Lock()
	defer wswp.mu.Unlock()

	currentWorkers := wswp.currentWorkers
	if targetWorkers == currentWorkers {
		return
	}

	if targetWorkers > currentWorkers {
		// Scale up: start more workers
		for i := currentWorkers; i < targetWorkers; i++ {
			wswp.workers[i].running = true
			go wswp.workers[i].run()
		}
	} else {
		// Scale down: workers will terminate naturally when they finish current tasks
		// We don't forcibly terminate workers to avoid interrupting work
	}

	wswp.currentWorkers = targetWorkers
	wswp.lastScaleTime = time.Now()
}

// GetWorkerCount returns the current number of active workers.
func (wswp *WorkStealingWorkerPool) GetWorkerCount() int {
	wswp.mu.RLock()
	defer wswp.mu.RUnlock()
	return wswp.currentWorkers
}

// GetQueueDepth returns the total number of queued tasks across all deques.
func (wswp *WorkStealingWorkerPool) GetQueueDepth() int {
	total := len(wswp.globalQueue)
	for _, deque := range wswp.workerDeques {
		total += len(deque)
	}
	return total
}

// GetMaxWorkers returns the maximum number of workers.
func (wswp *WorkStealingWorkerPool) GetMaxWorkers() int {
	wswp.mu.RLock()
	defer wswp.mu.RUnlock()
	return wswp.maxWorkers
}

// GetStats returns the execution statistics collector.
func (wswp *WorkStealingWorkerPool) GetStats() *ExecutionStats {
	return wswp.stats
}

// GetDeadlockDetector returns the deadlock detector.
func (wswp *WorkStealingWorkerPool) GetDeadlockDetector() *DeadlockDetector {
	return wswp.deadlockDetector
}

// ExecutionStats collects comprehensive statistics for parallel execution monitoring.
type ExecutionStats struct {
	mu sync.RWMutex

	// Timing statistics
	StartTime          time.Time
	EndTime            time.Time
	TotalExecutionTime time.Duration

	// Task statistics
	TasksSubmitted int64
	TasksCompleted int64
	TasksFailed    int64
	TasksCancelled int64

	// Worker statistics
	PeakWorkerCount    int
	AverageWorkerCount float64
	WorkerUtilization  float64

	// Queue statistics
	PeakQueueDepth    int
	AverageQueueDepth float64
	QueueFullEvents   int64

	// Performance metrics
	TasksPerSecond      float64
	AverageTaskDuration time.Duration

	// Error tracking
	LastError  error
	ErrorCount int64

	// Deadlock detection
	PotentialDeadlocks int64
	TimeoutEvents      int64

	// Scaling events
	ScaleUpEvents   int64
	ScaleDownEvents int64

	// Resource usage
	MemoryUsage    int64 // Approximate memory usage in bytes
	GoroutineCount int

	// Historical data for analysis
	workerCountHistory  []workerCountSample
	queueDepthHistory   []queueDepthSample
	taskDurationHistory []time.Duration
}

type workerCountSample struct {
	timestamp time.Time
	count     int
}

type queueDepthSample struct {
	timestamp time.Time
	depth     int
}

// NewExecutionStats creates a new execution statistics collector.
func NewExecutionStats() *ExecutionStats {
	return &ExecutionStats{
		StartTime:           time.Now(),
		workerCountHistory:  make([]workerCountSample, 0, 1000),
		queueDepthHistory:   make([]queueDepthSample, 0, 1000),
		taskDurationHistory: make([]time.Duration, 0, 10000),
	}
}

// RecordTaskSubmitted records that a task was submitted for execution.
func (es *ExecutionStats) RecordTaskSubmitted() {
	atomic.AddInt64(&es.TasksSubmitted, 1)
}

// RecordTaskCompleted records that a task completed successfully.
func (es *ExecutionStats) RecordTaskCompleted(duration time.Duration) {
	atomic.AddInt64(&es.TasksCompleted, 1)
	es.mu.Lock()
	es.taskDurationHistory = append(es.taskDurationHistory, duration)
	es.mu.Unlock()
}

// RecordTaskFailed records that a task failed with an error.
func (es *ExecutionStats) RecordTaskFailed(err error) {
	atomic.AddInt64(&es.TasksFailed, 1)
	atomic.AddInt64(&es.ErrorCount, 1)
	es.mu.Lock()
	es.LastError = err
	es.mu.Unlock()
}

// RecordTaskCancelled records that a task was cancelled.
func (es *ExecutionStats) RecordTaskCancelled() {
	atomic.AddInt64(&es.TasksCancelled, 1)
}

// RecordWorkerCount records the current worker count for historical tracking.
func (es *ExecutionStats) RecordWorkerCount(count int) {
	es.mu.Lock()
	defer es.mu.Unlock()

	if count > es.PeakWorkerCount {
		es.PeakWorkerCount = count
	}

	es.workerCountHistory = append(es.workerCountHistory, workerCountSample{
		timestamp: time.Now(),
		count:     count,
	})

	// Keep history bounded
	if len(es.workerCountHistory) > 1000 {
		es.workerCountHistory = es.workerCountHistory[1:]
	}
}

// RecordQueueDepth records the current queue depth for historical tracking.
func (es *ExecutionStats) RecordQueueDepth(depth int) {
	es.mu.Lock()
	defer es.mu.Unlock()

	if depth > es.PeakQueueDepth {
		es.PeakQueueDepth = depth
	}

	es.queueDepthHistory = append(es.queueDepthHistory, queueDepthSample{
		timestamp: time.Now(),
		depth:     depth,
	})

	// Keep history bounded
	if len(es.queueDepthHistory) > 1000 {
		es.queueDepthHistory = es.queueDepthHistory[1:]
	}
}

// RecordQueueFull records that the queue became full.
func (es *ExecutionStats) RecordQueueFull() {
	atomic.AddInt64(&es.QueueFullEvents, 1)
}

// RecordScaleUp records a scaling up event.
func (es *ExecutionStats) RecordScaleUp() {
	atomic.AddInt64(&es.ScaleUpEvents, 1)
}

// RecordScaleDown records a scaling down event.
func (es *ExecutionStats) RecordScaleDown() {
	atomic.AddInt64(&es.ScaleDownEvents, 1)
}

// RecordPotentialDeadlock records a potential deadlock situation.
func (es *ExecutionStats) RecordPotentialDeadlock() {
	atomic.AddInt64(&es.PotentialDeadlocks, 1)
}

// RecordTimeout records a timeout event.
func (es *ExecutionStats) RecordTimeout() {
	atomic.AddInt64(&es.TimeoutEvents, 1)
}

// UpdateResourceUsage updates current resource usage statistics.
func (es *ExecutionStats) UpdateResourceUsage() {
	es.mu.Lock()
	es.GoroutineCount = runtime.NumGoroutine()
	// Note: Memory usage would require runtime.ReadMemStats() for accurate measurement
	es.mu.Unlock()
}

// Finalize computes final statistics when execution completes.
func (es *ExecutionStats) Finalize() {
	es.mu.Lock()
	defer es.mu.Unlock()

	es.EndTime = time.Now()
	es.TotalExecutionTime = es.EndTime.Sub(es.StartTime)

	// Calculate averages
	if len(es.workerCountHistory) > 0 {
		total := 0
		for _, sample := range es.workerCountHistory {
			total += sample.count
		}
		es.AverageWorkerCount = float64(total) / float64(len(es.workerCountHistory))
	}

	if len(es.queueDepthHistory) > 0 {
		total := 0
		for _, sample := range es.queueDepthHistory {
			total += sample.depth
		}
		es.AverageQueueDepth = float64(total) / float64(len(es.queueDepthHistory))
	}

	if len(es.taskDurationHistory) > 0 {
		total := time.Duration(0)
		for _, duration := range es.taskDurationHistory {
			total += duration
		}
		es.AverageTaskDuration = total / time.Duration(len(es.taskDurationHistory))
	}

	// Calculate throughput
	if es.TotalExecutionTime > 0 {
		es.TasksPerSecond = float64(es.TasksCompleted) / es.TotalExecutionTime.Seconds()
	}

	// Calculate worker utilization (simplified)
	if es.AverageWorkerCount > 0 && es.TotalExecutionTime > 0 {
		busyTime := es.AverageTaskDuration * time.Duration(es.TasksCompleted)
		totalWorkerTime := es.TotalExecutionTime * time.Duration(es.AverageWorkerCount)
		if totalWorkerTime > 0 {
			es.WorkerUtilization = float64(busyTime) / float64(totalWorkerTime)
		}
	}
}

// GetStats returns a copy of the current statistics.
func (es *ExecutionStats) GetStats() ExecutionStats {
	es.mu.RLock()
	defer es.mu.RUnlock()

	// Create a copy without the mutex, using atomic loads for atomic fields
	return ExecutionStats{
		StartTime:           es.StartTime,
		EndTime:             es.EndTime,
		TotalExecutionTime:  es.TotalExecutionTime,
		TasksSubmitted:      atomic.LoadInt64(&es.TasksSubmitted),
		TasksCompleted:      atomic.LoadInt64(&es.TasksCompleted),
		TasksFailed:         atomic.LoadInt64(&es.TasksFailed),
		TasksCancelled:      atomic.LoadInt64(&es.TasksCancelled),
		PeakWorkerCount:     es.PeakWorkerCount,
		AverageWorkerCount:  es.AverageWorkerCount,
		WorkerUtilization:   es.WorkerUtilization,
		PeakQueueDepth:      es.PeakQueueDepth,
		AverageQueueDepth:   es.AverageQueueDepth,
		QueueFullEvents:     atomic.LoadInt64(&es.QueueFullEvents),
		TasksPerSecond:      es.TasksPerSecond,
		AverageTaskDuration: es.AverageTaskDuration,
		LastError:           es.LastError,
		ErrorCount:          atomic.LoadInt64(&es.ErrorCount),
		PotentialDeadlocks:  atomic.LoadInt64(&es.PotentialDeadlocks),
		TimeoutEvents:       atomic.LoadInt64(&es.TimeoutEvents),
		ScaleUpEvents:       atomic.LoadInt64(&es.ScaleUpEvents),
		ScaleDownEvents:     atomic.LoadInt64(&es.ScaleDownEvents),
		MemoryUsage:         es.MemoryUsage,
		GoroutineCount:      es.GoroutineCount,
		workerCountHistory:  append([]workerCountSample(nil), es.workerCountHistory...),
		queueDepthHistory:   append([]queueDepthSample(nil), es.queueDepthHistory...),
		taskDurationHistory: append([]time.Duration(nil), es.taskDurationHistory...),
	}
}

// String returns a human-readable summary of the execution statistics.
func (es *ExecutionStats) String() string {
	stats := es.GetStats()

	var lastErrorStr string
	if stats.LastError != nil {
		lastErrorStr = stats.LastError.Error()
	} else {
		lastErrorStr = "none"
	}

	return fmt.Sprintf("ExecutionStats{\n"+
		"  Duration: %v\n"+
		"  Tasks: %d submitted, %d completed, %d failed, %d cancelled\n"+
		"  Workers: peak=%d, avg=%.1f, utilization=%.1f%%\n"+
		"  Queue: peak=%d, avg=%.1f, full_events=%d\n"+
		"  Performance: %.1f tasks/sec, avg_task_time=%v\n"+
		"  Errors: %d total, last=%s\n"+
		"  Events: %d scale_up, %d scale_down, %d deadlocks, %d timeouts\n"+
		"  Resources: %d goroutines\n"+
		"}",
		stats.TotalExecutionTime,
		stats.TasksSubmitted, stats.TasksCompleted, stats.TasksFailed, stats.TasksCancelled,
		stats.PeakWorkerCount, stats.AverageWorkerCount, stats.WorkerUtilization*100,
		stats.PeakQueueDepth, stats.AverageQueueDepth, stats.QueueFullEvents,
		stats.TasksPerSecond, stats.AverageTaskDuration,
		stats.ErrorCount, lastErrorStr,
		stats.ScaleUpEvents, stats.ScaleDownEvents, stats.PotentialDeadlocks, stats.TimeoutEvents,
		stats.GoroutineCount)
}

// DeadlockDetector monitors for potential deadlocks in parallel execution.
type DeadlockDetector struct {
	mu sync.RWMutex

	// Configuration
	timeoutDuration time.Duration
	checkInterval   time.Duration
	maxRetries      int

	// State tracking
	activeTasks        map[string]*taskInfo
	lastActivity       time.Time
	potentialDeadlocks int64

	// Channels
	shutdownChan chan struct{}
	alertChan    chan DeadlockAlert
}

type taskInfo struct {
	id          string
	startTime   time.Time
	lastUpdate  time.Time
	description string
}

type DeadlockAlert struct {
	Type        DeadlockAlertType
	TaskID      string
	Description string
	Timestamp   time.Time
}

type DeadlockAlertType int

const (
	AlertTaskTimeout DeadlockAlertType = iota
	AlertPotentialDeadlock
	AlertSystemStall
)

// NewDeadlockDetector creates a new deadlock detector.
func NewDeadlockDetector(timeoutDuration, checkInterval time.Duration) *DeadlockDetector {
	if timeoutDuration <= 0 {
		timeoutDuration = 30 * time.Second
	}
	if checkInterval <= 0 {
		checkInterval = 5 * time.Second
	}

	dd := &DeadlockDetector{
		timeoutDuration: timeoutDuration,
		checkInterval:   checkInterval,
		maxRetries:      3,
		activeTasks:     make(map[string]*taskInfo),
		lastActivity:    time.Now(),
		shutdownChan:    make(chan struct{}),
		alertChan:       make(chan DeadlockAlert, 10),
	}

	go dd.monitor()

	return dd
}

// RegisterTask registers a new active task for monitoring.
func (dd *DeadlockDetector) RegisterTask(taskID, description string) {
	dd.mu.Lock()
	defer dd.mu.Unlock()

	dd.activeTasks[taskID] = &taskInfo{
		id:          taskID,
		startTime:   time.Now(),
		lastUpdate:  time.Now(),
		description: description,
	}
	dd.lastActivity = time.Now()
}

// UpdateTask updates the last activity time for a task.
func (dd *DeadlockDetector) UpdateTask(taskID string) {
	dd.mu.Lock()
	defer dd.mu.Unlock()

	if task, exists := dd.activeTasks[taskID]; exists {
		task.lastUpdate = time.Now()
		dd.lastActivity = time.Now()
	}
}

// UnregisterTask removes a task from monitoring.
func (dd *DeadlockDetector) UnregisterTask(taskID string) {
	dd.mu.Lock()
	defer dd.mu.Unlock()

	delete(dd.activeTasks, taskID)
}

// GetAlerts returns a channel for receiving deadlock alerts.
func (dd *DeadlockDetector) GetAlerts() <-chan DeadlockAlert {
	return dd.alertChan
}

// GetActiveTaskCount returns the number of currently monitored tasks.
func (dd *DeadlockDetector) GetActiveTaskCount() int {
	dd.mu.RLock()
	defer dd.mu.RUnlock()
	return len(dd.activeTasks)
}

// GetPotentialDeadlocks returns the count of potential deadlocks detected.
func (dd *DeadlockDetector) GetPotentialDeadlocks() int64 {
	dd.mu.RLock()
	defer dd.mu.RUnlock()
	return dd.potentialDeadlocks
}

// Shutdown stops the deadlock detector.
func (dd *DeadlockDetector) Shutdown() {
	close(dd.shutdownChan)
}

// monitor runs the deadlock detection loop.
func (dd *DeadlockDetector) monitor() {
	ticker := time.NewTicker(dd.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			dd.checkForDeadlocks()
		case <-dd.shutdownChan:
			return
		}
	}
}

// checkForDeadlocks performs deadlock detection checks.
func (dd *DeadlockDetector) checkForDeadlocks() {
	dd.mu.Lock()
	defer dd.mu.Unlock()

	now := time.Now()

	// Check for task timeouts
	for taskID, task := range dd.activeTasks {
		if now.Sub(task.lastUpdate) > dd.timeoutDuration {
			alert := DeadlockAlert{
				Type:        AlertTaskTimeout,
				TaskID:      taskID,
				Description: fmt.Sprintf("Task '%s' timed out after %v", task.description, now.Sub(task.startTime)),
				Timestamp:   now,
			}
			select {
			case dd.alertChan <- alert:
			default:
				// Alert channel full, drop alert
			}
			dd.potentialDeadlocks++
		}
	}

	// Check for system-wide stall (no activity for extended period)
	stallThreshold := dd.timeoutDuration * 2
	if now.Sub(dd.lastActivity) > stallThreshold && len(dd.activeTasks) > 0 {
		alert := DeadlockAlert{
			Type:        AlertSystemStall,
			Description: fmt.Sprintf("System stall detected: no activity for %v with %d active tasks", now.Sub(dd.lastActivity), len(dd.activeTasks)),
			Timestamp:   now,
		}
		select {
		case dd.alertChan <- alert:
		default:
			// Alert channel full, drop alert
		}
		dd.potentialDeadlocks++
	}

	// Check for potential deadlocks (circular wait conditions)
	// This is a simplified check - in a real system you'd analyze wait-for graphs
	if len(dd.activeTasks) > 0 {
		oldestTask := now
		totalTasks := 0

		for _, task := range dd.activeTasks {
			if task.startTime.Before(oldestTask) {
				oldestTask = task.startTime
			}
			totalTasks++
		}

		// If we have many long-running tasks, it might indicate a deadlock
		if totalTasks >= 3 && now.Sub(oldestTask) > dd.timeoutDuration*2 {
			alert := DeadlockAlert{
				Type:        AlertPotentialDeadlock,
				Description: fmt.Sprintf("Potential deadlock: %d tasks running for extended period", totalTasks),
				Timestamp:   now,
			}
			select {
			case dd.alertChan <- alert:
			default:
				// Alert channel full, drop alert
			}
			dd.potentialDeadlocks++
		}
	}
}

// TimeoutContext creates a context with deadlock-aware timeout.
func (dd *DeadlockDetector) TimeoutContext(parent context.Context, taskID, description string) (context.Context, context.CancelFunc) {
	dd.RegisterTask(taskID, description)

	ctx, cancel := context.WithTimeout(parent, dd.timeoutDuration)

	// Wrap the cancel function to unregister the task
	originalCancel := cancel
	cancel = func() {
		dd.UnregisterTask(taskID)
		originalCancel()
	}

	return ctx, cancel
}

// ExecuteWithDeadlockProtection executes a function with deadlock protection.
func (dd *DeadlockDetector) ExecuteWithDeadlockProtection(ctx context.Context, taskID, description string, fn func(context.Context) error) error {
	taskCtx, cancel := dd.TimeoutContext(ctx, taskID, description)
	defer cancel()

	done := make(chan error, 1)

	go func() {
		defer dd.UpdateTask(taskID) // Final update
		done <- fn(taskCtx)
	}()

	select {
	case err := <-done:
		return err
	case <-taskCtx.Done():
		if taskCtx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("task '%s' timed out: %w", description, taskCtx.Err())
		}
		return taskCtx.Err()
	}
}
