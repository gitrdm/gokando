```go
func ExampleFreeNameso_basic() {
	a := NewAtom("a")
	b := NewAtom("b")
	term := Lambda(a, App(a, b))

	results := Run(1, func(q *Var) Goal { return FreeNameso(term, q) })
	fmt.Println(results[0])
	// Output: (b . <nil>)
}

```


