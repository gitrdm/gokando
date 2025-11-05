```go
func ExampleTypeChecko_app() {
	b := NewAtom("b")
	intT := NewAtom("Int")
	id := NewAtom("id")
	env := EnvExtend(EnvExtend(Nil, b, intT), id, ArrType(intT, intT))
	term := App(id, b)
	results := Run(1, func(q *Var) Goal { return TypeChecko(term, env, intT) })
	fmt.Println(len(results) > 0)
	// Output: true
}

```


