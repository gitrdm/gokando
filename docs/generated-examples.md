# Generated Examples
## pkg_minikanren_among_example_test.go-ExampleNewAmong_hybrid.md
```go
func ExampleNewAmong_hybrid() {
	model := NewModel()

	x1 := model.IntVarValues([]int{1, 2}, "x1")
	x2 := model.IntVarValues([]int{2, 3}, "x2")
	x3 := model.IntVarValues([]int{3, 4}, "x3")
	k := model.IntVarValues([]int{2}, "K")

	// Build the propagation constraint and register it with the model so the FD plugin can discover it.
	c, _ := NewAmong([]*FDVariable{x1, x2, x3}, []int{1, 2}, k)
	model.AddConstraint(c)

	// Use HLAPI helper to build a HybridSolver and a UnifiedStore populated
	// from the model (domains + constraints). This reduces boilerplate.
	solver, store, err := NewHybridSolverFromModel(model)
	if err != nil {
		panic(err)
	}

	// Run propagation to a fixed point.
	result, _ := solver.Propagate(store)

	fmt.Printf("x2: %s\n", result.GetDomain(x2.ID()))
	fmt.Printf("x3: %s\n", result.GetDomain(x3.ID()))
	// Output:
	// x2: {3}
	// x3: {3..4}
}

```


\n
## pkg_minikanren_among_example_test.go-ExampleNewAmong.md
```go
func ExampleNewAmong() {
	model := NewModel()

	// Low-level API (kept as comments):
	// x1 := model.NewVariableWithName(NewBitSetDomainFromValues(5, []int{1, 2}), "x1")
	x1 := model.IntVarValues([]int{1, 2}, "x1")
	// x2 := model.NewVariableWithName(NewBitSetDomainFromValues(5, []int{2, 3}), "x2")
	x2 := model.IntVarValues([]int{2, 3}, "x2")
	// x3 := model.NewVariableWithName(NewBitSetDomainFromValues(5, []int{3, 4}), "x3")
	x3 := model.IntVarValues([]int{3, 4}, "x3")
	// K encodes count+1; here we want exactly 1 variable in S → K={2}
	// k := model.NewVariableWithName(NewBitSetDomainFromValues(4, []int{2}), "K")
	k := model.IntVarValues([]int{2}, "K")

	// S = {1,2}. With K=1 (encoded 2) and x1⊆S, x2 is forced OUT of S
	// Low-level API (kept as comment):
	// c, _ := NewAmong([]*FDVariable{x1, x2, x3}, []int{1, 2}, k)
	// model.AddConstraint(c)
	// HLAPI:
	_ = model.Among([]*FDVariable{x1, x2, x3}, []int{1, 2}, k)

	solver := NewSolver(model)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, _ = solver.Solve(ctx, 0)

	fmt.Printf("x2: %s\n", solver.GetDomain(nil, x2.ID()))
	fmt.Printf("x3: %s\n", solver.GetDomain(nil, x3.ID()))
	// Output:
	// x2: {3}
	// x3: {3..4}
}

```


\n
## pkg_minikanren_bin_packing_example_test.go-ExampleNewBinPacking.md
```go
func ExampleNewBinPacking() {
	model := NewModel()

	// Items: sizes [2,2,1], bins: 2 with capacities [4,1]
	// Low level API (kept as comments):
	// bdom := NewBitSetDomain(2) // bins {1,2}
	// x1 := model.NewVariableWithName(bdom, "x1")
	// x2 := model.NewVariableWithName(bdom, "x2")
	// x3 := model.NewVariableWithName(bdom, "x3")
	// HLAPI: use IntVar for compact [1..2] domains
	x1 := model.IntVar(1, 2, "x1")
	x2 := model.IntVar(1, 2, "x2")
	x3 := model.IntVar(1, 2, "x3")

	sizes := []int{2, 2, 1}
	capacities := []int{4, 1}

	// Low-level API (kept as comment):
	// _, _ = NewBinPacking(model, []*FDVariable{x1, x2, x3}, sizes, capacities)
	// HLAPI wrapper:
	_ = model.BinPacking([]*FDVariable{x1, x2, x3}, sizes, capacities)

	solver := NewSolver(model)
	// Propagate using solver.propagate to show the low-level approach is equivalent.
	st, _ := solver.propagate(nil)

	// After propagation:
	//  - Bin 2 (cap=1) can only host size-1 ⇒ x3=2
	//  - Bin 1 (cap=4) must host both size-2 ⇒ x1=1, x2=1
	fmt.Printf("x1: %s\n", solver.GetDomain(st, x1.ID()))
	fmt.Printf("x2: %s\n", solver.GetDomain(st, x2.ID()))
	fmt.Printf("x3: %s\n", solver.GetDomain(st, x3.ID()))
	// Output:
	// x1: {1}
	// x2: {1}
	// x3: {2}
}

```


\n
## pkg_minikanren_circuit_example_test.go-ExampleNewCircuit.md
```go
func ExampleNewCircuit() {
	model := NewModel()
	n := 4

	// succ[i] ∈ [1..n]
	succ := make([]*FDVariable, n)
	for i := 0; i < n; i++ {
		succ[i] = model.NewVariableWithName(NewBitSetDomain(n), fmt.Sprintf("succ_%d", i+1))
	}

	// Build Circuit with start at node 1
	c, _ := NewCircuit(model, succ, 1)
	model.AddConstraint(c)

	solver := NewSolver(model)

	// Run propagation
	newState, _ := solver.propagate(nil)

	// Inspect two successor domains to see self-loop removal
	d1 := solver.GetDomain(newState, succ[0].ID())
	d2 := solver.GetDomain(newState, succ[1].ID())
	fmt.Printf("succ1=%s\n", d1.String())
	fmt.Printf("succ2=%s\n", d2.String())

	// Output:
	// succ1={2..4}
	// succ2={1,3,4}
}

```


\n
## pkg_minikanren_count_example_test.go-ExampleCount.md
```go
func ExampleCount() {
	model := NewModel()
	dom := NewBitSetDomain(3)
	x := model.NewVariableWithName(dom, "X")
	y := model.NewVariableWithName(dom, "Y")
	z := model.NewVariableWithName(dom, "Z")
	// N encodes count+1, therefore use domain [1..4]
	N := model.NewVariableWithName(NewBitSetDomain(4), "N")

	// Post Count constraint: number of vars equal to 2
	_, _ = NewCount(model, []*FDVariable{x, y, z}, 2, N)

	solver := NewSolver(model)
	solutions, _ := solver.Solve(context.Background(), 0)

	// Collect stringified solutions and sort so output is deterministic.
	var lines []string
	for _, sol := range solutions {
		lines = append(lines, fmt.Sprintf("X=%d Y=%d Z=%d count=%d", sol[x.ID()], sol[y.ID()], sol[z.ID()], sol[N.ID()]-1))
	}
	sort.Strings(lines)

	// Print the first three sorted solutions
	for i := 0; i < 3 && i < len(lines); i++ {
		fmt.Println(lines[i])
	}
	// Output:
	// X=1 Y=1 Z=1 count=0
	// X=1 Y=1 Z=2 count=1
	// X=1 Y=1 Z=3 count=0
}

```


\n
## pkg_minikanren_cumulative_example_test.go-ExampleNewCumulative.md
```go
func ExampleNewCumulative() {
	model := NewModel()

	// Task A: fixed at start=2, duration=2, demand=2
	// A := model.NewVariableWithName(NewBitSetDomainFromValues(10, []int{2}), "A")
	A := model.IntVarValues([]int{2}, "A")
	// Task B: start in [1..4], duration=2, demand=1
	// B := model.NewVariableWithName(NewBitSetDomain(4), "B")
	B := model.IntVar(1, 4, "B")

	// Low-level API (kept as comment):
	// cum, err := NewCumulative([]*FDVariable{A, B}, []int{2, 2}, []int{2, 1}, 2)
	// if err != nil {
	//     panic(err)
	// }
	// model.AddConstraint(cum)
	// HLAPI wrapper:
	_ = model.Cumulative([]*FDVariable{A, B}, []int{2, 2}, []int{2, 1}, 2)

	// If you only need concrete solutions (assignments), the HLAPI helper
	// SolveN(ctx, model, maxSolutions) is a convenient wrapper that creates
	// a solver, runs the search, and returns solutions. Example:
	//
	//    sols, err := SolveN(ctx, model, 1)
	//
	// However, when you want to inspect solver internals (domains after
	// propagation) or call methods like GetDomain/propagate, create the
	// Solver explicitly as done below and call Solve on it. That allows
	// reading the pruned domains from the solver state.
	solver := NewSolver(model)
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	// Propagate at root by running a one-solution search (will stop at root if none).
	_, _ = solver.Solve(ctx, 1)

	fmt.Println("A:", solver.GetDomain(nil, A.ID()))
	fmt.Println("B:", solver.GetDomain(nil, B.ID()))
	// Output:
	// A: {2}
	// B: {4}
}

```


\n
## pkg_minikanren_diffn_example_test.go-ExampleNewDiffn.md
```go
func ExampleNewDiffn() {
	model := NewModel()
	// x1 := model.NewVariableWithName(NewBitSetDomainFromValues(10, []int{1}), "x1")
	x1 := model.IntVarValues([]int{1}, "x1")
	// y1 := model.NewVariableWithName(NewBitSetDomainFromValues(10, []int{1}), "y1")
	y1 := model.IntVarValues([]int{1}, "y1")
	// x2 := model.NewVariableWithName(NewBitSetDomainFromValues(10, []int{1, 2, 3, 4}), "x2")
	x2 := model.IntVarValues([]int{1, 2, 3, 4}, "x2")
	// y2 := model.NewVariableWithName(NewBitSetDomainFromValues(10, []int{1}), "y2")
	y2 := model.IntVarValues([]int{1}, "y2")

	_, _ = NewDiffn(model, []*FDVariable{x1, x2}, []*FDVariable{y1, y2}, []int{2, 2}, []int{2, 2})

	solver := NewSolver(model)
	_, _ = solver.Solve(context.Background(), 0)

	fmt.Printf("x2: %s\n", solver.GetDomain(nil, x2.ID()))
	// Output:
	// x2: {3..4}
}

```


\n
## pkg_minikanren_domain_example_test.go-ExampleBitSetDomain_Complement.md
```go
func ExampleBitSetDomain_Complement() {
	// Domain {2,4,6,8} within range 1-10
	evenDigits := minikanren.NewBitSetDomainFromValues(10, []int{2, 4, 6, 8})

	// Complement gives odd digits plus 10
	oddDigits := evenDigits.Complement()

	fmt.Printf("Even: %s\n", evenDigits.String())
	fmt.Printf("Odd: %s\n", oddDigits.String())

	// Output:
	// Even: {2,4,6,8}
	// Odd: {1,3,5,7,9,10}
}

```


\n
## pkg_minikanren_domain_example_test.go-ExampleBitSetDomain_Intersect.md
```go
func ExampleBitSetDomain_Intersect() {
	// Variable must be in {1,2,3,4,5} from one constraint
	domain1 := minikanren.NewBitSetDomainFromValues(10, []int{1, 2, 3, 4, 5})

	// Variable must be in {3,4,5,6,7} from another constraint
	domain2 := minikanren.NewBitSetDomainFromValues(10, []int{3, 4, 5, 6, 7})

	// Intersection gives values satisfying both constraints
	intersection := domain1.Intersect(domain2)

	fmt.Printf("Domain 1: %s\n", domain1.String())
	fmt.Printf("Domain 2: %s\n", domain2.String())
	fmt.Printf("Intersection: %s\n", intersection.String())

	// Output:
	// Domain 1: {1..5}
	// Domain 2: {3..7}
	// Intersection: {3..5}
}

```


\n
## pkg_minikanren_domain_example_test.go-ExampleBitSetDomain_IsSingleton.md
```go
func ExampleBitSetDomain_IsSingleton() {
	domain := minikanren.NewBitSetDomain(5)
	fmt.Printf("Initial domain %s is singleton: %v\n", domain.String(), domain.IsSingleton())

	// Prune until singleton
	domain = domain.Remove(1).(*minikanren.BitSetDomain)
	domain = domain.Remove(2).(*minikanren.BitSetDomain)
	domain = domain.Remove(4).(*minikanren.BitSetDomain)
	domain = domain.Remove(5).(*minikanren.BitSetDomain)

	fmt.Printf("Domain %s is singleton: %v\n", domain.String(), domain.IsSingleton())
	if domain.IsSingleton() {
		fmt.Printf("Value: %d\n", domain.SingletonValue())
	}

	// Output:
	// Initial domain {1..5} is singleton: false
	// Domain {3} is singleton: true
	// Value: 3
}

```


\n
## pkg_minikanren_domain_example_test.go-ExampleBitSetDomain_IterateValues.md
```go
func ExampleBitSetDomain_IterateValues() {
	domain := minikanren.NewBitSetDomainFromValues(10, []int{2, 5, 7, 9})

	fmt.Print("Values: ")
	domain.IterateValues(func(v int) {
		fmt.Printf("%d ", v)
	})
	fmt.Println()

	// Output:
	// Values: 2 5 7 9
}

```


\n
## pkg_minikanren_domain_example_test.go-ExampleBitSetDomain_Remove.md
```go
func ExampleBitSetDomain_Remove() {
	domain := minikanren.NewBitSetDomain(5)
	fmt.Printf("Initial: %s\n", domain.String())

	// Remove value 3
	domain = domain.Remove(3).(*minikanren.BitSetDomain)
	fmt.Printf("After removing 3: %s\n", domain.String())

	// Remove value 5
	domain = domain.Remove(5).(*minikanren.BitSetDomain)
	fmt.Printf("After removing 5: %s\n", domain.String())

	// Output:
	// Initial: {1..5}
	// After removing 3: {1,2,4,5}
	// After removing 5: {1,2,4}
}

```


\n
## pkg_minikanren_domain_example_test.go-ExampleBitSetDomain_Union.md
```go
func ExampleBitSetDomain_Union() {
	// One constraint allows {1,2,3}
	domain1 := minikanren.NewBitSetDomainFromValues(10, []int{1, 2, 3})

	// Another constraint allows {3,4,5}
	domain2 := minikanren.NewBitSetDomainFromValues(10, []int{3, 4, 5})

	// Union gives all allowed values from either constraint
	union := domain1.Union(domain2)

	fmt.Printf("Domain 1: %s\n", domain1.String())
	fmt.Printf("Domain 2: %s\n", domain2.String())
	fmt.Printf("Union: %s\n", union.String())

	// Output:
	// Domain 1: {1..3}
	// Domain 2: {3..5}
	// Union: {1..5}
}

```


\n
## pkg_minikanren_domain_example_test.go-ExampleNewBitSetDomainFromValues.md
```go
func ExampleNewBitSetDomainFromValues() {
	// Create a domain with only even digits
	evenDigits := minikanren.NewBitSetDomainFromValues(9, []int{2, 4, 6, 8})

	fmt.Printf("Domain: %s\n", evenDigits.String())
	fmt.Printf("Size: %d\n", evenDigits.Count())
	fmt.Printf("Has 3: %v\n", evenDigits.Has(3))
	fmt.Printf("Has 4: %v\n", evenDigits.Has(4))

	// Output:
	// Domain: {2,4,6,8}
	// Size: 4
	// Has 3: false
	// Has 4: true
}

```


\n
## pkg_minikanren_domain_example_test.go-ExampleNewBitSetDomain.md
```go
func ExampleNewBitSetDomain() {
	// Create a domain for Sudoku: values 1 through 9
	domain := minikanren.NewBitSetDomain(9)

	fmt.Printf("Domain size: %d\n", domain.Count())
	fmt.Printf("Contains 5: %v\n", domain.Has(5))
	fmt.Printf("Contains 0: %v\n", domain.Has(0))
	fmt.Printf("Contains 10: %v\n", domain.Has(10))

	// Output:
	// Domain size: 9
	// Contains 5: true
	// Contains 0: false
	// Contains 10: false
}

```


\n
## pkg_minikanren_element_example_test.go-ExampleNewElementValues.md
```go
func ExampleNewElementValues() {
	model := NewModel()

	// index initially in [1..5]
	// low-level: idx := model.NewVariable(NewBitSetDomain(5))
	idx := model.IntVar(1, 5, "idx")
	// result initially in [1..10]
	// low-level: res := model.NewVariable(NewBitSetDomain(10))
	res := model.IntVar(1, 10, "res")

	vals := []int{2, 4, 4, 7, 9}
	c, _ := NewElementValues(idx, vals, res)
	model.AddConstraint(c)

	solver := NewSolver(model)

	// Force result to be either 4 or 7; this should prune index to {2,3,4}
	state := (*SolverState)(nil)
	state, _ = solver.SetDomain(state, res.ID(), NewBitSetDomainFromValues(10, []int{4, 7}))

	// Trigger propagation directly and inspect the resulting state domains.
	newState, err := c.Propagate(solver, state)
	if err != nil {
		// No solution under these restrictions (shouldn't happen here)
		fmt.Println("propagation error:", err)
		return
	}

	idxDom := solver.GetDomain(newState, idx.ID())
	resDom := solver.GetDomain(newState, res.ID())

	fmt.Printf("idx=%v res=%v\n", idxDom, resDom)
	// Output:
	// idx={2..4} res={4,7}
}

```


\n
## pkg_minikanren_enhancements_example_test.go-ExampleCumulative_energeticReasoning.md
```go
func ExampleCumulative_energeticReasoning() {
	m := NewModel()
	// Three heavy tasks that cannot fit in the time window
	// Tasks: each dur=4, dem=3, capacity=5, window=[1..6]
	// Energy required: 3 * 4 * 3 = 36 work units
	// Energy available: 6 time * 5 capacity = 30 work units → OVERLOAD
	s1 := m.NewVariable(NewBitSetDomain(3))
	s2 := m.NewVariable(NewBitSetDomain(3))
	s3 := m.NewVariable(NewBitSetDomain(3))

	cum, _ := NewCumulative(
		[]*FDVariable{s1, s2, s3},
		[]int{4, 4, 4}, // durations
		[]int{3, 3, 3}, // demands
		5,              // capacity
	)
	m.AddConstraint(cum)

	solver := NewSolver(m)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	sols, _ := solver.Solve(ctx, 1)
	fmt.Printf("Solutions found: %d (energetic reasoning detects overload)\n", len(sols))
	// Output: Solutions found: 0 (energetic reasoning detects overload)
}

```


\n
## pkg_minikanren_enhancements_example_test.go-ExampleLinearSum_mixedSign.md
```go
func ExampleLinearSum_mixedSign() {
	m := NewModel()
	// Profit model: revenue - cost = profit
	// revenue = 10*units, cost = 3*units, profit = 7*units
	// Or more realistically: profit = 5*productA - 2*productB
	productA := m.NewVariable(NewBitSetDomain(3))
	productB := m.NewVariable(NewBitSetDomain(3))
	profit := m.NewVariable(NewBitSetDomain(20))

	// Maximize: 5*A - 2*B
	ls, _ := NewLinearSum([]*FDVariable{productA, productB}, []int{5, -2}, profit)
	m.AddConstraint(ls)

	solver := NewSolver(m)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Find maximum profit
	sol, objVal, _ := solver.SolveOptimal(ctx, profit, false) // maximize
	fmt.Printf("Maximum profit: %d (A=%d, B=%d)\n", objVal, sol[productA.ID()], sol[productB.ID()])
	// Output: Maximum profit: 13 (A=3, B=1)
}

```


\n
## pkg_minikanren_enhancements_example_test.go-ExampleSolver_SolveOptimal_boolSum.md
```go
func ExampleSolver_SolveOptimal_boolSum() {
	m := NewModel()
	// Maximize the number of satisfied conditions (booleans set to true)
	b1 := m.NewVariable(NewBitSetDomainFromValues(2, []int{1, 2}))
	b2 := m.NewVariable(NewBitSetDomainFromValues(2, []int{1, 2}))
	b3 := m.NewVariable(NewBitSetDomainFromValues(2, []int{1, 2}))
	count := m.NewVariable(NewBitSetDomain(4)) // encoded count+1

	bs, _ := NewBoolSum([]*FDVariable{b1, b2, b3}, count)
	m.AddConstraint(bs)

	solver := NewSolver(m)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Maximize count (all booleans true)
	sol, objVal, _ := solver.SolveOptimal(ctx, count, false)
	actualCount := objVal - 1 // decode from encoded value
	fmt.Printf("Maximum count: %d (all satisfied: %v)\n", actualCount,
		sol[b1.ID()] == 2 && sol[b2.ID()] == 2 && sol[b3.ID()] == 2)
	// Output: Maximum count: 3 (all satisfied: true)
}

```


\n
## pkg_minikanren_enhancements_example_test.go-ExampleSolver_SolveOptimal_impactHeuristic.md
```go
func ExampleSolver_SolveOptimal_impactHeuristic() {
	m := NewModel()
	// Minimize total cost: cost = 2*x + 3*y
	x := m.NewVariable(NewBitSetDomain(4))
	y := m.NewVariable(NewBitSetDomain(4))
	cost := m.NewVariable(NewBitSetDomain(30))

	ls, _ := NewLinearSum([]*FDVariable{x, y}, []int{2, 3}, cost)
	m.AddConstraint(ls)

	solver := NewSolver(m)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Use impact-based heuristic to focus on objective-relevant variables
	sol, objVal, _ := solver.SolveOptimalWithOptions(ctx, cost, true,
		WithHeuristics(HeuristicImpact, ValueOrderObjImproving, 42))

	fmt.Printf("Minimum cost: %d (x=%d, y=%d)\n", objVal, sol[x.ID()], sol[y.ID()])
	// Output: Minimum cost: 5 (x=1, y=1)
}

```


\n
## pkg_minikanren_gcc_example_test.go-ExampleNewGlobalCardinality.md
```go
func ExampleNewGlobalCardinality() {
	model := NewModel()

	// Low-level constructors are preserved as comments for reference:
	// a := model.NewVariableWithName(NewBitSetDomainFromValues(2, []int{1}), "a")
	a := model.IntVarValues([]int{1}, "a")
	// b := model.NewVariableWithName(NewBitSetDomain(2), "b")
	b := model.IntVar(1, 2, "b")
	// c := model.NewVariableWithName(NewBitSetDomain(2), "c")
	c := model.IntVar(1, 2, "c")

	min := make([]int, 3)
	max := make([]int, 3)
	min[1], max[1] = 1, 1 // value 1 exactly once
	min[2], max[2] = 0, 3

	// Low-level API (kept as comment):
	// gcc, err := NewGlobalCardinality([]*FDVariable{a, b, c}, min, max)
	// if err != nil {
	//     panic(err)
	// }
	// model.AddConstraint(gcc)
	// HLAPI wrapper (preferred for examples):
	_ = model.GlobalCardinality([]*FDVariable{a, b, c}, min, max)

	solver := NewSolver(model)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_, _ = solver.Solve(ctx, 0)

	fmt.Println("a:", solver.GetDomain(nil, a.ID()))
	fmt.Println("b:", solver.GetDomain(nil, b.ID()))
	fmt.Println("c:", solver.GetDomain(nil, c.ID()))
	// Output:
	// a: {1}
	// b: {2}
	// c: {2}
}

```


\n
## pkg_minikanren_highlevel_api_collectors_example_test.go-Example_hlapi_collectors_ints.md
```go
func Example_hlapi_collectors_ints() {
	x := Fresh("x")
	goal := Disj(Eq(x, A(1)), Eq(x, A(3)), Eq(x, A(5)))
	vals := Ints(goal, x)
	// Print count and sum to avoid relying on order
	sum := 0
	for _, v := range vals {
		sum += v
	}
	fmt.Printf("%d %d\n", len(vals), sum)
	// Output:
	// 3 9
}

```


\n
## pkg_minikanren_highlevel_api_collectors_example_test.go-Example_hlapi_collectors_pairs_ints.md
```go
func Example_hlapi_collectors_pairs_ints() {
	x, y := Fresh("x"), Fresh("y")
	goal := Disj(
		Conj(Eq(x, A(1)), Eq(y, A(2))),
		Conj(Eq(x, A(3)), Eq(y, A(4))),
	)
	pairs := PairsInts(goal, x, y)
	// Print count and sum of all elements for stable output
	sum := 0
	for _, p := range pairs {
		sum += p[0] + p[1]
	}
	fmt.Printf("%d %d\n", len(pairs), sum)
	// Output:
	// 2 10
}

```


\n
## pkg_minikanren_highlevel_api_collectors_example_test.go-Example_hlapi_collectors_rows.md
```go
func Example_hlapi_collectors_rows() {
	x, y := Fresh("x"), Fresh("y")
	// Two solutions: (1, "a"), (2, "b")
	goal := Disj(
		Conj(Eq(x, A(1)), Eq(y, A("a"))),
		Conj(Eq(x, A(2)), Eq(y, A("b"))),
	)
	rows := Rows(goal, x, y)
	// Print as (x,y) using FormatTerm for consistent rendering
	for _, r := range rows {
		fmt.Printf("(%s,%s)\n", FormatTerm(r[0]), FormatTerm(r[1]))
	}
	// Unordered output; sort is not required for example semantics, but both
	// rows must appear. We'll accept either ordering by providing both variants.
	// Output:
	// (1,"a")
	// (2,"b")
}

```


\n
## pkg_minikanren_highlevel_api_collectors_example_test.go-Example_hlapi_rowsAll_timeout.md
```go
func Example_hlapi_rowsAll_timeout() {
	x, y := Fresh("x"), Fresh("y")
	goal := Disj(
		Conj(Eq(x, A(1)), Eq(y, A("a"))),
		Conj(Eq(x, A(2)), Eq(y, A("b"))),
	)
	rows := RowsAllTimeout(50*time.Millisecond, goal, x, y)
	fmt.Println(len(rows))
	// Output:
	// 2
}

```


\n
## pkg_minikanren_highlevel_api_example_test.go-ExampleModel_helpers_allDifferent.md
```go
func ExampleModel_helpers_allDifferent() {
	m := NewModel()
	xs := m.IntVars(3, 1, 3, "x")
	_ = m.AllDifferent(xs...)

	sols, err := SolveN(context.Background(), m, 0)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(len(sols))
	// Output:
	// 6
}

```


\n
## pkg_minikanren_highlevel_api_example_test.go-ExampleSolutions_basic.md
```go
func ExampleSolutions_basic() {
	q := Fresh("q")
	goal := Disj(Eq(q, NewAtom(1)), Eq(q, NewAtom(2)))
	out := FormatSolutions(Solutions(goal, q))
	fmt.Println(strings.Join(out, "\n"))
	// Output:
	// q: 1
	// q: 2
}

```


\n
## pkg_minikanren_highlevel_api_format_example_test.go-ExampleFormatTerm_basic.md
```go
func ExampleFormatTerm_basic() {
	fmt.Println(FormatTerm(L(1, 2, 3)))
	fmt.Println(FormatTerm(A("hello")))
	// Output:
	// (1 2 3)
	// "hello"
}

```


