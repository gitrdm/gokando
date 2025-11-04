```go
func ExampleModel_Validate() {
	model := minikanren.NewModel()

	// Create a variable with normal domain
	x := model.NewVariable(minikanren.NewBitSetDomain(5))
	_ = x

	// Model is valid
	err := model.Validate()
	fmt.Printf("Valid model: %v\n", err == nil)

	// Create a variable with empty domain - this is an error
	emptyDomain := minikanren.NewBitSetDomainFromValues(5, []int{})
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


