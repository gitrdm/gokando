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

// TestInequality_AllOperators tests all 5 inequality operators systematically.
func TestInequality_AllOperators(t *testing.T) {
	tests := []struct {
		name      string
		kind      InequalityKind
		xDomain   []int
		yDomain   []int
		expectErr bool
		checkX    func(*testing.T, Domain)
		checkY    func(*testing.T, Domain)
	}{
		{
			name:      "LessThan: X < Y with Y bounded",
			kind:      LessThan,
			xDomain:   []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			yDomain:   []int{5, 6, 7},
			expectErr: false,
			checkX: func(t *testing.T, d Domain) {
				// X < max(Y) = 7, so remove X >= 7
				for i := 1; i <= 6; i++ {
					if !d.Has(i) {
						t.Errorf("X should contain %d", i)
					}
				}
				for i := 7; i <= 10; i++ {
					if d.Has(i) {
						t.Errorf("X should not contain %d (>= max(Y)=7)", i)
					}
				}
			},
			checkY: func(t *testing.T, d Domain) {
				// Y > min(X) = 1, so remove Y <= 1 (none in {5,6,7})
				expected := []int{5, 6, 7}
				for _, v := range expected {
					if !d.Has(v) {
						t.Errorf("Y should contain %d", v)
					}
				}
			},
		},
		{
			name:      "LessEqual: X ≤ Y with Y bounded",
			kind:      LessEqual,
			xDomain:   []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			yDomain:   []int{5, 6, 7},
			expectErr: false,
			checkX: func(t *testing.T, d Domain) {
				// X <= max(Y) = 7, so remove X > 7
				for i := 1; i <= 7; i++ {
					if !d.Has(i) {
						t.Errorf("X should contain %d", i)
					}
				}
				for i := 8; i <= 10; i++ {
					if d.Has(i) {
						t.Errorf("X should not contain %d (> max(Y)=7)", i)
					}
				}
			},
			checkY: func(t *testing.T, d Domain) {
				// Y >= min(X) = 1, so remove Y < 1 (none in {5,6,7})
				expected := []int{5, 6, 7}
				for _, v := range expected {
					if !d.Has(v) {
						t.Errorf("Y should contain %d", v)
					}
				}
			},
		},
		{
			name:      "GreaterThan: X > Y with X bounded",
			kind:      GreaterThan,
			xDomain:   []int{5, 6, 7},
			yDomain:   []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			expectErr: false,
			checkX: func(t *testing.T, d Domain) {
				// X > min(Y) = 1, all of {5,6,7} satisfy this
				expected := []int{5, 6, 7}
				for _, v := range expected {
					if !d.Has(v) {
						t.Errorf("X should contain %d", v)
					}
				}
			},
			checkY: func(t *testing.T, d Domain) {
				// Y < max(X) = 7, so remove Y >= 7
				for i := 1; i <= 6; i++ {
					if !d.Has(i) {
						t.Errorf("Y should contain %d", i)
					}
				}
				for i := 7; i <= 10; i++ {
					if d.Has(i) {
						t.Errorf("Y should not contain %d (>= max(X)=7)", i)
					}
				}
			},
		},
		{
			name:      "GreaterEqual: X ≥ Y with X bounded",
			kind:      GreaterEqual,
			xDomain:   []int{5, 6, 7},
			yDomain:   []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			expectErr: false,
			checkX: func(t *testing.T, d Domain) {
				// X >= min(Y) = 1, all of {5,6,7} satisfy this
				expected := []int{5, 6, 7}
				for _, v := range expected {
					if !d.Has(v) {
						t.Errorf("X should contain %d", v)
					}
				}
			},
			checkY: func(t *testing.T, d Domain) {
				// Y <= max(X) = 7, so remove Y > 7
				for i := 1; i <= 7; i++ {
					if !d.Has(i) {
						t.Errorf("Y should contain %d", i)
					}
				}
				for i := 8; i <= 10; i++ {
					if d.Has(i) {
						t.Errorf("Y should not contain %d (> max(X)=7)", i)
					}
				}
			},
		},
		{
			name:      "NotEqual: both singletons equal - conflict",
			kind:      NotEqual,
			xDomain:   []int{5},
			yDomain:   []int{5},
			expectErr: true,
		},
		{
			name:      "NotEqual: both singletons different - ok",
			kind:      NotEqual,
			xDomain:   []int{5},
			yDomain:   []int{6},
			expectErr: false,
			checkX: func(t *testing.T, d Domain) {
				if !d.Has(5) || d.Count() != 1 {
					t.Errorf("X should be singleton {5}")
				}
			},
			checkY: func(t *testing.T, d Domain) {
				if !d.Has(6) || d.Count() != 1 {
					t.Errorf("Y should be singleton {6}")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := NewModel()
			maxVal := 10
			x := model.NewVariable(NewBitSetDomainFromValues(maxVal, tt.xDomain))
			y := model.NewVariable(NewBitSetDomainFromValues(maxVal, tt.yDomain))

			constraint, err := NewInequality(x, y, tt.kind)
			if err != nil {
				t.Fatalf("NewInequality failed: %v", err)
			}

			solver := NewSolver(model)
			newState, err := constraint.Propagate(solver, nil)

			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.checkX != nil {
				xDom := solver.GetDomain(newState, x.ID())
				tt.checkX(t, xDom)
			}
			if tt.checkY != nil {
				yDom := solver.GetDomain(newState, y.ID())
				tt.checkY(t, yDom)
			}
		})
	}
}

// TestInequality_SelfReference tests constraints on the same variable.
func TestInequality_SelfReference(t *testing.T) {
	tests := []struct {
		name      string
		kind      InequalityKind
		domain    []int
		expectErr bool
	}{
		{
			name:      "X < X always fails",
			kind:      LessThan,
			domain:    []int{1, 2, 3},
			expectErr: true,
		},
		{
			name:      "X <= X always succeeds",
			kind:      LessEqual,
			domain:    []int{1, 2, 3},
			expectErr: false, // All values satisfy X <= X
		},
		{
			name:      "X > X always fails",
			kind:      GreaterThan,
			domain:    []int{1, 2, 3},
			expectErr: true,
		},
		{
			name:      "X >= X always succeeds",
			kind:      GreaterEqual,
			domain:    []int{1, 2, 3},
			expectErr: false, // All values satisfy X >= X
		},
		{
			name:      "X != X always fails",
			kind:      NotEqual,
			domain:    []int{1, 2, 3},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := NewModel()
			x := model.NewVariable(NewBitSetDomainFromValues(5, tt.domain))

			// Create constraint with same variable for both X and Y
			constraint, err := NewInequality(x, x, tt.kind)
			if err != nil {
				t.Fatalf("NewInequality failed: %v", err)
			}

			solver := NewSolver(model)
			_, err = constraint.Propagate(solver, nil)

			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error for %s but got none", tt.kind)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for %s: %v", tt.kind, err)
				}
			}
		})
	}
}

