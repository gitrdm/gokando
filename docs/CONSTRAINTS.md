# Constraint System Guide

## Overview

GoKanren implements a **simplified constraint system** that provides all standard miniKanren constraint operators but with **order-dependent behavior**. This design choice prioritizes performance and simplicity over the full constraint propagation found in some other miniKanren implementations.

## How Constraints Work

### Execution Model

Constraints in GoKanren are checked **when the constraint goal executes**, not continuously after each unification. This means:

1. **Immediate Check**: If all terms are concrete (non-variables), the constraint is checked immediately
2. **Deferred Check**: If any term is a variable, the constraint assumes it will be satisfied
3. **No Propagation**: Constraints don't automatically re-check after later unifications

### Order-Dependent Behavior

```go
// These two examples behave differently:

// Example 1: Constraint after unification (Recommended)
minikanren.Conj(
    minikanren.Eq(q, minikanren.NewAtom(42)),     // Bind q to 42
    minikanren.Symbolo(q),                        // Check: is 42 a symbol? NO → fails
)
// Result: No solutions (correct - 42 is not a symbol)

// Example 2: Constraint before unification  
minikanren.Conj(
    minikanren.Symbolo(q),                        // Check: is q a symbol? q is unbound → defers
    minikanren.Eq(q, minikanren.NewAtom(42)),     // Bind q to 42
)
// Result: One solution: 42 (incorrect - constraint was deferred)
```

## Best Practices

### 1. Constraint After Unification

**Always place constraints after the unification that binds the variables they check:**

```go
// ✅ Correct Pattern
minikanren.Run(10, func(q *minikanren.Var) minikanren.Goal {
    return minikanren.Conj(
        // First: establish bindings
        minikanren.Eq(q, candidate),
        
        // Then: check constraints  
        minikanren.Symbolo(q),
        minikanren.Neq(q, minikanren.NewAtom("forbidden")),
    )
})
```

### 2. Generate-and-Test Pattern

Use the **generate-and-test** pattern for reliable constraint checking:

```go
minikanren.Run(10, func(q *minikanren.Var) minikanren.Goal {
    candidates := minikanren.List(
        minikanren.NewAtom("hello"),
        minikanren.NewAtom(42),
        minikanren.NewAtom("world"),
    )
    
    return minikanren.Conj(
        // Generate: pick a candidate
        minikanren.Membero(q, candidates),
        
        // Test: apply constraints
        minikanren.Symbolo(q),                              // Must be string
        minikanren.Neq(q, minikanren.NewAtom("hello")),     // Must not be "hello"
    )
})
// Result: ["world"] 
```

### 3. Multiple Constraints

When using multiple constraints, order them from most to least restrictive:

```go
minikanren.Conj(
    minikanren.Eq(q, value),                    // Binding
    minikanren.Symbolo(q),                      // Type constraint (restrictive)
    minikanren.Neq(q, minikanren.NewAtom("bad")), // Value constraint (less restrictive)
    minikanren.Absento(minikanren.NewAtom("x"), q), // Structure constraint (least restrictive)
)
```

## Constraint Reference

### Type Constraints

#### Symbolo(term)
Ensures term is a string atom.

```go
// Works correctly
minikanren.Conj(
    minikanren.Eq(q, minikanren.NewAtom("symbol")),
    minikanren.Symbolo(q),  // ✅ Succeeds - "symbol" is a string
)

minikanren.Conj(
    minikanren.Eq(q, minikanren.NewAtom(42)),
    minikanren.Symbolo(q),  // ❌ Fails - 42 is not a string
)
```

#### Numbero(term)
Ensures term is a numeric atom.

```go
// Supports all Go numeric types
minikanren.Numbero(minikanren.NewAtom(42))      // int ✅
minikanren.Numbero(minikanren.NewAtom(3.14))    // float64 ✅ 
minikanren.Numbero(minikanren.NewAtom("42"))    // string ❌
```

### Value Constraints

#### Neq(t1, t2)
Ensures two terms are not equal.

```go
// Disequality constraint
minikanren.Conj(
    minikanren.Eq(q, minikanren.NewAtom("allowed")),
    minikanren.Neq(q, minikanren.NewAtom("forbidden")), // ✅ "allowed" ≠ "forbidden"
)

minikanren.Conj(
    minikanren.Eq(q, minikanren.NewAtom("forbidden")),
    minikanren.Neq(q, minikanren.NewAtom("forbidden")), // ❌ "forbidden" = "forbidden"
)
```

#### Absento(absent, term)
Ensures a value doesn't appear anywhere in a structure.

```go
list := minikanren.List(
    minikanren.NewAtom("good"),
    minikanren.NewAtom("ok"),
)

minikanren.Conj(
    minikanren.Eq(q, list),
    minikanren.Absento(minikanren.NewAtom("bad"), q), // ✅ "bad" not in list
)
```

### List Constraints

