```go
func ExampleSolutions_basic() {
	q := Fresh("q")
	goal := Disj(Eq(q, NewAtom(1)), Eq(q, NewAtom(2)))
	out := FormatSolutions(Solutions(goal, q))
	fmt.Println(strings.Join(out, "\n"))
	// Output:
	// q: 1
	// q: 2
}

```


