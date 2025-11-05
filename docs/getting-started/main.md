# Quick Start Guide

Get started with gokanlogic in minutes. This guide covers installation and basic usage for both relational (miniKanren) and constraint solving (FD) approaches.

## Installation

```bash
go get github.com/gitrdm/gokanlogic
```

**Requirements:**
- Go 1.23 or later
- No external dependencies

## Your First Program

### Relational Programming (miniKanren)

Solve logic puzzles using relational reasoning:

```go
package main

import (
    "fmt"
    . "github.com/gitrdm/gokanlogic/pkg/minikanren"
)

func main() {
    // Find X where X + 2 = 5
    results := Run(1, func(q *Var) Goal {
        return Eq(q, A(3))
    })
    
    fmt.Println("Result:", results[0])
}
```

### Constraint Solving (FD)

Solve numeric puzzles with finite domain constraints:

```go
package main

import (
    "context"
    "fmt"
    "github.com/gitrdm/gokanlogic/pkg/minikanren"
)

func main() {
    // Solve: X + Y = 10, both in range 1-9
    m := minikanren.NewModel()
    
    x := m.IntVar(1, 9, "X")
    y := m.IntVar(1, 9, "Y")
    sum := m.IntVar(10, 10, "sum")
    
    m.LinearSum([]*minikanren.FDVariable{x, y}, []int{1, 1}, sum)
    
    solver := minikanren.NewSolver(m)
    solutions, _ := solver.Solve(context.Background(), 1)
    
    fmt.Printf("X=%d, Y=%d\n", solutions[0][x.ID()], solutions[0][y.ID()])
}
```

## Next Steps

- [miniKanren Guide](minikanren.md) - Relational programming
- [Parallel Search](parallel.md) - Multi-core solving
- [Examples](../examples/README.md) - Working code samples
- [API Reference](../api-reference/minikanren.md) - Complete docs
