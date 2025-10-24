package minikanren

import (
	"context"
	"gokando/internal/parallel"
	"runtime"
	"sync"
)

// ParallelConfig holds configuration for parallel goal execution.
type ParallelConfig struct {
	// MaxWorkers is the maximum number of concurrent workers.
	// If 0, defaults to runtime.NumCPU().
	MaxWorkers int

	// MaxQueueSize is the maximum number of pending tasks.
	// If 0, defaults to MaxWorkers * 10.
	MaxQueueSize int

	// EnableBackpressure enables backpressure control to prevent
	// memory exhaustion during large search spaces.
	EnableBackpressure bool

	// RateLimit sets the maximum operations per second.
	// If 0, no rate limiting is applied.
	RateLimit int
}

// DefaultParallelConfig returns a default configuration for parallel execution.
func DefaultParallelConfig() *ParallelConfig {
	return &ParallelConfig{
		MaxWorkers:         runtime.NumCPU(),
		MaxQueueSize:       runtime.NumCPU() * 10,
		EnableBackpressure: true,
		RateLimit:          0, // No rate limiting by default
	}
}

// ParallelExecutor manages parallel execution of miniKanren goals.
type ParallelExecutor struct {
	config           *ParallelConfig
	workerPool       *parallel.WorkerPool
	backpressureCtrl *parallel.BackpressureController
	rateLimiter      *parallel.RateLimiter
	mu               sync.RWMutex
	shutdown         bool
}

// NewParallelExecutor creates a new parallel executor with the given configuration.
func NewParallelExecutor(config *ParallelConfig) *ParallelExecutor {
	if config == nil {
		config = DefaultParallelConfig()
	}

	if config.MaxWorkers <= 0 {
		config.MaxWorkers = runtime.NumCPU()
	}

	if config.MaxQueueSize <= 0 {
		config.MaxQueueSize = config.MaxWorkers * 10
	}

	pe := &ParallelExecutor{
		config:     config,
		workerPool: parallel.NewWorkerPool(config.MaxWorkers),
	}

	if config.EnableBackpressure {
		pe.backpressureCtrl = parallel.NewBackpressureController(config.MaxQueueSize)
	}

	if config.RateLimit > 0 {
		pe.rateLimiter = parallel.NewRateLimiter(config.RateLimit)
	}

	return pe
}

// Shutdown gracefully shuts down the parallel executor.
func (pe *ParallelExecutor) Shutdown() {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	if pe.shutdown {
		return
	}

	pe.shutdown = true
	pe.workerPool.Shutdown()

	if pe.rateLimiter != nil {
		pe.rateLimiter.Close()
	}
}

// ParallelDisj creates a disjunction goal that evaluates all sub-goals
// in parallel using the parallel executor. This can significantly improve
// performance when dealing with computationally intensive goals or
// large search spaces.
func (pe *ParallelExecutor) ParallelDisj(goals ...Goal) Goal {
	if len(goals) == 0 {
		return Failure
	}

	if len(goals) == 1 {
		return goals[0]
	}

	return func(ctx context.Context, store ConstraintStore) *Stream {
		stream := NewStream()

		go func() {
			defer stream.Close()

			// Check if executor is shutdown
			pe.mu.RLock()
			if pe.shutdown {
				pe.mu.RUnlock()
				return
			}
			pe.mu.RUnlock()

			var wg sync.WaitGroup
			resultChan := make(chan ConstraintStore, len(goals)*2)

			// Execute each goal in parallel
			for _, goal := range goals {
				wg.Add(1)

				goalToExecute := goal
				task := func() {
					defer wg.Done()

					// Apply backpressure if enabled
					if pe.backpressureCtrl != nil {
						if err := pe.backpressureCtrl.CheckBackpressure(ctx); err != nil {
							return
						}
						pe.backpressureCtrl.AddLoad(1)
						defer pe.backpressureCtrl.RemoveLoad(1)
					}

					// Apply rate limiting if enabled
					if pe.rateLimiter != nil {
						if err := pe.rateLimiter.Wait(ctx); err != nil {
							return
						}
					}

					// Execute the goal
					goalStream := goalToExecute(ctx, store)

					// Forward all results from this goal
					for {
						select {
						case <-ctx.Done():
							return
						default:
						}

						subs, hasMore := goalStream.Take(1)
						if len(subs) == 0 {
							if !hasMore {
								break
							}
							continue
						}

						select {
						case resultChan <- subs[0]:
						case <-ctx.Done():
							return
						}
					}
				}

				// Submit task to worker pool
				if err := pe.workerPool.Submit(ctx, task); err != nil {
					wg.Done() // Balance the Add(1) above
					continue
				}
			}

			// Start a goroutine to close resultChan when all workers are done
			go func() {
				wg.Wait()
				close(resultChan)
			}()

			// Forward all results to the output stream
			for result := range resultChan {
				select {
				case <-ctx.Done():
					return
				default:
					stream.Put(result)
				}
			}
		}()

		return stream
	}
}

