package minikanren

// fd_arith.go: rich arithmetic constraint support for FDStore

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
	if s.monitor != nil {
		s.monitor.RecordConstraint()
	}
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

// Arithmetic constraint types for variable relationships
type ArithmeticConstraintType int

const (
	ArithmeticPlus ArithmeticConstraintType = iota
	ArithmeticMultiply
	ArithmeticEquality
	ArithmeticMinus
	ArithmeticQuotient
	ArithmeticModulo
)

// ArithmeticLink represents a relationship between three variables: x op y = z
type ArithmeticLink struct {
	x, y, z *FDVar
	op      ArithmeticConstraintType
}

// AddPlusConstraint enforces x + y = z with bidirectional propagation.
// All variables must be from the same FDStore.
func (s *FDStore) AddPlusConstraint(x, y, z *FDVar) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if x == nil || y == nil || z == nil {
		return ErrInvalidArgument
	}

	// Initialize arithmetic links map if needed
	if s.arithmeticLinks == nil {
		s.arithmeticLinks = make(map[int][]ArithmeticLink)
	}

	// Add links for bidirectional propagation
	link := ArithmeticLink{x: x, y: y, z: z, op: ArithmeticPlus}
	s.arithmeticLinks[x.ID] = append(s.arithmeticLinks[x.ID], link)
	s.arithmeticLinks[y.ID] = append(s.arithmeticLinks[y.ID], link)
	s.arithmeticLinks[z.ID] = append(s.arithmeticLinks[z.ID], link)

	// Initial propagation
	if err := s.propagatePlusConstraint(x, y, z); err != nil {
		return err
	}

	// Enqueue all variables for further propagation
	s.enqueue(x.ID)
	s.enqueue(y.ID)
	s.enqueue(z.ID)

	if s.monitor != nil {
		s.monitor.RecordConstraint()
	}
	return s.propagateLocked()
}

// propagatePlusConstraint performs the core propagation logic for x + y = z
func (s *FDStore) propagatePlusConstraint(x, y, z *FDVar) error {
	// For each pair of known domains, compute the possible values for the third

	// If x and y are known, restrict z to x+y
	if x.domain.IsSingleton() && y.domain.IsSingleton() {
		sum := x.domain.SingletonValue() + y.domain.SingletonValue()
		if sum >= 1 && sum <= s.domainSize {
			newZ := BitSet{n: s.domainSize, words: make([]uint64, (s.domainSize+63)/64)}
			idx := (sum - 1) / 64
			off := uint((sum - 1) % 64)
			newZ.words[idx] |= 1 << off

			intersected := z.domain.Intersect(newZ)
			if !bitSetEquals(intersected, z.domain) {
				s.trail = append(s.trail, FDChange{vid: z.ID, domain: z.domain.Clone()})
				z.domain = intersected
				if z.domain.Count() == 0 {
					return ErrDomainEmpty
				}
			}
		} else {
			// Sum is out of bounds, domain becomes empty
			return ErrDomainEmpty
		}
	}

	// If x and z are known, restrict y to z-x
	if x.domain.IsSingleton() && z.domain.IsSingleton() {
		diff := z.domain.SingletonValue() - x.domain.SingletonValue()
		if diff >= 1 && diff <= s.domainSize {
			newY := BitSet{n: s.domainSize, words: make([]uint64, (s.domainSize+63)/64)}
			idx := (diff - 1) / 64
			off := uint((diff - 1) % 64)
			newY.words[idx] |= 1 << off

			intersected := y.domain.Intersect(newY)
			if !bitSetEquals(intersected, y.domain) {
				s.trail = append(s.trail, FDChange{vid: y.ID, domain: y.domain.Clone()})
				y.domain = intersected
				if y.domain.Count() == 0 {
					return ErrDomainEmpty
				}
			}
		} else {
			return ErrDomainEmpty
		}
	}

	// If y and z are known, restrict x to z-y
	if y.domain.IsSingleton() && z.domain.IsSingleton() {
		diff := z.domain.SingletonValue() - y.domain.SingletonValue()
		if diff >= 1 && diff <= s.domainSize {
			newX := BitSet{n: s.domainSize, words: make([]uint64, (s.domainSize+63)/64)}
			idx := (diff - 1) / 64
			off := uint((diff - 1) % 64)
			newX.words[idx] |= 1 << off

			intersected := x.domain.Intersect(newX)
			if !bitSetEquals(intersected, x.domain) {
				s.trail = append(s.trail, FDChange{vid: x.ID, domain: x.domain.Clone()})
				x.domain = intersected
				if x.domain.Count() == 0 {
					return ErrDomainEmpty
				}
			}
		} else {
			return ErrDomainEmpty
		}
	}

	// General case: restrict domains based on possible combinations
	return s.propagatePlusGeneral(x, y, z)
}

