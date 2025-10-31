package minikanren

import (
	"context"
	"errors"
	"fmt"
	"math/bits"
	"math/rand"
	"runtime"
	"sort"
	"sync"
)

// VariableOrderingHeuristic defines strategies for selecting the next variable to assign
type VariableOrderingHeuristic int

const (
	// HeuristicDomDeg uses domain size / degree (constraints) - smallest first
	HeuristicDomDeg VariableOrderingHeuristic = iota
	// HeuristicDom uses domain size only - smallest first
	HeuristicDom
	// HeuristicDeg uses degree (constraints) only - largest first
	HeuristicDeg
	// HeuristicLex uses lexicographic order (variable ID)
	HeuristicLex
	// HeuristicRandom uses random ordering
	HeuristicRandom
	// HeuristicActivity uses constraint activity (not yet implemented)
	HeuristicActivity
)

// ValueOrderingHeuristic defines strategies for ordering values within a domain
type ValueOrderingHeuristic int

const (
	// ValueOrderAsc orders values ascending (1,2,3,...)
	ValueOrderAsc ValueOrderingHeuristic = iota
	// ValueOrderDesc orders values descending (...,3,2,1)
	ValueOrderDesc
	// ValueOrderRandom orders values randomly
	ValueOrderRandom
	// ValueOrderMid starts from middle value outward
	ValueOrderMid
)

// SolverConfig holds configuration for the FD solver
type SolverConfig struct {
	VariableHeuristic VariableOrderingHeuristic
	ValueHeuristic    ValueOrderingHeuristic
	RandomSeed        int64 // for reproducible random heuristics
}

// DefaultSolverConfig returns a default solver configuration
func DefaultSolverConfig() *SolverConfig {
	return &SolverConfig{
		VariableHeuristic: HeuristicDomDeg,
		ValueHeuristic:    ValueOrderAsc,
		RandomSeed:        42,
	}
}

// Generic BitSet-backed Domain for FD variables. Values are 1-based indices.
type BitSet struct {
	n     int
	words []uint64
}

func NewBitSet(n int) BitSet {
	w := (n + 63) / 64
	bs := BitSet{n: n, words: make([]uint64, w)}
	// set lower n bits
	for i := 0; i < n; i++ {
		idx := i / 64
		off := uint(i % 64)
		bs.words[idx] |= 1 << off
	}
	return bs
}

func (b BitSet) Clone() BitSet {
	words := make([]uint64, len(b.words))
	copy(words, b.words)
	return BitSet{n: b.n, words: words}
}

func (b BitSet) Has(v int) bool {
	if v < 1 || v > b.n {
		return false
	}
	i := (v - 1) / 64
	off := uint((v - 1) % 64)
	return ((b.words[i] >> off) & 1) == 1
}

func (b BitSet) RemoveValue(v int) BitSet {
	if v < 1 || v > b.n {
		return b.Clone()
	}
	nb := b.Clone()
	i := (v - 1) / 64
	off := uint((v - 1) % 64)
	nb.words[i] &^= (1 << off)
	return nb
}

func (b BitSet) Count() int {
	cnt := 0
	for _, w := range b.words {
		cnt += bits.OnesCount64(w)
	}
	return cnt
}

func (b BitSet) IsSingleton() bool { return b.Count() == 1 }

func (b BitSet) SingletonValue() int {
	for i, w := range b.words {
		if w == 0 {
			continue
		}
		off := bits.TrailingZeros64(w)
		return i*64 + off + 1
	}
	return -1
}

func (b BitSet) IterateValues(f func(v int)) {
	for i, w := range b.words {
		for w != 0 {
			t := w & -w
			off := bits.TrailingZeros64(w)
			f(i*64 + off + 1)
			w &^= t
		}
	}
}

// Intersect returns a new BitSet containing values present in both this and other BitSet
func (b BitSet) Intersect(other BitSet) BitSet {
	if b.n != other.n {
		// Return empty if domains have different sizes
		return BitSet{n: b.n, words: make([]uint64, len(b.words))}
	}
	result := BitSet{n: b.n, words: make([]uint64, len(b.words))}
	for i := range b.words {
		result.words[i] = b.words[i] & other.words[i]
	}
	return result
}

// Union returns a new BitSet containing values present in either this or other BitSet
func (b BitSet) Union(other BitSet) BitSet {
	if b.n != other.n {
		// If different sizes, union up to the smaller size
		minLen := len(b.words)
		if len(other.words) < minLen {
			minLen = len(other.words)
		}
		result := BitSet{n: b.n, words: make([]uint64, len(b.words))}
		for i := 0; i < minLen; i++ {
			result.words[i] = b.words[i] | other.words[i]
		}
		// Copy remaining words from the larger BitSet
		if len(b.words) > len(other.words) {
			copy(result.words[minLen:], b.words[minLen:])
		} else if len(other.words) > len(b.words) {
			copy(result.words[minLen:], other.words[minLen:])
		}
		return result
	}
	result := BitSet{n: b.n, words: make([]uint64, len(b.words))}
	for i := range b.words {
		result.words[i] = b.words[i] | other.words[i]
	}
	return result
}

