package minikanren

import (
	"context"
	"testing"
	"time"
)

// ============================================================================
// Mixed-sign LinearSum Benchmarks
// ============================================================================

// BenchmarkLinearSum_PositiveCoeffs benchmarks traditional all-positive LinearSum.
func BenchmarkLinearSum_PositiveCoeffs(b *testing.B) {
	for i := 0; i < b.N; i++ {
		m := NewModel()
		vars := make([]*FDVariable, 10)
		coeffs := make([]int, 10)
		for j := 0; j < 10; j++ {
			vars[j] = m.NewVariable(NewBitSetDomain(10))
			coeffs[j] = j + 1 // all positive
		}
		total := m.NewVariable(NewBitSetDomain(1000))
		ls, _ := NewLinearSum(vars, coeffs, total)
		m.AddConstraint(ls)

		solver := NewSolver(m)
		solver.propagate(nil)
	}
}

// BenchmarkLinearSum_MixedSignCoeffs benchmarks mixed-sign LinearSum.
func BenchmarkLinearSum_MixedSignCoeffs(b *testing.B) {
	for i := 0; i < b.N; i++ {
		m := NewModel()
		vars := make([]*FDVariable, 10)
		coeffs := make([]int, 10)
		for j := 0; j < 10; j++ {
			vars[j] = m.NewVariable(NewBitSetDomain(10))
			// Alternate positive and negative
			if j%2 == 0 {
				coeffs[j] = j + 1
			} else {
				coeffs[j] = -(j + 1)
			}
		}
		total := m.NewVariable(NewBitSetDomain(1000))
		ls, _ := NewLinearSum(vars, coeffs, total)
		m.AddConstraint(ls)

		solver := NewSolver(m)
		solver.propagate(nil)
	}
}

// ============================================================================
// Optimization Heuristic Benchmarks
// ============================================================================

// BenchmarkOptimize_DefaultHeuristic benchmarks optimization with default DomDeg heuristic.
func BenchmarkOptimize_DefaultHeuristic(b *testing.B) {
	for i := 0; i < b.N; i++ {
		m := NewModel()
		x := m.NewVariable(NewBitSetDomain(6))
		y := m.NewVariable(NewBitSetDomain(6))
		z := m.NewVariable(NewBitSetDomain(6))
		total := m.NewVariable(NewBitSetDomain(30))

		ls, _ := NewLinearSum([]*FDVariable{x, y, z}, []int{3, 2, 1}, total)
		m.AddConstraint(ls)

		solver := NewSolver(m)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		solver.SolveOptimal(ctx, total, true)
		cancel()
	}
}

// BenchmarkOptimize_ImpactHeuristic benchmarks optimization with HeuristicImpact.
func BenchmarkOptimize_ImpactHeuristic(b *testing.B) {
	for i := 0; i < b.N; i++ {
		m := NewModel()
		x := m.NewVariable(NewBitSetDomain(6))
		y := m.NewVariable(NewBitSetDomain(6))
		z := m.NewVariable(NewBitSetDomain(6))
		total := m.NewVariable(NewBitSetDomain(30))

		ls, _ := NewLinearSum([]*FDVariable{x, y, z}, []int{3, 2, 1}, total)
		m.AddConstraint(ls)

		solver := NewSolver(m)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		solver.SolveOptimalWithOptions(ctx, total, true,
			WithHeuristics(HeuristicImpact, ValueOrderObjImproving, 42))
		cancel()
	}
}

// ============================================================================
// BoolSum Objective Benchmarks
// ============================================================================

// BenchmarkOptimize_BoolSumObjective benchmarks optimization with BoolSum objective.
func BenchmarkOptimize_BoolSumObjective(b *testing.B) {
	for i := 0; i < b.N; i++ {
		m := NewModel()
		vars := make([]*FDVariable, 8)
		for j := 0; j < 8; j++ {
			vars[j] = m.NewVariable(NewBitSetDomainFromValues(2, []int{1, 2}))
		}
		count := m.NewVariable(NewBitSetDomain(9))

		bs, _ := NewBoolSum(vars, count)
		m.AddConstraint(bs)

		solver := NewSolver(m)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		solver.SolveOptimal(ctx, count, true) // minimize
		cancel()
	}
}

// ============================================================================
// Cumulative Energetic Reasoning Benchmarks
// ============================================================================

// BenchmarkCumulative_TimeTableOnly simulates cumulative with minimal energetic checks.
// (Note: our implementation always runs energetic reasoning, so this measures full cost)
func BenchmarkCumulative_TimeTableOnly(b *testing.B) {
	for i := 0; i < b.N; i++ {
		m := NewModel()
		starts := make([]*FDVariable, 5)
		for j := 0; j < 5; j++ {
			starts[j] = m.NewVariable(NewBitSetDomain(10))
		}

		cum, _ := NewCumulative(starts,
			[]int{2, 2, 2, 2, 2},
			[]int{1, 1, 1, 1, 1},
			3)
		m.AddConstraint(cum)

		solver := NewSolver(m)
		solver.propagate(nil)
	}
}

// BenchmarkCumulative_WithEnergeticReasoning benchmarks full energetic reasoning.
func BenchmarkCumulative_WithEnergeticReasoning(b *testing.B) {
	for i := 0; i < b.N; i++ {
		m := NewModel()
		starts := make([]*FDVariable, 8)
		for j := 0; j < 8; j++ {
			starts[j] = m.NewVariable(NewBitSetDomain(15))
		}

		cum, _ := NewCumulative(starts,
			[]int{3, 3, 3, 3, 3, 3, 3, 3},
			[]int{2, 2, 2, 2, 2, 2, 2, 2},
			5)
		m.AddConstraint(cum)

		solver := NewSolver(m)
		solver.propagate(nil)
	}
}
