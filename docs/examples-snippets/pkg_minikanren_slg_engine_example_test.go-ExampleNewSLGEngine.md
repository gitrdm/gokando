```go
func ExampleNewSLGEngine() {
	engine := NewSLGEngine(nil)

	stats := engine.Stats()
	fmt.Printf("Initial subgoals: %d\n", stats.CachedSubgoals)
	fmt.Printf("Max answers per subgoal: %d\n", engine.config.MaxAnswersPerSubgoal)

	// Output:
	// Initial subgoals: 0
	// Max answers per subgoal: 10000
}

```


