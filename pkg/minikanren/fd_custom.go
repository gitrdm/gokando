package minikanren

// fd_custom.go: custom constraint interfaces for FDStore

// CustomConstraint represents a user-defined constraint that can propagate
type CustomConstraint interface {
	// Variables returns the list of variables this constraint involves
	Variables() []*FDVar

	// Propagate performs constraint propagation, potentially narrowing domains
	// Returns true if any domain was changed, false otherwise
	// If the constraint becomes inconsistent, returns an error
	Propagate(store *FDStore) (bool, error)

	// IsSatisfied returns true if the constraint is satisfied given current domains
	// This is used for checking consistency during search
	IsSatisfied() bool
}

// customConstraintLink represents a custom constraint
type customConstraintLink struct {
	constraint CustomConstraint
}

// AddCustomConstraint adds a user-defined custom constraint to the store
func (s *FDStore) AddCustomConstraint(constraint CustomConstraint) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if constraint == nil {
		return ErrInvalidArgument
	}

	// Initialize custom constraints map if needed
	if s.customConstraints == nil {
		s.customConstraints = make([]CustomConstraint, 0)
	}

	// Add the constraint
	s.customConstraints = append(s.customConstraints, constraint)

	// Perform initial propagation
	changed, err := constraint.Propagate(s)
	if err != nil {
		return err
	}

	// If domains changed, enqueue all variables for further propagation
	if changed {
		vars := constraint.Variables()
		for _, v := range vars {
			s.enqueue(v.ID)
		}
		if s.monitor != nil {
			s.monitor.RecordConstraint()
		}
		return s.propagateLocked()
	}

	return nil
}

// propagateCustomConstraintsLocked runs propagation for all custom constraints
func (s *FDStore) propagateCustomConstraintsLocked() error {
	if s.customConstraints == nil {
		return nil
	}

	changed := false
	for _, constraint := range s.customConstraints {
		constraintChanged, err := constraint.Propagate(s)
		if err != nil {
			return err
		}
		if constraintChanged {
			changed = true
		}
	}

	// If any constraint changed domains, we need to run full propagation
	if changed {
		return s.propagateLocked()
	}

	return nil
}

// Example custom constraint implementations

// SumConstraint enforces that the sum of variables equals a target value
type SumConstraint struct {
	vars   []*FDVar
	target int
}

// NewSumConstraint creates a new sum constraint
func NewSumConstraint(vars []*FDVar, target int) *SumConstraint {
	return &SumConstraint{vars: vars, target: target}
}

// Variables returns the variables involved in this constraint
func (c *SumConstraint) Variables() []*FDVar {
	return c.vars
}

// Propagate performs constraint propagation for the sum constraint
func (c *SumConstraint) Propagate(store *FDStore) (bool, error) {
	// Simple sum propagation: if all but one variable are fixed,
	// we can determine the value of the remaining variable

	fixedSum := 0
	unfixedVars := make([]*FDVar, 0)

	for _, v := range c.vars {
		if v.domain.IsSingleton() {
			fixedSum += v.domain.SingletonValue()
		} else {
			unfixedVars = append(unfixedVars, v)
		}
	}

	// If only one variable is unfixed, we can compute its required value
	if len(unfixedVars) == 1 {
		requiredValue := c.target - fixedSum
		if requiredValue < 1 || requiredValue > store.domainSize {
			return false, ErrInconsistent
		}

		// Try to assign the required value
		v := unfixedVars[0]
		if !v.domain.Has(requiredValue) {
			return false, ErrInconsistent
		}

		// Create singleton domain for the required value
		singletonDom := NewBitSet(store.domainSize)
		for i := 1; i <= store.domainSize; i++ {
			if i != requiredValue {
				singletonDom = singletonDom.RemoveValue(i)
			}
		}

		// Intersect domains directly (since we're already in a locked context)
		newDom := v.domain.Intersect(singletonDom)
		if !bitSetEquals(newDom, v.domain) {
			store.trail = append(store.trail, FDChange{vid: v.ID, domain: v.domain.Clone()})
			v.domain = newDom
			if v.domain.Count() == 0 {
				return false, ErrDomainEmpty
			}
			store.enqueue(v.ID)
			return true, nil
		}
		return false, nil
	}

	// More sophisticated propagation could be added here:
	// - Prune domains based on minimum/maximum possible sums
	// - Use bounds consistency, etc.

	return false, nil
}

// IsSatisfied checks if the sum constraint is satisfied
func (c *SumConstraint) IsSatisfied() bool {
	sum := 0
	for _, v := range c.vars {
		if !v.domain.IsSingleton() {
			return false // Not fully assigned yet
		}
		sum += v.domain.SingletonValue()
	}
	return sum == c.target
}

// AllDifferentConstraint is a custom version of the all-different constraint
// This demonstrates how built-in constraints can be reimplemented as custom constraints
type AllDifferentConstraint struct {
	vars []*FDVar
}

// NewAllDifferentConstraint creates a new all-different constraint
func NewAllDifferentConstraint(vars []*FDVar) *AllDifferentConstraint {
	return &AllDifferentConstraint{vars: vars}
}

// Variables returns the variables involved in this constraint
func (c *AllDifferentConstraint) Variables() []*FDVar {
	return c.vars
}

// Propagate performs constraint propagation for all-different
func (c *AllDifferentConstraint) Propagate(store *FDStore) (bool, error) {
	// Simple propagation: if any value appears in multiple singleton domains, inconsistent
	valueCount := make(map[int]int)

	for _, v := range c.vars {
		if v.domain.IsSingleton() {
			val := v.domain.SingletonValue()
			valueCount[val]++
			if valueCount[val] > 1 {
				return false, ErrInconsistent
			}
		}
	}

	// Remove singleton values from other domains
	changed := false
	for _, v := range c.vars {
		if v.domain.IsSingleton() {
			val := v.domain.SingletonValue()
			for _, other := range c.vars {
				if other != v && other.domain.Has(val) {
					store.trail = append(store.trail, FDChange{vid: other.ID, domain: other.domain.Clone()})
					other.domain = other.domain.RemoveValue(val)
					if other.domain.Count() == 0 {
						return false, ErrDomainEmpty
					}
					store.enqueue(other.ID)
					changed = true
				}
			}
		}
	}

	return changed, nil
}

// IsSatisfied checks if all variables have distinct values
func (c *AllDifferentConstraint) IsSatisfied() bool {
	values := make(map[int]bool)
	for _, v := range c.vars {
		if !v.domain.IsSingleton() {
			return false
		}
		val := v.domain.SingletonValue()
		if values[val] {
			return false
		}
		values[val] = true
	}
	return true
}
