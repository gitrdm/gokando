# SEND + MORE = MONEY (walkthrough)

This walkthrough shows how to model parts of the classic cryptarithm using reification and Count in the finite-domain (FD) solver, focusing on declarative modeling patterns rather than a full end-to-end solver.

Key ideas
- Shifted digit encoding to stay within positive FD domains
- Count to state “no leading zeros” declaratively
- Reification to bind meta-properties (e.g., M = 1)
- AllDifferent for distinct letters

Why shifted encoding?
FD domains are 1-indexed positive ranges [1..Max]. To represent base-10 digits 0..9, encode each digit d as FD value d+1 ∈ [1..10]. In this encoding:
- FD value 1 represents digit 0
- FD value 2 represents digit 1
- …
- FD value 10 represents digit 9

With this trick, we can use ValueEqualsReified and Count naturally.

Minimal declarative model sketch (Go)

```go
package main

import (
    "context"
    "fmt"

    "github.com/gitrdm/gokanlogic/pkg/minikanren"
)

func main() {
    // Digits 0..9 → FD values 1..10
    model := minikanren.NewModel(10)

    // Letter variables (encoded digits)
    S := model.NewVariable()
    E := model.NewVariable()
    N := model.NewVariable()
    D := model.NewVariable()
    M := model.NewVariable()
    O := model.NewVariable()
    R := model.NewVariable()
    Y := model.NewVariable()

    // All letters must be distinct
    _ = model.AddConstraint(minikanren.NewAllDifferent([]*minikanren.FDVariable{S, E, N, D, M, O, R, Y}))

    // 1) No leading zeros: S and M cannot be digit 0 (encoded as FD value 1)
    //    Count([S, M], target=1) must be 0 → encoded countVar = 1 (0+1)
    countVar := model.NewVariable()
    // countVar in [1..1] encodes actual count=0
    _ = countVar.SetDomain(minikanren.NewBitSetDomainFromValues(10, []int{1}))
    _, _ = minikanren.NewCount(model, []*minikanren.FDVariable{S, M}, 1, countVar)

    // 2) Reify M = digit 1 (common fact for this puzzle)
    //    "M is 1" means encoded M == 2; force boolean to true ({2})
    bM := model.NewVariable()
    _ = bM.SetDomain(minikanren.NewBitSetDomainFromValues(10, []int{2})) // {2} means true
    _, _ = minikanren.NewValueEqualsReified(M, 2, bM)

  // (Optional) Further column-wise arithmetic would use additional constraints.
  // Today, you can combine these FD constraints with a relational layer (Project)
  // to check column sums, or wait for a full linear-sum constraint in the library.
  // Note: a basic unweighted BoundsSum exists for integers, but SEND+MORE=MONEY
  // needs weighted sums and carry handling, which BoundsSum doesn't cover yet.

    solver := minikanren.NewSolver(model)
    sols, _ := solver.Solve(context.Background(), 1)
    fmt.Printf("solutions: %d\n", len(sols))

    // Inspect pruned domains after initial propagation
    mDom := solver.GetDomain(nil, M.ID())
    sDom := solver.GetDomain(nil, S.ID())
    fmt.Printf("M domain after reification: %s (expect {2})\n", mDom)
    fmt.Printf("S domain excludes 1 (zero): Has(1)=%v\n", sDom.Has(1))
}
```

What this achieves
- No leading zeros is expressed declaratively via Count, without manually removing 1 from S and M
- M=1 is expressed as a reified equality to a constant, then forcing the boolean to true; this prunes M to the singleton {2}
- AllDifferent captures the global “distinct letters” requirement

Where the rest fits
- Column arithmetic like D+E = Y + 10*C1 can be layered on using:
  - The relational layer (Project) to check sums, today
  - Future linear-sum constraints (Sum/Element) in the FD library for fully declarative column modeling

Tips
- Remember boolean encoding is {1:false, 2:true}
- Count uses encoded totals: a count of k is represented as k+1
- For digit problems, use the shifted encoding (d ↦ d+1) to keep everything in [1..Max]

Related API examples
- ExampleReifiedConstraint (reification basics)
- ExampleCount (Count with BoolSum)
```