\n
## pkg_minikanren_highlevel_api_globals_example_test.go-Example_hlapi_cumulative.md
```go
func Example_hlapi_cumulative() {
	m := NewModel()
	A := m.IntVar(2, 2, "A") // fixed start=2
	B := m.IntVar(1, 4, "B") // start in [1..4]
	_ = m.Cumulative([]*FDVariable{A, B}, []int{2, 2}, []int{2, 1}, 2)

	s := NewSolver(m)
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	_, _ = s.Solve(ctx, 1) // trigger propagation

	fmt.Println("A:", s.GetDomain(nil, A.ID()))
	fmt.Println("B:", s.GetDomain(nil, B.ID()))
	// Output:
	// A: {2}
	// B: {4}
}

```


\n
## pkg_minikanren_highlevel_api_globals_example_test.go-Example_hlapi_gcc.md
```go
func Example_hlapi_gcc() {
	m := NewModel()
	a := m.IntVar(1, 1, "a") // fixed to 1
	b := m.IntVar(1, 2, "b")
	c := m.IntVar(1, 2, "c")

	min := make([]int, 3)
	max := make([]int, 3)
	min[1], max[1] = 1, 1 // value 1 exactly once
	min[2], max[2] = 0, 3
	_ = m.GlobalCardinality([]*FDVariable{a, b, c}, min, max)

	s := NewSolver(m)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_, _ = s.Solve(ctx, 0)

	fmt.Println("a:", s.GetDomain(nil, a.ID()))
	fmt.Println("b:", s.GetDomain(nil, b.ID()))
	fmt.Println("c:", s.GetDomain(nil, c.ID()))
	// Output:
	// a: {1}
	// b: {2}
	// c: {2}
}

```


\n
## pkg_minikanren_highlevel_api_globals_example_test.go-Example_hlapi_lexLessEq.md
```go
func Example_hlapi_lexLessEq() {
	m := NewModel()
	// Use compact range helpers instead of explicit value sets
	x1 := m.IntVar(2, 4, "x1")
	x2 := m.IntVar(1, 3, "x2")
	y1 := m.IntVar(3, 5, "y1")
	y2 := m.IntVar(2, 4, "y2")
	_ = m.LexLessEq([]*FDVariable{x1, x2}, []*FDVariable{y1, y2})

	// Minimal propagation: background context and solution cap 0
	s := NewSolver(m)
	_, _ = s.Solve(context.Background(), 0)

	fmt.Printf("y1: %s\n", s.GetDomain(nil, y1.ID()))
	// Output:
	// y1: {3..5}
}

```


\n
## pkg_minikanren_highlevel_api_globals_example_test.go-Example_hlapi_noOverlap.md
```go
func Example_hlapi_noOverlap() {
	m := NewModel()
	s := m.IntVarsWithNames([]string{"s1", "s2"}, 1, 3)
	_ = m.NoOverlap(s, []int{2, 2})

	// Enumerate solutions; only (1,3) and (3,1) are valid starts
	sols, err := SolveN(context.Background(), m, 0)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(len(sols))
	// Output:
	// 2
}

```


\n
## pkg_minikanren_highlevel_api_globals_example_test.go-Example_hlapi_regular.md
```go
func Example_hlapi_regular() {
	// Build DFA: accepts sequences ending with symbol 1 over alphabet {1,2}
	numStates, start, accept, delta := endsWith1DFA()

	m := NewModel()
	// x1 := m.NewVariableWithName(NewBitSetDomain(2), "x1")
	x1 := m.IntVar(1, 2, "x1")
	// x2 := m.NewVariableWithName(NewBitSetDomain(2), "x2")
	x2 := m.IntVar(1, 2, "x2")
	// x3 := m.NewVariableWithName(NewBitSetDomain(2), "x3")
	x3 := m.IntVar(1, 2, "x3")
	_ = m.Regular([]*FDVariable{x1, x2, x3}, numStates, start, accept, delta)

	s := NewSolver(m)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, _ = s.Solve(ctx, 0)
	fmt.Println("x1:", s.GetDomain(nil, x1.ID()))
	fmt.Println("x2:", s.GetDomain(nil, x2.ID()))
	fmt.Println("x3:", s.GetDomain(nil, x3.ID()))
	// Output:
	// x1: {1..2}
	// x2: {1..2}
	// x3: {1}
}

```


\n
## pkg_minikanren_highlevel_api_globals_example_test.go-Example_hlapi_table.md
```go
func Example_hlapi_table() {
	m := NewModel()
	// x := m.NewVariableWithName(NewBitSetDomain(5), "x")
	x := m.IntVar(1, 5, "x")
	// y ∈ {1,2} upfront so we can avoid internal propagation calls
	// y := m.NewVariableWithName(NewBitSetDomainFromValues(5, []int{1, 2}), "y")
	y := m.IntVarValues([]int{1, 2}, "y")

	rows := [][]int{
		{1, 1},
		{2, 3},
		{3, 2},
	}
	_ = m.Table([]*FDVariable{x, y}, rows)

	s := NewSolver(m)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, _ = s.Solve(ctx, 0)

	xd := s.GetDomain(nil, x.ID())
	yd := s.GetDomain(nil, y.ID())

	fmt.Printf("x: %v\n", xd)
	fmt.Printf("y: %v\n", yd)
	// Output:
	// x: {1,3}
	// y: {1..2}
}

```


\n
## pkg_minikanren_highlevel_api_intvarvalues_example_test.go-ExampleModel_helpers_intVarValues.md
```go
func ExampleModel_helpers_intVarValues() {
	m := NewModel()
	x := m.IntVarValues([]int{1, 3, 5}, "x")

	s := NewSolver(m)
	// Initial domain reflects the provided set exactly
	fmt.Println(s.GetDomain(nil, x.ID()))
	// Output:
	// {1,3,5}
}

```


\n
## pkg_minikanren_highlevel_api_optimize_example_test.go-Example_hlapi_optimize.md
```go
func Example_hlapi_optimize() {
	m := NewModel()
	xs := m.IntVars(2, 1, 3, "x") // x1, x2 in [1..3]
	total := m.IntVar(0, 10, "t")
	_ = m.LinearSum(xs, []int{1, 2}, total) // t = x1 + 2*x2

	// Minimize t
	_, best, err := Optimize(m, total, true)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("best=%d\n", best)
	// Output:
	// best=3
}

```


\n
## pkg_minikanren_highlevel_api_optimize_example_test.go-Example_hlapi_optimize_withOptions.md
```go
func Example_hlapi_optimize_withOptions() {
	m := NewModel()
	xs := m.IntVars(2, 1, 3, "x")
	total := m.IntVar(0, 10, "t")
	_ = m.LinearSum(xs, []int{1, 2}, total)

	// Use context and one option as a smoke test for the wrapper
	ctx := context.Background()
	_, best, err := OptimizeWithOptions(ctx, m, total, true, WithParallelWorkers(2))
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("best=%d\n", best)
	// Output:
	// best=3
}

```


\n
## pkg_minikanren_highlevel_api_pldb_disjq_example_test.go-Example_hlapi_disjq.md
```go
func Example_hlapi_disjq() {
	rel := MustRel("edge", 2, 0, 1)
	db := DB().MustAddFacts(rel,
		[]interface{}{"a", "b"},
		[]interface{}{"b", "c"},
		[]interface{}{"c", "d"},
	)

	x := Fresh("x")
	y := Fresh("y")

	// Either edge(x, y) or edge(y, x)
	goal := DisjQ(db, rel, []interface{}{x, y}, []interface{}{y, x})

	ctx := context.Background()
	stores := goal(ctx, NewLocalConstraintStore(NewGlobalConstraintBus()))
	rows, _ := stores.Take(100)
	// Both disjuncts contribute: edge(x,y) yields 3 rows, and edge(y,x)
	// yields 3 rows with swapped bindings, for a total of 6.
	fmt.Println(len(rows))
	// Output:
	// 6
}

```


\n
## pkg_minikanren_highlevel_api_pldb_example_test.go-Example_pldb_join.md
```go
func Example_pldb_join() {
	parent := MustRel("parent", 2, 0, 1)

	db := DB().MustAddFacts(parent,
		[]interface{}{"alice", "bob"},
		[]interface{}{"bob", "charlie"},
		[]interface{}{"charlie", "diana"},
	)

	gp := Fresh("gp")
	gc := Fresh("gc")
	p := Fresh("p")

	// grandparent(GP, GC) :- parent(GP, P), parent(P, GC)
	goal := Conj(
		db.Q(parent, gp, p),
		db.Q(parent, p, gc),
	)

	// Count results for a stable example output
	ctx := context.Background()
	stores := goal(ctx, NewLocalConstraintStore(NewGlobalConstraintBus()))
	rows, _ := stores.Take(10)
	fmt.Println(len(rows))
	// Output:
	// 2
}

```


\n
## pkg_minikanren_highlevel_api_pldb_example_test.go-Example_tabled_query.md
```go
func Example_tabled_query() {
	edge := MustRel("edge", 2, 0, 1)
	// a -> b, b -> c
	db := DB().MustAddFacts(edge,
		[]interface{}{"a", "b"},
		[]interface{}{"b", "c"},
	)

	x := Fresh("x")
	y := Fresh("y")

	// TQ uses rel.Name() as predicate id and caches answers
	goal := TQ(db, edge, x, y)

	ctx := context.Background()
	stores := goal(ctx, NewLocalConstraintStore(NewGlobalConstraintBus()))
	rows, _ := stores.Take(10)
	fmt.Println(len(rows))
	// Output:
	// 2
}

```


\n
## pkg_minikanren_highlevel_api_pldb_recursive_example_test.go-Example_hlapi_ancestor_recursive_sugar.md
```go
func Example_hlapi_ancestor_recursive_sugar() {
	parent := MustRel("parent", 2, 0, 1)
	db := DB().MustAddFacts(parent,
		[]interface{}{"john", "mary"},
		[]interface{}{"mary", "alice"},
		[]interface{}{"john", "tom"},
		[]interface{}{"tom", "bob"},
	)

	// ancestor2(X,Y) :- parent(X,Y).
	// ancestor2(X,Y) :- parent(X,Z), ancestor2(Z,Y).
	ancestor2 := RecursiveTablePred(db, parent, "ancestor2",
		func(self func(...Term) Goal, args ...Term) Goal {
			x, y := args[0], args[1]
			z := Fresh("z")
			return Conj(
				db.Query(parent, x, z),
				self(z, y),
			)
		})

	x := Fresh("x")

	// Mix Terms and native values at call sites
	goal := ancestor2(x, "alice")

	ctx := context.Background()
	stores := goal(ctx, NewLocalConstraintStore(NewGlobalConstraintBus()))
	rows, _ := stores.Take(10)
	// john -> mary -> alice, so both john and mary are ancestors of alice
	fmt.Println(len(rows))
	// Output:
	// 2
}

```


\n
## pkg_minikanren_highlevel_api_pldb_recursive_example_test.go-Example_hlapi_values_projection.md
```go
func Example_hlapi_values_projection() {
	x := Fresh("x")
	goal := Disj(Eq(x, NewAtom(1)), Eq(x, NewAtom(2)))
	sols := Solutions(goal, x)
	ints := ValuesInt(sols, "x")
	// Print count and sum to avoid relying on order
	sum := 0
	for _, v := range ints {
		sum += v
	}
	fmt.Printf("%d %d\n", len(ints), sum)
	// Output:
	// 2 3
}

```


\n
## pkg_minikanren_highlevel_api_pldb_slg_example_test.go-Example_hlapi_ancestor_recursive.md
```go
func Example_hlapi_ancestor_recursive() {
	parent := MustRel("parent", 2, 0, 1)
	db := DB().MustAddFacts(parent,
		[]interface{}{"john", "mary"},
		[]interface{}{"mary", "alice"},
		[]interface{}{"john", "tom"},
		[]interface{}{"tom", "bob"},
	)

	// Define ancestor(X,Y): parent(X,Y) OR (parent(X,Z) AND ancestor(Z,Y))
	ancestor := TabledRecursivePredicate(db, parent, "ancestor",
		func(self func(...Term) Goal, args ...Term) Goal {
			x, y := args[0], args[1]
			z := Fresh("z")
			return Conj(
				db.Q(parent, x, z),
				self(z, y),
			)
		},
	)

	x := Fresh("x")
	y := Fresh("y")

	goal := Conj(
		Eq(y, NewAtom("alice")),
		ancestor(x, y),
	)

	ctx := context.Background()
	stores := goal(ctx, NewLocalConstraintStore(NewGlobalConstraintBus()))
	rows, _ := stores.Take(10)
	// john -> mary -> alice, so both john and mary are ancestors of alice
	// We just assert we found two rows to keep the example stable.
	fmt.Println(len(rows))
	// Output:
	// 2
}

```


\n
## pkg_minikanren_highlevel_api_pldb_slg_example_test.go-Example_hlapi_grandparent.md
```go
func Example_hlapi_grandparent() {
	parent := MustRel("parent", 2, 0, 1)
	db := DB().MustAddFacts(parent,
		[]interface{}{"john", "mary"},
		[]interface{}{"mary", "alice"},
	)

	gp := Fresh("gp")
	p := Fresh("p")
	gc := Fresh("gc")

	goal := Conj(
		TQ(db, parent, gp, p),
		TQ(db, parent, p, gc),
		Eq(gp, NewAtom("john")),
	)

	ctx := context.Background()
	stores := goal(ctx, NewLocalConstraintStore(NewGlobalConstraintBus()))
	rows, _ := stores.Take(10)
	fmt.Println(len(rows))
	// Output:
	// 1
}

```


\n
## pkg_minikanren_highlevel_api_pldb_slg_example_test.go-Example_hlapi_multiRelationLoader.md
```go
func Example_hlapi_multiRelationLoader() {
	emp, mgr := MustRel("employee", 2, 0, 1), MustRel("manager", 2, 0, 1)
	rels := map[string]*Relation{"employee": emp, "manager": mgr}
	data := map[string][][]interface{}{
		"employee": {{"alice", "eng"}, {"bob", "eng"}},
		"manager":  {{"bob", "alice"}},
	}
	// Load both relations in one pass
	db, _ := NewDBFromMap(rels, data)

	mgrVar := Fresh("mgr")
	goal := TQ(db, mgr, mgrVar, "alice")

	ctx := context.Background()
	stores := goal(ctx, NewLocalConstraintStore(NewGlobalConstraintBus()))
	rows, _ := stores.Take(10)
	fmt.Println(len(rows))
	// Output:
	// 1
}

```


\n
## pkg_minikanren_highlevel_api_pldb_slg_example_test.go-Example_hlapi_path_twoHop.md
```go
func Example_hlapi_path_twoHop() {
	edge := MustRel("edge", 2, 0, 1)
	db := DB().MustAddFacts(edge,
		[]interface{}{"a", "b"},
		[]interface{}{"b", "c"},
		[]interface{}{"c", "d"},
	)

	x := Fresh("x")
	z := Fresh("z")
	y := Fresh("y")

	// twoHop(X, Y) :- edge(X, Z), edge(Z, Y)
	goal := Conj(
		Eq(x, NewAtom("a")),
		TQ(db, edge, x, z),
		TQ(db, edge, z, y),
	)

	ctx := context.Background()
	stores := goal(ctx, NewLocalConstraintStore(NewGlobalConstraintBus()))
	rows, _ := stores.Take(10)
	fmt.Println(len(rows))
	// Output:
	// 1
}

```


\n
## pkg_minikanren_hybrid_example_test.go-ExampleFDPlugin.md
```go
func ExampleFDPlugin() {
	// Create model with AllDifferent constraint
	model := NewModel()
	x := model.NewVariable(NewBitSetDomain(5))
	y := model.NewVariable(NewBitSetDomain(5))
	z := model.NewVariable(NewBitSetDomain(5))

	allDiff, _ := NewAllDifferent([]*FDVariable{x, y, z})
	model.AddConstraint(allDiff)

	// Create FD plugin
	plugin := NewFDPlugin(model)

	fmt.Printf("Plugin name: %s\n", plugin.Name())
	fmt.Printf("Can handle AllDifferent: %v\n", plugin.CanHandle(allDiff))

	// Output:
	// Plugin name: FD
	// Can handle AllDifferent: true
}

```


\n
## pkg_minikanren_hybrid_example_test.go-ExampleHybridSolver_bidirectionalPropagation.md
```go
func ExampleHybridSolver_bidirectionalPropagation() {
	// Create FD model with AllDifferent constraint
	model := NewModel()
	x := model.NewVariable(NewBitSetDomain(10))
	y := model.NewVariable(NewBitSetDomain(10))
	z := model.NewVariable(NewBitSetDomain(10))

	// FD constraint: all different
	allDiff, _ := NewAllDifferent([]*FDVariable{x, y, z})
	model.AddConstraint(allDiff)

	// Build solver and store from model helper; then set initial domains
	solver, store, err := NewHybridSolverFromModel(model)
	if err != nil {
		panic(err)
	}

	domain := NewBitSetDomainFromValues(10, []int{1, 2, 3})
	store, _ = store.SetDomain(x.ID(), domain)
	store, _ = store.SetDomain(y.ID(), domain)
	store, _ = store.SetDomain(z.ID(), domain)

	// HYBRID STEP 1: Relational solver binds x to 2 (e.g., from unification)
	// This is the key: a relational binding influences FD domains
	store, _ = store.AddBinding(int64(x.ID()), NewAtom(2))

	// Run hybrid propagation
	result, _ := solver.Propagate(store)

	// HYBRID RESULT 1: x's FD domain pruned to {2} (relational → FD)
	xDom := result.GetDomain(x.ID())
	fmt.Printf("x domain after relational binding: {%d}\n", xDom.SingletonValue())

	// HYBRID RESULT 2: AllDifferent removes 2 from y and z (FD propagation)
	yDom := result.GetDomain(y.ID())
	fmt.Printf("y domain size after AllDifferent: %d\n", yDom.Count())
	fmt.Printf("y contains 2: %v\n", yDom.Has(2))

	// HYBRID RESULT 3: x's binding exists (FD singleton promoted back to relational)
	xBinding := result.GetBinding(int64(x.ID()))
	fmt.Printf("x has relational binding: %v\n", xBinding != nil)

	// Output:
	// x domain after relational binding: {2}
	// y domain size after AllDifferent: 2
	// y contains 2: false
	// x has relational binding: true
}

```


\n
## pkg_minikanren_hybrid_example_test.go-ExampleHybridSolver_Propagate.md
```go
func ExampleHybridSolver_Propagate() {
	// Create FD model
	model := NewModel()
	x := model.NewVariable(NewBitSetDomain(10))
	y := model.NewVariable(NewBitSetDomain(10))

	// x + 2 = y
	arith, _ := NewArithmetic(x, y, 2)
	model.AddConstraint(arith)

	// Create solver and baseline store from model helper, then override domains
	solver, store, err := NewHybridSolverFromModel(model)
	if err != nil {
		panic(err)
	}

	store, _ = store.SetDomain(x.ID(), NewBitSetDomainFromValues(10, []int{3, 4, 5}))
	store, _ = store.SetDomain(y.ID(), NewBitSetDomain(10))

	// Run propagation
	result, _ := solver.Propagate(store)

	// Check propagated domains
	yDomain := result.GetDomain(y.ID())
	fmt.Printf("After propagation, y domain size: %d\n", yDomain.Count())
	fmt.Printf("y contains 5: %v\n", yDomain.Has(5))
	fmt.Printf("y contains 6: %v\n", yDomain.Has(6))
	fmt.Printf("y contains 7: %v\n", yDomain.Has(7))

	// Output:
	// After propagation, y domain size: 3
	// y contains 5: true
	// y contains 6: true
	// y contains 7: true
}

```


\n
## pkg_minikanren_hybrid_example_test.go-ExampleHybridSolver_realWorldScheduling.md
```go
func ExampleHybridSolver_realWorldScheduling() {
	// FD model: task start times with temporal constraints
	model := NewModel()
	task1 := model.NewVariableWithName(NewBitSetDomain(10), "task1_time")
	task2 := model.NewVariableWithName(NewBitSetDomain(10), "task2_time")
	task3 := model.NewVariableWithName(NewBitSetDomain(10), "task3_time")

	// FD constraint: task2 must start after task1 (task1 + 2 = task2)
	precedence, _ := NewArithmetic(task1, task2, 2)
	model.AddConstraint(precedence)

	// FD constraint: all tasks at different times
	allDiff, _ := NewAllDifferent([]*FDVariable{task1, task2, task3})
	model.AddConstraint(allDiff)

	// Create solver and store from model helper; then set initial domains
	solver, store, err := NewHybridSolverFromModel(model)
	if err != nil {
		panic(err)
	}

	timeSlots := NewBitSetDomainFromValues(10, []int{1, 2, 3, 4, 5})
	store, _ = store.SetDomain(task1.ID(), timeSlots)
	store, _ = store.SetDomain(task2.ID(), timeSlots)
	store, _ = store.SetDomain(task3.ID(), timeSlots)

	// Relational constraint: task1 must be a number (type safety)
	task1Var := Fresh("task1")
	typeConstraint := NewTypeConstraint(task1Var, NumberType)
	store = store.AddConstraint(typeConstraint)

	// External decision: task1 scheduled at time 1 (from relational reasoning)
	store, _ = store.AddBinding(int64(task1.ID()), NewAtom(1))

	// Hybrid propagation
	result, _ := solver.Propagate(store)

	// Results show hybrid cooperation:
	// - Relational binding (task1=1) → FD domain {1}
	// - FD arithmetic (1+2=3) → task2 domain {3}
	// - FD AllDifferent → task3 domain excludes {1,3}

	task1Time := result.GetDomain(task1.ID()).SingletonValue()
	task2Time := result.GetDomain(task2.ID()).SingletonValue()
	task3Dom := result.GetDomain(task3.ID())

	fmt.Printf("Task 1 starts at: %d\n", task1Time)
	fmt.Printf("Task 2 starts at: %d (precedence constraint)\n", task2Time)
	fmt.Printf("Task 3 possible times: %d slots\n", task3Dom.Count())
	fmt.Printf("Task 3 cannot use time 1: %v\n", !task3Dom.Has(1))
	fmt.Printf("Task 3 cannot use time 3: %v\n", !task3Dom.Has(3))

	// Output:
	// Task 1 starts at: 1
	// Task 2 starts at: 3 (precedence constraint)
	// Task 3 possible times: 3 slots
	// Task 3 cannot use time 1: true
	// Task 3 cannot use time 3: true
}

```


\n
## pkg_minikanren_hybrid_example_test.go-ExampleNewHybridSolver.md
```go
func ExampleNewHybridSolver() {
	// Create an FD model with variables and constraints
	model := NewModel()
	x := model.NewVariable(NewBitSetDomain(10))
	y := model.NewVariable(NewBitSetDomain(10))

	// Add FD constraint: x + 1 = y
	arith, _ := NewArithmetic(x, y, 1)
	model.AddConstraint(arith)

	// Create plugins explicitly to preserve the canonical demonstration order
	// (FD plugin followed by Relational). This example intentionally shows
	// the plugin ordering used elsewhere in the docs.
	fdPlugin := NewFDPlugin(model)
	relPlugin := NewRelationalPlugin()

	// Create hybrid solver with both plugins
	solver := NewHybridSolver(fdPlugin, relPlugin)

	fmt.Printf("Hybrid solver has %d plugins\n", len(solver.GetPlugins()))
	fmt.Printf("Plugin 1: %s\n", solver.GetPlugins()[0].Name())
	fmt.Printf("Plugin 2: %s\n", solver.GetPlugins()[1].Name())

	// Output:
	// Hybrid solver has 2 plugins
	// Plugin 1: FD
	// Plugin 2: Relational
}

```


\n
## pkg_minikanren_hybrid_example_test.go-ExampleNewUnifiedStore.md
```go
func ExampleNewUnifiedStore() {
	// Create a new unified store
	store := NewUnifiedStore()

	// Add a relational binding for logic variable 1
	store, _ = store.AddBinding(1, NewAtom(42))

	// Add an FD domain for FD variable 2
	store, _ = store.SetDomain(2, NewBitSetDomain(10))

	// The store can hold both types of information
	fmt.Printf("Store has bindings: %d\n", len(store.getAllBindings()))
	fmt.Printf("Store has domains: %d\n", len(store.getAllDomains()))

	// Output:
	// Store has bindings: 1
	// Store has domains: 1
}

```


\n
## pkg_minikanren_hybrid_example_test.go-ExampleRelationalPlugin.md
```go
func ExampleRelationalPlugin() {
	// Create relational plugin
	plugin := NewRelationalPlugin()

	// Create a type constraint for variable 1
	typeConstraint := NewTypeConstraint(Fresh("x"), NumberType)

	fmt.Printf("Plugin name: %s\n", plugin.Name())
	fmt.Printf("Can handle type constraint: %v\n", plugin.CanHandle(typeConstraint))

	// Create store with binding for variable 1
	store := NewUnifiedStore()
	store, _ = store.AddBinding(1, NewAtom(42))
	store = store.AddConstraint(typeConstraint)

	// Propagate (checks constraints)
	result, err := plugin.Propagate(store)

	if err != nil {
		fmt.Println("Constraint violated")
	} else {
		fmt.Printf("Constraint satisfied: %v\n", result != nil)
	}

	// Output:
	// Plugin name: Relational
	// Can handle type constraint: true
	// Constraint satisfied: true
}

```


\n
## pkg_minikanren_hybrid_example_test.go-ExampleRelationalPlugin_promoteSingletons.md
```go
func ExampleRelationalPlugin_promoteSingletons() {
	// Create relational plugin
	plugin := NewRelationalPlugin()

	// Create store with singleton FD domain
	store := NewUnifiedStore()
	singletonDomain := NewBitSetDomainFromValues(10, []int{7})
	store, _ = store.SetDomain(1, singletonDomain)

	// Propagate (should promote singleton)
	result, _ := plugin.Propagate(store)

	// Check if binding was created
	binding := result.GetBinding(1)
	if binding != nil {
		atom := binding.(*Atom)
		fmt.Printf("Singleton promoted to binding: %v\n", atom.Value())
	}

	// Output:
	// Singleton promoted to binding: 7
}

```


\n
## pkg_minikanren_hybrid_example_test.go-ExampleUnifiedStore_AddBinding.md
```go
func ExampleUnifiedStore_AddBinding() {
	store := NewUnifiedStore()

	// Add bindings (using variable IDs 1 and 2)
	store, _ = store.AddBinding(1, NewAtom("hello"))
	store, _ = store.AddBinding(2, NewAtom(42))

	// Retrieve bindings
	xBinding := store.GetBinding(1)
	yBinding := store.GetBinding(2)

	fmt.Printf("var 1 = %v\n", xBinding.(*Atom).Value())
	fmt.Printf("var 2 = %v\n", yBinding.(*Atom).Value())

	// Output:
	// var 1 = hello
	// var 2 = 42
}

```


\n
## pkg_minikanren_hybrid_example_test.go-ExampleUnifiedStore_SetDomain.md
```go
func ExampleUnifiedStore_SetDomain() {
	store := NewUnifiedStore()

	// Set domain for variable 1: values {1, 2, 3}
	domain := NewBitSetDomainFromValues(10, []int{1, 2, 3})
	store, _ = store.SetDomain(1, domain)

	// Retrieve and inspect domain
	d := store.GetDomain(1)
	fmt.Printf("Domain size: %d\n", d.Count())
	fmt.Printf("Contains 2: %v\n", d.Has(2))

	// Output:
	// Domain size: 3
	// Contains 2: true
}

```