// Complement returns a new BitSet containing all values NOT in this BitSet within the domain 1..n
func (b BitSet) Complement() BitSet {
	result := BitSet{n: b.n, words: make([]uint64, len(b.words))}
	// Start with full domain (bits 0 to n-1 set for values 1..n)
	for i := range result.words {
		result.words[i] = ^uint64(0)
	}
	// Mask out values beyond n
	if b.n%64 != 0 {
		// Keep only bits 0 to n-1
		lastWordMask := (uint64(1) << uint(b.n)) - 1
		result.words[len(result.words)-1] &= lastWordMask
	}
	// Remove the values that are in the original set
	for i := range b.words {
		result.words[i] &^= b.words[i]
	}
	return result
}

// FDVar is a finite-domain variable
type FDVar struct {
	ID     int
	domain BitSet
	peers  []*FDVar
}

// offsetLink connects two FDVars with an integer offset: other = self + offset
type offsetLink struct {
	other  *FDVar
	offset int
}

// Extend FDVar with offset links
// (placed here to avoid changing many other files)
// Note: we keep it unexported and simple; propagation logic in FDStore will consult these.
// We'll attach via a small map in FDStore to avoid changing serialized layout of FDVar across code paths.

// FDChange represents a single domain change for undo
type FDChange struct {
	vid    int
	domain BitSet
}

// FDStore manages finite-domain variables and constraints for constraint satisfaction problems.
// It provides efficient propagation and backtracking search with various heuristics.
//
// Key features:
// - BitSet-based domains for memory efficiency
// - AC-3 style propagation for constraints
// - Regin AllDifferent filtering for permutation constraints
// - Offset arithmetic constraints for modeling relationships
// - Iterative backtracking with pluggable strategies
// - Context-aware cancellation and timeouts
//
// Typical usage:
//
//	store := NewFDStoreWithDomain(maxValue)
//	vars := store.MakeFDVars(n)
//	// Add constraints...
//	solutions, err := store.SolveWithStrategy(ctx, strategy, limit)
type FDStore struct {
	mu         sync.Mutex
	vars       []*FDVar
	idToVar    map[int]*FDVar
	queue      []int      // variable ids to propagate
	trail      []FDChange // undo trail
	domainSize int
	// offsetLinks maps a variable id to offset links used for arithmetic propagation
	offsetLinks map[int][]offsetLink
	// ineqLinks maps a variable id to inequality links used for inequality propagation
	ineqLinks map[int][]ineqLink
	// arithmeticLinks maps a variable id to arithmetic links used for rich arithmetic propagation
	arithmeticLinks map[int][]ArithmeticLink
	// customConstraints holds user-defined constraints
	customConstraints []CustomConstraint
	// strategy holds solver strategy configuration (replaces old config)
	strategy *StrategyConfig
	// monitor tracks solving statistics (optional)
	monitor *SolverMonitor
}

// NewFDStore creates a store with default domain size 9 (1..9)
func NewFDStore() *FDStore { return NewFDStoreWithDomain(9) }

// NewFDStoreWithDomain creates a store with domain values 1..n
func NewFDStoreWithDomain(n int) *FDStore {
	return &FDStore{
		vars:       make([]*FDVar, 0, 128),
		idToVar:    make(map[int]*FDVar),
		queue:      make([]int, 0, 128),
		trail:      make([]FDChange, 0, 1024),
		domainSize: n,
		strategy:   DefaultStrategyConfig(),
	}
}

// NewFDStoreWithStrategy creates a store with custom strategy configuration
func NewFDStoreWithStrategy(n int, strategy *StrategyConfig) *FDStore {
	if strategy == nil {
		strategy = DefaultStrategyConfig()
	}
	return &FDStore{
		vars:       make([]*FDVar, 0, 128),
		idToVar:    make(map[int]*FDVar),
		queue:      make([]int, 0, 128),
		trail:      make([]FDChange, 0, 1024),
		domainSize: n,
		strategy:   strategy,
	}
}

// NewFDStoreWithConfig creates a store with custom solver configuration (backward compatibility)
func NewFDStoreWithConfig(n int, config *SolverConfig) *FDStore {
	if config == nil {
		config = DefaultSolverConfig()
	}
	// Convert old config to new strategy config
	strategy := &StrategyConfig{
		Labeling:   solverConfigToLabeling(config),
		Search:     NewDFSSearch(),
		RandomSeed: config.RandomSeed,
	}
	return NewFDStoreWithStrategy(n, strategy)
}