// propagatePlusGeneral handles the general case where not all variables are singleton
func (s *FDStore) propagatePlusGeneral(x, y, z *FDVar) error {
	// Compute possible values for z given x and y domains
	possibleZ := BitSet{n: s.domainSize, words: make([]uint64, (s.domainSize+63)/64)}
	x.domain.IterateValues(func(xv int) {
		y.domain.IterateValues(func(yv int) {
			sum := xv + yv
			if sum >= 1 && sum <= s.domainSize {
				idx := (sum - 1) / 64
				off := uint((sum - 1) % 64)
				possibleZ.words[idx] |= 1 << off
			}
		})
	})

	intersectedZ := z.domain.Intersect(possibleZ)
	if !bitSetEquals(intersectedZ, z.domain) {
		s.trail = append(s.trail, FDChange{vid: z.ID, domain: z.domain.Clone()})
		z.domain = intersectedZ
		if z.domain.Count() == 0 {
			return ErrDomainEmpty
		}
		s.enqueue(z.ID)
	}

	// Compute possible values for x given y and z domains
	possibleX := BitSet{n: s.domainSize, words: make([]uint64, (s.domainSize+63)/64)}
	y.domain.IterateValues(func(yv int) {
		z.domain.IterateValues(func(zv int) {
			diff := zv - yv
			if diff >= 1 && diff <= s.domainSize {
				idx := (diff - 1) / 64
				off := uint((diff - 1) % 64)
				possibleX.words[idx] |= 1 << off
			}
		})
	})

	intersectedX := x.domain.Intersect(possibleX)
	if !bitSetEquals(intersectedX, x.domain) {
		s.trail = append(s.trail, FDChange{vid: x.ID, domain: x.domain.Clone()})
		x.domain = intersectedX
		if x.domain.Count() == 0 {
			return ErrDomainEmpty
		}
		s.enqueue(x.ID)
	}

	// Compute possible values for y given x and z domains
	possibleY := BitSet{n: s.domainSize, words: make([]uint64, (s.domainSize+63)/64)}
	x.domain.IterateValues(func(xv int) {
		z.domain.IterateValues(func(zv int) {
			diff := zv - xv
			if diff >= 1 && diff <= s.domainSize {
				idx := (diff - 1) / 64
				off := uint((diff - 1) % 64)
				possibleY.words[idx] |= 1 << off
			}
		})
	})

	intersectedY := y.domain.Intersect(possibleY)
	if !bitSetEquals(intersectedY, y.domain) {
		s.trail = append(s.trail, FDChange{vid: y.ID, domain: y.domain.Clone()})
		y.domain = intersectedY
		if y.domain.Count() == 0 {
			return ErrDomainEmpty
		}
		s.enqueue(y.ID)
	}

	return nil
}

// AddMultiplyConstraint enforces x * y = z with bidirectional propagation.
// All variables must be from the same FDStore.
func (s *FDStore) AddMultiplyConstraint(x, y, z *FDVar) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if x == nil || y == nil || z == nil {
		return ErrInvalidArgument
	}

	// Initialize arithmetic links map if needed
	if s.arithmeticLinks == nil {
		s.arithmeticLinks = make(map[int][]ArithmeticLink)
	}

	// Add links for bidirectional propagation
	link := ArithmeticLink{x: x, y: y, z: z, op: ArithmeticMultiply}
	s.arithmeticLinks[x.ID] = append(s.arithmeticLinks[x.ID], link)
	s.arithmeticLinks[y.ID] = append(s.arithmeticLinks[y.ID], link)
	s.arithmeticLinks[z.ID] = append(s.arithmeticLinks[z.ID], link)

	// Initial propagation
	if err := s.propagateMultiplyConstraint(x, y, z); err != nil {
		return err
	}

	// Enqueue all variables for further propagation
	s.enqueue(x.ID)
	s.enqueue(y.ID)
	s.enqueue(z.ID)

	if s.monitor != nil {
		s.monitor.RecordConstraint()
	}
	return s.propagateLocked()
}

// propagateMultiplyConstraint performs the core propagation logic for x * y = z
func (s *FDStore) propagateMultiplyConstraint(x, y, z *FDVar) error {
	// For each pair of known domains, compute the possible values for the third

	// If x and y are known, restrict z to x*y
	if x.domain.IsSingleton() && y.domain.IsSingleton() {
		product := x.domain.SingletonValue() * y.domain.SingletonValue()
		if product >= 1 && product <= s.domainSize {
			newZ := BitSet{n: s.domainSize, words: make([]uint64, (s.domainSize+63)/64)}
			idx := (product - 1) / 64
			off := uint((product - 1) % 64)
			newZ.words[idx] |= 1 << off

			intersected := z.domain.Intersect(newZ)
			if !bitSetEquals(intersected, z.domain) {
				s.trail = append(s.trail, FDChange{vid: z.ID, domain: z.domain.Clone()})
				z.domain = intersected
				if z.domain.Count() == 0 {
					return ErrDomainEmpty
				}
			}
		} else {
			// Product is out of bounds, domain becomes empty
			return ErrDomainEmpty
		}
	}

	// If x and z are known, restrict y to z/x (integer division)
	if x.domain.IsSingleton() && z.domain.IsSingleton() {
		xv := x.domain.SingletonValue()
		zv := z.domain.SingletonValue()
		if xv != 0 && zv%xv == 0 {
			quotient := zv / xv
			if quotient >= 1 && quotient <= s.domainSize {
				newY := BitSet{n: s.domainSize, words: make([]uint64, (s.domainSize+63)/64)}
				idx := (quotient - 1) / 64
				off := uint((quotient - 1) % 64)
				newY.words[idx] |= 1 << off

				intersected := y.domain.Intersect(newY)
				if !bitSetEquals(intersected, y.domain) {
					s.trail = append(s.trail, FDChange{vid: y.ID, domain: y.domain.Clone()})
					y.domain = intersected
					if y.domain.Count() == 0 {
						return ErrDomainEmpty
					}
				}
			} else {
				return ErrDomainEmpty
			}
		} else {
			return ErrDomainEmpty
		}
	}

	// If y and z are known, restrict x to z/y (integer division)
	if y.domain.IsSingleton() && z.domain.IsSingleton() {
		yv := y.domain.SingletonValue()
		zv := z.domain.SingletonValue()
		if yv != 0 && zv%yv == 0 {
			quotient := zv / yv
			if quotient >= 1 && quotient <= s.domainSize {
				newX := BitSet{n: s.domainSize, words: make([]uint64, (s.domainSize+63)/64)}
				idx := (quotient - 1) / 64
				off := uint((quotient - 1) % 64)
				newX.words[idx] |= 1 << off

				intersected := x.domain.Intersect(newX)
				if !bitSetEquals(intersected, x.domain) {
					s.trail = append(s.trail, FDChange{vid: x.ID, domain: x.domain.Clone()})
					x.domain = intersected
					if x.domain.Count() == 0 {
						return ErrDomainEmpty
					}
				}
			} else {
				return ErrDomainEmpty
			}
		} else {
			return ErrDomainEmpty
		}
	}

	// General case: restrict domains based on possible combinations
	return s.propagateMultiplyGeneral(x, y, z)
}

