```go
func ExampleFDPlugin() {
	// Create model with AllDifferent constraint
	model := NewModel()
	x := model.NewVariable(NewBitSetDomain(5))
	y := model.NewVariable(NewBitSetDomain(5))
	z := model.NewVariable(NewBitSetDomain(5))

	allDiff, _ := NewAllDifferent([]*FDVariable{x, y, z})
	model.AddConstraint(allDiff)

	// Create FD plugin
	plugin := NewFDPlugin(model)

	fmt.Printf("Plugin name: %s\n", plugin.Name())
	fmt.Printf("Can handle AllDifferent: %v\n", plugin.CanHandle(allDiff))

	// Output:
	// Plugin name: FD
	// Can handle AllDifferent: true
}

```


