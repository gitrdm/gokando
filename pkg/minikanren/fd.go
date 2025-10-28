package minikanren

import (
	"fmt"
	"math/bits"
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

// FDStore manages variables, constraints and propagation
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

// assign domain to singleton value v, returns false on contradiction
func (s *FDStore) Assign(v *FDVar, value int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if value < 1 || value > s.domainSize {
		return false
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
		return false
	}
	s.trail = append(s.trail, FDChange{vid: v.ID, domain: v.domain.Clone()})
	v.domain = newDom
	s.enqueue(v.ID)
	return s.propagateLocked()
}

// Remove removes a value from a variable's domain
func (s *FDStore) Remove(v *FDVar, value int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !v.domain.Has(value) {
		return true
	}
	s.trail = append(s.trail, FDChange{vid: v.ID, domain: v.domain.Clone()})
	v.domain = v.domain.RemoveValue(value)
	// check empty
	if v.domain.Count() == 0 {
		return false
	}
	s.enqueue(v.ID)
	return s.propagateLocked()
}

// propagateLocked runs a simple AC-3 style propagation loop (requires lock)
func (s *FDStore) propagateLocked() bool {
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
						return false
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
							return false
						}
						s.enqueue(other.ID)
					}
				}
			}
		}
	}
	return true
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

// Solve using recursive backtracking with MRV heuristic
func (s *FDStore) Solve(limit int) ([][]int, error) {
	s.mu.Lock()
	// quick propagation initial
	ok := s.propagateLocked()
	s.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("inconsistent")
	}
	solutions := make([][]int, 0)
	var dfs func() bool
	dfs = func() bool {
		s.mu.Lock()
		// check if complete
		allAssigned := true
		for _, v := range s.vars {
			if !v.domain.IsSingleton() {
				allAssigned = false
				break
			}
		}
		if allAssigned {
			sol := make([]int, len(s.vars))
			for i, v := range s.vars {
				sol[i] = v.domain.SingletonValue()
			}
			solutions = append(solutions, sol)
			s.mu.Unlock()
			if limit > 0 && len(solutions) >= limit {
				return true
			}
			return false
		}

		// MRV: pick variable with smallest domain >1
		best := -1
		bestSize := 1 << 30
		for _, v := range s.vars {
			c := v.domain.Count()
			if c > 1 && c < bestSize {
				bestSize = c
				best = v.ID
			}
		}
		if best == -1 {
			s.mu.Unlock()
			return false
		}
		var choices []int
		dom := s.idToVar[best].domain
		dom.IterateValues(func(val int) { choices = append(choices, val) })
		s.mu.Unlock()

		// try each choice with snapshot/undo
		for _, val := range choices {
			snap := s.snapshot()
			if !s.Assign(s.idToVar[best], val) {
				s.undo(snap)
				continue
			}
			stop := dfs()
			s.undo(snap)
			if stop {
				return true
			}
		}
		return false
	}
	dfs()
	return solutions, nil
}
