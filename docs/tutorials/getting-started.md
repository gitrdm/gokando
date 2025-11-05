## Getting started with gokanlogic

This short page shows a minimal example of creating a model and variables using the gokanlogic `minikanren` package. The examples below are extracted from the project's Example tests and include deterministic `// Output:` blocks so they can be verified automatically.

### Create a model and variables

The following Example demonstrates creating a model and two variables. It prints a short description and the variables' string forms.

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

### Reified constraints (boolean reification)

Reification converts a constraint into a boolean variable. The next example shows how to reify `X + 0 = Y` into boolean `B` and prints a few deterministic example solutions.

```go
func ExampleReifiedConstraint() {
    model := NewModel()
    x := model.NewVariableWithName(NewBitSetDomain(3), "X")
    y := model.NewVariableWithName(NewBitSetDomain(3), "Y")
    b := model.NewVariableWithName(NewBitSetDomain(2), "B") // {1,2} maps to {false,true}

    arith, _ := NewArithmetic(x, y, 0) // X + 0 = Y
    reified, _ := NewReifiedConstraint(arith, b)
    model.AddConstraint(reified)

    solver := NewSolver(model)
    solutions, _ := solver.Solve(context.Background(), 0)

    // Collect and sort output to make the example deterministic.
    var lines []string
    for _, sol := range solutions {
        lines = append(lines, fmt.Sprintf("X=%d Y=%d B=%t", sol[x.ID()], sol[y.ID()], sol[b.ID()] == 2))
    }
    sort.Strings(lines)

    for i := 0; i < 3 && i < len(lines); i++ {
        fmt.Println(lines[i])
    }
    // Output:
    // X=1 Y=1 B=true
    // X=1 Y=2 B=false
    // X=1 Y=3 B=false
}
```

Notes
- These snippets were extracted automatically from the repository's Example tests and include verified output blocks. Use `go test -run Example ./...` to validate them locally.