// solverConfigToLabeling converts old SolverConfig to new LabelingStrategy
func solverConfigToLabeling(config *SolverConfig) LabelingStrategy {
	switch config.VariableHeuristic {
	case HeuristicDomDeg:
		return NewFirstFailLabeling()
	case HeuristicDom:
		return NewDomainSizeLabeling()
	case HeuristicDeg:
		return NewDegreeLabeling()
	case HeuristicLex:
		return NewLexicographicLabeling()
	case HeuristicRandom:
		return NewRandomLabeling(config.RandomSeed)
	case HeuristicActivity:
		// Fall back to first-fail
		return NewFirstFailLabeling()
	default:
		return NewFirstFailLabeling()
	}
}

func (s *FDStore) NewVar() *FDVar {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := len(s.vars)
	v := &FDVar{ID: id, domain: NewBitSet(s.domainSize), peers: nil}
	s.vars = append(s.vars, v)
	s.idToVar[id] = v
	return v
}

// AddAllDifferent registers pairwise peers and enqueues initial propagation
func (s *FDStore) AddAllDifferent(vars []*FDVar) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := 0; i < len(vars); i++ {
		for j := 0; j < len(vars); j++ {
			if i == j {
				continue
			}
			vars[i].peers = append(vars[i].peers, vars[j])
		}
		s.enqueue(vars[i].ID)
	}
	if s.monitor != nil {
		s.monitor.RecordConstraint()
	}
}

func (s *FDStore) enqueue(vid int) {
	s.queue = append(s.queue, vid)
}

// snapshot returns current trail size for backtracking
func (s *FDStore) snapshot() int { return len(s.trail) }

// undo to snapshot
func (s *FDStore) undo(to int) {
	for i := len(s.trail) - 1; i >= to; i-- {
		ch := s.trail[i]
		if v, ok := s.idToVar[ch.vid]; ok {
			v.domain = ch.domain
		}
		s.trail = s.trail[:i]
	}
}

// assign domain to singleton value v, returns error on contradiction
func (s *FDStore) Assign(v *FDVar, value int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if value < 1 || value > s.domainSize {
		return ErrInvalidValue
	}
	newDom := NewBitSet(s.domainSize)
	// clear and set only value
	newDom = newDom.RemoveValue(0) // no-op to get a clone-like behavior
	// create zeroed then set bit
	newDom = BitSet{n: s.domainSize, words: make([]uint64, len(newDom.words))}
	idx := (value - 1) / 64
	off := uint((value - 1) % 64)
	newDom.words[idx] = 1 << off
	if v.domain.n == 0 {
		v.domain = newDom
		s.enqueue(v.ID)
		return s.propagateLocked()
	}
	// if v.domain equals newDom, still propagate
	equal := true
	if len(v.domain.words) != len(newDom.words) {
		equal = false
	}
	if equal {
		for i := range v.domain.words {
			if v.domain.words[i] != newDom.words[i] {
				equal = false
				break
			}
		}
	}
	if equal {
		return s.propagateLocked()
	}
	// check intersection
	intersect := false
	for i := range v.domain.words {
		if (v.domain.words[i] & newDom.words[i]) != 0 {
			intersect = true
			break
		}
	}
	if !intersect {
		return ErrInconsistent
	}
	s.trail = append(s.trail, FDChange{vid: v.ID, domain: v.domain.Clone()})
	v.domain = newDom
	s.enqueue(v.ID)
	return s.propagateLocked()
}

// Remove removes a value from a variable's domain
func (s *FDStore) Remove(v *FDVar, value int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !v.domain.Has(value) {
		return nil
	}
	s.trail = append(s.trail, FDChange{vid: v.ID, domain: v.domain.Clone()})
	v.domain = v.domain.RemoveValue(value)
	// check empty
	if v.domain.Count() == 0 {
		return ErrDomainEmpty
	}
	s.enqueue(v.ID)
	return s.propagateLocked()
}

