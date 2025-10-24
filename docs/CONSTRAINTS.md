# Constraint System Guide

## Overview

GoKanren implements a **powerful constraint system** with **order-independent behavior**. Constraints can be placed before or after unification - the system automatically coordinates constraint checking regardless of goal ordering. This provides maximum flexibility while maintaining high performance.

## How Constraints Work

### Hybrid Architecture

GoKanren uses a hybrid constraint architecture for optimal performance:

1. **LocalConstraintStore**: Each goal execution maintains a local constraint store
2. **GlobalConstraintBus**: Coordinates constraints across multiple stores
3. **Automatic Synchronization**: Unification operations check all relevant constraints
4. **Thread-Safe Design**: Full concurrency support with minimal overhead

### Order-Independent Execution

```go
// Both patterns work identically:

// Pattern 1: Constraint then unification
minikanren.Run(1, func(q *minikanren.Var) minikanren.Goal {
    return minikanren.Conj(
        minikanren.Symbolo(q),                        // Constraint added to store
        minikanren.Eq(q, minikanren.NewAtom("test")), // Unification checks constraints
    )
})
// Result: ["test"]

// Pattern 2: Unification then constraint
minikanren.Run(1, func(q *minikanren.Var) minikanren.Goal {
    return minikanren.Conj(
        minikanren.Eq(q, minikanren.NewAtom("test")), // Unification succeeds
        minikanren.Symbolo(q),                        // Constraint validated
    )
})
// Result: ["test"]
```

### Automatic Constraint Validation

Constraints are automatically checked whenever a relevant unification occurs:

```go
// Constraint properly rejects invalid binding regardless of order
minikanren.Run(1, func(q *minikanren.Var) minikanren.Goal {
    return minikanren.Conj(
        minikanren.Symbolo(q),                    // q must be a symbol
        minikanren.Eq(q, minikanren.NewAtom(42)), // Attempt to bind q to number
    )
})
// Result: [] (no solutions - constraint violation detected)
```

## Constraint Categories

### Type Constraints

#### Symbolo(term)
Ensures term is a string atom.

```go
// Works with any goal ordering
minikanren.Run(3, func(q *minikanren.Var) minikanren.Goal {
    candidates := minikanren.List(
        minikanren.NewAtom("hello"),
        minikanren.NewAtom(42),
        minikanren.NewAtom("world"),
        minikanren.NewAtom(true),
    )
    
    return minikanren.Conj(
        minikanren.Symbolo(q),          // Type constraint
        minikanren.Membero(q, candidates), // Generate candidates
    )
})
// Result: ["hello", "world"]
```

#### Numbero(term)
Ensures term is a numeric atom.

```go
// Supports all Go numeric types
minikanren.Run(3, func(q *minikanren.Var) minikanren.Goal {
    candidates := minikanren.List(
        minikanren.NewAtom(42),      // int
        minikanren.NewAtom(3.14),    // float64
        minikanren.NewAtom("42"),    // string (will be rejected)
        minikanren.NewAtom(true),    // bool (will be rejected)
    )
    
    return minikanren.Conj(
        minikanren.Numbero(q),         // Type constraint first
        minikanren.Membero(q, candidates),
    )
})
// Result: [42, 3.14]
```

### Value Constraints

#### Neq(t1, t2)
Ensures two terms are not equal (disequality constraint).

```go
// Flexible ordering with disequality
minikanren.Run(5, func(q *minikanren.Var) minikanren.Goal {
    candidates := minikanren.List(
        minikanren.NewAtom("allowed"),
        minikanren.NewAtom("forbidden"), 
        minikanren.NewAtom("ok"),
        minikanren.NewAtom("bad"),
    )
    
    return minikanren.Conj(
        minikanren.Neq(q, minikanren.NewAtom("forbidden")), // Constraint first
        minikanren.Neq(q, minikanren.NewAtom("bad")),       // Multiple constraints
        minikanren.Membero(q, candidates),                   // Then generate
    )
})
// Result: ["allowed", "ok"]
```

#### Absento(absent, term)
Ensures a value doesn't appear anywhere in a structure.

```go
// Structural constraint with flexible ordering
minikanren.Run(2, func(q *minikanren.Var) minikanren.Goal {
    candidates := minikanren.List(
        minikanren.List(
            minikanren.NewAtom("good"),
            minikanren.NewAtom("ok"),
        ),
        minikanren.List(
            minikanren.NewAtom("bad"),    // Contains forbidden element
            minikanren.NewAtom("ok"),
        ),
        minikanren.List(
            minikanren.NewAtom("great"),
            minikanren.NewAtom("fine"),
        ),
    )
    
    return minikanren.Conj(
        minikanren.Absento(minikanren.NewAtom("bad"), q), // Structure constraint
        minikanren.Membero(q, candidates),                 // Generate candidates
    )
})
// Result: [["good", "ok"], ["great", "fine"]]
```

