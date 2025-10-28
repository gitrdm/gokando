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
	if err := s.Assign(a, 1); err != nil {
		t.Fatalf("failed to assign a: %v", err)
	}
	if err := s.Assign(b, 1); err != nil {
		t.Fatalf("failed to assign b: %v", err)
	}
	if err := s.ReginFilterLocked([]*FDVar{a, b}); err == nil {
		t.Fatalf("expected Regin to detect impossible matching but it returned ok")
	}
}

// Test that Regin preserves supported values and propagates singletons
func TestReginSingletonPropagation(t *testing.T) {
	s := NewFDStoreWithDomain(3)
	a := s.NewVar()
	b := s.NewVar()
	c := s.NewVar()

	// assign c=1, should propagate to remove 1 from a and b
	if err := s.Assign(c, 1); err != nil {
		t.Fatalf("failed to assign c: %v", err)
	}
	if err := s.ReginFilterLocked([]*FDVar{a, b, c}); err != nil {
		t.Fatalf("ReginFilterLocked failed: %v", err)
	}
	// check that 1 is removed from a and b
	if a.domain.Has(1) {
		t.Fatalf("expected 1 removed from a")
	}
	if b.domain.Has(1) {
		t.Fatalf("expected 1 removed from b")
	}
}
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
			if err := s.Assign(vars[i], v); err != nil {
				t.Fatalf("failed to assign given at %d: %v", i, err)
			}
		}
	}
	// add Regin-based AllDifferent constraints
	for r := 0; r < 9; r++ {
		row := make([]*FDVar, 9)
		for c := 0; c < 9; c++ {
			row[c] = vars[r*9+c]
		}
		if err := s.AddAllDifferentRegin(row); err != nil {
			t.Fatalf("Regin row constraint failed at row %d: %v", r, err)
		}
	}
	for c := 0; c < 9; c++ {
		col := make([]*FDVar, 9)
		for r := 0; r < 9; r++ {
			col[r] = vars[r*9+c]
		}
		if err := s.AddAllDifferentRegin(col); err != nil {
			t.Fatalf("Regin col constraint failed at col %d: %v", c, err)
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
			if err := s.AddAllDifferentRegin(block); err != nil {
				t.Fatalf("Regin block constraint failed at block %d,%d: %v", br, bc, err)
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
