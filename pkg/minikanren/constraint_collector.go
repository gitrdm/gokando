package minikanren

import (
	"fmt"
	"sync"
)

// ReifiedConstraint wraps a constraint and associates it with a reification
// variable. This is a temporary solution until reification is more deeply
// integrated into the constraint system.
type ReifiedConstraint struct {
	Constraint
	Var int64
	Val int
}

package minikanren

import (
	"fmt"
	"sync"
)

// ReifiedConstraint wraps a constraint and associates it with a reification
// variable. This is a temporary solution until reification is more deeply
// integrated into the constraint system.
type ReifiedConstraint struct {
	Constraint
	Var int64
	Val int
}

// ConstraintCollector is a special implementation of ConstraintStore
// that collects constraints instead of solving them. This is used by
// high-level APIs like FDSolve to gather all constraints from a goal
// before passing them to a batch solver.
type ConstraintCollector struct {
	constraints []Constraint
	reified     []Reification
	mu          sync.Mutex
}

// NewConstraintCollector creates a new, empty constraint collector.
func NewConstraintCollector() *ConstraintCollector {
	return &ConstraintCollector{}
}

// AddConstraint adds a constraint to the collector. For reified constraints,
// it records the reification mapping.
func (cc *ConstraintCollector) AddConstraint(constraint Constraint) error {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	if rc, ok := constraint.(*ReifiedConstraint); ok {
		cc.reified = append(cc.reified, Reification{
			Var: rc.Var,
			Val: rc.Val,
		})
		cc.constraints = append(cc.constraints, rc.Constraint)
	} else {
		cc.constraints = append(cc.constraints, constraint)
	}
	return nil
}

// AddConstraintDeferred is a no-op for the collector, as all constraints
// are deferred by nature in this context.
func (cc *ConstraintCollector) AddConstraintDeferred(constraint Constraint) error {
	return cc.AddConstraint(constraint)
}

// AddBinding is not supported by the collector and will panic.
// The collector is only for gathering constraints, not for managing bindings.
func (cc *ConstraintCollector) AddBinding(varID int64, term Term) error {
	panic("AddBinding should not be called on a ConstraintCollector")
}

// GetBinding is not supported by the collector and will panic.
func (cc *ConstraintCollector) GetBinding(varID int64) Term {
	panic("GetBinding should not be called on a ConstraintCollector")
}

// GetSubstitution is not supported by the collector and will panic.
func (cc *ConstraintCollector) GetSubstitution() *Substitution {
	panic("GetSubstitution should not be called on a ConstraintCollector")
}

// GetConstraints returns the list of constraints collected so far.
func (cc *ConstraintCollector) GetConstraints() []Constraint {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	return cc.constraints
}

// GetReified returns the list of reification mappings collected so far.
func (cc *ConstraintCollector) GetReified() []Reification {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	return cc.reified
}

// Clone creates a shallow copy of the collector. The underlying constraint
// slices are copied, but the constraints themselves are not cloned.
func (cc *ConstraintCollector) Clone() ConstraintStore {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	newC := &ConstraintCollector{
		constraints: make([]Constraint, len(cc.constraints)),
		reified:     make([]Reification, len(cc.reified)),
	}
	copy(newC.constraints, cc.constraints)
	copy(newC.reified, cc.reified)
	return newC
}

// String provides a human-readable representation of the collected constraints.
func (cc *ConstraintCollector) String() string {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	return fmt.Sprintf("ConstraintCollector with %d constraints and %d reifications", len(cc.constraints), len(cc.reified))
}

type ConstraintCollector struct {
	constraints []Constraint
	reified     []Reification
	parent      ConstraintStore
	mu          sync.Mutex
}

// NewConstraintCollector creates a new, empty constraint collector
// that wraps the given parent store.
func NewConstraintCollector(parent ConstraintStore) *ConstraintCollector {
	return &ConstraintCollector{parent: parent}
}

// AddConstraint adds a constraint to the collector. For reified constraints,
// it records the reification mapping.
func (cc *ConstraintCollector) AddConstraint(constraint Constraint) error {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	if rc, ok := constraint.(*ReifiedConstraint); ok {
		cc.reified = append(cc.reified, Reification{
			Var: rc.Var,
			Val: rc.Val,
		})
		cc.constraints = append(cc.constraints, rc.Constraint)
	} else {
		cc.constraints = append(cc.constraints, constraint)
	}
	return nil
}

// AddConstraintDeferred is a no-op for the collector, as all constraints
// are deferred by nature in this context.
func (cc *ConstraintCollector) AddConstraintDeferred(constraint Constraint) error {
	return cc.AddConstraint(constraint)
}

// AddBinding delegates the binding to the parent store.
func (cc *ConstraintCollector) AddBinding(varID int64, term Term) error {
	return cc.parent.AddBinding(varID, term)
}

// GetBinding delegates the binding lookup to the parent store.
func (cc *ConstraintCollector) GetBinding(varID int64) Term {
	return cc.parent.GetBinding(varID)
}

// GetSubstitution delegates substitution retrieval to the parent store.
func (cc *ConstraintCollector) GetSubstitution() *Substitution {
	return cc.parent.GetSubstitution()
}

// GetConstraints returns the list of constraints collected so far.
func (cc *ConstraintCollector) GetConstraints() []Constraint {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	// Return a copy to be safe
	c := make([]Constraint, len(cc.constraints))
	copy(c, cc.constraints)
	return c
}

// GetReified returns the list of reification mappings collected so far.
func (cc *ConstraintCollector) GetReified() []Reification {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	// Return a copy to be safe
	r := make([]Reification, len(cc.reified))
	copy(r, cc.reified)
	return r
}

// Clone creates a shallow copy of the collector. The underlying constraint
// slices are copied, but the constraints themselves are not cloned.
func (cc *ConstraintCollector) Clone() ConstraintStore {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	newC := &ConstraintCollector{
		constraints: make([]Constraint, len(cc.constraints)),
		reified:     make([]Reification, len(cc.reified)),
		parent:      cc.parent.Clone(),
	}
	copy(newC.constraints, cc.constraints)
	copy(newC.reified, cc.reified)
	return newC
}

// String provides a human-readable representation of the collected constraints.
func (cc *ConstraintCollector) String() string {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	return fmt.Sprintf("ConstraintCollector with %d constraints and %d reifications, wrapping: %s", len(cc.constraints), len(cc.reified), cc.parent)
}
