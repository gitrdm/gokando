```go
func ExampleNewSLGEngine_customConfig() {
	config := &SLGConfig{
		MaxTableSize:          5000,
		MaxAnswersPerSubgoal:  100,
		MaxFixpointIterations: 500,
	}

	engine := NewSLGEngine(config)
	fmt.Printf("Max table size: %d\n", engine.config.MaxTableSize)
	fmt.Printf("Max fixpoint iterations: %d\n", engine.config.MaxFixpointIterations)

	// Output:
	// Max table size: 5000
	// Max fixpoint iterations: 500
}

```