// propagateMultiplyGeneral handles the general case where not all variables are singleton
func (s *FDStore) propagateMultiplyGeneral(x, y, z *FDVar) error {
	// Compute possible values for z given x and y domains
	possibleZ := BitSet{n: s.domainSize, words: make([]uint64, (s.domainSize+63)/64)}
	x.domain.IterateValues(func(xv int) {
		y.domain.IterateValues(func(yv int) {
			product := xv * yv
			if product >= 1 && product <= s.domainSize {
				idx := (product - 1) / 64
				off := uint((product - 1) % 64)
				possibleZ.words[idx] |= 1 << off
			}
		})
	})

	intersectedZ := z.domain.Intersect(possibleZ)
	if !bitSetEquals(intersectedZ, z.domain) {
		s.trail = append(s.trail, FDChange{vid: z.ID, domain: z.domain.Clone()})
		z.domain = intersectedZ
		if z.domain.Count() == 0 {
			return ErrDomainEmpty
		}
		s.enqueue(z.ID)
	}

	// Compute possible values for x given y and z domains
	possibleX := BitSet{n: s.domainSize, words: make([]uint64, (s.domainSize+63)/64)}
	y.domain.IterateValues(func(yv int) {
		z.domain.IterateValues(func(zv int) {
			if yv != 0 && zv%yv == 0 {
				quotient := zv / yv
				if quotient >= 1 && quotient <= s.domainSize {
					idx := (quotient - 1) / 64
					off := uint((quotient - 1) % 64)
					possibleX.words[idx] |= 1 << off
				}
			}
		})
	})

	intersectedX := x.domain.Intersect(possibleX)
	if !bitSetEquals(intersectedX, x.domain) {
		s.trail = append(s.trail, FDChange{vid: x.ID, domain: x.domain.Clone()})
		x.domain = intersectedX
		if x.domain.Count() == 0 {
			return ErrDomainEmpty
		}
		s.enqueue(x.ID)
	}

	// Compute possible values for y given x and z domains
	possibleY := BitSet{n: s.domainSize, words: make([]uint64, (s.domainSize+63)/64)}
	x.domain.IterateValues(func(xv int) {
		z.domain.IterateValues(func(zv int) {
			if xv != 0 && zv%xv == 0 {
				quotient := zv / xv
				if quotient >= 1 && quotient <= s.domainSize {
					idx := (quotient - 1) / 64
					off := uint((quotient - 1) % 64)
					possibleY.words[idx] |= 1 << off
				}
			}
		})
	})

	intersectedY := y.domain.Intersect(possibleY)
	if !bitSetEquals(intersectedY, y.domain) {
		s.trail = append(s.trail, FDChange{vid: y.ID, domain: y.domain.Clone()})
		y.domain = intersectedY
		if y.domain.Count() == 0 {
			return ErrDomainEmpty
		}
		s.enqueue(y.ID)
	}

	return nil
}

// AddEqualityConstraint enforces x = y = z with bidirectional propagation.
// All variables must be from the same FDStore.
func (s *FDStore) AddEqualityConstraint(x, y, z *FDVar) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if x == nil || y == nil || z == nil {
		return ErrInvalidArgument
	}

	// Initialize arithmetic links map if needed
	if s.arithmeticLinks == nil {
		s.arithmeticLinks = make(map[int][]ArithmeticLink)
	}

	// Add links for bidirectional propagation
	link := ArithmeticLink{x: x, y: y, z: z, op: ArithmeticEquality}
	s.arithmeticLinks[x.ID] = append(s.arithmeticLinks[x.ID], link)
	s.arithmeticLinks[y.ID] = append(s.arithmeticLinks[y.ID], link)
	s.arithmeticLinks[z.ID] = append(s.arithmeticLinks[z.ID], link)

	// Initial propagation
	if err := s.propagateEqualityConstraint(x, y, z); err != nil {
		return err
	}

	// Enqueue all variables for further propagation
	s.enqueue(x.ID)
	s.enqueue(y.ID)
	s.enqueue(z.ID)

	if s.monitor != nil {
		s.monitor.RecordConstraint()
	}
	return s.propagateLocked()
}

