// Package minikanren provides tests for constraint reification.
package minikanren

import (
	"context"
	"testing"
)

// TestReifiedConstraint_Basic tests basic reification with Arithmetic constraints.
func TestReifiedConstraint_Basic(t *testing.T) {
	// Create model: X ∈ {1..5}, Y ∈ {1..5}, B ∈ {1,2}
	// Reify: B = (X + 0 = Y), i.e., B=2 iff X=Y
	model := NewModel()
	x := model.NewVariable(NewBitSetDomain(5))
	y := model.NewVariable(NewBitSetDomain(5))
	b := model.NewVariable(NewBitSetDomain(2)) // {1, 2} for {false, true}

	// Create constraint X + 0 = Y
	arith, err := NewArithmetic(x, y, 0)
	if err != nil {
		t.Fatalf("NewArithmetic failed: %v", err)
	}

	// Reify the constraint
	reified, err := NewReifiedConstraint(arith, b)
	if err != nil {
		t.Fatalf("NewReifiedConstraint failed: %v", err)
	}

	model.AddConstraint(reified)

	solver := NewSolver(model)
	ctx := context.Background()

	solutions, err := solver.Solve(ctx, 100)
	if err != nil {
		t.Fatalf("Solve failed: %v", err)
	}

	// Verify solutions
	// When B=2 (true): X=Y
	// When B=1 (false): X≠Y
	for _, sol := range solutions {
		xVal := sol[x.ID()]
		yVal := sol[y.ID()]
		bVal := sol[b.ID()]

		if bVal == 2 {
			// Boolean is true, constraint must be satisfied
			if xVal != yVal {
				t.Errorf("Solution %v: B=2 (true) but X(%d) ≠ Y(%d)", sol, xVal, yVal)
			}
		} else if bVal == 1 {
			// Boolean is false, constraint must be violated
			if xVal == yVal {
				t.Errorf("Solution %v: B=1 (false) but X(%d) = Y(%d)", sol, xVal, yVal)
			}
		} else {
			t.Errorf("Solution %v: B has invalid value %d (expected 1 or 2)", sol, bVal)
		}
	}

	// Count solutions where B=2 (X=Y) and B=1 (X≠Y)
	trueCount := 0
	falseCount := 0
	for _, sol := range solutions {
		if sol[b.ID()] == 2 {
			trueCount++
		} else {
			falseCount++
		}
	}

	// Should have 5 solutions with X=Y (B=2) and 20 solutions with X≠Y (B=1)
	if trueCount != 5 {
		t.Errorf("Expected 5 solutions with B=2 (X=Y), got %d", trueCount)
	}
	if falseCount != 20 {
		t.Errorf("Expected 20 solutions with B=1 (X≠Y), got %d", falseCount)
	}
}

// TestReifiedConstraint_ForcedTrue tests forcing a reified constraint to be true.
func TestReifiedConstraint_ForcedTrue(t *testing.T) {
	// X ∈ {1..5}, Y ∈ {1..5}, B = {2} (forced true)
	// Reify: B = (X + 0 = Y)
	// Should get only solutions where X = Y
	model := NewModel()
	x := model.NewVariable(NewBitSetDomain(5))
	y := model.NewVariable(NewBitSetDomain(5))
	b := model.NewVariable(NewBitSetDomainFromValues(2, []int{2})) // Force true

	arith, err := NewArithmetic(x, y, 0)
	if err != nil {
		t.Fatalf("NewArithmetic failed: %v", err)
	}

	reified, err := NewReifiedConstraint(arith, b)
	if err != nil {
		t.Fatalf("NewReifiedConstraint failed: %v", err)
	}

	model.AddConstraint(reified)

	solver := NewSolver(model)
	ctx := context.Background()

	solutions, err := solver.Solve(ctx, 100)
	if err != nil {
		t.Fatalf("Solve failed: %v", err)
	}

	// All solutions must have X = Y
	if len(solutions) != 5 {
		t.Errorf("Expected 5 solutions, got %d", len(solutions))
	}

	for _, sol := range solutions {
		if sol[x.ID()] != sol[y.ID()] {
			t.Errorf("Solution %v: X ≠ Y but B is forced to 2 (true)", sol)
		}
	}
}

