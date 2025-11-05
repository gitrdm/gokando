```go
func ExampleNewModel() {
	model := minikanren.NewModel()

	// Create variables for a simple problem
	domain := minikanren.NewBitSetDomain(5)
	x := model.NewVariable(domain)
	y := model.NewVariable(domain)

	fmt.Printf("Model has %d variables\n", model.VariableCount())
	fmt.Printf("Variable x: %s\n", x.String())
	fmt.Printf("Variable y: %s\n", y.String())

	// Output:
	// Model has 2 variables
	// Variable x: v0∈{1..5}
	// Variable y: v1∈{1..5}
}

```


