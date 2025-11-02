package main

import (
	"context"
	"fmt"
	"sort"
	"time"

	mk "github.com/gitrdm/gokando/pkg/minikanren"
)

// small symmetric TSP instance (n=5) with obvious optimal tour
// Distances are 1-based indexed: nodes 1..5
func distances() [][]int {
	// 0 at diagonal, symmetric values elsewhere
	return [][]int{
		// 0th row unused to keep 1-based indexing consistent in code
		{},
		{0, 0, 2, 9, 10, 7}, // padded to length 6, entries [1..5] used
		{0, 2, 0, 6, 4, 3},
		{0, 9, 6, 0, 8, 5},
		{0, 10, 4, 8, 0, 6},
		{0, 7, 3, 5, 6, 0},
	}
}

func main() {
	fmt.Println("=== Small TSP with Circuit (n=5) ===")

	n := 5
	start := 1
	d := distances()

	model := mk.NewModel()

	// succ[i] in [1..n]
	succ := make([]*mk.FDVariable, n)
	for i := 0; i < n; i++ {
		succ[i] = model.NewVariableWithName(mk.NewBitSetDomain(n), fmt.Sprintf("succ_%d", i+1))
	}

	// Post Circuit
	circ, err := mk.NewCircuit(model, succ, start)
	if err != nil {
		panic(err)
	}
	model.AddConstraint(circ)

	// Solve: enumerate up to 200 solutions (should cover all Hamiltonian cycles)
	solver := mk.NewSolver(model)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	sols, err := solver.Solve(ctx, 200)
	if err != nil {
		fmt.Printf("Solve error: %v\n", err)
		return
	}
	if len(sols) == 0 {
		fmt.Println("No tours found (unexpected for Circuit)")
		return
	}

	type tour struct {
		order []int
		cost  int
	}

	tours := make([]tour, 0, len(sols))
	for _, sol := range sols {
		// Build succ mapping and traverse from start
		next := func(i int) int { return sol[succ[i-1].ID()] }

		seen := make(map[int]bool, n)
		order := make([]int, 0, n+1)
		cur := start
		for {
			if seen[cur] {
				break
			}
			seen[cur] = true
			order = append(order, cur)
			cur = next(cur)
		}
		// Must return to start and visit all nodes
		if cur != start || len(order) != n {
			continue
		}

		// Compute cost of cycle (including edge from last to start via succ map)
		cost := 0
		for i := 0; i < n; i++ {
			from := order[i]
			to := next(from)
			cost += d[from][to]
		}
		tours = append(tours, tour{order: order, cost: cost})
	}

	if len(tours) == 0 {
		fmt.Println("No Hamiltonian cycles assembled from solutions")
		return
	}

	// Normalize tours by rotational symmetry to avoid duplicates starting at different nodes
	// Here we already enforce start=1 as first node, but we also canonicalize orientation
	for i := range tours {
		// ensure order starts at 1 (already true), and choose lexicographically smaller orientation
		o := tours[i].order
		// construct reverse
		rev := make([]int, len(o))
		for k := range o {
			rev[k] = o[len(o)-1-k]
		}
		// rotate rev so that it starts at start
		if rev[0] != start {
			// find start index
			idx := 0
			for ; idx < len(rev); idx++ {
				if rev[idx] == start {
					break
				}
			}
			tmp := make([]int, len(rev))
			copy(tmp, rev[idx:])
			copy(tmp[len(rev)-idx:], rev[:idx])
			rev = tmp
		}
		// pick lexicographically smaller between o and rev
		pickRev := false
		for k := 0; k < len(o); k++ {
			if rev[k] < o[k] {
				pickRev = true
				break
			}
			if rev[k] > o[k] {
				break
			}
		}
		if pickRev {
			tours[i].order = rev
		}
	}

	// Deduplicate tours by order sequence
	type key struct{ seq string }
	uniq := map[key]tour{}
	for _, t := range tours {
		s := fmt.Sprint(t.order)
		uniq[key{s}] = t
	}
	tours = tours[:0]
	for _, t := range uniq {
		tours = append(tours, t)
	}

	sort.Slice(tours, func(i, j int) bool { return tours[i].cost < tours[j].cost })

	best := tours[0]
	fmt.Printf("Found %d unique tours. Best cost = %d\n", len(tours), best.cost)
	fmt.Printf("Best cycle: ")
	for i := 0; i < len(best.order); i++ {
		if i > 0 {
			fmt.Print(" -> ")
		}
		fmt.Print(best.order[i])
	}
	// show return to start
	fmt.Printf(" -> %d\n", start)
}