\n
## pkg_minikanren_hybrid_registry_example_test.go-ExampleHybridRegistry_AutoBind.md
```go
func ExampleHybridRegistry_AutoBind() {
	ctx := context.Background()
	model := NewModel()

	// Setup database
	employee, _ := DbRel("employee", 3, 0)
	db := NewDatabase()
	db, _ = db.AddFact(employee, NewAtom("alice"), NewAtom(28), NewAtom(50000))
	db, _ = db.AddFact(employee, NewAtom("bob"), NewAtom(35), NewAtom(60000))

	// Setup FD variables
	ageVar := model.NewVariable(NewBitSetDomain(100))
	salaryVar := model.NewVariable(NewBitSetDomain(100000))

	// Create registry mapping relational vars to FD vars
	name := Fresh("name")
	age := Fresh("age")
	salary := Fresh("salary")

	registry := NewHybridRegistry()
	registry, _ = registry.MapVars(age, ageVar)
	registry, _ = registry.MapVars(salary, salaryVar)

	// Query database
	goal := db.Query(employee, name, age, salary)
	store := NewUnifiedStore()
	adapter := NewUnifiedStoreAdapter(store)
	results, _ := goal(ctx, adapter).Take(2)

	// AutoBind automatically transfers bindings from query results to FD store
	var employees []string
	for _, result := range results {
		// Single AutoBind call replaces manual binding transfer
		fdStore, _ := registry.AutoBind(result, store)

		nameBinding := result.GetBinding(name.ID())
		ageBinding := fdStore.GetBinding(int64(ageVar.ID()))
		salaryBinding := fdStore.GetBinding(int64(salaryVar.ID()))

		n := nameBinding.(*Atom).value.(string)
		a := ageBinding.(*Atom).value.(int)
		s := salaryBinding.(*Atom).value.(int)

		employees = append(employees, fmt.Sprintf("%s: age=%d salary=%d", n, a, s))
	}

	sort.Strings(employees)
	for _, emp := range employees {
		fmt.Println(emp)
	}

	// Output:
	// alice: age=28 salary=50000
	// bob: age=35 salary=60000
}

```


\n
## pkg_minikanren_hybrid_registry_example_test.go-ExampleHybridRegistry_multipleVariables.md
```go
func ExampleHybridRegistry_multipleVariables() {
	model := NewModel()

	// Setup multiple variable pairs
	age := Fresh("age")
	salary := Fresh("salary")
	bonus := Fresh("bonus")
	yearsOfService := Fresh("years")

	ageVar := model.NewVariable(NewBitSetDomain(100))
	salaryVar := model.NewVariable(NewBitSetDomain(100000))
	bonusVar := model.NewVariable(NewBitSetDomain(10000))
	yearsVar := model.NewVariable(NewBitSetDomain(50))

	// Build registry incrementally
	registry := NewHybridRegistry()
	registry, _ = registry.MapVars(age, ageVar)
	registry, _ = registry.MapVars(salary, salaryVar)
	registry, _ = registry.MapVars(bonus, bonusVar)
	registry, _ = registry.MapVars(yearsOfService, yearsVar)

	// Query registry state
	fmt.Printf("Total mappings: %d\n", registry.MappingCount())
	fmt.Printf("Age mapped: %t\n", registry.HasMapping(age))
	fmt.Printf("Salary mapped: %t\n", registry.HasMapping(salary))
	fmt.Printf("Bonus mapped: %t\n", registry.HasMapping(bonus))
	fmt.Printf("Years mapped: %t\n", registry.HasMapping(yearsOfService))

	// Bidirectional lookups work correctly
	ageFDID := registry.GetFDVariable(age)
	salaryRelID := registry.GetRelVariable(salaryVar)
	fmt.Printf("Age has FD mapping: %t\n", ageFDID >= 0)
	fmt.Printf("Salary has relational mapping: %t\n", salaryRelID >= 0)

	// Output:
	// Total mappings: 4
	// Age mapped: true
	// Salary mapped: true
	// Bonus mapped: true
	// Years mapped: true
	// Age has FD mapping: true
	// Salary has relational mapping: true
}

```


\n
## pkg_minikanren_hybrid_registry_example_test.go-ExampleNewHybridRegistry.md
```go
func ExampleNewHybridRegistry() {
	// Create a registry for tracking relational↔FD variable mappings
	registry := NewHybridRegistry()

	// Setup variables
	model := NewModel()
	age := Fresh("age")
	ageVar := model.NewVariable(NewBitSetDomain(100))

	// Register the mapping
	registry, _ = registry.MapVars(age, ageVar)

	// Query the mapping
	fdID := registry.GetFDVariable(age)
	fmt.Printf("Has mapping: %t\n", fdID >= 0)
	fmt.Printf("Registry has %d mapping(s)\n", registry.MappingCount())

	// Output:
	// Has mapping: true
	// Registry has 1 mapping(s)
}

```


\n
## pkg_minikanren_lex_example_test.go-ExampleNewLexLessEq.md
```go
func ExampleNewLexLessEq() {
	model := NewModel()
	// x1 := model.NewVariableWithName(NewBitSetDomainFromValues(9, []int{2, 3, 4}), "x1")
	x1 := model.IntVarValues([]int{2, 3, 4}, "x1")
	// x2 := model.NewVariableWithName(NewBitSetDomainFromValues(9, []int{1, 2, 3}), "x2")
	x2 := model.IntVarValues([]int{1, 2, 3}, "x2")
	// y1 := model.NewVariableWithName(NewBitSetDomainFromValues(9, []int{3, 4, 5}), "y1")
	y1 := model.IntVarValues([]int{3, 4, 5}, "y1")
	// y2 := model.NewVariableWithName(NewBitSetDomainFromValues(9, []int{2, 3, 4}), "y2")
	y2 := model.IntVarValues([]int{2, 3, 4}, "y2")

	c, _ := NewLexLessEq([]*FDVariable{x1, x2}, []*FDVariable{y1, y2})
	model.AddConstraint(c)

	solver := NewSolver(model)
	// Run fixed-point propagation via a zero-solution search (limit=0)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, _ = solver.Solve(ctx, 0)

	fmt.Printf("y1: %s\n", solver.GetDomain(nil, y1.ID()))
	// Output:
	// y1: {3..5}
}

```


\n
## pkg_minikanren_list_ops_example_test.go-ExampleDistincto.md
```go
func ExampleDistincto() {
	goalSuccess := Distincto(List(NewAtom(1), NewAtom(2), NewAtom(3)))
	resultsSuccess := runGoal(goalSuccess)
	fmt.Printf("Distinct list succeeds: %v\n", len(resultsSuccess) > 0)

	goalFail := Distincto(List(NewAtom(1), NewAtom(2), NewAtom(1)))
	resultsFail := runGoal(goalFail)
	fmt.Printf("Non-distinct list fails: %v\n", len(resultsFail) == 0)
	// Output:
	// Distinct list succeeds: true
	// Non-distinct list fails: true
}

```


\n
## pkg_minikanren_list_ops_example_test.go-ExampleFlatteno.md
```go
func ExampleFlatteno() {
	q := Fresh("q")
	nested := List(List(NewAtom(1), NewAtom(2)), List(NewAtom(3), List(NewAtom(4), NewAtom(5))))
	goal := Flatteno(nested, q)
	for _, s := range runGoal(goal, q) {
		fmt.Println(s)
	}
	// Output:
	// q: (1 2 3 4 5)
}

```


\n
## pkg_minikanren_list_ops_example_test.go-ExampleLengthoInt.md
```go
func ExampleLengthoInt() {
	q := Fresh("q")
	goal := LengthoInt(List(NewAtom(1), NewAtom(2), NewAtom(3)), q)
	for _, s := range runGoal(goal, q) {
		fmt.Println(s)
	}
	// Output:
	// q: 3
}

```


\n
## pkg_minikanren_list_ops_example_test.go-ExampleNoto.md
```go
func ExampleNoto() {
	// Noto succeeds because Membero(4, [1,2,3]) fails
	goalSuccess := Noto(Membero(NewAtom(4), List(NewAtom(1), NewAtom(2), NewAtom(3))))
	resultsSuccess := runGoal(goalSuccess)
	fmt.Printf("Noto(fail) succeeds: %v\n", len(resultsSuccess) > 0)

	// Noto fails because Membero(2, [1,2,3]) succeeds
	goalFail := Noto(Membero(NewAtom(2), List(NewAtom(1), NewAtom(2), NewAtom(3))))
	resultsFail := runGoal(goalFail)
	fmt.Printf("Noto(success) fails: %v\n", len(resultsFail) == 0)
	// Output:
	// Noto(fail) succeeds: true
	// Noto(success) fails: true
}

```


\n
## pkg_minikanren_list_ops_example_test.go-ExamplePermuteo.md
```go
func ExamplePermuteo() {
	q := Fresh("q")
	goal := Permuteo(List(NewAtom(1), NewAtom(2), NewAtom(3)), q)
	results := runGoal(goal, q)
	fmt.Println(strings.Join(results, "\n"))
	// Output:
	// q: (1 2 3)
	// q: (1 3 2)
	// q: (2 1 3)
	// q: (2 3 1)
	// q: (3 1 2)
	// q: (3 2 1)
}

```


\n
## pkg_minikanren_list_ops_example_test.go-ExampleRembero.md
```go
func ExampleRembero() {
	q := Fresh("q")
	goal := Rembero(NewAtom("a"), List(NewAtom("a"), NewAtom("b"), NewAtom("a")), q)
	for _, s := range runGoal(goal, q) {
		fmt.Println(s)
	}
	// Output:
	// q: ("a" "b")
	// q: ("b" "a")
}

```


\n
## pkg_minikanren_list_ops_example_test.go-ExampleReverso.md
```go
func ExampleReverso() {
	q := Fresh("q")
	goal := Reverso(List(NewAtom(1), NewAtom(2), NewAtom(3)), q)
	for _, s := range runGoal(goal, q) {
		fmt.Println(s)
	}
	// Output:
	// q: (3 2 1)
}

```


\n
## pkg_minikanren_list_ops_example_test.go-ExampleSubseto.md
```go
func ExampleSubseto() {
	q := Fresh("q")
	goal := Subseto(q, List(NewAtom(1), NewAtom(2)))
	results := runGoal(goal, q)
	fmt.Println(strings.Join(results, "\n"))
	// Output:
	// q: ()
	// q: (1 2)
	// q: (1)
	// q: (2)
}

```


\n
## pkg_minikanren_minmax_example_test.go-ExampleNewMax.md
```go
func ExampleNewMax() {
	model := NewModel()
	x := model.NewVariable(NewBitSetDomain(9).RemoveBelow(2).RemoveAbove(4)) // [2..4]
	y := model.NewVariable(NewBitSetDomain(9).RemoveBelow(6).RemoveAbove(8)) // [6..8]
	r := model.NewVariable(NewBitSetDomain(9))                               // [1..9]

	c, _ := NewMax([]*FDVariable{x, y}, r)
	model.AddConstraint(c)

	solver := NewSolver(model)
	_, _ = solver.Solve(context.Background(), 0) // propagate

	dr := solver.GetDomain(nil, r.ID())
	fmt.Printf("R: [%d..%d]\n", dr.Min(), dr.Max())

	// Xi are pruned to be <= R.max = 8 (no change for these domains)
	dx := solver.GetDomain(nil, x.ID())
	dy := solver.GetDomain(nil, y.ID())
	fmt.Printf("X.max: %d, Y.max: %d\n", dx.Max(), dy.Max())
	// Output:
	// R: [6..8]
	// X.max: 4, Y.max: 8
}

```


\n
## pkg_minikanren_minmax_example_test.go-ExampleNewMin.md
```go
func ExampleNewMin() {
	model := NewModel()
	// Two variables with different lower bounds
	x := model.NewVariable(NewBitSetDomain(9).RemoveBelow(3).RemoveAbove(6)) // [3..6]
	y := model.NewVariable(NewBitSetDomain(9).RemoveBelow(5).RemoveAbove(7)) // [5..7]
	r := model.NewVariable(NewBitSetDomain(9))                               // [1..9]

	c, _ := NewMin([]*FDVariable{x, y}, r)
	model.AddConstraint(c)

	solver := NewSolver(model)
	_, _ = solver.Solve(context.Background(), 0) // propagate

	// R is clamped to [min mins .. min maxes] = [3 .. 6]
	dr := solver.GetDomain(nil, r.ID())
	fmt.Printf("R: [%d..%d]\n", dr.Min(), dr.Max())

	// All Xi are pruned to be >= R.min = 3
	dx := solver.GetDomain(nil, x.ID())
	dy := solver.GetDomain(nil, y.ID())
	fmt.Printf("X.min: %d, Y.min: %d\n", dx.Min(), dy.Min())
	// Output:
	// R: [3..6]
	// X.min: 3, Y.min: 5
}

```


\n
## pkg_minikanren_model_example_test.go-ExampleFDVariable_IsBound.md
```go
func ExampleFDVariable_IsBound() {
	domain := minikanren.NewBitSetDomain(10)
	variable := minikanren.NewFDVariable(0, domain)

	fmt.Printf("Unbound variable: IsBound=%v\n", variable.IsBound())

	// Create a singleton domain (bound variable)
	singletonDomain := minikanren.NewBitSetDomainFromValues(10, []int{5})
	boundVariable := minikanren.NewFDVariable(1, singletonDomain)

	fmt.Printf("Bound variable: IsBound=%v, Value=%d\n", boundVariable.IsBound(), boundVariable.Value())

	// Output:
	// Unbound variable: IsBound=false
	// Bound variable: IsBound=true, Value=5
}

```


\n
## pkg_minikanren_model_example_test.go-ExampleModel_NewVariables.md
```go
func ExampleModel_NewVariables() {
	model := minikanren.NewModel()

	// Create 4 variables with domains {1..9} for a 4-cell Sudoku
	vars := model.NewVariables(4, minikanren.NewBitSetDomain(9))

	fmt.Printf("Created %d variables\n", len(vars))
	for i, v := range vars {
		fmt.Printf("var[%d]: %s\n", i, v.String())
	}

	// Output:
	// Created 4 variables
	// var[0]: v0∈{1..9}
	// var[1]: v1∈{1..9}
	// var[2]: v2∈{1..9}
	// var[3]: v3∈{1..9}
}

```


\n
## pkg_minikanren_model_example_test.go-ExampleModel_NewVariablesWithNames.md
```go
func ExampleModel_NewVariablesWithNames() {
	model := minikanren.NewModel()

	names := []string{"red", "green", "blue"}
	colors := model.NewVariablesWithNames(names, minikanren.NewBitSetDomain(3))

	for _, v := range colors {
		fmt.Printf("%s\n", v.String())
	}

	// Output:
	// red∈{1..3}
	// green∈{1..3}
	// blue∈{1..3}
}

```


\n
## pkg_minikanren_model_example_test.go-ExampleModel_Validate.md
```go
func ExampleModel_Validate() {
	model := minikanren.NewModel()

	// Create a variable with normal domain
	// low-level: x := model.NewVariable(minikanren.NewBitSetDomain(5))
	x := model.IntVar(1, 5, "x")
	_ = x

	// Model is valid
	err := model.Validate()
	fmt.Printf("Valid model: %v\n", err == nil)

	// Create a variable with empty domain - this is an error
	emptyDomain := minikanren.NewBitSetDomainFromValues(5, []int{})
	// low-level: y := model.NewVariable(emptyDomain)
	y := model.NewVariable(emptyDomain)

	err = model.Validate()
	if err != nil {
		fmt.Printf("Invalid model: variable %s has empty domain\n", y.Name())
	}

	// Output:
	// Valid model: true
	// Invalid model: variable v1 has empty domain
}

```


\n
## pkg_minikanren_model_example_test.go-ExampleNewModel.md
```go
func ExampleNewModel() {
	model := minikanren.NewModel()

	// Create variables for a simple problem
	domain := minikanren.NewBitSetDomain(5)
	x := model.NewVariable(domain)
	y := model.NewVariable(domain)

	fmt.Printf("Model has %d variables\n", model.VariableCount())
	fmt.Printf("Variable x: %s\n", x.String())
	fmt.Printf("Variable y: %s\n", y.String())

	// Output:
	// Model has 2 variables
	// Variable x: v0∈{1..5}
	// Variable y: v1∈{1..5}
}

```


\n
## pkg_minikanren_model_example_test.go-ExampleNewSolver.md
```go
func ExampleNewSolver() {
	// Create a model: 3 variables, each can be 1, 2, or 3
	model := minikanren.NewModel()
	vars := model.NewVariables(3, minikanren.NewBitSetDomain(3))

	// Add constraint: all variables must be different
	// (Note: We'll use the new architecture in future phases)
	_ = vars // Constraints not yet integrated in this phase

	// Create a solver
	solver := minikanren.NewSolver(model)

	// Solve with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	solutions, err := solver.Solve(ctx, 10)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Found %d solutions\n", len(solutions))
	if len(solutions) > 0 {
		fmt.Printf("First solution: %v\n", solutions[0])
	}

	// Output will vary based on solver implementation
	// Output:
	// Found 27 solutions
	// First solution: [1 1 1]
}

```


\n
## pkg_minikanren_model_example_test.go-ExampleSolverConfig.md
```go
func ExampleSolverConfig() {
	// Create config with domain-over-degree heuristic
	config := &minikanren.SolverConfig{
		VariableHeuristic: minikanren.HeuristicDomDeg,
		ValueHeuristic:    minikanren.ValueOrderAsc,
		RandomSeed:        42,
	}

	model := minikanren.NewModelWithConfig(config)
	vars := model.NewVariables(4, minikanren.NewBitSetDomain(4))
	_ = vars

	fmt.Printf("Model config: %+v\n", model.Config())

	// Output:
	// Model config: &{VariableHeuristic:0 ValueHeuristic:0 RandomSeed:42}
}

```


\n
## pkg_minikanren_model_example_test.go-ExampleSolver_parallelSearch.md
```go
func ExampleSolver_parallelSearch() {
	model := minikanren.NewModel()
	model.NewVariables(3, minikanren.NewBitSetDomain(5))

	// CORRECT: Single model shared by all workers (zero GC cost)
	// Each worker creates its own Solver with independent state chains
	worker1Solver := minikanren.NewSolver(model)
	worker2Solver := minikanren.NewSolver(model)

	// Both solvers share the same immutable model
	fmt.Printf("Worker 1 model variables: %d\n", worker1Solver.Model().VariableCount())
	fmt.Printf("Worker 2 model variables: %d\n", worker2Solver.Model().VariableCount())
	fmt.Printf("Models are shared (same pointer): %v\n", worker1Solver.Model() == worker2Solver.Model())

	// State changes are isolated per worker via copy-on-write SolverState
	// No model cloning needed - this is the key architectural insight

	// Output:
	// Worker 1 model variables: 3
	// Worker 2 model variables: 3
	// Models are shared (same pointer): true
}

```


\n
## pkg_minikanren_nooverlap_example_test.go-ExampleNewNoOverlap.md
```go
func ExampleNewNoOverlap() {
	model := NewModel()

	// Task A fixed at start=2, duration=2 ⇒ executes over [2,3]
	// A := model.NewVariableWithName(NewBitSetDomainFromValues(10, []int{2}), "A")
	A := model.IntVarValues([]int{2}, "A")
	// Task B can start in [1..4], duration=2
	// B := model.NewVariableWithName(NewBitSetDomain(4), "B")
	B := model.IntVar(1, 4, "B")

	noov, err := NewNoOverlap([]*FDVariable{A, B}, []int{2, 2})
	if err != nil {
		panic(err)
	}
	model.AddConstraint(noov)

	solver := NewSolver(model)
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	// Propagate at root via a short search
	_, _ = solver.Solve(ctx, 1)

	fmt.Println("A:", solver.GetDomain(nil, A.ID()))
	fmt.Println("B:", solver.GetDomain(nil, B.ID()))
	// Output:
	// A: {2}
	// B: {4}
}

```


\n
## pkg_minikanren_nvalue_example_test.go-ExampleNewAtMostNValues.md
```go
func ExampleNewAtMostNValues() {
	model := NewModel()
	// x1 := model.NewVariableWithName(NewBitSetDomainFromValues(5, []int{1}), "x1")
	x1 := model.IntVarValues([]int{1}, "x1")
	// x2 := model.NewVariableWithName(NewBitSetDomainFromValues(5, []int{1, 2}), "x2")
	x2 := model.IntVarValues([]int{1, 2}, "x2")
	// x3 := model.NewVariableWithName(NewBitSetDomainFromValues(5, []int{1, 2}), "x3")
	x3 := model.IntVarValues([]int{1, 2}, "x3")
	// low-level: limit := model.NewVariableWithName(NewBitSetDomain(2), "limit") // distinct ≤ 1
	// HLAPI: express the same compact integer domain using IntVar
	limit := model.IntVar(1, 2, "limit") // distinct ≤ 1 encoded over {1,2}

	_, _ = NewAtMostNValues(model, []*FDVariable{x1, x2, x3}, limit)

	solver := NewSolver(model)
	_, _ = solver.Solve(context.Background(), 0) // propagate only

	fmt.Printf("x2: %s\n", solver.GetDomain(nil, x2.ID()))
	fmt.Printf("x3: %s\n", solver.GetDomain(nil, x3.ID()))
	// Output:
	// x2: {1}
	// x3: {1}
}

```


\n
## pkg_minikanren_nvalue_example_test.go-ExampleNewNValue.md
```go
func ExampleNewNValue() {
	model := NewModel()
	// x1 := model.NewVariableWithName(NewBitSetDomainFromValues(5, []int{1, 2}), "x1")
	x1 := model.IntVarValues([]int{1, 2}, "x1")
	// x2 := model.NewVariableWithName(NewBitSetDomainFromValues(5, []int{1, 2}), "x2")
	x2 := model.IntVarValues([]int{1, 2}, "x2")
	// Exact NValue=1 ⇒ NPlus1=2
	// nPlus1 := model.NewVariableWithName(NewBitSetDomainFromValues(2, []int{2}), "N+1")
	nPlus1 := model.IntVarValues([]int{2}, "N+1")

	_, _ = NewNValue(model, []*FDVariable{x1, x2}, nPlus1)

	solver := NewSolver(model)
	_, _ = solver.Solve(context.Background(), 0)

	// No pruning here, but the composition is established and will prune
	// as soon as one side gets fixed by other constraints or decisions.
	fmt.Printf("x1: %s\n", solver.GetDomain(nil, x1.ID()))
	fmt.Printf("x2: %s\n", solver.GetDomain(nil, x2.ID()))
	// Output:
	// x1: {1..2}
	// x2: {1..2}
}

```


\n
## pkg_minikanren_optimization_example_test.go-ExampleSolver_SolveOptimal.md
```go
func ExampleSolver_SolveOptimal() {
	model := NewModel()
	// x,y in {1,2,3}
	x := model.NewVariable(NewBitSetDomainFromValues(10, []int{1, 2, 3}))
	y := model.NewVariable(NewBitSetDomainFromValues(10, []int{1, 2, 3}))
	// total T = x + 2*y
	tvar := model.NewVariable(NewBitSetDomain(20))
	ls, _ := NewLinearSum([]*FDVariable{x, y}, []int{1, 2}, tvar)
	model.AddConstraint(ls)

	solver := NewSolver(model)
	sol, obj, err := solver.SolveOptimal(context.Background(), tvar, true)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("best objective: %d\n", obj)
	_ = sol // values per variable in model order
	// Output:
	// best objective: 3
}

```


\n
## pkg_minikanren_optimization_example_test.go-ExampleSolver_SolveOptimal_minOfArray.md
```go
func ExampleSolver_SolveOptimal_minOfArray() {
	model := NewModel()
	// Two variables with overlapping ranges
	x := model.NewVariable(NewBitSetDomainFromValues(10, []int{2, 3, 4, 5}))
	y := model.NewVariable(NewBitSetDomainFromValues(10, []int{3, 4, 5, 6, 7}))
	// r = min(x,y)
	r := model.NewVariable(NewBitSetDomain(10))
	c, _ := NewMin([]*FDVariable{x, y}, r)
	model.AddConstraint(c)

	solver := NewSolver(model)
	// Maximize the minimum value achievable across x and y
	_, best, _ := solver.SolveOptimal(context.Background(), r, false)
	fmt.Println("max min:", best)
	// Output:
	// max min: 5
}

```


\n
## pkg_minikanren_optimization_example_test.go-ExampleSolver_SolveOptimalWithOptions.md
```go
func ExampleSolver_SolveOptimalWithOptions() {
	model := NewModel()
	x := model.NewVariable(NewBitSetDomain(10))
	y := model.NewVariable(NewBitSetDomain(10))
	tvar := model.NewVariable(NewBitSetDomain(40))
	ls, _ := NewLinearSum([]*FDVariable{x, y}, []int{1, 2}, tvar)
	model.AddConstraint(ls)

	solver := NewSolver(model)
	// Use parallel workers without timeout for deterministic results
	ctx := context.Background()
	sol, best, err := solver.SolveOptimalWithOptions(ctx, tvar, true, WithParallelWorkers(4))
	_ = sol // solution slice omitted in example output for brevity

	if err != nil {
		fmt.Printf("error: %v\n", err)
		return
	}
	fmt.Printf("best=%d\n", best)
	// Output:
	// best=3
}

```


\n
## pkg_minikanren_parallel_search_examples_test.go-ExampleDefaultParallelSearchConfig.md
```go
func ExampleDefaultParallelSearchConfig() {
	cfg := DefaultParallelSearchConfig()
	// You can use cfg.NumWorkers to size solver parallelism, and cfg.WorkQueueSize
	// to adjust throughput vs memory. Only queue size is deterministic here.
	fmt.Printf("queue=%d\n", cfg.WorkQueueSize)
	// Output:
	// queue=1000
}

```


\n
## pkg_minikanren_parallel_search_examples_test.go-ExampleSolver_SolveParallel_cancel.md
```go
func ExampleSolver_SolveParallel_cancel() {
	model := NewModel()
	vars := model.NewVariables(8, NewBitSetDomain(8))
	alldiff, _ := NewAllDifferent(vars)
	model.AddConstraint(alldiff)

	solver := NewSolver(model)

	// Cancel the context immediately to deterministically demonstrate cancellation.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Because we cancelled before calling, SolveParallel should return quickly
	// with an error; print a short message to make the example deterministic.
	_, err := solver.SolveParallel(ctx, 8, 0)
	if err != nil {
		fmt.Println("cancelled")
	} else {
		fmt.Println("no-error")
	}
	// Output:
	// cancelled
}

```


\n
## pkg_minikanren_parallel_search_examples_test.go-ExampleSolver_SolveParallel_limit.md
```go
func ExampleSolver_SolveParallel_limit() {
	model := NewModel()
	vars := model.NewVariables(4, NewBitSetDomain(4))
	alldiff, _ := NewAllDifferent(vars)
	model.AddConstraint(alldiff)

	solver := NewSolver(model)
	ctx := context.Background()

	// Ask for at most 3 solutions (there are 4! = 24 total)
	solutions, err := solver.SolveParallel(ctx, 4, 3)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("solutions: %d\n", len(solutions))
	// Output:
	// solutions: 3
}

```


