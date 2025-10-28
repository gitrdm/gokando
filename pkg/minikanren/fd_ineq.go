package minikanren

// fd_ineq.go: arithmetic inequality constraints for FDStore

// InequalityType represents the type of inequality constraint
type InequalityType int

const (
	IneqLessThan     InequalityType = iota // X < Y
	IneqLessEqual                          // X <= Y
	IneqGreaterThan                        // X > Y
	IneqGreaterEqual                       // X >= Y
	IneqNotEqual                           // X != Y
)

// ineqLink represents an inequality constraint between two variables
type ineqLink struct {
	other *FDVar
	typ   InequalityType
}

// AddInequalityConstraint adds an inequality constraint between two variables.
// The constraint enforces the relationship specified by the inequality type.
func (s *FDStore) AddInequalityConstraint(x, y *FDVar, typ InequalityType) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if x == nil || y == nil {
		return ErrInvalidArgument
	}

	// Initialize inequality links map if needed
	if s.ineqLinks == nil {
		s.ineqLinks = make(map[int][]ineqLink)
	}

	// Add bidirectional links (most inequalities are symmetric in propagation)
	s.ineqLinks[x.ID] = append(s.ineqLinks[x.ID], ineqLink{other: y, typ: typ})

	// For symmetric inequalities, add reverse link
	switch typ {
	case IneqNotEqual:
		s.ineqLinks[y.ID] = append(s.ineqLinks[y.ID], ineqLink{other: x, typ: typ})
	case IneqLessThan:
		s.ineqLinks[y.ID] = append(s.ineqLinks[y.ID], ineqLink{other: x, typ: IneqGreaterThan})
	case IneqLessEqual:
		s.ineqLinks[y.ID] = append(s.ineqLinks[y.ID], ineqLink{other: x, typ: IneqGreaterEqual})
	case IneqGreaterThan:
		s.ineqLinks[y.ID] = append(s.ineqLinks[y.ID], ineqLink{other: x, typ: IneqLessThan})
	case IneqGreaterEqual:
		s.ineqLinks[y.ID] = append(s.ineqLinks[y.ID], ineqLink{other: x, typ: IneqLessEqual})
	}

	// Perform initial propagation
	if err := s.propagateInequalityLocked(x, y, typ); err != nil {
		return err
	}
	if err := s.propagateInequalityLocked(y, x, reverseInequalityType(typ)); err != nil {
		return err
	}

	// Enqueue both variables for further propagation
	s.enqueue(x.ID)
	s.enqueue(y.ID)
	if s.monitor != nil {
		s.monitor.RecordConstraint()
	}
	return s.propagateLocked()
}

// propagateInequalityLocked performs initial pruning for an inequality constraint
func (s *FDStore) propagateInequalityLocked(x, y *FDVar, typ InequalityType) error {
	switch typ {
	case IneqLessThan:
		return s.propagateLessThan(x, y)
	case IneqLessEqual:
		return s.propagateLessEqual(x, y)
	case IneqGreaterThan:
		return s.propagateGreaterThan(x, y)
	case IneqGreaterEqual:
		return s.propagateGreaterEqual(x, y)
	case IneqNotEqual:
		return s.propagateNotEqual(x, y)
	default:
		return ErrInvalidArgument
	}
}

// propagateLessThan prunes domains for X < Y constraint
func (s *FDStore) propagateLessThan(x, y *FDVar) error {
	// X < Y means X cannot take values >= min possible Y
	minY := findMinValue(y.domain)
	if minY > 1 {
		// Remove values from X that are >= minY
		newXDom := x.domain.Clone()
		for val := minY; val <= s.domainSize; val++ {
			newXDom = newXDom.RemoveValue(val)
		}
		if !bitSetEquals(newXDom, x.domain) {
			s.trail = append(s.trail, FDChange{vid: x.ID, domain: x.domain.Clone()})
			x.domain = newXDom
			if x.domain.Count() == 0 {
				return ErrDomainEmpty
			}
		}
	}

	// Y cannot take values <= max possible X
	maxX := findMaxValue(x.domain)
	if maxX < s.domainSize {
		newYDom := y.domain.Clone()
		for val := 1; val <= maxX; val++ {
			newYDom = newYDom.RemoveValue(val)
		}
		if !bitSetEquals(newYDom, y.domain) {
			s.trail = append(s.trail, FDChange{vid: y.ID, domain: y.domain.Clone()})
			y.domain = newYDom
			if y.domain.Count() == 0 {
				return ErrDomainEmpty
			}
		}
	}

	return nil
}

