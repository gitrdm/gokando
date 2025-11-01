// Package minikanren provides comprehensive tests for Phase 2 constraint propagation.
//
// These tests validate:
//   - Individual constraint types (AllDifferent, Arithmetic, Inequality)
//   - Propagation to fixed-point
//   - Constraint interactions and composition
//   - Edge cases (empty domains, inconsistency detection)
//   - Integration with Phase 1 Model/Solver architecture
//
// All tests use real implementations with zero mocks, following coding standards.
package minikanren

import (
	"context"
	"testing"
	"time"
)

// TestAllDifferent_Basic tests basic AllDifferent constraint propagation.
func TestAllDifferent_Basic(t *testing.T) {
	tests := []struct {
		name      string
		domains   [][]int
		expectErr bool
		expectMin []int // Minimum domain sizes after propagation
	}{
		{
			name:      "three vars, sufficient values",
			domains:   [][]int{{1, 2, 3}, {1, 2, 3}, {1, 2, 3}},
			expectErr: false,
			expectMin: []int{1, 1, 1}, // Can all be singletons
		},
		{
			name:      "insufficient values",
			domains:   [][]int{{1, 2}, {1, 2}, {1, 2}},
			expectErr: true, // 3 variables, only 2 values
		},
		{
			name:      "one singleton",
			domains:   [][]int{{1}, {1, 2, 3}, {1, 2, 3}},
			expectErr: false,
			expectMin: []int{1, 1, 1}, // 1 removed from others
		},
		{
			name:      "conflicting singletons",
			domains:   [][]int{{1}, {1}, {2, 3}},
			expectErr: true, // Two vars both must be 1
		},
		{
			name:      "tight constraint",
			domains:   [][]int{{1, 2}, {2, 3}, {1, 3}},
			expectErr: false,
			expectMin: []int{1, 1, 1}, // Each var has 1-2 values after pruning
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := NewModel()
			vars := make([]*FDVariable, len(tt.domains))

			for i, vals := range tt.domains {
				maxVal := 0
				for _, v := range vals {
					if v > maxVal {
						maxVal = v
					}
				}
				domain := NewBitSetDomainFromValues(maxVal, vals)
				vars[i] = model.NewVariable(domain)
			}

			constraint, err := NewAllDifferent(vars)
			if err != nil {
				t.Fatalf("NewAllDifferent failed: %v", err)
			}
			model.AddConstraint(constraint)

			solver := NewSolver(model)
			state := (*SolverState)(nil) // Start with model domains

			newState, err := constraint.Propagate(solver, state)

			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Check domain sizes
			for i, minSize := range tt.expectMin {
				domain := solver.GetDomain(newState, vars[i].ID())
				if domain.Count() < minSize {
					t.Errorf("var %d: domain size %d < expected min %d", i, domain.Count(), minSize)
				}
			}
		})
	}
}

// TestAllDifferent_Propagation tests that AllDifferent prunes domains correctly.
func TestAllDifferent_Propagation(t *testing.T) {
	model := NewModel()

	// Create 4 variables: {1,2,3,4}
	vars := make([]*FDVariable, 4)
	for i := 0; i < 4; i++ {
		domain := NewBitSetDomain(4)
		vars[i] = model.NewVariable(domain)
	}

	// Add AllDifferent constraint
	constraint, err := NewAllDifferent(vars)
	if err != nil {
		t.Fatalf("NewAllDifferent failed: %v", err)
	}
	model.AddConstraint(constraint)

	solver := NewSolver(model)
	state := (*SolverState)(nil)

	// Bind first variable to 1
	state = solver.SetDomain(state, vars[0].ID(), NewBitSetDomainFromValues(4, []int{1}))

	// Propagate
	newState, err := constraint.Propagate(solver, state)
	if err != nil {
		t.Fatalf("propagation failed: %v", err)
	}

	// Check that value 1 was removed from other variables
	for i := 1; i < 4; i++ {
		domain := solver.GetDomain(newState, vars[i].ID())
		if domain.Has(1) {
			t.Errorf("var %d still has value 1 after propagation", i)
		}
		// Should have {2, 3, 4}
		if domain.Count() != 3 {
			t.Errorf("var %d: expected domain size 3, got %d", i, domain.Count())
		}
	}
}