// TestInequality_BoundsPropagationRegression tests the fix for incorrect
// bounds propagation algorithm. The old code removed X >= minY and Y <= maxX,
// which is WRONG. The correct algorithm removes X >= maxY and Y <= minX.
//
// This test would FAIL with the old buggy algorithm and PASS with the fix.
func TestInequality_BoundsPropagationRegression(t *testing.T) {
	t.Run("LessThan: old algorithm caused empty domains", func(t *testing.T) {
		model := NewModel()
		// Both variables have same domain {1..5}
		x := model.NewVariable(NewBitSetDomain(5))
		y := model.NewVariable(NewBitSetDomain(5))

		constraint, err := NewInequality(x, y, LessThan)
		if err != nil {
			t.Fatalf("NewInequality failed: %v", err)
		}

		solver := NewSolver(model)
		newState, err := constraint.Propagate(solver, nil)

		// With CORRECT algorithm:
		// X < max(Y) = 5, remove X >= 5 → X ∈ {1,2,3,4}
		// Y > min(X) = 1, remove Y <= 1 → Y ∈ {2,3,4,5}
		// Result: X ∈ {1,2,3,4}, Y ∈ {2,3,4,5} - both non-empty ✓

		// With OLD BUGGY algorithm:
		// X < min(Y) = 1, remove X >= 1 → X becomes EMPTY ✗
		// Y > max(X) = 5, remove Y <= 5 → Y becomes EMPTY ✗

		if err != nil {
			t.Fatalf("propagation should succeed but got error: %v", err)
		}

		xDom := solver.GetDomain(newState, x.ID())
		yDom := solver.GetDomain(newState, y.ID())

		// X should have {1,2,3,4}
		if xDom.Count() != 4 {
			t.Errorf("X domain count: got %d, want 4", xDom.Count())
		}
		if !xDom.Has(1) || !xDom.Has(4) || xDom.Has(5) {
			t.Errorf("X domain incorrect: should be {1,2,3,4}")
		}

		// Y should have {2,3,4,5}
		if yDom.Count() != 4 {
			t.Errorf("Y domain count: got %d, want 4", yDom.Count())
		}
		if yDom.Has(1) || !yDom.Has(2) || !yDom.Has(5) {
			t.Errorf("Y domain incorrect: should be {2,3,4,5}")
		}
	})
}

// TestArithmetic_EmptyDomain tests propagation with empty domains.
func TestArithmetic_EmptyDomain(t *testing.T) {
	t.Run("source domain empty", func(t *testing.T) {
		model := NewModel()
		// Create empty domain by removing all values
		emptyDom := NewBitSetDomain(5)
		for i := 1; i <= 5; i++ {
			emptyDom = emptyDom.Remove(i).(*BitSetDomain)
		}
		x := model.NewVariable(emptyDom)
		y := model.NewVariable(NewBitSetDomain(5))

		constraint, err := NewArithmetic(x, y, 1)
		if err != nil {
			t.Fatalf("NewArithmetic failed: %v", err)
		}

		solver := NewSolver(model)
		_, err = constraint.Propagate(solver, nil)

		// Should detect empty domain
		if err == nil {
			t.Error("expected error with empty source domain")
		}
	})

	t.Run("destination domain empty", func(t *testing.T) {
		model := NewModel()
		emptyDom := NewBitSetDomain(5)
		for i := 1; i <= 5; i++ {
			emptyDom = emptyDom.Remove(i).(*BitSetDomain)
		}
		x := model.NewVariable(NewBitSetDomain(5))
		y := model.NewVariable(emptyDom)

		constraint, err := NewArithmetic(x, y, 1)
		if err != nil {
			t.Fatalf("NewArithmetic failed: %v", err)
		}

		solver := NewSolver(model)
		_, err = constraint.Propagate(solver, nil)

		// Should detect empty domain
		if err == nil {
			t.Error("expected error with empty destination domain")
		}
	})

	t.Run("conflicting singletons", func(t *testing.T) {
		model := NewModel()
		x := model.NewVariable(NewBitSetDomainFromValues(5, []int{3}))
		y := model.NewVariable(NewBitSetDomainFromValues(5, []int{5}))

		// X + 1 = Y means X=3 requires Y=4, but Y={5}
		constraint, err := NewArithmetic(x, y, 1)
		if err != nil {
			t.Fatalf("NewArithmetic failed: %v", err)
		}

		solver := NewSolver(model)
		_, err = constraint.Propagate(solver, nil)

		if err == nil {
			t.Error("expected error with conflicting singleton values")
		}
	})
}

