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

### Booleans and Encoding Semantics

For consistency with 1-indexed finite domains, boolean variables use the domain {1,2} where:

- 1 means false
- 2 means true

This encoding allows boolean variables to participate in the same domain operations and constraints as integer variables while keeping all domains strictly positive.

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

### Reification: Turning Constraints into Booleans

Reification links the truth of a constraint to a boolean variable. This enables modeling rich logical patterns such as implications, cardinalities, and conditional constraints.

Key components:
- ReifiedConstraint: wraps a core constraint C and a boolean B so that B=2 iff C holds, B=1 iff ¬C holds
- EqualityReified: specialized, bidirectional version for X == Y ↔ B
- ValueEqualsReified: specialized reification for X == constant ↔ B

Propagation semantics:
- If B is 2 (true), the wrapped constraint is enforced and can prune variable domains
- If B is 1 (false), the negation of the constraint is enforced for core constraints (Arithmetic, Inequality, AllDifferent)
- If B is {1,2} (unknown), domains of X/Y are not pruned by the wrapped constraint to avoid bias; however, if the wrapped constraint becomes impossible under current domains, B is set to 1 (false)

Example:

```go
// X + 1 = Y reified into boolean B
model := minikanren.NewModel(9)
X := model.NewVariable()
Y := model.NewVariable()
// Boolean variable uses {1:false, 2:true}
B := model.NewVariable()
_ = B.SetDomain(minikanren.NewBitSetDomainFromValues(9, []int{1,2}))

arith, _ := minikanren.NewArithmetic(X, 1, Y)
reif, _ := minikanren.NewReifiedConstraint(arith, B)
_ = model.AddConstraint(reif)

solver := minikanren.NewSolver(model)
solutions, _ := solver.Solve(context.Background(), 1)
_ = solutions
```

See ExampleReifiedConstraint in the package for a runnable example.

### Counting with Reification (Count)

The Count constraint models: among vars, exactly count of them equal targetValue.

Implementation overview:
- For each variable vi, create a boolean bi reifying vi == targetValue (using ValueEqualsReified)
- Sum the booleans with BoolSum into a total T
- Encode count via T = count + 1 using a 1-indexed total domain; i.e., if k variables are true, T takes value k+1

API:

```go
// NewCount(model, vars, targetValue, countVar)
// - vars: []*FDVariable to inspect
// - targetValue: integer to count
// - countVar: variable whose domain encodes the count as [1..len(vars)+1]
//   actualCount = countVarSingleton - 1
c, err := minikanren.NewCount(model, vars, targetValue, countVar)
if err != nil { /* handle error */ }
_ = model.AddConstraint(c)
```

Encoding and domains:
- Each boolean bi has domain {1,2} (false/true)
- The total T (internal) and the provided countVar both live in [1 .. len(vars)+1]
- The actual count is T-1; this preserves the solver’s positive-domain invariant

Propagation strength:
- Extremes: if count=1 then exactly one var can equal targetValue; if count=0 the target is removed from all vars
- Bounds: if count ≥ m, at least m variables will be forced to the target when possible; if count ≤ u, at most u variables can equal the target, pruning others accordingly

Example:

```go
model := minikanren.NewModel(9)
X, Y, Z := model.NewVariable(), model.NewVariable(), model.NewVariable()
// All in 1..9 by default

// Count how many are equal to 5
N := model.NewVariable()
// N encodes [1..4] → counts 0..3
_ = N.SetDomain(minikanren.NewBitSetDomain(4))

_, _ = minikanren.NewCount(model, []*minikanren.FDVariable{X, Y, Z}, 5, N)

solver := minikanren.NewSolver(model)
solutions, _ := solver.Solve(context.Background(), 0)
_ = solutions
```

Notes and best practices:
- Create the countVar in the same Model as vars
- Use the Model’s MaxValue to validate domains; NewCount enforces that countVar’s max ≤ len(vars)+1
- Booleans use {1:false,2:true}; avoid directly assigning 0/1

See ExampleCount in the package for a runnable example.

### Counting Membership in a Set (Among)

The Among global constraint counts how many variables take values from a given set S, and constrains that count via a dedicated variable K using the solver's positive-domain encoding.

Concepts and encoding:
- Given variables vars = [X1..Xn] and a set of values S, Among(vars, S, K) ensures count({i | Xi ∈ S}) = K-1, where K ∈ [1..n+1]. This mirrors the Count encoding and keeps all domains strictly positive.

Propagation strength (bounds-consistent):
- Let m = number of Xi whose domains are subsets of S (mandatory), and p = number of Xi whose domains intersect S (possible). Then m ≤ count ≤ p. The constraint prunes K to [m+1..p+1].
- If m equals K's maximum count (K.max-1), all remaining may-in variables are forced OUT of S.
- If p equals K's minimum count (K.min-1), all may-in variables are forced INTO S.

API:

```go
// NewAmong(vars, values, k)
//  - vars: []*FDVariable to inspect
//  - values: []int representing set S
//  - k: FD variable with domain in [1..len(vars)+1] encoding the count as K = count+1
c, err := minikanren.NewAmong(vars, []int{1,2,3}, k)
if err != nil { /* handle error */ }
_ = model.AddConstraint(c)
```

Example: See ExampleNewAmong in the package and the runnable demo at examples/among-demo.

### Post-solve Domain Inspection and Solver Semantics

For convenience, the solver caches the root-level propagated state after an initial Solve. This allows querying domains without threading an explicit state:

- solver.GetDomain(nil, varID) returns the domain from the last propagated root state when state is nil
- Root-level inconsistency now yields zero solutions (no error). However, model validation errors (e.g., variable with empty domain before solving) still return an error

This behavior makes it straightforward to inspect pruned domains after Solve() in tests and examples.

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