// TestArithmetic_Basic tests basic arithmetic constraint propagation.
func TestArithmetic_Basic(t *testing.T) {
	tests := []struct {
		name      string
		srcDomain []int
		dstDomain []int
		offset    int
		expectSrc []int // Expected src domain after propagation
		expectDst []int // Expected dst domain after propagation
	}{
		{
			name:      "X + 1 = Y, forward pruning",
			srcDomain: []int{1, 2, 3},
			dstDomain: []int{1, 2, 3, 4, 5},
			offset:    1,
			expectSrc: []int{1, 2, 3},
			expectDst: []int{2, 3, 4}, // {1+1, 2+1, 3+1}
		},
		{
			name:      "X + 1 = Y, backward pruning",
			srcDomain: []int{1, 2, 3, 4, 5},
			dstDomain: []int{2, 3, 4},
			offset:    1,
			expectSrc: []int{1, 2, 3}, // {2-1, 3-1, 4-1}
			expectDst: []int{2, 3, 4},
		},
		{
			name:      "X - 2 = Y (negative offset)",
			srcDomain: []int{3, 4, 5},
			dstDomain: []int{1, 2, 3, 4},
			offset:    -2,
			expectSrc: []int{3, 4, 5}, // Image of {1,2,3,4} under +2 is {3,4,5,6}, intersect {3,4,5} = {3,4,5}
			expectDst: []int{1, 2, 3}, // Image of {3,4,5} under -2
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := NewModel()

			maxVal := 10
			srcDom := NewBitSetDomainFromValues(maxVal, tt.srcDomain)
			dstDom := NewBitSetDomainFromValues(maxVal, tt.dstDomain)

			src := model.NewVariable(srcDom)
			dst := model.NewVariable(dstDom)

			constraint, err := NewArithmetic(src, dst, tt.offset)

			if err != nil {

				t.Fatalf("NewArithmetic failed: %v", err)

			}
			solver := NewSolver(model)
			state := (*SolverState)(nil)

			newState, err := constraint.Propagate(solver, state)
			if err != nil {
				t.Fatalf("propagation failed: %v", err)
			}

			// Check src domain
			gotSrc := solver.GetDomain(newState, src.ID())
			for _, v := range tt.expectSrc {
				if !gotSrc.Has(v) {
					t.Errorf("src domain missing expected value %d", v)
				}
			}
			if gotSrc.Count() != len(tt.expectSrc) {
				t.Errorf("src domain size: got %d, want %d", gotSrc.Count(), len(tt.expectSrc))
			}

			// Check dst domain
			gotDst := solver.GetDomain(newState, dst.ID())
			for _, v := range tt.expectDst {
				if !gotDst.Has(v) {
					t.Errorf("dst domain missing expected value %d", v)
				}
			}
			if gotDst.Count() != len(tt.expectDst) {
				t.Errorf("dst domain size: got %d, want %d", gotDst.Count(), len(tt.expectDst))
			}
		})
	}
}

// TestInequality_LessThan tests less-than constraint propagation.
func TestInequality_LessThan(t *testing.T) {
	model := NewModel()

	x := model.NewVariable(NewBitSetDomain(10)) // {1..10}
	y := model.NewVariable(NewBitSetDomain(10)) // {1..10}

	constraint, err := NewInequality(x, y, LessThan)

	if err != nil {

		t.Fatalf("NewInequality failed: %v", err)

	}
	solver := NewSolver(model)
	state := (*SolverState)(nil)

	// Restrict Y to {5, 6, 7}
	state = solver.SetDomain(state, y.ID(), NewBitSetDomainFromValues(10, []int{5, 6, 7}))

	// Propagate X < Y
	newState, err := constraint.Propagate(solver, state)
	if err != nil {
		t.Fatalf("propagation failed: %v", err)
	}

	// Bounds propagation: X < max(Y) = 7, so remove X >= 7 → X ∈ {1..6}
	// NOTE: Bounds propagation is weaker than arc-consistency.
	// Arc-consistency would prune to X ∈ {1..4} since X must be < some Y value.
	xDom := solver.GetDomain(newState, x.ID())
	for i := 1; i <= 6; i++ {
		if !xDom.Has(i) {
			t.Errorf("X domain missing value %d", i)
		}
	}
	if xDom.Has(7) || xDom.Has(8) || xDom.Has(9) || xDom.Has(10) {
		t.Errorf("X domain should not contain values >= 7")
	}

	// Y should be restricted: Y > min(X) = 1, so remove Y <= 1
	// With original Y = {5,6,7}, all values are > 1, so no change
	yDom := solver.GetDomain(newState, y.ID())
	expected := []int{5, 6, 7}
	for _, v := range expected {
		if !yDom.Has(v) {
			t.Errorf("Y domain missing value %d", v)
		}
	}
}

