package minikanren

import (
	"context"
	"errors"
	"math/bits"
	"sort"
	"sync"
)

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
// - Iterative backtracking with dom/deg heuristics
// - Context-aware cancellation and timeouts
//
// Typical usage:
//
//	store := NewFDStoreWithDomain(maxValue)
//	vars := store.MakeFDVars(n)
//	// Add constraints...
//	solutions, err := store.Solve(ctx, limit)
type FDStore struct {
	mu         sync.Mutex
	vars       []*FDVar
	idToVar    map[int]*FDVar
	queue      []int      // variable ids to propagate
	trail      []FDChange // undo trail
	domainSize int
	// offsetLinks maps a variable id to offset links used for arithmetic propagation
	offsetLinks map[int][]offsetLink
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
	}
	return nil
}

// DomainSnapshot returns a copy of domains for debugging/inspection
func (s *FDStore) DomainSnapshot() []BitSet {
	s.mu.Lock()
	defer s.mu.Unlock()
	snap := make([]BitSet, len(s.vars))
	for i, v := range s.vars {
		snap[i] = v.domain.Clone()
	}
	return snap
}

// Solve using iterative backtracking with MRV heuristic
func (s *FDStore) Solve(ctx context.Context, limit int) ([][]int, error) {
	s.mu.Lock()
	// quick propagation initial
	if err := s.propagateLocked(); err != nil {
		s.mu.Unlock()
		return nil, err
	}
	s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	solutions := make([][]int, 0)

	// Iterative backtracking using a stack
	type frame struct {
		snap    int // trail snapshot
		varID   int // variable being tried
		valIdx  int // index in choices
		choices []int
	}
	stack := []frame{}

	// Initial check
	s.mu.Lock()
	allAssigned := true
	for _, v := range s.vars {
		if !v.domain.IsSingleton() {
			allAssigned = false
			break
		}
	}
	s.mu.Unlock()
	if allAssigned {
		s.mu.Lock()
		sol := make([]int, len(s.vars))
		for i, v := range s.vars {
			sol[i] = v.domain.SingletonValue()
		}
		solutions = append(solutions, sol)
		s.mu.Unlock()
		return solutions, nil
	}

	// Push initial frame
	s.mu.Lock()
	best, choices := s.selectNextVariable()
	s.mu.Unlock()
	if best == -1 {
		return solutions, nil
	}
	stack = append(stack, frame{snap: s.snapshot(), varID: best, valIdx: 0, choices: choices})

	for len(stack) > 0 {
		// Check cancellation
		select {
		case <-ctx.Done():
			return solutions, ctx.Err()
		default:
		}

		f := &stack[len(stack)-1]

		if f.valIdx >= len(f.choices) {
			// Backtrack
			s.undo(f.snap)
			stack = stack[:len(stack)-1]
			continue
		}

		val := f.choices[f.valIdx]
		f.valIdx++

		// Try assignment
		if err := s.Assign(s.idToVar[f.varID], val); err != nil {
			continue
		}

		// Check if complete
		s.mu.Lock()
		allAssigned := true
		for _, v := range s.vars {
			if !v.domain.IsSingleton() {
				allAssigned = false
				break
			}
		}
		s.mu.Unlock()
		if allAssigned {
			s.mu.Lock()
			sol := make([]int, len(s.vars))
			for i, v := range s.vars {
				sol[i] = v.domain.SingletonValue()
			}
			solutions = append(solutions, sol)
			s.mu.Unlock()
			s.undo(f.snap)
			if limit > 0 && len(solutions) >= limit {
				return solutions, nil
			}
			continue
		}

		// Find next variable
		s.mu.Lock()
		nextBest, nextChoices := s.selectNextVariable()
		s.mu.Unlock()
		if nextBest == -1 {
			s.undo(f.snap)
			continue
		}

		// Push new frame
		stack = append(stack, frame{snap: s.snapshot(), varID: nextBest, valIdx: 0, choices: nextChoices})
	}

	return solutions, nil
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

// variableDegree returns the degree (number of constraints) for a variable
func (s *FDStore) variableDegree(v *FDVar) int {
	degree := len(v.peers)
	if s.offsetLinks != nil {
		if links, ok := s.offsetLinks[v.ID]; ok {
			degree += len(links)
		}
	}
	return degree
}

// selectNextVariable selects the next variable to assign using dom/deg heuristic
func (s *FDStore) selectNextVariable() (int, []int) {
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
	// Sort choices ascending for value ordering heuristic
	sort.Ints(bestChoices)
	return bestID, bestChoices
}

// FD errors
var (
	ErrInconsistent    = errors.New("constraint store is inconsistent")
	ErrInvalidValue    = errors.New("value out of domain")
	ErrDomainEmpty     = errors.New("domain became empty")
	ErrInvalidArgument = errors.New("invalid argument")
)