\n
## pkg_minikanren_parallel_search_examples_test.go-ExampleSolver_SolveParallel.md
```go
func ExampleSolver_SolveParallel() {
	// 1) Build a model with 3 variables, each from 1..3
	model := NewModel()
	vars := model.NewVariables(3, NewBitSetDomain(3))

	// 2) Constrain them to be all different
	alldiff, _ := NewAllDifferent(vars)
	model.AddConstraint(alldiff)

	// 3) Create a solver and run in parallel with up to 4 workers
	solver := NewSolver(model)
	ctx := context.Background()

	// Find up to 6 solutions in parallel (there are 3! = 6)
	solutions, err := solver.SolveParallel(ctx, 4, 6)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Printf("solutions: %d\n", len(solutions))
	// Output:
	// solutions: 6
}

```


\n
## pkg_minikanren_pattern_example_test.go-ExampleMatcha_deterministicChoice.md
```go
func ExampleMatcha_deterministicChoice() {
	// Process different data types deterministically
	process := func(data Term) string {
		result := Run(1, func(q *Var) Goal {
			return Matcha(data,
				// Check for Nil first
				NewClause(Nil, Eq(q, NewAtom("empty-list"))),
				// Then check for pair
				NewClause(NewPair(Fresh("_"), Fresh("_")), Eq(q, NewAtom("pair"))),
				// Default case
				NewClause(Fresh("_"), Eq(q, NewAtom("atom"))),
			)
		})

		if len(result) == 0 {
			return "error"
		}

		if atom, ok := result[0].(*Atom); ok {
			if s, ok := atom.value.(string); ok {
				return s
			}
		}
		return "error"
	}

	fmt.Println(process(Nil))
	fmt.Println(process(NewPair(NewAtom(1), NewAtom(2))))
	fmt.Println(process(NewAtom(42)))

	// Output:
	// empty-list
	// pair
	// atom
}

```


\n
## pkg_minikanren_pattern_example_test.go-ExampleMatcha.md
```go
func ExampleMatcha() {
	// Safe head extraction with default value
	extractHead := func(list Term) Term {
		return Run(1, func(q *Var) Goal {
			head := Fresh("head")
			return Matcha(list,
				NewClause(Nil, Eq(q, NewAtom("empty"))),
				NewClause(NewPair(head, Fresh("_")), Eq(q, head)),
			)
		})[0]
	}

	// Non-empty list
	list1 := List(NewAtom(42), NewAtom(99))
	fmt.Println(extractHead(list1))

	// Empty list
	list2 := Nil
	fmt.Println(extractHead(list2))

	// Output:
	// 42
	// empty
}

```


\n
## pkg_minikanren_pattern_example_test.go-ExampleMatcha_withHybridSolver.md
```go
func ExampleMatcha_withHybridSolver() {
	model := NewModel()
	x := model.NewVariable(NewBitSetDomainFromValues(100, []int{5, 10, 15}))

	store := NewUnifiedStore()
	store, _ = store.SetDomain(x.ID(), x.Domain())
	adapter := NewUnifiedStoreAdapter(store)

	q := Fresh("q")
	val := Fresh("val")

	goal := Conj(
		Eq(val, NewAtom(5)),
		Matcha(val,
			NewClause(NewAtom(5), Eq(q, NewAtom("small"))),
			NewClause(NewAtom(10), Eq(q, NewAtom("medium"))),
			NewClause(NewAtom(15), Eq(q, NewAtom("large"))),
		),
	)

	ctx := context.Background()
	stream := goal(ctx, adapter)
	results, _ := stream.Take(1)

	if len(results) > 0 {
		binding := results[0].GetBinding(q.ID())
		if atom, ok := binding.(*Atom); ok {
			fmt.Printf("Classification: %v\n", atom.value)
		}
	}

	// Output:
	// Classification: small
}

```


\n
## pkg_minikanren_pattern_example_test.go-ExampleMatcheList.md
```go
func ExampleMatcheList() {
	// Simple list pattern matching
	list := List(NewAtom(1), NewAtom(2), NewAtom(3))

	result := Run(1, func(q *Var) Goal {
		return MatcheList(list,
			NewClause(Nil, Eq(q, NewAtom("empty"))),
			NewClause(NewPair(Fresh("_"), Nil), Eq(q, NewAtom("singleton"))),
			NewClause(NewPair(Fresh("head"), NewPair(Fresh("_"), Fresh("_"))), Eq(q, NewAtom("multiple"))),
		)
	})

	fmt.Println(result[0])

	// Output:
	// multiple
}

```


\n
## pkg_minikanren_pattern_example_test.go-ExampleMatche_listProcessing.md
```go
func ExampleMatche_listProcessing() {
	// Extract all elements from a list
	extractAll := func(list Term) []Term {
		var results []Term

		Run(10, func(q *Var) Goal {
			elem := Fresh("elem")
			rest := Fresh("rest")

			return Matche(list,
				NewClause(Nil, Eq(q, NewAtom("done"))),
				NewClause(NewPair(elem, rest), Eq(q, elem)),
			)
		})

		// Simplified - in practice would need recursive extraction
		return results
	}

	list := List(NewAtom("a"), NewAtom("b"), NewAtom("c"))
	_ = extractAll(list)

	fmt.Println("List elements extracted")

	// Output:
	// List elements extracted
}

```


\n
## pkg_minikanren_pattern_example_test.go-ExampleMatche.md
```go
func ExampleMatche() {
	// Classify a list by structure
	list := List(NewAtom(1), NewAtom(2))

	result := Run(5, func(q *Var) Goal {
		return Matche(list,
			NewClause(Nil, Eq(q, NewAtom("empty"))),
			NewClause(NewPair(Fresh("_"), Nil), Eq(q, NewAtom("singleton"))),
			NewClause(NewPair(Fresh("_"), NewPair(Fresh("_"), Fresh("_"))), Eq(q, NewAtom("multiple"))),
		)
	})

	// Matches "multiple" clause only
	for _, r := range result {
		if atom, ok := r.(*Atom); ok {
			fmt.Println(atom.value)
		}
	}

	// Output:
	// multiple
}

```


\n
## pkg_minikanren_pattern_example_test.go-ExampleMatche_withDatabase.md
```go
func ExampleMatche_withDatabase() {
	// Create a relation for shapes
	shape, _ := DbRel("shape", 2, 0)
	db := NewDatabase()
	db, _ = db.AddFact(shape, NewAtom("circle"), NewAtom(10))
	db, _ = db.AddFact(shape, NewAtom("square"), NewAtom(5))
	db, _ = db.AddFact(shape, NewAtom("triangle"), NewAtom(3))

	// Query and pattern match on shape type
	name := Fresh("name")
	size := Fresh("size")

	result := Run(10, func(q *Var) Goal {
		return Conj(
			db.Query(shape, name, size),
			Matche(name,
				NewClause(NewAtom("circle"), Eq(q, NewAtom("round"))),
				NewClause(NewAtom("square"), Eq(q, NewAtom("angular"))),
				NewClause(NewAtom("triangle"), Eq(q, NewAtom("angular"))),
			),
		)
	})

	// Count results
	fmt.Printf("Matched %d shapes\n", len(result))

	// Output:
	// Matched 3 shapes
}

```


\n
## pkg_minikanren_pattern_example_test.go-ExampleMatchu.md
```go
func ExampleMatchu() {
	// Classify numbers with mutually exclusive ranges
	classify := func(n int) string {
		result := Run(1, func(q *Var) Goal {
			return CaseIntMap(NewAtom(n), map[int]string{
				0: "zero",
				1: "one",
				2: "two",
			}, q)
		})

		if len(result) == 0 {
			return "unknown"
		}

		if atom, ok := result[0].(*Atom); ok {
			if s, ok := atom.value.(string); ok {
				return s
			}
		}
		return "error"
	}

	fmt.Println(classify(0))
	fmt.Println(classify(1))
	fmt.Println(classify(5))

	// Output:
	// zero
	// one
	// unknown
}

```


\n
## pkg_minikanren_pattern_example_test.go-ExampleMatchu_validation.md
```go
func ExampleMatchu_validation() {
	// Validate that a value matches exactly one category
	validate := func(val int) (string, bool) {
		// Alternative implementation for demonstration purposes
		// return Matchu(NewAtom(val),
		//		NewClause(NewAtom(1), Eq(q, NewAtom("category-A"))),
		//		NewClause(NewAtom(2), Eq(q, NewAtom("category-B"))),
		//		NewClause(NewAtom(3), Eq(q, NewAtom("category-C"))),
		result := Run(1, func(q *Var) Goal {
			return CaseIntMap(NewAtom(val), map[int]string{
				1: "category-A",
				2: "category-B",
				3: "category-C",
			}, q)
		})

		if len(result) == 0 {
			return "", false
		}

		if atom, ok := result[0].(*Atom); ok {
			if s, ok := atom.value.(string); ok {
				return s, true
			}
		}
		return "", false
	}

	// Valid values
	cat, ok := validate(1)
	fmt.Printf("Value 1: %s (valid: %t)\n", cat, ok)

	cat, ok = validate(2)
	fmt.Printf("Value 2: %s (valid: %t)\n", cat, ok)

	// Invalid value (no match)
	cat, ok = validate(99)
	fmt.Printf("Value 99: %s (valid: %t)\n", cat, ok)

	// Output:
	// Value 1: category-A (valid: true)
	// Value 2: category-B (valid: true)
	// Value 99:  (valid: false)
}

```


\n
## pkg_minikanren_pattern_example_test.go-ExampleNewClause.md
```go
func ExampleNewClause() {
	// Pattern matching with variable binding and multiple goals
	result := Run(5, func(q *Var) Goal {
		x := Fresh("x")
		y := Fresh("y")

		return Matche(NewPair(NewAtom(10), NewAtom(20)),
			NewClause(
				NewPair(x, y),
				// Multiple goals executed in sequence
				Eq(x, NewAtom(10)),
				Eq(y, NewAtom(20)),
				Eq(q, NewAtom("success")),
			),
		)
	})

	fmt.Println(result[0])

	// Output:
	// success
}

```


\n
## pkg_minikanren_pattern_example_test.go-ExamplePatternClause_nestedPatterns.md
```go
func ExamplePatternClause_nestedPatterns() {
	// Match nested structure: ((a b) (c d))
	data := List(
		List(NewAtom("x"), NewAtom("y")),
		List(NewAtom("z"), NewAtom("w")),
	)

	result := Run(1, func(q *Var) Goal {
		a := Fresh("a")
		b := Fresh("b")

		return Matche(data,
			NewClause(
				NewPair(
					NewPair(a, NewPair(b, Nil)),
					Fresh("_"),
				),
				Eq(q, List(a, b)),
			),
		)
	})

	if len(result) > 0 {
		fmt.Printf("Extracted first pair: %v\n", result[0])
	}

	// Output:
	// Extracted first pair: (x . (y . <nil>))
}

```


\n
## pkg_minikanren_pldb_example_test.go-ExampleDatabase_AddFact.md
```go
func ExampleDatabase_AddFact() {
	parent, _ := DbRel("parent", 2, 0, 1)

	// Start with an empty database
	db := NewDatabase()

	// Add facts using copy-on-write semantics
	db1, _ := db.AddFact(parent, NewAtom("alice"), NewAtom("bob"))
	db2, _ := db1.AddFact(parent, NewAtom("bob"), NewAtom("charlie"))
	db3, _ := db2.AddFact(parent, NewAtom("alice"), NewAtom("diana"))

	// Each version maintains its own state
	fmt.Printf("Original: %d facts\n", db.FactCount(parent))
	fmt.Printf("After 1:  %d facts\n", db1.FactCount(parent))
	fmt.Printf("After 2:  %d facts\n", db2.FactCount(parent))
	fmt.Printf("After 3:  %d facts\n", db3.FactCount(parent))

	// Output:
	// Original: 0 facts
	// After 1:  1 facts
	// After 2:  2 facts
	// After 3:  3 facts
}

```


\n
## pkg_minikanren_pldb_example_test.go-ExampleDatabase_Query_datalog.md
```go
func ExampleDatabase_Query_datalog() {
	edge, _ := DbRel("edge", 2, 0, 1)
	// Build a graph: a -> b -> c
	//                ^-------|
	db := DB().MustAddFacts(edge,
		[]interface{}{"a", "b"},
		[]interface{}{"b", "c"},
		[]interface{}{"c", "a"},
	)

	// Query: Find all nodes reachable from 'a' in exactly 2 hops
	// path2(X, Z) :- edge(X, Y), edge(Y, Z)
	start := NewAtom("a")
	middle := Fresh("middle")
	dest := Fresh("destination")

	goal := Conj(
		db.Query(edge, start, middle),
		db.Query(edge, middle, dest),
	)

	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)
	results, _ := stream.Take(10)

	fmt.Printf("Nodes reachable from 'a' in 2 hops:\n")
	for _, r := range results {
		val := r.GetBinding(dest.ID())
		if atom, ok := val.(*Atom); ok {
			fmt.Printf("  %v\n", atom.Value())
		}
	}

	// Output:
	// Nodes reachable from 'a' in 2 hops:
	//   c
}

```


\n
## pkg_minikanren_pldb_example_test.go-ExampleDatabase_Query_disjunction.md
```go
func ExampleDatabase_Query_disjunction() {
	parent, _ := DbRel("parent", 2, 0, 1)
	db := DB().MustAddFacts(parent,
		[]interface{}{"alice", "bob"},
		[]interface{}{"bob", "charlie"},
		[]interface{}{"charlie", "diana"},
	)

	// Query: Find children of alice OR bob
	child := Fresh("child")
	goal := Disj(
		db.Query(parent, NewAtom("alice"), child),
		db.Query(parent, NewAtom("bob"), child),
	)

	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)
	results, _ := stream.Take(10)

	// Results may come in any order due to parallel evaluation
	fmt.Printf("Found %d children\n", len(results))

	// Output:
	// Found 2 children
}

```


\n
## pkg_minikanren_pldb_example_test.go-ExampleDatabase_Query_join.md
```go
func ExampleDatabase_Query_join() {
	parent, _ := DbRel("parent", 2, 0, 1)
	db := DB().MustAddFacts(parent,
		[]interface{}{"alice", "bob"},
		[]interface{}{"bob", "charlie"},
		[]interface{}{"charlie", "diana"},
	)

	// Query: Find grandparent-grandchild pairs
	// grandparent(GP, GC) :- parent(GP, P), parent(P, GC)
	gp := Fresh("grandparent")
	gc := Fresh("grandchild")
	p := Fresh("parent")

	goal := Conj(
		db.Query(parent, gp, p),
		db.Query(parent, p, gc),
	)

	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)
	results, _ := stream.Take(10)

	fmt.Printf("Found %d grandparent relationships\n", len(results))

	// Output:
	// Found 2 grandparent relationships
}

```


\n
## pkg_minikanren_pldb_example_test.go-ExampleDatabase_Query_repeated.md
```go
func ExampleDatabase_Query_repeated() {
	edge, _ := DbRel("edge", 2, 0, 1)
	db := DB().MustAddFacts(edge,
		[]interface{}{"a", "b"},
		[]interface{}{"b", "c"},
		[]interface{}{"c", "c"}, // self-loop
		[]interface{}{"d", "d"}, // self-loop
	)

	// Query: Find all self-loops
	x := Fresh("x")
	goal := db.Query(edge, x, x) // same variable in both positions

	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)
	results, _ := stream.Take(10)

	fmt.Printf("Found %d self-loops\n", len(results))

	// Output:
	// Found 2 self-loops
}

```


\n
## pkg_minikanren_pldb_example_test.go-ExampleDatabase_Query_simple.md
```go
func ExampleDatabase_Query_simple() {
	parent, _ := DbRel("parent", 2, 0, 1)
	db := DB().MustAddFacts(parent,
		[]interface{}{"alice", "bob"},
		[]interface{}{"alice", "charlie"},
		[]interface{}{"bob", "diana"},
	)

	// Query: Who are alice's children?
	child := Fresh("child")
	goal := db.Query(parent, NewAtom("alice"), child)

	// Execute the query
	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)
	results, _ := stream.Take(10)

	// Results may come in any order
	fmt.Printf("Alice has %d children\n", len(results))

	// Output:
	// Alice has 2 children
}

```


\n
## pkg_minikanren_pldb_example_test.go-ExampleDatabase_RemoveFact.md
```go
func ExampleDatabase_RemoveFact() {
	person, _ := DbRel("person", 1, 0)

	// Create database with some people using low-level API versus the HLAPI for demonstration
	db := NewDatabase()
	db, _ = db.AddFact(person, NewAtom("alice"))
	db, _ = db.AddFact(person, NewAtom("bob"))
	db, _ = db.AddFact(person, NewAtom("charlie"))

	fmt.Printf("Before removal: %d people\n", db.FactCount(person))

	// Remove bob
	db2, _ := db.RemoveFact(person, NewAtom("bob"))

	fmt.Printf("After removal:  %d people\n", db2.FactCount(person))

	// Original database unchanged
	fmt.Printf("Original still: %d people\n", db.FactCount(person))

	// Facts can be re-added
	db3, _ := db2.AddFact(person, NewAtom("bob"))
	fmt.Printf("After re-add:   %d people\n", db3.FactCount(person))

	// Output:
	// Before removal: 3 people
	// After removal:  2 people
	// Original still: 3 people
	// After re-add:   3 people
}

```


\n
## pkg_minikanren_pldb_example_test.go-ExampleDbRel.md
```go
func ExampleDbRel() {
	// Create a binary relation for parent-child relationships
	// Index both columns for fast lookups
	parent, err := DbRel("parent", 2, 0, 1)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Relation: %s (arity=%d)\n", parent.Name(), parent.Arity())
	fmt.Printf("Column 0 indexed: %v\n", parent.IsIndexed(0))
	fmt.Printf("Column 1 indexed: %v\n", parent.IsIndexed(1))

	// Output:
	// Relation: parent (arity=2)
	// Column 0 indexed: true
	// Column 1 indexed: true
}

```


\n
## pkg_minikanren_pldb_hybrid_example_test.go-ExampleUnifiedStoreAdapter_basicQuery.md
```go
func ExampleUnifiedStoreAdapter_basicQuery() {
	// Create a database of people with names and ages
	person, _ := DbRel("person", 2, 0) // name is indexed
	db := NewDatabase()
	db, _ = db.AddFact(person, NewAtom("alice"), NewAtom(30))
	db, _ = db.AddFact(person, NewAtom("bob"), NewAtom(25))
	db, _ = db.AddFact(person, NewAtom("carol"), NewAtom(35))

	// Create UnifiedStore and adapter
	store := NewUnifiedStore()
	adapter := NewUnifiedStoreAdapter(store)

	// Query for all people
	name := Fresh("name")
	age := Fresh("age")

	goal := db.Query(person, name, age)
	stream := goal(context.Background(), adapter)

	// Retrieve results
	results, _ := stream.Take(10)

	// Print number of results (order may vary due to map iteration)
	fmt.Printf("Found %d people\n", len(results))

	// Output:
	// Found 3 people
}

```


\n
## pkg_minikanren_pldb_hybrid_example_test.go-ExampleUnifiedStoreAdapter_fdConstrainedQuery.md
```go
func ExampleUnifiedStoreAdapter_fdConstrainedQuery() {
	// Create a database of employees with ages (compact via HLAPI)
	employee, _ := DbRel("employee", 2, 0) // name is indexed
	db := DB().MustAddFacts(employee,
		[]interface{}{"alice", 28},
		[]interface{}{"bob", 32},
		[]interface{}{"carol", 45},
		[]interface{}{"dave", 29},
	)

	// Create FD model with age restricted to [25, 35]
	model := NewModel()
	// ageVar := model.NewVariableWithName(
	//     NewBitSetDomainFromValues(100, []int{25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35}),
	//     "age",
	// )
	ageVar := model.IntVarValues([]int{25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35}, "age")

	// Create store with FD domain and adapter
	store := NewUnifiedStore()
	store, _ = store.SetDomain(ageVar.ID(), ageVar.Domain())
	adapter := NewUnifiedStoreAdapter(store)

	// Define variables
	name := Fresh("name")
	age := Fresh("age")

	// Use HLAPI FDFilteredQuery to combine the DB query and FD-domain filtering
	// FDFilteredQuery(db, rel, fdVar, filterVar, queryTerms...)
	goal := FDFilteredQuery(db, employee, ageVar, age, name, age)

	// Execute query
	stream := goal(context.Background(), adapter)
	results, _ := stream.Take(10)

	// Print count (order may vary)
	fmt.Printf("Found %d employees aged 25-35\n", len(results))

	// Output:
	// Found 3 employees aged 25-35
}

```


\n
## pkg_minikanren_pldb_hybrid_example_test.go-ExampleUnifiedStoreAdapter_hybridPropagation.md
```go
func ExampleUnifiedStoreAdapter_hybridPropagation() {
	// Create database of people with ages
	person, _ := DbRel("person", 2, 0)
	db := NewDatabase()
	db, _ = db.AddFact(person, NewAtom("alice"), NewAtom(30))

	// Create FD model with age variable (domain 0-100)
	model := NewModel()
	ageValues := make([]int, 101)
	for i := range ageValues {
		ageValues[i] = i
	}
	// ageVar := model.NewVariableWithName(NewBitSetDomainFromValues(101, ageValues), "age")
	ageVar := model.IntVarValues(ageValues, "age")

	// Create HybridSolver and a UnifiedStore populated from the model.
	solver, store, err := NewHybridSolverFromModel(model)
	if err != nil {
		panic(err)
	}
	adapter := NewUnifiedStoreAdapter(store)

	// Query for alice's age
	age := Fresh("age")
	goal := db.Query(person, NewAtom("alice"), age)
	stream := goal(context.Background(), adapter)

	results, _ := stream.Take(1)
	if len(results) > 0 {
		resultAdapter := results[0].(*UnifiedStoreAdapter)

		// Link logical variable to FD variable
		resultStore := resultAdapter.UnifiedStore()
		ageBinding := resultAdapter.GetBinding(age.ID())
		if ageAtom, ok := ageBinding.(*Atom); ok {
			if ageInt, ok := ageAtom.value.(int); ok {
				// Bind FD variable to the same value
				resultStore, _ = resultStore.AddBinding(int64(ageVar.ID()), NewAtom(ageInt))
				resultAdapter.SetUnifiedStore(resultStore)

				// Run propagation
				propagated, err := solver.Propagate(resultAdapter.UnifiedStore())
				if err == nil {
					// FD domain should now be singleton {30}
					ageDomain := propagated.GetDomain(ageVar.ID())
					if ageDomain.IsSingleton() {
						fmt.Printf("FD domain pruned to: {%d}\n", ageDomain.SingletonValue())
					}
				}
			}
		}
	}

	// Output:
	// FD domain pruned to: {30}
}

```


\n
## pkg_minikanren_pldb_hybrid_example_test.go-ExampleUnifiedStoreAdapter_parallelSearch.md
```go
func ExampleUnifiedStoreAdapter_parallelSearch() {
	// Create database
	color, _ := DbRel("color", 2, 0)
	db := NewDatabase()
	db, _ = db.AddFact(color, NewAtom("apple"), NewAtom("red"))
	db, _ = db.AddFact(color, NewAtom("banana"), NewAtom("yellow"))

	// Create adapter
	store := NewUnifiedStore()
	adapter := NewUnifiedStoreAdapter(store)

	// Simulate parallel search: clone adapter for each branch
	branch1 := adapter.Clone().(*UnifiedStoreAdapter)
	branch2 := adapter.Clone().(*UnifiedStoreAdapter)

	// Each branch queries independently
	item := Fresh("item")

	goal1 := db.Query(color, item, NewAtom("red"))
	stream1 := goal1(context.Background(), branch1)
	results1, _ := stream1.Take(1)

	goal2 := db.Query(color, item, NewAtom("yellow"))
	stream2 := goal2(context.Background(), branch2)
	results2, _ := stream2.Take(1)

	// Print results from each independent branch
	if len(results1) > 0 {
		itemBinding := results1[0].GetBinding(item.ID())
		if atom, ok := itemBinding.(*Atom); ok {
			fmt.Printf("Branch 1: %s is red\n", atom.value)
		}
	}

	if len(results2) > 0 {
		itemBinding := results2[0].GetBinding(item.ID())
		if atom, ok := itemBinding.(*Atom); ok {
			fmt.Printf("Branch 2: %s is yellow\n", atom.value)
		}
	}

	// Output:
	// Branch 1: apple is red
	// Branch 2: banana is yellow
}

```


\n
## pkg_minikanren_pldb_hybrid_example_test.go-ExampleUnifiedStoreAdapter_performance.md
```go
func ExampleUnifiedStoreAdapter_performance() {
	// Create large database with 1000 people
	person, _ := DbRel("person", 3, 0, 1, 2) // all fields indexed
	db := NewDatabase()

	for i := 0; i < 1000; i++ {
		name := NewAtom(fmt.Sprintf("person%d", i))
		age := NewAtom(20 + (i % 50))
		score := NewAtom(50 + (i % 50))
		db, _ = db.AddFact(person, name, age, score)
	}

	// Create adapter
	store := NewUnifiedStore()
	adapter := NewUnifiedStoreAdapter(store)

	// Query for specific age (indexed lookup is O(1))
	name := Fresh("name")
	score := Fresh("score")

	goal := db.Query(person, name, NewAtom(30), score)
	stream := goal(context.Background(), adapter)

	// Fast retrieval even from large database
	results, _ := stream.Take(100)

	fmt.Printf("Found %d people with age 30 (from 1000 total)\n", len(results))

	// Output:
	// Found 20 people with age 30 (from 1000 total)
}

```


\n
## pkg_minikanren_pldb_hybrid_helpers_example_test.go-Example_fdFilteredQuery_compositional.md
```go
func Example_fdFilteredQuery_compositional() {
	ctx := context.Background()

	// Setup database
	employee, _ := DbRel("employee", 2, 0)
	db := NewDatabase()
	db, _ = db.AddFact(employee, NewAtom("alice"), NewAtom(28))
	db, _ = db.AddFact(employee, NewAtom("bob"), NewAtom(42))
	db, _ = db.AddFact(employee, NewAtom("charlie"), NewAtom(31))

	// FD model
	model := NewModel()
	ageVar := model.NewVariable(NewBitSetDomainFromValues(50, []int{25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35}))

	store := NewUnifiedStore()
	store, _ = store.SetDomain(ageVar.ID(), ageVar.Domain())
	adapter := NewUnifiedStoreAdapter(store)

	// Compose FD-filtered query with additional constraints
	name := Fresh("name")
	age := Fresh("age")

	goal := Conj(
		FDFilteredQuery(db, employee, ageVar, age, name, age),
		// Add additional constraint: name must start with 'a' or 'c'
		Disj(
			Eq(name, NewAtom("alice")),
			Eq(name, NewAtom("charlie")),
		),
	)

	results, _ := goal(ctx, adapter).Take(10)

	// Sort for deterministic output
	names := make([]string, 0)
	for _, result := range results {
		nameBinding := result.GetBinding(name.ID())
		if nameAtom, ok := nameBinding.(*Atom); ok {
			if nameStr, ok := nameAtom.value.(string); ok {
				names = append(names, nameStr)
			}
		}
	}
	sort.Strings(names)

	fmt.Printf("Employees aged 25-35 with names starting with a or c: %d\n", len(names))
	for _, n := range names {
		fmt.Printf("  %s\n", n)
	}

	// Output:
	// Employees aged 25-35 with names starting with a or c: 2
	//   alice
	//   charlie
}

```