// propagateLocked runs a simple AC-3 style propagation loop (requires lock)
func (s *FDStore) propagateLocked() error {
	for len(s.queue) > 0 {
		vid := s.queue[0]
		s.queue = s.queue[1:]
		v := s.idToVar[vid]
		if v == nil {
			continue
		}
		// if v is singleton, remove its value from peers
		if v.domain.IsSingleton() {
			val := v.domain.SingletonValue()
			for _, p := range v.peers {
				if p.domain.Has(val) {
					s.trail = append(s.trail, FDChange{vid: p.ID, domain: p.domain.Clone()})
					p.domain = p.domain.RemoveValue(val)
					if p.domain.Count() == 0 {
						return ErrDomainEmpty
					}
					s.enqueue(p.ID)
				}
			}
		} else {
			// collect singleton values among peers
			// currently unused, left for future pruning
			_ = 0
		}
		// propagate offset links (arithmetic constraints)
		if s.offsetLinks != nil {
			if links, ok := s.offsetLinks[vid]; ok {
				for _, l := range links {
					// compute image of v under offset
					img := imageOfDomain(v.domain, l.offset, s.domainSize)
					// intersect with other domain
					other := l.other
					if other == nil {
						continue
					}
					newDom := intersectBitSet(other.domain, img)
					if !bitSetEquals(newDom, other.domain) {
						s.trail = append(s.trail, FDChange{vid: other.ID, domain: other.domain.Clone()})
						other.domain = newDom
						if other.domain.Count() == 0 {
							return ErrDomainEmpty
						}
						s.enqueue(other.ID)
					}
				}
			}
		}
		// propagate inequality links
		if s.ineqLinks != nil {
			if links, ok := s.ineqLinks[vid]; ok {
				for _, l := range links {
					if err := s.propagateInequalityLocked(v, l.other, InequalityType(l.typ)); err != nil {
						return err
					}
				}
			}
		}
		// propagate arithmetic links
		if s.arithmeticLinks != nil {
			if links, ok := s.arithmeticLinks[vid]; ok {
				for _, l := range links {
					// Only propagate if this variable is involved in the constraint
					if l.op == ArithmeticPlus {
						if err := s.propagatePlusConstraint(l.x, l.y, l.z); err != nil {
							return err
						}
					}
					if l.op == ArithmeticMultiply {
						if err := s.propagateMultiplyConstraint(l.x, l.y, l.z); err != nil {
							return err
						}
					}
					if l.op == ArithmeticEquality {
						if err := s.propagateEqualityConstraint(l.x, l.y, l.z); err != nil {
							return err
						}
					}
					if l.op == ArithmeticMinus {
						if err := s.propagateMinusConstraint(l.x, l.y, l.z); err != nil {
							return err
						}
					}
					if l.op == ArithmeticQuotient {
						if err := s.propagateQuotientConstraint(l.x, l.y, l.z); err != nil {
							return err
						}
					}
					if l.op == ArithmeticModulo {
						if err := s.propagateModuloConstraint(l.x, l.y, l.z); err != nil {
							return err
						}
					}
					// Add other arithmetic operations here as they are implemented
				}
			}
		}
	}

	// After processing all queued variables, propagate custom constraints
	return s.propagateCustomConstraintsLocked()
}

// SetMonitor enables statistics collection for this store
func (s *FDStore) SetMonitor(monitor *SolverMonitor) {
	s.mu.Lock()
	s.monitor = monitor
	s.mu.Unlock()

	if s.monitor != nil {
		s.monitor.CaptureInitialDomains(s)
	}
}

// GetMonitor returns the current monitor, or nil if monitoring is disabled
func (s *FDStore) GetMonitor() *SolverMonitor {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.monitor
}

// GetStats returns current solving statistics, or nil if monitoring is disabled
func (s *FDStore) GetStats() *FDSolverStats {
	if s.monitor == nil {
		return nil
	}
	return s.monitor.GetStats()
}

// GetDomain returns a copy of the variable's current domain
func (s *FDStore) GetDomain(v *FDVar) BitSet {
	s.mu.Lock()
	defer s.mu.Unlock()
	return v.domain.Clone()
}

// Domain returns a copy of the variable's current domain (thread-safe)
func (v *FDVar) Domain() BitSet {
	return v.domain.Clone()
}

// IsSingleton returns true if the variable's domain contains exactly one value
func (v *FDVar) IsSingleton() bool {
	return v.domain.IsSingleton()
}

// SingletonValue returns the single value if the domain is singleton, panics otherwise
func (v *FDVar) SingletonValue() int {
	return v.domain.SingletonValue()
}

// FDVar is a finite-domain variable

// IntersectDomains intersects the domain of v with the given BitSet
func (s *FDStore) IntersectDomains(v *FDVar, other BitSet) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	newDom := v.domain.Intersect(other)
	if bitSetEquals(newDom, v.domain) {
		return nil // no change
	}

	s.trail = append(s.trail, FDChange{vid: v.ID, domain: v.domain.Clone()})
	v.domain = newDom
	if v.domain.Count() == 0 {
		return ErrDomainEmpty
	}
	s.enqueue(v.ID)
	return s.propagateLocked()
}

// UnionDomains unions the domain of v with the given BitSet
func (s *FDStore) UnionDomains(v *FDVar, other BitSet) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	newDom := v.domain.Union(other)
	if bitSetEquals(newDom, v.domain) {
		return nil // no change
	}

	s.trail = append(s.trail, FDChange{vid: v.ID, domain: v.domain.Clone()})
	v.domain = newDom
	s.enqueue(v.ID)
	return s.propagateLocked()
}

// ComplementDomain replaces the domain of v with its complement
func (s *FDStore) ComplementDomain(v *FDVar) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	newDom := v.domain.Complement()
	if bitSetEquals(newDom, v.domain) {
		return nil // no change
	}

	s.trail = append(s.trail, FDChange{vid: v.ID, domain: v.domain.Clone()})
	v.domain = newDom
	if v.domain.Count() == 0 {
		return ErrDomainEmpty
	}
	s.enqueue(v.ID)
	return s.propagateLocked()
}

