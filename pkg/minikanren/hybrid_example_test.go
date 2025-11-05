// Package minikanren provides example usage of the hybrid solver framework.
// These examples demonstrate how to combine relational and FD solving.
package minikanren

import (
	"fmt"
)

// ExampleNewUnifiedStore demonstrates creating a unified store for hybrid solving.
// The UnifiedStore holds both relational bindings and FD domains,
// enabling attributed variables.
//
// IMPORTANT: This is a low-level API. For complete hybrid solving,
// see ExampleHybridSolver_bidirectionalPropagation.
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

// ExampleUnifiedStore_AddBinding shows how to add relational bindings to the store.
// In real usage, bindings are created through unification in the relational solver.
//
// IMPORTANT: Variable IDs must be consistent when using both bindings and domains
// on the same variable.
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

// ExampleUnifiedStore_SetDomain shows how to set FD domains in the store.
// Domains are pruned by the FD solver during propagation.
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

// ExampleNewHybridSolver demonstrates creating a hybrid solver with both
// relational and FD plugins. This enables solving problems that require
// both types of reasoning.
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

// ExampleHybridSolver_Propagate shows how to run hybrid propagation.
// The solver iterates through all plugins until a fixed point is reached.
//
// This example shows FD-only propagation. For true hybrid solving with both
// relational and FD constraints, see ExampleHybridSolver_bidirectionalPropagation.
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

// ExampleFDPlugin demonstrates using the FD plugin to wrap the
// Phase 2 FD propagation system for use in hybrid solving.
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

// ExampleRelationalPlugin demonstrates using the relational plugin to check
// miniKanren constraints (type constraints, disequality, etc.).
//
// This is a standalone relational example. For true hybrid solving,
// see ExampleHybridSolver_bidirectionalPropagation.
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

// ExampleRelationalPlugin_promoteSingletons shows how the relational plugin
// promotes FD singleton domains to relational bindings, enabling cross-solver
// propagation.
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

// ExampleHybridSolver_bidirectionalPropagation demonstrates TRUE hybrid solving:
// relational bindings influencing FD domains AND FD propagation creating bindings.
//
// This showcases the key innovation of Phase 3: attributed variables that
// participate in both relational and FD reasoning simultaneously.
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

// ExampleHybridSolver_realWorldScheduling shows a practical hybrid problem:
// combining miniKanren type constraints with FD temporal constraints.
//
// Problem: Schedule 3 tasks with type requirements and time constraints.
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
