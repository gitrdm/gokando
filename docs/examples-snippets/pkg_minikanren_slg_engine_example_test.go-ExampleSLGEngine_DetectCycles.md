```go
func ExampleSLGEngine_DetectCycles() {
	engine := NewSLGEngine(nil)

	// Create three subgoals with dependencies
	patternA := NewCallPattern("ancestor", []Term{NewAtom("alice"), NewAtom("x")})
	patternB := NewCallPattern("ancestor", []Term{NewAtom("bob"), NewAtom("x")})
	patternC := NewCallPattern("ancestor", []Term{NewAtom("charlie"), NewAtom("x")})

	entryA, _ := engine.subgoals.GetOrCreate(patternA)
	entryB, _ := engine.subgoals.GetOrCreate(patternB)
	entryC, _ := engine.subgoals.GetOrCreate(patternC)

	// Create cycle: A -> B -> C -> B
	entryA.AddDependency(entryB)
	entryB.AddDependency(entryC)
	entryC.AddDependency(entryB)

	// Detect cycles
	sccs := engine.DetectCycles()

	fmt.Printf("Found %d SCCs\n", len(sccs))

	// Check if cyclic
	if engine.IsCyclic() {
		fmt.Println("Graph contains cycles")
	}

	// Find the cyclic SCC
	for _, scc := range sccs {
		if len(scc.nodes) > 1 {
			fmt.Printf("Cyclic SCC has %d nodes\n", len(scc.nodes))
		}
	}

	// Output:
	// Found 2 SCCs
	// Graph contains cycles
	// Cyclic SCC has 2 nodes
}

```