// FDVar is a finite-domain variable

// Solve using iterative backtracking with pluggable strategies
func (s *FDStore) Solve(ctx context.Context, limit int) ([][]int, error) {
	return s.SolveWithStrategy(ctx, s.strategy, limit)
}

// SolveWithStrategy solves using the specified strategy configuration
func (s *FDStore) SolveWithStrategy(ctx context.Context, strategy *StrategyConfig, limit int) ([][]int, error) {
	if strategy == nil {
		strategy = DefaultStrategyConfig()
	}

	// Enable monitoring if requested
	if strategy.MonitorEnabled && s.monitor == nil {
		s.SetMonitor(NewSolverMonitor())
	}

	return strategy.Search.Search(ctx, s, strategy.Labeling, limit)
}

// GetStrategy returns the current strategy configuration
func (s *FDStore) GetStrategy() *StrategyConfig {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.strategy.Clone()
}

// SetStrategy updates the strategy configuration
func (s *FDStore) SetStrategy(strategy *StrategyConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if strategy == nil {
		strategy = DefaultStrategyConfig()
	}
	s.strategy = strategy
}

// SetLabelingStrategy updates only the labeling strategy
func (s *FDStore) SetLabelingStrategy(labeling LabelingStrategy) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.strategy == nil {
		s.strategy = DefaultStrategyConfig()
	}
	s.strategy.Labeling = labeling
}

// SetSearchStrategy updates only the search strategy
func (s *FDStore) SetSearchStrategy(search SearchStrategy) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.strategy == nil {
		s.strategy = DefaultStrategyConfig()
	}
	s.strategy.Search = search
}

// GetLabelingStrategy returns the current labeling strategy
func (s *FDStore) GetLabelingStrategy() LabelingStrategy {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.strategy == nil {
		return NewFirstFailLabeling()
	}
	return s.strategy.Labeling
}

// GetSearchStrategy returns the current search strategy
func (s *FDStore) GetSearchStrategy() SearchStrategy {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.strategy == nil {
		return NewDFSSearch()
	}
	return s.strategy.Search
}

// MakeFDVars creates n new FD variables with the store's default domain.
// The variables are initialized with full domains (1..domainSize).
// Returns a slice of *FDVar ready for constraint application.
func (s *FDStore) MakeFDVars(n int) []*FDVar {
	vars := make([]*FDVar, n)
	for i := 0; i < n; i++ {
		vars[i] = s.NewVar()
	}
	return vars
}

// AddOffsetLink adds an offset constraint: dst = src + offset
// This establishes a bidirectional relationship where changes to either variable
// propagate to restrict the other's domain. Useful for modeling arithmetic relationships
// like diagonals in N-Queens or temporal constraints.
func (s *FDStore) AddOffsetLink(src *FDVar, offset int, dst *FDVar) error {
	return s.AddOffsetConstraint(src, offset, dst)
}

// ApplyAllDifferentRegin applies the Regin AllDifferent constraint to the variables.
// This ensures all variables take distinct values, using efficient bipartite matching
// to prune domains beyond basic pairwise propagation. Essential for permutation problems
// like Sudoku rows/columns or N-Queens columns.
func (s *FDStore) ApplyAllDifferentRegin(vars []*FDVar) error {
	return s.AddAllDifferentRegin(vars)
}

// selectNextVariableAdvanced selects the next variable using the configured heuristic
func (s *FDStore) selectNextVariableAdvanced(config *SolverConfig) (int, []int) {
	switch config.VariableHeuristic {
	case HeuristicDomDeg:
		return s.selectNextVariableDomDeg()
	case HeuristicDom:
		return s.selectNextVariableDom()
	case HeuristicDeg:
		return s.selectNextVariableDeg()
	case HeuristicLex:
		return s.selectNextVariableLex()
	case HeuristicRandom:
		return s.selectNextVariableRandom(config.RandomSeed)
	case HeuristicActivity:
		// Fall back to dom/deg for now
		return s.selectNextVariableDomDeg()
	default:
		return s.selectNextVariableDomDeg()
	}
}

// selectNextVariableDomDeg implements the original dom/deg heuristic
func (s *FDStore) selectNextVariableDomDeg() (int, []int) {
	bestID := -1
	bestScore := -1.0
	var bestChoices []int
	for _, v := range s.vars {
		size := v.domain.Count()
		if size <= 1 {
			continue
		}
		degree := s.variableDegree(v)
		score := float64(size) / float64(1+degree) // dom/deg
		if bestID == -1 || score < bestScore {
			bestScore = score
			bestID = v.ID
		}
	}
	if bestID == -1 {
		return -1, nil
	}
	dom := s.idToVar[bestID].domain
	dom.IterateValues(func(val int) { bestChoices = append(bestChoices, val) })
	sort.Ints(bestChoices) // ascending order
	return bestID, bestChoices
}

