```go
func ExampleSubgoalTable() {
	table := NewSubgoalTable()

	// Create a call pattern
	pattern := NewCallPattern("edge", []Term{NewAtom("a"), NewAtom("b")})

	// Get or create a subgoal entry
	entry, created := table.GetOrCreate(pattern)
	fmt.Printf("Created new entry: %v\n", created)
	fmt.Printf("Entry status: %s\n", entry.Status())

	// Subsequent calls return the same entry
	entry2, created2 := table.GetOrCreate(pattern)
	fmt.Printf("Created on second call: %v\n", created2)
	fmt.Printf("Same entry: %v\n", entry == entry2)

	fmt.Printf("Total subgoals: %d\n", table.TotalSubgoals())

	// Output:
	// Created new entry: true
	// Entry status: Active
	// Created on second call: false
	// Same entry: true
	// Total subgoals: 1
}

```


