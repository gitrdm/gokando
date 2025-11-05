```go
func ExampleAlphaEqo_nested() {
	a := NewAtom("a")
	b := NewAtom("b")

	// λa.λb.a  vs  λa.λb.b  (not alpha-equivalent)
	t1 := Lambda(a, Lambda(b, a))
	t2 := Lambda(a, Lambda(b, b))

	ctx := context.Background()
	goal := AlphaEqo(t1, t2)
	stream := goal(ctx, NewLocalConstraintStore(NewGlobalConstraintBus()))
	rs, _ := stream.Take(1)
	fmt.Println(len(rs))
	// Output: 0
}

```


