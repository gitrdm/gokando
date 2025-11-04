```go
func ExampleModel_Validate() {
	model := minikanren.NewModel()

	// Create a variable with normal domain
	// low-level: x := model.NewVariable(minikanren.NewBitSetDomain(5))
	x := model.IntVar(1, 5, "x")
	_ = x

	// Model is valid
	err := model.Validate()
	fmt.Printf("Valid model: %v\n", err == nil)

	// Create a variable with empty domain - this is an error
	emptyDomain := minikanren.NewBitSetDomainFromValues(5, []int{})
	// low-level: y := model.NewVariable(emptyDomain)
	y := model.NewVariable(emptyDomain)

	err = model.Validate()
	if err != nil {
		fmt.Printf("Invalid model: variable %s has empty domain\n", y.Name())
	}

	// Output:
	// Valid model: true
	// Invalid model: variable v1 has empty domain
}

```


