# miniKanren Core

miniKanren is a relational programming language embedded in Go, providing a powerful framework for logic programming and constraint solving. This implementation emphasizes thread-safety, parallel execution, and seamless integration with Go's type system.

## Overview

miniKanren enables relational programming where you describe relationships between entities rather than specifying step-by-step procedures. The system automatically finds values that satisfy all given constraints.

Key features:
- **Logic variables** that can be bound to values during execution
- **Unification** for pattern matching and value binding
- **Goals** as first-class entities representing constraints
- **Streams** for lazy evaluation of solution spaces
- **Thread-safe** concurrent execution
- **Parallel evaluation** with configurable worker pools

## Tabling & WFS (SLG)

For recursive relations and negation, gokanlogic includes a production SLG tabling engine implementing Well‑Founded Semantics (WFS):

- What it is and when to use it: see `docs/minikanren/tabling.md`.
- How to call it ergonomically: use the wrappers in `pkg/minikanren/slg_wrappers.go` (examples in `pkg/minikanren/slg_wrappers_example_test.go`).
- Core implementation: `pkg/minikanren/slg_engine.go`, `pkg/minikanren/slg_wfs.go`, and `pkg/minikanren/tabling.go`.

This documentation avoids duplicating code; the source files above are the authority.

## Core Concepts

### Terms

Everything in miniKanren is a `Term`. Terms can be:

```go
// Atoms
atom := minikanren.NewAtom("hello")
number := minikanren.NewAtom(42)

// Variables
x := minikanren.Fresh("x")     // Named variable
y := minikanren.Fresh("")      // Anonymous variable

// Pairs (lists)
list := minikanren.List(
    minikanren.NewAtom(1),
    minikanren.NewAtom(2),
    minikanren.NewAtom(3),
)
// Equivalent to: (1 . (2 . (3 . nil)))
```

### Substitutions

A substitution maps variables to their bound values. The substitution system tracks variable bindings and supports walking (following chains of bindings) and deep walking (recursively resolving all variables).

```go
sub := minikanren.NewSubstitution()

// Bind variable to value
sub.Bind(x, minikanren.NewAtom("value"))

// Walk finds the final value of a variable
finalValue := sub.Walk(x) // Returns the atom "value"

// DeepWalk resolves all variables in a term
complexTerm := minikanren.NewPair(x, y)
resolved := sub.DeepWalk(complexTerm) // Resolves any variables in the pair
```

### Streams

Streams provide lazy evaluation of solution spaces. They can be:
- **Empty**: No solutions
- **Singleton**: One solution
- **Multiple**: Many solutions (possibly infinite)

```go
// Create a stream
stream := minikanren.NewStream()

// Put solutions into the stream
go func() {
    defer stream.Close()
    stream.Put(store1)
    stream.Put(store2)
    // ... more solutions
}()

// Take solutions from the stream
solutions, hasMore := stream.Take(10) // Get up to 10 solutions
```

## Goals and Execution

### Goals

Goals are functions that take a constraint store and return a stream of constraint stores representing solutions. Goals represent constraints that must be satisfied.

```go
// A goal function signature
type Goal func(context.Context, ConstraintStore) *Stream

// Simple goal that succeeds
successGoal := func(ctx context.Context, store ConstraintStore) *Stream {
    stream := NewStream()
    go func() {
        defer stream.Close()
        stream.Put(store) // Return the store unchanged
    }()
    return stream
}

// Goal that fails
failureGoal := func(ctx context.Context, store ConstraintStore) *Stream {
    stream := NewStream()
    stream.Close() // Close without putting anything
    return stream
}
```

### Basic Goals

#### Unification (`Eq`)

The fundamental operation that makes two terms equal:

```go
// Bind variable to value
goal := minikanren.Eq(x, minikanren.NewAtom("hello"))

// Unify two variables
goal := minikanren.Eq(x, y)

// Unify with complex terms
goal := minikanren.Eq(
    minikanren.List(x, y),
    minikanren.List(minikanren.NewAtom(1), minikanren.NewAtom(2)),
)
```

