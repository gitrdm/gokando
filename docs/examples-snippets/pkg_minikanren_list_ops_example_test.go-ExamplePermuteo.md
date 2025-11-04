```go
func ExamplePermuteo() {
	q := Fresh("q")
	goal := Permuteo(List(NewAtom(1), NewAtom(2), NewAtom(3)), q)
	results := runGoal(goal, q)
	fmt.Println(strings.Join(results, "\n"))
	// Output:
	// q: (1 2 3)
	// q: (1 3 2)
	// q: (2 1 3)
	// q: (2 3 1)
	// q: (3 1 2)
	// q: (3 2 1)
}

```


