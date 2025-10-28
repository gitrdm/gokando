# Finite Domain (FD) Solver

The finite domain (FD) solver is a complete constraint satisfaction problem (CSP) engine integrated with miniKanren. It provides efficient domain propagation, advanced heuristics, and comprehensive monitoring for solving combinatorial problems.

## Overview

The FD solver implements a state-of-the-art constraint satisfaction engine with:

- **BitSet-based domains** for memory-efficient representation of integer domains
- **AC-3 style propagation** for efficient constraint propagation
- **Regin filtering** for advanced all-different constraints
- **Multiple search heuristics** (domain size, degree, lexicographic, random)
- **Thread-safe monitoring** and statistics collection
- **Custom constraint framework** for user-defined constraints
- **Iterative backtracking** with configurable value ordering

## Core Concepts

### Domains

FD variables have domains representing the set of possible integer values they can take. Domains are implemented using efficient BitSet structures:

```go
// Create a domain with values 1-9
store := minikanren.NewFDStoreWithDomain(9)

// Create variables with full domains
vars := store.MakeFDVars(3) // Three variables, each with domain {1,2,3,4,5,6,7,8,9}
```

### Constraints

Constraints restrict the possible values variables can take. The solver supports:

- **All-different**: Variables must take distinct values
- **Arithmetic**: Offset relationships between variables
- **Inequality**: Ordering constraints between variables
- **Custom**: User-defined constraint logic

### Propagation

The solver uses constraint propagation to reduce domains before search:

```go
// Add all-different constraint with Regin filtering
if err := store.AddAllDifferentRegin(vars); err != nil {
    // Handle inconsistency
}

// Add arithmetic constraint: var[0] + 2 = var[1]
if err := store.AddOffsetConstraint(vars[0], 2, vars[1]); err != nil {
    // Handle inconsistency
}
```

### Search

When propagation alone cannot find a solution, the solver uses backtracking search:

```go
// Solve with limit on number of solutions
solutions, err := store.Solve(ctx, 10)
if err != nil {
    // Handle error
}

// Each solution is a slice of integers
for _, sol := range solutions {
    fmt.Printf("Solution: %v\n", sol)
}
```

## Domain Operations

### Creating Domains

```go
// Domain size 9 (values 1-9) - Sudoku standard
store := minikanren.NewFDStoreWithDomain(9)

// Custom configuration
config := &minikanren.SolverConfig{
    VariableHeuristic: minikanren.HeuristicDomDeg,
    ValueHeuristic:    minikanren.ValueOrderAsc,
    RandomSeed:        42,
}
store := minikanren.NewFDStoreWithConfig(9, config)
```

### Domain Manipulation

```go
var v *minikanren.FDVar = store.NewVar()

// Assign singleton value
if err := store.Assign(v, 5); err != nil {
    // Domain becomes {5}
}

// Remove a value from domain
if err := store.Remove(v, 3); err != nil {
    // Value 3 removed from domain
}

// Intersect with another domain
otherDomain := minikanren.NewBitSet(9)
// ... set some values in otherDomain ...
if err := store.IntersectDomains(v, otherDomain); err != nil {
    // Domain intersected with otherDomain
}

// Union with another domain
if err := store.UnionDomains(v, otherDomain); err != nil {
    // Domain unioned with otherDomain
}

// Complement domain
if err := store.ComplementDomain(v); err != nil {
    // Domain becomes complement within 1..domainSize
}
```

## Constraint Types

### All-Different Constraints

The all-different constraint ensures variables take distinct values:

```go
// Basic all-different (pairwise propagation only)
store.AddAllDifferent(vars)

// Advanced all-different with Regin filtering
store.AddAllDifferentRegin(vars)
```

Regin filtering uses maximum bipartite matching to detect when values cannot participate in any complete assignment, providing stronger pruning than basic pairwise propagation.

### Arithmetic Constraints

Offset constraints model arithmetic relationships:

```go
// var1 + offset = var2
store.AddOffsetConstraint(var1, offset, var2)

// Example: diagonal constraints in N-Queens
for i := 0; i < n; i++ {
    // Queen i at column vars[i]
    // Diagonal 1: column + row = constant
    store.AddOffsetConstraint(vars[i], i, diag1[i])
    // Diagonal 2: column - row = constant
    store.AddOffsetConstraint(vars[i], -i+n, diag2[i])
}
```

### Inequality Constraints

Ordering constraints between variables:

```go
// var1 < var2
store.AddInequalityConstraint(var1, var2, minikanren.IneqLessThan)

// var1 <= var2
store.AddInequalityConstraint(var1, var2, minikanren.IneqLessEqual)

// var1 > var2
store.AddInequalityConstraint(var1, var2, minikanren.IneqGreaterThan)

// var1 >= var2
store.AddInequalityConstraint(var1, var2, minikanren.IneqGreaterEqual)

// var1 != var2
store.AddInequalityConstraint(var1, var2, minikanren.IneqNotEqual)
```

### Custom Constraints

Implement your own constraint logic:

```go
type SumConstraint struct {
    vars   []*minikanren.FDVar
    target int
}

func (c *SumConstraint) Variables() []*minikanren.FDVar {
    return c.vars
}

func (c *SumConstraint) Propagate(store *minikanren.FDStore) (bool, error) {
    // Implement propagation logic
    // Return true if domains changed, false otherwise
    // Return error if constraint becomes inconsistent
}

func (c *SumConstraint) IsSatisfied() bool {
    // Check if constraint is satisfied with current domains
}

// Add to store
constraint := &SumConstraint{vars: myVars, target: 15}
if err := store.AddCustomConstraint(constraint); err != nil {
    // Handle error
}
```

## Search Heuristics

### Variable Ordering Heuristics

Control which variable is selected next during search:

```go
config := &minikanren.SolverConfig{
    VariableHeuristic: minikanren.HeuristicDomDeg, // Default: smallest domain/degree ratio
    // Other options:
    // HeuristicDom     // Smallest domain size
    // HeuristicDeg     // Highest degree (most constraints)
    // HeuristicLex     // Lexicographic order
    // HeuristicRandom  // Random order
}
```

- **Dom/Deg**: `(domain size) / (degree + 1)` - balances domain size and constraint count
- **Domain**: Smallest domain first - good for finding solutions quickly
- **Degree**: Most constrained variables first - reduces branching factor
- **Lexicographic**: Deterministic ordering - good for reproducibility
- **Random**: Randomized ordering - good for exploring different solution paths

### Value Ordering Heuristics

Control the order values are tried within a domain:

```go
config.ValueHeuristic = minikanren.ValueOrderAsc // Default: ascending order
// Other options:
// ValueOrderDesc    // Descending order
// ValueOrderRandom  // Random order
// ValueOrderMid     // Start from middle, alternate outward
```

## Monitoring and Statistics

Track solver performance and behavior:

```go
// Enable monitoring
monitor := minikanren.NewSolverMonitor()
store.SetMonitor(monitor)

// Solve problems...
solutions, err := store.Solve(ctx, 100)

// Get statistics
stats := monitor.GetStats()
fmt.Printf("Solutions found: %d\n", stats.SolutionsFound)
fmt.Printf("Nodes explored: %d\n", stats.NodesExplored)
fmt.Printf("Backtracks: %d\n", stats.Backtracks)
fmt.Printf("Search time: %v\n", stats.SearchTime)
fmt.Printf("Propagation time: %v\n", stats.PropagationTime)
```

Available statistics:
- **Search metrics**: nodes explored, backtracks, solutions found, search time
- **Propagation metrics**: propagation operations, propagation time, constraints added
- **Memory metrics**: peak trail size, peak queue size
- **Domain metrics**: initial/final domains, domain reductions per variable

## Integration with miniKanren

FD constraints integrate seamlessly with miniKanren goals:

```go
// FD all-different goal
goal := minikanren.FDAllDifferentGoal(vars, 9)

// FD N-Queens goal
goal := minikanren.FDQueensGoal(vars, 8)

// FD inequality goal
goal := minikanren.FDInequalityGoal(x, y, minikanren.IneqLessThan)

// Custom constraint goal
constraint := &MyCustomConstraint{/* ... */}
goal := minikanren.FDCustomGoal(vars, constraint)

// Execute with miniKanren
results := minikanren.Run(10, func(q *minikanren.Var) minikanren.Goal {
    return minikanren.Conj(
        goal, // FD constraint
        // ... other miniKanren goals
    )
})
```

## Performance Tuning

### Configuration Options

```go
config := &minikanren.SolverConfig{
    VariableHeuristic: minikanren.HeuristicDomDeg,
    ValueHeuristic:    minikanren.ValueOrderMid,
    RandomSeed:        time.Now().UnixNano(), // For reproducible randomness
}
store := minikanren.NewFDStoreWithConfig(9, config)
```

### When to Use Different Heuristics