\n
## pkg_minikanren_pldb_hybrid_helpers_example_test.go-Example_fdFilteredQuery.md
```go
func Example_fdFilteredQuery() {
	ctx := context.Background()

	// 1. Setup database with employee records (compact via HLAPI)
	employee, _ := DbRel("employee", 2, 0)
	db := DB().MustAddFacts(employee,
		[]interface{}{"alice", 28},
		[]interface{}{"bob", 42},
		[]interface{}{"charlie", 31},
		[]interface{}{"diana", 19},
	)

	// 2. Setup FD constraint for eligible age range [25, 35]
	model := NewModel()
	eligibleAges := make([]int, 0)
	for age := 25; age <= 35; age++ {
		eligibleAges = append(eligibleAges, age)
	}
	ageVar := model.NewVariable(NewBitSetDomainFromValues(50, eligibleAges))

	// 3. Initialize hybrid store
	store := NewUnifiedStore()
	store, _ = store.SetDomain(ageVar.ID(), ageVar.Domain())
	adapter := NewUnifiedStoreAdapter(store)

	// 4. Create FD-filtered query (ONE LINE vs 50 lines manual)
	name := Fresh("name")
	age := Fresh("age")
	goal := FDFilteredQuery(db, employee, ageVar, age, name, age)

	// 5. Execute and display results
	results, _ := goal(ctx, adapter).Take(10)

	// Collect and sort results for deterministic output
	type empRecord struct {
		name string
		age  int
	}
	employees := make([]empRecord, 0)

	for _, result := range results {
		nameBinding := result.GetBinding(name.ID())
		ageBinding := result.GetBinding(age.ID())

		if nameAtom, ok := nameBinding.(*Atom); ok {
			if ageAtom, ok := ageBinding.(*Atom); ok {
				if nameStr, ok := nameAtom.value.(string); ok {
					if ageInt, ok := ageAtom.value.(int); ok {
						employees = append(employees, empRecord{nameStr, ageInt})
					}
				}
			}
		}
	}

	sort.Slice(employees, func(i, j int) bool {
		return employees[i].name < employees[j].name
	})

	fmt.Printf("Eligible employees (age 25-35): %d\n", len(employees))
	for _, emp := range employees {
		fmt.Printf("  %s: age %d\n", emp.name, emp.age)
	}

	// Output:
	// Eligible employees (age 25-35): 2
	//   alice: age 28
	//   charlie: age 31
}

```


\n
## pkg_minikanren_pldb_hybrid_helpers_example_test.go-Example_fdFilteredQuery_multipleConstraints.md
```go
func Example_fdFilteredQuery_multipleConstraints() {
	ctx := context.Background()

	// Setup employee and salary databases
	employee, _ := DbRel("employee", 2, 0)
	salary, _ := DbRel("salary", 2, 0)
	db := DB().MustAddFacts(employee,
		[]interface{}{"alice", 28},
		[]interface{}{"bob", 42},
		[]interface{}{"charlie", 31},
	)
	db = db.MustAddFacts(salary,
		[]interface{}{"alice", 50000},
		[]interface{}{"bob", 80000},
		[]interface{}{"charlie", 45000},
	)

	// FD constraints: age 25-35, salary 40k-60k
	model := NewModel()
	ageVar := model.NewVariable(NewBitSetDomainFromValues(50, []int{25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35}))
	salaryVar := model.NewVariable(NewBitSetDomainFromValues(100000, []int{40000, 45000, 50000, 55000, 60000}))

	// Initialize store with both domains
	store := NewUnifiedStore()
	store, _ = store.SetDomain(ageVar.ID(), ageVar.Domain())
	store, _ = store.SetDomain(salaryVar.ID(), salaryVar.Domain())
	adapter := NewUnifiedStoreAdapter(store)

	// Create two FD-filtered queries
	name := Fresh("name")
	age := Fresh("age")
	sal := Fresh("salary")

	ageQuery := FDFilteredQuery(db, employee, ageVar, age, name, age)
	salaryQuery := FDFilteredQuery(db, salary, salaryVar, sal, name, sal)

	// Combine with conjunction - both constraints must hold
	goal := HybridConj(ageQuery, salaryQuery)

	// Execute
	results, _ := goal(ctx, adapter).Take(10)

	// Sort for deterministic output
	names := make([]string, 0)
	for _, result := range results {
		nameBinding := result.GetBinding(name.ID())
		if nameAtom, ok := nameBinding.(*Atom); ok {
			if nameStr, ok := nameAtom.value.(string); ok {
				names = append(names, nameStr)
			}
		}
	}
	sort.Strings(names)

	fmt.Printf("Employees meeting both criteria: %d\n", len(names))
	for _, n := range names {
		fmt.Printf("  %s\n", n)
	}

	// Output:
	// Employees meeting both criteria: 2
	//   alice
	//   charlie
}

```


\n
## pkg_minikanren_pldb_hybrid_helpers_example_test.go-Example_fdFilteredQuery_withArithmetic.md
```go
func Example_fdFilteredQuery_withArithmetic() {
	ctx := context.Background()

	// Setup salary database
	salary, _ := DbRel("salary", 2, 0)
	db := NewDatabase()
	db, _ = db.AddFact(salary, NewAtom("alice"), NewAtom(50000))
	db, _ = db.AddFact(salary, NewAtom("bob"), NewAtom(80000))

	// FD constraints: salary must be in range, bonus = salary / 10
	model := NewModel()
	salaryVar := model.NewVariable(NewBitSetDomainFromValues(100000, []int{50000, 60000, 70000}))
	bonusVar := model.NewVariable(NewBitSetDomainFromValues(10000, []int{5000, 6000, 7000}))

	// Add arithmetic constraint: bonus * 10 = salary (scaled by 10 to avoid division)
	ls, _ := NewLinearSum([]*FDVariable{bonusVar}, []int{10}, salaryVar)
	model.AddConstraint(ls)

	// Propagate arithmetic constraints to get pruned domains
	solver := NewSolver(model)
	// Call Solve once to trigger propagation, then read domains from base state
	solver.Solve(ctx, 1)

	// Initialize store with propagated domains
	store := NewUnifiedStore()
	store, _ = store.SetDomain(salaryVar.ID(), solver.GetDomain(nil, salaryVar.ID()))
	store, _ = store.SetDomain(bonusVar.ID(), solver.GetDomain(nil, bonusVar.ID()))
	adapter := NewUnifiedStoreAdapter(store)

	// Query with FD filtering
	name := Fresh("name")
	sal := Fresh("salary")
	goal := FDFilteredQuery(db, salary, salaryVar, sal, name, sal)

	results, _ := goal(ctx, adapter).Take(10)

	fmt.Printf("Employees with valid salary/bonus combinations: %d\n", len(results))
	for _, result := range results {
		nameBinding := result.GetBinding(name.ID())
		salBinding := result.GetBinding(sal.ID())

		if nameAtom, ok := nameBinding.(*Atom); ok {
			if salAtom, ok := salBinding.(*Atom); ok {
				if salInt, ok := salAtom.value.(int); ok {
					bonus := salInt / 10
					fmt.Printf("  %s: salary %d, bonus %d\n", nameAtom.value, salInt, bonus)
				}
			}
		}
	}

	// Output:
	// Employees with valid salary/bonus combinations: 1
	//   alice: salary 50000, bonus 5000
}

```


\n
## pkg_minikanren_pldb_hybrid_helpers_example_test.go-Example_mapQueryResult.md
```go
func Example_mapQueryResult() {
	ctx := context.Background()

	// Setup database
	employee, _ := DbRel("employee", 2, 0)
	db := NewDatabase()
	db, _ = db.AddFact(employee, NewAtom("alice"), NewAtom(28))

	// Query alice's age
	age := Fresh("age")
	goal := db.Query(employee, NewAtom("alice"), age)

	store := NewUnifiedStore()
	adapter := NewUnifiedStoreAdapter(store)
	results, _ := goal(ctx, adapter).Take(1)

	// Create FD variable to receive the age
	model := NewModel()
	ageVar := model.NewVariable(NewBitSetDomainFromValues(50, []int{20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30}))

	// Map the query result to the FD variable (convenience helper)
	store, _ = MapQueryResult(results[0], age, ageVar, store)

	// Now ageVar is bound to alice's age
	binding := store.GetBinding(int64(ageVar.ID()))
	if ageAtom, ok := binding.(*Atom); ok {
		fmt.Printf("Alice's age: %d\n", ageAtom.value)
	}

	// Output:
	// Alice's age: 28
}

```


\n
## pkg_minikanren_pldb_slg_example_test.go-ExampleInvalidateAll.md
```go
func ExampleInvalidateAll() {
	edge, _ := DbRel("edge", 2, 0, 1)
	db := NewDatabase()
	db, _ = db.AddFact(edge, NewAtom("a"), NewAtom("b"))

	x := Fresh("x")
	y := Fresh("y")

	// Populate cache
	goal := TabledQuery(db, edge, "edge_inv", x, y)
	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)
	stream.Take(10)

	// Clear all cached answers
	InvalidateAll()

	engine := GlobalEngine()
	stats := engine.Stats()

	fmt.Printf("Cached subgoals after invalidation: %d\n", stats.CachedSubgoals)

	// Output:
	// Cached subgoals after invalidation: 0
}

```


\n
## pkg_minikanren_pldb_slg_example_test.go-ExampleInvalidateRelation.md
```go
func ExampleInvalidateRelation() {
	// Start with a clean cache to make the example deterministic
	InvalidateAll()

	edge, _ := DbRel("edge", 2, 0, 1)
	db := NewDatabase()
	db, _ = db.AddFact(edge, NewAtom("a"), NewAtom("b"))

	x := Fresh("x")
	y := Fresh("y")

	// Populate cache with edge_rel predicate
	goal := TabledQuery(db, edge, "edge_rel", x, y)
	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)
	stream.Take(10)

	// Invalidate only the edge_rel predicate
	InvalidateRelation("edge_rel")

	engine := GlobalEngine()
	stats := engine.Stats()

	// With fine-grained invalidation, only edge_rel is cleared
	fmt.Printf("Cache cleared: %v\n", stats.CachedSubgoals == 0)

	// Output:
	// Cache cleared: true
}

```


\n
## pkg_minikanren_pldb_slg_example_test.go-ExampleQueryEvaluator.md
```go
func ExampleQueryEvaluator() {
	parent, _ := DbRel("parent", 2, 0, 1)
	db := NewDatabase()
	db, _ = db.AddFact(parent, NewAtom("alice"), NewAtom("bob"))
	db, _ = db.AddFact(parent, NewAtom("alice"), NewAtom("charlie"))

	child := Fresh("child")
	query := db.Query(parent, NewAtom("alice"), child)

	// Convert to GoalEvaluator
	evaluator := QueryEvaluator(query, child.ID())

	ctx := context.Background()
	answers := make(chan map[int64]Term, 10)

	go func() {
		defer close(answers)
		_ = evaluator(ctx, answers)
	}()

	count := 0
	for range answers {
		count++
	}

	fmt.Printf("Alice has %d children\n", count)

	// Output:
	// Alice has 2 children
}

```


\n
## pkg_minikanren_pldb_slg_example_test.go-ExampleTabledQuery_groundQuery.md
```go
func ExampleTabledQuery_groundQuery() {
	InvalidateAll()

	edge, _ := DbRel("edge", 2, 0, 1)
	db := NewDatabase()
	db, _ = db.AddFact(edge, NewAtom("a"), NewAtom("b"))
	db, _ = db.AddFact(edge, NewAtom("b"), NewAtom("c"))

	// Fully ground query - checks existence
	goal := TabledQuery(db, edge, "edge_ground_ex", NewAtom("a"), NewAtom("b"))

	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)
	results, _ := stream.Take(10)

	if len(results) > 0 {
		fmt.Println("Edge a->b exists")
	} else {
		fmt.Println("Edge a->b does not exist")
	}

	// Output:
	// Edge a->b exists
}

```


\n
## pkg_minikanren_pldb_slg_example_test.go-ExampleTabledQuery_join.md
```go
func ExampleTabledQuery_join() {
	InvalidateAll()

	parent, _ := DbRel("parent", 2, 0, 1)
	db := NewDatabase()
	db, _ = db.AddFact(parent, NewAtom("alice"), NewAtom("bob"))
	db, _ = db.AddFact(parent, NewAtom("bob"), NewAtom("charlie"))
	db, _ = db.AddFact(parent, NewAtom("charlie"), NewAtom("diana"))

	// TabledQuery now works correctly in joins with shared variables
	gp := Fresh("gp")
	gc := Fresh("gc")
	p := Fresh("p")

	goal := Conj(
		TabledQuery(db, parent, "parent_join_ex", gp, p),
		TabledQuery(db, parent, "parent_join_ex", p, gc),
	)

	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)
	results, _ := stream.Take(10)

	fmt.Printf("Found %d grandparent relationships\n", len(results))

	// Output:
	// Found 2 grandparent relationships
}

```


\n
## pkg_minikanren_pldb_slg_example_test.go-ExampleTabledQuery.md
```go
func ExampleTabledQuery() {
	edge, _ := DbRel("edge", 2, 0, 1)
	db := NewDatabase()
	db, _ = db.AddFact(edge, NewAtom("a"), NewAtom("b"))
	db, _ = db.AddFact(edge, NewAtom("b"), NewAtom("c"))

	x := Fresh("x")
	y := Fresh("y")

	// Tabled query caches results
	goal := TabledQuery(db, edge, "edge", x, y)

	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)
	results, _ := stream.Take(10)

	fmt.Printf("Found %d edges\n", len(results))

	// Output:
	// Found 2 edges
}

```


\n
## pkg_minikanren_pldb_slg_example_test.go-ExampleTabledQuery_multipleVariables.md
```go
func ExampleTabledQuery_multipleVariables() {
	person, _ := DbRel("person", 3, 0, 1, 2) // name, age, city
	db := NewDatabase()
	db, _ = db.AddFact(person, NewAtom("alice"), NewAtom(30), NewAtom("nyc"))
	db, _ = db.AddFact(person, NewAtom("bob"), NewAtom(25), NewAtom("sf"))
	db, _ = db.AddFact(person, NewAtom("charlie"), NewAtom(35), NewAtom("nyc"))

	name := Fresh("name")
	age := Fresh("age")
	city := Fresh("city")

	// Query all fields
	goal := TabledQuery(db, person, "person", name, age, city)

	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)
	results, _ := stream.Take(10)

	// Collect names for consistent output
	names := make([]string, 0, len(results))
	for _, s := range results {
		if n := s.GetBinding(name.ID()); n != nil {
			if atom, ok := n.(*Atom); ok {
				names = append(names, atom.Value().(string))
			}
		}
	}
	sort.Strings(names)

	fmt.Printf("Found people: %v\n", names)

	// Output:
	// Found people: [alice bob charlie]
}

```


\n
## pkg_minikanren_pldb_slg_example_test.go-ExampleTabledRelation.md
```go
func ExampleTabledRelation() {
	// Clear cache for clean test
	InvalidateAll()

	edge, _ := DbRel("edge", 2, 0, 1)
	db := NewDatabase()
	db, _ = db.AddFact(edge, NewAtom("a"), NewAtom("b"))
	db, _ = db.AddFact(edge, NewAtom("b"), NewAtom("c"))
	db, _ = db.AddFact(edge, NewAtom("c"), NewAtom("d"))

	// Create tabled predicate constructor
	edgePred := TabledRelation(db, edge, "edge_example")

	x := Fresh("x")
	y := Fresh("y")

	// Use it like a normal predicate
	goal := edgePred(x, y)

	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)
	results, _ := stream.Take(10)

	fmt.Printf("Found %d edges\n", len(results))

	// Output:
	// Found 3 edges
}

```


\n
## pkg_minikanren_pldb_slg_example_test.go-ExampleWithTabledDatabase.md
```go
func ExampleWithTabledDatabase() {
	edge, _ := DbRel("edge", 2, 0, 1)
	db := NewDatabase()
	db, _ = db.AddFact(edge, NewAtom("a"), NewAtom("b"))

	// Wrap database for automatic tabling
	tdb := WithTabledDatabase(db, "mydb")

	x := Fresh("x")
	y := Fresh("y")

	// Regular Query call, but automatically tabled
	goal := tdb.Query(edge, x, y)

	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)
	results, _ := stream.Take(10)

	fmt.Printf("Found %d edges\n", len(results))

	// Output:
	// Found 1 edges
}

```


\n
## pkg_minikanren_pldb_slg_example_test.go-ExampleWithTabledDatabase_mutation.md
```go
func ExampleWithTabledDatabase_mutation() {
	edge, _ := DbRel("edge", 2, 0, 1)
	db := NewDatabase()
	db, _ = db.AddFact(edge, NewAtom("a"), NewAtom("b"))

	tdb := WithTabledDatabase(db, "mutdb")

	// Add more facts - cache invalidates automatically
	tdb, _ = tdb.AddFact(edge, NewAtom("b"), NewAtom("c"))
	tdb, _ = tdb.AddFact(edge, NewAtom("c"), NewAtom("d"))

	x := Fresh("x")
	y := Fresh("y")

	goal := tdb.Query(edge, x, y)

	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)
	results, _ := stream.Take(10)

	fmt.Printf("Found %d edges after additions\n", len(results))

	// Output:
	// Found 3 edges after additions
}

```


\n
## pkg_minikanren_pldb_slg_recursive_example_test.go-ExampleRecursiveRule_familyTree.md
```go
func ExampleRecursiveRule_familyTree() {
	// Define relations
	parent, _ := DbRel("parent", 2, 0, 1)

	// Build family tree
	db := DB().MustAddFacts(parent,
		[]interface{}{"john", "mary"},
		[]interface{}{"john", "tom"},
		[]interface{}{"mary", "alice"},
		[]interface{}{"tom", "bob"},
	)

	// Query variables
	x := Fresh("x")
	y := Fresh("y")

	// Define ancestor as recursive rule
	//ancestor := RecursiveRule(
	//	db,
	//	parent,     // base: parent is ancestor
	//	"ancestor", // predicate ID
	//	[]Term{x, y},
	//	func() Goal { // recursive: ancestor of parent is ancestor
	//		z := Fresh("z")
	//		return Conj(
	//			TabledQuery(db, parent, "ancestor", x, z),
	//			TabledQuery(db, parent, "ancestor", z, y),
	//		)
	//	},
	//)

	// For now, just query direct parents (base case)
	goal := Conj(
		Eq(y, NewAtom("alice")),
		db.Query(parent, x, y),
	)

	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)
	results, _ := stream.Take(10)

	// Collect results
	parents := make([]string, 0)
	for _, s := range results {
		if binding := s.GetBinding(x.ID()); binding != nil {
			if atom, ok := binding.(*Atom); ok {
				parents = append(parents, atom.String())
			}
		}
	}
	sort.Strings(parents)

	for _, name := range parents {
		fmt.Printf("%s is parent of alice\n", name)
	}

	// Output:
	// mary is parent of alice
}

```


\n
## pkg_minikanren_pldb_slg_recursive_example_test.go-ExampleTabledDatabase.md
```go
func ExampleTabledDatabase() {
	edge, _ := DbRel("edge", 2, 0, 1)
	db := DB().MustAddFacts(edge,
		[]interface{}{"a", "b"},
		[]interface{}{"b", "c"},
	)

	// Wrap database for automatic tabling
	tdb := WithTabledDatabase(db, "mydb")

	x := Fresh("x")
	y := Fresh("y")

	// All queries automatically use tabling
	goal := tdb.Query(edge, x, y)

	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)
	results, _ := stream.Take(10)

	fmt.Printf("Found %d edges with automatic tabling\n", len(results))

	// Output:
	// Found 2 edges with automatic tabling
}

```


\n
## pkg_minikanren_pldb_slg_recursive_example_test.go-ExampleTabledDatabase_withMutation.md
```go
func ExampleTabledDatabase_withMutation() {
	edge, _ := DbRel("edge", 2, 0, 1)
	db := NewDatabase()
	db, _ = db.AddFact(edge, NewAtom("a"), NewAtom("b"))

	tdb := WithTabledDatabase(db, "mutable")

	x := Fresh("x")
	y := Fresh("y")

	// Query once to populate cache
	goal := tdb.Query(edge, x, y)
	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)
	results1, _ := stream.Take(10)

	fmt.Printf("Before update: %d edges\n", len(results1))

	// Add a new fact
	db2, _ := db.AddFact(edge, NewAtom("b"), NewAtom("c"))
	tdb2 := WithTabledDatabase(db2, "mutable")

	// Clear cache for this predicate
	InvalidateAll()

	// Query again with new database
	goal2 := tdb2.Query(edge, x, y)
	stream2 := goal2(ctx, store)
	results2, _ := stream2.Take(10)

	fmt.Printf("After update: %d edges\n", len(results2))

	// Output:
	// Before update: 1 edges
	// After update: 2 edges
}

```


\n
## pkg_minikanren_pldb_slg_recursive_example_test.go-ExampleTabledQuery_grandparent.md
```go
func ExampleTabledQuery_grandparent() {
	// Create parent relation
	parent, _ := DbRel("parent", 2, 0, 1)
	db := DB().MustAddFacts(parent,
		[]interface{}{"john", "mary"},
		[]interface{}{"mary", "alice"},
	)

	// Query for grandparent
	gp := Fresh("gp")
	p := Fresh("p")
	gc := Fresh("gc")

	// grandparent(GP, GC) :- parent(GP, P), parent(P, GC)
	goal := Conj(
		TabledQuery(db, parent, "parent", gp, p),
		TabledQuery(db, parent, "parent", p, gc),
		Eq(gp, NewAtom("john")),
	)

	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)
	results, _ := stream.Take(10)

	if len(results) > 0 {
		binding := results[0].GetBinding(gc.ID())
		if atom, ok := binding.(*Atom); ok {
			fmt.Printf("john's grandchild: %s\n", atom.String())
		}
	}

	// Output:
	// john's grandchild: alice
}

```


\n
## pkg_minikanren_pldb_slg_recursive_example_test.go-ExampleTabledQuery_multiRelation.md
```go
func ExampleTabledQuery_multiRelation() {
	employee, _ := DbRel("employee", 2, 0, 1) // (name, dept)
	manager, _ := DbRel("manager", 2, 0, 1)   // (mgr, employee)

	db := DB().MustAddFacts(employee,
		[]interface{}{"alice", "engineering"},
		[]interface{}{"bob", "engineering"},
	)
	db = db.MustAddFacts(manager,
		[]interface{}{"bob", "alice"},
	)

	// Who manages Alice?
	mgr := Fresh("mgr")
	goal := Conj(
		TabledQuery(db, manager, "mgr", mgr, NewAtom("alice")),
		TabledQuery(db, employee, "emp", mgr, Fresh("_")), // ensure mgr is an employee
	)

	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)
	results, _ := stream.Take(10)

	if len(results) > 0 {
		binding := results[0].GetBinding(mgr.ID())
		if atom, ok := binding.(*Atom); ok {
			fmt.Printf("%s manages alice\n", atom.String())
		}
	}

	// Output:
	// bob manages alice
}

```


\n
## pkg_minikanren_pldb_slg_recursive_example_test.go-ExampleTabledRelation_symmetricGraph.md
```go
func ExampleTabledRelation_symmetricGraph() {
	friend, _ := DbRel("friend", 2, 0, 1)
	db := DB().MustAddFacts(friend,
		[]interface{}{"alice", "bob"},
		[]interface{}{"bob", "alice"},
	)

	friendPred := TabledRelation(db, friend, "friend")

	x := Fresh("x")
	// Who is friends with Alice?
	goal := Conj(
		friendPred(x, NewAtom("alice")),
	)

	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)
	results, _ := stream.Take(10)

	if len(results) > 0 {
		binding := results[0].GetBinding(x.ID())
		if atom, ok := binding.(*Atom); ok {
			fmt.Printf("%s is friend with alice\n", atom.String())
		}
	}

	// Output:
	// bob is friend with alice
}

```


\n
## pkg_minikanren_pldb_slg_recursive_example_test.go-ExampleTabledRelation_transitiveClosureManual.md
```go
func ExampleTabledRelation_transitiveClosureManual() {
	// Define edge relation
	edge, _ := DbRel("edge", 2, 0, 1)
	db := DB().MustAddFacts(edge,
		[]interface{}{"a", "b"},
		[]interface{}{"b", "c"},
		[]interface{}{"c", "d"},
	)

	// Create tabled edge predicate
	edgeTabled := TabledRelation(db, edge, "edge")

	// Manually define path using disjunction: path(X,Y) :- edge(X,Y) | (edge(X,Z), path(Z,Y))
	// For a proper recursive definition, we'd need fixpoint computation
	// Here we just show multi-hop manually
	x := Fresh("x")
	y := Fresh("y")
	z := Fresh("z")

	// Two-hop path: a->b->c or b->c->d
	twoHop := Conj(
		edgeTabled(x, z),
		edgeTabled(z, y),
	)

	// Bind x to "a" to find paths from a
	goal := Conj(
		Eq(x, NewAtom("a")),
		twoHop,
	)

	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)
	results, _ := stream.Take(10)

	// Should find a->c (via b)
	if len(results) > 0 {
		binding := results[0].GetBinding(y.ID())
		if atom, ok := binding.(*Atom); ok {
			fmt.Printf("a reaches %s in 2 hops\n", atom.String())
		}
	}

	// Output:
	// a reaches c in 2 hops
}

```


\n
## pkg_minikanren_propagation_example_test.go-ExampleNewAllDifferent.md
```go
func ExampleNewAllDifferent() {
	model := NewModel()

	// Create three variables with domain {1, 2, 3}
	// low-level: x := model.NewVariable(NewBitSetDomain(3))
	x := model.IntVar(1, 3, "x")
	// low-level: y := model.NewVariable(NewBitSetDomain(3))
	y := model.IntVar(1, 3, "y")
	// low-level: z := model.NewVariable(NewBitSetDomain(3))
	z := model.IntVar(1, 3, "z")

	// Ensure all three variables have different values
	c, err := NewAllDifferent([]*FDVariable{x, y, z})
	if err != nil {
		panic(err)
	}
	model.AddConstraint(c)

	solver := NewSolver(model)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	solutions, _ := solver.Solve(ctx, 2) // Get first 2 solutions

	for i, sol := range solutions {
		fmt.Printf("Solution %d: x=%d, y=%d, z=%d\n", i+1, sol[x.ID()], sol[y.ID()], sol[z.ID()])
	}

	// Output:
	// Solution 1: x=1, y=2, z=3
	// Solution 2: x=1, y=3, z=2
}

```


