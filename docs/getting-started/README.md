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

Version: 0.9.1

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
