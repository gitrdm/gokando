```go
func ExampleSCC_AnswerCount() {
	pattern1 := NewCallPattern("p", []Term{NewAtom(1)})
	pattern2 := NewCallPattern("q", []Term{NewAtom(2)})

	entry1 := NewSubgoalEntry(pattern1)
	entry2 := NewSubgoalEntry(pattern2)

	// Add answers to both entries
	entry1.Answers().Insert(map[int64]Term{1: NewAtom("a")})
	entry1.Answers().Insert(map[int64]Term{1: NewAtom("b")})
	entry2.Answers().Insert(map[int64]Term{1: NewAtom("c")})

	scc := &SCC{nodes: []*SubgoalEntry{entry1, entry2}}

	fmt.Printf("Total answers in SCC: %d\n", scc.AnswerCount())

	// Output:
	// Total answers in SCC: 3
}

```


