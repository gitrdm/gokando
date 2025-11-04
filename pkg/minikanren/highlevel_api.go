package minikanren

// This file provides a thin, additive High-Level API (HLAPI) over the existing
// miniKanren + FD solver primitives. It reduces boilerplate for common tasks
// like running goals and building FD models, while delegating all heavy lifting
// to the production core. All functions are pure wrappers with literate docs.

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// A creates an Atom term from any Go value. Shorthand for NewAtom/AtomFromValue.
// Examples: A(1), A("hello"), A(true)
func A(value interface{}) Term { return NewAtom(value) }

// L builds a miniKanren list from values. Each element is converted to a Term:
// - Term values are used as-is
// - Other values are wrapped via A(...)
// Example: L(1, 2, 3) → (1 2 3)
func L(values ...interface{}) Term {
	terms := make([]Term, len(values))
	for i, v := range values {
		if t, ok := v.(Term); ok {
			terms[i] = t
		} else {
			terms[i] = A(v)
		}
	}
	return List(terms...)
}

// DomainRange returns a domain representing the inclusive range [min..max].
// If min <= 1, this is equivalent to NewBitSetDomain(max). For min>1, values
// outside the range are removed in one bulk operation. Empty ranges return an
// empty domain.
func DomainRange(min, max int) Domain {
	if max <= 0 || min > max {
		return NewBitSetDomain(0)
	}
	if min <= 1 {
		return NewBitSetDomain(max)
	}
	// Build base domain [1..max], then remove below min.
	return NewBitSetDomain(max).RemoveBelow(min)
}

// DomainValues returns a domain containing only the provided values. Values
// out of range are ignored. Empty input yields an empty domain.
func DomainValues(vals ...int) Domain {
	if len(vals) == 0 {
		return NewBitSetDomain(0)
	}
	// Compute max to size the domain efficiently
	max := 0
	for _, v := range vals {
		if v > max {
			max = v
		}
	}
	if max <= 0 {
		return NewBitSetDomain(0)
	}
	return NewBitSetDomainFromValues(max, vals)
}

// IntVar creates a new FD variable with integer domain [min..max]. If name is
// non-empty a named variable is created (useful in debugging and formatted output).
func (m *Model) IntVar(min, max int, name string) *FDVariable {
	d := DomainRange(min, max)
	if name != "" {
		return m.NewVariableWithName(d, name)
	}
	return m.NewVariable(d)
}

// IntVars creates count FD variables with domain [min..max]. If baseName is
// non-empty, variables are named baseName1, baseName2, ... baseNameN; otherwise
// anonymous variables are created.
func (m *Model) IntVars(count, min, max int, baseName string) []*FDVariable {
	if count <= 0 {
		return nil
	}
	d := DomainRange(min, max)
	if baseName == "" {
		return m.NewVariables(count, d)
	}
	names := make([]string, count)
	for i := 0; i < count; i++ {
		names[i] = fmt.Sprintf("%s%d", baseName, i+1)
	}
	return m.NewVariablesWithNames(names, d)
}

// IntVarsWithNames creates FD variables with domain [min..max] using the given
// names. Handy for small models that benefit from explicit names.
func (m *Model) IntVarsWithNames(names []string, min, max int) []*FDVariable {
	d := DomainRange(min, max)
	return m.NewVariablesWithNames(names, d)
}

// IntVarValues creates a new FD variable whose domain is exactly the provided
// non-contiguous set of values. If name is non-empty, the variable is named.
// Duplicate values are ignored. Empty or all non-positive values yield an
// empty domain which will cause the model to be immediately infeasible.
func (m *Model) IntVarValues(values []int, name string) *FDVariable {
	d := DomainValues(values...)
	if name != "" {
		return m.NewVariableWithName(d, name)
	}
	return m.NewVariable(d)
}

// AllDifferent posts an AllDifferent constraint over vars.
func (m *Model) AllDifferent(vars ...*FDVariable) error {
	if len(vars) == 0 {
		return fmt.Errorf("AllDifferent: need at least one variable")
	}
	c, err := NewAllDifferent(vars)
	if err != nil {
		return err
	}
	m.AddConstraint(c)
	return nil
}

// LinearSum posts Σ coeffs[i]*vars[i] = total, using bounds-consistent propagation.
func (m *Model) LinearSum(vars []*FDVariable, coeffs []int, total *FDVariable) error {
	c, err := NewLinearSum(vars, coeffs, total)
	if err != nil {
		return err
	}
	m.AddConstraint(c)
	return nil
}

