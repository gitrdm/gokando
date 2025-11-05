```go
func ExampleDefaultParallelSearchConfig() {
	cfg := DefaultParallelSearchConfig()
	// You can use cfg.NumWorkers to size solver parallelism, and cfg.WorkQueueSize
	// to adjust throughput vs memory. Only queue size is deterministic here.
	fmt.Printf("queue=%d\n", cfg.WorkQueueSize)
	// Output:
	// queue=1000
}

```