// TestReifiedConstraint_ForcedFalse tests forcing a reified constraint to be false.
func TestReifiedConstraint_ForcedFalse(t *testing.T) {
	// X ∈ {1..5}, Y ∈ {1..5}, B = {1} (forced false)
	// Reify: B = (X + 0 = Y)
	// Should get only solutions where X ≠ Y
	model := NewModel()
	x := model.NewVariable(NewBitSetDomain(5))
	y := model.NewVariable(NewBitSetDomain(5))
	b := model.NewVariable(NewBitSetDomainFromValues(2, []int{1})) // Force false

	arith, err := NewArithmetic(x, y, 0)
	if err != nil {
		t.Fatalf("NewArithmetic failed: %v", err)
	}

	reified, err := NewReifiedConstraint(arith, b)
	if err != nil {
		t.Fatalf("NewReifiedConstraint failed: %v", err)
	}

	model.AddConstraint(reified)

	solver := NewSolver(model)
	ctx := context.Background()

	solutions, err := solver.Solve(ctx, 100)
	if err != nil {
		t.Fatalf("Solve failed: %v", err)
	}

	// All solutions must have X ≠ Y
	if len(solutions) != 20 {
		t.Errorf("Expected 20 solutions, got %d", len(solutions))
	}

	for _, sol := range solutions {
		if sol[x.ID()] == sol[y.ID()] {
			t.Errorf("Solution %v: X = Y but B is forced to 1 (false)", sol)
		}
	}
}

// TestReifiedConstraint_Inequality tests reification with inequality constraints.
func TestReifiedConstraint_Inequality(t *testing.T) {
	// X ∈ {1..5}, Y ∈ {1..5}, B ∈ {1,2}
	// Reify: B = (X < Y)
	model := NewModel()
	x := model.NewVariable(NewBitSetDomain(5))
	y := model.NewVariable(NewBitSetDomain(5))
	b := model.NewVariable(NewBitSetDomain(2))

	ineq, err := NewInequality(x, y, LessThan)
	if err != nil {
		t.Fatalf("NewInequality failed: %v", err)
	}

	reified, err := NewReifiedConstraint(ineq, b)
	if err != nil {
		t.Fatalf("NewReifiedConstraint failed: %v", err)
	}

	model.AddConstraint(reified)

	solver := NewSolver(model)
	ctx := context.Background()

	solutions, err := solver.Solve(ctx, 100)
	if err != nil {
		t.Fatalf("Solve failed: %v", err)
	}

	// Verify each solution
	for _, sol := range solutions {
		xVal := sol[x.ID()]
		yVal := sol[y.ID()]
		bVal := sol[b.ID()]

		if bVal == 2 {
			// Boolean is true, X < Y must hold
			if xVal >= yVal {
				t.Errorf("Solution %v: B=2 (true) but X(%d) >= Y(%d)", sol, xVal, yVal)
			}
		} else if bVal == 1 {
			// Boolean is false, X < Y must NOT hold
			if xVal < yVal {
				t.Errorf("Solution %v: B=1 (false) but X(%d) < Y(%d)", sol, xVal, yVal)
			}
		}
	}

	// Count: should have 10 solutions with X<Y (B=2) and 15 with X>=Y (B=1)
	trueCount := 0
	falseCount := 0
	for _, sol := range solutions {
		if sol[b.ID()] == 2 {
			trueCount++
		} else {
			falseCount++
		}
	}

	if trueCount != 10 {
		t.Errorf("Expected 10 solutions with B=2 (X<Y), got %d", trueCount)
	}
	if falseCount != 15 {
		t.Errorf("Expected 15 solutions with B=1 (X>=Y), got %d", falseCount)
	}
}