// propagateEqualityConstraint performs the core propagation logic for x = y = z
func (s *FDStore) propagateEqualityConstraint(x, y, z *FDVar) error {
	// Compute the intersection of all domains
	intersection := x.domain.Intersect(y.domain).Intersect(z.domain)
	if intersection.Count() == 0 {
		return ErrDomainEmpty
	}

	// If any domain is singleton, all must be restricted to that value
	if x.domain.IsSingleton() {
		val := x.domain.SingletonValue()
		if !y.domain.Has(val) || !z.domain.Has(val) {
			return ErrDomainEmpty
		}
		// Restrict y and z to val
		if !y.domain.IsSingleton() {
			newY := BitSet{n: s.domainSize, words: make([]uint64, (s.domainSize+63)/64)}
			idx := (val - 1) / 64
			off := uint((val - 1) % 64)
			newY.words[idx] |= 1 << off
			intersectedY := y.domain.Intersect(newY)
			if !bitSetEquals(intersectedY, y.domain) {
				s.trail = append(s.trail, FDChange{vid: y.ID, domain: y.domain.Clone()})
				y.domain = intersectedY
				if y.domain.Count() == 0 {
					return ErrDomainEmpty
				}
			}
		}
		if !z.domain.IsSingleton() {
			newZ := BitSet{n: s.domainSize, words: make([]uint64, (s.domainSize+63)/64)}
			idx := (val - 1) / 64
			off := uint((val - 1) % 64)
			newZ.words[idx] |= 1 << off
			intersectedZ := z.domain.Intersect(newZ)
			if !bitSetEquals(intersectedZ, z.domain) {
				s.trail = append(s.trail, FDChange{vid: z.ID, domain: z.domain.Clone()})
				z.domain = intersectedZ
				if z.domain.Count() == 0 {
					return ErrDomainEmpty
				}
			}
		}
	} else if y.domain.IsSingleton() {
		val := y.domain.SingletonValue()
		if !x.domain.Has(val) || !z.domain.Has(val) {
			return ErrDomainEmpty
		}
		// Restrict x and z to val
		if !x.domain.IsSingleton() {
			newX := BitSet{n: s.domainSize, words: make([]uint64, (s.domainSize+63)/64)}
			idx := (val - 1) / 64
			off := uint((val - 1) % 64)
			newX.words[idx] |= 1 << off
			intersectedX := x.domain.Intersect(newX)
			if !bitSetEquals(intersectedX, x.domain) {
				s.trail = append(s.trail, FDChange{vid: x.ID, domain: x.domain.Clone()})
				x.domain = intersectedX
				if x.domain.Count() == 0 {
					return ErrDomainEmpty
				}
			}
		}
		if !z.domain.IsSingleton() {
			newZ := BitSet{n: s.domainSize, words: make([]uint64, (s.domainSize+63)/64)}
			idx := (val - 1) / 64
			off := uint((val - 1) % 64)
			newZ.words[idx] |= 1 << off
			intersectedZ := z.domain.Intersect(newZ)
			if !bitSetEquals(intersectedZ, z.domain) {
				s.trail = append(s.trail, FDChange{vid: z.ID, domain: z.domain.Clone()})
				z.domain = intersectedZ
				if z.domain.Count() == 0 {
					return ErrDomainEmpty
				}
			}
		}
	} else if z.domain.IsSingleton() {
		val := z.domain.SingletonValue()
		if !x.domain.Has(val) || !y.domain.Has(val) {
			return ErrDomainEmpty
		}
		// Restrict x and y to val
		if !x.domain.IsSingleton() {
			newX := BitSet{n: s.domainSize, words: make([]uint64, (s.domainSize+63)/64)}
			idx := (val - 1) / 64
			off := uint((val - 1) % 64)
			newX.words[idx] |= 1 << off
			intersectedX := x.domain.Intersect(newX)
			if !bitSetEquals(intersectedX, x.domain) {
				s.trail = append(s.trail, FDChange{vid: x.ID, domain: x.domain.Clone()})
				x.domain = intersectedX
				if x.domain.Count() == 0 {
					return ErrDomainEmpty
				}
			}
		}
		if !y.domain.IsSingleton() {
			newY := BitSet{n: s.domainSize, words: make([]uint64, (s.domainSize+63)/64)}
			idx := (val - 1) / 64
			off := uint((val - 1) % 64)
			newY.words[idx] |= 1 << off
			intersectedY := y.domain.Intersect(newY)
			if !bitSetEquals(intersectedY, y.domain) {
				s.trail = append(s.trail, FDChange{vid: y.ID, domain: y.domain.Clone()})
				y.domain = intersectedY
				if y.domain.Count() == 0 {
					return ErrDomainEmpty
				}
			}
		}
	} else {
		// General case: restrict all domains to their intersection
		if !bitSetEquals(intersection, x.domain) {
			s.trail = append(s.trail, FDChange{vid: x.ID, domain: x.domain.Clone()})
			x.domain = intersection
		}
		if !bitSetEquals(intersection, y.domain) {
			s.trail = append(s.trail, FDChange{vid: y.ID, domain: y.domain.Clone()})
			y.domain = intersection
		}
		if !bitSetEquals(intersection, z.domain) {
			s.trail = append(s.trail, FDChange{vid: z.ID, domain: z.domain.Clone()})
			z.domain = intersection
		}
	}

	return nil
}

