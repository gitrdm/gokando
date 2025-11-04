```go
func ExampleSubgoalStatus() {
	statuses := []SubgoalStatus{
		StatusActive,
		StatusComplete,
		StatusFailed,
		StatusInvalidated,
	}

	for _, status := range statuses {
		fmt.Printf("%s\n", status.String())
	}

	// Output:
	// Active
	// Complete
	// Failed
	// Invalidated
}

```


