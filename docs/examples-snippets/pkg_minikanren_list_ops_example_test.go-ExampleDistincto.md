```go
func ExampleDistincto() {
	goalSuccess := Distincto(List(NewAtom(1), NewAtom(2), NewAtom(3)))
	resultsSuccess := runGoal(goalSuccess)
	fmt.Printf("Distinct list succeeds: %v\n", len(resultsSuccess) > 0)

	goalFail := Distincto(List(NewAtom(1), NewAtom(2), NewAtom(1)))
	resultsFail := runGoal(goalFail)
	fmt.Printf("Non-distinct list fails: %v\n", len(resultsFail) == 0)
	// Output:
	// Distinct list succeeds: true
	// Non-distinct list fails: true
}

```


