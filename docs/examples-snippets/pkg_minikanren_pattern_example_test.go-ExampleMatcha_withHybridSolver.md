```go
func ExampleMatcha_withHybridSolver() {
	model := NewModel()
	x := model.NewVariable(NewBitSetDomainFromValues(100, []int{5, 10, 15}))

	store := NewUnifiedStore()
	store, _ = store.SetDomain(x.ID(), x.Domain())
	adapter := NewUnifiedStoreAdapter(store)

	q := Fresh("q")
	val := Fresh("val")

	goal := Conj(
		Eq(val, NewAtom(5)),
		Matcha(val,
			NewClause(NewAtom(5), Eq(q, NewAtom("small"))),
			NewClause(NewAtom(10), Eq(q, NewAtom("medium"))),
			NewClause(NewAtom(15), Eq(q, NewAtom("large"))),
		),
	)

	ctx := context.Background()
	stream := goal(ctx, adapter)
	results, _ := stream.Take(1)

	if len(results) > 0 {
		binding := results[0].GetBinding(q.ID())
		if atom, ok := binding.(*Atom); ok {
			fmt.Printf("Classification: %v\n", atom.value)
		}
	}

	// Output:
	// Classification: small
}

```