// ParallelRun executes a goal in parallel and returns up to n solutions.
// This function creates a parallel executor, runs the goal, and cleans up.
func ParallelRun(n int, goalFunc func(*Var) Goal) []Term {
	return ParallelRunWithConfig(n, goalFunc, nil)
}

// ParallelRunWithConfig executes a goal in parallel with custom configuration.
func ParallelRunWithConfig(n int, goalFunc func(*Var) Goal, config *ParallelConfig) []Term {
	ctx := context.Background()
	return ParallelRunWithContext(ctx, n, goalFunc, config)
}

// ParallelRunWithContext executes a goal in parallel with context and configuration.
func ParallelRunWithContext(ctx context.Context, n int, goalFunc func(*Var) Goal, config *ParallelConfig) []Term {
	executor := NewParallelExecutor(config)
	defer executor.Shutdown()

	q := Fresh("q")
	goal := goalFunc(q)

	initialStore := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, initialStore)

	solutions, _ := stream.Take(n)

	var results []Term
	for _, store := range solutions {
		value := store.GetSubstitution().Walk(q)
		results = append(results, value)
	}

	return results
}

// ParallelStream represents a stream that can be evaluated in parallel.
// It wraps the standard Stream with additional parallel capabilities.
type ParallelStream struct {
	*Stream
	executor *ParallelExecutor
	ctx      context.Context
}

// NewParallelStream creates a new parallel stream with the given executor.
func NewParallelStream(ctx context.Context, executor *ParallelExecutor) *ParallelStream {
	return &ParallelStream{
		Stream:   NewStream(),
		executor: executor,
		ctx:      ctx,
	}
}

// ParallelMap applies a function to each constraint store in the stream in parallel.
func (ps *ParallelStream) ParallelMap(fn func(ConstraintStore) ConstraintStore) *ParallelStream {
	resultStream := NewParallelStream(ps.ctx, ps.executor)

	go func() {
		defer resultStream.Close()

		var wg sync.WaitGroup
		resultChan := make(chan ConstraintStore, ps.executor.config.MaxWorkers*2)

		// Process constraint stores in parallel
		for {
			stores, hasMore := ps.Take(ps.executor.config.MaxWorkers)
			if len(stores) == 0 {
				if !hasMore {
					break
				}
				continue
			}

			for _, store := range stores {
				wg.Add(1)

				storeToProcess := store
				task := func() {
					defer wg.Done()

					// Apply backpressure if enabled
					if ps.executor.backpressureCtrl != nil {
						if err := ps.executor.backpressureCtrl.CheckBackpressure(ps.ctx); err != nil {
							return
						}
						ps.executor.backpressureCtrl.AddLoad(1)
						defer ps.executor.backpressureCtrl.RemoveLoad(1)
					}

					// Apply rate limiting if enabled
					if ps.executor.rateLimiter != nil {
						if err := ps.executor.rateLimiter.Wait(ps.ctx); err != nil {
							return
						}
					}

					result := fn(storeToProcess)
					if result != nil {
						select {
						case resultChan <- result:
						case <-ps.ctx.Done():
						}
					}
				}

				if err := ps.executor.workerPool.Submit(ps.ctx, task); err != nil {
					wg.Done() // Balance the Add(1) above
					continue
				}
			}
		}

		// Wait for all tasks to complete and close result channel
		go func() {
			wg.Wait()
			close(resultChan)
		}()

		// Forward results
		for result := range resultChan {
			resultStream.Put(result)
		}
	}()

	return resultStream
}

// ParallelFilter filters constraint stores in the stream in parallel.
func (ps *ParallelStream) ParallelFilter(predicate func(ConstraintStore) bool) *ParallelStream {
	return ps.ParallelMap(func(store ConstraintStore) ConstraintStore {
		if predicate(store) {
			return store
		}
		return nil
	})
}

// Collect gathers all constraint stores from the parallel stream.
func (ps *ParallelStream) Collect() []ConstraintStore {
	var results []ConstraintStore

	for {
		stores, hasMore := ps.Take(100) // Take in batches
		results = append(results, stores...)

		if !hasMore {
			break
		}
	}

	return results
}