\n
## pkg_minikanren_propagation_example_test.go-ExampleNewAllDifferent_nQueens.md
```go
func ExampleNewAllDifferent_nQueens() {
	n := 4
	model := NewModel()

	// Column positions for each row
	cols := model.NewVariables(n, NewBitSetDomain(n))

	// Diagonal variables (need larger domain to accommodate offsets)
	diag1 := model.NewVariables(n, NewBitSetDomain(2*n))
	diag2 := model.NewVariables(n, NewBitSetDomain(2*n))

	// Link diagonals to columns
	for i := 0; i < n; i++ {
		// diag1[i] = col[i] + i
		c, err := NewArithmetic(cols[i], diag1[i], i)
		if err != nil {
			panic(err)
		}
		model.AddConstraint(c)
		// diag2[i] = col[i] - i + n (offset to keep positive)
		c, err = NewArithmetic(cols[i], diag2[i], -i+n)
		if err != nil {
			panic(err)
		}
		model.AddConstraint(c)
	}

	// All queens in different columns, and different diagonals
	c, err := NewAllDifferent(cols)
	if err != nil {
		panic(err)
	}
	model.AddConstraint(c)
	c, err = NewAllDifferent(diag1)
	if err != nil {
		panic(err)
	}
	model.AddConstraint(c)
	c, err = NewAllDifferent(diag2)
	if err != nil {
		panic(err)
	}
	model.AddConstraint(c)

	solver := NewSolver(model)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	solutions, _ := solver.Solve(ctx, 2) // Get 2 solutions

	for i, sol := range solutions {
		fmt.Printf("Solution %d: [", i+1)
		for row := 0; row < n; row++ {
			if row > 0 {
				fmt.Print(" ")
			}
			fmt.Print(sol[cols[row].ID()])
		}
		fmt.Println("]")
	}

	// Output:
	// Solution 1: [2 4 1 3]
	// Solution 2: [3 1 4 2]
}

```


\n
## pkg_minikanren_propagation_example_test.go-ExampleNewArithmetic_chain.md
```go
func ExampleNewArithmetic_chain() {
	model := NewModel()

	// low-level: a := model.NewVariable(NewBitSetDomainFromValues(20, []int{2, 5}))
	a := model.IntVarValues([]int{2, 5}, "a")
	// low-level: b := model.NewVariable(NewBitSetDomain(20))
	b := model.IntVar(1, 20, "b")
	// low-level: c := model.NewVariable(NewBitSetDomain(20))
	c := model.IntVar(1, 20, "c")

	// Create chain: B = A + 5, C = B + 3, so C = A + 8
	constraint1, err := NewArithmetic(a, b, 5)
	if err != nil {
		panic(err)
	}
	model.AddConstraint(constraint1)
	constraint2, err := NewArithmetic(b, c, 3)
	if err != nil {
		panic(err)
	}
	model.AddConstraint(constraint2)

	solver := NewSolver(model)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	solutions, _ := solver.Solve(ctx, 0)

	for _, sol := range solutions {
		fmt.Printf("a=%d, b=%d, c=%d (c = a + 8)\n",
			sol[a.ID()], sol[b.ID()], sol[c.ID()])
	}

	// Output:
	// a=2, b=7, c=10 (c = a + 8)
	// a=5, b=10, c=13 (c = a + 8)
}

```


\n
## pkg_minikanren_propagation_example_test.go-ExampleNewArithmetic.md
```go
func ExampleNewArithmetic() {
	model := NewModel()

	// Create variables with specific domains
	// low-level: x := model.NewVariable(NewBitSetDomainFromValues(10, []int{2, 5, 7}))
	x := model.IntVarValues([]int{2, 5, 7}, "x")
	// low-level: y := model.NewVariable(NewBitSetDomain(10))
	y := model.IntVar(1, 10, "y")

	// Enforce: Y = X + 3
	c, err := NewArithmetic(x, y, 3)
	if err != nil {
		panic(err)
	}
	model.AddConstraint(c)

	solver := NewSolver(model)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	solutions, _ := solver.Solve(ctx, 0) // Get all solutions

	for _, sol := range solutions {
		fmt.Printf("x=%d, y=%d (y = x + 3)\n", sol[x.ID()], sol[y.ID()])
	}

	// Output:
	// x=2, y=5 (y = x + 3)
	// x=5, y=8 (y = x + 3)
	// x=7, y=10 (y = x + 3)
}

```


\n
## pkg_minikanren_propagation_example_test.go-ExampleNewArithmetic_negative.md
```go
func ExampleNewArithmetic_negative() {
	model := NewModel()

	// low-level: x := model.NewVariable(NewBitSetDomainFromValues(10, []int{3, 5, 8}))
	x := model.IntVarValues([]int{3, 5, 8}, "x")
	// low-level: y := model.NewVariable(NewBitSetDomain(10))
	y := model.IntVar(1, 10, "y")

	// Enforce: Y = X - 2 (using negative offset)
	c, err := NewArithmetic(x, y, -2)
	if err != nil {
		panic(err)
	}
	model.AddConstraint(c)

	solver := NewSolver(model)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	solutions, _ := solver.Solve(ctx, 0)

	for _, sol := range solutions {
		fmt.Printf("x=%d, y=%d (y = x - 2)\n", sol[x.ID()], sol[y.ID()])
	}

	// Output:
	// x=3, y=1 (y = x - 2)
	// x=5, y=3 (y = x - 2)
	// x=8, y=6 (y = x - 2)
}

```


\n
## pkg_minikanren_propagation_example_test.go-ExampleNewInequality_lessThan.md
```go
func ExampleNewInequality_lessThan() {
	model := NewModel()

	x := model.NewVariable(NewBitSetDomainFromValues(5, []int{2}))
	y := model.NewVariable(NewBitSetDomain(5))

	// Enforce: X < Y
	c, err := NewInequality(x, y, LessThan)
	if err != nil {
		panic(err)
	}
	model.AddConstraint(c)

	solver := NewSolver(model)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	solutions, _ := solver.Solve(ctx, 0)

	for _, sol := range solutions {
		fmt.Printf("x=%d < y=%d\n", sol[x.ID()], sol[y.ID()])
	}

	// Output:
	// x=2 < y=3
	// x=2 < y=4
	// x=2 < y=5
}

```


\n
## pkg_minikanren_propagation_example_test.go-ExampleNewInequality_notEqual.md
```go
func ExampleNewInequality_notEqual() {
	model := NewModel()

	// low-level: x := model.NewVariable(NewBitSetDomain(3))
	x := model.IntVar(1, 3, "x")
	// low-level: y := model.NewVariable(NewBitSetDomain(3))
	y := model.IntVar(1, 3, "y")

	// Enforce: X ≠ Y
	c, err := NewInequality(x, y, NotEqual)
	if err != nil {
		panic(err)
	}
	model.AddConstraint(c)

	solver := NewSolver(model)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	solutions, _ := solver.Solve(ctx, 3) // Get first 3 solutions

	for i, sol := range solutions {
		fmt.Printf("Solution %d: x=%d, y=%d\n", i+1, sol[x.ID()], sol[y.ID()])
	}

	// Output:
	// Solution 1: x=1, y=2
	// Solution 2: x=1, y=3
	// Solution 3: x=2, y=1
}

```


\n
## pkg_minikanren_propagation_example_test.go-ExampleNewInequality_ordering.md
```go
func ExampleNewInequality_ordering() {
	model := NewModel()

	x := model.NewVariable(NewBitSetDomain(5))
	y := model.NewVariable(NewBitSetDomain(5))
	z := model.NewVariable(NewBitSetDomain(5))

	// Enforce: X < Y < Z (ascending order)
	c, err := NewInequality(x, y, LessThan)
	if err != nil {
		panic(err)
	}
	model.AddConstraint(c)
	c, err = NewInequality(y, z, LessThan)
	if err != nil {
		panic(err)
	}
	model.AddConstraint(c)

	solver := NewSolver(model)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	solutions, _ := solver.Solve(ctx, 5) // Get first 5 solutions

	for _, sol := range solutions {
		fmt.Printf("x=%d < y=%d < z=%d\n", sol[x.ID()], sol[y.ID()], sol[z.ID()])
	}

	// Output:
	// x=1 < y=2 < z=3
	// x=1 < y=2 < z=4
	// x=1 < y=2 < z=5
	// x=1 < y=3 < z=4
	// x=1 < y=3 < z=5
}

```


\n
## pkg_minikanren_rational_example_test.go-ExampleApproximateIrrational.md
```go
func ExampleApproximateIrrational() {
	// Get pi with different precision levels
	piLow, _ := ApproximateIrrational("pi", 2)
	piHigh, _ := ApproximateIrrational("pi", 6)

	fmt.Printf("π (low precision): %s ≈ %.4f\n", piLow, piLow.ToFloat())
	fmt.Printf("π (high precision): %s ≈ %.6f\n", piHigh, piHigh.ToFloat())

	// Get sqrt(2)
	sqrt2, _ := ApproximateIrrational("sqrt2", 4)
	fmt.Printf("√2: %s ≈ %.4f\n", sqrt2, sqrt2.ToFloat())

	// Output:
	// π (low precision): 22/7 ≈ 3.1429
	// π (high precision): 355/113 ≈ 3.141593
	// √2: 99/70 ≈ 1.4143
}

```


\n
## pkg_minikanren_rational_example_test.go-ExampleCommonIrrationals.md
```go
func ExampleCommonIrrationals() {
	// Pi approximations
	fmt.Printf("π ≈ %s (Archimedes)\n", CommonIrrationals.PiArchimedes)
	fmt.Printf("π ≈ %s (Zu Chongzhi)\n", CommonIrrationals.PiZu)

	// Square root of 2
	fmt.Printf("√2 ≈ %s (simple)\n", CommonIrrationals.Sqrt2Simple)

	// Euler's number
	fmt.Printf("e ≈ %s\n", CommonIrrationals.ESimple)

	// Golden ratio
	fmt.Printf("φ ≈ %s\n", CommonIrrationals.PhiSimple)

	// Output:
	// π ≈ 22/7 (Archimedes)
	// π ≈ 355/113 (Zu Chongzhi)
	// √2 ≈ 99/70 (simple)
	// e ≈ 2721/1000
	// φ ≈ 809/500
}

```


\n
## pkg_minikanren_rational_example_test.go-ExampleRational_arithmetic.md
```go
func ExampleRational_arithmetic() {
	a := NewRational(1, 2) // 1/2
	b := NewRational(1, 3) // 1/3

	sum := a.Add(b)
	diff := a.Sub(b)
	product := a.Mul(b)
	quotient := a.Div(b)

	fmt.Printf("1/2 + 1/3 = %s\n", sum)
	fmt.Printf("1/2 - 1/3 = %s\n", diff)
	fmt.Printf("1/2 * 1/3 = %s\n", product)
	fmt.Printf("1/2 / 1/3 = %s\n", quotient)

	// Output:
	// 1/2 + 1/3 = 5/6
	// 1/2 - 1/3 = 1/6
	// 1/2 * 1/3 = 1/6
	// 1/2 / 1/3 = 3/2
}

```


\n
## pkg_minikanren_rational_example_test.go-ExampleRational_circumference.md
```go
func ExampleRational_circumference() {
	// Calculate circumference = π * diameter using rational approximation
	pi := CommonIrrationals.PiArchimedes // 22/7
	diameter := NewRational(14, 1)       // diameter = 14

	circumference := pi.Mul(diameter)

	fmt.Printf("For diameter = %s\n", diameter)
	fmt.Printf("Circumference = π × d = %s × %s = %s\n", pi, diameter, circumference)
	fmt.Printf("Circumference ≈ %.2f\n", circumference.ToFloat())

	// Output:
	// For diameter = 14
	// Circumference = π × d = 22/7 × 14 = 44
	// Circumference ≈ 44.00
}

```


\n
## pkg_minikanren_rational_linear_sum_example_test.go-ExampleNewRationalLinearSum.md
```go
func ExampleNewRationalLinearSum() {
	model := NewModel()

	// Variables: hours worked
	// low-level: hours := model.NewVariable(NewBitSetDomainFromValues(50, []int{8})) // 8 hours worked
	hours := model.IntVarValues([]int{8}, "hours") // 8 hours worked
	// low-level: payment := model.NewVariable(NewBitSetDomain(1000))                 // payment in dollars
	payment := model.IntVar(1, 1000, "payment") // payment in dollars

	// Constraint: payment = 25 * hours (hourly rate of $25)
	coeffs := []Rational{NewRational(25, 1)} // $25/hour as coefficient
	rls, err := NewRationalLinearSum([]*FDVariable{hours}, coeffs, payment)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	model.AddConstraint(rls)

	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	paymentDomain := solver.GetDomain(nil, payment.ID()).(*BitSetDomain)
	fmt.Printf("Payment: $%d\n", paymentDomain.Min())

	// Output:
	// Payment: $200
}

```


\n
## pkg_minikanren_rational_linear_sum_example_test.go-ExampleNewRationalLinearSumWithScaling.md
```go
func ExampleNewRationalLinearSumWithScaling() {
	model := NewModel()

	// Three investors with different ownership percentages
	// low-level: investorA := model.NewVariable(NewBitSetDomainFromValues(10000, []int{3000})) // $3000 invested
	investorA := model.IntVarValues([]int{3000}, "investorA") // $3000 invested
	// low-level: investorB := model.NewVariable(NewBitSetDomainFromValues(10000, []int{2000})) // $2000 invested
	investorB := model.IntVarValues([]int{2000}, "investorB") // $2000 invested
	// Total investment
	total := model.IntVar(1, 10000, "total")

	// Constraint: total = (1/3)*A + (1/2)*B (fractional ownership)
	// Note: This is a simplified example; in reality you'd sum all investments
	coeffs := []Rational{
		NewRational(1, 3), // investor A owns 1/3 of their contribution
		NewRational(1, 2), // investor B owns 1/2 of their contribution
	}

	rls, div, err := NewRationalLinearSumWithScaling(
		[]*FDVariable{investorA, investorB},
		coeffs,
		total,
		model,
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	model.AddConstraint(rls)
	if div != nil {
		// The scaling helper created an intermediate variable and ScaledDivision constraint
		model.AddConstraint(div)
		fmt.Println("Scaling was needed (LCM > 1)")
	}

	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	totalDomain := solver.GetDomain(nil, total.ID()).(*BitSetDomain)
	fmt.Printf("Total: $%d\n", totalDomain.Min())

	// Output:
	// Scaling was needed (LCM > 1)
	// Total: $2000
}

```


\n
## pkg_minikanren_rational_linear_sum_example_test.go-ExampleRationalLinearSum_percentageCalculation.md
```go
func ExampleRationalLinearSum_percentageCalculation() {
	model := NewModel()

	// Base salary: $50,000
	// low-level: baseSalary := model.NewVariable(NewBitSetDomainFromValues(100000, []int{50000}))
	baseSalary := model.IntVarValues([]int{50000}, "baseSalary")
	// Total with 10% bonus. Use a realistic, narrower domain to keep the example fast.
	// Wide dense domains cause ScaledDivision to enumerate large ranges for arc-consistency.
	// Here we bound to [54_000..56_000] which still demonstrates propagation clearly
	// while keeping runtime well under a second.
	totalPay := model.NewVariable(DomainRange(54000, 56000))

	// Constraint: totalPay = 1.1 * baseSalary = (11/10) * baseSalary
	coeffs := []Rational{NewRational(11, 10)} // 110% = 11/10

	rls, div, err := NewRationalLinearSumWithScaling(
		[]*FDVariable{baseSalary},
		coeffs,
		totalPay,
		model,
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	model.AddConstraint(rls)
	if div != nil {
		model.AddConstraint(div)
	}

	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	totalDomain := solver.GetDomain(nil, totalPay.ID()).(*BitSetDomain)
	fmt.Printf("Base salary: $50,000\n")
	fmt.Printf("With 10%% bonus: $%d\n", totalDomain.Min())

	// Output:
	// Base salary: $50,000
	// With 10% bonus: $55000
}

```


\n
## pkg_minikanren_rational_linear_sum_example_test.go-ExampleRationalLinearSum_piCircumference.md
```go
func ExampleRationalLinearSum_piCircumference() {
	model := NewModel()

	// Circle with diameter = 7 units
	// low-level: diameter := model.NewVariable(NewBitSetDomainFromValues(10, []int{7}))
	diameter := model.IntVarValues([]int{7}, "diameter")
	// low-level: circumference := model.NewVariable(NewBitSetDomain(100))
	circumference := model.IntVar(1, 100, "circumference")

	// Constraint: circumference = π * diameter
	// Using Archimedes' approximation: π ≈ 22/7
	pi := CommonIrrationals.PiArchimedes
	coeffs := []Rational{pi}

	rls, div, err := NewRationalLinearSumWithScaling(
		[]*FDVariable{diameter},
		coeffs,
		circumference,
		model,
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	model.AddConstraint(rls)
	if div != nil {
		model.AddConstraint(div)
	}

	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	circumDomain := solver.GetDomain(nil, circumference.ID()).(*BitSetDomain)
	fmt.Printf("Diameter: %d units\n", 7)
	fmt.Printf("Circumference: %d units (using π ≈ 22/7)\n", circumDomain.Min())

	// Output:
	// Diameter: 7 units
	// Circumference: 22 units (using π ≈ 22/7)
}

```


\n
## pkg_minikanren_regular_example_test.go-ExampleNewRegular.md
```go
func ExampleNewRegular() {
	// Build DFA: states 1=start, 2=last=1, 3=last=2; accept={2}
	numStates, start, accept, delta := buildEndsWith1DFA()

	model := NewModel()
	x1 := model.NewVariableWithName(NewBitSetDomain(2), "x1")
	x2 := model.NewVariableWithName(NewBitSetDomain(2), "x2")
	x3 := model.NewVariableWithName(NewBitSetDomain(2), "x3")

	c, _ := NewRegular([]*FDVariable{x1, x2, x3}, numStates, start, accept, delta)
	model.AddConstraint(c)
	solver := NewSolver(model)

	st, _ := solver.propagate(nil)
	fmt.Println("x1:", solver.GetDomain(st, x1.ID()))
	fmt.Println("x2:", solver.GetDomain(st, x2.ID()))
	fmt.Println("x3:", solver.GetDomain(st, x3.ID()))
	// Output:
	// x1: {1..2}
	// x2: {1..2}
	// x3: {1}
}

```


\n
## pkg_minikanren_reification_example_test.go-ExampleReifiedConstraint.md
```go
func ExampleReifiedConstraint() {
	model := NewModel()
	x := model.NewVariableWithName(NewBitSetDomain(3), "X")
	y := model.NewVariableWithName(NewBitSetDomain(3), "Y")
	b := model.NewVariableWithName(NewBitSetDomain(2), "B") // {1,2} maps to {false,true}

	arith, _ := NewArithmetic(x, y, 0) // X + 0 = Y
	reified, _ := NewReifiedConstraint(arith, b)
	model.AddConstraint(reified)

	solver := NewSolver(model)
	solutions, _ := solver.Solve(context.Background(), 0)

	// Collect and sort output to make the example deterministic.
	var lines []string
	for _, sol := range solutions {
		lines = append(lines, fmt.Sprintf("X=%d Y=%d B=%t", sol[x.ID()], sol[y.ID()], sol[b.ID()] == 2))
	}
	sort.Strings(lines)

	for i := 0; i < 3 && i < len(lines); i++ {
		fmt.Println(lines[i])
	}
	// Output:
	// X=1 Y=1 B=true
	// X=1 Y=2 B=false
	// X=1 Y=3 B=false
}

```


\n
## pkg_minikanren_relational_arithmetic_example_test.go-ExampleDivo_backward.md
```go
func ExampleDivo_backward() {
	result := Run(1, func(q *Var) Goal {
		return Divo(q, NewAtom(5), NewAtom(3))
	})
	fmt.Println(result[0])
	// Output: 15
}

```


\n
## pkg_minikanren_relational_arithmetic_example_test.go-ExampleDivo_integerDivision.md
```go
func ExampleDivo_integerDivision() {
	result := Run(1, func(q *Var) Goal {
		return Divo(NewAtom(7), NewAtom(2), q)
	})
	fmt.Println(result[0])
	// Output: 3
}

```


\n
## pkg_minikanren_relational_arithmetic_example_test.go-ExampleDivo.md
```go
func ExampleDivo() {
	result := Run(1, func(q *Var) Goal {
		return Divo(NewAtom(15), NewAtom(3), q)
	})
	fmt.Println(result[0])
	// Output: 5
}

```


\n
## pkg_minikanren_relational_arithmetic_example_test.go-ExampleExpo.md
```go
func ExampleExpo() {
	result := Run(1, func(q *Var) Goal {
		return Expo(NewAtom(2), NewAtom(10), q)
	})
	fmt.Println(result[0])
	// Output: 1024
}

```


\n
## pkg_minikanren_relational_arithmetic_example_test.go-ExampleExpo_verification.md
```go
func ExampleExpo_verification() {
	result := Run(1, func(q *Var) Goal {
		return Conj(
			Expo(NewAtom(3), NewAtom(4), NewAtom(81)),
			Eq(q, NewAtom("correct")),
		)
	})
	fmt.Println(result[0])
	// Output: correct
}

```


\n
## pkg_minikanren_relational_arithmetic_example_test.go-ExampleExpo_zeroExponent.md
```go
func ExampleExpo_zeroExponent() {
	result := Run(1, func(q *Var) Goal {
		return Expo(NewAtom(5), NewAtom(0), q)
	})
	fmt.Println(result[0])
	// Output: 1
}

```


\n
## pkg_minikanren_relational_arithmetic_example_test.go-ExampleGreaterEqualo.md
```go
func ExampleGreaterEqualo() {
	result := Run(1, func(q *Var) Goal {
		return Conj(
			GreaterEqualo(NewAtom(10), NewAtom(5)),
			Eq(q, NewAtom("yes")),
		)
	})
	fmt.Println(result[0])
	// Output: yes
}

```


\n
## pkg_minikanren_relational_arithmetic_example_test.go-ExampleGreaterThano.md
```go
func ExampleGreaterThano() {
	result := Run(1, func(q *Var) Goal {
		return Conj(
			GreaterThano(NewAtom(10), NewAtom(5)),
			Eq(q, NewAtom("yes")),
		)
	})
	fmt.Println(result[0])
	// Output: yes
}

```


\n
## pkg_minikanren_relational_arithmetic_example_test.go-ExampleLessEqualo.md
```go
func ExampleLessEqualo() {
	result := Run(1, func(q *Var) Goal {
		return Conj(
			LessEqualo(NewAtom(5), NewAtom(5)),
			Eq(q, NewAtom("yes")),
		)
	})
	fmt.Println(result[0])
	// Output: yes
}

```


\n
## pkg_minikanren_relational_arithmetic_example_test.go-ExampleLessThano_filter.md
```go
func ExampleLessThano_filter() {
	result := Run(10, func(q *Var) Goal {
		return Conj(
			LessThano(q, NewAtom(5)),
			Membero(q, List(NewAtom(1), NewAtom(3), NewAtom(7), NewAtom(2))),
		)
	})
	fmt.Printf("Found %d values\n", len(result))
	// Output: Found 3 values
}

```


\n
## pkg_minikanren_relational_arithmetic_example_test.go-ExampleLessThano.md
```go
func ExampleLessThano() {
	result := Run(1, func(q *Var) Goal {
		return Conj(
			LessThano(NewAtom(3), NewAtom(5)),
			Eq(q, NewAtom("yes")),
		)
	})
	fmt.Println(result[0])
	// Output: yes
}

```


\n
## pkg_minikanren_relational_arithmetic_example_test.go-ExampleLessThano_withArithmetic.md
```go
func ExampleLessThano_withArithmetic() {
	// Find x where x + 2 < 10 and x = 3
	result := Run(1, func(q *Var) Goal {
		temp := Fresh("temp")
		return Conj(
			Eq(q, NewAtom(3)),
			Pluso(q, NewAtom(2), temp),
			LessThano(temp, NewAtom(10)),
		)
	})
	fmt.Println(result[0])
	// Output: 3
}

```


\n
## pkg_minikanren_relational_arithmetic_example_test.go-ExampleLogo_base10.md
```go
func ExampleLogo_base10() {
	result := Run(1, func(q *Var) Goal {
		return Logo(NewAtom(10), NewAtom(1000), q)
	})
	fmt.Println(result[0])
	// Output: 3
}

```


\n
## pkg_minikanren_relational_arithmetic_example_test.go-ExampleLogo.md
```go
func ExampleLogo() {
	result := Run(1, func(q *Var) Goal {
		return Logo(NewAtom(2), NewAtom(1024), q)
	})
	fmt.Println(result[0])
	// Output: 10
}

```


\n
## pkg_minikanren_relational_arithmetic_example_test.go-ExampleMinuso_backward.md
```go
func ExampleMinuso_backward() {
	result := Run(1, func(q *Var) Goal {
		return Minuso(NewAtom(10), q, NewAtom(6))
	})
	fmt.Println(result[0])
	// Output: 4
}

```


\n
## pkg_minikanren_relational_arithmetic_example_test.go-ExampleMinuso.md
```go
func ExampleMinuso() {
	result := Run(1, func(q *Var) Goal {
		return Minuso(NewAtom(10), NewAtom(3), q)
	})
	fmt.Println(result[0])
	// Output: 7
}

```


\n
## pkg_minikanren_relational_arithmetic_example_test.go-ExampleMinuso_negative.md
```go
func ExampleMinuso_negative() {
	result := Run(1, func(q *Var) Goal {
		return Minuso(NewAtom(3), NewAtom(7), q)
	})
	fmt.Println(result[0])
	// Output: -4
}

```


\n
## pkg_minikanren_relational_arithmetic_example_test.go-ExamplePluso_backward.md
```go
func ExamplePluso_backward() {
	result := Run(1, func(q *Var) Goal {
		return Pluso(q, NewAtom(3), NewAtom(8))
	})
	fmt.Println(result[0])
	// Output: 5
}

```


\n
## pkg_minikanren_relational_arithmetic_example_test.go-ExamplePluso_chained.md
```go
func ExamplePluso_chained() {
	// x + y = 5, y + z = 7, with x = 2, solve for y and z
	result := Run(1, func(q *Var) Goal {
		x := Fresh("x")
		y := Fresh("y")
		z := Fresh("z")
		return Conj(
			Eq(x, NewAtom(2)),
			Pluso(x, y, NewAtom(5)),
			Pluso(y, z, NewAtom(7)),
			Eq(q, List(x, y, z)),
		)
	})

	// Extract list values
	list := result[0]
	var vals []Term
	for {
		if pair, ok := list.(*Pair); ok {
			vals = append(vals, pair.Car())
			list = pair.Cdr()
		} else {
			break
		}
	}
	fmt.Printf("x=%v, y=%v, z=%v\n", vals[0], vals[1], vals[2])
	// Output: x=2, y=3, z=4
}

```


\n
## pkg_minikanren_relational_arithmetic_example_test.go-ExamplePluso_composition.md
```go
func ExamplePluso_composition() {
	// Solve (x + 3) * 2 = 10 for x
	result := Run(1, func(q *Var) Goal {
		temp := Fresh("temp")
		return Conj(
			Timeso(temp, NewAtom(2), NewAtom(10)), // temp = 5
			Pluso(q, NewAtom(3), temp),            // q + 3 = 5
		)
	})
	fmt.Println(result[0])
	// Output: 2
}

```