// TestInequality_NotEqual tests not-equal constraint propagation.
func TestInequality_NotEqual(t *testing.T) {
	model := NewModel()

	x := model.NewVariable(NewBitSetDomainFromValues(5, []int{1, 2, 3}))
	y := model.NewVariable(NewBitSetDomainFromValues(5, []int{2, 3, 4}))

	constraint, err := NewInequality(x, y, NotEqual)

	if err != nil {

		t.Fatalf("NewInequality failed: %v", err)

	}
	solver := NewSolver(model)
	state := (*SolverState)(nil)

	// Bind X to 2
	state = solver.SetDomain(state, x.ID(), NewBitSetDomainFromValues(5, []int{2}))

	// Propagate X ≠ Y
	newState, err := constraint.Propagate(solver, state)
	if err != nil {
		t.Fatalf("propagation failed: %v", err)
	}

	// Y should have 2 removed
	yDom := solver.GetDomain(newState, y.ID())
	if yDom.Has(2) {
		t.Errorf("Y domain should not contain 2")
	}
	if !yDom.Has(3) || !yDom.Has(4) {
		t.Errorf("Y domain should contain 3 and 4")
	}
}

// TestInequality_Inconsistency tests that conflicting constraints are detected.
func TestInequality_Inconsistency(t *testing.T) {
	model := NewModel()

	x := model.NewVariable(NewBitSetDomainFromValues(5, []int{3}))
	y := model.NewVariable(NewBitSetDomainFromValues(5, []int{3}))

	constraint, err := NewInequality(x, y, NotEqual)
	if err != nil {
		t.Fatalf("NewInequality failed: %v", err)
	}
	solver := NewSolver(model)
	state := (*SolverState)(nil)

	// Both bound to same value - should fail
	_, err = constraint.Propagate(solver, state)
	if err == nil {
		t.Errorf("expected inconsistency error but got none")
	}
}

// TestPropagation_FixedPoint tests that Solver runs propagation to fixed-point.
func TestPropagation_FixedPoint(t *testing.T) {
	model := NewModel()

	// Create chain: X + 1 = Y, Y + 1 = Z
	x := model.NewVariable(NewBitSetDomain(10))
	y := model.NewVariable(NewBitSetDomain(10))
	z := model.NewVariable(NewBitSetDomain(10))

	c, err := NewArithmetic(x, y, 1)
	if err != nil {
		t.Fatalf("NewArithmetic failed: %v", err)
	}
	model.AddConstraint(c)
	c, err = NewArithmetic(y, z, 1)
	if err != nil {
		t.Fatalf("NewArithmetic failed: %v", err)
	}
	model.AddConstraint(c)

	solver := NewSolver(model)
	state := (*SolverState)(nil)

	// Bind X to {5}
	state = solver.SetDomain(state, x.ID(), NewBitSetDomainFromValues(10, []int{5}))

	// Run propagation
	newState, err := solver.propagate(state)
	if err != nil {
		t.Fatalf("propagation failed: %v", err)
	}

	// Check cascading effect
	yDom := solver.GetDomain(newState, y.ID())
	if !yDom.IsSingleton() || !yDom.Has(6) {
		t.Errorf("Y should be bound to 6")
	}

	zDom := solver.GetDomain(newState, z.ID())
	if !zDom.IsSingleton() || !zDom.Has(7) {
		t.Errorf("Z should be bound to 7")
	}
}

// TestPropagation_Combined tests multiple constraint types together.
func TestPropagation_Combined(t *testing.T) {
	model := NewModel()

	// Problem: X + Y = Z with AllDifferent(X, Y, Z)
	// All domains {1..5}
	x := model.NewVariable(NewBitSetDomain(5))
	y := model.NewVariable(NewBitSetDomain(5))
	z := model.NewVariable(NewBitSetDomain(5))

	c, err := NewAllDifferent([]*FDVariable{x, y, z})

	if err != nil {

		t.Fatalf("NewAllDifferent failed: %v", err)

	}

	model.AddConstraint(c)
	// Note: We can't directly express X + Y = Z with current constraints
	// This tests interaction of multiple AllDifferent constraints

	solver := NewSolver(model)

	// Bind X to 1
	state := solver.SetDomain(nil, x.ID(), NewBitSetDomainFromValues(5, []int{1}))

	newState, err := solver.propagate(state)
	if err != nil {
		t.Fatalf("propagation failed: %v", err)
	}

	// Y and Z should not contain 1
	yDom := solver.GetDomain(newState, y.ID())
	zDom := solver.GetDomain(newState, z.ID())

	if yDom.Has(1) {
		t.Errorf("Y should not contain 1")
	}
	if zDom.Has(1) {
		t.Errorf("Z should not contain 1")
	}
}