// AddMinusConstraint enforces x - y = z with bidirectional propagation.
// All variables must be from the same FDStore.
func (s *FDStore) AddMinusConstraint(x, y, z *FDVar) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if x == nil || y == nil || z == nil {
		return ErrInvalidArgument
	}

	// Initialize arithmetic links map if needed
	if s.arithmeticLinks == nil {
		s.arithmeticLinks = make(map[int][]ArithmeticLink)
	}

	// Add links for bidirectional propagation
	link := ArithmeticLink{x: x, y: y, z: z, op: ArithmeticMinus}
	s.arithmeticLinks[x.ID] = append(s.arithmeticLinks[x.ID], link)
	s.arithmeticLinks[y.ID] = append(s.arithmeticLinks[y.ID], link)
	s.arithmeticLinks[z.ID] = append(s.arithmeticLinks[z.ID], link)

	// Initial propagation
	if err := s.propagateMinusConstraint(x, y, z); err != nil {
		return err
	}

	// Enqueue all variables for further propagation
	s.enqueue(x.ID)
	s.enqueue(y.ID)
	s.enqueue(z.ID)

	if s.monitor != nil {
		s.monitor.RecordConstraint()
	}
	return s.propagateLocked()
}

// propagateMinusConstraint performs the core propagation logic for x - y = z
func (s *FDStore) propagateMinusConstraint(x, y, z *FDVar) error {
	// For each pair of known domains, compute the possible values for the third

	// If x and y are known, restrict z to x-y
	if x.domain.IsSingleton() && y.domain.IsSingleton() {
		diff := x.domain.SingletonValue() - y.domain.SingletonValue()
		if diff >= 1 && diff <= s.domainSize {
			newZ := BitSet{n: s.domainSize, words: make([]uint64, (s.domainSize+63)/64)}
			idx := (diff - 1) / 64
			off := uint((diff - 1) % 64)
			newZ.words[idx] |= 1 << off

			intersected := z.domain.Intersect(newZ)
			if !bitSetEquals(intersected, z.domain) {
				s.trail = append(s.trail, FDChange{vid: z.ID, domain: z.domain.Clone()})
				z.domain = intersected
				if z.domain.Count() == 0 {
					return ErrDomainEmpty
				}
			}
		} else {
			// Difference is out of bounds, domain becomes empty
			return ErrDomainEmpty
		}
	}

	// If x and z are known, restrict y to x-z
	if x.domain.IsSingleton() && z.domain.IsSingleton() {
		diff := x.domain.SingletonValue() - z.domain.SingletonValue()
		if diff >= 1 && diff <= s.domainSize {
			newY := BitSet{n: s.domainSize, words: make([]uint64, (s.domainSize+63)/64)}
			idx := (diff - 1) / 64
			off := uint((diff - 1) % 64)
			newY.words[idx] |= 1 << off

			intersected := y.domain.Intersect(newY)
			if !bitSetEquals(intersected, y.domain) {
				s.trail = append(s.trail, FDChange{vid: y.ID, domain: y.domain.Clone()})
				y.domain = intersected
				if y.domain.Count() == 0 {
					return ErrDomainEmpty
				}
			}
		} else {
			return ErrDomainEmpty
		}
	}

	// If y and z are known, restrict x to y+z
	if y.domain.IsSingleton() && z.domain.IsSingleton() {
		sum := y.domain.SingletonValue() + z.domain.SingletonValue()
		if sum >= 1 && sum <= s.domainSize {
			newX := BitSet{n: s.domainSize, words: make([]uint64, (s.domainSize+63)/64)}
			idx := (sum - 1) / 64
			off := uint((sum - 1) % 64)
			newX.words[idx] |= 1 << off

			intersected := x.domain.Intersect(newX)
			if !bitSetEquals(intersected, x.domain) {
				s.trail = append(s.trail, FDChange{vid: x.ID, domain: x.domain.Clone()})
				x.domain = intersected
				if x.domain.Count() == 0 {
					return ErrDomainEmpty
				}
			}
		} else {
			return ErrDomainEmpty
		}
	}

	// General case: restrict domains based on possible combinations
	return s.propagateMinusGeneral(x, y, z)
}

// propagateMinusGeneral handles the general case where not all variables are singleton
func (s *FDStore) propagateMinusGeneral(x, y, z *FDVar) error {
	// Compute possible values for z given x and y domains
	possibleZ := BitSet{n: s.domainSize, words: make([]uint64, (s.domainSize+63)/64)}
	x.domain.IterateValues(func(xv int) {
		y.domain.IterateValues(func(yv int) {
			diff := xv - yv
			if diff >= 1 && diff <= s.domainSize {
				idx := (diff - 1) / 64
				off := uint((diff - 1) % 64)
				possibleZ.words[idx] |= 1 << off
			}
		})
	})

	intersectedZ := z.domain.Intersect(possibleZ)
	if !bitSetEquals(intersectedZ, z.domain) {
		s.trail = append(s.trail, FDChange{vid: z.ID, domain: z.domain.Clone()})
		z.domain = intersectedZ
		if z.domain.Count() == 0 {
			return ErrDomainEmpty
		}
		s.enqueue(z.ID)
	}

	// Compute possible values for x given y and z domains
	possibleX := BitSet{n: s.domainSize, words: make([]uint64, (s.domainSize+63)/64)}
	y.domain.IterateValues(func(yv int) {
		z.domain.IterateValues(func(zv int) {
			sum := yv + zv
			if sum >= 1 && sum <= s.domainSize {
				idx := (sum - 1) / 64
				off := uint((sum - 1) % 64)
				possibleX.words[idx] |= 1 << off
			}
		})
	})

	intersectedX := x.domain.Intersect(possibleX)
	if !bitSetEquals(intersectedX, x.domain) {
		s.trail = append(s.trail, FDChange{vid: x.ID, domain: x.domain.Clone()})
		x.domain = intersectedX
		if x.domain.Count() == 0 {
			return ErrDomainEmpty
		}
		s.enqueue(x.ID)
	}

	// Compute possible values for y given x and z domains
	possibleY := BitSet{n: s.domainSize, words: make([]uint64, (s.domainSize+63)/64)}
	x.domain.IterateValues(func(xv int) {
		z.domain.IterateValues(func(zv int) {
			diff := xv - zv
			if diff >= 1 && diff <= s.domainSize {
				idx := (diff - 1) / 64
				off := uint((diff - 1) % 64)
				possibleY.words[idx] |= 1 << off
			}
		})
	})

	intersectedY := y.domain.Intersect(possibleY)
	if !bitSetEquals(intersectedY, y.domain) {
		s.trail = append(s.trail, FDChange{vid: y.ID, domain: y.domain.Clone()})
		y.domain = intersectedY
		if y.domain.Count() == 0 {
			return ErrDomainEmpty
		}
		s.enqueue(y.ID)
	}

	return nil
}

