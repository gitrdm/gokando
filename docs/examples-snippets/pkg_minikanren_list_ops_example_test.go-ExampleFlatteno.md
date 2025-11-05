```go
func ExampleFlatteno() {
	q := Fresh("q")
	nested := List(List(NewAtom(1), NewAtom(2)), List(NewAtom(3), List(NewAtom(4), NewAtom(5))))
	goal := Flatteno(nested, q)
	for _, s := range runGoal(goal, q) {
		fmt.Println(s)
	}
	// Output:
	// q: (1 2 3 4 5)
}

```