### List Constraints

#### Membero(elem, list)
Relational membership - can generate or test elements.

```go
list := minikanren.List(
    minikanren.NewAtom(1),
    minikanren.NewAtom(2), 
    minikanren.NewAtom(3),
)

// Generate mode: find all members
minikanren.Run(10, func(q *minikanren.Var) minikanren.Goal {
    return minikanren.Membero(q, list)
})
// Result: [1, 2, 3]

// Test mode: check membership combined with other constraints
minikanren.Run(10, func(q *minikanren.Var) minikanren.Goal {
    return minikanren.Conj(
        minikanren.Numbero(q),                      // Must be number
        minikanren.Membero(q, list),                // Must be in list
        minikanren.Neq(q, minikanren.NewAtom(2)),   // Must not be 2
    )
})
// Result: [1, 3]
```

### Control Constraints

#### Onceo(goal)
Succeeds at most once (cut operator).

```go
// Control solution multiplicity
minikanren.Run(10, func(q *minikanren.Var) minikanren.Goal {
    return minikanren.Onceo(minikanren.Disj(
        minikanren.Eq(q, minikanren.NewAtom("first")),
        minikanren.Eq(q, minikanren.NewAtom("second")),
        minikanren.Eq(q, minikanren.NewAtom("third")),
    ))
})
// Result: ["first"] (only first solution)
```

## Advanced Constraint Patterns

### Multiple Constraint Coordination

Combine multiple constraints with automatic coordination:

```go
minikanren.Run(10, func(q *minikanren.Var) minikanren.Goal {
    candidates := minikanren.List(
        minikanren.NewAtom("hello"),
        minikanren.NewAtom(42),
        minikanren.NewAtom("world"),
        minikanren.NewAtom("forbidden"),
        minikanren.NewAtom(3.14),
    )
    
    return minikanren.Conj(
        // All constraints can be placed first - order doesn't matter
        minikanren.Symbolo(q),                              // Must be string
        minikanren.Neq(q, minikanren.NewAtom("forbidden")), // Must not be "forbidden"
        minikanren.Absento(minikanren.NewAtom("x"), q),     // Must not contain "x"
        
        // Generate candidates
        minikanren.Membero(q, candidates),
    )
})
// Result: ["hello", "world"]
```

### Conditional Constraints with Project

Use `Project` to apply constraints based on computed values:

```go
minikanren.Run(5, func(q *minikanren.Var) minikanren.Goal {
    x := minikanren.Fresh("x")
    y := minikanren.Fresh("y")
    
    return minikanren.Conj(
        // Constraints can be applied in any order
        minikanren.Numbero(x),
        minikanren.Numbero(y),
        
        // Generate some candidates
        minikanren.Membero(x, minikanren.List(
            minikanren.NewAtom(5),
            minikanren.NewAtom(10),
            minikanren.NewAtom(15),
        )),
        minikanren.Membero(y, minikanren.List(
            minikanren.NewAtom(2),
            minikanren.NewAtom(3),
            minikanren.NewAtom(4),
        )),
        
        // Apply conditional logic
        minikanren.Project([]minikanren.Term{x, y}, func(values []minikanren.Term) minikanren.Goal {
            if atom1, ok := values[0].(*minikanren.Atom); ok {
                if atom2, ok := values[1].(*minikanren.Atom); ok {
                    if val1, ok := atom1.Value().(int); ok {
                        if val2, ok := atom2.Value().(int); ok {
                            sum := val1 + val2
                            // Apply constraint based on computation
                            return minikanren.Conj(
                                minikanren.Eq(q, minikanren.NewAtom(sum)),
                                minikanren.Numbero(q),
                            )
                        }
                    }
                }
            }
            return minikanren.Failure
        }),
    )
})
```

### Complex List Processing

Process lists with multiple structural constraints:

```go
// Find lists where first element is a symbol and "bad" is absent
minikanren.Run(3, func(q *minikanren.Var) minikanren.Goal {
    head := minikanren.Fresh("head")
    tail := minikanren.Fresh("tail")
    
    candidates := minikanren.List(
        minikanren.List(
            minikanren.NewAtom("good"),
            minikanren.NewAtom("elements"),
        ),
        minikanren.List(
            minikanren.NewAtom(42),      // First element not symbol
            minikanren.NewAtom("test"),
        ),
        minikanren.List(
            minikanren.NewAtom("has"),
            minikanren.NewAtom("bad"),   // Contains "bad"
        ),
        minikanren.List(
            minikanren.NewAtom("perfect"),
            minikanren.NewAtom("match"),
        ),
    )
    
    return minikanren.Conj(
        // Structure constraints (order independent)
        minikanren.Cons(head, tail, q),                     // q is [head|tail]
        minikanren.Symbolo(head),                           // First element is symbol
        minikanren.Absento(minikanren.NewAtom("bad"), q),   // "bad" not in list
        
        // Generate candidates
        minikanren.Membero(q, candidates),
    )
})
// Result: [["good", "elements"], ["perfect", "match"]]
```

