// Package main solves graph coloring problems using GoKando.package graphcoloring

// Graph Coloring Problem: Color the vertices of a graph such that no two
// adjacent vertices share the same color, using the minimum number of colors.
//
// This example demonstrates:
// - Pure relational constraints with Neq
// - Parallel search with ParallelDisj and ParallelRun
// - Performance comparison between sequential and parallel execution
//
// The example uses a map of Australia with 7 regions and demonstrates
// the classic 3-coloring problem.
package main

import (
	"fmt"
	"os"
	"time"

	. "github.com/gitrdm/gokando/pkg/minikanren"
)

func main() {
	fmt.Println("=== Graph Coloring: Map of Australia ===")
	fmt.Println()
	fmt.Println("Regions to color:")
	fmt.Println("  WA (Western Australia)")
	fmt.Println("  NT (Northern Territory)")
	fmt.Println("  SA (South Australia)")
	fmt.Println("  Q  (Queensland)")
	fmt.Println("  NSW (New South Wales)")
	fmt.Println("  V  (Victoria)")
	fmt.Println("  T  (Tasmania)")
	fmt.Println()
	fmt.Println("Adjacencies:")
	fmt.Println("  WA: NT, SA")
	fmt.Println("  NT: WA, SA, Q")
	fmt.Println("  SA: WA, NT, Q, NSW, V")
	fmt.Println("  Q:  NT, SA, NSW")
	fmt.Println("  NSW: Q, SA, V")
	fmt.Println("  V:  SA, NSW")
	fmt.Println("  T:  (island, no adjacencies)")
	fmt.Println()

	// Allow user to choose mode
	useParallel := true
	if len(os.Args) > 1 && os.Args[1] == "seq" {
		useParallel = false
	}

	if useParallel {
		fmt.Println("🚀 Using PARALLEL search with ParallelRun and ParallelDisj")
	} else {
		fmt.Println("⏱️  Using SEQUENTIAL search")
	}
	fmt.Println()

	start := time.Now()
	var results []Term

	if useParallel {
		results = ParallelRun(1, australiaColoringParallel)
	} else {
		results = Run(1, australiaColoringSequential)
	}

	elapsed := time.Since(start)

	if len(results) == 0 {
		fmt.Println("❌ No solution found!")
		return
	}

	fmt.Printf("✓ Solution found in %v!\n\n", elapsed)
	displaySolution(results[0])

	fmt.Println()
	if useParallel {
		fmt.Println("💡 Run with 'seq' argument to compare with sequential search:")
		fmt.Println("   ./graph-coloring seq")
	} else {
		fmt.Println("💡 Run without arguments to see parallel search:")
		fmt.Println("   ./graph-coloring")
	}
}

// australiaColoringSequential solves the graph coloring using sequential search
func australiaColoringSequential(q *Var) Goal {
	// Create variables for each region's color
	wa := Fresh("WA")
	nt := Fresh("NT")
	sa := Fresh("SA")
	qld := Fresh("Q")
	nsw := Fresh("NSW")
	vic := Fresh("V")
	tas := Fresh("T")

	// Available colors
	red := NewAtom("red")
	green := NewAtom("green")
	blue := NewAtom("blue")

	// Helper: region can be one of three colors (sequential)
	color := func(region Term) Goal {
		return Disj(
			Eq(region, red),
			Eq(region, green),
			Eq(region, blue),
		)
	}

	return Conj(
		// Each region must have a color
		color(wa), color(nt), color(sa), color(qld),
		color(nsw), color(vic), color(tas),

		// Adjacent regions must have different colors
		// WA adjacencies
		Neq(wa, nt),
		Neq(wa, sa),

		// NT adjacencies
		Neq(nt, wa),
		Neq(nt, sa),
		Neq(nt, qld),

		// SA adjacencies (most connected)
		Neq(sa, wa),
		Neq(sa, nt),
		Neq(sa, qld),
		Neq(sa, nsw),
		Neq(sa, vic),

		// Q adjacencies
		Neq(qld, nt),
		Neq(qld, sa),
		Neq(qld, nsw),

		// NSW adjacencies
		Neq(nsw, qld),
		Neq(nsw, sa),
		Neq(nsw, vic),

		// V adjacencies
		Neq(vic, sa),
		Neq(vic, nsw),

		// T is an island - no adjacencies

		// Return solution
		Eq(q, List(
			List(NewAtom("WA"), wa),
			List(NewAtom("NT"), nt),
			List(NewAtom("SA"), sa),
			List(NewAtom("Q"), qld),
			List(NewAtom("NSW"), nsw),
			List(NewAtom("V"), vic),
			List(NewAtom("T"), tas),
		)),
	)
}