#### Conjunction (`Conj`)

All goals must succeed:

```go
goal := minikanren.Conj(
    minikanren.Eq(x, minikanren.NewAtom(1)),
    minikanren.Eq(y, minikanren.NewAtom(2)),
    minikanren.Eq(z, minikanren.NewAtom(3)),
)
```

#### Disjunction (`Disj`)

At least one goal must succeed:
goal := minikanren.Disj(
    minikanren.Eq(x, minikanren.NewAtom(1)),
    minikanren.Eq(x, minikanren.NewAtom(2)),
    minikanren.Eq(x, minikanren.NewAtom(3)),
)
```

### Execution

#### Running Goals

Execute goals to find solutions:

```go
// Find up to 5 solutions
results := minikanren.Run(5, func(q *minikanren.Var) minikanren.Goal {
    return minikanren.Eq(q, minikanren.NewAtom("solution"))
})

// Find all solutions (use carefully - may be infinite)
results := minikanren.RunStar(func(q *minikanren.Var) minikanren.Goal {
    return myGoal(q)
})

// With context for cancellation/timeouts
ctx, cancel := context.WithTimeout(context.Background(), time.Second)
defer cancel()
results := minikanren.RunWithContext(ctx, 10, goalFunc)
```

#### Fresh Variables

Create fresh logic variables for each execution:

```go
results := minikanren.Run(1, func(q *minikanren.Var) minikanren.Goal {
    x := minikanren.Fresh("x")
    y := minikanren.Fresh("y")
    return minikanren.Conj(
        minikanren.Eq(x, minikanren.NewAtom(1)),
        minikanren.Eq(y, minikanren.NewAtom(2)),
        minikanren.Eq(q, minikanren.List(x, y)),
    )
})
// Result: [(1 2)]
```

## Constraint System

### Order-Independent Constraints

Unlike traditional miniKanren implementations, this system provides order-independent constraints. Constraints work regardless of when they're added relative to unification:

```go
// ✅ Both orders work identically

// Constraint before unification
goal := minikanren.Conj(
    minikanren.Numbero(q),                       // Add constraint
    minikanren.Eq(q, minikanren.NewAtom(42)),    // Unification checks constraint
)

// Unification before constraint
goal := minikanren.Conj(
    minikanren.Eq(q, minikanren.NewAtom(42)),    // Bind first
    minikanren.Numbero(q),                       // Check constraint after
)
```

### Built-in Constraints

#### Type Constraints

```go
// Term must be a symbol/string
minikanren.Symbolo(term)

// Term must be a number
minikanren.Numbero(term)
```

#### Disequality

```go
// Terms must not be equal
minikanren.Neq(term1, term2)
```

#### Absence

```go
// absent must not appear anywhere in term
minikanren.Absento(absent, term)
```

#### List Operations

```go
// Check if term is the empty list
minikanren.Nullo(term)

// Check if term is a non-empty list
minikanren.Pairo(term)

// Extract head of list
minikanren.Car(list, head)

// Extract tail of list
minikanren.Cdr(list, tail)

