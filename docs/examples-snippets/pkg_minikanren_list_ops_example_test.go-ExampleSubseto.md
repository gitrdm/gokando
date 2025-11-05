```go
func ExampleSubseto() {
	q := Fresh("q")
	goal := Subseto(q, List(NewAtom(1), NewAtom(2)))
	results := runGoal(goal, q)
	fmt.Println(strings.Join(results, "\n"))
	// Output:
	// q: ()
	// q: (1 2)
	// q: (1)
	// q: (2)
}

```