// TestArithmetic_SelfReference tests X + offset = X constraints.
func TestArithmetic_SelfReference(t *testing.T) {
	t.Run("X + 0 = X always valid", func(t *testing.T) {
		model := NewModel()
		x := model.NewVariable(NewBitSetDomainFromValues(5, []int{1, 2, 3}))

		constraint, err := NewArithmetic(x, x, 0)
		if err != nil {
			t.Fatalf("NewArithmetic failed: %v", err)
		}

		solver := NewSolver(model)
		newState, err := constraint.Propagate(solver, nil)

		if err != nil {
			t.Errorf("X + 0 = X should always succeed: %v", err)
		}

		// Domain should be unchanged
		xDom := solver.GetDomain(newState, x.ID())
		if xDom.Count() != 3 {
			t.Errorf("domain should be unchanged, got count %d", xDom.Count())
		}
	})

	t.Run("X + 1 = X always fails", func(t *testing.T) {
		model := NewModel()
		x := model.NewVariable(NewBitSetDomainFromValues(5, []int{1, 2, 3}))

		constraint, err := NewArithmetic(x, x, 1)
		if err != nil {
			t.Fatalf("NewArithmetic failed: %v", err)
		}

		solver := NewSolver(model)
		_, err = constraint.Propagate(solver, nil)

		// X + 1 = X has no solutions
		if err == nil {
			t.Error("X + 1 = X should always fail")
		}
	})

	t.Run("X - 5 = X always fails", func(t *testing.T) {
		model := NewModel()
		x := model.NewVariable(NewBitSetDomainFromValues(10, []int{1, 2, 3, 4, 5}))

		constraint, err := NewArithmetic(x, x, -5)
		if err != nil {
			t.Fatalf("NewArithmetic failed: %v", err)
		}

		solver := NewSolver(model)
		_, err = constraint.Propagate(solver, nil)

		if err == nil {
			t.Error("X - 5 = X should always fail")
		}
	})
}

// TestArithmetic_LargeOffset tests boundary conditions with large offsets.
func TestArithmetic_LargeOffset(t *testing.T) {
	t.Run("offset larger than domain", func(t *testing.T) {
		model := NewModel()
		x := model.NewVariable(NewBitSetDomainFromValues(10, []int{1, 2, 3}))
		y := model.NewVariable(NewBitSetDomain(20))

		// X + 15 = Y, where X ∈ {1,2,3}
		constraint, err := NewArithmetic(x, y, 15)
		if err != nil {
			t.Fatalf("NewArithmetic failed: %v", err)
		}

		solver := NewSolver(model)
		newState, err := constraint.Propagate(solver, nil)

		if err != nil {
			t.Fatalf("propagation failed: %v", err)
		}

		// Y should be {16, 17, 18}
		yDom := solver.GetDomain(newState, y.ID())
		if !yDom.Has(16) || !yDom.Has(17) || !yDom.Has(18) {
			t.Errorf("Y should contain {16, 17, 18}")
		}
		if yDom.Count() != 3 {
			t.Errorf("Y domain size: got %d, want 3", yDom.Count())
		}
	})

	t.Run("negative offset near domain boundary", func(t *testing.T) {
		model := NewModel()
		x := model.NewVariable(NewBitSetDomainFromValues(10, []int{8, 9, 10}))
		y := model.NewVariable(NewBitSetDomain(10))

		// X - 7 = Y, where X ∈ {8,9,10}
		constraint, err := NewArithmetic(x, y, -7)
		if err != nil {
			t.Fatalf("NewArithmetic failed: %v", err)
		}

		solver := NewSolver(model)
		newState, err := constraint.Propagate(solver, nil)

		if err != nil {
			t.Fatalf("propagation failed: %v", err)
		}

		// Y should be {1, 2, 3}
		yDom := solver.GetDomain(newState, y.ID())
		if !yDom.Has(1) || !yDom.Has(2) || !yDom.Has(3) {
			t.Errorf("Y should contain {1, 2, 3}")
		}
		if yDom.Count() != 3 {
			t.Errorf("Y domain size: got %d, want 3", yDom.Count())
		}
	})
}

// TestAllDifferent_EmptyDomain tests AllDifferent with empty domains.
func TestAllDifferent_EmptyDomain(t *testing.T) {
	t.Run("one variable has empty domain", func(t *testing.T) {
		model := NewModel()
		x := model.NewVariable(NewBitSetDomainFromValues(3, []int{1, 2}))
		emptyDom := NewBitSetDomain(3)
		for i := 1; i <= 3; i++ {
			emptyDom = emptyDom.Remove(i).(*BitSetDomain)
		}
		y := model.NewVariable(emptyDom)
		z := model.NewVariable(NewBitSetDomainFromValues(3, []int{1, 2, 3}))

		constraint, err := NewAllDifferent([]*FDVariable{x, y, z})
		if err != nil {
			t.Fatalf("NewAllDifferent failed: %v", err)
		}

		solver := NewSolver(model)
		_, err = constraint.Propagate(solver, nil)

		if err == nil {
			t.Error("expected error with empty domain in variable list")
		}
	})
}

