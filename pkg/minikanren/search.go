package minikanren

import (
	"context"
)

// DFSSearch implements depth-first search with backtracking.
// This is the most common search strategy for constraint satisfaction problems.
// It explores solutions by going deep into the search tree before backtracking.
type DFSSearch struct{}

// NewDFSSearch creates a new depth-first search strategy.
func NewDFSSearch() *DFSSearch {
	return &DFSSearch{}
}

// Search implements depth-first search with iterative backtracking.
func (s *DFSSearch) Search(ctx context.Context, store *FDStore, labeling LabelingStrategy, limit int) ([][]int, error) {
	solutions := make([][]int, 0)

	// Iterative backtracking using a stack
	type frame struct {
		snap    int   // trail snapshot
		varID   int   // variable being tried
		valIdx  int   // index in choices
		choices []int // available values
	}
	stack := []frame{}

	// Initial propagation
	store.mu.Lock()
	if store.monitor != nil {
		store.monitor.StartPropagation()
	}
	if err := store.propagateLocked(); err != nil {
		store.mu.Unlock()
		if store.monitor != nil {
			store.monitor.EndPropagation()
		}
		return nil, err
	}
	if store.monitor != nil {
		store.monitor.EndPropagation()
	}
	store.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Check for initial solution
	store.mu.Lock()
	allAssigned := true
	for _, v := range store.vars {
		if !v.domain.IsSingleton() {
			allAssigned = false
			break
		}
	}
	if allAssigned {
		sol := make([]int, len(store.vars))
		for i, v := range store.vars {
			sol[i] = v.domain.SingletonValue()
		}
		solutions = append(solutions, sol)
		store.mu.Unlock()
		return solutions, nil
	}
	store.mu.Unlock()

	// Push initial frame
	varID, choices := labeling.SelectVariable(store)
	if varID == -1 {
		return solutions, nil
	}
	stack = append(stack, frame{snap: store.snapshot(), varID: varID, valIdx: 0, choices: choices})

	for len(stack) > 0 {
		// Check cancellation
		select {
		case <-ctx.Done():
			if store.monitor != nil {
				store.monitor.FinishSearch()
				store.monitor.CaptureFinalDomains(store)
			}
			return solutions, ctx.Err()
		default:
		}

		if store.monitor != nil {
			store.monitor.RecordDepth(len(stack))
			store.monitor.RecordTrailSize(len(store.trail))
			store.monitor.RecordQueueSize(len(store.queue))
		}

		f := &stack[len(stack)-1]

		if f.valIdx >= len(f.choices) {
			// Backtrack
			if store.monitor != nil {
				store.monitor.RecordBacktrack()
			}
			store.undo(f.snap)
			stack = stack[:len(stack)-1]
			continue
		}

		if store.monitor != nil {
			store.monitor.RecordNode()
		}

		val := f.choices[f.valIdx]
		f.valIdx++

		// Try assignment
		if err := store.Assign(store.idToVar[f.varID], val); err != nil {
			continue
		}

		// Check if complete
		store.mu.Lock()
		allAssigned := true
		for _, v := range store.vars {
			if !v.domain.IsSingleton() {
				allAssigned = false
				break
			}
		}
		store.mu.Unlock()
		if allAssigned {
			store.mu.Lock()
			sol := make([]int, len(store.vars))
			for i, v := range store.vars {
				sol[i] = v.domain.SingletonValue()
			}
			solutions = append(solutions, sol)
			store.mu.Unlock()
			store.undo(f.snap)
			if store.monitor != nil {
				store.monitor.RecordSolution()
			}
			if limit > 0 && len(solutions) >= limit {
				if store.monitor != nil {
					store.monitor.FinishSearch()
					store.monitor.CaptureFinalDomains(store)
				}
				return solutions, nil
			}
			continue
		}

		// Find next variable
		nextVarID, nextChoices := labeling.SelectVariable(store)
		if nextVarID == -1 {
			store.undo(f.snap)
			continue
		}

		// Push new frame
		stack = append(stack, frame{snap: store.snapshot(), varID: nextVarID, valIdx: 0, choices: nextChoices})
	}

	if store.monitor != nil {
		store.monitor.FinishSearch()
		store.monitor.CaptureFinalDomains(store)
	}

	return solutions, nil
}