// Cumulative posts a Cumulative(starts, durations, demands, capacity) global
// constraint to the model. See NewCumulative for contract and semantics.
func (m *Model) Cumulative(starts []*FDVariable, durations, demands []int, capacity int) error {
	c, err := NewCumulative(starts, durations, demands, capacity)
	if err != nil {
		return err
	}
	m.AddConstraint(c)
	return nil
}

// NoOverlap posts a NoOverlap(starts, durations) global constraint to the model.
// This is modeled via Cumulative with unit demands and capacity 1.
func (m *Model) NoOverlap(starts []*FDVariable, durations []int) error {
	c, err := NewNoOverlap(starts, durations)
	if err != nil {
		return err
	}
	m.AddConstraint(c)
	return nil
}

// GlobalCardinality posts a GCC over vars with per-value min/max occurrence bounds.
// See NewGlobalCardinality for requirements regarding slice lengths and indexing.
func (m *Model) GlobalCardinality(vars []*FDVariable, minCount, maxCount []int) error {
	c, err := NewGlobalCardinality(vars, minCount, maxCount)
	if err != nil {
		return err
	}
	m.AddConstraint(c)
	return nil
}

// LexLess posts a strict lexicographic ordering constraint X < Y.
func (m *Model) LexLess(xs, ys []*FDVariable) error {
	c, err := NewLexLess(xs, ys)
	if err != nil {
		return err
	}
	m.AddConstraint(c)
	return nil
}

// LexLessEq posts a non-strict lexicographic ordering X <= Y.
func (m *Model) LexLessEq(xs, ys []*FDVariable) error {
	c, err := NewLexLessEq(xs, ys)
	if err != nil {
		return err
	}
	m.AddConstraint(c)
	return nil
}

// Regular posts a Regular(vars, numStates, start, acceptStates, delta) DFA constraint.
func (m *Model) Regular(vars []*FDVariable, numStates, start int, acceptStates []int, delta [][]int) error {
	c, err := NewRegular(vars, numStates, start, acceptStates, delta)
	if err != nil {
		return err
	}
	m.AddConstraint(c)
	return nil
}

// Table posts an extensional Table constraint over the given variables and rows.
func (m *Model) Table(vars []*FDVariable, rows [][]int) error {
	c, err := NewTable(vars, rows)
	if err != nil {
		return err
	}
	m.AddConstraint(c)
	return nil
}

// SolveN solves the model and returns up to maxSolutions solutions using the
// default sequential solver. For advanced control, use NewSolver(m) directly.
func SolveN(ctx context.Context, m *Model, maxSolutions int) ([][]int, error) {
	solver := NewSolver(m)
	return solver.Solve(ctx, maxSolutions)
}

// Solve is SolveN with context.Background().
func Solve(m *Model, maxSolutions int) ([][]int, error) {
	return SolveN(context.Background(), m, maxSolutions)
}

// Optimize finds a solution that optimizes the objective variable.
// It is a thin wrapper around Solver.SolveOptimal with context.Background().
//
// Parameters:
//   - obj: FD variable representing the objective (e.g., a LinearSum total)
//   - minimize: true to minimize, false to maximize
//
// Returns the best solution (values for all model variables) and the best
// objective value. Errors mirror the underlying solver behavior.
func Optimize(m *Model, obj *FDVariable, minimize bool) ([]int, int, error) {
	solver := NewSolver(m)
	return solver.SolveOptimal(context.Background(), obj, minimize)
}

// OptimizeWithOptions is like Optimize but accepts a context and solver options
// for time/node limits or parallel workers. See WithParallelWorkers, WithNodeLimit,
// and other OptimizeOption helpers.
func OptimizeWithOptions(ctx context.Context, m *Model, obj *FDVariable, minimize bool, opts ...OptimizeOption) ([]int, int, error) {
	solver := NewSolver(m)
	return solver.SolveOptimalWithOptions(ctx, obj, minimize, opts...)
}

