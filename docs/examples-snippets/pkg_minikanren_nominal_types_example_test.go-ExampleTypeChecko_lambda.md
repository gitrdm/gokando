```go
func ExampleTypeChecko_lambda() {
	a := NewAtom("a")
	T := Fresh("T")
	term := Lambda(a, a)
	ty := ArrType(T, T)
	results := Run(1, func(q *Var) Goal { return TypeChecko(term, Nil, ty) })
	fmt.Println(len(results) > 0)
	// Output: true
}

```