// Construct list from head and tail
minikanren.Cons(head, tail, list)
```

#### Membership

```go
// elem appears in list
minikanren.Membero(elem, list)
```

### Committed Choice

#### Conditional (`Conda`)

If-then-else with cut semantics:

```go
goal := minikanren.Conda(
    // If x is 1, then y must be 2
    minikanren.Conj(
        minikanren.Eq(x, minikanren.NewAtom(1)),
        minikanren.Eq(y, minikanren.NewAtom(2)),
    ),
    // Else x must be 2 and y can be anything
    minikanren.Conj(
        minikanren.Eq(x, minikanren.NewAtom(2)),
        minikanren.Symbolo(y),
    ),
)
```

#### Unique Conditional (`Condu`)

Like `Conda` but ensures uniqueness:

```go
goal := minikanren.Condu(
    minikanren.Conj(
        minikanren.Eq(x, minikanren.NewAtom(1)),
        minikanren.Eq(y, minikanren.NewAtom(2)),
    ),
)
```

### Cut Operations

```go
// Succeed at most once
minikanren.Onceo(goal)
```

## Advanced Features

### Projection

Project variables to compute with their values:

```go
goal := minikanren.Project([]*minikanren.Var{x, y}, func() minikanren.Goal {
    // x and y are now bound to their values
    xVal := x // Access the actual value
    yVal := y // Access the actual value

    // Compute something based on the values
    sum := xVal + yVal
    return minikanren.Eq(z, minikanren.NewAtom(sum))
})
```

### List Relations

#### Append

The classic append relation:

```go
// appendo(l1, l2, l3) means l1 + l2 = l3
goal := minikanren.Appendo(
    minikanren.List(minikanren.NewAtom(1), minikanren.NewAtom(2)),
    minikanren.List(minikanren.NewAtom(3)),
    result, // Will be bound to (1 2 3)
)
```

#### Custom List Relations

Build more complex list operations using the primitives:

```go
// Reverse relation
func reverseo(l, r Term) Goal {
    return minikanren.Disj(
        // Base case: empty list reverses to empty list
        minikanren.Conj(
            minikanren.Nullo(l),
            minikanren.Eq(r, minikanren.NewAtom(nil)),
        ),
        // Recursive case
        func(ctx context.Context, store ConstraintStore) *Stream {
            a := Fresh("a")
            d := Fresh("d")
            res := Fresh("res")
            return minikanren.Conj(
                minikanren.Pairo(l),
                minikanren.Car(l, a),
                minikanren.Cdr(l, d),
                reverseo(d, res),
                minikanren.Appendo(res, minikanren.List(a), r),
            )(ctx, store)
        },
    )
}
```

## Parallel Execution

### Parallel Disjunction

Execute disjunctive branches in parallel:

```go
// Sequential (default)
goal := minikanren.Disj(goal1, goal2, goal3)

// Parallel execution
executor := minikanren.NewParallelExecutor(minikanren.DefaultParallelConfig())
goal := executor.ParallelDisj(goal1, goal2, goal3)
```

### Parallel Stream Processing

Process streams in parallel:

```go
// Create parallel stream
ps := minikanren.NewParallelStream(ctx, executor)

// Map operation in parallel
resultStream := ps.ParallelMap(func(store ConstraintStore) ConstraintStore {
    // Process each solution
    return processSolution(store)
})

// Filter in parallel
filteredStream := ps.ParallelFilter(func(store ConstraintStore) bool {
    // Return true to keep solution
    return isValidSolution(store)
})

// Collect results
results := ps.Collect()
```

### Configuration

Tune parallel execution:

```go
config := &minikanren.ParallelConfig{
    MaxWorkers:         8,                // Number of worker goroutines
    MaxQueueSize:       100,              // Maximum pending tasks
    EnableBackpressure: true,             // Prevent memory exhaustion
    RateLimit:          50,               // Operations per second (0 = unlimited)
}

executor := minikanren.NewParallelExecutor(config)
```

### Parallel Run Functions

Execute goals in parallel:

```go
// Simple parallel execution
results := minikanren.ParallelRun(10, goalFunc)

// With custom configuration
results := minikanren.ParallelRunWithConfig(10, goalFunc, config)

// With context
results := minikanren.ParallelRunWithContext(ctx, 10, goalFunc, config)
```

## Constraint Stores

### Local Constraint Store

The primary constraint store implementation:

```go
// Create with global bus for efficiency
store := minikanren.NewLocalConstraintStore(minikanren.GetDefaultGlobalBus())

// Add bindings
if err := store.AddBinding(varID, value); err != nil {
    // Handle constraint violation
}

// Get substitution
sub := store.GetSubstitution()

