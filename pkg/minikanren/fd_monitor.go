package minikanren

// fd_monitor.go: lock-free monitoring and statistics for FD solver

import (
	"fmt"
	"sync/atomic"
	"time"
)

// SolverStats holds statistics about the FD solving process.
// All fields use atomic operations for lock-free updates.
type SolverStats struct {
	// Search statistics
	NodesExplored  int64         // Number of search nodes explored
	Backtracks     int64         // Number of backtracks performed
	SolutionsFound int64         // Number of solutions found
	SearchTime     time.Duration // Time spent in search
	MaxDepth       int64         // Maximum search depth reached

	// Propagation statistics
	PropagationCount int64 // Number of propagation operations
	PropagationTime  int64 // Time spent in propagation (nanoseconds)
	ConstraintsAdded int64 // Number of constraints added

	// Domain statistics (not used in Phase 1, will be implemented in Phase 2)
	// InitialDomains   []BitSet // Initial domain snapshots
	// FinalDomains     []BitSet // Final domain snapshots
	// DomainReductions []int    // Domain size reductions per variable

	// Memory statistics
	PeakTrailSize int64 // Peak size of the undo trail
	PeakQueueSize int64 // Peak size of the propagation queue
}

// SolverMonitor provides lock-free monitoring capabilities for the FD solver.
// All operations use atomic instructions for safe concurrent access without locks.
// Designed to match the lock-free copy-on-write architecture of the solver.
type SolverMonitor struct {
	stats     SolverStats
	startTime time.Time
	propStart atomic.Int64 // Propagation start time in nanoseconds (0 = not started)
}

// NewSolverMonitor creates a new solver monitor.
// Uses atomic operations for lock-free statistics collection.
func NewSolverMonitor() *SolverMonitor {
	return &SolverMonitor{
		startTime: time.Now(),
	}
}

// GetStats returns a snapshot of the current statistics.
// Returns nil if the monitor is nil.
// Safe to call concurrently from multiple goroutines.
func (m *SolverMonitor) GetStats() *SolverStats {
	if m == nil {
		return nil
	}
	// Atomic loads to get consistent snapshot
	return &SolverStats{
		NodesExplored:    atomic.LoadInt64(&m.stats.NodesExplored),
		Backtracks:       atomic.LoadInt64(&m.stats.Backtracks),
		SolutionsFound:   atomic.LoadInt64(&m.stats.SolutionsFound),
		SearchTime:       m.stats.SearchTime, // Only written once at end
		MaxDepth:         atomic.LoadInt64(&m.stats.MaxDepth),
		PropagationCount: atomic.LoadInt64(&m.stats.PropagationCount),
		PropagationTime:  atomic.LoadInt64(&m.stats.PropagationTime),
		ConstraintsAdded: atomic.LoadInt64(&m.stats.ConstraintsAdded),
		PeakTrailSize:    atomic.LoadInt64(&m.stats.PeakTrailSize),
		PeakQueueSize:    atomic.LoadInt64(&m.stats.PeakQueueSize),
	}
}

// StartPropagation marks the beginning of a propagation operation.
// Safe to call on nil monitor. Lock-free.
func (m *SolverMonitor) StartPropagation() {
	if m == nil {
		return
	}
	m.propStart.Store(time.Now().UnixNano())
}

// EndPropagation marks the end of a propagation operation.
// Safe to call on nil monitor. Lock-free.
func (m *SolverMonitor) EndPropagation() {
	if m == nil {
		return
	}
	startNano := m.propStart.Load()
	if startNano != 0 {
		elapsed := time.Now().UnixNano() - startNano
		atomic.AddInt64(&m.stats.PropagationTime, elapsed)
		atomic.AddInt64(&m.stats.PropagationCount, 1)
		m.propStart.Store(0)
	}
}

// RecordBacktrack records a backtrack operation.
// Safe to call on nil monitor. Lock-free.
func (m *SolverMonitor) RecordBacktrack() {
	if m == nil {
		return
	}
	atomic.AddInt64(&m.stats.Backtracks, 1)
}

