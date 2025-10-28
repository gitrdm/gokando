package minikanren

import "sort"

// Regin-style AllDifferent filtering (simplified but correct):
// For each variable/value, remove values that cannot participate in any complete matching.

func (s *FDStore) setDomainLocked(v *FDVar, newDom BitSet) {
	s.trail = append(s.trail, FDChange{vid: v.ID, domain: v.domain.Clone()})
	v.domain = newDom
	s.enqueue(v.ID)
}

// maxMatching computes a maximum bipartite matching between variables and values.
// It returns a map value->varID (1-based value indices) and the matched count.
// Uses an augmenting-path DFS with a token-based visited array and a small
// heuristic ordering (try smaller domains first).
func maxMatching(vars []*FDVar, domainSize int) (map[int]int, int) {
	varN := len(vars)
	matchVal := make([]int, domainSize+1) // value->var index or -1
	for i := range matchVal {
		matchVal[i] = -1
	}
	matchVar := make([]int, varN) // var index -> value or -1
	for i := range matchVar {
		matchVar[i] = -1
	}

	// ordering heuristic: try variables with smaller domains first
	order := make([]int, varN)
	for i := 0; i < varN; i++ {
		order[i] = i
	}
	sort.Slice(order, func(i, j int) bool {
		return vars[order[i]].domain.Count() < vars[order[j]].domain.Count()
	})

	// visited token array to avoid repeated allocations/clears
	seenToken := make([]int, domainSize+1)
	token := 1

	var tryAugment func(vi int, tok int) bool
	tryAugment = func(vi int, tok int) bool {
		found := false
		vars[vi].domain.IterateValues(func(val int) {
			if found {
				return
			}
			if val < 1 || val > domainSize {
				return
			}
			if seenToken[val] == tok {
				return
			}
			seenToken[val] = tok
			if matchVal[val] == -1 {
				matchVal[val] = vi
				matchVar[vi] = val
				found = true
				return
			}
			// try to reassign existing matched variable
			if tryAugment(matchVal[val], tok) {
				matchVal[val] = vi
				matchVar[vi] = val
				found = true
				return
			}
		})
		return found
	}

	matched := 0
	// First, assign singletons deterministically
	for _, vi := range order {
		if vars[vi].domain.IsSingleton() {
			val := vars[vi].domain.SingletonValue()
			if val >= 1 && val <= domainSize && matchVal[val] == -1 {
				matchVal[val] = vi
				matchVar[vi] = val
				matched++
			} else {
				// conflict or out of range
				return map[int]int{}, matched
			}
		}
	}

	// Try to augment for remaining variables in order
	for _, vi := range order {
		if matchVar[vi] != -1 {
			continue
		}
		token++
		if tryAugment(vi, token) {
			matched++
		}
	}

	// build map
	res := make(map[int]int)
	for val := 1; val <= domainSize; val++ {
		if matchVal[val] != -1 {
			res[val] = matchVal[val]
		} else {
			res[val] = -1
		}
	}
	return res, matched
}

func (s *FDStore) ReginFilterLocked(vars []*FDVar) error {
	n := len(vars)
	if n == 0 {
		return nil
	}

	// initial matching
	matchMap, matched := maxMatching(vars, s.domainSize)
	if matched < n {
		return ErrInconsistent
	}

	// For each variable, test each value in its domain for support
	for vi, v := range vars {
		// collect values to possibly remove
		toRemove := make([]int, 0)
		v.domain.IterateValues(func(val int) {
			// if current matching maps this val to vi, it's supported
			if matchMap[val] == vi {
				return
			}

			// try forcing v=val and check for full matching
			snap := s.snapshot()
			// create singleton domain for v
			newDom := BitSet{n: s.domainSize, words: make([]uint64, len(v.domain.words))}
			idx := (val - 1) / 64
			off := uint((val - 1) % 64)
			newDom.words[idx] = 1 << off
			s.setDomainLocked(v, newDom)

			// build remaining vars slice (they reference same var objects; singletons included)
			_, m := maxMatching(vars, s.domainSize)
			s.undo(snap)
			if m < n {
				toRemove = append(toRemove, val)
			}
		})

		if len(toRemove) > 0 {
			// apply removals
			for _, val := range toRemove {
				// already locked
				s.trail = append(s.trail, FDChange{vid: v.ID, domain: v.domain.Clone()})
				v.domain = v.domain.RemoveValue(val)
				if v.domain.Count() == 0 {
					return ErrDomainEmpty
				}
				s.enqueue(v.ID)
			}
		}
	}
	return nil
}

// AddAllDifferentRegin registers an AllDifferent constraint and applies Regin filtering.
func (s *FDStore) AddAllDifferentRegin(vars []*FDVar) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	// register peers for basic singleton propagation
	for i := 0; i < len(vars); i++ {
		for j := 0; j < len(vars); j++ {
			if i == j {
				continue
			}
			vars[i].peers = append(vars[i].peers, vars[j])
		}
	}
	// run Regin filter
	if err := s.ReginFilterLocked(vars); err != nil {
		return err
	}
	// enqueue all vars for further propagation
	for _, v := range vars {
		s.enqueue(v.ID)
	}
	// run regular propagation to propagate singletons
	return s.propagateLocked()
}
