```go
func ExampleRembero() {
	q := Fresh("q")
	goal := Rembero(NewAtom("a"), List(NewAtom("a"), NewAtom("b"), NewAtom("a")), q)
	for _, s := range runGoal(goal, q) {
		fmt.Println(s)
	}
	// Output:
	// q: ("a" "b")
	// q: ("b" "a")
}

```


