package minikanren_test

import (
	"context"
	"fmt"
	"time"

	"github.com/gitrdm/gokanlogic/pkg/minikanren"
)

// ExampleNewModel demonstrates creating a constraint model and adding variables.
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

// ExampleModel_NewVariables demonstrates creating multiple variables at once.
// This is the most common pattern for array-based problems.
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

// ExampleModel_NewVariablesWithNames demonstrates creating named variables
// for easier debugging and error messages.
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

// ExampleNewSolver demonstrates solving a simple constraint satisfaction problem.
// This example shows the complete workflow: model construction, solving, and
// extracting solutions.
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

// ExampleModel_Validate demonstrates validating a model before solving.
// Validation catches common errors early with clear messages.
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

// ExampleFDVariable_IsBound demonstrates checking if a variable is bound to a value.
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

// ExampleSolverConfig demonstrates customizing solver behavior with heuristics.
// Different heuristics can dramatically affect solving performance.
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

// ExampleSolver_parallelSearch demonstrates the correct approach for parallel search.
// Multiple workers share the same immutable Model but use independent SolverState chains.
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
