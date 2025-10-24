# GoKanren Quick Reference

## Core Operators

### Variable Creation
```go
x := minikanren.Fresh("x")  // Create named variable
y := minikanren.Fresh("y")  // Create another variable
```

### Unification
```go
minikanren.Eq(x, minikanren.NewAtom("hello"))     // x = "hello"
minikanren.Eq(x, y)                               // x = y
```

### Logical Operators
```go
minikanren.Conj(goal1, goal2, goal3)  // AND: all must succeed
minikanren.Disj(goal1, goal2, goal3)  // OR: any can succeed
```

### Query Execution
```go
// Get up to n solutions
results := minikanren.Run(5, func(q *minikanren.Var) minikanren.Goal {
    return goal
})

// Get all solutions (use carefully!)
results := minikanren.RunStar(func(q *minikanren.Var) minikanren.Goal {
    return goal
})
```

## Constraint Operators

### Type Constraints
```go
minikanren.Symbolo(x)   // x must be a string
minikanren.Numbero(x)   // x must be a number
```

### Value Constraints  
```go
minikanren.Neq(x, y)                    // x ≠ y
minikanren.Absento(needle, haystack)    // needle not in haystack
```

### List Operations
```go
minikanren.Membero(elem, list)          // elem is member of list
minikanren.Appendo(l1, l2, l3)          // l1 + l2 = l3
minikanren.Car(list, head)              // head is first element
minikanren.Cdr(list, tail)              // tail is rest of list
minikanren.Cons(head, tail, list)       // construct list
minikanren.Nullo(list)                  // list is empty
minikanren.Pairo(list)                  // list is non-empty
```

### Control Flow
```go
minikanren.Onceo(goal)                  // succeed at most once
minikanren.Conda(                       // if-then-else with cut
    []minikanren.Goal{cond1, then1},
    []minikanren.Goal{cond2, then2},
    []minikanren.Goal{minikanren.Success, elseGoal},
)
```

## Data Construction

### Atoms
```go
minikanren.NewAtom("string")    // String atom
minikanren.NewAtom(42)          // Number atom  
minikanren.NewAtom(true)        // Boolean atom
```

### Lists
```go
minikanren.List(                        // Proper list
    minikanren.NewAtom(1),
    minikanren.NewAtom(2), 
    minikanren.NewAtom(3),
)

minikanren.Nil                          // Empty list
```

### Pairs
```go
minikanren.NewPair(car, cdr)            // Cons cell
```

## Parallel Execution

### Basic Parallel
```go
results := minikanren.ParallelRun(10, func(q *minikanren.Var) minikanren.Goal {
    return minikanren.ParallelDisj(
        goal1, goal2, goal3,
    )
})
```

### With Configuration
```go
config := &minikanren.ParallelConfig{
    MaxWorkers:          8,
    BufferSize:          1000,
    BackpressureEnabled: true,
}

results := minikanren.ParallelRunWithConfig(10, goalFunc, config)
```

## Common Patterns

### Generate and Test
```go
minikanren.Run(10, func(q *minikanren.Var) minikanren.Goal {
    return minikanren.Conj(
        // Generate candidates
        minikanren.Membero(q, candidates),
        // Test constraints
        minikanren.Symbolo(q),
        minikanren.Neq(q, minikanren.NewAtom("excluded")),
    )
})
```

### List Processing
```go
// Find all ways to append two lists to get [1,2,3]
minikanren.Run(10, func(q *minikanren.Var) minikanren.Goal {
    x := minikanren.Fresh("x")
    y := minikanren.Fresh("y")
    target := minikanren.List(
        minikanren.NewAtom(1),
        minikanren.NewAtom(2), 
        minikanren.NewAtom(3),
    )
    
    return minikanren.Conj(
        minikanren.Appendo(x, y, target),
        minikanren.Eq(q, minikanren.List(x, y)),
    )
})
```

### Constraint Ordering (Important!)
```go
// ✅ Correct: bind first, then constrain
minikanren.Conj(
    minikanren.Eq(x, value),      // Bind variable
    minikanren.Symbolo(x),        // Check constraint
)

// ❌ Unreliable: constraint before binding
minikanren.Conj(
    minikanren.Symbolo(x),        // Constraint on unbound var
    minikanren.Eq(x, value),      // Binding happens after
)
```

## Performance Tips

1. **Use Run(n, ...) instead of RunStar** for large solution spaces
2. **Order constraints after unification** for reliable behavior  
3. **Use parallel execution** for complex goals with independent choice points
4. **Avoid parallel execution** for simple queries (overhead dominates)
5. **Use context cancellation** for long-running queries

## Error Handling

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

results := minikanren.RunWithContext(ctx, 100, goalFunc)
// Automatically stops after 5 seconds or 100 results
```