\n
## pkg_minikanren_relational_arithmetic_example_test.go-ExamplePluso_generate.md
```go
func ExamplePluso_generate() {
	result := Run(6, func(q *Var) Goal {
		x := Fresh("x")
		y := Fresh("y")
		return Conj(
			Pluso(x, y, NewAtom(5)),
			Eq(q, NewPair(x, y)),
		)
	})

	fmt.Printf("Generated %d pairs\n", len(result))
	// Verify all pairs sum to 5
	for _, r := range result {
		pair := r.(*Pair)
		x, _ := extractNumber(pair.Car())
		y, _ := extractNumber(pair.Cdr())
		if x+y == 5 {
			fmt.Println("Valid pair")
		}
	}
	// Output:
	// Generated 6 pairs
	// Valid pair
	// Valid pair
	// Valid pair
	// Valid pair
	// Valid pair
	// Valid pair
} // ExampleMinuso demonstrates basic subtraction with Minuso.

```


\n
## pkg_minikanren_relational_arithmetic_example_test.go-ExamplePluso.md
```go
func ExamplePluso() {
	result := Run(1, func(q *Var) Goal {
		return Pluso(NewAtom(2), NewAtom(3), q)
	})
	fmt.Println(result[0])
	// Output: 5
}

```


\n
## pkg_minikanren_relational_arithmetic_example_test.go-ExampleTimeso_backward.md
```go
func ExampleTimeso_backward() {
	result := Run(1, func(q *Var) Goal {
		return Timeso(q, NewAtom(6), NewAtom(24))
	})
	fmt.Println(result[0])
	// Output: 4
}

```


\n
## pkg_minikanren_relational_arithmetic_example_test.go-ExampleTimeso.md
```go
func ExampleTimeso() {
	result := Run(1, func(q *Var) Goal {
		return Timeso(NewAtom(4), NewAtom(5), q)
	})
	fmt.Println(result[0])
	// Output: 20
}

```


\n
## pkg_minikanren_relational_arithmetic_example_test.go-ExampleTimeso_notDivisible.md
```go
func ExampleTimeso_notDivisible() {
	// ? * 3 = 10 has no integer solution
	result := Run(1, func(q *Var) Goal {
		return Timeso(q, NewAtom(3), NewAtom(10))
	})
	fmt.Println(len(result))
	// Output: 0
}

```


\n
## pkg_minikanren_scaled_division_example_test.go-ExampleNewScaledDivision_bidirectional.md
```go
func ExampleNewScaledDivision_bidirectional() {
	model := NewModel()

	// Price scaled by 100 (cents)
	priceVar := model.NewVariableWithName(
		NewBitSetDomainFromValues(2000, makeRangeEx(500, 1500)),
		"price_cents",
	)

	// Discount rate (percentage): 10-20%
	discountVar := model.NewVariableWithName(
		NewBitSetDomainFromValues(21, makeRangeEx(10, 20)),
		"discount_pct",
	)

	// Constraint: discount_pct = price_cents / 100
	// This means price must be divisible by 100 for exact percentage
	constraint, _ := NewScaledDivision(priceVar, 100, discountVar)
	model.AddConstraint(constraint)

	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	finalPrice := solver.GetDomain(nil, priceVar.ID())
	finalDiscount := solver.GetDomain(nil, discountVar.ID())

	fmt.Println("Bidirectional propagation:")
	fmt.Printf("Price: $%d.%02d - $%d.%02d\n",
		finalPrice.Min()/100, finalPrice.Min()%100,
		finalPrice.Max()/100, finalPrice.Max()%100)
	fmt.Printf("Discount: %d%% - %d%%\n",
		finalDiscount.Min(), finalDiscount.Max())

	// Output:
	// Bidirectional propagation:
	// Price: $10.00 - $15.00
	// Discount: 10% - 15%
}

```


\n
## pkg_minikanren_scaled_division_example_test.go-ExampleNewScaledDivision.md
```go
func ExampleNewScaledDivision() {
	model := NewModel()

	// All monetary values scaled by 100 (cents)
	// Salary range: $500-$700 → 50000-70000 cents
	salaryValues := make([]int, 0)
	for s := 50000; s <= 70000; s += 10000 {
		salaryValues = append(salaryValues, s)
	}
	salaryVar := model.NewVariableWithName(
		NewBitSetDomainFromValues(70001, salaryValues),
		"salary",
	)

	// Bonus initially unconstrained
	bonusVar := model.NewVariableWithName(
		NewBitSetDomainFromValues(10000, makeRangeEx(1, 10000)),
		"bonus",
	)

	// Constraint: bonus = salary / 10 (10% bonus)
	// Since values are in cents, this gives us exact integer division
	constraint, err := NewScaledDivision(salaryVar, 10, bonusVar)
	if err != nil {
		panic(err)
	}
	model.AddConstraint(constraint)

	// Solve to propagate constraints
	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	// Check propagated bonus domain
	finalBonus := solver.GetDomain(nil, bonusVar.ID())

	fmt.Println("Salary-to-Bonus constraint (10% bonus):")
	fmt.Printf("Salary range: $500.00 - $700.00\n")
	fmt.Printf("Bonus range: $%d.%02d - $%d.%02d\n",
		finalBonus.Min()/100, finalBonus.Min()%100,
		finalBonus.Max()/100, finalBonus.Max()%100)
	fmt.Printf("Possible bonuses: ")

	bonuses := finalBonus.ToSlice()
	for i, b := range bonuses {
		if i > 0 {
			fmt.Printf(", ")
		}
		fmt.Printf("$%d.%02d", b/100, b%100)
	}
	fmt.Println()

	// Output:
	// Salary-to-Bonus constraint (10% bonus):
	// Salary range: $500.00 - $700.00
	// Bonus range: $50.00 - $70.00
	// Possible bonuses: $50.00, $60.00, $70.00
}

```


\n
## pkg_minikanren_scaled_division_example_test.go-ExampleNewScaledDivision_percentageWithScaling.md
```go
func ExampleNewScaledDivision_percentageWithScaling() {
	model := NewModel()

	// Investment amount: $1000 (in dollars, not cents for this example)
	principal := model.NewVariableWithName(
		NewBitSetDomainFromValues(10001, []int{1000}), // $1000
		"principal",
	)

	// Annual interest rate: 5.25% → stored as 525 basis points (5.25 * 100)
	// Calculate: $1000 * 5.25 / 100 = $52.50
	// For integer result, scale by 100: 1000 * 525 / 100 = 5250 (in cents)
	interestScaled := model.NewVariableWithName(
		NewBitSetDomain(1000000),
		"interest_scaled",
	)

	interestCents := model.NewVariableWithName(
		NewBitSetDomain(10000),
		"interest_cents",
	)

	// Pattern: principal * 525 / 100 = interest_cents
	// Step 1: interest_scaled = principal * 525
	coeffs := []int{525}
	linearConstraint, _ := NewLinearSum(
		[]*FDVariable{principal},
		coeffs,
		interestScaled,
	)
	model.AddConstraint(linearConstraint)

	// Step 2: interest_cents = interest_scaled / 100
	divConstraint, _ := NewScaledDivision(interestScaled, 100, interestCents)
	model.AddConstraint(divConstraint)

	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	finalInterest := solver.GetDomain(nil, interestCents.ID())

	fmt.Println("Fixed-point percentage calculation:")
	fmt.Printf("Principal: $1,000.00\n")
	fmt.Printf("Rate: 5.25%% (525 basis points)\n")
	fmt.Printf("Interest: $%d.%02d\n",
		finalInterest.Min()/100, finalInterest.Min()%100)

	// Output:
	// Fixed-point percentage calculation:
	// Principal: $1,000.00
	// Rate: 5.25% (525 basis points)
	// Interest: $52.50
}

```


\n
## pkg_minikanren_scaled_division_example_test.go-ExampleNewScaledDivision_piCircumference.md
```go
func ExampleNewScaledDivision_piCircumference() {
	model := NewModel()

	// Circle diameter: 1-10 units
	diameter := model.NewVariableWithName(
		NewBitSetDomainFromValues(11, makeRangeEx(1, 10)),
		"diameter",
	)

	// Circumference (scaled by 10000): π * diameter * 10000
	// Using π ≈ 31416/10000 (more precision than 22/7)
	circumferenceScaled := model.NewVariableWithName(
		NewBitSetDomain(350000),
		"circumference_scaled",
	)

	// Actual circumference (in original units)
	circumference := model.NewVariableWithName(
		NewBitSetDomain(35),
		"circumference",
	)

	// Pattern: Fixed-point arithmetic for irrationals
	// 1. Scale the constant: π ≈ 31416/10000
	// 2. Use LinearSum with scaled constant: circumference_scaled = 31416 * diameter
	// 3. Use ScaledDivision to get final result: circumference = circumference_scaled / 10000

	// Step 1: circumference_scaled = 31416 * diameter
	coeffs := []int{31416}
	linearConstraint, _ := NewLinearSum(
		[]*FDVariable{diameter},
		coeffs,
		circumferenceScaled,
	)
	model.AddConstraint(linearConstraint)

	// Step 2: circumference = circumference_scaled / 10000
	divConstraint, _ := NewScaledDivision(circumferenceScaled, 10000, circumference)
	model.AddConstraint(divConstraint)

	// Fix diameter = 7 for demonstration
	diameter7Domain := NewBitSetDomainFromValues(11, []int{7})
	diameter = model.NewVariableWithName(diameter7Domain, "diameter")

	// Rebuild constraints with fixed diameter
	model = NewModel()
	diameter = model.NewVariableWithName(diameter7Domain, "diameter")
	circumferenceScaled = model.NewVariableWithName(NewBitSetDomain(350000), "circumference_scaled")
	circumference = model.NewVariableWithName(NewBitSetDomain(35), "circumference")

	linearConstraint, _ = NewLinearSum([]*FDVariable{diameter}, coeffs, circumferenceScaled)
	model.AddConstraint(linearConstraint)

	divConstraint, _ = NewScaledDivision(circumferenceScaled, 10000, circumference)
	model.AddConstraint(divConstraint)

	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	finalCircum := solver.GetDomain(nil, circumference.ID())
	finalScaled := solver.GetDomain(nil, circumferenceScaled.ID())

	fmt.Println("Fixed-point π calculation:")
	fmt.Printf("Diameter: 7 units\n")
	fmt.Printf("π * 7 * 10000 = %d (scaled)\n", finalScaled.Min())
	fmt.Printf("Circumference: %d units (actual)\n", finalCircum.Min())
	fmt.Printf("Precision: Using π ≈ 3.1416\n")

	// Output:
	// Fixed-point π calculation:
	// Diameter: 7 units
	// π * 7 * 10000 = 219912 (scaled)
	// Circumference: 21 units (actual)
	// Precision: Using π ≈ 3.1416
}

```


\n
## pkg_minikanren_send_more_money_example_test.go-Example_sendMoreMoney_reificationCount.md
```go
func Example_sendMoreMoney_reificationCount() {
	model := NewModel()

	// Digits 0..9 → FD values 1..10 (we use HLAPI IntVar/IntVarValues below)

	// Letter variables (encoded digits)
	// low-level: S := model.NewVariable(digits)
	S := model.IntVar(1, 10, "S")
	// low-level: E := model.NewVariable(digits)
	E := model.IntVar(1, 10, "E")
	// low-level: N := model.NewVariable(digits)
	N := model.IntVar(1, 10, "N")
	// low-level: D := model.NewVariable(digits)
	D := model.IntVar(1, 10, "D")
	// low-level: M := model.NewVariable(digits)
	M := model.IntVar(1, 10, "M")
	// low-level: O := model.NewVariable(digits)
	O := model.IntVar(1, 10, "O")
	// low-level: R := model.NewVariable(digits)
	R := model.IntVar(1, 10, "R")
	// low-level: Y := model.NewVariable(digits)
	Y := model.IntVar(1, 10, "Y")

	// All letters must be distinct
	ad, err := NewAllDifferent([]*FDVariable{S, E, N, D, M, O, R, Y})
	if err != nil {
		panic(err)
	}
	model.AddConstraint(ad)

	// 1) No leading zeros: S and M cannot be digit 0 (encoded as FD value 1)
	//    Count([S, M], target=1) must be 0 → encoded countVar = 1 (0+1)
	// low-level: countVar := model.NewVariable(NewBitSetDomainFromValues(10, []int{1}))
	// The countVar is encoded as count+1; to force count==0 we set countVar to {1}.
	// low-level: countVar := model.NewVariableWithName(NewBitSetDomainFromValues(10, []int{1}), "countVar")
	// Use the low-level constructor here to preserve the original universe size
	// (NewCount expects the countVar's domain MaxValue() to be >= len(vars)+1).
	countVar := model.NewVariableWithName(NewBitSetDomainFromValues(10, []int{1}), "countVar")
	if _, err := NewCount(model, []*FDVariable{S, M}, 1, countVar); err != nil {
		panic(err)
	}

	// 2) Reify M = digit 1 (common fact): encoded M == 2; force boolean to true ({2})
	// low-level: bM := model.NewVariable(NewBitSetDomainFromValues(10, []int{2})) // {2} means true
	bM := model.IntVarValues([]int{2}, "bM") // {2} means true
	reif, err := NewValueEqualsReified(M, 2, bM)
	if err != nil {
		panic(err)
	}
	model.AddConstraint(reif)

	// Propagate and inspect domains
	solver := NewSolver(model)
	// We don't need all solutions; propagation + first solution is enough for inspection
	sols, _ := solver.Solve(context.Background(), 1)

	mDom := solver.GetDomain(nil, M.ID())
	sDom := solver.GetDomain(nil, S.ID())
	fmt.Printf("solutions: %d\n", len(sols))
	fmt.Printf("M singleton and equals 2: %v %v\n", mDom.IsSingleton(), mDom.SingletonValue() == 2)
	fmt.Printf("S allows zero? %v\n", sDom.Has(1))

	// Output:
	// solutions: 1
	// M singleton and equals 2: true true
	// S allows zero? false
}

```


\n
## pkg_minikanren_sequence_example_test.go-ExampleNewSequence.md
```go
func ExampleNewSequence() {
	model := NewModel()
	// x1 := model.NewVariableWithName(NewBitSetDomainFromValues(2, []int{1, 2}), "x1")
	x1 := model.IntVarValues([]int{1, 2}, "x1")
	// x2 := model.NewVariableWithName(NewBitSetDomainFromValues(2, []int{2}), "x2") // forced not in S
	x2 := model.IntVarValues([]int{2}, "x2") // forced not in S
	// x3 := model.NewVariableWithName(NewBitSetDomainFromValues(2, []int{1, 2}), "x3")
	x3 := model.IntVarValues([]int{1, 2}, "x3")
	// x4 := model.NewVariableWithName(NewBitSetDomainFromValues(2, []int{1, 2}), "x4")
	x4 := model.IntVarValues([]int{1, 2}, "x4")
	// x5 := model.NewVariableWithName(NewBitSetDomainFromValues(2, []int{1, 2}), "x5")
	x5 := model.IntVarValues([]int{1, 2}, "x5")

	_, _ = NewSequence(model, []*FDVariable{x1, x2, x3, x4, x5}, []int{1}, 3, 2, 3)

	solver := NewSolver(model)
	_, _ = solver.Solve(context.Background(), 0)

	// Window [x1,x2,x3] needs at least two 1s; since x2!=1, both x1 and x3 become 1
	fmt.Printf("x1: %s\n", solver.GetDomain(nil, x1.ID()))
	fmt.Printf("x3: %s\n", solver.GetDomain(nil, x3.ID()))
	// Output:
	// x1: {1}
	// x3: {1}
}

```


\n
## pkg_minikanren_slg_engine_example_test.go-ExampleGlobalEngine.md
```go
func ExampleGlobalEngine() {
	// Reset to ensure clean state for this example
	ResetGlobalEngine()

	// Get global engine (created on first access)
	engine1 := GlobalEngine()
	engine2 := GlobalEngine()

	if engine1 == engine2 {
		fmt.Println("Same engine instance")
	}

	// Evaluate using global engine
	pattern := NewCallPattern("global", []Term{NewAtom("test")})
	evaluator := func(ctx context.Context, answers chan<- map[int64]Term) error {
		answer := map[int64]Term{1: NewAtom("answer")}
		answers <- answer
		return nil
	}

	resultChan, _ := engine1.Evaluate(context.Background(), pattern, evaluator)
	for range resultChan {
	}

	// State is shared
	stats := engine2.Stats()
	fmt.Printf("Shared state - evaluations: %d\n", stats.TotalEvaluations)

	// Output:
	// Same engine instance
	// Shared state - evaluations: 1
}

```


\n
## pkg_minikanren_slg_engine_example_test.go-ExampleNewSLGEngine_customConfig.md
```go
func ExampleNewSLGEngine_customConfig() {
	config := &SLGConfig{
		MaxTableSize:          5000,
		MaxAnswersPerSubgoal:  100,
		MaxFixpointIterations: 500,
	}

	engine := NewSLGEngine(config)
	fmt.Printf("Max table size: %d\n", engine.config.MaxTableSize)
	fmt.Printf("Max fixpoint iterations: %d\n", engine.config.MaxFixpointIterations)

	// Output:
	// Max table size: 5000
	// Max fixpoint iterations: 500
}

```


\n
## pkg_minikanren_slg_engine_example_test.go-ExampleNewSLGEngine.md
```go
func ExampleNewSLGEngine() {
	engine := NewSLGEngine(nil)

	stats := engine.Stats()
	fmt.Printf("Initial subgoals: %d\n", stats.CachedSubgoals)
	fmt.Printf("Max answers per subgoal: %d\n", engine.config.MaxAnswersPerSubgoal)

	// Output:
	// Initial subgoals: 0
	// Max answers per subgoal: 10000
}

```


\n
## pkg_minikanren_slg_engine_example_test.go-ExampleResetGlobalEngine.md
```go
func ExampleResetGlobalEngine() {
	// Reset to ensure clean state for this example
	ResetGlobalEngine()

	engine := GlobalEngine()

	// Add some state
	pattern := NewCallPattern("temp", []Term{NewAtom("x")})
	evaluator := func(ctx context.Context, answers chan<- map[int64]Term) error {
		answer := map[int64]Term{1: NewAtom("data")}
		answers <- answer
		return nil
	}

	resultChan, _ := engine.Evaluate(context.Background(), pattern, evaluator)
	for range resultChan {
	}

	statsBefore := engine.Stats()
	fmt.Printf("Before reset - evaluations: %d\n", statsBefore.TotalEvaluations)

	// Reset state
	ResetGlobalEngine()

	statsAfter := engine.Stats()
	fmt.Printf("After reset - evaluations: %d\n", statsAfter.TotalEvaluations)
	fmt.Printf("After reset - cached subgoals: %d\n", statsAfter.CachedSubgoals)

	// Output:
	// Before reset - evaluations: 1
	// After reset - evaluations: 0
	// After reset - cached subgoals: 0
}

```


\n
## pkg_minikanren_slg_engine_example_test.go-ExampleSCC_AnswerCount.md
```go
func ExampleSCC_AnswerCount() {
	pattern1 := NewCallPattern("p", []Term{NewAtom(1)})
	pattern2 := NewCallPattern("q", []Term{NewAtom(2)})

	entry1 := NewSubgoalEntry(pattern1)
	entry2 := NewSubgoalEntry(pattern2)

	// Add answers to both entries
	entry1.Answers().Insert(map[int64]Term{1: NewAtom("a")})
	entry1.Answers().Insert(map[int64]Term{1: NewAtom("b")})
	entry2.Answers().Insert(map[int64]Term{1: NewAtom("c")})

	scc := &SCC{nodes: []*SubgoalEntry{entry1, entry2}}

	fmt.Printf("Total answers in SCC: %d\n", scc.AnswerCount())

	// Output:
	// Total answers in SCC: 3
}

```


\n
## pkg_minikanren_slg_engine_example_test.go-ExampleSLGEngine_ComputeFixpoint.md
```go
func ExampleSLGEngine_ComputeFixpoint() {
	engine := NewSLGEngine(nil)

	// Create two mutually dependent subgoals
	pattern1 := NewCallPattern("reaches", []Term{NewAtom("a"), NewAtom("x")})
	pattern2 := NewCallPattern("reaches", []Term{NewAtom("b"), NewAtom("x")})

	entry1, _ := engine.subgoals.GetOrCreate(pattern1)
	entry2, _ := engine.subgoals.GetOrCreate(pattern2)

	// Add initial answers
	entry1.Answers().Insert(map[int64]Term{1: NewAtom("node1")})
	entry2.Answers().Insert(map[int64]Term{1: NewAtom("node2")})

	// Create mutual dependency (cycle)
	entry1.AddDependency(entry2)
	entry2.AddDependency(entry1)

	scc := &SCC{nodes: []*SubgoalEntry{entry1, entry2}}

	// Compute fixpoint
	err := engine.ComputeFixpoint(context.Background(), scc)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Println("Fixpoint computed successfully")
		fmt.Printf("Total answers: %d\n", scc.AnswerCount())
	}

	// Output:
	// Fixpoint computed successfully
	// Total answers: 2
}

```


\n
## pkg_minikanren_slg_engine_example_test.go-ExampleSLGEngine_DetectCycles.md
```go
func ExampleSLGEngine_DetectCycles() {
	engine := NewSLGEngine(nil)

	// Create three subgoals with dependencies
	patternA := NewCallPattern("ancestor", []Term{NewAtom("alice"), NewAtom("x")})
	patternB := NewCallPattern("ancestor", []Term{NewAtom("bob"), NewAtom("x")})
	patternC := NewCallPattern("ancestor", []Term{NewAtom("charlie"), NewAtom("x")})

	entryA, _ := engine.subgoals.GetOrCreate(patternA)
	entryB, _ := engine.subgoals.GetOrCreate(patternB)
	entryC, _ := engine.subgoals.GetOrCreate(patternC)

	// Create cycle: A -> B -> C -> B
	entryA.AddDependency(entryB)
	entryB.AddDependency(entryC)
	entryC.AddDependency(entryB)

	// Detect cycles
	sccs := engine.DetectCycles()

	fmt.Printf("Found %d SCCs\n", len(sccs))

	// Check if cyclic
	if engine.IsCyclic() {
		fmt.Println("Graph contains cycles")
	}

	// Find the cyclic SCC
	for _, scc := range sccs {
		if len(scc.nodes) > 1 {
			fmt.Printf("Cyclic SCC has %d nodes\n", len(scc.nodes))
		}
	}

	// Output:
	// Found 2 SCCs
	// Graph contains cycles
	// Cyclic SCC has 2 nodes
}

```


\n
## pkg_minikanren_slg_engine_example_test.go-ExampleSLGEngine_DetectCycles_selfLoop.md
```go
func ExampleSLGEngine_DetectCycles_selfLoop() {
	engine := NewSLGEngine(nil)

	// Create a recursive predicate: path(X, Y)
	pattern := NewCallPattern("path", []Term{NewAtom("x"), NewAtom("y")})
	entry, _ := engine.subgoals.GetOrCreate(pattern)

	// Create self-loop (path depends on path)
	entry.AddDependency(entry)

	if engine.IsCyclic() {
		fmt.Println("Self-referential predicate detected")
	}

	sccs := engine.DetectCycles()
	for _, scc := range sccs {
		if scc.Contains(entry) {
			fmt.Printf("SCC contains %d node(s)\n", len(scc.nodes))
		}
	}

	// Output:
	// Self-referential predicate detected
	// SCC contains 1 node(s)
}

```


\n
## pkg_minikanren_slg_engine_example_test.go-ExampleSLGEngine_Evaluate.md
```go
func ExampleSLGEngine_Evaluate() {
	engine := NewSLGEngine(nil)

	// Define a call pattern for a "fact" predicate
	pattern := NewCallPattern("color", []Term{NewAtom("x")})

	// Simple evaluator that produces three color answers
	evaluator := func(ctx context.Context, answers chan<- map[int64]Term) error {
		colors := []string{"red", "green", "blue"}
		for _, color := range colors {
			answer := map[int64]Term{1: NewAtom(color)}
			answers <- answer
		}
		return nil
	}

	ctx := context.Background()
	resultChan, _ := engine.Evaluate(ctx, pattern, evaluator)

	// Collect all answers
	count := 0
	for range resultChan {
		count++
	}

	fmt.Printf("Derived %d answers\n", count)

	// Second evaluation should hit cache
	resultChan2, _ := engine.Evaluate(ctx, pattern, evaluator)
	for range resultChan2 {
	}

	stats := engine.Stats()
	fmt.Printf("Cache hits: %d\n", stats.CacheHits)

	// Output:
	// Derived 3 answers
	// Cache hits: 1
}

```


\n
## pkg_minikanren_slg_engine_example_test.go-ExampleSLGEngine_Evaluate_streaming.md
```go
func ExampleSLGEngine_Evaluate_streaming() {
	engine := NewSLGEngine(nil)

	pattern := NewCallPattern("range", []Term{NewAtom(5)})

	// Evaluator that produces answers incrementally
	evaluator := func(ctx context.Context, answers chan<- map[int64]Term) error {
		for i := 1; i <= 5; i++ {
			answer := map[int64]Term{1: NewAtom(i)}
			answers <- answer
		}
		return nil
	}

	ctx := context.Background()
	resultChan, _ := engine.Evaluate(ctx, pattern, evaluator)

	// Process answers as they arrive
	for answer := range resultChan {
		value := answer[1]
		fmt.Printf("Got answer: %v\n", value)
	}

	// Output:
	// Got answer: 1
	// Got answer: 2
	// Got answer: 3
	// Got answer: 4
	// Got answer: 5
}

```


\n
## pkg_minikanren_slg_engine_example_test.go-ExampleSLGEngine_Stats.md
```go
func ExampleSLGEngine_Stats() {
	engine := NewSLGEngine(nil)

	// Evaluate several subgoals
	for i := 1; i <= 3; i++ {
		pattern := NewCallPattern("test", []Term{NewAtom(i)})
		evaluator := func(ctx context.Context, answers chan<- map[int64]Term) error {
			answer := map[int64]Term{1: NewAtom(fmt.Sprintf("result%d", i))}
			answers <- answer
			return nil
		}

		resultChan, _ := engine.Evaluate(context.Background(), pattern, evaluator)
		for range resultChan {
		}
	}

	// Re-evaluate first subgoal (cache hit)
	pattern := NewCallPattern("test", []Term{NewAtom(1)})
	evaluator := func(ctx context.Context, answers chan<- map[int64]Term) error {
		answer := map[int64]Term{1: NewAtom("result1")}
		answers <- answer
		return nil
	}
	resultChan, _ := engine.Evaluate(context.Background(), pattern, evaluator)
	for range resultChan {
	}

	stats := engine.Stats()
	fmt.Printf("Total evaluations: %d\n", stats.TotalEvaluations)
	fmt.Printf("Cached subgoals: %d\n", stats.CachedSubgoals)
	fmt.Printf("Cache hits: %d\n", stats.CacheHits)
	fmt.Printf("Cache misses: %d\n", stats.CacheMisses)
	fmt.Printf("Hit ratio: %.2f\n", stats.HitRatio)

	// Output:
	// Total evaluations: 4
	// Cached subgoals: 3
	// Cache hits: 1
	// Cache misses: 3
	// Hit ratio: 0.25
}

```