// Name returns the search strategy name.
func (s *DFSSearch) Name() string {
	return "dfs"
}

// Description returns detailed information about the DFS strategy.
func (s *DFSSearch) Description() string {
	return "Depth-first search: explores solutions by going deep into the search tree before backtracking, most common for CSP"
}

// SupportsPruning returns true as DFS supports basic pruning through backtracking.
func (s *DFSSearch) SupportsPruning() bool {
	return true
}

// BFSSearch implements breadth-first search.
// This strategy explores all solutions at a given depth before proceeding deeper.
// Useful for finding shortest solutions or when solution quality matters more than speed.
type BFSSearch struct {
	maxDepth int // Maximum search depth to prevent memory explosion
}

// NewBFSSearch creates a new breadth-first search strategy with default max depth.
func NewBFSSearch() *BFSSearch {
	return &BFSSearch{maxDepth: 1000}
}

// NewBFSSearchWithDepth creates a new BFS strategy with specified maximum depth.
func NewBFSSearchWithDepth(maxDepth int) *BFSSearch {
	return &BFSSearch{maxDepth: maxDepth}
}

// Search implements breadth-first search using a queue.
func (s *BFSSearch) Search(ctx context.Context, store *FDStore, labeling LabelingStrategy, limit int) ([][]int, error) {
	solutions := make([][]int, 0)

	// Initial propagation
	store.mu.Lock()
	if store.monitor != nil {
		store.monitor.StartPropagation()
	}
	if err := store.propagateLocked(); err != nil {
		store.mu.Unlock()
		if store.monitor != nil {
			store.monitor.EndPropagation()
		}
		return nil, err
	}
	if store.monitor != nil {
		store.monitor.EndPropagation()
	}
	store.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Check for initial solution
	store.mu.Lock()
	allAssigned := true
	for _, v := range store.vars {
		if !v.domain.IsSingleton() {
			allAssigned = false
			break
		}
	}
	if allAssigned {
		sol := make([]int, len(store.vars))
		for i, v := range store.vars {
			sol[i] = v.domain.SingletonValue()
		}
		solutions = append(solutions, sol)
		store.mu.Unlock()
		return solutions, nil
	}
	store.mu.Unlock()

	// BFS using a queue of partial assignments
	type state struct {
		snapshot int   // store snapshot
		assigned []int // assigned values by variable ID
		depth    int   // current depth
	}
	queue := []state{}

	// Initialize with empty assignment
	queue = append(queue, state{snapshot: store.snapshot(), assigned: make([]int, len(store.vars)), depth: 0})

	for len(queue) > 0 {
		// Check cancellation
		select {
		case <-ctx.Done():
			if store.monitor != nil {
				store.monitor.FinishSearch()
				store.monitor.CaptureFinalDomains(store)
			}
			return solutions, ctx.Err()
		default:
		}

		current := queue[0]
		queue = queue[1:]

		// Restore state
		store.undo(current.snapshot)
		for i, val := range current.assigned {
			if val != 0 { // 0 means unassigned
				if err := store.Assign(store.vars[i], val); err != nil {
					// Invalid assignment, skip this branch
					continue
				}
			}
		}

		if store.monitor != nil {
			store.monitor.RecordDepth(current.depth)
			store.monitor.RecordTrailSize(len(store.trail))
			store.monitor.RecordQueueSize(len(queue))
		}

		// Check if complete
		store.mu.Lock()
		allAssigned := true
		for _, v := range store.vars {
			if !v.domain.IsSingleton() {
				allAssigned = false
				break
			}
		}
		store.mu.Unlock()
		if allAssigned {
			store.mu.Lock()
			sol := make([]int, len(store.vars))
			for i, v := range store.vars {
				sol[i] = v.domain.SingletonValue()
			}
			solutions = append(solutions, sol)
			store.mu.Unlock()
			if store.monitor != nil {
				store.monitor.RecordSolution()
			}
			if limit > 0 && len(solutions) >= limit {
				if store.monitor != nil {
					store.monitor.FinishSearch()
					store.monitor.CaptureFinalDomains(store)
				}
				return solutions, nil
			}
			continue
		}

		// Don't expand beyond max depth
		if current.depth >= s.maxDepth {
			continue
		}

		// Find next variable to branch on
		varID, choices := labeling.SelectVariable(store)
		if varID == -1 {
			continue
		}

		// Create child states for each choice
		for _, val := range choices {
			newAssigned := make([]int, len(current.assigned))
			copy(newAssigned, current.assigned)
			newAssigned[varID] = val

			child := state{
				snapshot: store.snapshot(),
				assigned: newAssigned,
				depth:    current.depth + 1,
			}
			queue = append(queue, child)
		}
	}

	if store.monitor != nil {
		store.monitor.FinishSearch()
		store.monitor.CaptureFinalDomains(store)
	}

	return solutions, nil
}

