// Package minikanren provides constraint solving with parallel search capabilities.
// This file implements parallel backtracking search using a shared work queue (channel).

package minikanren

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
)

// ParallelSearchConfig holds configuration for parallel backtracking search.
type ParallelSearchConfig struct {
	// NumWorkers is the number of parallel worker goroutines.
	// If 0 or negative, defaults to runtime.NumCPU().
	NumWorkers int

	// WorkQueueSize is the buffer size for the work channel.
	// Larger values allow more work to be queued, potentially improving
	// load balancing at the cost of memory.
	WorkQueueSize int
}

// DefaultParallelSearchConfig returns the default parallel search configuration.
func DefaultParallelSearchConfig() *ParallelSearchConfig {
	return &ParallelSearchConfig{
		NumWorkers:    runtime.NumCPU(),
		WorkQueueSize: 1000,
	}
}

// workItem represents a single node in the search tree to explore.
type workItem struct {
	state      *SolverState // Current solver state
	varID      int          // Variable to assign next
	values     []int        // Possible values for the variable
	valueIndex int          // Index of next value to try
	depth      int          // Depth in search tree (for debugging)
}

// SolveParallel performs parallel backtracking search to find solutions.
// Uses multiple workers sharing a work queue via a buffered channel.
//
// Parameters:
//   - ctx: Context for cancellation
//   - numWorkers: Number of parallel workers (0 = runtime.NumCPU())
//   - maxSolutions: Maximum solutions to find (0 = find all)
//
// Returns found solutions and any error encountered.
func (s *Solver) SolveParallel(ctx context.Context, numWorkers, maxSolutions int) ([][]int, error) {
	if numWorkers <= 0 {
		numWorkers = runtime.NumCPU()
	}

	// Perform initial propagation
	initialState := (*SolverState)(nil)
	propagatedState, err := s.propagate(initialState)
	if err != nil {
		return nil, err
	}

	// Check if already solved after propagation
	if s.isComplete(propagatedState) {
		solution := s.extractSolution(propagatedState)
		return [][]int{solution}, nil
	}

	// Select initial variable and values
	varID, values := s.selectVariable(propagatedState)
	if varID == -1 {
		// No variables to assign
		return nil, nil
	}

	// Create channels for work and solutions
	workChan := make(chan *workItem, 1000)
	solutionChan := make(chan []int, numWorkers*2)

	// Track solutions and outstanding tasks
	var solutionsFound atomic.Int64
	var tasksWG sync.WaitGroup

	// Add initial work
	workChan <- &workItem{
		state:      propagatedState,
		varID:      varID,
		values:     values,
		valueIndex: 0,
		depth:      0,
	}
	tasksWG.Add(1)

	// Start workers
	workerCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			s.parallelWorker(workerCtx, cancel, workerID, workChan, solutionChan, &tasksWG, &solutionsFound, maxSolutions)
		}(i)
	}

	// Close work channel when all tasks are completed so workers exit cleanly
	go func() {
		tasksWG.Wait()
		close(workChan)
	}()

	// Close solution channel when workers are done
	go func() {
		wg.Wait()
		close(solutionChan)
	}()

	// Collect solutions. If we hit the limit, cancel workers and keep draining
	// the channel to avoid deadlocks from blocked senders.
	solutions := make([][]int, 0)
	reachedLimit := false
	for solution := range solutionChan {
		if !reachedLimit {
			solutions = append(solutions, solution)
			if maxSolutions > 0 && len(solutions) >= maxSolutions {
				reachedLimit = true
				cancel() // stop producing, but continue draining
			}
		}
		// else: discard additional solutions until solutionChan closes
	}

	return solutions, ctx.Err()
}

// parallelWorker processes work items from the shared work channel.
func (s *Solver) parallelWorker(ctx context.Context, cancel context.CancelFunc, workerID int, workChan chan *workItem, solutionChan chan []int, tasksWG *sync.WaitGroup, solutionsFound *atomic.Int64, maxSolutions int) {
	for {
		select {
		case <-ctx.Done():
			// Drain remaining queued items so tasksWG can reach zero
			for work := range workChan {
				s.ReleaseState(work.state)
				tasksWG.Done()
			}
			return
		case work, ok := <-workChan:
			if !ok {
				// All tasks completed and channel closed
				return
			}
			// Process this work item
			s.processWork(ctx, work, workChan, solutionChan, solutionsFound, tasksWG, maxSolutions)
			// Release the work item's state now that we're done with it
			s.ReleaseState(work.state)
			// Mark this task as done
			tasksWG.Done()
		}
	}
}

// processWork processes a single work item, trying all values for the variable.
// Does NOT release work.state - caller is responsible.
func (s *Solver) processWork(ctx context.Context, work *workItem, workChan chan *workItem, solutionChan chan []int, solutionsFound *atomic.Int64, tasksWG *sync.WaitGroup, maxSolutions int) {
	// Try each value for this variable
	for work.valueIndex < len(work.values) {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Check solution limit
		if maxSolutions > 0 && solutionsFound.Load() >= int64(maxSolutions) {
			return
		}

		value := work.values[work.valueIndex]
		work.valueIndex++

		// Assign value
		domain := s.GetDomain(work.state, work.varID)
		newDomain := NewBitSetDomainFromValues(domain.MaxValue(), []int{value})
		newState, _ := s.SetDomain(work.state, work.varID, newDomain)

		// Propagate
		propagatedState, err := s.propagate(newState)
		if err != nil {
			s.ReleaseState(newState)
			continue
		}

		// Check if complete
		if s.isComplete(propagatedState) {
			solution := s.extractSolution(propagatedState)
			solutionsFound.Add(1)

			select {
			case solutionChan <- solution:
			case <-ctx.Done():
				s.ReleaseState(propagatedState)
				return
			}

			s.ReleaseState(propagatedState)
			continue
		}

		// Select next variable
		nextVarID, nextValues := s.selectVariable(propagatedState)
		if nextVarID == -1 {
			s.ReleaseState(propagatedState)
			continue
		}

		// Add new work to channel
		// NOTE: The new work item now owns propagatedState,
		// so we don't release it here
		newWork := &workItem{
			state:      propagatedState,
			varID:      nextVarID,
			values:     nextValues,
			valueIndex: 0,
			depth:      work.depth + 1,
		}

		// Try to enqueue new work. Register task before enqueue; roll back if not queued.
		tasksWG.Add(1)
		select {
		case workChan <- newWork:
			// queued successfully
		case <-ctx.Done():
			tasksWG.Done()
			s.ReleaseState(propagatedState)
			return
		default:
			// Channel full: roll back and process inline to guarantee progress
			tasksWG.Done()
			s.processWork(ctx, newWork, workChan, solutionChan, solutionsFound, tasksWG, maxSolutions)
			// We're the caller; release its state after processing
			s.ReleaseState(propagatedState)
		}
	}
}