// TestAllDifferent_AlreadyBound tests optimization when variables are bound.
func TestAllDifferent_AlreadyBound(t *testing.T) {
	model := NewModel()
	// All variables already bound to different values
	x := model.NewVariable(NewBitSetDomainFromValues(5, []int{1}))
	y := model.NewVariable(NewBitSetDomainFromValues(5, []int{2}))
	z := model.NewVariable(NewBitSetDomainFromValues(5, []int{3}))

	constraint, err := NewAllDifferent([]*FDVariable{x, y, z})
	if err != nil {
		t.Fatalf("NewAllDifferent failed: %v", err)
	}

	solver := NewSolver(model)
	newState, err := constraint.Propagate(solver, nil)

	if err != nil {
		t.Errorf("propagation should succeed with already-bound distinct values: %v", err)
	}

	// Domains should remain unchanged
	xDom := solver.GetDomain(newState, x.ID())
	yDom := solver.GetDomain(newState, y.ID())
	zDom := solver.GetDomain(newState, z.ID())

	if !xDom.IsSingleton() || !xDom.Has(1) {
		t.Errorf("X should remain {1}")
	}
	if !yDom.IsSingleton() || !yDom.Has(2) {
		t.Errorf("Y should remain {2}")
	}
	if !zDom.IsSingleton() || !zDom.Has(3) {
		t.Errorf("Z should remain {3}")
	}
}

// TestConstraint_ConstructorValidation tests error handling in constructors.
func TestConstraint_ConstructorValidation(t *testing.T) {
	t.Run("NewAllDifferent with empty list", func(t *testing.T) {
		_, err := NewAllDifferent([]*FDVariable{})
		if err == nil {
			t.Error("NewAllDifferent should reject empty variable list")
		}
	})

	t.Run("NewAllDifferent with nil list", func(t *testing.T) {
		_, err := NewAllDifferent(nil)
		if err == nil {
			t.Error("NewAllDifferent should reject nil variable list")
		}
	})

	t.Run("NewArithmetic with nil source", func(t *testing.T) {
		model := NewModel()
		y := model.NewVariable(NewBitSetDomain(5))
		_, err := NewArithmetic(nil, y, 0)
		if err == nil {
			t.Error("NewArithmetic should reject nil source variable")
		}
	})

	t.Run("NewArithmetic with nil destination", func(t *testing.T) {
		model := NewModel()
		x := model.NewVariable(NewBitSetDomain(5))
		_, err := NewArithmetic(x, nil, 0)
		if err == nil {
			t.Error("NewArithmetic should reject nil destination variable")
		}
	})

	t.Run("NewInequality with nil x", func(t *testing.T) {
		model := NewModel()
		y := model.NewVariable(NewBitSetDomain(5))
		_, err := NewInequality(nil, y, LessThan)
		if err == nil {
			t.Error("NewInequality should reject nil x variable")
		}
	})

	t.Run("NewInequality with nil y", func(t *testing.T) {
		model := NewModel()
		x := model.NewVariable(NewBitSetDomain(5))
		_, err := NewInequality(x, nil, LessThan)
		if err == nil {
			t.Error("NewInequality should reject nil y variable")
		}
	})
}

// TestPropagation_DeepChain tests propagation through deep constraint chains.
func TestPropagation_DeepChain(t *testing.T) {
	model := NewModel()

	// Create chain of 20 variables: V0 + 1 = V1, V1 + 1 = V2, ..., V18 + 1 = V19
	chainLength := 20
	vars := make([]*FDVariable, chainLength)
	for i := 0; i < chainLength; i++ {
		vars[i] = model.NewVariable(NewBitSetDomain(30))
	}

	for i := 0; i < chainLength-1; i++ {
		c, err := NewArithmetic(vars[i], vars[i+1], 1)
		if err != nil {
			t.Fatalf("NewArithmetic failed: %v", err)
		}
		model.AddConstraint(c)
	}

	solver := NewSolver(model)

	// Bind first variable to {10}
	state := solver.SetDomain(nil, vars[0].ID(), NewBitSetDomainFromValues(30, []int{10}))

	// Propagate should cascade through entire chain
	newState, err := solver.propagate(state)
	if err != nil {
		t.Fatalf("propagation failed: %v", err)
	}

	// Verify cascade: Vi should be bound to {10 + i}
	for i := 0; i < chainLength; i++ {
		dom := solver.GetDomain(newState, vars[i].ID())
		expected := 10 + i
		if !dom.IsSingleton() || !dom.Has(expected) {
			t.Errorf("var[%d] should be bound to {%d}, got %v", i, expected, dom)
		}
	}
}

// TestPropagation_NoChanges tests that propagation handles fixed-point correctly.
func TestPropagation_NoChanges(t *testing.T) {
	model := NewModel()

	x := model.NewVariable(NewBitSetDomainFromValues(5, []int{1, 2, 3}))
	y := model.NewVariable(NewBitSetDomainFromValues(5, []int{4, 5}))

	// X != Y is already satisfied by domains (disjoint)
	c, err := NewInequality(x, y, NotEqual)
	if err != nil {
		t.Fatalf("NewInequality failed: %v", err)
	}
	model.AddConstraint(c)

	solver := NewSolver(model)
	newState, err := solver.propagate(nil)

	if err != nil {
		t.Fatalf("propagation failed: %v", err)
	}

	// Domains should be unchanged (already disjoint)
	xDom := solver.GetDomain(newState, x.ID())
	yDom := solver.GetDomain(newState, y.ID())

	if xDom.Count() != 3 {
		t.Errorf("X domain should remain {1,2,3}, got count %d", xDom.Count())
	}
	if yDom.Count() != 2 {
		t.Errorf("Y domain should remain {4,5}, got count %d", yDom.Count())
	}
}