// propagateLessEqual prunes domains for X <= Y constraint
func (s *FDStore) propagateLessEqual(x, y *FDVar) error {
	// X <= Y means X cannot take values > min possible Y
	minY := findMinValue(y.domain)
	if minY > 1 {
		newXDom := x.domain.Clone()
		for val := minY + 1; val <= s.domainSize; val++ {
			newXDom = newXDom.RemoveValue(val)
		}
		if !bitSetEquals(newXDom, x.domain) {
			s.trail = append(s.trail, FDChange{vid: x.ID, domain: x.domain.Clone()})
			x.domain = newXDom
			if x.domain.Count() == 0 {
				return ErrDomainEmpty
			}
		}
	}

	// Y cannot take values < max possible X
	maxX := findMaxValue(x.domain)
	if maxX < s.domainSize {
		newYDom := y.domain.Clone()
		for val := 1; val < maxX; val++ {
			newYDom = newYDom.RemoveValue(val)
		}
		if !bitSetEquals(newYDom, y.domain) {
			s.trail = append(s.trail, FDChange{vid: y.ID, domain: y.domain.Clone()})
			y.domain = newYDom
			if y.domain.Count() == 0 {
				return ErrDomainEmpty
			}
		}
	}

	return nil
}

// propagateGreaterThan prunes domains for X > Y constraint
func (s *FDStore) propagateGreaterThan(x, y *FDVar) error {
	return s.propagateLessThan(y, x) // X > Y is equivalent to Y < X
}

// propagateGreaterEqual prunes domains for X >= Y constraint
func (s *FDStore) propagateGreaterEqual(x, y *FDVar) error {
	return s.propagateLessEqual(y, x) // X >= Y is equivalent to Y <= X
}

// propagateNotEqual prunes domains for X != Y constraint
func (s *FDStore) propagateNotEqual(x, y *FDVar) error {
	// If both domains are singletons and equal, inconsistency
	if x.domain.IsSingleton() && y.domain.IsSingleton() {
		if x.domain.SingletonValue() == y.domain.SingletonValue() {
			return ErrInconsistent
		}
		return nil
	}

	// If one is singleton, remove that value from the other
	if x.domain.IsSingleton() {
		val := x.domain.SingletonValue()
		if y.domain.Has(val) {
			s.trail = append(s.trail, FDChange{vid: y.ID, domain: y.domain.Clone()})
			y.domain = y.domain.RemoveValue(val)
			if y.domain.Count() == 0 {
				return ErrDomainEmpty
			}
		}
	}

	if y.domain.IsSingleton() {
		val := y.domain.SingletonValue()
		if x.domain.Has(val) {
			s.trail = append(s.trail, FDChange{vid: x.ID, domain: x.domain.Clone()})
			x.domain = x.domain.RemoveValue(val)
			if x.domain.Count() == 0 {
				return ErrDomainEmpty
			}
		}
	}

	return nil
}

// Helper functions for finding min/max values in domains
func findMinValue(dom BitSet) int {
	for i := 1; i <= dom.n; i++ {
		if dom.Has(i) {
			return i
		}
	}
	return dom.n + 1 // No values
}

func findMaxValue(dom BitSet) int {
	for i := dom.n; i >= 1; i-- {
		if dom.Has(i) {
			return i
		}
	}
	return 0 // No values
}

func reverseInequalityType(typ InequalityType) InequalityType {
	switch typ {
	case IneqLessThan:
		return IneqGreaterThan
	case IneqLessEqual:
		return IneqGreaterEqual
	case IneqGreaterThan:
		return IneqLessThan
	case IneqGreaterEqual:
		return IneqLessEqual
	case IneqNotEqual:
		return IneqNotEqual
	default:
		return typ
	}
}
