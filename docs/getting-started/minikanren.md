# Getting Started with minikanren

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

Version: 1.0.0

This package offers a complete set of miniKanren operators with high-performance
concurrent execution capabilities, designed for production use.


## Overview

**Import Path:** `github.com/gitrdm/gokando/pkg/minikanren`

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

Version: 1.0.0

This package offers a complete set of miniKanren operators with high-performance
concurrent execution capabilities, designed for production use.


## Installation

### Install the package

```bash
go get github.com/gitrdm/gokando/pkg/minikanren
```

### Verify installation

Create a simple test file to verify the package works:

```go
package main

import (
    "fmt"
    "github.com/gitrdm/gokando/pkg/minikanren"
)

func main() {
    fmt.Println("minikanren package imported successfully!")
}
```

Run it:

```bash
go run main.go
```

## Quick Start

Here's a basic example to get you started with minikanren:

```go
package main

import (
    "fmt"
    "log"

    "github.com/gitrdm/gokando/pkg/minikanren"
)

func main() {
    // TODO: Add basic usage example
    fmt.Println("Hello from minikanren!")
}
```

## Key Features

### Types

- **AbsenceConstraint** - AbsenceConstraint implements the absence constraint (absento). It ensures that a specific term does not occur anywhere within another term's structure, providing structural constraint checking. This constraint performs recursive structural inspection to detect the presence of the forbidden term at any level of nesting.

- **AllDifferentConstraint** - AllDifferentConstraint is a custom version of the all-different constraint This demonstrates how built-in constraints can be reimplemented as custom constraints

- **Atom** - Atom represents an atomic value (symbol, number, string, etc.). Atoms are immutable and represent themselves.

- **BitSet** - Generic BitSet-backed Domain for FD variables. Values are 1-based indices.

- **Constraint** - Constraint represents a logical constraint that can be checked against variable bindings. Constraints are the core abstraction that enables order-independent constraint logic programming. Constraints must be thread-safe as they may be checked concurrently during parallel goal evaluation.

- **ConstraintEvent** - ConstraintEvent represents a notification about constraint-related activities. Used for coordinating between local stores and the global constraint bus.

- **ConstraintEventType** - ConstraintEventType categorizes different kinds of constraint events for efficient processing by the global constraint bus.

- **ConstraintResult** - ConstraintResult represents the outcome of evaluating a constraint. Constraints can be satisfied (no violation), violated (goal should fail), or pending (waiting for more variable bindings).

- **ConstraintStore** - ConstraintStore represents a collection of constraints and variable bindings. This interface abstracts over both local and global constraint storage.

- **ConstraintViolationError** - ConstraintViolationError represents an error caused by constraint violations. It provides detailed information about which constraint was violated and why.

- **CustomConstraint** - fd_custom.go: custom constraint interfaces for FDStore CustomConstraint represents a user-defined constraint that can propagate

- **DisequalityConstraint** - DisequalityConstraint implements the disequality constraint (≠). It ensures that two terms are not equal, providing order-independent constraint semantics for the Neq operation. The constraint tracks two terms and checks that they never become equal through unification. If both terms are variables, the constraint remains pending until at least one is bound to a concrete value.

- **FDChange** - Extend FDVar with offset links (placed here to avoid changing many other files) Note: we keep it unexported and simple; propagation logic in FDStore will consult these. We'll attach via a small map in FDStore to avoid changing serialized layout of FDVar across code paths. FDChange represents a single domain change for undo

- **FDStore** - - Offset arithmetic constraints for modeling relationships - Iterative backtracking with dom/deg heuristics - Context-aware cancellation and timeouts Typical usage: store := NewFDStoreWithDomain(maxValue) vars := store.MakeFDVars(n) // Add constraints... solutions, err := store.Solve(ctx, limit)

- **FDVar** - FDVar is a finite-domain variable

- **GlobalConstraintBus** - GlobalConstraintBus coordinates constraint checking across multiple local constraint stores. It handles cross-store constraints and provides a coordination point for complex constraint interactions. The bus is designed to minimize coordination overhead - most constraints should be local and not require global coordination.