#### Membero(elem, list)
Relational membership - can generate or test.

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

// Test mode: check membership
minikanren.Run(1, func(q *minikanren.Var) minikanren.Goal {
    return minikanren.Conj(
        minikanren.Membero(minikanren.NewAtom(2), list), // Test: is 2 in list?
        minikanren.Eq(q, minikanren.NewAtom("found")),
    )
})
// Result: ["found"]
```

### Control Constraints

#### Onceo(goal)
Succeeds at most once (cut operator).

```go
// Without Onceo: multiple solutions
minikanren.Run(10, func(q *minikanren.Var) minikanren.Goal {
    return minikanren.Disj(
        minikanren.Eq(q, minikanren.NewAtom(1)),
        minikanren.Eq(q, minikanren.NewAtom(2)), 
        minikanren.Eq(q, minikanren.NewAtom(3)),
    )
})
// Result: [1, 2, 3]

// With Onceo: only first solution
minikanren.Run(10, func(q *minikanren.Var) minikanren.Goal {
    return minikanren.Onceo(minikanren.Disj(
        minikanren.Eq(q, minikanren.NewAtom(1)),
        minikanren.Eq(q, minikanren.NewAtom(2)),
        minikanren.Eq(q, minikanren.NewAtom(3)),
    ))
})
// Result: [1]
```

## Advanced Patterns

### Conditional Constraints

Use `Project` to apply constraints based on computed values:

```go
minikanren.Run(10, func(q *minikanren.Var) minikanren.Goal {
    x := minikanren.Fresh("x")
    y := minikanren.Fresh("y")
    
    return minikanren.Conj(
        minikanren.Eq(x, minikanren.NewAtom(10)),
        minikanren.Eq(y, minikanren.NewAtom(20)),
        
        minikanren.Project([]minikanren.Term{x, y}, func(values []minikanren.Term) minikanren.Goal {
            // Extract values and apply conditional logic
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

### Complex List Constraints

Combine multiple list operations with constraints:

```go
// Find lists where first element is a symbol and "bad" is not present
minikanren.Run(10, func(q *minikanren.Var) minikanren.Goal {
    head := minikanren.Fresh("head")
    tail := minikanren.Fresh("tail")
    
    return minikanren.Conj(
        // Structure: q is a non-empty list
        minikanren.Cons(head, tail, q),
        
        // Constraints on structure
        minikanren.Symbolo(head),                                    // First element is symbol
        minikanren.Absento(minikanren.NewAtom("bad"), q),           // "bad" not anywhere in list
        
        // Example binding
        minikanren.Eq(q, minikanren.List(
            minikanren.NewAtom("good"),
            minikanren.NewAtom("ok"),
        )),
    )
})
```

## Debugging Constraints

### Common Issues

1. **Constraint placed before unification**:
   ```go
   // Problem: constraint sees unbound variable
   minikanren.Conj(
       minikanren.Symbolo(q),      // q is unbound - constraint defers
       minikanren.Eq(q, value),    // q gets bound after constraint
   )
   
   // Solution: unify first
   minikanren.Conj(
       minikanren.Eq(q, value),    // q gets bound
       minikanren.Symbolo(q),      // constraint sees bound value
   )
   ```

2. **Expecting constraint propagation**:
   ```go
   // This doesn't work as expected in our implementation
   x := minikanren.Fresh("x")
   y := minikanren.Fresh("y")
   
   minikanren.Conj(
       minikanren.Neq(x, y),       // Sets up disequality
       minikanren.Eq(x, value),    // Binds x
       minikanren.Eq(y, value),    // Binds y to same value - should fail but doesn't
   )
   ```

### Debugging Tips

1. **Test constraints individually** with concrete values
2. **Use generate-and-test pattern** for complex constraint combinations  
3. **Check variable binding order** in conjunctions
4. **Add debug output** to see intermediate bindings

## Performance Implications

### Constraint Overhead

- **Type constraints** (`Symbolo`, `Numbero`): Low overhead, fast checks
- **Structural constraints** (`Absento`): Higher overhead, recursive traversal
- **Relational constraints** (`Membero`): Variable overhead based on list size

### Optimization Tips

1. **Place most restrictive constraints first** to fail fast
2. **Use concrete bindings** before constraints when possible
3. **Avoid complex structural constraints** on large data structures
4. **Consider parallel execution** for independent constraint checking

## Migration from Other miniKanren Implementations

If migrating from Scheme miniKanren or core.logic:

1. **Reorder constraint goals** to come after unification
2. **Use generate-and-test pattern** for complex constraint combinations
3. **Test constraint behavior** with your specific use cases
4. **Take advantage of parallel execution** for performance

The order-dependent constraint model is a deliberate design choice that provides:
- **Predictable performance** (no hidden constraint propagation)
- **Simple implementation** (easier to understand and debug)  
- **High performance** (minimal overhead for constraint checking)
- **Thread safety** (no global constraint store to synchronize)