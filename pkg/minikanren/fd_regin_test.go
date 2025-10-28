package minikanren

import (
	"context"
	"testing"
)

// Test that Regin detects impossible matchings when two vars are singleton to same value
func TestReginDetectsImpossibleMatching(t *testing.T) {
	s := NewFDStoreWithDomain(3)
	a := s.NewVar()
	b := s.NewVar()
	// force both to 1
	if !s.Assign(a, 1) {
		t.Fatal("failed to assign a")
	}
	if !s.Assign(b, 1) {
		t.Fatal("failed to assign b")
	}
	ok := s.ReginFilterLocked([]*FDVar{a, b})
	if ok {
		t.Fatalf("expected Regin to detect impossible matching but it returned ok")
	}
}

// Test that Regin preserves supported values and propagates singletons
func TestReginSingletonPropagation(t *testing.T) {
	s := NewFDStoreWithDomain(3)
	a := s.NewVar()
	b := s.NewVar()
	c := s.NewVar()
	// assign c = 1; others still have full domains
	if !s.Assign(c, 1) {
		t.Fatal("failed to assign c")
	}
	// apply AllDifferentRegin on all three
	ok := s.ReginFilterLocked([]*FDVar{a, b, c})
	if !ok {
		t.Fatal("Regin unexpectedly failed on simple singleton propagation")
	}
	// c must be singleton 1, a and b must not have 1 in their domains
	if c.domain.SingletonValue() != 1 {
		t.Fatalf("expected c==1, got %d", c.domain.SingletonValue())
	}
	if a.domain.Has(1) || b.domain.Has(1) {
		t.Fatalf("expected peers to have value 1 removed")
	}
}

// Test that the Sudoku example (partial) solves with Regin-enabled AllDifferent
func TestReginSudokuPartialSolve(t *testing.T) {
	// reuse the same puzzle as examples/sudoku
	puzzle := [81]int{
		5, 3, 0, 0, 7, 0, 0, 0, 0,
		6, 0, 0, 1, 9, 5, 0, 0, 0,
		0, 9, 8, 0, 0, 0, 0, 6, 0,

		8, 0, 0, 0, 6, 0, 0, 0, 3,
		4, 0, 0, 8, 0, 3, 0, 0, 1,
		7, 0, 0, 0, 2, 0, 0, 0, 6,

		0, 6, 0, 0, 0, 0, 2, 8, 0,
		0, 0, 0, 4, 1, 9, 0, 0, 5,
		0, 0, 0, 0, 8, 0, 0, 7, 9,
	}

	s := NewFDStore()
	vars := make([]*FDVar, 81)
	for i := 0; i < 81; i++ {
		vars[i] = s.NewVar()
	}
	// apply givens
	for i := 0; i < 81; i++ {
		v := puzzle[i]
		if v != 0 {
			if !s.Assign(vars[i], v) {
				t.Fatalf("failed to assign given at %d", i)
			}
		}
	}
	// add Regin-based AllDifferent constraints
	for r := 0; r < 9; r++ {
		row := make([]*FDVar, 9)
		for c := 0; c < 9; c++ {
			row[c] = vars[r*9+c]
		}
		if !s.AddAllDifferentRegin(row) {
			t.Fatalf("Regin row constraint failed at row %d", r)
		}
	}
	for c := 0; c < 9; c++ {
		col := make([]*FDVar, 9)
		for r := 0; r < 9; r++ {
			col[r] = vars[r*9+c]
		}
		if !s.AddAllDifferentRegin(col) {
			t.Fatalf("Regin col constraint failed at col %d", c)
		}
	}
	for br := 0; br < 3; br++ {
		for bc := 0; bc < 3; bc++ {
			block := make([]*FDVar, 0, 9)
			for r := 0; r < 3; r++ {
				for c := 0; c < 3; c++ {
					idx := (br*3+r)*9 + (bc*3 + c)
					block = append(block, vars[idx])
				}
			}
			if !s.AddAllDifferentRegin(block) {
				t.Fatalf("Regin block constraint failed at block %d,%d", br, bc)
			}
		}
	}

	sols, err := s.Solve(context.Background(), 1)
	if err != nil {
		t.Fatalf("Solve error: %v", err)
	}
	if len(sols) == 0 {
		t.Fatalf("expected at least one solution for Sudoku partial")
	}
}
