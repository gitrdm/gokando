// Package minikanren provides reified set-membership for FD variables.
//
// InSetReified links an integer variable v and a boolean b (1=false, 2=true)
// such that b = 2 iff v ∈ S, where S is a fixed set of allowed values.
//
// Propagation is bidirectional and safe:
//   - If v's domain has no intersection with S → set b = 1
//   - If v is singleton in S → set b = 2
//   - If b = 2 → prune v to v∈S (intersect)
//   - If b = 1 → prune v to v∉S (remove all S values)
//
// This is used by higher-level globals like Sequence to create membership
// booleans over a fixed set without resorting to large per-value tables.
package minikanren

import "fmt"

type InSetReified struct {
	v       *FDVariable
	set     []int
	boolVar *FDVariable // domain subset of {1,2}
}

// NewInSetReified creates a reified membership constraint b ↔ (v ∈ setValues).
func NewInSetReified(v *FDVariable, setValues []int, boolVar *FDVariable) (*InSetReified, error) {
	if v == nil {
		return nil, fmt.Errorf("NewInSetReified: v cannot be nil")
	}
	if boolVar == nil {
		return nil, fmt.Errorf("NewInSetReified: boolVar cannot be nil")
	}
	if len(setValues) == 0 {
		return nil, fmt.Errorf("NewInSetReified: setValues cannot be empty")
	}
	// Defensive copy and dedupe small set
	seen := map[int]struct{}{}
	set := make([]int, 0, len(setValues))
	for _, val := range setValues {
		if val < 1 {
			return nil, fmt.Errorf("NewInSetReified: values must be positive, got %d", val)
		}
		if _, ok := seen[val]; !ok {
			seen[val] = struct{}{}
			set = append(set, val)
		}
	}
	return &InSetReified{v: v, set: set, boolVar: boolVar}, nil
}

func (c *InSetReified) Variables() []*FDVariable { return []*FDVariable{c.v, c.boolVar} }
func (c *InSetReified) Type() string             { return "InSetReified" }
func (c *InSetReified) String() string {
	return fmt.Sprintf("InSetReified(v=%d∈S -> b=%d)", c.v.ID(), c.boolVar.ID())
}

// Propagate enforces b ↔ (v ∈ S) with bidirectional pruning.
func (c *InSetReified) Propagate(solver *Solver, state *SolverState) (*SolverState, error) {
	if solver == nil {
		return nil, fmt.Errorf("InSetReified.Propagate: nil solver")
	}
	vDom := solver.GetDomain(state, c.v.ID())
	bDom := solver.GetDomain(state, c.boolVar.ID())
	if vDom == nil || vDom.Count() == 0 {
		return nil, fmt.Errorf("InSetReified.Propagate: v has empty domain")
	}
	if bDom == nil || bDom.Count() == 0 {
		return nil, fmt.Errorf("InSetReified.Propagate: b has empty domain")
	}
	has1 := bDom.Has(1)
	has2 := bDom.Has(2)
	if bDom.Count() > 2 || (!has1 && !has2) {
		return nil, fmt.Errorf("InSetReified.Propagate: b domain must be subset of {1,2}, got %s", bDom.String())
	}

	// Compute intersection and outside sets relative to vDom
	maxV := vDom.MaxValue()
	inVals := []int{}
	c.v.Domain().IterateValues(func(int) {}) // avoid unused method warning in some editors
	for _, val := range c.set {
		if val <= maxV && vDom.Has(val) {
			inVals = append(inVals, val)
		}
	}
	cur := state

	// If no intersection, b must be false
	if len(inVals) == 0 {
		if has2 {
			nd := bDom.Remove(2)
			cur, _ = solver.SetDomain(cur, c.boolVar.ID(), nd)
			bDom = nd
			has2 = false
		}
	}

	// If v singleton and in S, b must be true; if singleton and not in S, b must be false
	if vDom.IsSingleton() {
		v := vDom.SingletonValue()
		in := false
		for _, s := range c.set {
			if s == v {
				in = true
				break
			}
		}
		if in && has1 {
			nd := bDom.Remove(1)
			cur, _ = solver.SetDomain(cur, c.boolVar.ID(), nd)
			bDom = nd
			has1 = false
		}
		if !in && has2 {
			nd := bDom.Remove(2)
			cur, _ = solver.SetDomain(cur, c.boolVar.ID(), nd)
			bDom = nd
			has2 = false
		}
	}

	// Reflect from b to v
	has1 = bDom.Has(1)
	has2 = bDom.Has(2)
	if has2 && !has1 {
		// b = true ⇒ v ∈ S
		nd := NewBitSetDomainFromValues(maxV, inVals)
		if nd.Count() == 0 {
			return nil, fmt.Errorf("InSetReified: b=2 but v∈S is empty")
		}
		if !nd.Equal(vDom) {
			cur, _ = solver.SetDomain(cur, c.v.ID(), nd)
			vDom = nd
		}
	} else if has1 && !has2 {
		// b = false ⇒ v ∉ S
		nd := vDom
		for _, val := range c.set {
			if nd.Has(val) {
				nd = nd.Remove(val)
				if nd.Count() == 0 {
					return nil, fmt.Errorf("InSetReified: b=1 empties v by removing %d", val)
				}
			}
		}
		if !nd.Equal(vDom) {
			cur, _ = solver.SetDomain(cur, c.v.ID(), nd)
			vDom = nd
		}
	}

	return cur, nil
}
