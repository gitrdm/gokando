```go
func ExampleCallGoal() {
	res := Run(10, func(q *Var) Goal {
		// Create a goal and store it in an atom
		goalAtom := NewAtom(Eq(q, NewAtom("ok")))
		// Invoke the goal indirectly
		return CallGoal(goalAtom)
	})
	for _, r := range res {
		fmt.Println(r)
	}
	// Output: ok
}

```