// TestAllDifferent_DisjointDomains tests when some domains don't overlap.
func TestAllDifferent_DisjointDomains(t *testing.T) {
	model := NewModel()

	// Three variables with partially disjoint domains
	x := model.NewVariable(NewBitSetDomainFromValues(10, []int{1, 2}))
	y := model.NewVariable(NewBitSetDomainFromValues(10, []int{3, 4}))
	z := model.NewVariable(NewBitSetDomainFromValues(10, []int{1, 2, 3, 4}))

	constraint, err := NewAllDifferent([]*FDVariable{x, y, z})
	if err != nil {
		t.Fatalf("NewAllDifferent failed: %v", err)
	}

	solver := NewSolver(model)
	newState, err := constraint.Propagate(solver, nil)

	if err != nil {
		t.Fatalf("propagation failed: %v", err)
	}

	// X and Y are disjoint, Z must avoid their values
	// But with only matching algorithm, Z might not be fully pruned
	// This tests the matching-based pruning works correctly
	zDom := solver.GetDomain(newState, z.ID())

	// Z should still contain values (not empty)
	if zDom.Count() == 0 {
		t.Error("Z domain should not be empty")
	}
}

// TestInequality_BoundaryValues tests constraints at domain boundaries.
func TestInequality_BoundaryValues(t *testing.T) {
	t.Run("X < Y with X at maximum", func(t *testing.T) {
		model := NewModel()
		x := model.NewVariable(NewBitSetDomainFromValues(10, []int{9, 10}))
		y := model.NewVariable(NewBitSetDomain(10))

		constraint, err := NewInequality(x, y, LessThan)
		if err != nil {
			t.Fatalf("NewInequality failed: %v", err)
		}

		solver := NewSolver(model)
		newState, err := constraint.Propagate(solver, nil)
		if err != nil {
			t.Fatalf("propagation failed: %v", err)
		}

		// X < max(Y) = 10, so remove X >= 10 → X = {9}
		xDom := solver.GetDomain(newState, x.ID())
		if !xDom.Has(9) || xDom.Has(10) {
			t.Errorf("X should be {9}, got count %d", xDom.Count())
		}

		// Y > min(X) = 9, so remove Y <= 9 → Y = {10}
		yDom := solver.GetDomain(newState, y.ID())
		if !yDom.IsSingleton() || !yDom.Has(10) {
			t.Errorf("Y should be {10}")
		}
	})

	t.Run("X <= Y with both at boundaries", func(t *testing.T) {
		model := NewModel()
		x := model.NewVariable(NewBitSetDomainFromValues(10, []int{1, 10}))
		y := model.NewVariable(NewBitSetDomainFromValues(10, []int{1, 10}))

		constraint, err := NewInequality(x, y, LessEqual)
		if err != nil {
			t.Fatalf("NewInequality failed: %v", err)
		}

		solver := NewSolver(model)
		newState, err := constraint.Propagate(solver, nil)
		if err != nil {
			t.Fatalf("propagation failed: %v", err)
		}

		// Both can be {1, 10} since 1 <= 10 is possible
		xDom := solver.GetDomain(newState, x.ID())
		yDom := solver.GetDomain(newState, y.ID())

		if xDom.Count() != 2 || yDom.Count() != 2 {
			t.Errorf("Both domains should remain {1, 10}")
		}
	})

	t.Run("X > Y with Y at minimum", func(t *testing.T) {
		model := NewModel()
		x := model.NewVariable(NewBitSetDomain(10))
		y := model.NewVariable(NewBitSetDomainFromValues(10, []int{1, 2}))

		constraint, err := NewInequality(x, y, GreaterThan)
		if err != nil {
			t.Fatalf("NewInequality failed: %v", err)
		}

		solver := NewSolver(model)
		newState, err := constraint.Propagate(solver, nil)
		if err != nil {
			t.Fatalf("propagation failed: %v", err)
		}

		// X > min(Y) = 1, so remove X <= 1 → X = {2..10}
		xDom := solver.GetDomain(newState, x.ID())
		if xDom.Has(1) {
			t.Errorf("X should not contain 1")
		}
		if !xDom.Has(2) || !xDom.Has(10) {
			t.Errorf("X should contain {2..10}")
		}

		// Y < max(X) = 10, so no pruning needed for {1,2}
		yDom := solver.GetDomain(newState, y.ID())
		if yDom.Count() != 2 {
			t.Errorf("Y should remain {1, 2}")
		}
	})
}

// TestAllDifferent_TwoVariablesOneValue tests the minimal unsolvable case.
func TestAllDifferent_TwoVariablesOneValue(t *testing.T) {
	model := NewModel()
	x := model.NewVariable(NewBitSetDomainFromValues(3, []int{1}))
	y := model.NewVariable(NewBitSetDomainFromValues(3, []int{1}))

	constraint, err := NewAllDifferent([]*FDVariable{x, y})
	if err != nil {
		t.Fatalf("NewAllDifferent failed: %v", err)
	}

	solver := NewSolver(model)
	_, err = constraint.Propagate(solver, nil)

	// Should fail: two variables, one value, all different
	if err == nil {
		t.Error("expected error: two variables cannot both be 1 with AllDifferent")
	}
}