// TestReifiedConstraint_AllDifferent tests reification with AllDifferent.
func TestReifiedConstraint_AllDifferent(t *testing.T) {
	// X, Y, Z ∈ {1..3}, B ∈ {1,2}
	// Reify: B = AllDifferent(X, Y, Z)
	model := NewModel()
	x := model.NewVariable(NewBitSetDomain(3))
	y := model.NewVariable(NewBitSetDomain(3))
	z := model.NewVariable(NewBitSetDomain(3))
	b := model.NewVariable(NewBitSetDomain(2))

	allDiff, err := NewAllDifferent([]*FDVariable{x, y, z})
	if err != nil {
		t.Fatalf("NewAllDifferent failed: %v", err)
	}

	reified, err := NewReifiedConstraint(allDiff, b)
	if err != nil {
		t.Fatalf("NewReifiedConstraint failed: %v", err)
	}

	model.AddConstraint(reified)

	solver := NewSolver(model)
	ctx := context.Background()

	solutions, err := solver.Solve(ctx, 100)
	if err != nil {
		t.Fatalf("Solve failed: %v", err)
	}

	// Verify each solution
	for _, sol := range solutions {
		xVal := sol[x.ID()]
		yVal := sol[y.ID()]
		zVal := sol[z.ID()]
		bVal := sol[b.ID()]

		allDifferent := (xVal != yVal) && (yVal != zVal) && (xVal != zVal)

		if bVal == 2 {
			// Boolean is true, all must be different
			if !allDifferent {
				t.Errorf("Solution %v: B=2 (true) but not all different", sol)
			}
		} else if bVal == 1 {
			// Boolean is false, not all different
			if allDifferent {
				t.Errorf("Solution %v: B=1 (false) but all are different", sol)
			}
		}
	}

	// Count: 6 solutions with all different (3!), 21 with some equal
	trueCount := 0
	falseCount := 0
	for _, sol := range solutions {
		if sol[b.ID()] == 2 {
			trueCount++
		} else {
			falseCount++
		}
	}

	if trueCount != 6 {
		t.Errorf("Expected 6 solutions with B=2 (all different), got %d", trueCount)
	}
	if falseCount != 21 {
		t.Errorf("Expected 21 solutions with B=1 (not all different), got %d", falseCount)
	}
}

// TestReifiedConstraint_Conflict tests reification with impossible constraints.
func TestReifiedConstraint_Conflict(t *testing.T) {
	// X = {1}, Y = {2}, B = {2} (forced true)
	// Reify: B = (X + 0 = Y)
	// Should fail since X=1, Y=2, but B=2 requires X=Y
	model := NewModel()
	x := model.NewVariable(NewBitSetDomainFromValues(5, []int{1}))
	y := model.NewVariable(NewBitSetDomainFromValues(5, []int{2}))
	b := model.NewVariable(NewBitSetDomainFromValues(2, []int{2})) // Force true

	arith, err := NewArithmetic(x, y, 0)
	if err != nil {
		t.Fatalf("NewArithmetic failed: %v", err)
	}

	reified, err := NewReifiedConstraint(arith, b)
	if err != nil {
		t.Fatalf("NewReifiedConstraint failed: %v", err)
	}

	model.AddConstraint(reified)

	solver := NewSolver(model)
	ctx := context.Background()

	solutions, err := solver.Solve(ctx, 100)
	if err != nil {
		t.Fatalf("Solve failed: %v", err)
	}

	// Should have no solutions
	if len(solutions) != 0 {
		t.Errorf("Expected 0 solutions (conflict), got %d: %v", len(solutions), solutions)
	}
}

// TestReifiedConstraint_EmptyDomain tests reification with empty domains.
func TestReifiedConstraint_EmptyDomain(t *testing.T) {
	model := NewModel()
	x := model.NewVariable(NewBitSetDomain(5))
	y := model.NewVariable(NewBitSetDomain(5))

	// Create boolean with empty domain
	bEmpty := NewBitSetDomainFromValues(2, []int{}) // Empty
	b := model.NewVariable(bEmpty)

	arith, err := NewArithmetic(x, y, 0)
	if err != nil {
		t.Fatalf("NewArithmetic failed: %v", err)
	}

	reified, err := NewReifiedConstraint(arith, b)
	if err != nil {
		t.Fatalf("NewReifiedConstraint failed: %v", err)
	}

	model.AddConstraint(reified)

	solver := NewSolver(model)
	ctx := context.Background()

	_, err = solver.Solve(ctx, 100)
	if err == nil {
		t.Fatalf("Solve should fail model validation for empty domain boolean")
	}
}