// Name returns the search strategy name.
func (s *BFSSearch) Name() string {
	return "bfs"
}

// Description returns detailed information about the BFS strategy.
func (s *BFSSearch) Description() string {
	return "Breadth-first search: explores all solutions at current depth before going deeper, useful for finding optimal solutions"
}

// SupportsPruning returns false as BFS doesn't support advanced pruning.
func (s *BFSSearch) SupportsPruning() bool {
	return false
}

// LimitedDepthSearch implements depth-limited search.
// This prevents excessive memory usage by limiting search depth.
// Useful for large problems where full search is impractical.
type LimitedDepthSearch struct {
	maxDepth int
}

// NewLimitedDepthSearch creates a new limited depth search strategy.
func NewLimitedDepthSearch(maxDepth int) *LimitedDepthSearch {
	if maxDepth <= 0 {
		maxDepth = 100
	}
	return &LimitedDepthSearch{maxDepth: maxDepth}
}

// Search implements depth-limited DFS.
func (s *LimitedDepthSearch) Search(ctx context.Context, store *FDStore, labeling LabelingStrategy, limit int) ([][]int, error) {
	solutions := make([][]int, 0)

	// Iterative backtracking with depth limit
	type frame struct {
		snap    int
		varID   int
		valIdx  int
		choices []int
		depth   int
	}
	stack := []frame{}

	// Initial propagation
	store.mu.Lock()
	if store.monitor != nil {
		store.monitor.StartPropagation()
	}
	if err := store.propagateLocked(); err != nil {
		store.mu.Unlock()
		if store.monitor != nil {
			store.monitor.EndPropagation()
		}
		return nil, err
	}
	if store.monitor != nil {
		store.monitor.EndPropagation()
	}
	store.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Check for initial solution
	store.mu.Lock()
	allAssigned := true
	for _, v := range store.vars {
		if !v.domain.IsSingleton() {
			allAssigned = false
			break
		}
	}
	if allAssigned {
		sol := make([]int, len(store.vars))
		for i, v := range store.vars {
			sol[i] = v.domain.SingletonValue()
		}
		solutions = append(solutions, sol)
		store.mu.Unlock()
		return solutions, nil
	}
	store.mu.Unlock()

	// Push initial frame
	varID, choices := labeling.SelectVariable(store)
	if varID == -1 {
		return solutions, nil
	}
	stack = append(stack, frame{snap: store.snapshot(), varID: varID, valIdx: 0, choices: choices, depth: 1})

	for len(stack) > 0 {
		// Check cancellation
		select {
		case <-ctx.Done():
			if store.monitor != nil {
				store.monitor.FinishSearch()
				store.monitor.CaptureFinalDomains(store)
			}
			return solutions, ctx.Err()
		default:
		}

		if store.monitor != nil {
			store.monitor.RecordDepth(len(stack))
			store.monitor.RecordTrailSize(len(store.trail))
			store.monitor.RecordQueueSize(len(store.queue))
		}

		f := &stack[len(stack)-1]

		if f.valIdx >= len(f.choices) || f.depth > s.maxDepth {
			// Backtrack or depth limit reached
			if store.monitor != nil {
				if f.depth > s.maxDepth {
					store.monitor.RecordBacktrack() // Count as backtrack due to depth limit
				} else {
					store.monitor.RecordBacktrack()
				}
			}
			store.undo(f.snap)
			stack = stack[:len(stack)-1]
			continue
		}

		if store.monitor != nil {
			store.monitor.RecordNode()
		}

		val := f.choices[f.valIdx]
		f.valIdx++

		// Try assignment
		if err := store.Assign(store.idToVar[f.varID], val); err != nil {
			continue
		}

		// Check if complete
		store.mu.Lock()
		allAssigned := true
		for _, v := range store.vars {
			if !v.domain.IsSingleton() {
				allAssigned = false
				break
			}
		}
		store.mu.Unlock()
		if allAssigned {
			store.mu.Lock()
			sol := make([]int, len(store.vars))
			for i, v := range store.vars {
				sol[i] = v.domain.SingletonValue()
			}
			solutions = append(solutions, sol)
			store.mu.Unlock()
			store.undo(f.snap)
			if store.monitor != nil {
				store.monitor.RecordSolution()
			}
			if limit > 0 && len(solutions) >= limit {
				if store.monitor != nil {
					store.monitor.FinishSearch()
					store.monitor.CaptureFinalDomains(store)
				}
				return solutions, nil
			}
			continue
		}

		// Find next variable
		nextVarID, nextChoices := labeling.SelectVariable(store)
		if nextVarID == -1 {
			store.undo(f.snap)
			continue
		}

		// Push new frame (increment depth)
		stack = append(stack, frame{snap: store.snapshot(), varID: nextVarID, valIdx: 0, choices: nextChoices, depth: f.depth + 1})
	}

	if store.monitor != nil {
		store.monitor.FinishSearch()
		store.monitor.CaptureFinalDomains(store)
	}

	return solutions, nil
}