// Clone for backtracking
newStore := store.Clone()
```

### Global Constraint Bus

Manages constraint propagation across stores:

```go
// Get default bus (shared across the application)
bus := minikanren.GetDefaultGlobalBus()

// Create pooled bus for isolation
bus := minikanren.GetPooledGlobalBus()
defer minikanren.ReturnPooledGlobalBus(bus)
```

## Error Handling and Context

### Context Support

All operations support context for cancellation:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

results := minikanren.RunWithContext(ctx, 100, func(q *Var) Goal {
    // Long-running goal that can be cancelled
    return complexGoal(q)
})
```

### Error Types

Goals can fail in various ways:

```go
// Constraint violation
if err := store.AddBinding(varID, value); err != nil {
    // Handle constraint failure
}

// Context cancellation
select {
case <-ctx.Done():
    // Handle cancellation
default:
    // Continue
}
```

## Examples

### Simple Queries

```go
// What is X?
results := minikanren.Run(1, func(q *minikanren.Var) minikanren.Goal {
    return minikanren.Eq(q, minikanren.NewAtom("hello"))
})
// Result: [hello]

// What values can X take?
results := minikanren.Run(3, func(q *minikanren.Var) minikanren.Goal {
    return minikanren.Disj(
        minikanren.Eq(q, minikanren.NewAtom(1)),
        minikanren.Eq(q, minikanren.NewAtom(2)),
        minikanren.Eq(q, minikanren.NewAtom(3)),
    )
})
// Result: [1, 2, 3]
```

### List Processing

```go
// Find lists that append to (1 2 3 4)
results := minikanren.Run(5, func(q *minikanren.Var) minikanren.Goal {
    a := minikanren.Fresh("a")
    b := minikanren.Fresh("b")
    return minikanren.Conj(
        minikanren.Appendo(a, b, q),
        minikanren.Eq(q, minikanren.List(
            minikanren.NewAtom(1),
            minikanren.NewAtom(2),
            minikanren.NewAtom(3),
            minikanren.NewAtom(4),
        )),
    )
})
// Results: various splits like ([], [1,2,3,4]), ([1], [2,3,4]), etc.
```

### Constraint Satisfaction

```go
// Find numbers X, Y such that X + Y = 10 and X < Y
results := minikanren.Run(5, func(q *minikanren.Var) minikanren.Goal {
    x := minikanren.Fresh("x")
    y := minikanren.Fresh("y")
    return minikanren.Conj(
        minikanren.Numbero(x),
        minikanren.Numbero(y),
        minikanren.Eq(q, minikanren.List(x, y)),
        // X + Y = 10 would require arithmetic constraints
        // For now, just enumerate possibilities
        minikanren.Disj(
            minikanren.Conj(minikanren.Eq(x, minikanren.NewAtom(3)), minikanren.Eq(y, minikanren.NewAtom(7))),
            minikanren.Conj(minikanren.Eq(x, minikanren.NewAtom(4)), minikanren.Eq(y, minikanren.NewAtom(6))),
            minikanren.Conj(minikanren.Eq(x, minikanren.NewAtom(2)), minikanren.Eq(y, minikanren.NewAtom(8))),
        ),
    )
})
```

### Parallel Search

```go
// Search large solution space in parallel
config := &minikanren.ParallelConfig{
    MaxWorkers: 8,
    MaxQueueSize: 1000,
    EnableBackpressure: true,
}

results := minikanren.ParallelRunWithConfig(100, func(q *minikanren.Var) minikanren.Goal {
    // Complex goal with many independent branches
    return minikanren.Disj(
        complexBranch1(q),
        complexBranch2(q),
        complexBranch3(q),
        // ... many more branches
    )
}, config)
```

This miniKanren implementation provides a powerful, thread-safe foundation for relational programming in Go, with seamless integration of constraint solving and parallel execution capabilities.</content>
<parameter name="filePath">/home/rdmerrio/gits/gokanlogic/docs/minikanren/core.md