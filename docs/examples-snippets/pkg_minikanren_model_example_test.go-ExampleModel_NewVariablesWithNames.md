```go
func ExampleModel_NewVariablesWithNames() {
	model := minikanren.NewModel()

	names := []string{"red", "green", "blue"}
	colors := model.NewVariablesWithNames(names, minikanren.NewBitSetDomain(3))

	for _, v := range colors {
		fmt.Printf("%s\n", v.String())
	}

	// Output:
	// red∈{1..3}
	// green∈{1..3}
	// blue∈{1..3}
}

```