- **Dom/Deg**: General-purpose, good balance of speed and pruning
- **Domain**: When you want to find any solution quickly
- **Degree**: For problems with complex constraint interactions
- **Random**: For exploring diverse solution spaces or avoiding local minima

### Memory Considerations

- Domains use BitSet representation - very memory efficient
- Trail-based backtracking minimizes memory allocation
- Propagation queue and trail grow with problem complexity
- Monitor domain snapshots if tracking statistics

## Error Handling

The solver returns specific errors for different failure modes:

```go
solutions, err := store.Solve(ctx, 10)
if err != nil {
    switch err {
    case minikanren.ErrInconsistent:
        // Problem has no solution
    case minikanren.ErrInvalidValue:
        // Value outside domain range
    case minikanren.ErrDomainEmpty:
        // Domain became empty during propagation
    case minikanren.ErrInvalidArgument:
        // Invalid constraint parameters
    default:
        // Other errors (context cancellation, etc.)
    }
}
```

## Examples

### Sudoku Solver

```go
func solveSudoku(puzzle [9][9]int) ([9][9]int, error) {
    store := minikanren.NewFDStoreWithDomain(9)
    vars := make([][]*minikanren.FDVar, 9)
    for i := range vars {
        vars[i] = make([]*minikanren.FDVar, 9)
        for j := range vars[i] {
            vars[i][j] = store.NewVar()
        }
    }

    // Add given values
    for i := 0; i < 9; i++ {
        for j := 0; j < 9; j++ {
            if puzzle[i][j] != 0 {
                if err := store.Assign(vars[i][j], puzzle[i][j]); err != nil {
                    return [9][9]int{}, err
                }
            }
        }
    }

    // Add row constraints
    for i := 0; i < 9; i++ {
        row := make([]*minikanren.FDVar, 9)
        for j := 0; j < 9; j++ {
            row[j] = vars[i][j]
        }
        if err := store.AddAllDifferentRegin(row); err != nil {
            return [9][9]int{}, err
        }
    }

    // Add column constraints
    for j := 0; j < 9; j++ {
        col := make([]*minikanren.FDVar, 9)
        for i := 0; i < 9; i++ {
            col[i] = vars[i][j]
        }
        if err := store.AddAllDifferentRegin(col); err != nil {
            return [9][9]int{}, err
        }
    }

    // Add box constraints
    for box := 0; box < 9; box++ {
        bx := make([]*minikanren.FDVar, 9)
        for k := 0; k < 9; k++ {
            i := (box/3)*3 + k/3
            j := (box%3)*3 + k%3
            bx[k] = vars[i][j]
        }
        if err := store.AddAllDifferentRegin(bx); err != nil {
            return [9][9]int{}, err
        }
    }

    // Solve
    solutions, err := store.Solve(context.Background(), 1)
    if err != nil || len(solutions) == 0 {
        return [9][9]int{}, err
    }

    // Convert solution back to grid
    var result [9][9]int
    for i := 0; i < 9; i++ {
        for j := 0; j < 9; j++ {
            result[i][j] = solutions[0][i*9+j]
        }
    }

    return result, nil
}
```

### N-Queens Problem

```go
func solveNQueens(n int) ([][]int, error) {
    store := minikanren.NewFDStoreWithDomain(n)

    // Create queen position variables (columns 1-n)
    queens := store.MakeFDVars(n)

    // Create diagonal variables (extended domain for offsets)
    diag1 := store.MakeFDVars(n) // column + row
    diag2 := store.MakeFDVars(n) // column - row + n

    // Queens in different columns
    if err := store.AddAllDifferentRegin(queens); err != nil {
        return nil, err
    }

    // Queens in different diagonals
    if err := store.AddAllDifferentRegin(diag1); err != nil {
        return nil, err
    }
    if err := store.AddAllDifferentRegin(diag2); err != nil {
        return nil, err
    }

    // Link queen positions to diagonals
    for i := 0; i < n; i++ {
        // diag1[i] = queens[i] + i
        if err := store.AddOffsetConstraint(queens[i], i, diag1[i]); err != nil {
            return nil, err
        }
        // diag2[i] = queens[i] - i + n
        if err := store.AddOffsetConstraint(queens[i], -i+n, diag2[i]); err != nil {
            return nil, err
        }
    }

    return store.Solve(context.Background(), 0) // All solutions
}
```

This FD solver provides a powerful, efficient foundation for solving complex combinatorial problems with a clean, idiomatic Go API.</content>
<parameter name="filePath">/home/rdmerrio/gits/gokando/docs/minikanren/finite-domains.md