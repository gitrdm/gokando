```go
func ExampleNewCallPattern_variables() {
	// Two calls with different variable IDs but same structure
	v1 := &Var{id: 42, name: "x"}
	v2 := &Var{id: 73, name: "y"}
	pattern1 := NewCallPattern("path", []Term{v1, v2})

	v3 := &Var{id: 100, name: "p"}
	v4 := &Var{id: 200, name: "q"}
	pattern2 := NewCallPattern("path", []Term{v3, v4})

	fmt.Printf("Pattern 1: %s\n", pattern1.ArgStructure())
	fmt.Printf("Pattern 2: %s\n", pattern2.ArgStructure())
	fmt.Printf("Are equal: %v\n", pattern1.Equal(pattern2))

	// Output:
	// Pattern 1: X0,X1
	// Pattern 2: X0,X1
	// Are equal: true
}

```