// Name returns the search strategy name.
func (s *LimitedDepthSearch) Name() string {
	return "limited-depth"
}

// Description returns detailed information about the limited depth strategy.
func (s *LimitedDepthSearch) Description() string {
	return "Limited depth search: restricts search depth to prevent memory explosion, useful for large problems"
}

// SupportsPruning returns true as it supports depth-based pruning.
func (s *LimitedDepthSearch) SupportsPruning() bool {
	return true
}

// IterativeDeepeningSearch implements iterative deepening depth-first search.
// This combines the space efficiency of DFS with the optimality of BFS by
// progressively increasing depth limits.
type IterativeDeepeningSearch struct {
	maxDepth  int
	increment int
}

// NewIterativeDeepeningSearch creates a new iterative deepening search strategy.
func NewIterativeDeepeningSearch(maxDepth, increment int) *IterativeDeepeningSearch {
	if maxDepth <= 0 {
		maxDepth = 100
	}
	if increment <= 0 {
		increment = 1
	}
	return &IterativeDeepeningSearch{maxDepth: maxDepth, increment: increment}
}

// Search implements iterative deepening by running multiple limited depth searches.
func (s *IterativeDeepeningSearch) Search(ctx context.Context, store *FDStore, labeling LabelingStrategy, limit int) ([][]int, error) {
	allSolutions := make([][]int, 0)

	for depth := s.increment; depth <= s.maxDepth; depth += s.increment {
		// Check cancellation
		select {
		case <-ctx.Done():
			return allSolutions, ctx.Err()
		default:
		}

		// Create a fresh store for this depth iteration
		// Note: This is a simplified implementation. In practice, we'd need to copy the store state.
		search := NewLimitedDepthSearch(depth)
		solutions, err := search.Search(ctx, store, labeling, limit-len(allSolutions))
		if err != nil {
			return nil, err
		}

		// Add new solutions (avoiding duplicates)
		for _, sol := range solutions {
			found := false
			for _, existing := range allSolutions {
				if slicesEqual(sol, existing) {
					found = true
					break
				}
			}
			if !found {
				allSolutions = append(allSolutions, sol)
				if limit > 0 && len(allSolutions) >= limit {
					return allSolutions, nil
				}
			}
		}
	}

	return allSolutions, nil
}

// slicesEqual checks if two int slices are equal.
func slicesEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Name returns the search strategy name.
func (s *IterativeDeepeningSearch) Name() string {
	return "iterative-deepening"
}

// Description returns detailed information about the iterative deepening strategy.
func (s *IterativeDeepeningSearch) Description() string {
	return "Iterative deepening: combines DFS space efficiency with BFS optimality by increasing depth limits progressively"
}

// SupportsPruning returns true as it supports depth-based pruning.
func (s *IterativeDeepeningSearch) SupportsPruning() bool {
	return true
}