// selectNextVariableDom selects variable with smallest domain
func (s *FDStore) selectNextVariableDom() (int, []int) {
	bestID := -1
	bestSize := -1
	var bestChoices []int
	for _, v := range s.vars {
		size := v.domain.Count()
		if size <= 1 {
			continue
		}
		if bestID == -1 || size < bestSize {
			bestSize = size
			bestID = v.ID
		}
	}
	if bestID == -1 {
		return -1, nil
	}
	dom := s.idToVar[bestID].domain
	dom.IterateValues(func(val int) { bestChoices = append(bestChoices, val) })
	sort.Ints(bestChoices)
	return bestID, bestChoices
}

// selectNextVariableDeg selects variable with highest degree (most constraints)
func (s *FDStore) selectNextVariableDeg() (int, []int) {
	bestID := -1
	bestDegree := -1
	var bestChoices []int
	for _, v := range s.vars {
		size := v.domain.Count()
		if size <= 1 {
			continue
		}
		degree := s.variableDegree(v)
		if bestID == -1 || degree > bestDegree {
			bestDegree = degree
			bestID = v.ID
		}
	}
	if bestID == -1 {
		return -1, nil
	}
	dom := s.idToVar[bestID].domain
	dom.IterateValues(func(val int) { bestChoices = append(bestChoices, val) })
	sort.Ints(bestChoices)
	return bestID, bestChoices
}

// selectNextVariableLex selects the first variable by ID
func (s *FDStore) selectNextVariableLex() (int, []int) {
	for _, v := range s.vars {
		size := v.domain.Count()
		if size <= 1 {
			continue
		}
		dom := v.domain
		var choices []int
		dom.IterateValues(func(val int) { choices = append(choices, val) })
		sort.Ints(choices)
		return v.ID, choices
	}
	return -1, nil
}

// selectNextVariableRandom selects a random unassigned variable
func (s *FDStore) selectNextVariableRandom(seed int64) (int, []int) {
	r := rand.New(rand.NewSource(seed))

	// Collect candidates
	var candidates []*FDVar
	for _, v := range s.vars {
		if v.domain.Count() > 1 {
			candidates = append(candidates, v)
		}
	}

	if len(candidates) == 0 {
		return -1, nil
	}

	// Select random variable
	selected := candidates[r.Intn(len(candidates))]
	dom := selected.domain
	var choices []int
	dom.IterateValues(func(val int) { choices = append(choices, val) })

	// Shuffle choices randomly
	for i := len(choices) - 1; i > 0; i-- {
		j := r.Intn(i + 1)
		choices[i], choices[j] = choices[j], choices[i]
	}

	return selected.ID, choices
}

// variableDegree returns the degree (number of constraints) for a variable
func (s *FDStore) variableDegree(v *FDVar) int {
	degree := len(v.peers)
	if s.offsetLinks != nil {
		if links, ok := s.offsetLinks[v.ID]; ok {
			degree += len(links)
		}
	}
	if s.ineqLinks != nil {
		if links, ok := s.ineqLinks[v.ID]; ok {
			degree += len(links)
		}
	}
	if s.arithmeticLinks != nil {
		if links, ok := s.arithmeticLinks[v.ID]; ok {
			degree += len(links)
		}
	}
	return degree
}

// FD errors
var (
	ErrInconsistent    = errors.New("constraint store is inconsistent")
	ErrInvalidValue    = errors.New("value out of domain")
	ErrDomainEmpty     = errors.New("domain became empty")
	ErrInvalidArgument = errors.New("invalid argument")
)

// orderValues orders the values according to the configured heuristic
func orderValues(choices []int, heuristic ValueOrderingHeuristic, seed int64) []int {
	result := make([]int, len(choices))
	copy(result, choices)

	switch heuristic {
	case ValueOrderAsc:
		sort.Ints(result)
	case ValueOrderDesc:
		sort.Sort(sort.Reverse(sort.IntSlice(result)))
	case ValueOrderRandom:
		r := rand.New(rand.NewSource(seed))
		for i := len(result) - 1; i > 0; i-- {
			j := r.Intn(i + 1)
			result[i], result[j] = result[j], result[i]
		}
	case ValueOrderMid:
		sort.Ints(result)
		// Reorder to start from middle and alternate outward
		mid := len(result) / 2
		ordered := make([]int, 0, len(result))
		left, right := mid, mid

		if len(result)%2 == 1 {
			ordered = append(ordered, result[mid])
			left--
			right++
		} else {
			right++
		}

		for left >= 0 || right < len(result) {
			if right < len(result) {
				ordered = append(ordered, result[right])
				right++
			}
			if left >= 0 {
				ordered = append(ordered, result[left])
				left--
			}
		}
		result = ordered
	}

	return result
}