// TestReifiedConstraint_InvalidBooleanDomain tests creating reified constraint with invalid boolean domain.
func TestReifiedConstraint_InvalidBooleanDomain(t *testing.T) {
	model := NewModel()
	x := model.NewVariable(NewBitSetDomain(10))
	y := model.NewVariable(NewBitSetDomain(10))
	b := model.NewVariable(NewBitSetDomain(10)) // Domain {1..10}, not {1,2}

	arith, err := NewArithmetic(x, y, 0)
	if err != nil {
		t.Fatalf("NewArithmetic failed: %v", err)
	}

	// Creating the reified constraint should succeed (we don't validate at creation)
	_, err = NewReifiedConstraint(arith, b)
	if err != nil {
		t.Fatalf("NewReifiedConstraint should not fail at creation: %v", err)
	}

	// But propagation should fail with invalid boolean domain
	// We'll test this is handled gracefully during solving
}

// TestReifiedConstraint_NilChecks tests error handling for nil parameters.
func TestReifiedConstraint_NilChecks(t *testing.T) {
	model := NewModel()
	x := model.NewVariable(NewBitSetDomain(5))
	y := model.NewVariable(NewBitSetDomain(5))
	b := model.NewVariable(NewBitSetDomain(2))

	arith, err := NewArithmetic(x, y, 0)
	if err != nil {
		t.Fatalf("NewArithmetic failed: %v", err)
	}

	// Test nil constraint
	_, err = NewReifiedConstraint(nil, b)
	if err == nil {
		t.Error("Expected error for nil constraint, got nil")
	}

	// Test nil boolean variable
	_, err = NewReifiedConstraint(arith, nil)
	if err == nil {
		t.Error("Expected error for nil boolVar, got nil")
	}
}

// TestReifiedConstraint_Variables tests the Variables() method.
func TestReifiedConstraint_Variables(t *testing.T) {
	model := NewModel()
	x := model.NewVariable(NewBitSetDomain(5))
	y := model.NewVariable(NewBitSetDomain(5))
	b := model.NewVariable(NewBitSetDomain(2))

	arith, err := NewArithmetic(x, y, 0)
	if err != nil {
		t.Fatalf("NewArithmetic failed: %v", err)
	}

	reified, err := NewReifiedConstraint(arith, b)
	if err != nil {
		t.Fatalf("NewReifiedConstraint failed: %v", err)
	}

	vars := reified.Variables()

	// Should contain x, y, and b
	if len(vars) != 3 {
		t.Errorf("Expected 3 variables, got %d", len(vars))
	}

	// Check that all three are present
	hasX, hasY, hasB := false, false, false
	for _, v := range vars {
		if v.ID() == x.ID() {
			hasX = true
		}
		if v.ID() == y.ID() {
			hasY = true
		}
		if v.ID() == b.ID() {
			hasB = true
		}
	}

	if !hasX || !hasY || !hasB {
		t.Errorf("Variables missing: hasX=%v, hasY=%v, hasB=%v", hasX, hasY, hasB)
	}
}

// TestReifiedConstraint_TypeAndString tests Type() and String() methods.
func TestReifiedConstraint_TypeAndString(t *testing.T) {
	model := NewModel()
	x := model.NewVariable(NewBitSetDomain(5))
	y := model.NewVariable(NewBitSetDomain(5))
	b := model.NewVariable(NewBitSetDomain(2))

	arith, err := NewArithmetic(x, y, 0)
	if err != nil {
		t.Fatalf("NewArithmetic failed: %v", err)
	}

	reified, err := NewReifiedConstraint(arith, b)
	if err != nil {
		t.Fatalf("NewReifiedConstraint failed: %v", err)
	}

	// Check type
	if reified.Type() != "Reified(Arithmetic)" {
		t.Errorf("Expected type 'Reified(Arithmetic)', got '%s'", reified.Type())
	}

	// Check string contains expected parts
	str := reified.String()
	if len(str) == 0 {
		t.Error("String() returned empty string")
	}
}