// Shared parallel executor for graph coloring
var graphColoringExecutor *ParallelExecutor

func init() {
	graphColoringExecutor = NewParallelExecutor(DefaultParallelConfig())
}

// australiaColoringParallel solves the graph coloring using parallel search
func australiaColoringParallel(q *Var) Goal {
	// Create variables for each region's color
	wa := Fresh("WA")
	nt := Fresh("NT")
	sa := Fresh("SA")
	qld := Fresh("Q")
	nsw := Fresh("NSW")
	vic := Fresh("V")
	tas := Fresh("T")

	// Available colors
	red := NewAtom("red")
	green := NewAtom("green")
	blue := NewAtom("blue")

	// Helper: region can be one of three colors (parallel exploration)
	colorParallel := func(region Term) Goal {
		return graphColoringExecutor.ParallelDisj(
			Eq(region, red),
			Eq(region, green),
			Eq(region, blue),
		)
	}

	return Conj(
		// Each region must have a color (explored in parallel)
		colorParallel(wa),
		colorParallel(nt),
		colorParallel(sa),
		colorParallel(qld),
		colorParallel(nsw),
		colorParallel(vic),
		colorParallel(tas),

		// Adjacent regions must have different colors
		// WA adjacencies
		Neq(wa, nt),
		Neq(wa, sa),

		// NT adjacencies
		Neq(nt, wa),
		Neq(nt, sa),
		Neq(nt, qld),

		// SA adjacencies (most connected - central hub)
		Neq(sa, wa),
		Neq(sa, nt),
		Neq(sa, qld),
		Neq(sa, nsw),
		Neq(sa, vic),

		// Q adjacencies
		Neq(qld, nt),
		Neq(qld, sa),
		Neq(qld, nsw),

		// NSW adjacencies
		Neq(nsw, qld),
		Neq(nsw, sa),
		Neq(nsw, vic),

		// V adjacencies
		Neq(vic, sa),
		Neq(vic, nsw),

		// T is an island - no adjacencies

		// Return solution
		Eq(q, List(
			List(NewAtom("WA"), wa),
			List(NewAtom("NT"), nt),
			List(NewAtom("SA"), sa),
			List(NewAtom("Q"), qld),
			List(NewAtom("NSW"), nsw),
			List(NewAtom("V"), vic),
			List(NewAtom("T"), tas),
		)),
	)
}

// displaySolution pretty-prints the coloring solution
func displaySolution(result Term) {
	regions := []string{"WA", "NT", "SA", "Q", "NSW", "V", "T"}
	colors := make(map[string]string)

	// Navigate through the list of region-color pairs
	current := result
	for i := 0; i < 7; i++ {
		pair, ok := current.(*Pair)
		if !ok {
			break
		}

		// Each element is a pair like (WA . red)
		regionPair, ok := pair.Car().(*Pair)
		if !ok {
			break
		}

		// Extract the color from the cdr of the region pair
		color := extractAtom(regionPair.Cdr())
		if color != "" {
			colors[regions[i]] = color
		}

		// Move to next element
		current = pair.Cdr()
	}

	// Display the solution with color indicators
	fmt.Println("Region Coloring:")
	for _, region := range regions {
		color := colors[region]
		emoji := colorEmoji(color)
		fmt.Printf("  %-4s : %s %s\n", region, emoji, color)
	}

	// Verify it's a valid 3-coloring
	usedColors := make(map[string]bool)
	for _, color := range colors {
		if color != "" {
			usedColors[color] = true
		}
	}
	fmt.Printf("\n✅ Valid %d-coloring found\n", len(usedColors))
}

// extractAtom extracts the string value from an Atom term
func extractAtom(term Term) string {
	// If it's a Pair, get the first element (car)
	if pair, ok := term.(*Pair); ok {
		term = pair.Car()
	}
	if atom, ok := term.(*Atom); ok {
		return fmt.Sprintf("%v", atom.Value())
	}
	return ""
}

// colorEmoji returns an emoji for visual representation
func colorEmoji(color string) string {
	switch color {
	case "red":
		return "🔴"
	case "green":
		return "🟢"
	case "blue":
		return "🔵"
	default:
		return "⚪"
	}
}