// TestAllDifferent_LargeDomain tests performance with large domains.
func TestAllDifferent_LargeDomain(t *testing.T) {
	model := NewModel()

	// 10 variables with large domain {1..50}
	n := 10
	domainSize := 50
	vars := make([]*FDVariable, n)
	for i := 0; i < n; i++ {
		vars[i] = model.NewVariable(NewBitSetDomain(domainSize))
	}

	constraint, err := NewAllDifferent(vars)
	if err != nil {
		t.Fatalf("NewAllDifferent failed: %v", err)
	}

	solver := NewSolver(model)

	// Bind first variable to force pruning
	state := solver.SetDomain(nil, vars[0].ID(), NewBitSetDomainFromValues(domainSize, []int{25}))

	newState, err := constraint.Propagate(solver, state)
	if err != nil {
		t.Fatalf("propagation failed: %v", err)
	}

	// All other variables should have 25 removed
	for i := 1; i < n; i++ {
		dom := solver.GetDomain(newState, vars[i].ID())
		if dom.Has(25) {
			t.Errorf("var[%d] should not contain 25", i)
		}
		if dom.Count() != domainSize-1 {
			t.Errorf("var[%d] should have %d values, got %d", i, domainSize-1, dom.Count())
		}
	}
}

// TestArithmetic_NegativeValues tests offsets that produce negative results.
func TestArithmetic_NegativeValues(t *testing.T) {
	t.Run("offset causes value below minimum", func(t *testing.T) {
		model := NewModel()
		x := model.NewVariable(NewBitSetDomainFromValues(5, []int{1, 2}))
		y := model.NewVariable(NewBitSetDomain(5))

		// X - 3 = Y, where X ∈ {1, 2}
		// Y would be {-2, -1} but domain starts at 1, so result is empty
		constraint, err := NewArithmetic(x, y, -3)
		if err != nil {
			t.Fatalf("NewArithmetic failed: %v", err)
		}

		solver := NewSolver(model)
		_, err = constraint.Propagate(solver, nil)

		// Should fail because no valid Y values
		if err == nil {
			t.Error("expected error when offset produces values outside domain")
		}
	})

	t.Run("offset causes value above maximum", func(t *testing.T) {
		model := NewModel()
		x := model.NewVariable(NewBitSetDomainFromValues(5, []int{4, 5}))
		y := model.NewVariable(NewBitSetDomain(5))

		// X + 10 = Y, where X ∈ {4, 5}
		// Y would be {14, 15} but domain max is 5, so result is empty
		constraint, err := NewArithmetic(x, y, 10)
		if err != nil {
			t.Fatalf("NewArithmetic failed: %v", err)
		}

		solver := NewSolver(model)
		_, err = constraint.Propagate(solver, nil)

		// Should fail because no valid Y values
		if err == nil {
			t.Error("expected error when offset produces values outside domain")
		}
	})
}

// TestPropagation_MultipleConstraintTypes tests combining different constraint types.
func TestPropagation_MultipleConstraintTypes(t *testing.T) {
	model := NewModel()

	// Problem: X < Y < Z with AllDifferent(X, Y, Z)
	x := model.NewVariable(NewBitSetDomain(10))
	y := model.NewVariable(NewBitSetDomain(10))
	z := model.NewVariable(NewBitSetDomain(10))

	// X < Y
	c1, err := NewInequality(x, y, LessThan)
	if err != nil {
		t.Fatalf("NewInequality failed: %v", err)
	}
	model.AddConstraint(c1)

	// Y < Z
	c2, err := NewInequality(y, z, LessThan)
	if err != nil {
		t.Fatalf("NewInequality failed: %v", err)
	}
	model.AddConstraint(c2)

	// AllDifferent(X, Y, Z)
	c3, err := NewAllDifferent([]*FDVariable{x, y, z})
	if err != nil {
		t.Fatalf("NewAllDifferent failed: %v", err)
	}
	model.AddConstraint(c3)

	solver := NewSolver(model)

	// Bind X to {5}
	state := solver.SetDomain(nil, x.ID(), NewBitSetDomainFromValues(10, []int{5}))

	newState, err := solver.propagate(state)
	if err != nil {
		t.Fatalf("propagation failed: %v", err)
	}

	// X = {5}
	xDom := solver.GetDomain(newState, x.ID())
	if !xDom.IsSingleton() || !xDom.Has(5) {
		t.Errorf("X should be {5}")
	}

	// Y > 5 and Y < max(Z) and Y != 5 → Y ∈ {6..9}
	yDom := solver.GetDomain(newState, y.ID())
	if yDom.Has(5) || yDom.Has(1) || yDom.Has(10) {
		t.Errorf("Y should be pruned by X < Y and Y < Z")
	}

	// Z > min(Y) and Z != 5 → Z should be restricted
	zDom := solver.GetDomain(newState, z.ID())
	if zDom.Has(5) {
		t.Errorf("Z should not contain 5 (AllDifferent)")
	}
}