// AddQuotientConstraint enforces x / y = z with bidirectional propagation.
// All variables must be from the same FDStore. Division is integer division.
func (s *FDStore) AddQuotientConstraint(x, y, z *FDVar) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if x == nil || y == nil || z == nil {
		return ErrInvalidArgument
	}

	// Initialize arithmetic links map if needed
	if s.arithmeticLinks == nil {
		s.arithmeticLinks = make(map[int][]ArithmeticLink)
	}

	// Add links for bidirectional propagation
	link := ArithmeticLink{x: x, y: y, z: z, op: ArithmeticQuotient}
	s.arithmeticLinks[x.ID] = append(s.arithmeticLinks[x.ID], link)
	s.arithmeticLinks[y.ID] = append(s.arithmeticLinks[y.ID], link)
	s.arithmeticLinks[z.ID] = append(s.arithmeticLinks[z.ID], link)

	// Initial propagation
	if err := s.propagateQuotientConstraint(x, y, z); err != nil {
		return err
	}

	// Enqueue all variables for further propagation
	s.enqueue(x.ID)
	s.enqueue(y.ID)
	s.enqueue(z.ID)

	if s.monitor != nil {
		s.monitor.RecordConstraint()
	}
	return s.propagateLocked()
}

// propagateQuotientConstraint performs the core propagation logic for x / y = z
func (s *FDStore) propagateQuotientConstraint(x, y, z *FDVar) error {
	// For each pair of known domains, compute the possible values for the third

	// If x and y are known, restrict z to x/y (integer division)
	if x.domain.IsSingleton() && y.domain.IsSingleton() {
		xv := x.domain.SingletonValue()
		yv := y.domain.SingletonValue()
		if yv != 0 {
			quotient := xv / yv
			if quotient >= 1 && quotient <= s.domainSize {
				newZ := BitSet{n: s.domainSize, words: make([]uint64, (s.domainSize+63)/64)}
				idx := (quotient - 1) / 64
				off := uint((quotient - 1) % 64)
				newZ.words[idx] |= 1 << off

				intersected := z.domain.Intersect(newZ)
				if !bitSetEquals(intersected, z.domain) {
					s.trail = append(s.trail, FDChange{vid: z.ID, domain: z.domain.Clone()})
					z.domain = intersected
					if z.domain.Count() == 0 {
						return ErrDomainEmpty
					}
				}
			} else {
				// Quotient is out of bounds, domain becomes empty
				return ErrDomainEmpty
			}
		} else {
			// Division by zero, domain becomes empty
			return ErrDomainEmpty
		}
	}

	// If x and z are known, restrict y such that x/y = z (y divides x and x/y = z)
	if x.domain.IsSingleton() && z.domain.IsSingleton() {
		xv := x.domain.SingletonValue()
		zv := z.domain.SingletonValue()
		if zv != 0 && xv%zv == 0 {
			// y must be xv/zv and must be in domain
			yValue := xv / zv
			if yValue >= 1 && yValue <= s.domainSize {
				newY := BitSet{n: s.domainSize, words: make([]uint64, (s.domainSize+63)/64)}
				idx := (yValue - 1) / 64
				off := uint((yValue - 1) % 64)
				newY.words[idx] |= 1 << off

				intersected := y.domain.Intersect(newY)
				if !bitSetEquals(intersected, y.domain) {
					s.trail = append(s.trail, FDChange{vid: y.ID, domain: y.domain.Clone()})
					y.domain = intersected
					if y.domain.Count() == 0 {
						return ErrDomainEmpty
					}
				}
			} else {
				return ErrDomainEmpty
			}
		} else {
			// z doesn't divide x, domain becomes empty
			return ErrDomainEmpty
		}
	}

	// If y and z are known, restrict x to y*z
	if y.domain.IsSingleton() && z.domain.IsSingleton() {
		product := y.domain.SingletonValue() * z.domain.SingletonValue()
		if product >= 1 && product <= s.domainSize {
			newX := BitSet{n: s.domainSize, words: make([]uint64, (s.domainSize+63)/64)}
			idx := (product - 1) / 64
			off := uint((product - 1) % 64)
			newX.words[idx] |= 1 << off

			intersected := x.domain.Intersect(newX)
			if !bitSetEquals(intersected, x.domain) {
				s.trail = append(s.trail, FDChange{vid: x.ID, domain: x.domain.Clone()})
				x.domain = intersected
				if x.domain.Count() == 0 {
					return ErrDomainEmpty
				}
			}
		} else {
			return ErrDomainEmpty
		}
	}

	// General case: restrict domains based on possible combinations
	return s.propagateQuotientGeneral(x, y, z)
}

