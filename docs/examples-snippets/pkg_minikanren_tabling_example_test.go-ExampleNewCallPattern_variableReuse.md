```go
func ExampleNewCallPattern_variableReuse() {
	v := &Var{id: 42, name: "x"}
	// path(X, X) - same variable twice
	pattern := NewCallPattern("path", []Term{v, v})

	fmt.Printf("Structure: %s\n", pattern.ArgStructure())

	// Output:
	// Structure: X0,X0
}

```