- **GlobalConstraintBusPool** - GlobalConstraintBusPool manages a pool of reusable constraint buses

- **Goal** - Goal represents a constraint or a combination of constraints. Goals are functions that take a constraint store and return a stream of constraint stores representing all possible ways to satisfy the goal. Goals can be composed to build complex relational programs. The constraint store contains both variable bindings and active constraints, enabling order-independent constraint logic programming.

- **InequalityType** - fd_ineq.go: arithmetic inequality constraints for FDStore InequalityType represents the type of inequality constraint

- **LocalConstraintStore** - LocalConstraintStore interface defines the operations needed by the GlobalConstraintBus to coordinate with local stores.

- **LocalConstraintStoreImpl** - LocalConstraintStoreImpl provides a concrete implementation of LocalConstraintStore for managing constraints and variable bindings within a single goal context. The store maintains two separate collections: - Local constraints: Checked quickly without global coordination - Local bindings: Variable-to-term mappings for this context When constraints or bindings are added, the store first checks all local constraints for immediate violations, then coordinates with the global bus if necessary for cross-store constraints.

- **MembershipConstraint** - MembershipConstraint implements the membership constraint (membero). It ensures that an element is a member of a list, providing relational list membership checking that can work in both directions.

- **Pair** - Pair represents a cons cell (pair) in miniKanren. Pairs are used to build lists and other compound structures.

- **ParallelConfig** - ParallelConfig holds configuration for parallel goal execution.

- **ParallelExecutor** - ParallelExecutor manages parallel execution of miniKanren goals.

- **ParallelStream** - ParallelStream represents a stream that can be evaluated in parallel. It wraps the standard Stream with additional parallel capabilities.

- **SolverConfig** - SolverConfig holds configuration for the FD solver

- **SolverMonitor** - SolverMonitor provides monitoring capabilities for the FD solver

- **SolverStats** - SolverStats holds statistics about the FD solving process

- **Stream** - Stream represents a (potentially infinite) sequence of constraint stores. Streams are the core data structure for representing multiple solutions in miniKanren. Each constraint store contains variable bindings and active constraints representing a consistent logical state. This implementation uses channels for thread-safe concurrent access and supports parallel evaluation with proper constraint coordination.

- **Substitution** - Substitution represents a mapping from variables to terms. It's used to track bindings during unification and goal evaluation. The implementation is thread-safe and supports concurrent access.

- **SumConstraint** - Example custom constraint implementations SumConstraint enforces that the sum of variables equals a target value

- **Term** - Term represents any value in the miniKanren universe. Terms can be atoms, variables, compound structures, or any Go value. All Term implementations must be comparable and thread-safe.

- **TypeConstraint** - TypeConstraint implements type-based constraints (symbolo, numbero, etc.). It ensures that a term has a specific type, enabling type-safe relational programming patterns.

- **TypeConstraintKind** - TypeConstraintKind represents the different types that can be constrained.

- **ValueOrderingHeuristic** - ValueOrderingHeuristic defines strategies for ordering values within a domain

- **Var** - Var represents a logic variable in miniKanren. Variables can be bound to values through unification. Each variable has a unique identifier to distinguish it from others.

- **VariableOrderingHeuristic** - VariableOrderingHeuristic defines strategies for selecting the next variable to assign

- **VersionInfo** - VersionInfo provides detailed version information.

### Functions

- **GetVersion** - GetVersion returns the current version string.

- **ReturnPooledGlobalBus** - ReturnPooledGlobalBus returns a bus to the pool

## Usage Examples

For more detailed examples, see the [Examples](../examples/README.md) section.

## Next Steps

- [Full API Reference](../api-reference/minikanren.md) - Complete API documentation
- [Examples](../examples/README.md) - Working examples and tutorials
- [Best Practices](../guides/minikanren/best-practices.md) - Recommended patterns and usage

## Documentation Links

- [pkg.go.dev Documentation](https://pkg.go.dev/github.com/gitrdm/gokando/pkg/minikanren)
- [Source Code](https://github.com/gitrdm/gokando/tree/master/pkg/minikanren)
- [GitHub Issues](https://github.com/gitrdm/gokando/issues)
