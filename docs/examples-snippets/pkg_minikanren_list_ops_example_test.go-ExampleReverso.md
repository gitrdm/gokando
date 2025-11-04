```go
func ExampleReverso() {
	q := Fresh("q")
	goal := Reverso(List(NewAtom(1), NewAtom(2), NewAtom(3)), q)
	for _, s := range runGoal(goal, q) {
		fmt.Println(s)
	}
	// Output:
	// q: (3 2 1)
}

```


