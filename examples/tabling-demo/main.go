package main

import (
	"context"
	"fmt"

	"github.com/gitrdm/gokando/pkg/minikanren"
)

func main() {
	ctx := context.Background()

	fmt.Println("=== Tabling Demo ===")
	fmt.Println("Demonstrating memoization of goal results")
	fmt.Println()

	// Create a simple goal that returns multiple results
	callCount := 0
	baseGoal := func(ctx context.Context, store minikanren.ConstraintStore) minikanren.ResultStream {
		callCount++
		stream := minikanren.NewStream()
		go func() {
			defer stream.Close()
			// Simulate some computation and return multiple results
			stream.Put(ctx, store) // First result
			stream.Put(ctx, store) // Second result
			stream.Put(ctx, store) // Third result
		}()
		return stream
	}

	// Create tabled version
	tabledGoal := minikanren.TableGoal(baseGoal)

	fmt.Printf("Executing tabled goal first time...\n")
	results1, _, err := tabledGoal(ctx, minikanren.NewLocalConstraintStore(nil)).Take(ctx, 10)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("First execution: got %d results, goal called %d times\n", len(results1), callCount)

	fmt.Printf("Executing tabled goal second time...\n")
	results2, _, err := tabledGoal(ctx, minikanren.NewLocalConstraintStore(nil)).Take(ctx, 10)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Second execution: got %d results, goal called %d times\n", len(results2), callCount)

	// Show tabling statistics
	stats := minikanren.GetTablingStats()
	fmt.Println()
	fmt.Printf("Tabling Statistics: %s\n", stats.String())

	fmt.Println()
	fmt.Println("Notice that the goal was only executed once, even though")
	fmt.Println("we ran the tabled goal twice. The second execution used")
	fmt.Println("the cached results from the first execution.")
}