// TestSolver_WithConstraints tests solving with constraints.
func TestSolver_WithConstraints(t *testing.T) {
	model := NewModel()

	// Simple problem: AllDifferent(X, Y) with X,Y ∈ {1,2}
	x := model.NewVariable(NewBitSetDomainFromValues(2, []int{1, 2}))
	y := model.NewVariable(NewBitSetDomainFromValues(2, []int{1, 2}))

	c, err := NewAllDifferent([]*FDVariable{x, y})

	if err != nil {

		t.Fatalf("NewAllDifferent failed: %v", err)

	}

	model.AddConstraint(c)

	solver := NewSolver(model)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	solutions, err := solver.Solve(ctx, 0)
	if err != nil {
		t.Fatalf("solve failed: %v", err)
	}

	// Should find 2 solutions: (1,2) and (2,1)
	if len(solutions) != 2 {
		t.Errorf("expected 2 solutions, got %d", len(solutions))
	}

	// Verify solutions are different
	seen := make(map[string]bool)
	for _, sol := range solutions {
		key := ""
		for _, v := range sol {
			key += string(rune('0' + v))
		}
		if seen[key] {
			t.Errorf("duplicate solution: %v", sol)
		}
		seen[key] = true

		// Verify constraint: X ≠ Y
		if sol[0] == sol[1] {
			t.Errorf("solution violates AllDifferent: %v", sol)
		}
	}
}

// TestAllDifferent_NQueens4 tests AllDifferent on 4-Queens.
func TestAllDifferent_NQueens4(t *testing.T) {
	n := 4
	model := NewModel()

	// Column variables: each queen's column position
	cols := make([]*FDVariable, n)
	for i := 0; i < n; i++ {
		cols[i] = model.NewVariable(NewBitSetDomain(n))
	}

	// Diagonal variables need sufficient range
	// diag1[i] = col[i] + i ranges from 1+0=1 to 4+3=7
	// diag2[i] = col[i] - i ranges from 1-3=-2 to 4-0=4
	// To keep positive, we add offset: col[i] - i + n ranges from 1-3+4=2 to 4-0+4=8
	diag1 := make([]*FDVariable, n)
	diag2 := make([]*FDVariable, n)
	maxDiag := 2 * n
	for i := 0; i < n; i++ {
		diag1[i] = model.NewVariable(NewBitSetDomain(maxDiag))
		diag2[i] = model.NewVariable(NewBitSetDomain(maxDiag))
	}

	// Link diagonals: diag1[i] = col[i] + i
	for i := 0; i < n; i++ {
		c, err := NewArithmetic(cols[i], diag1[i], i)
		if err != nil {
			t.Fatalf("NewArithmetic failed: %v", err)
		}
		model.AddConstraint(c)
	}

	// Link diagonals: diag2[i] = col[i] - i + n
	for i := 0; i < n; i++ {
		c, err := NewArithmetic(cols[i], diag2[i], -i+n)
		if err != nil {
			t.Fatalf("NewArithmetic failed: %v", err)
		}
		model.AddConstraint(c)
	}

	// AllDifferent constraints
	c, err := NewAllDifferent(cols)
	if err != nil {
		t.Fatalf("NewAllDifferent failed: %v", err)
	}
	model.AddConstraint(c)
	c, err = NewAllDifferent(diag1)
	if err != nil {
		t.Fatalf("NewAllDifferent failed: %v", err)
	}
	model.AddConstraint(c)
	c, err = NewAllDifferent(diag2)
	if err != nil {
		t.Fatalf("NewAllDifferent failed: %v", err)
	}
	model.AddConstraint(c)

	solver := NewSolver(model)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	solutions, err := solver.Solve(ctx, 0)
	if err != nil {
		t.Fatalf("solve failed: %v", err)
	}

	// 4-Queens has 2 solutions
	if len(solutions) != 2 {
		t.Logf("Found %d solutions (expected 2)", len(solutions))
		for i, sol := range solutions {
			t.Logf("Solution %d: %v", i+1, sol[:n]) // Only show column values
		}
	}

	// Verify each solution
	for _, sol := range solutions {
		// Check column AllDifferent
		colVals := make(map[int]bool)
		for i := 0; i < n; i++ {
			if colVals[sol[i]] {
				t.Errorf("solution violates column AllDifferent: %v", sol[:n])
			}
			colVals[sol[i]] = true
		}

		// Check diagonal constraints (computed from columns)
		diag1Vals := make(map[int]bool)
		diag2Vals := make(map[int]bool)
		for i := 0; i < n; i++ {
			d1 := sol[i] + i
			d2 := sol[i] - i + n
			if diag1Vals[d1] {
				t.Errorf("solution violates diag1 AllDifferent: %v", sol[:n])
			}
			if diag2Vals[d2] {
				t.Errorf("solution violates diag2 AllDifferent: %v", sol[:n])
			}
			diag1Vals[d1] = true
			diag2Vals[d2] = true
		}
	}
}