## Performance Characteristics

### Constraint Overhead

The hybrid constraint system provides optimal performance characteristics:

- **Local Constraints**: Very low overhead, checked during unification
- **Global Coordination**: Minimal synchronization for cross-store constraints
- **Memory Efficient**: Local stores are lightweight and short-lived
- **Thread-Safe**: Lock-free operations for read-heavy workloads

### Optimization Tips

1. **Constraint Placement**: Place constraints anywhere convenient - the system handles coordination
2. **Multiple Constraints**: Combine multiple constraints freely without ordering concerns
3. **Parallel Execution**: Constraints work seamlessly with parallel goal execution
4. **Large Structures**: Structural constraints (like `Absento`) scale well with the hybrid architecture

### Performance Comparison

```go
// All these patterns have similar performance characteristics:

// Pattern 1: Traditional "unify then constrain"
minikanren.Conj(
    minikanren.Eq(q, candidate),
    minikanren.Symbolo(q),
    minikanren.Neq(q, forbidden),
)

// Pattern 2: "Constrain then unify"  
minikanren.Conj(
    minikanren.Symbolo(q),
    minikanren.Neq(q, forbidden),
    minikanren.Eq(q, candidate),
)

// Pattern 3: Mixed ordering
minikanren.Conj(
    minikanren.Symbolo(q),
    minikanren.Eq(q, candidate),
    minikanren.Neq(q, forbidden),
)
```

## Migration from Order-Dependent Systems

If migrating from an order-dependent miniKanren implementation:

### Key Benefits
1. **Simplified Logic**: No need to worry about constraint placement
2. **Better Composability**: Goals can be reordered freely  
3. **Easier Debugging**: Constraint violations are consistently detected
4. **Performance**: No performance penalty for natural constraint placement

### Updated Patterns

```go
// OLD pattern (order-dependent): Generate-and-test required
minikanren.Conj(
    minikanren.Membero(q, candidates),  // Must generate first
    minikanren.Symbolo(q),              // Then test
)

// NEW pattern (order-independent): Natural constraint expression
minikanren.Conj(
    minikanren.Symbolo(q),              // Express constraints naturally
    minikanren.Membero(q, candidates),  // Order doesn't matter
)
```

### What Changed
- **Constraint Timing**: Constraints now checked automatically during unification
- **Goal Ordering**: No longer matters for correctness
- **Error Detection**: More reliable constraint violation detection
- **Composability**: Goals can be combined in any order

### What Stayed the Same
- **API Compatibility**: All constraint function signatures unchanged
- **Semantics**: Constraint meanings and behaviors unchanged
- **Performance**: Similar or better performance characteristics

## Debugging Constraints

### Understanding Constraint Flow

Since constraints are order-independent, debugging focuses on constraint logic rather than ordering:

```go
// Debug by examining constraint combinations
minikanren.Run(1, func(q *minikanren.Var) minikanren.Goal {
    return minikanren.Conj(
        // Each constraint can be evaluated independently
        minikanren.Symbolo(q),                        // Type requirement
        minikanren.Neq(q, minikanren.NewAtom("bad")), // Value exclusion
        minikanren.Eq(q, minikanren.NewAtom("test")), // Binding
    )
    // Automatic coordination ensures all constraints are satisfied
})
```

### Common Debugging Patterns

1. **Test constraints individually**:
   ```go
   // Test each constraint in isolation
   minikanren.Run(1, func(q *minikanren.Var) minikanren.Goal {
       return minikanren.Conj(
           minikanren.Eq(q, testValue),
           constraintToTest(q),
       )
   })
   ```

2. **Use multiple constraints**:
   ```go
   // Combine constraints to narrow down issues
   minikanren.Run(10, func(q *minikanren.Var) minikanren.Goal {
       return minikanren.Conj(
           constraint1(q),
           constraint2(q),
           // Add constraints one by one to isolate issues
           generateCandidates(q),
       )
   })
   ```

3. **Check constraint coordination**:
   ```go
   // Verify constraints work together correctly
   result1 := minikanren.Run(10, func(q *minikanren.Var) minikanren.Goal {
       return minikanren.Conj(constraintA, unificationB, constraintC)
   })
   
   result2 := minikanren.Run(10, func(q *minikanren.Var) minikanren.Goal {
       return minikanren.Conj(constraintC, constraintA, unificationB)  
   })
   
   // Results should be identical regardless of ordering
   ```

The order-independent constraint system provides a robust foundation for logic programming with maximum flexibility and consistent behavior.