// ParallelConstraintPropagator coordinates parallel constraint propagation
// across multiple workers while maintaining correctness and dependencies.
type ParallelConstraintPropagator struct {
	store           *FDStore
	workers         int
	taskChan        chan propagationTask
	resultChan      chan propagationResult
	doneChan        chan struct{}
	wg              sync.WaitGroup
	mu              sync.RWMutex
	activeTasks     int
	dependencyGraph map[int][]int // variable ID -> dependent variable IDs
}

// propagationTask represents a single propagation task
type propagationTask struct {
	variableID int
	taskType   propagationTaskType
	priority   int // for ordering dependent tasks
}

// propagationTaskType defines the type of propagation task
type propagationTaskType int

const (
	taskSingletonPropagation propagationTaskType = iota
	taskOffsetPropagation
	taskInequalityPropagation
	taskCustomPropagation
)

// propagationResult represents the result of a propagation task
type propagationResult struct {
	task     propagationTask
	err      error
	newTasks []propagationTask // additional tasks generated by this propagation
}

// NewParallelConstraintPropagator creates a new parallel constraint propagator.
func NewParallelConstraintPropagator(store *FDStore, workers int) *ParallelConstraintPropagator {
	if workers <= 0 {
		workers = runtime.NumCPU()
	}

	pcp := &ParallelConstraintPropagator{
		store:           store,
		workers:         workers,
		taskChan:        make(chan propagationTask, workers*10),
		resultChan:      make(chan propagationResult, workers*10),
		doneChan:        make(chan struct{}),
		dependencyGraph: make(map[int][]int),
	}

	// Build initial dependency graph
	pcp.buildDependencyGraph()

	// Start worker goroutines
	for i := 0; i < workers; i++ {
		pcp.wg.Add(1)
		go pcp.worker()
	}

	// Start coordinator
	go pcp.coordinator()

	return pcp
}

// buildDependencyGraph analyzes the constraint store to build a dependency graph
// showing which variables depend on others for propagation.
func (pcp *ParallelConstraintPropagator) buildDependencyGraph() {
	store := pcp.store

	// Add dependencies from peer relationships
	for _, v := range store.vars {
		for _, peer := range v.peers {
			pcp.dependencyGraph[v.ID] = append(pcp.dependencyGraph[v.ID], peer.ID)
		}
	}

	// Add dependencies from offset links
	if store.offsetLinks != nil {
		for srcID, links := range store.offsetLinks {
			for _, link := range links {
				if link.other != nil {
					pcp.dependencyGraph[srcID] = append(pcp.dependencyGraph[srcID], link.other.ID)
					pcp.dependencyGraph[link.other.ID] = append(pcp.dependencyGraph[link.other.ID], srcID)
				}
			}
		}
	}

	// Add dependencies from inequality links
	if store.ineqLinks != nil {
		for srcID, links := range store.ineqLinks {
			for _, link := range links {
				if link.other != nil {
					pcp.dependencyGraph[srcID] = append(pcp.dependencyGraph[srcID], link.other.ID)
					pcp.dependencyGraph[link.other.ID] = append(pcp.dependencyGraph[link.other.ID], srcID)
				}
			}
		}
	}
}

// SubmitTask submits a propagation task for parallel execution.
func (pcp *ParallelConstraintPropagator) SubmitTask(variableID int, taskType propagationTaskType) error {
	task := propagationTask{
		variableID: variableID,
		taskType:   taskType,
		priority:   pcp.calculatePriority(variableID),
	}

	select {
	case pcp.taskChan <- task:
		pcp.mu.Lock()
		pcp.activeTasks++
		pcp.mu.Unlock()
		return nil
	case <-pcp.doneChan:
		return fmt.Errorf("propagator is shutting down")
	}
}

// calculatePriority calculates task priority based on dependency depth
// Higher priority (lower number) for variables with more dependencies.
func (pcp *ParallelConstraintPropagator) calculatePriority(variableID int) int {
	// Simple priority based on number of dependents
	// More dependents = higher priority (lower number)
	return len(pcp.dependencyGraph[variableID])
}

