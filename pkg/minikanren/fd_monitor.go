package minikanren

// fd_monitor.go: monitoring and statistics for FD solver

import (
	"fmt"
	"sync"
	"time"
)

// SolverStats holds statistics about the FD solving process
type SolverStats struct {
	// Search statistics
	NodesExplored  int           // Number of search nodes explored
	Backtracks     int           // Number of backtracks performed
	SolutionsFound int           // Number of solutions found
	SearchTime     time.Duration // Time spent in search
	MaxDepth       int           // Maximum search depth reached

	// Propagation statistics
	PropagationCount int           // Number of propagation operations
	PropagationTime  time.Duration // Time spent in propagation
	ConstraintsAdded int           // Number of constraints added

	// Domain statistics
	InitialDomains   []BitSet // Initial domain snapshots
	FinalDomains     []BitSet // Final domain snapshots
	DomainReductions []int    // Domain size reductions per variable

	// Memory statistics
	PeakTrailSize int // Peak size of the undo trail
	PeakQueueSize int // Peak size of the propagation queue
}

// SolverMonitor provides monitoring capabilities for the FD solver
type SolverMonitor struct {
	mu        sync.Mutex
	stats     *SolverStats
	startTime time.Time
	propStart time.Time
}

// NewSolverMonitor creates a new solver monitor
func NewSolverMonitor() *SolverMonitor {
	return &SolverMonitor{
		stats: &SolverStats{
			InitialDomains:   make([]BitSet, 0),
			FinalDomains:     make([]BitSet, 0),
			DomainReductions: make([]int, 0),
		},
		startTime: time.Now(),
	}
}

// GetStats returns a copy of the current statistics
func (m *SolverMonitor) GetStats() *SolverStats {
	m.mu.Lock()
	defer m.mu.Unlock()
	stats := *m.stats
	return &stats
}

// StartPropagation marks the beginning of a propagation operation
func (m *SolverMonitor) StartPropagation() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.propStart = time.Now()
}

// EndPropagation marks the end of a propagation operation
func (m *SolverMonitor) EndPropagation() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.propStart.IsZero() {
		m.stats.PropagationTime += time.Since(m.propStart)
		m.stats.PropagationCount++
		m.propStart = time.Time{}
	}
}

// RecordBacktrack records a backtrack operation
func (m *SolverMonitor) RecordBacktrack() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stats.Backtracks++
}

// RecordNode records exploring a search node
func (m *SolverMonitor) RecordNode() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stats.NodesExplored++
}

// RecordSolution records finding a solution
func (m *SolverMonitor) RecordSolution() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stats.SolutionsFound++
}

// RecordDepth records the current search depth
func (m *SolverMonitor) RecordDepth(depth int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if depth > m.stats.MaxDepth {
		m.stats.MaxDepth = depth
	}
}

// RecordConstraint records adding a constraint
func (m *SolverMonitor) RecordConstraint() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stats.ConstraintsAdded++
}

// RecordTrailSize records the current trail size
func (m *SolverMonitor) RecordTrailSize(size int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if size > m.stats.PeakTrailSize {
		m.stats.PeakTrailSize = size
	}
}

// RecordQueueSize records the current queue size
func (m *SolverMonitor) RecordQueueSize(size int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if size > m.stats.PeakQueueSize {
		m.stats.PeakQueueSize = size
	}
}

// CaptureInitialDomains captures the initial domain state
func (m *SolverMonitor) CaptureInitialDomains(store *FDStore) {
	store.mu.Lock()
	defer store.mu.Unlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	m.stats.InitialDomains = make([]BitSet, len(store.vars))
	for i, v := range store.vars {
		m.stats.InitialDomains[i] = v.domain.Clone()
	}
}

// CaptureFinalDomains captures the final domain state and computes reductions
func (m *SolverMonitor) CaptureFinalDomains(store *FDStore) {
	store.mu.Lock()
	defer store.mu.Unlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	m.stats.FinalDomains = make([]BitSet, len(store.vars))
	m.stats.DomainReductions = make([]int, len(store.vars))

	for i, v := range store.vars {
		m.stats.FinalDomains[i] = v.domain.Clone()
		if i < len(m.stats.InitialDomains) {
			initialSize := m.stats.InitialDomains[i].Count()
			finalSize := m.stats.FinalDomains[i].Count()
			m.stats.DomainReductions[i] = initialSize - finalSize
		}
	}
}

// FinishSearch marks the end of the search process
func (m *SolverMonitor) FinishSearch() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stats.SearchTime = time.Since(m.startTime)
}

// String returns a formatted string representation of the statistics
func (s *SolverStats) String() string {
	return fmt.Sprintf(
		"Solver Statistics:\n"+
			"  Search: %d nodes, %d backtracks, %d solutions, %v time, max depth %d\n"+
			"  Propagation: %d ops, %v time, %d constraints\n"+
			"  Memory: peak trail %d, peak queue %d\n"+
			"  Domains: %d variables, avg reduction %.1f",
		s.NodesExplored, s.Backtracks, s.SolutionsFound, s.SearchTime, s.MaxDepth,
		s.PropagationCount, s.PropagationTime, s.ConstraintsAdded,
		s.PeakTrailSize, s.PeakQueueSize,
		len(s.DomainReductions), s.averageReduction(),
	)
}

// averageReduction computes the average domain size reduction
func (s *SolverStats) averageReduction() float64 {
	if len(s.DomainReductions) == 0 {
		return 0
	}
	total := 0
	for _, r := range s.DomainReductions {
		total += r
	}
	return float64(total) / float64(len(s.DomainReductions))
}
