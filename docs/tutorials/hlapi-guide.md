## HLAPI guide (high-level API)

This page presents a couple of common HLAPI usage patterns: collecting integer solutions and running an optimization.

### Collect integer results deterministically

Use the `Ints` HLAPI helper to collect integer values produced by a relational goal. The example below prints the number of values and their sum (order-independent), which makes the example deterministic.

```go
func Example_hlapi_collectors_ints() {
    x := Fresh("x")
    goal := Disj(Eq(x, A(1)), Eq(x, A(3)), Eq(x, A(5)))
    vals := Ints(goal, x)
    // Print count and sum to avoid relying on order
    sum := 0
    for _, v := range vals {
        sum += v
    }
    fmt.Printf("%d %d\n", len(vals), sum)
    // Output:
    // 3 9
}
```

### Optimize a linear objective

The HLAPI provides a convenient `Optimize` function that returns the best objective value when optimizing an integer variable. The snippet below minimizes the linear expression `t = x1 + 2*x2`.

```go
func Example_hlapi_optimize() {
    m := NewModel()
    xs := m.IntVars(2, 1, 3, "x") // x1, x2 in [1..3]
    total := m.IntVar(0, 10, "t")
    _ = m.LinearSum(xs, []int{1, 2}, total) // t = x1 + 2*x2

    // Minimize t
    _, best, err := Optimize(m, total, true)
    if err != nil {
        fmt.Println("error:", err)
        return
    }
    fmt.Printf("best=%d\n", best)
    // Output:
    // best=3
}
```

Notes
- These examples are small building blocks — combine them with the rest of the HLAPI (Rows, Pairs, SolutionsCtx) to write concise, testable examples for tutorials.

### Collectors: Rows, Ints, Pairs, Triples

The HLAPI provides a small set of collectors that make it trivial to gather solution values into Go slices. Use these when you want to work with the results programmatically instead of printing raw Solutions.

- Ints(goal, x) -> []int
- Strings(goal, x) -> []string
- Rows(goal, vars...) -> [][]int (each row is a set of values for the provided variables)
- PairsInts/TriplesInts return typed tuples for convenience.

Example: collect rows as integers and compute an aggregate without depending on ordering.

```go
func Example_hlapi_collectors_rows() {
    // suppose `goal` yields assignments to x,y
    rows := Rows(goal, x, y)
    // compute an order-independent summary
    total := 0
    for _, r := range rows {
        total += r[0] + r[1]
    }
    fmt.Println("count", len(rows), "sum", total)
    // Output: (example depends on goal)
}
```

### SolutionsCtx and timeouts

When running search you often need to limit CPU or wall-clock time. Use the Context-aware helpers (`SolutionsCtx`, `Solve(ctx, ...)`, `SolveParallel(ctx, ...)`) and add timeouts or cancellation points.

Example pattern (collect all solutions with a timeout):

```go
ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
defer cancel()
solutions, err := SolutionsCtx(ctx, goal)
if err != nil {
    // handle timeout or cancellation
}
// process solutions
```

When examples depend on timing (parallel search, cancellation), prefer deterministic approaches in documentation: either cancel the context before the call to demonstrate the API (deterministic) or rely on configuration values that are not time sensitive.

### Optimization: OptimizeWithOptions

`Optimize` is convenient for basic use, but for more control prefer `OptimizeWithOptions` where you can pass search options and tie-breakers.

```go
opts := OptimizeOptions{MaxIterations: 1000, Timeout: 5 * time.Second}
_, best, err := OptimizeWithOptions(m, total, true, opts)
```

### IntVarValues and non-contiguous domains

`IntVarValues(values []int, name string)` is a small HLAPI helper to declare an IntVar whose domain is the explicit list `values` (useful for non-contiguous domains). It keeps examples compact and readable.

```go
// x ∈ {1,3,5}
v := m.IntVarValues([]int{1, 3, 5}, "x")
```

### Formatting terms and small helpers

`FormatTerm` and `FormatSolutions` are helpers used in examples to produce human-friendly strings for terms and solution maps. Prefer these in docs to keep examples short and readable.

```go
fmt.Println(FormatTerm(t))
fmt.Println(FormatSolutions(solutions[0]))
```

### Best practices for example-driven docs

- Keep each Example small and focused (show one concept).
- Make output deterministic: sort lines or aggregate values (count/sum) instead of relying on order.
- Avoid relying on unexported helpers — make examples self-contained so `go test` can run them in CI.
- Include an explicit `// Output:` block for every Example that prints, and verify with `go test -run Example ./...`.

### Next steps

- Expand this guide with short recipes: "Modeling schedules with Cumulative", "Using PLDB queries in tutorials", and "Tabling patterns". Each recipe should embed one or two verified Example snippets extracted from the source.