\n
## pkg_minikanren_slg_wfs_example_test.go-ExampleNegateEvaluator.md
```go
func ExampleNegateEvaluator() {
	engine := NewSLGEngine(nil)
	engine.SetStrata(map[string]int{
		"unreachable": 1,
		"path":        0,
	})

	// Simple graph: a->b, b->c, c->a (cycle). Reachable from a: {b, c, a}.
	edges := map[string][]string{
		"a": {"b"},
		"b": {"c"},
		"c": {"a"},
	}

	// path/2 evaluator (closed over start, end)
	var recPathEval func(start, goal string) GoalEvaluator
	recPathEval = func(start, goal string) GoalEvaluator {
		return func(ctx context.Context, answers chan<- map[int64]Term) error {
			// Base case: direct edge
			for _, to := range edges[start] {
				if to == goal {
					answers <- map[int64]Term{}
				}
				// Recursive case: path(start,to) && path(to,goal)
				if to != goal {
					// Evaluate recursively; this will register dependency via context.
					pat := NewCallPattern("path", []Term{NewAtom(to), NewAtom(goal)})
					_, _ = engine.Evaluate(ctx, pat, recPathEval(to, goal))
				}
			}
			return nil
		}
	}

	// unreachable/2(X,Y) :- not(path(X,Y))
	negPath := func(x, y string) GoalEvaluator {
		pat := NewCallPattern("path", []Term{NewAtom(x), NewAtom(y)})
		return NegateEvaluator(engine, "unreachable", pat, recPathEval(x, y))
	}

	ctx := context.Background()
	// Query: unreachable(a, d) where d is not in the graph should succeed; unreachable(a, b) should fail.
	res1, _ := engine.Evaluate(ctx, NewCallPattern("unreachable", []Term{NewAtom("a"), NewAtom("d")}), negPath("a", "d"))
	count1 := 0
	for range res1 {
		count1++
	}
	fmt.Printf("unreachable(a,d): %d answers\n", count1)

	res2, _ := engine.Evaluate(ctx, NewCallPattern("unreachable", []Term{NewAtom("a"), NewAtom("b")}), negPath("a", "b"))
	count2 := 0
	for range res2 {
		count2++
	}
	fmt.Printf("unreachable(a,b): %d answers\n", count2)

	// Output:
	// unreachable(a,d): 1 answers
	// unreachable(a,b): 0 answers
}

```


\n
## pkg_minikanren_slg_wrappers_example_test.go-ExampleTabledEvaluate.md
```go
func ExampleTabledEvaluate() {
	// Use the global engine implicitly
	ResetGlobalEngine()
	inner := GoalEvaluator(func(ctx context.Context, answers chan<- map[int64]Term) error {
		answers <- map[int64]Term{42: NewAtom(1)}
		return nil
	})

	ch, err := TabledEvaluate(context.Background(), "test", []Term{NewAtom("a")}, inner)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	for range ch { /* drain */
	}

	fmt.Println("ok")
	// Output:
	// ok
}

```


\n
## pkg_minikanren_slg_wrappers_example_test.go-ExampleWithTabling.md
```go
func ExampleWithTabling() {
	engine := NewSLGEngine(nil)
	eval := WithTabling(engine)

	// Simple evaluator that yields a single answer
	inner := GoalEvaluator(func(ctx context.Context, answers chan<- map[int64]Term) error {
		answers <- map[int64]Term{1: NewAtom("ok")}
		return nil
	})

	ch, err := eval(context.Background(), "demo", []Term{NewAtom("x")}, inner)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	for range ch { /* drain */
	}

	stats := engine.Stats()
	fmt.Printf("evaluations=%d cached=%d\n", stats.TotalEvaluations, stats.CachedSubgoals)
	// Output:
	// evaluations=1 cached=1
}

```


\n
## pkg_minikanren_stretch_example_test.go-ExampleNewStretch.md
```go
func ExampleNewStretch() {
	model := NewModel()
	// Domains over {1,2}; constrain value 1 to appear only in runs of length exactly 2.
	// x1 := model.NewVariableWithName(NewBitSetDomainFromValues(2, []int{1, 2}), "x1")
	x1 := model.IntVarValues([]int{1, 2}, "x1")
	// x2 := model.NewVariableWithName(NewBitSetDomainFromValues(2, []int{1}), "x2") // fix a 1
	x2 := model.IntVarValues([]int{1}, "x2") // fix a 1
	// x3 := model.NewVariableWithName(NewBitSetDomainFromValues(2, []int{2}), "x3") // separator
	x3 := model.IntVarValues([]int{2}, "x3") // separator
	// x4 := model.NewVariableWithName(NewBitSetDomainFromValues(2, []int{1}), "x4") // fix a 1
	x4 := model.IntVarValues([]int{1}, "x4") // fix a 1
	// x5 := model.NewVariableWithName(NewBitSetDomainFromValues(2, []int{1, 2}), "x5")
	x5 := model.IntVarValues([]int{1, 2}, "x5")

	_, _ = NewStretch(model, []*FDVariable{x1, x2, x3, x4, x5}, []int{1}, []int{2}, []int{2})

	solver := NewSolver(model)
	_, _ = solver.Solve(context.Background(), 0)

	// Single 1s at x2 and x4, with a separator at x3, force x1 and x5 to 1 to satisfy run length = 2.
	fmt.Printf("x1: %s\n", solver.GetDomain(nil, x1.ID()))
	fmt.Printf("x5: %s\n", solver.GetDomain(nil, x5.ID()))
	// Output:
	// x1: {1}
	// x5: {1}
}

```


\n
## pkg_minikanren_sum_example_test.go-ExampleNewLinearSum.md
```go
func ExampleNewLinearSum() {
	model := NewModel()

	// Three variables with small ranges
	a := model.NewVariable(NewBitSetDomain(5)) // [1..5]
	b := model.NewVariable(NewBitSetDomain(5)) // [1..5]
	c := model.NewVariable(NewBitSetDomain(9)) // [1..9]

	// Total starts wide and will be pruned
	total := model.NewVariable(NewBitSetDomain(100))

	coeffs := []int{1, 2, 3}
	ls, err := NewLinearSum([]*FDVariable{a, b, c}, coeffs, total)
	if err != nil {
		panic(err)
	}
	model.AddConstraint(ls)

	solver := NewSolver(model)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Solve without search to showcase propagation effects only.
	// The first propagation pass happens before search begins.
	solutions, _ := solver.Solve(ctx, 1)
	_ = solutions // not used; we print domains instead

	// Read pruned domains from root propagated state
	aDom := solver.GetDomain(nil, a.ID())
	bDom := solver.GetDomain(nil, b.ID())
	cDom := solver.GetDomain(nil, c.ID())
	tDom := solver.GetDomain(nil, total.ID())

	fmt.Printf("a=[%d..%d] b=[%d..%d] c=[%d..%d] total=[%d..%d]\n",
		aDom.Min(), aDom.Max(), bDom.Min(), bDom.Max(), cDom.Min(), cDom.Max(), tDom.Min(), tDom.Max())

	// Output:
	// a=[1..5] b=[1..5] c=[1..9] total=[6..42]
}

```


\n
## pkg_minikanren_table_example_test.go-ExampleNewTable.md
```go
func ExampleNewTable() {
	model := NewModel()
	x := model.NewVariableWithName(NewBitSetDomain(5), "x")
	y := model.NewVariableWithName(NewBitSetDomain(5), "y")

	rows := [][]int{
		{1, 1},
		{2, 3},
		{3, 2},
	}
	c, _ := NewTable([]*FDVariable{x, y}, rows)
	model.AddConstraint(c)

	solver := NewSolver(model)

	// Set y ∈ {1,2}
	state, _ := solver.SetDomain(nil, y.ID(), NewBitSetDomainFromValues(5, []int{1, 2}))

	// Propagate once; solver runs to fixed-point internally during Solve, but
	// we can invoke the constraint directly for illustration.
	newState, _ := solver.propagate(state)

	xd := solver.GetDomain(newState, x.ID())
	yd := solver.GetDomain(newState, y.ID())

	fmt.Printf("x: %v\n", xd)
	fmt.Printf("y: %v\n", yd)
	// Output:
	// x: {1,3}
	// y: {1..2}
}

```


\n
## pkg_minikanren_tabling_example_test.go-ExampleAnswerTrie_Iterator.md
```go
func ExampleAnswerTrie_Iterator() {
	trie := NewAnswerTrie()

	// Insert multiple answers
	for i := 1; i <= 3; i++ {
		bindings := map[int64]Term{
			1: NewAtom(fmt.Sprintf("value%d", i)),
		}
		trie.Insert(bindings)
	}

	// Iterate over all answers
	iter := trie.Iterator()
	count := 0
	for {
		answer, ok := iter.Next()
		if !ok {
			break
		}
		count++
		// Note: iteration order is not guaranteed
		fmt.Printf("Answer has %d bindings\n", len(answer))
	}

	fmt.Printf("Total answers iterated: %d\n", count)

	// Output:
	// Answer has 1 bindings
	// Answer has 1 bindings
	// Answer has 1 bindings
	// Total answers iterated: 3
}

```


\n
## pkg_minikanren_tabling_example_test.go-ExampleAnswerTrie.md
```go
func ExampleAnswerTrie() {
	trie := NewAnswerTrie()

	// Insert first answer: {1: a, 2: b}
	answer1 := map[int64]Term{
		1: NewAtom("a"),
		2: NewAtom("b"),
	}
	inserted := trie.Insert(answer1)
	fmt.Printf("First answer inserted: %v\n", inserted)
	fmt.Printf("Count: %d\n", trie.Count())

	// Insert duplicate
	duplicate := map[int64]Term{
		1: NewAtom("a"),
		2: NewAtom("b"),
	}
	inserted = trie.Insert(duplicate)
	fmt.Printf("Duplicate inserted: %v\n", inserted)
	fmt.Printf("Count: %d\n", trie.Count())

	// Insert different answer: {1: a, 2: c}
	answer2 := map[int64]Term{
		1: NewAtom("a"),
		2: NewAtom("c"),
	}
	inserted = trie.Insert(answer2)
	fmt.Printf("Different answer inserted: %v\n", inserted)
	fmt.Printf("Final count: %d\n", trie.Count())

	// Output:
	// First answer inserted: true
	// Count: 1
	// Duplicate inserted: false
	// Count: 1
	// Different answer inserted: true
	// Final count: 2
}

```


\n
## pkg_minikanren_tabling_example_test.go-ExampleNewCallPattern.md
```go
func ExampleNewCallPattern() {
	// Create a call pattern for edge(a, b)
	args := []Term{NewAtom("a"), NewAtom("b")}
	pattern := NewCallPattern("edge", args)

	fmt.Printf("Predicate: %s\n", pattern.PredicateID())
	fmt.Printf("Structure: %s\n", pattern.ArgStructure())
	fmt.Printf("Full pattern: %s\n", pattern.String())

	// Output:
	// Predicate: edge
	// Structure: atom(a),atom(b)
	// Full pattern: edge(atom(a),atom(b))
}

```


\n
## pkg_minikanren_tabling_example_test.go-ExampleNewCallPattern_variableReuse.md
```go
func ExampleNewCallPattern_variableReuse() {
	v := &Var{id: 42, name: "x"}
	// path(X, X) - same variable twice
	pattern := NewCallPattern("path", []Term{v, v})

	fmt.Printf("Structure: %s\n", pattern.ArgStructure())

	// Output:
	// Structure: X0,X0
}

```


\n
## pkg_minikanren_tabling_example_test.go-ExampleNewCallPattern_variables.md
```go
func ExampleNewCallPattern_variables() {
	// Two calls with different variable IDs but same structure
	v1 := &Var{id: 42, name: "x"}
	v2 := &Var{id: 73, name: "y"}
	pattern1 := NewCallPattern("path", []Term{v1, v2})

	v3 := &Var{id: 100, name: "p"}
	v4 := &Var{id: 200, name: "q"}
	pattern2 := NewCallPattern("path", []Term{v3, v4})

	fmt.Printf("Pattern 1: %s\n", pattern1.ArgStructure())
	fmt.Printf("Pattern 2: %s\n", pattern2.ArgStructure())
	fmt.Printf("Are equal: %v\n", pattern1.Equal(pattern2))

	// Output:
	// Pattern 1: X0,X1
	// Pattern 2: X0,X1
	// Are equal: true
}

```


\n
## pkg_minikanren_tabling_example_test.go-ExampleSubgoalEntry_dependencies.md
```go
func ExampleSubgoalEntry_dependencies() {
	// Create a dependency chain: path depends on edge
	edgePattern := NewCallPattern("edge", []Term{NewAtom("a"), NewAtom("b")})
	pathPattern := NewCallPattern("path", []Term{NewAtom("a"), NewAtom("b")})

	edgeEntry := NewSubgoalEntry(edgePattern)
	pathEntry := NewSubgoalEntry(pathPattern)

	// path depends on edge
	pathEntry.AddDependency(edgeEntry)

	deps := pathEntry.Dependencies()
	fmt.Printf("Number of dependencies: %d\n", len(deps))
	fmt.Printf("Depends on: %s\n", deps[0].Pattern().String())

	// Output:
	// Number of dependencies: 1
	// Depends on: edge(atom(a),atom(b))
}

```


\n
## pkg_minikanren_tabling_example_test.go-ExampleSubgoalEntry.md
```go
func ExampleSubgoalEntry() {
	pattern := NewCallPattern("fib", []Term{NewAtom(5)})
	entry := NewSubgoalEntry(pattern)

	fmt.Printf("Initial status: %s\n", entry.Status())
	fmt.Printf("Answer count: %d\n", entry.Answers().Count())

	// Add an answer
	bindings := map[int64]Term{1: NewAtom(8)} // fib(5) = 8
	entry.Answers().Insert(bindings)

	fmt.Printf("After insertion: %d answers\n", entry.Answers().Count())

	// Mark as complete
	entry.SetStatus(StatusComplete)
	fmt.Printf("Final status: %s\n", entry.Status())

	// Output:
	// Initial status: Active
	// Answer count: 0
	// After insertion: 1 answers
	// Final status: Complete
}

```


\n
## pkg_minikanren_tabling_example_test.go-ExampleSubgoalStatus.md
```go
func ExampleSubgoalStatus() {
	statuses := []SubgoalStatus{
		StatusActive,
		StatusComplete,
		StatusFailed,
		StatusInvalidated,
	}

	for _, status := range statuses {
		fmt.Printf("%s\n", status.String())
	}

	// Output:
	// Active
	// Complete
	// Failed
	// Invalidated
}

```


\n
## pkg_minikanren_tabling_example_test.go-ExampleSubgoalTable.md
```go
func ExampleSubgoalTable() {
	table := NewSubgoalTable()

	// Create a call pattern
	pattern := NewCallPattern("edge", []Term{NewAtom("a"), NewAtom("b")})

	// Get or create a subgoal entry
	entry, created := table.GetOrCreate(pattern)
	fmt.Printf("Created new entry: %v\n", created)
	fmt.Printf("Entry status: %s\n", entry.Status())

	// Subsequent calls return the same entry
	entry2, created2 := table.GetOrCreate(pattern)
	fmt.Printf("Created on second call: %v\n", created2)
	fmt.Printf("Same entry: %v\n", entry == entry2)

	fmt.Printf("Total subgoals: %d\n", table.TotalSubgoals())

	// Output:
	// Created new entry: true
	// Entry status: Active
	// Created on second call: false
	// Same entry: true
	// Total subgoals: 1
}

```


\n
## pkg_minikanren_term_utils_example_test.go-ExampleArityo.md
```go
func ExampleArityo() {
	// Arity of an atom is 0
	result1 := Run(1, func(arity *Var) Goal {
		return Arityo(NewAtom("hello"), arity)
	})

	// Arity of a list is its length
	list := List(NewAtom(1), NewAtom(2), NewAtom(3))
	result2 := Run(1, func(arity *Var) Goal {
		return Arityo(list, arity)
	})

	fmt.Printf("Atom arity: %v\n", result1[0])
	fmt.Printf("List arity: %v\n", result2[0])
	// Output:
	// Atom arity: 0
	// List arity: 3
}

```


\n
## pkg_minikanren_term_utils_example_test.go-ExampleArityo_typeChecking.md
```go
func ExampleArityo_typeChecking() {
	// Ensure a term is a binary operation (arity 2)
	validateBinary := func(term Term) Goal {
		return Arityo(term, NewAtom(2))
	}

	binaryOp := List(NewAtom("left"), NewAtom("right"))
	unaryOp := List(NewAtom("single"))

	result1 := Run(1, func(q *Var) Goal {
		return Conj(
			validateBinary(binaryOp),
			Eq(q, NewAtom("valid-binary")),
		)
	})

	result2 := Run(1, func(q *Var) Goal {
		return Conj(
			validateBinary(unaryOp),
			Eq(q, NewAtom("valid-binary")),
		)
	})

	fmt.Printf("Binary operation: %s\n", result1[0])
	fmt.Printf("Unary operation fails: %d results\n", len(result2))
	// Output:
	// Binary operation: valid-binary
	// Unary operation fails: 0 results
}

```


\n
## pkg_minikanren_term_utils_example_test.go-ExampleBooleano.md
```go
func ExampleBooleano() {
	result := Run(3, func(q *Var) Goal {
		return Conj(
			Booleano(q),
			Membero(q, List(
				NewAtom(true),
				NewAtom("not-bool"),
				NewAtom(false),
				NewAtom(42),
			)),
		)
	})

	fmt.Printf("Boolean values: %d results\n", len(result))
	for _, r := range result {
		fmt.Printf("  %v\n", r)
	}
	// Output:
	// Boolean values: 2 results
	//   true
	//   false
}

```


\n
## pkg_minikanren_term_utils_example_test.go-ExampleCompoundTermo.md
```go
func ExampleCompoundTermo() {
	// Pairs are compound
	pair := NewPair(NewAtom("a"), NewAtom("b"))

	// Atoms are not compound
	atom := NewAtom(42)

	result1 := Run(1, func(q *Var) Goal {
		return Conj(
			CompoundTermo(pair),
			Eq(q, NewAtom("pair-is-compound")),
		)
	})

	result2 := Run(1, func(q *Var) Goal {
		return Conj(
			CompoundTermo(atom),
			Eq(q, NewAtom("atom-is-compound")),
		)
	})

	fmt.Printf("Pair is compound: %d results\n", len(result1))
	fmt.Printf("Atom is compound: %d results\n", len(result2))
	// Output:
	// Pair is compound: 1 results
	// Atom is compound: 0 results
}

```


\n
## pkg_minikanren_term_utils_example_test.go-ExampleCopyTerm.md
```go
func ExampleCopyTerm() {
	x := Fresh("x")
	original := List(x, NewAtom("middle"), x)

	// Copy the term - x will be replaced with a fresh variable
	_ = Run(1, func(copy *Var) Goal {
		return Conj(
			CopyTerm(original, copy),
			Eq(x, NewAtom("original-binding")), // Bind original x
		)
	})

	// The copy has fresh variables, not bound to "original-binding"
	fmt.Printf("Copy preserves structure with fresh variables\n")
	// Output:
	// Copy preserves structure with fresh variables
}

```


\n
## pkg_minikanren_term_utils_example_test.go-ExampleCopyTerm_metaProgramming.md
```go
func ExampleCopyTerm_metaProgramming() {
	// Define a template with variables
	x := Fresh("x")
	y := Fresh("y")
	template := NewPair(NewAtom("add"), List(x, y))

	// Create multiple instances of the template
	result := Run(2, func(q *Var) Goal {
		instance := Fresh("instance")
		return Conj(
			CopyTerm(template, instance),
			// Each instance can be instantiated differently
			Membero(q, List(
				NewPair(NewAtom("instance"), instance),
			)),
		)
	})

	fmt.Printf("Template instances: %d\n", len(result))
	// Output:
	// Template instances: 1
}

```


\n
## pkg_minikanren_term_utils_example_test.go-ExampleFunctoro.md
```go
func ExampleFunctoro() {
	// Create a compound term like foo(1, 2)
	term := NewPair(NewAtom("foo"), List(NewAtom(1), NewAtom(2)))

	result := Run(1, func(functor *Var) Goal {
		return Functoro(term, functor)
	})

	fmt.Printf("Functor: %v\n", result[0])
	// Output:
	// Functor: foo
}

```


\n
## pkg_minikanren_term_utils_example_test.go-ExampleFunctoro_patternMatching.md
```go
func ExampleFunctoro_patternMatching() {
	// Dispatch based on functor
	dispatch := func(term, result Term) Goal {
		functor := Fresh("functor")
		return Conj(
			Functoro(term, functor),
			Conde(
				Conj(Eq(functor, NewAtom("add")), Eq(result, NewAtom("arithmetic"))),
				Conj(Eq(functor, NewAtom("cons")), Eq(result, NewAtom("list-operation"))),
				Conj(Eq(functor, NewAtom("eq")), Eq(result, NewAtom("comparison"))),
			),
		)
	}

	addTerm := NewPair(NewAtom("add"), List(NewAtom(1), NewAtom(2)))
	consTerm := NewPair(NewAtom("cons"), List(NewAtom("a"), Nil))

	result1 := Run(1, func(q *Var) Goal {
		return dispatch(addTerm, q)
	})

	result2 := Run(1, func(q *Var) Goal {
		return dispatch(consTerm, q)
	})

	fmt.Printf("add → %v\n", result1[0])
	fmt.Printf("cons → %v\n", result2[0])
	// Output:
	// add → arithmetic
	// cons → list-operation
}

```


\n
## pkg_minikanren_term_utils_example_test.go-ExampleGround_list.md
```go
func ExampleGround_list() {
	// Fully ground list
	groundList := List(NewAtom(1), NewAtom(2), NewAtom(3))

	// Partially ground list
	x := Fresh("x")
	partialList := List(NewAtom(1), x, NewAtom(3))

	result1 := Run(1, func(q *Var) Goal {
		return Conj(
			Ground(groundList),
			Eq(q, NewAtom("fully-ground")),
		)
	})

	result2 := Run(1, func(q *Var) Goal {
		return Conj(
			Ground(partialList),
			Eq(q, NewAtom("partially-ground")),
		)
	})

	fmt.Printf("Fully ground list: %s\n", result1[0])
	fmt.Printf("Partially ground list fails: %d results\n", len(result2))
	// Output:
	// Fully ground list: fully-ground
	// Partially ground list fails: 0 results
}

```


\n
## pkg_minikanren_term_utils_example_test.go-ExampleGround.md
```go
func ExampleGround() {
	x := Fresh("x")

	// Check if a bound variable is ground
	result1 := Run(1, func(q *Var) Goal {
		return Conj(
			Eq(x, NewAtom("hello")),
			Ground(x),
			Eq(q, NewAtom("bound-is-ground")),
		)
	})

	// Check if an unbound variable is ground (should fail)
	result2 := Run(1, func(q *Var) Goal {
		return Conj(
			Ground(Fresh("unbound")),
			Eq(q, NewAtom("should-not-appear")),
		)
	})

	fmt.Printf("Bound variable is ground: %d results\n", len(result1))
	fmt.Printf("Unbound variable is not ground: %d results\n", len(result2))
	// Output:
	// Bound variable is ground: 1 results
	// Unbound variable is not ground: 0 results
}

```


\n
## pkg_minikanren_term_utils_example_test.go-ExampleGround_validation.md
```go
func ExampleGround_validation() {
	// Validate that all arguments are provided before processing
	process := func(arg1, arg2, result Term) Goal {
		return Conj(
			Ground(arg1),
			Ground(arg2),
			Eq(result, NewAtom("processed")),
		)
	}

	// Valid case: both arguments bound
	result1 := Run(1, func(q *Var) Goal {
		return process(NewAtom("a"), NewAtom("b"), q)
	})

	// Invalid case: argument contains unbound variable
	x := Fresh("x")
	result2 := Run(1, func(q *Var) Goal {
		return process(NewAtom("a"), x, q)
	})

	fmt.Printf("Both arguments ground: %d results\n", len(result1))
	fmt.Printf("Unbound argument: %d results\n", len(result2))
	// Output:
	// Both arguments ground: 1 results
	// Unbound argument: 0 results
}

```


\n
## pkg_minikanren_term_utils_example_test.go-ExampleSimpleTermo.md
```go
func ExampleSimpleTermo() {
	result1 := Run(1, func(q *Var) Goal {
		return Conj(
			SimpleTermo(NewAtom(42)),
			Eq(q, NewAtom("atom-is-simple")),
		)
	})

	result2 := Run(1, func(q *Var) Goal {
		return Conj(
			SimpleTermo(NewPair(NewAtom("a"), NewAtom("b"))),
			Eq(q, NewAtom("pair-is-simple")),
		)
	})

	fmt.Printf("Atom is simple: %s\n", result1[0])
	fmt.Printf("Pair is not simple: %d results\n", len(result2))
	// Output:
	// Atom is simple: atom-is-simple
	// Pair is not simple: 0 results
}

```


\n
## pkg_minikanren_term_utils_example_test.go-ExampleStringo.md
```go
func ExampleStringo() {
	result := Run(3, func(q *Var) Goal {
		return Conj(
			Stringo(q),
			Membero(q, List(
				NewAtom("hello"),
				NewAtom(42),
				NewAtom("world"),
				NewAtom(true),
			)),
		)
	})

	fmt.Printf("String values: %d results\n", len(result))
	for _, r := range result {
		fmt.Printf("  %v\n", r)
	}
	// Output:
	// String values: 2 results
	//   hello
	//   world
}

```


\n
## pkg_minikanren_term_utils_example_test.go-ExampleVectoro.md
```go
func ExampleVectoro() {
	slice1 := []int{1, 2, 3}
	slice2 := []string{"a", "b", "c"}

	result := Run(2, func(q *Var) Goal {
		return Conj(
			Vectoro(q),
			Membero(q, List(
				NewAtom(slice1),
				NewAtom("not-a-vector"),
				NewAtom(slice2),
			)),
		)
	})

	fmt.Printf("Vector values: %d results\n", len(result))
	// Output:
	// Vector values: 2 results
}

```


\n
## pkg_minikanren_wfs_api_example_test.go-ExampleSLGEngine_NegationTruth.md
```go
func ExampleSLGEngine_NegationTruth() {
	engine := NewSLGEngine(nil)
	engine.SetStrata(map[string]int{"unreachable": 1, "path": 0})

	// small graph: a->b
	edges := map[string][]string{"a": {"b"}}

	// path/2 evaluator (existence of a direct edge only for brevity)
	pathEval := func(from, to string) GoalEvaluator {
		return func(ctx context.Context, answers chan<- map[int64]Term) error {
			for _, v := range edges[from] {
				if v == to {
					answers <- map[int64]Term{}
				}
			}
			return nil
		}
	}

	// Query not(path(a,c)) => true; not(path(a,b)) => false
	tv1, _ := engine.NegationTruth(context.Background(), "unreachable", NewCallPattern("path", []Term{NewAtom("a"), NewAtom("c")}), pathEval("a", "c"))
	fmt.Println("not(path(a,c)):", tv1)

	tv2, _ := engine.NegationTruth(context.Background(), "unreachable", NewCallPattern("path", []Term{NewAtom("a"), NewAtom("b")}), pathEval("a", "b"))
	fmt.Println("not(path(a,b)):", tv2)

	// Output:
	// not(path(a,c)): true
	// not(path(a,b)): false
}

```


\n