// SolutionsN runs a goal against a fresh local store and returns up to n
// solutions projected onto the provided variables. Each solution is a map from
// variable name to the reified value term. If no vars are provided, an empty
// string key is used for each result to preserve cardinality.
func SolutionsN(ctx context.Context, n int, goal Goal, vars ...*Var) []map[string]Term {
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)

	// Collect results in batches until we have n or stream closes.
	results := make([]map[string]Term, 0)
	for len(results) < n || n <= 0 {
		batchSize := 10
		if n > 0 {
			rem := n - len(results)
			if rem < batchSize {
				batchSize = rem
			}
		}
		rs, more := stream.Take(batchSize)
		for _, st := range rs {
			entry := make(map[string]Term, len(vars))
			if len(vars) == 0 {
				entry[""] = st.GetSubstitution().DeepWalk(NewAtom(nil))
			} else {
				for _, v := range vars {
					name := friendlyVarName(v)
					entry[name] = st.GetSubstitution().DeepWalk(v)
				}
			}
			results = append(results, entry)
			if n > 0 && len(results) >= n {
				break
			}
		}
		if !more || (n > 0 && len(results) >= n) {
			break
		}
	}
	return results
}

// Solutions is SolutionsN with n<=0 (all results). WARNING: may not terminate
// on goals with infinite streams.
func Solutions(goal Goal, vars ...*Var) []map[string]Term {
	return SolutionsN(context.Background(), 0, goal, vars...)
}

// SolutionsCtx is an alias for SolutionsN that improves discoverability when
// passing an explicit context and solution cap together.
// It returns up to n solutions (n<=0 for all solutions, which may not terminate).
func SolutionsCtx(ctx context.Context, n int, goal Goal, vars ...*Var) []map[string]Term {
	return SolutionsN(ctx, n, goal, vars...)
}

// RowsN runs a goal and returns up to n rows of reified Terms projected in the
// order of vars provided. Each row corresponds to one solution. If no vars are
// provided, each row contains a single Atom(nil) to preserve cardinality. When
// n<=0, all solutions are returned (which may not terminate for infinite goals).
func RowsN(ctx context.Context, n int, goal Goal, vars ...*Var) [][]Term {
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)

	rows := make([][]Term, 0)
	for len(rows) < n || n <= 0 {
		batchSize := 10
		if n > 0 {
			rem := n - len(rows)
			if rem < batchSize {
				batchSize = rem
			}
		}
		rs, more := stream.Take(batchSize)
		for _, st := range rs {
			if len(vars) == 0 {
				rows = append(rows, []Term{st.GetSubstitution().DeepWalk(NewAtom(nil))})
				continue
			}
			row := make([]Term, len(vars))
			for i, v := range vars {
				row[i] = st.GetSubstitution().DeepWalk(v)
			}
			rows = append(rows, row)
			if n > 0 && len(rows) >= n {
				break
			}
		}
		if !more || (n > 0 && len(rows) >= n) {
			break
		}
	}
	return rows
}

// Rows is RowsN with n<=0 (all results). WARNING: may not terminate on goals
// with infinite streams.
func Rows(goal Goal, vars ...*Var) [][]Term { return RowsN(context.Background(), 0, goal, vars...) }

// IntsN solves for a single variable and returns up to n integer values.
// Non-int bindings are skipped. When n<=0, all results are returned.
func IntsN(ctx context.Context, n int, goal Goal, v *Var) []int {
	rows := RowsN(ctx, n, goal, v)
	out := make([]int, 0, len(rows))
	for _, r := range rows {
		if len(r) == 1 {
			if iv, ok := AsInt(r[0]); ok {
				out = append(out, iv)
			}
		}
	}
	return out
}

// Ints is IntsN with n<=0 (all results).
func Ints(goal Goal, v *Var) []int { return IntsN(context.Background(), 0, goal, v) }

// StringsN solves for a single variable and returns up to n string values.
// Non-string bindings are skipped. When n<=0, all results are returned.
func StringsN(ctx context.Context, n int, goal Goal, v *Var) []string {
	rows := RowsN(ctx, n, goal, v)
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		if len(r) == 1 {
			if sv, ok := AsString(r[0]); ok {
				out = append(out, sv)
			}
		}
	}
	return out
}

// Strings is StringsN with n<=0 (all results).
func Strings(goal Goal, v *Var) []string { return StringsN(context.Background(), 0, goal, v) }

// FormatSolutions pretty-prints a slice of solutions for human-friendly output.
// Each solution is rendered as "name: value, name2: value2" with lists and strings
// formatted pleasantly. Output is sorted for stable tests.
func FormatSolutions(solutions []map[string]Term) []string {
	out := make([]string, 0, len(solutions))
	for _, sol := range solutions {
		// Stable order by variable name
		names := make([]string, 0, len(sol))
		for k := range sol {
			names = append(names, k)
		}
		sort.Strings(names)
		parts := make([]string, 0, len(names))
		for _, name := range names {
			parts = append(parts, fmt.Sprintf("%s: %s", nameOrQ(name), pretty(sol[name])))
		}
		out = append(out, strings.Join(parts, ", "))
	}
	sort.Strings(out)
	return out
}