// WaitForCompletion waits for all propagation tasks to complete.
func (pcp *ParallelConstraintPropagator) WaitForCompletion(ctx context.Context) error {
	done := make(chan error, 1)

	go func() {
		pcp.mu.Lock()
		for pcp.activeTasks > 0 {
			pcp.mu.Unlock()
			select {
			case result := <-pcp.resultChan:
				pcp.mu.Lock()
				pcp.activeTasks--

				if result.err != nil {
					done <- result.err
					pcp.mu.Unlock()
					return
				}

				// Submit any new tasks generated by this result
				for _, newTask := range result.newTasks {
					select {
					case pcp.taskChan <- newTask:
						pcp.activeTasks++
					case <-pcp.doneChan:
						done <- fmt.Errorf("propagator shutdown during task submission")
						pcp.mu.Unlock()
						return
					}
				}
				pcp.mu.Unlock()
			case <-ctx.Done():
				done <- ctx.Err()
				return
			}
			pcp.mu.Lock()
		}
		pcp.mu.Unlock()
		done <- nil
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Shutdown gracefully shuts down the parallel propagator.
func (pcp *ParallelConstraintPropagator) Shutdown() {
	close(pcp.doneChan)
	close(pcp.taskChan)
	pcp.wg.Wait()
}

// worker processes propagation tasks.
func (pcp *ParallelConstraintPropagator) worker() {
	defer pcp.wg.Done()

	for {
		select {
		case task, ok := <-pcp.taskChan:
			if !ok {
				return // shutdown
			}

			result := pcp.processTask(task)
			select {
			case pcp.resultChan <- result:
			case <-pcp.doneChan:
				return
			}

		case <-pcp.doneChan:
			return
		}
	}
}

// processTask executes a single propagation task.
func (pcp *ParallelConstraintPropagator) processTask(task propagationTask) propagationResult {
	result := propagationResult{task: task}

	// Acquire store lock for this task
	pcp.store.mu.Lock()
	defer pcp.store.mu.Unlock()

	v := pcp.store.idToVar[task.variableID]
	if v == nil {
		return result // variable not found, not an error
	}

	switch task.taskType {
	case taskSingletonPropagation:
		result.newTasks = pcp.propagateSingleton(v)
	case taskOffsetPropagation:
		result.newTasks = pcp.propagateOffsets(v)
	case taskInequalityPropagation:
		result.newTasks = pcp.propagateInequalities(v)
	case taskCustomPropagation:
		err := pcp.store.propagateCustomConstraintsLocked()
		result.err = err
	}

	return result
}

// propagateSingleton handles singleton propagation for a variable.
func (pcp *ParallelConstraintPropagator) propagateSingleton(v *FDVar) []propagationTask {
	var newTasks []propagationTask

	// if v is singleton, remove its value from peers
	if v.domain.IsSingleton() {
		val := v.domain.SingletonValue()
		for _, p := range v.peers {
			if p.domain.Has(val) {
				pcp.store.trail = append(pcp.store.trail, FDChange{vid: p.ID, domain: p.domain.Clone()})
				p.domain = p.domain.RemoveValue(val)
				if p.domain.Count() == 0 {
					// Domain became empty - this will be handled by the caller
					continue
				}
				pcp.store.enqueue(p.ID)
				newTasks = append(newTasks, propagationTask{
					variableID: p.ID,
					taskType:   taskSingletonPropagation,
					priority:   pcp.calculatePriority(p.ID),
				})
			}
		}
	}

	return newTasks
}

// propagateOffsets handles offset constraint propagation.
func (pcp *ParallelConstraintPropagator) propagateOffsets(v *FDVar) []propagationTask {
	var newTasks []propagationTask

	if pcp.store.offsetLinks == nil {
		return newTasks
	}

	if links, ok := pcp.store.offsetLinks[v.ID]; ok {
		for _, l := range links {
			img := imageOfDomain(v.domain, l.offset, pcp.store.domainSize)
			other := l.other
			if other == nil {
				continue
			}

			newDom := intersectBitSet(other.domain, img)
			if !bitSetEquals(newDom, other.domain) {
				pcp.store.trail = append(pcp.store.trail, FDChange{vid: other.ID, domain: other.domain.Clone()})
				other.domain = newDom
				if other.domain.Count() == 0 {
					continue
				}
				pcp.store.enqueue(other.ID)
				newTasks = append(newTasks, propagationTask{
					variableID: other.ID,
					taskType:   taskOffsetPropagation,
					priority:   pcp.calculatePriority(other.ID),
				})
			}
		}
	}

	return newTasks
}

// propagateInequalities handles inequality constraint propagation.
func (pcp *ParallelConstraintPropagator) propagateInequalities(v *FDVar) []propagationTask {
	var newTasks []propagationTask

	if pcp.store.ineqLinks == nil {
		return newTasks
	}

	if links, ok := pcp.store.ineqLinks[v.ID]; ok {
		for _, l := range links {
			if err := pcp.store.propagateInequalityLocked(v, l.other, InequalityType(l.typ)); err != nil {
				// Error will be handled by the caller
				continue
			}
			// If propagation succeeded and changed domains, enqueue the other variable
			if l.other != nil {
				newTasks = append(newTasks, propagationTask{
					variableID: l.other.ID,
					taskType:   taskInequalityPropagation,
					priority:   pcp.calculatePriority(l.other.ID),
				})
			}
		}
	}

	return newTasks
}

// coordinator manages the overall propagation process and handles dependencies.
func (pcp *ParallelConstraintPropagator) coordinator() {
	// This could be enhanced to handle dependency ordering more sophisticatedly
	// For now, we rely on the priority system and let workers handle conflicts
	// through the store's mutex
}
