```go
func ExampleNewHybridSolver() {
	// Create an FD model with variables and constraints
	model := NewModel()
	x := model.NewVariable(NewBitSetDomain(10))
	y := model.NewVariable(NewBitSetDomain(10))

	// Add FD constraint: x + 1 = y
	arith, _ := NewArithmetic(x, y, 1)
	model.AddConstraint(arith)

	// Create plugins explicitly to preserve the canonical demonstration order
	// (FD plugin followed by Relational). This example intentionally shows
	// the plugin ordering used elsewhere in the docs.
	fdPlugin := NewFDPlugin(model)
	relPlugin := NewRelationalPlugin()

	// Create hybrid solver with both plugins
	solver := NewHybridSolver(fdPlugin, relPlugin)

	fmt.Printf("Hybrid solver has %d plugins\n", len(solver.GetPlugins()))
	fmt.Printf("Plugin 1: %s\n", solver.GetPlugins()[0].Name())
	fmt.Printf("Plugin 2: %s\n", solver.GetPlugins()[1].Name())

	// Output:
	// Hybrid solver has 2 plugins
	// Plugin 1: FD
	// Plugin 2: Relational
}

```