// propagateQuotientGeneral handles the general case where not all variables are singleton
func (s *FDStore) propagateQuotientGeneral(x, y, z *FDVar) error {
	// Compute possible values for z given x and y domains
	possibleZ := BitSet{n: s.domainSize, words: make([]uint64, (s.domainSize+63)/64)}
	x.domain.IterateValues(func(xv int) {
		y.domain.IterateValues(func(yv int) {
			if yv != 0 {
				quotient := xv / yv
				if quotient >= 1 && quotient <= s.domainSize {
					idx := (quotient - 1) / 64
					off := uint((quotient - 1) % 64)
					possibleZ.words[idx] |= 1 << off
				}
			}
		})
	})

	intersectedZ := z.domain.Intersect(possibleZ)
	if !bitSetEquals(intersectedZ, z.domain) {
		s.trail = append(s.trail, FDChange{vid: z.ID, domain: z.domain.Clone()})
		z.domain = intersectedZ
		if z.domain.Count() == 0 {
			return ErrDomainEmpty
		}
		s.enqueue(z.ID)
	}

	// Compute possible values for x given y and z domains
	possibleX := BitSet{n: s.domainSize, words: make([]uint64, (s.domainSize+63)/64)}
	y.domain.IterateValues(func(yv int) {
		z.domain.IterateValues(func(zv int) {
			product := yv * zv
			if product >= 1 && product <= s.domainSize {
				idx := (product - 1) / 64
				off := uint((product - 1) % 64)
				possibleX.words[idx] |= 1 << off
			}
		})
	})

	intersectedX := x.domain.Intersect(possibleX)
	if !bitSetEquals(intersectedX, x.domain) {
		s.trail = append(s.trail, FDChange{vid: x.ID, domain: x.domain.Clone()})
		x.domain = intersectedX
		if x.domain.Count() == 0 {
			return ErrDomainEmpty
		}
		s.enqueue(x.ID)
	}

	// Compute possible values for y given x and z domains
	possibleY := BitSet{n: s.domainSize, words: make([]uint64, (s.domainSize+63)/64)}
	x.domain.IterateValues(func(xv int) {
		z.domain.IterateValues(func(zv int) {
			if zv != 0 && xv%zv == 0 {
				quotient := xv / zv
				if quotient >= 1 && quotient <= s.domainSize {
					idx := (quotient - 1) / 64
					off := uint((quotient - 1) % 64)
					possibleY.words[idx] |= 1 << off
				}
			}
		})
	})

	intersectedY := y.domain.Intersect(possibleY)
	if !bitSetEquals(intersectedY, y.domain) {
		s.trail = append(s.trail, FDChange{vid: y.ID, domain: y.domain.Clone()})
		y.domain = intersectedY
		if y.domain.Count() == 0 {
			return ErrDomainEmpty
		}
		s.enqueue(y.ID)
	}

	return nil
}

// AddModuloConstraint enforces x % y = z with bidirectional propagation.
// All variables must be from the same FDStore. Modulo is integer modulo operation.
func (s *FDStore) AddModuloConstraint(x, y, z *FDVar) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if x == nil || y == nil || z == nil {
		return ErrInvalidArgument
	}

	// Initialize arithmetic links map if needed
	if s.arithmeticLinks == nil {
		s.arithmeticLinks = make(map[int][]ArithmeticLink)
	}

	// Add links for bidirectional propagation
	link := ArithmeticLink{x: x, y: y, z: z, op: ArithmeticModulo}
	s.arithmeticLinks[x.ID] = append(s.arithmeticLinks[x.ID], link)
	s.arithmeticLinks[y.ID] = append(s.arithmeticLinks[y.ID], link)
	s.arithmeticLinks[z.ID] = append(s.arithmeticLinks[z.ID], link)

	// Initial propagation
	if err := s.propagateModuloConstraint(x, y, z); err != nil {
		return err
	}

	// Enqueue all variables for further propagation
	s.enqueue(x.ID)
	s.enqueue(y.ID)
	s.enqueue(z.ID)

	if s.monitor != nil {
		s.monitor.RecordConstraint()
	}
	return s.propagateLocked()
}