// RecordNode records exploring a search node.
// Safe to call on nil monitor. Lock-free.
func (m *SolverMonitor) RecordNode() {
	if m == nil {
		return
	}
	atomic.AddInt64(&m.stats.NodesExplored, 1)
}

// RecordSolution records finding a solution.
// Safe to call on nil monitor. Lock-free.
func (m *SolverMonitor) RecordSolution() {
	if m == nil {
		return
	}
	atomic.AddInt64(&m.stats.SolutionsFound, 1)
}

// RecordDepth records the current search depth.
// Safe to call on nil monitor. Lock-free using compare-and-swap.
func (m *SolverMonitor) RecordDepth(depth int) {
	if m == nil {
		return
	}
	depth64 := int64(depth)
	// Atomic max update using compare-and-swap loop
	for {
		old := atomic.LoadInt64(&m.stats.MaxDepth)
		if depth64 <= old {
			break
		}
		if atomic.CompareAndSwapInt64(&m.stats.MaxDepth, old, depth64) {
			break
		}
	}
}

// RecordConstraint records adding a constraint.
// Safe to call on nil monitor. Lock-free.
func (m *SolverMonitor) RecordConstraint() {
	if m == nil {
		return
	}
	atomic.AddInt64(&m.stats.ConstraintsAdded, 1)
}

// RecordTrailSize records the current trail size.
// Safe to call on nil monitor. Lock-free using compare-and-swap.
func (m *SolverMonitor) RecordTrailSize(size int) {
	if m == nil {
		return
	}
	size64 := int64(size)
	// Atomic max update using compare-and-swap loop
	for {
		old := atomic.LoadInt64(&m.stats.PeakTrailSize)
		if size64 <= old {
			break
		}
		if atomic.CompareAndSwapInt64(&m.stats.PeakTrailSize, old, size64) {
			break
		}
	}
}

// RecordQueueSize records the current queue size.
// Safe to call on nil monitor. Lock-free using compare-and-swap.
func (m *SolverMonitor) RecordQueueSize(size int) {
	if m == nil {
		return
	}
	size64 := int64(size)
	// Atomic max update using compare-and-swap loop
	for {
		old := atomic.LoadInt64(&m.stats.PeakQueueSize)
		if size64 <= old {
			break
		}
		if atomic.CompareAndSwapInt64(&m.stats.PeakQueueSize, old, size64) {
			break
		}
	}
}

// CaptureInitialDomains captures the initial domain state.
// Safe to call on nil monitor.
// Phase 2 implementation - currently a no-op.
func (m *SolverMonitor) CaptureInitialDomains(store *FDStore) {
	if m == nil {
		return
	}
	// Phase 2: Will capture domain snapshots here
}

// CaptureFinalDomains captures the final domain state and computes reductions.
// Safe to call on nil monitor.
// Phase 2 implementation - currently a no-op.
func (m *SolverMonitor) CaptureFinalDomains(store *FDStore) {
	if m == nil {
		return
	}
	// Phase 2: Will capture domain snapshots and compute reductions here
}

// FinishSearch marks the end of the search process.
// Safe to call on nil monitor. Only called once at end, no synchronization needed.
func (m *SolverMonitor) FinishSearch() {
	if m == nil {
		return
	}
	m.stats.SearchTime = time.Since(m.startTime)
} // String returns a formatted string representation of the statistics.
func (s *SolverStats) String() string {
	return fmt.Sprintf(
		"Solver Statistics:\n"+
			"  Nodes Explored:  %d\n"+
			"  Backtracks:      %d\n"+
			"  Solutions:       %d\n"+
			"  Max Depth:       %d\n"+
			"  Search Time:     %v\n"+
			"  Propagations:    %d\n"+
			"  Prop Time:       %v\n"+
			"  Constraints:     %d\n"+
			"  Peak Trail:      %d\n"+
			"  Peak Queue:      %d\n",
		s.NodesExplored,
		s.Backtracks,
		s.SolutionsFound,
		s.MaxDepth,
		s.SearchTime,
		s.PropagationCount,
		time.Duration(s.PropagationTime),
		s.ConstraintsAdded,
		s.PeakTrailSize,
		s.PeakQueueSize,
	)
}
