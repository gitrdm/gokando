# Getting Started

This guide will help you get up and running quickly with gokando.

## Installation

### Requirements

- Go 1.21 or later
- No external dependencies required

### Install via go get

```bash
go get github.com/gitrdm/gokando@latest
```

### Install specific version

```bash
go get github.com/gitrdm/gokando@v0.1.0
```

## Quick Start

Here's a simple example to get you started:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/gitrdm/gokando"
)

func main() {
    // Your code here
    fmt.Println("Hello from gokando!")
}
```

## Available Packages

gokando provides the following packages:

### [main](main.md)

This example shows how to use the core primitives to solve
simple relational programming problems.


**Quick Links:**

- [Getting Started](main.md) - Installation and getting started
- [API Reference](../api-reference/main.md) - Complete API documentation
- [Examples](../examples/README.md) - Working examples
- [Best Practices](../guides/main/best-practices.md) - Recommended patterns

### [main](main.md)

Package main solves the apartment floor puzzle using GoKando.

The puzzle: Baker, Cooper, Fletcher, Miller, and Smith live on different
floors of an apartment house that contains only five floors.

Constraints:
  - Baker does not live on the top floor.
  - Cooper does not live on the bottom floor.
  - Fletcher does not live on either the top or the bottom floor.
  - Miller lives on a higher floor than does Cooper.
  - Smith does not live on a floor adjacent to Fletcher's.
  - Fletcher does not live on a floor adjacent to Cooper's.

Question: Where does everyone live?


**Quick Links:**

- [Getting Started](main.md) - Installation and getting started
- [API Reference](../api-reference/main.md) - Complete API documentation
- [Examples](../examples/README.md) - Working examples
- [Best Practices](../guides/main/best-practices.md) - Recommended patterns

### [main](main.md)

Graph Coloring Problem: Color the vertices of a graph such that no two
adjacent vertices share the same color, using the minimum number of colors.

This example demonstrates:
- Pure relational constraints with Neq
- Parallel search with ParallelDisj and ParallelRun
- Performance comparison between sequential and parallel execution

The example uses a map of Australia with 7 regions and demonstrates
the classic 3-coloring problem.


**Quick Links:**

- [Getting Started](main.md) - Installation and getting started
- [API Reference](../api-reference/main.md) - Complete API documentation
- [Examples](../examples/README.md) - Working examples
- [Best Practices](../guides/main/best-practices.md) - Recommended patterns

### [main](main.md)

Package main solves the N-Queens puzzle using GoKando.

The N-Queens puzzle: Place N queens on an N×N chessboard such that no two queens
attack each other. Queens can attack any piece on the same row, column, or diagonal.

This implementation uses Project to verify the constraints efficiently for small N (4-8).
For larger boards, a more sophisticated constraint propagation approach would be needed.


**Quick Links:**

- [Getting Started](main.md) - Installation and getting started
- [API Reference](../api-reference/main.md) - Complete API documentation
- [Examples](../examples/README.md) - Working examples
- [Best Practices](../guides/main/best-practices.md) - Recommended patterns

### [main](main.md)

Package main solves the Twelve Statements puzzle using GoKando.

The puzzle: Given twelve statements about themselves, determine which are true.

 1. This is a numbered list of twelve statements.
 2. Exactly 3 of the last 6 statements are true.
 3. Exactly 2 of the even-numbered statements are true.
 4. If statement 5 is true, then statements 6 and 7 are both true.
 5. The 3 preceding statements are all false.
 6. Exactly 4 of the odd-numbered statements are true.
 7. Either statement 2 or 3 is true, but not both.
 8. If statement 7 is true, then 5 and 6 are both true.
 9. Exactly 3 of the first 6 statements are true.

10. The next two statements are both true.
11. Exactly 1 of statements 7, 8 and 9 are true.
12. Exactly 4 of the preceding statements are true.


**Quick Links:**

- [Getting Started](main.md) - Installation and getting started
- [API Reference](../api-reference/main.md) - Complete API documentation
- [Examples](../examples/README.md) - Working examples
- [Best Practices](../guides/main/best-practices.md) - Recommended patterns

### [main](main.md)

Package main solves the famous Zebra puzzle (Einstein's Riddle) using GoKando.

The Zebra puzzle is a logic puzzle with the following constraints:
  - There are five houses.
  - The English man lives in the red house.
  - The Swede has a dog.
  - The Dane drinks tea.
  - The green house is immediately to the left of the white house.
  - They drink coffee in the green house.
  - The man who smokes Pall Mall has a bird.
  - In the yellow house they smoke Dunhill.
  - In the middle house they drink milk.
  - The Norwegian lives in the first house.
  - The Blend-smoker lives in the house next to the house with a cat.
  - In a house next to the house with a horse, they smoke Dunhill.
  - The man who smokes Blue Master drinks beer.
  - The German smokes Prince.
  - The Norwegian lives next to the blue house.
  - They drink water in a house next to the house where they smoke Blend.

Question: Who owns the zebra?


**Quick Links:**

- [Getting Started](main.md) - Installation and getting started
- [API Reference](../api-reference/main.md) - Complete API documentation
- [Examples](../examples/README.md) - Working examples
- [Best Practices](../guides/main/best-practices.md) - Recommended patterns

### [parallel](parallel.md)

Package parallel provides advanced parallel execution capabilities
for miniKanren goals. This package contains internal utilities
for managing concurrent goal evaluation with proper resource
management and backpressure control.


**Quick Links:**

- [Getting Started](parallel.md) - Installation and getting started
- [API Reference](../api-reference/parallel.md) - Complete API documentation
- [Examples](../examples/README.md) - Working examples
- [Best Practices](../guides/parallel/best-practices.md) - Recommended patterns

### [minikanren](minikanren.md)

Package minikanren provides constraint system infrastructure for order-independent
constraint logic programming. This file defines the core interfaces and types
for managing constraints in a hybrid local/global architecture.

The constraint system uses a two-tier approach:
  - Local constraints: managed within individual goal contexts for fast checking
  - Global constraints: coordinated across contexts when constraints span multiple stores

This design provides order-independent constraint semantics while maintaining
high performance for the common case of locally-scoped constraints.

Package minikanren provides concrete implementations of constraints
for the hybrid constraint system. These constraints implement the
Constraint interface and provide the core constraint logic for
disequality, absence, type checking, and other relational operations.

Each constraint implementation follows the same pattern:
  - Efficient local checking when all variables are bound
  - Graceful handling of unbound variables (returns ConstraintPending)
  - Thread-safe operations for concurrent constraint checking
  - Proper variable dependency tracking for optimization

The constraint implementations are designed to be:
  - Fast: Optimized for the common case of local constraint checking
  - Safe: Thread-safe and defensive against malformed input
  - Debuggable: Comprehensive error messages and string representations

Package minikanren provides a thread-safe, parallel implementation of miniKanren
in Go. This implementation follows the core principles of relational programming
while leveraging Go's concurrency primitives for parallel execution.

miniKanren is a domain-specific language for constraint logic programming.
It provides a minimal set of operators for building relational programs:
  - Unification (==): Constrains two terms to be equal
  - Fresh variables: Introduces new logic variables
  - Disjunction (conde): Represents choice points
  - Conjunction: Combines goals that must all succeed
  - Run: Executes a goal and returns solutions

This implementation is designed for production use with:
  - Thread-safe operations using sync package primitives
  - Parallel goal evaluation using goroutines and channels
  - Type-safe interfaces leveraging Go's type system
  - Comprehensive error handling and resource management

Package minikanren provides the LocalConstraintStore implementation for
managing constraints and variable bindings within individual goal contexts.

The LocalConstraintStore is the core component of the hybrid constraint system,
providing fast local constraint checking while coordinating with the global
constraint bus when necessary for cross-store constraints.

Key design principles:
  - Fast path: Local constraint checking without coordination overhead
  - Slow path: Global coordination only when cross-store constraints are involved
  - Thread-safe: Safe for concurrent access and parallel goal evaluation
  - Efficient cloning: Optimized for parallel execution where stores are frequently copied

Package minikanren provides a thread-safe parallel implementation of miniKanren in Go.

Version: 0.11.0

This package offers a complete set of miniKanren operators with high-performance
concurrent execution capabilities, designed for production use.


**Quick Links:**

- [Getting Started](minikanren.md) - Installation and getting started
- [API Reference](../api-reference/minikanren.md) - Complete API documentation
- [Examples](../examples/README.md) - Working examples
- [Best Practices](../guides/minikanren/best-practices.md) - Recommended patterns

## Next Steps

- [API Reference](../api-reference/README.md) - Complete API documentation
- [Examples](../examples/README.md) - Working examples and tutorials
- [Guides](../guides/README.md) - In-depth guides and best practices
- [GitHub Repository](https://github.com/gitrdm/gokando)

## Need Help?

- [GitHub Issues](https://github.com/gitrdm/gokando/issues)
- [GitHub Discussions](https://github.com/gitrdm/gokando/discussions)
- [FAQ](../guides/faq.md)