// friendlyVarName extracts the user-provided name from a Var if present; falls
// back to the full Var string (e.g., _q_13) and ultimately to "q".
func friendlyVarName(v *Var) string {
	if v == nil {
		return "q"
	}
	s := v.String() // "_name_id" or "_id"
	if strings.HasPrefix(s, "_") {
		segs := strings.Split(s, "_")
		if len(segs) >= 3 && segs[1] != "" {
			return segs[1]
		}
	}
	return "q"
}

func nameOrQ(name string) string {
	if name == "" {
		return "q"
	}
	return name
}

// pretty renders a Term in a compact, friendly format:
// - Empty list as ()
// - Proper lists as (a b c)
// - Improper lists as (a b . tail)
// - Strings quoted
// - Other atoms via fmt %v
func pretty(t Term) string {
	// Empty list: Atom(nil)
	if a, ok := t.(*Atom); ok {
		if a.Value() == nil {
			return "()"
		}
		switch v := a.Value().(type) {
		case string:
			return fmt.Sprintf("%q", v)
		default:
			return fmt.Sprintf("%v", v)
		}
	}

	// Pairs: proper or improper list
	if p, ok := t.(*Pair); ok {
		elems := []string{}
		tail := Term(p)
		for {
			pr, ok := tail.(*Pair)
			if !ok {
				break
			}
			elems = append(elems, pretty(pr.Car()))
			tail = pr.Cdr()
		}
		if a, ok := tail.(*Atom); ok && a.Value() == nil {
			return "(" + strings.Join(elems, " ") + ")"
		}
		return "(" + strings.Join(elems, " ") + " . " + pretty(tail) + ")"
	}
	return t.String()
}

// FormatTerm returns the canonical human-friendly string for a reified Term.
// It mirrors the formatting used by FormatSolutions:
// - Empty list: ()
// - Proper lists: (a b c)
// - Improper lists: (a b . tail)
// - Strings quoted; other atoms via fmt %%v
func FormatTerm(t Term) string { return pretty(t) }

// AsInt attempts to extract an int from a reified Term (Atom). Returns false on mismatch.
func AsInt(t Term) (int, bool) {
	if a, ok := t.(*Atom); ok {
		if v, ok2 := a.Value().(int); ok2 {
			return v, true
		}
	}
	return 0, false
}

// MustInt extracts an int from a Term or panics. Intended for examples/tests.
func MustInt(t Term) int {
	if v, ok := AsInt(t); ok {
		return v
	}
	panic(fmt.Sprintf("expected int Atom, got %T: %v", t, t))
}

// AsString attempts to extract a string from a reified Term (Atom).
func AsString(t Term) (string, bool) {
	if a, ok := t.(*Atom); ok {
		if v, ok2 := a.Value().(string); ok2 {
			return v, true
		}
	}
	return "", false
}

// MustString extracts a string from a Term or panics.
func MustString(t Term) string {
	if v, ok := AsString(t); ok {
		return v
	}
	panic(fmt.Sprintf("expected string Atom, got %T: %v", t, t))
}

// AsList collects a proper Scheme-like list into a Go slice of Terms.
// Returns false for non-list or improper lists.
func AsList(t Term) ([]Term, bool) {
	if a, ok := t.(*Atom); ok && a.Value() == nil {
		return []Term{}, true
	}
	elems := []Term{}
	cur := t
	for {
		p, ok := cur.(*Pair)
		if !ok {
			// must end with empty list to be proper
			if a, ok := cur.(*Atom); ok && a.Value() == nil {
				return elems, true
			}
			return nil, false
		}
		elems = append(elems, p.Car())
		cur = p.Cdr()
	}
}

// ValuesInt projects a named value from Solutions(...) into a slice of ints.
// Missing or non-int entries are skipped.
func ValuesInt(results []map[string]Term, name string) []int {
	out := make([]int, 0, len(results))
	for _, r := range results {
		if t, ok := r[name]; ok {
			if v, ok2 := AsInt(t); ok2 {
				out = append(out, v)
			}
		}
	}
	return out
}

// ValuesString projects a named value from Solutions(...) into a slice of strings.
// Missing or non-string entries are skipped.
func ValuesString(results []map[string]Term, name string) []string {
	out := make([]string, 0, len(results))
	for _, r := range results {
		if t, ok := r[name]; ok {
			if v, ok2 := AsString(t); ok2 {
				out = append(out, v)
			}
		}
	}
	return out
}