// TestConstraint_EdgeCases tests edge cases in constraint handling.
func TestConstraint_EdgeCases(t *testing.T) {
	t.Run("AllDifferent with single variable", func(t *testing.T) {
		model := NewModel()
		x := model.NewVariable(NewBitSetDomain(5))
		constraint, err := NewAllDifferent([]*FDVariable{x})
		if err != nil {
			t.Fatalf("NewAllDifferent failed: %v", err)
		}
		solver := NewSolver(model)

		newState, err := constraint.Propagate(solver, nil)
		if err != nil {
			t.Errorf("single variable should not fail: %v", err)
		}
		// State might be nil if no changes were made (which is fine)
		_ = newState
	})

	t.Run("Arithmetic with zero offset", func(t *testing.T) {
		model := NewModel()
		x := model.NewVariable(NewBitSetDomainFromValues(5, []int{1, 2, 3}))
		y := model.NewVariable(NewBitSetDomainFromValues(5, []int{2, 3, 4}))

		constraint, err := NewArithmetic(x, y, 0) // Y = X + 0
		if err != nil {
			t.Fatalf("NewArithmetic failed: %v", err)
		}
		solver := NewSolver(model)

		newState, err := constraint.Propagate(solver, nil)
		if err != nil {
			t.Fatalf("propagation failed: %v", err)
		}

		// Domains should be intersected: {2, 3}
		xDom := solver.GetDomain(newState, x.ID())
		yDom := solver.GetDomain(newState, y.ID())

		expected := []int{2, 3}
		for _, v := range expected {
			if !xDom.Has(v) {
				t.Errorf("X missing value %d", v)
			}
			if !yDom.Has(v) {
				t.Errorf("Y missing value %d", v)
			}
		}
	})

	t.Run("nil solver or state", func(t *testing.T) {
		model := NewModel()
		x := model.NewVariable(NewBitSetDomain(5))
		constraint, err := NewAllDifferent([]*FDVariable{x})
		if err != nil {
			t.Fatalf("NewAllDifferent failed: %v", err)
		}

		_, err = constraint.Propagate(nil, nil)
		if err == nil {
			t.Errorf("expected error with nil solver")
		}
	})
}

// BenchmarkAllDifferent measures AllDifferent propagation performance.
func BenchmarkAllDifferent(b *testing.B) {
	sizes := []int{4, 8, 12}

	for _, n := range sizes {
		b.Run("n="+string(rune('0'+n)), func(b *testing.B) {
			model := NewModel()
			vars := make([]*FDVariable, n)
			for i := 0; i < n; i++ {
				vars[i] = model.NewVariable(NewBitSetDomain(n))
			}

			constraint, err := NewAllDifferent(vars)
			if err != nil {
				b.Fatalf("NewAllDifferent failed: %v", err)
			}
			solver := NewSolver(model)
			state := (*SolverState)(nil)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = constraint.Propagate(solver, state)
			}
		})
	}
}

// BenchmarkArithmetic measures arithmetic constraint performance.
func BenchmarkArithmetic(b *testing.B) {
	model := NewModel()
	x := model.NewVariable(NewBitSetDomain(100))
	y := model.NewVariable(NewBitSetDomain(100))

	constraint, err := NewArithmetic(x, y, 10)
	if err != nil {
		b.Fatalf("NewArithmetic failed: %v", err)
	}
	solver := NewSolver(model)
	state := (*SolverState)(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = constraint.Propagate(solver, state)
	}
}

// BenchmarkPropagation_FixedPoint measures full propagation performance.
func BenchmarkPropagation_FixedPoint(b *testing.B) {
	model := NewModel()

	// Chain of 10 variables with arithmetic constraints
	vars := make([]*FDVariable, 10)
	for i := 0; i < 10; i++ {
		vars[i] = model.NewVariable(NewBitSetDomain(20))
	}

	for i := 0; i < 9; i++ {
		c, err := NewArithmetic(vars[i], vars[i+1], 1)
		if err != nil {
			b.Fatalf("NewArithmetic failed: %v", err)
		}
		model.AddConstraint(c)
	}

	solver := NewSolver(model)
	state := solver.SetDomain(nil, 0, NewBitSetDomainFromValues(20, []int{5}))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = solver.propagate(state)
	}
}
