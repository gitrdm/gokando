# gokanlogic

[![Version](https://img.shields.io/badge/version-1.1.0-blue.svg)](https://github.com/gitrdm/gokanlogic/releases)
[![Go Version](https://img.shields.io/badge/go-1.25%2B-00ADD8.svg)](https://golang.org/doc/devel/release.html)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

gokanlogic is a Go implementation of miniKanren with a finite-domain (FD) constraint solver and optional parallel execution. It aims to be practical for learning, experiments, and small projects. The core provides relational programming primitives; the FD layer helps with combinatorial search; and a thin high-level API (HLAPI) reduces boilerplate for common tasks. It's not an industrial solver, but it is hopefully useful in some domains that need logic and FD support.

## Installation

### Requirements
- **Go Version**: Go 1.25+ (per `go.mod`)
- **OS**: Linux, macOS, or Windows

### Install from Source
```bash
# Clone the repository
git clone https://github.com/gitrdm/gokanlogic.git
cd gokanlogic

# Install dependencies
go mod tidy

# Run tests to verify installation
go test ./...

# Run benchmarks (optional)
go test -bench=. ./...
```

### Quick Start (relational HLAPI)
```go
package main

import (
    "fmt"
    mk "github.com/gitrdm/gokanlogic/pkg/minikanren"
)

func main() {
    // Simple unification with Solutions + A()
    q := mk.Fresh("q")
    sols := mk.Solutions(mk.Eq(q, mk.A("hello")), q)
    fmt.Println(mk.FormatSolutions(sols)) // [q: "hello"]

    // Working with lists using L() and Appendo
    q2 := mk.Fresh("q")
    goal := mk.Appendo(mk.L(1, 2), mk.L(3), q2)
    sols2 := mk.Solutions(goal, q2)
    fmt.Println(mk.FormatSolutions(sols2)) // [q: (1 2 3)]
}
```

### Quick Start (FD HLAPI)
```go
// Create a small FD model with AllDifferent
m := mk.NewModel()
x := m.IntVar(1, 3, "x")
y := m.IntVar(1, 3, "y")
z := m.IntVar(1, 3, "z")
_ = m.AllDifferent(x, y, z)

solver := mk.NewSolver(m)
vals, _ := solver.Solve(context.Background(), 1) // first solution
_ = vals // values indexed by variable IDs
```

## Documentation

For more details, see the [`docs/`](docs/) directory:

### Core
- Relational miniKanren: logic variables, unification, goals, and streams
    - Common operators: `Fresh`, `Eq`, `Conj`, `Disj`, `Neq`, `Appendo`, etc.
    - Relational HLAPI helpers: `A`, `L`, `Solutions`, `Rows`, typed collectors (`Ints`, `Strings`, `Pairs*`), `FormatTerm`
    - Parallel helpers are available; examples show how to split work across goroutines

- Finite Domain (FD) solver: domains and constraints for CSP-style search
    - Variables: `Model.IntVar`, `IntVarValues`, `IntVars`
    - Constraints: `AllDifferent`, `LexLessEq`, `LinearSum`, `Among`, `Table`, `Regular`, `Cumulative`, `NoOverlap`
    - Solving: `NewSolver(model)`, `Solve`, and simple `Optimize` helpers

- Nominal Logic Programming: binders and alpha-equivalence aware reasoning
    - Binders: `Tie`/`Lambda`, freshness `Fresho`, alpha-equivalence `AlphaEqo`
    - Substitution: `Substo` (capture-avoiding)
    - Applications and reductions: `App`, `BetaReduceo`, `BetaNormalizeo`
    - Analysis and typing: `FreeNameso`, `TypeChecko` (simply-typed λ-calculus)
    - See API reference: `docs/api-reference/nominal.md`

### Examples and Guides
- **[Getting Started](docs/getting-started/)**: Short walkthroughs
- **[Examples](examples/)**: Working examples: apartment (FD), graph-coloring (relational), n-queens (relational) and n-queens-parallel-fd (FD + parallel), zebra, sudoku
- **[API Reference](docs/api-reference/)**: High-level API and package details

## What you get

- miniKanren core with common operators and a small HLAPI to cut boilerplate
- Optional parallel evaluation patterns for splitting independent work
- An FD solver suitable for many small/medium examples, with a clean model API
- A set of examples that show both relational and FD styles
- Nominal logic utilities for working with λ-terms, reduction, and simple typing

## Testing

```bash
# Run all tests
go test ./...

# Run with verbose output
go test ./... -v

# Run with race detection
go test -race ./...

# Run benchmarks
go test -bench=. ./...
```

## Contributing

- Please keep examples and docs runnable: `go test ./...`
- Prefer small, focused PRs with clear reasoning
- Document new public APIs and add tests where practical

## Versioning

- Latest tagged release: see Git tags (current: `v1.1.0`).
- Recommended bump: MINOR → `v1.1.0` (from `v1.0.1`). This release adds new, backwards-compatible public APIs for nominal logic programming:
    - `App`, `BetaReduceo`, `BetaNormalizeo`, `FreeNameso`, `TypeChecko`, plus docs and examples.
    - No breaking changes to existing packages; existing public APIs continue to work.
    - Docs updated: `docs/api-reference/nominal.md` and roadmap reflect completion of Phase 7.1.4.

## License

MIT License - see LICENSE file for details.
