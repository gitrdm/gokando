```go
func ExampleFresho_violation() {
	a := NewAtom("a")

	// term: (a . ()) — 'a' appears free in the list
	list := NewPair(a, Nil)

	ctx := context.Background()
	goal := Fresho(a, list)
	stream := goal(ctx, NewLocalConstraintStore(NewGlobalConstraintBus()))
	results, _ := stream.Take(1)
	fmt.Println(len(results))
	// Output: 0
}

```


