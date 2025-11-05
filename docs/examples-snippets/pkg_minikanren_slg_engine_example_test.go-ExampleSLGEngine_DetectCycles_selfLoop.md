```go
func ExampleSLGEngine_DetectCycles_selfLoop() {
	engine := NewSLGEngine(nil)

	// Create a recursive predicate: path(X, Y)
	pattern := NewCallPattern("path", []Term{NewAtom("x"), NewAtom("y")})
	entry, _ := engine.subgoals.GetOrCreate(pattern)

	// Create self-loop (path depends on path)
	entry.AddDependency(entry)

	if engine.IsCyclic() {
		fmt.Println("Self-referential predicate detected")
	}

	sccs := engine.DetectCycles()
	for _, scc := range sccs {
		if scc.Contains(entry) {
			fmt.Printf("SCC contains %d node(s)\n", len(scc.nodes))
		}
	}

	// Output:
	// Self-referential predicate detected
	// SCC contains 1 node(s)
}

```