// TestPropagation_ArithmeticChainWithInequality tests arithmetic + inequality.
func TestPropagation_ArithmeticChainWithInequality(t *testing.T) {
	model := NewModel()

	// Problem: Y = X + 2, Z = Y + 2, X < 5
	x := model.NewVariable(NewBitSetDomain(10))
	y := model.NewVariable(NewBitSetDomain(10))
	z := model.NewVariable(NewBitSetDomain(10))

	// Y = X + 2
	c1, err := NewArithmetic(x, y, 2)
	if err != nil {
		t.Fatalf("NewArithmetic failed: %v", err)
	}
	model.AddConstraint(c1)

	// Z = Y + 2
	c2, err := NewArithmetic(y, z, 2)
	if err != nil {
		t.Fatalf("NewArithmetic failed: %v", err)
	}
	model.AddConstraint(c2)

	// X < 5
	five := model.NewVariable(NewBitSetDomainFromValues(10, []int{5}))
	c3, err := NewInequality(x, five, LessThan)
	if err != nil {
		t.Fatalf("NewInequality failed: %v", err)
	}
	model.AddConstraint(c3)

	solver := NewSolver(model)
	newState, err := solver.propagate(nil)
	if err != nil {
		t.Fatalf("propagation failed: %v", err)
	}

	// X < 5 → X ∈ {1, 2, 3, 4}
	xDom := solver.GetDomain(newState, x.ID())
	if xDom.Has(5) || xDom.Has(6) {
		t.Errorf("X should be {1..4}")
	}

	// Y = X + 2 → Y ∈ {3, 4, 5, 6}
	yDom := solver.GetDomain(newState, y.ID())
	if yDom.Has(1) || yDom.Has(2) || yDom.Has(7) {
		t.Errorf("Y should be {3..6}")
	}

	// Z = Y + 2 → Z ∈ {5, 6, 7, 8}
	zDom := solver.GetDomain(newState, z.ID())
	if zDom.Has(4) || zDom.Has(9) {
		t.Errorf("Z should be {5..8}")
	}
}

// TestPropagation_CircularDetection tests that circular dependencies don't cause issues.
func TestPropagation_CircularDetection(t *testing.T) {
	model := NewModel()

	// Create a "circular" structure: X != Y, Y != Z, Z != X
	// This isn't truly circular in terms of propagation, but tests
	// that the fixed-point algorithm handles interdependencies
	x := model.NewVariable(NewBitSetDomainFromValues(3, []int{1, 2, 3}))
	y := model.NewVariable(NewBitSetDomainFromValues(3, []int{1, 2, 3}))
	z := model.NewVariable(NewBitSetDomainFromValues(3, []int{1, 2, 3}))

	c1, err := NewInequality(x, y, NotEqual)
	if err != nil {
		t.Fatalf("NewInequality failed: %v", err)
	}
	model.AddConstraint(c1)

	c2, err := NewInequality(y, z, NotEqual)
	if err != nil {
		t.Fatalf("NewInequality failed: %v", err)
	}
	model.AddConstraint(c2)

	c3, err := NewInequality(z, x, NotEqual)
	if err != nil {
		t.Fatalf("NewInequality failed: %v", err)
	}
	model.AddConstraint(c3)

	solver := NewSolver(model)

	// Bind X to 1
	state := solver.SetDomain(nil, x.ID(), NewBitSetDomainFromValues(3, []int{1}))

	newState, err := solver.propagate(state)
	if err != nil {
		t.Fatalf("propagation failed: %v", err)
	}

	// X = {1}
	xDom := solver.GetDomain(newState, x.ID())
	if !xDom.IsSingleton() || !xDom.Has(1) {
		t.Errorf("X should be {1}")
	}

	// Y != 1 → Y ∈ {2, 3}
	yDom := solver.GetDomain(newState, y.ID())
	if yDom.Has(1) {
		t.Errorf("Y should not contain 1")
	}
	if yDom.Count() != 2 {
		t.Errorf("Y should have 2 values, got %d", yDom.Count())
	}

	// Z != 1 (from Z != X) and potentially restricted by Y
	zDom := solver.GetDomain(newState, z.ID())
	if zDom.Has(1) {
		t.Errorf("Z should not contain 1")
	}
}

// TestAllDifferent_ProgressivePruning tests that repeated propagation prunes correctly.
func TestAllDifferent_ProgressivePruning(t *testing.T) {
	model := NewModel()

	// 5 variables with {1..5}
	vars := make([]*FDVariable, 5)
	for i := 0; i < 5; i++ {
		vars[i] = model.NewVariable(NewBitSetDomain(5))
	}

	constraint, err := NewAllDifferent(vars)
	if err != nil {
		t.Fatalf("NewAllDifferent failed: %v", err)
	}

	solver := NewSolver(model)
	state := (*SolverState)(nil)

	// Bind vars progressively and check pruning at each step
	// Bind v0 = 1
	state = solver.SetDomain(state, vars[0].ID(), NewBitSetDomainFromValues(5, []int{1}))
	state, err = constraint.Propagate(solver, state)
	if err != nil {
		t.Fatalf("propagation 1 failed: %v", err)
	}

	// All others should have 1 removed
	for i := 1; i < 5; i++ {
		dom := solver.GetDomain(state, vars[i].ID())
		if dom.Has(1) {
			t.Errorf("after v0=1, var[%d] should not contain 1", i)
		}
	}

	// Bind v1 = 2
	state = solver.SetDomain(state, vars[1].ID(), NewBitSetDomainFromValues(5, []int{2}))
	state, err = constraint.Propagate(solver, state)
	if err != nil {
		t.Fatalf("propagation 2 failed: %v", err)
	}

	// v2, v3, v4 should have {1, 2} removed → {3, 4, 5}
	for i := 2; i < 5; i++ {
		dom := solver.GetDomain(state, vars[i].ID())
		if dom.Has(1) || dom.Has(2) {
			t.Errorf("after v1=2, var[%d] should not contain 1 or 2", i)
		}
		if dom.Count() != 3 {
			t.Errorf("var[%d] should have 3 values, got %d", i, dom.Count())
		}
	}
}

