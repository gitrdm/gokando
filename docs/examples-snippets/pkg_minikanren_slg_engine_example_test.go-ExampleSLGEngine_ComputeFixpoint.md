```go
func ExampleSLGEngine_ComputeFixpoint() {
	engine := NewSLGEngine(nil)

	// Create two mutually dependent subgoals
	pattern1 := NewCallPattern("reaches", []Term{NewAtom("a"), NewAtom("x")})
	pattern2 := NewCallPattern("reaches", []Term{NewAtom("b"), NewAtom("x")})

	entry1, _ := engine.subgoals.GetOrCreate(pattern1)
	entry2, _ := engine.subgoals.GetOrCreate(pattern2)

	// Add initial answers
	entry1.Answers().Insert(map[int64]Term{1: NewAtom("node1")})
	entry2.Answers().Insert(map[int64]Term{1: NewAtom("node2")})

	// Create mutual dependency (cycle)
	entry1.AddDependency(entry2)
	entry2.AddDependency(entry1)

	scc := &SCC{nodes: []*SubgoalEntry{entry1, entry2}}

	// Compute fixpoint
	err := engine.ComputeFixpoint(context.Background(), scc)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Println("Fixpoint computed successfully")
		fmt.Printf("Total answers: %d\n", scc.AnswerCount())
	}

	// Output:
	// Fixpoint computed successfully
	// Total answers: 2
}

```


