# GoKando - Thread-Safe Parallel miniKanren in Go

[![Version](https://img.shields.io/badge/version-0.10.0-blue.svg)](https://github.com/gitrdm/gokando/releases)
[![Go Version](https://img.shields.io/badge/go-1.18%2B-00ADD8.svg)](https://golang.org/doc/devel/release.html)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

GoKando is a production-quality implementation of miniKanren in Go, designed with thread-safety and parallel execution as first-class concerns. This implementation provides a complete set of miniKanren operators with high-performance concurrent execution and an integrated finite domain constraint solver.

## Installation

### Requirements
- **Go Version**: Go 1.18+ (uses generics and modern concurrency patterns)
- **OS**: Any platform supporting Go (Linux, macOS, Windows)

### Install from Source
```bash
# Clone the repository
git clone https://github.com/gitrdm/gokando.git
cd gokando

# Install dependencies
go mod tidy

# Run tests to verify installation
go test ./...

# Run benchmarks (optional)
go test -bench=. ./...
```

### Quick Start
```go
package main

import (
    "fmt"
    "github.com/gitrdm/gokando/pkg/minikanren"
)

func main() {
    // Simple unification
    results := minikanren.Run(1, func(q *minikanren.Var) minikanren.Goal {
        return minikanren.Eq(q, minikanren.NewAtom("hello"))
    })
    fmt.Println(results) // [hello]

    // List operations
    results = minikanren.Run(1, func(q *minikanren.Var) minikanren.Goal {
        return minikanren.Appendo(
            minikanren.List(minikanren.NewAtom(1), minikanren.NewAtom(2)),
            minikanren.List(minikanren.NewAtom(3)),
            q,
        )
    })
    fmt.Println(results) // [(1 2 3)]
}
```

## Documentation

For comprehensive documentation, see the [`docs/`](docs/) directory:

### Core miniKanren
- **[Core Concepts](docs/minikanren/core.md)**: Logic variables, unification, goals, streams, and constraint system
  - **Complete operator set**: `Fresh`, `Eq`, `Conj`, `Disj`, `Run`, `RunStar`
  - **Order-independent constraints**: Type constraints (`Symbolo`, `Numbero`), disequality (`Neq`), absence (`Absento`)
  - **List operations**: `Appendo`, `Caro`, `Cdru`, `Conso`, `Nullo`, `Pairo`, `Membero`
  - **Advanced features**: Committed choice (`Conda`, `Condu`), projection, cut operators (`Onceo`)
  - **Parallel execution**: Worker pools, backpressure control, rate limiting, context cancellation

- **[Finite Domain Solver](docs/minikanren/finite-domains.md)**: Complete FD constraint solver with domain operations, heuristics, and monitoring
  - **Domain system**: BitSet-based domains, assignment, removal, intersection, union, complement
  - **Constraint propagation**: AC-3 algorithm, Regin filtering for all-different constraints
  - **Arithmetic constraints**: Offset links for modeling relationships (e.g., N-Queens diagonals)
  - **Inequality constraints**: `<`, `<=`, `>`, `>=`, `!=` operators with propagation
  - **Search heuristics**: Dom/Deg, Domain, Degree, Lexicographic, Random ordering
  - **Custom constraints**: Extensible framework with `SumConstraint`, `AllDifferentConstraint`
  - **Monitoring**: Comprehensive statistics, performance tracking, domain reduction metrics
  - **Integration**: Seamless FD goals (`FDAllDifferentGoal`, `FDQueensGoal`, `FDInequalityGoal`)

### Examples and Guides
- **[Getting Started](docs/getting-started/)**: Tutorials and basic usage examples
- **[Examples](examples/)**: Complete working examples including Zebra Puzzle and Apartment Floor Puzzle
- **[API Reference](docs/api-reference/)**: Detailed API documentation

## Key Features

- **Complete miniKanren**: All standard operators plus advanced constraints
- **Thread-Safe**: Safe for concurrent use across goroutines
- **Parallel Execution**: Configurable worker pools with backpressure control
- **Finite Domain Solver**: Integrated CSP solver with advanced heuristics
- **Type-Safe**: Leverages Go's type system for safe relational programming
- **Production Ready**: Extensive testing and comprehensive documentation

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

1. Ensure all tests pass: `go test ./...`
2. Run benchmarks to check performance: `go test -bench=. ./...`
3. Follow Go conventions and document new APIs
4. Add tests for new functionality

## License

MIT License - see LICENSE file for details.
