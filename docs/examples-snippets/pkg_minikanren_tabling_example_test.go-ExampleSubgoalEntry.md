```go
func ExampleSubgoalEntry() {
	pattern := NewCallPattern("fib", []Term{NewAtom(5)})
	entry := NewSubgoalEntry(pattern)

	fmt.Printf("Initial status: %s\n", entry.Status())
	fmt.Printf("Answer count: %d\n", entry.Answers().Count())

	// Add an answer
	bindings := map[int64]Term{1: NewAtom(8)} // fib(5) = 8
	entry.Answers().Insert(bindings)

	fmt.Printf("After insertion: %d answers\n", entry.Answers().Count())

	// Mark as complete
	entry.SetStatus(StatusComplete)
	fmt.Printf("Final status: %s\n", entry.Status())

	// Output:
	// Initial status: Active
	// Answer count: 0
	// After insertion: 1 answers
	// Final status: Complete
}

```


