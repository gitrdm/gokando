// Package main solves graph coloring problems using GoKando.
//
// Graph Coloring Problem: Color the vertices of a graph such that no two
// adjacent vertices share the same color, using the minimum number of colors.
//
// This example demonstrates:
// - Relational HLAPI with A(), L() term sugar
// - SolutionsN() for structured result extraction
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

	// Create fresh variables for regions
	wa := Fresh("WA")
	nt := Fresh("NT")
	sa := Fresh("SA")
	qld := Fresh("Q")
	nsw := Fresh("NSW")
	vic := Fresh("V")
	tas := Fresh("T")

	var goal Goal
	if useParallel {
		goal = australiaColoringParallel(wa, nt, sa, qld, nsw, vic, tas)
	} else {
		goal = australiaColoringSequential(wa, nt, sa, qld, nsw, vic, tas)
	}

	// Use HLAPI Rows to get all region colors as a table
	results := Rows(goal, wa, nt, sa, qld, nsw, vic, tas)

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
func australiaColoringSequential(wa, nt, sa, qld, nsw, vic, tas *Var) Goal {
	// Available colors (using HLAPI A() sugar)
	red := A("red")
	green := A("green")
	blue := A("blue")

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
	)
}

// Shared parallel executor for graph coloring
var graphColoringExecutor *ParallelExecutor

func init() {
	graphColoringExecutor = NewParallelExecutor(DefaultParallelConfig())
}

// australiaColoringParallel solves the graph coloring using parallel search
func australiaColoringParallel(wa, nt, sa, qld, nsw, vic, tas *Var) Goal {
	// Available colors (using HLAPI A() sugar)
	red := A("red")
	green := A("green")
	blue := A("blue")

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
	)
}

// displaySolution pretty-prints the coloring solution using HLAPI result format
func displaySolution(row []Term) {
	// Order of variables passed to Rows(): WA, NT, SA, Q, NSW, V, T
	regionNames := []string{"WA", "NT", "SA", "Q", "NSW", "V", "T"}
	colors := make(map[string]string)

	// Extract colors from HLAPI Rows() result
	for i, region := range regionNames {
		if i < len(row) {
			if atom, ok := row[i].(*Atom); ok {
				if colorStr, ok := atom.Value().(string); ok {
					colors[region] = colorStr
				}
			}
		}
	}

	// Display the solution with color indicators
	fmt.Println("Region Coloring:")
	for _, region := range regionNames {
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
