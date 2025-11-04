```go
func ExampleQueryEvaluator() {
	parent, _ := DbRel("parent", 2, 0, 1)
	db := NewDatabase()
	db, _ = db.AddFact(parent, NewAtom("alice"), NewAtom("bob"))
	db, _ = db.AddFact(parent, NewAtom("alice"), NewAtom("charlie"))

	child := Fresh("child")
	query := db.Query(parent, NewAtom("alice"), child)

	// Convert to GoalEvaluator
	evaluator := QueryEvaluator(query, child.ID())

	ctx := context.Background()
	answers := make(chan map[int64]Term, 10)

	go func() {
		defer close(answers)
		_ = evaluator(ctx, answers)
	}()

	count := 0
	for range answers {
		count++
	}

	fmt.Printf("Alice has %d children\n", count)

	// Output:
	// Alice has 2 children
}

```