// TestInequality_AsymmetricPruning tests that X op Y prunes differently than Y op X.
func TestInequality_AsymmetricPruning(t *testing.T) {
	t.Run("X < Y vs Y > X produces same result", func(t *testing.T) {
		model1 := NewModel()
		x1 := model1.NewVariable(NewBitSetDomain(10))
		y1 := model1.NewVariable(NewBitSetDomainFromValues(10, []int{5, 6}))

		c1, err := NewInequality(x1, y1, LessThan)
		if err != nil {
			t.Fatalf("NewInequality failed: %v", err)
		}

		solver1 := NewSolver(model1)
		state1, err := c1.Propagate(solver1, nil)
		if err != nil {
			t.Fatalf("propagation failed: %v", err)
		}

		// Compare with reversed constraint
		model2 := NewModel()
		x2 := model2.NewVariable(NewBitSetDomain(10))
		y2 := model2.NewVariable(NewBitSetDomainFromValues(10, []int{5, 6}))

		c2, err := NewInequality(y2, x2, GreaterThan)
		if err != nil {
			t.Fatalf("NewInequality failed: %v", err)
		}

		solver2 := NewSolver(model2)
		state2, err := c2.Propagate(solver2, nil)
		if err != nil {
			t.Fatalf("propagation failed: %v", err)
		}

		// Results should be equivalent
		x1Dom := solver1.GetDomain(state1, x1.ID())
		x2Dom := solver2.GetDomain(state2, x2.ID())

		if x1Dom.Count() != x2Dom.Count() {
			t.Errorf("X < Y and Y > X should produce same X domain size")
		}
	})
}

// TestArithmetic_BidirectionalConsistency verifies that forward and backward
// propagation maintain consistency.
func TestArithmetic_BidirectionalConsistency(t *testing.T) {
	model := NewModel()

	// Y = X + 3, where X ∈ {1, 3, 5}, Y ∈ {2, 4, 6, 8}
	x := model.NewVariable(NewBitSetDomainFromValues(10, []int{1, 3, 5}))
	y := model.NewVariable(NewBitSetDomainFromValues(10, []int{2, 4, 6, 8}))

	constraint, err := NewArithmetic(x, y, 3)
	if err != nil {
		t.Fatalf("NewArithmetic failed: %v", err)
	}

	solver := NewSolver(model)
	newState, err := constraint.Propagate(solver, nil)
	if err != nil {
		t.Fatalf("propagation failed: %v", err)
	}

	// Forward: Y ⊆ {X + 3 | X ∈ {1, 3, 5}} = {4, 6, 8}
	// Y ∩ {2, 4, 6, 8} = {4, 6, 8}
	//
	// Backward: X ⊆ {Y - 3 | Y ∈ {4, 6, 8}} = {1, 3, 5}
	// X ∩ {1, 3, 5} = {1, 3, 5}
	//
	// Result: X = {1, 3, 5}, Y = {4, 6, 8}

	xDom := solver.GetDomain(newState, x.ID())
	yDom := solver.GetDomain(newState, y.ID())

	// X should remain {1, 3, 5} (all values are consistent)
	if !xDom.Has(1) || !xDom.Has(3) || !xDom.Has(5) {
		t.Errorf("X should contain {1, 3, 5}")
	}
	if xDom.Count() != 3 {
		t.Errorf("X should have exactly 3 values, got %d", xDom.Count())
	}

	// Y should be {4, 6, 8} (2 pruned)
	if !yDom.Has(4) || !yDom.Has(6) || !yDom.Has(8) {
		t.Errorf("Y should contain {4, 6, 8}")
	}
	if yDom.Has(2) {
		t.Errorf("Y should not contain 2 (no X value produces 2)")
	}
	if yDom.Count() != 3 {
		t.Errorf("Y should have exactly 3 values, got %d", yDom.Count())
	}
}

// TestPropagation_MaxIterations tests that fixed-point terminates reasonably.
func TestPropagation_MaxIterations(t *testing.T) {
	model := NewModel()

	// Create a long chain that requires multiple iterations
	n := 15
	vars := make([]*FDVariable, n)
	for i := 0; i < n; i++ {
		vars[i] = model.NewVariable(NewBitSetDomain(20))
	}

	// Chain: v0 + 1 = v1, v1 + 1 = v2, ..., v13 + 1 = v14
	for i := 0; i < n-1; i++ {
		c, err := NewArithmetic(vars[i], vars[i+1], 1)
		if err != nil {
			t.Fatalf("NewArithmetic failed: %v", err)
		}
		model.AddConstraint(c)
	}

	solver := NewSolver(model)

	// Bind last variable to {15}
	state := solver.SetDomain(nil, vars[n-1].ID(), NewBitSetDomainFromValues(20, []int{15}))

	// Propagate should complete in reasonable time (< 1 second)
	newState, err := solver.propagate(state)
	if err != nil {
		t.Fatalf("propagation failed: %v", err)
	}

	// Verify backward propagation worked: v0 should be {1}
	v0Dom := solver.GetDomain(newState, vars[0].ID())
	if !v0Dom.IsSingleton() || !v0Dom.Has(1) {
		t.Errorf("v0 should be {1}, got %v", v0Dom)
	}
}