// propagateModuloConstraint performs the core propagation logic for x % y = z
func (s *FDStore) propagateModuloConstraint(x, y, z *FDVar) error {
	// For each pair of known domains, compute the possible values for the third

	// If x and y are known, restrict z to x % y
	if x.domain.IsSingleton() && y.domain.IsSingleton() {
		xv := x.domain.SingletonValue()
		yv := y.domain.SingletonValue()
		if yv != 0 {
			modulo := xv % yv
			if modulo >= 0 && modulo <= s.domainSize {
				newZ := BitSet{n: s.domainSize, words: make([]uint64, (s.domainSize+63)/64)}
				idx := (modulo - 1) / 64
				off := uint((modulo - 1) % 64)
				newZ.words[idx] |= 1 << off

				intersected := z.domain.Intersect(newZ)
				if !bitSetEquals(intersected, z.domain) {
					s.trail = append(s.trail, FDChange{vid: z.ID, domain: z.domain.Clone()})
					z.domain = intersected
					if z.domain.Count() == 0 {
						return ErrDomainEmpty
					}
				}
			} else {
				// Modulo result is out of bounds, domain becomes empty
				return ErrDomainEmpty
			}
		} else {
			// Division by zero, domain becomes empty
			return ErrDomainEmpty
		}
	}

	// If x and z are known, restrict y such that x % y = z
	if x.domain.IsSingleton() && z.domain.IsSingleton() {
		xv := x.domain.SingletonValue()
		zv := z.domain.SingletonValue()
		// y must be > z and y divides (xv - zv)
		if zv >= 0 && zv < xv {
			possibleY := BitSet{n: s.domainSize, words: make([]uint64, (s.domainSize+63)/64)}
			for yv := zv + 1; yv <= s.domainSize && yv <= xv; yv++ {
				if (xv-zv)%yv == 0 {
					idx := (yv - 1) / 64
					off := uint((yv - 1) % 64)
					possibleY.words[idx] |= 1 << off
				}
			}

			intersected := y.domain.Intersect(possibleY)
			if !bitSetEquals(intersected, y.domain) {
				s.trail = append(s.trail, FDChange{vid: y.ID, domain: y.domain.Clone()})
				y.domain = intersected
				if y.domain.Count() == 0 {
					return ErrDomainEmpty
				}
			}
		} else {
			return ErrDomainEmpty
		}
	}

	// If y and z are known, restrict x such that x % y = z
	if y.domain.IsSingleton() && z.domain.IsSingleton() {
		yv := y.domain.SingletonValue()
		zv := z.domain.SingletonValue()
		if yv > 0 && zv >= 0 && zv < yv {
			// x must be congruent to z modulo y: x ≡ z (mod y)
			// So x = y * k + z for some integer k, where x is in domain
			possibleX := BitSet{n: s.domainSize, words: make([]uint64, (s.domainSize+63)/64)}
			for k := 0; ; k++ {
				xval := yv*k + zv
				if xval > s.domainSize {
					break
				}
				if xval >= 1 {
					idx := (xval - 1) / 64
					off := uint((xval - 1) % 64)
					possibleX.words[idx] |= 1 << off
				}
			}

			intersected := x.domain.Intersect(possibleX)
			if !bitSetEquals(intersected, x.domain) {
				s.trail = append(s.trail, FDChange{vid: x.ID, domain: x.domain.Clone()})
				x.domain = intersected
				if x.domain.Count() == 0 {
					return ErrDomainEmpty
				}
			}
		} else {
			return ErrDomainEmpty
		}
	}

	// General case: restrict domains based on possible combinations
	return s.propagateModuloGeneral(x, y, z)
}

// propagateModuloGeneral handles the general case where not all variables are singleton
func (s *FDStore) propagateModuloGeneral(x, y, z *FDVar) error {
	// Compute possible values for z given x and y domains
	possibleZ := BitSet{n: s.domainSize, words: make([]uint64, (s.domainSize+63)/64)}
	x.domain.IterateValues(func(xv int) {
		y.domain.IterateValues(func(yv int) {
			if yv != 0 {
				modulo := xv % yv
				if modulo >= 0 && modulo <= s.domainSize {
					idx := (modulo - 1) / 64
					off := uint((modulo - 1) % 64)
					possibleZ.words[idx] |= 1 << off
				}
			}
		})
	})

	intersectedZ := z.domain.Intersect(possibleZ)
	if !bitSetEquals(intersectedZ, z.domain) {
		s.trail = append(s.trail, FDChange{vid: z.ID, domain: z.domain.Clone()})
		z.domain = intersectedZ
		if z.domain.Count() == 0 {
			return ErrDomainEmpty
		}
		s.enqueue(z.ID)
	}

	// Compute possible values for x given y and z domains
	possibleX := BitSet{n: s.domainSize, words: make([]uint64, (s.domainSize+63)/64)}
	y.domain.IterateValues(func(yv int) {
		z.domain.IterateValues(func(zv int) {
			if yv > 0 && zv >= 0 && zv < yv {
				// x must be congruent to z modulo y
				for k := 0; ; k++ {
					xval := yv*k + zv
					if xval > s.domainSize {
						break
					}
					if xval >= 1 {
						idx := (xval - 1) / 64
						off := uint((xval - 1) % 64)
						possibleX.words[idx] |= 1 << off
					}
				}
			}
		})
	})

	intersectedX := x.domain.Intersect(possibleX)
	if !bitSetEquals(intersectedX, x.domain) {
		s.trail = append(s.trail, FDChange{vid: x.ID, domain: x.domain.Clone()})
		x.domain = intersectedX
		if x.domain.Count() == 0 {
			return ErrDomainEmpty
		}
		s.enqueue(x.ID)
	}

	// Compute possible values for y given x and z domains
	possibleY := BitSet{n: s.domainSize, words: make([]uint64, (s.domainSize+63)/64)}
	x.domain.IterateValues(func(xv int) {
		z.domain.IterateValues(func(zv int) {
			if zv >= 0 && zv <= xv { // y must be > z and x >= z
				remainder := xv - zv
				for yv := zv + 1; yv <= s.domainSize; yv++ {
					if remainder%yv == 0 {
						idx := (yv - 1) / 64
						off := uint((yv - 1) % 64)
						possibleY.words[idx] |= 1 << off
					}
				}
			}
		})
	})

	intersectedY := y.domain.Intersect(possibleY)
	if !bitSetEquals(intersectedY, y.domain) {
		s.trail = append(s.trail, FDChange{vid: y.ID, domain: y.domain.Clone()})
		y.domain = intersectedY
		if y.domain.Count() == 0 {
			return ErrDomainEmpty
		}
		s.enqueue(y.ID)
	}

	return nil
}
