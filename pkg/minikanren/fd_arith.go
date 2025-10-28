package minikanren

// fd_arith.go: simple arithmetic link support for FDStore

// AddOffsetConstraint enforces dst = src + offset (integer constant). Domains are 1..domainSize.
// It installs bidirectional propagation so changes to either variable restrict the other.
func (s *FDStore) AddOffsetConstraint(src *FDVar, offset int, dst *FDVar) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if src == nil || dst == nil {
		return ErrInvalidArgument
	}

	// store links in a map keyed by var id
	if s.offsetLinks == nil {
		s.offsetLinks = make(map[int][]offsetLink)
	}

	// add forward and backward links
	s.offsetLinks[src.ID] = append(s.offsetLinks[src.ID], offsetLink{other: dst, offset: offset})
	s.offsetLinks[dst.ID] = append(s.offsetLinks[dst.ID], offsetLink{other: src, offset: -offset})

	// immediately propagate one-step: intersect dst with image(src) and src with preimage(dst)
	img := imageOfDomain(src.domain, offset, s.domainSize)
	newDst := intersectBitSet(dst.domain, img)
	if !bitSetEquals(newDst, dst.domain) {
		s.trail = append(s.trail, FDChange{vid: dst.ID, domain: dst.domain.Clone()})
		dst.domain = newDst
		if dst.domain.Count() == 0 {
			return ErrDomainEmpty
		}
	}

	pre := imageOfDomain(dst.domain, -offset, s.domainSize)
	newSrc := intersectBitSet(src.domain, pre)
	if !bitSetEquals(newSrc, src.domain) {
		s.trail = append(s.trail, FDChange{vid: src.ID, domain: src.domain.Clone()})
		src.domain = newSrc
		if src.domain.Count() == 0 {
			return ErrDomainEmpty
		}
	}

	// Enqueue both for further propagation
	s.enqueue(src.ID)
	s.enqueue(dst.ID)
	return s.propagateLocked()
}

// imageOfDomain returns a BitSet representing {v+offset | v in dom} intersected with 1..n
func imageOfDomain(dom BitSet, offset int, n int) BitSet {
	res := NewBitSet(n)
	// zero it
	res = BitSet{n: n, words: make([]uint64, len(dom.words))}
	dom.IterateValues(func(v int) {
		nv := v + offset
		if nv >= 1 && nv <= n {
			idx := (nv - 1) / 64
			off := uint((nv - 1) % 64)
			res.words[idx] |= 1 << off
		}
	})
	return res
}

func intersectBitSet(a, b BitSet) BitSet {
	// assume same length
	nb := BitSet{n: a.n, words: make([]uint64, len(a.words))}
	for i := range a.words {
		nb.words[i] = a.words[i] & b.words[i]
	}
	return nb
}

func bitSetEquals(a, b BitSet) bool {
	if a.n != b.n || len(a.words) != len(b.words) {
		return false
	}
	for i := range a.words {
		if a.words[i] != b.words[i] {
			return false
		}
	}
	return true
}
