```go
func ExampleModel_NewVariables() {
	model := minikanren.NewModel()

	// Create 4 variables with domains {1..9} for a 4-cell Sudoku
	vars := model.NewVariables(4, minikanren.NewBitSetDomain(9))

	fmt.Printf("Created %d variables\n", len(vars))
	for i, v := range vars {
		fmt.Printf("var[%d]: %s\n", i, v.String())
	}

	// Output:
	// Created 4 variables
	// var[0]: v0∈{1..9}
	// var[1]: v1∈{1..9}
	// var[2]: v2∈{1..9}
	// var[3]: v3∈{1..9}
}

```


