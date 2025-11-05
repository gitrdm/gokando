# gokanlogic Examples

This directory contains complete example programs demonstrating the comprehensive constraint solving capabilities of the gokanlogic library, including miniKanren, finite domain (FD) solving, and hybrid approaches.

**Recent Updates:** The examples have been significantly updated to reflect the current state of the library, showcasing modern APIs, sophisticated global constraints (Circuit, Table, Regular, Cumulative, etc.), and professional-grade constraint solving capabilities.

### Send More Money Cryptarithm (FD Solver)

**Path:** `examples/send-more-money/`

The classic cryptarithm puzzle demonstrating **FD solver** with arithmetic constraint solving.

**Run:**
```bash
cd examples/send-more-money
go run main.go
```

**The Puzzle:**
Find unique digits for letters S,E,N,D,M,O,R,Y such that SEND + MORE = MONEY, where S and M cannot be zero.

**Features demonstrated:**
- **FD solver with global constraints**: AllDifferent for digit uniqueness
- **Arithmetic modeling**: Direct constraint representation of cryptarithm equations  
- **Carry propagation**: Proper handling of multi-digit arithmetic with carries
- **Leading zero constraints**: Ensuring S and M are non-zero
- **Efficient solving**: Finds solutions quickly using constraint propagation

**Sample Output:**
```
=== FD SEND + MORE = MONEY ===
Letter → Digit mapping:
  S → 9
  E → 5
  N → 6
  D → 7
  M → 1
  O → 0
  R → 8
  Y → 2

  9567
+ 1085
------
 10652
```

The example now successfully solves the classic SEND + MORE = MONEY cryptarithm using the FD solver with sophisticated constraint propagation. This demonstrates the power of the finite domain constraint solver for arithmetic puzzle solving.

### Magic Square (FD Solver)

**Path:** `examples/magic-square/`

The classic 3x3 magic square puzzle demonstrating **FD solver** with arithmetic and AllDifferent constraints.

**Run:**
```bash
cd examples/magic-square
go run main.go
```

**The Puzzle:**
Find a 3x3 grid of distinct digits (1-9) where each row, column, and diagonal sums to 15, using the FD solver with arithmetic and AllDifferent constraints.

**Features demonstrated:**
- **FD solver with arithmetic constraints**: Sum constraints for rows, columns, and diagonals
- **AllDifferent constraint**: Ensures all grid values are unique (1-9)
- **Constraint composition**: Multiple interacting constraint types
- **Efficient search**: Finding valid magic squares through constraint propagation

**Sample Output:**
```
=== FD Magic Square (3x3) ===
Solution:
 2  9  4 
 7  5  3 
 6  1  8
```

**Features demonstrated:**
- **FD solver with arithmetic constraints**: Sum constraints for rows, columns, and diagonals
- **AllDifferent constraint**: Ensures all grid values are unique (1-9)
- **Global constraint propagation**: Efficient pruning of incompatible value combinations
- **Complete constraint satisfaction**: Finding valid 3x3 magic squares efficiently

The example successfully demonstrates the power of the FD solver for arithmetic constraint problems, finding valid magic squares where each row, column, and diagonal sums to 15.

### Knight's Tour (Modern FD Solver)

**Path:** `examples/knights-tour/`

The classic Knight's Tour puzzle demonstrating **modern FD solver** with global constraints including Circuit and Table constraints.

**Run:**
```bash
cd examples/knights-tour
go run main.go
```

**The Puzzle:**
Find a sequence of knight moves on a 6x6 chessboard that visits every square exactly once. Knights move in an L-shape: 2 squares in one direction and 1 square perpendicular. Note: 6x6 is the smallest board size that admits knight's tours (5x5 and smaller are mathematically impossible).

**Features demonstrated:**
- **Modern Model/Solver API** with sophisticated global constraints
- **Circuit constraint**: Models Hamiltonian cycles for visiting all squares exactly once
- **Table constraint**: Efficiently encodes valid knight moves between squares
- **Constraint composition**: Combining multiple global constraints for complex problems
- **Advanced constraint propagation**: Significant search space pruning
- **Successful solution finding**: Actually solves a challenging combinatorial problem

**Sample Output:**
```
=== Knight's Tour on 6x6 Board (Modern FD Solver) ===
Note: 6x6 is the smallest board size that admits knight's tours.

Generated 160 valid knight moves for 6x6 board
✓ Found a knight's tour!

Board showing move sequence:
  1   6  11  30  27   4 
 10  31   2   5  12  29 
  7  36   9  28   3  26 
 32  23  34  19  16  13 
 35   8  21  14  25  18 
 22  33  24  17  20  15 

✅ Knight visited all 36 squares exactly once!
```

#### Modern FD Solver Success Story

This example showcases the impressive capabilities of the modern FD solver:

**✅ Successfully Implemented and Working:**
- **Modern Model/Solver API**: Clean, composable constraint modeling
- **Circuit global constraint**: Professional-grade Hamiltonian cycle constraint
- **Table global constraint**: Efficient extensional constraint representation  
- **Sophisticated constraint propagation**: Advanced domain pruning and consistency
- **Constraint composition**: Multiple global constraints working together seamlessly
- **Actual problem solving**: Finds real knight's tours, not just constraint validation

**🔍 Key Insights:**
- **160 valid knight moves generated** showing comprehensive move encoding
- **Successful tour discovery** demonstrates the solver can handle complex combinatorial problems
- **Circuit + Table combination** provides state-of-the-art constraint modeling
- **Reasonable solving time** shows the constraint propagation is effective
- **Mathematical correctness**: Uses 6x6 board (smallest size where tours exist)

**🚀 Why This Works:**
- **Proper problem size**: 6x6 boards admit knight's tours (unlike 5x5 which are impossible)
- **Global constraints**: Circuit ensures Hamiltonian structure, Table enforces move validity
- **Effective propagation**: Combined constraints prune the search space efficiently
- **Modern architecture**: Professional constraint solver design

This example demonstrates that the FD solver has evolved into a production-ready constraint programming system capable of solving challenging real-world combinatorial problems.
```

**Sample Output:**
```
=== Knight's Tour on 5x5 Board ===

Found 2 complete assignments, validating against knight move rules...
✓ Expected result: constraint validation working - no valid knight's tour found among 2 assignments

✓ FD Solver successfully exercised!

This example demonstrates:
- FDStore with AllDifferent constraints for uniqueness
- Custom constraint framework using public domain access methods
- Post-assignment validation of complex combinatorial constraints

Note: Complete knight's tours require sophisticated constraint
propagation algorithms. The solver finds assignments satisfying
uniqueness, but knight move constraints are validated separately.

This reveals an important limitation: while the framework works,
some constraint problems need stronger propagation algorithms.
```

#### FD Solver Capabilities and Current Limitations

Based on the current FD solver implementation, here are key insights about its constraint capabilities:

**✅ Currently Implemented and Working:**
- **BitSet-based finite domains** with efficient 1-based indexing
- **AC-3 propagation algorithm** for basic constraint consistency
- **AllDifferent constraints** with both pairwise and Regin filtering (bipartite matching)
- **Arithmetic offset constraints** (X = Y + constant) with bidirectional propagation
- **Inequality constraints** (<, ≤, >, ≥, ≠) with bounds-based domain pruning
- **Custom constraint framework** for user-defined constraints
- **Multiple search heuristics** (dom/deg, dom, deg, lex, random)
- **Backtracking search** with trail-based undo and monitoring capabilities
- **Global constraints**: Circuit, Table, Regular, Cumulative, GCC, Among, Lex (see other examples)

**🔍 Key Insights:**
- The FD solver excels at **basic combinatorial problems** with AllDifferent and arithmetic constraints
- **Knight's tours require specialized global constraints** like circuit/path constraints for modeling graph structures
- Current implementation demonstrates **validation capabilities** but knight move constraints are complex
- The **constraint framework is solid** - many other global constraints work well (see TSP, Table, Regular examples)

**🚀 Current Alternative:**
See the **TSP example with Circuit constraint** which successfully models and solves Hamiltonian cycles, demonstrating that the global constraint framework is robust for many complex combinatorial problems.

### Graph Coloring (Australia Map)

**Path:** `examples/graph-coloring/`

A classic graph coloring problem demonstrating **parallel search** with `ParallelRun` and `ParallelDisj`.

**Run:**
```bash
cd examples/graph-coloring
go run main.go          # parallel search (default)
go run main.go seq      # sequential search for comparison
```

**The Problem:**
Color the 7 regions of Australia (WA, NT, SA, Q, NSW, V, T) using 3 colors (red, green, blue) such that no two adjacent regions share the same color.

**Features demonstrated:**
- **Parallel search** with `ParallelRun` and `ParallelDisj`
- Graph constraint satisfaction with `Neq` for adjacency rules
- Performance comparison between sequential and parallel execution
- Clean relational encoding of graph structure
- Custom `ParallelExecutor` configuration

**Sample Output:**
```
🚀 Using PARALLEL search with ParallelRun and ParallelDisj

✓ Solution found in 10.55864ms!

Region Coloring:
  WA   : 🔴 red
  NT   : 🔵 blue
  SA   : 🟢 green
  Q    : 🔴 red
  NSW  : 🔵 blue
  V    : 🔴 red
  T    : 🔴 red

✅ Valid 3-coloring found
```

### Sudoku Puzzle

**Path:** `examples/sudoku/`

A classic Sudoku puzzle demonstrating **FD solver** with AllDifferent constraints and complex constraint propagation.

**Run:**
```bash
cd examples/sudoku
go run main.go
```

**The Puzzle:**
Solve a 9x9 Sudoku puzzle where each row, column, and 3x3 box must contain all digits 1-9 exactly once.

**Features demonstrated:**
- **AllDifferent constraints**: Applied to rows, columns, and 3x3 boxes
- **Efficient constraint propagation**: Fast solving of complex constraint satisfaction problems
- **Pre-filled constraints**: Handling partially completed puzzles
- **Constraint composition**: Multiple overlapping AllDifferent constraints

**Sample Output:**
```
Solved in 1.102µs, found 1 solutions
5 3 4 6 7 8 9 1 2 
6 7 2 1 9 5 3 4 8 
1 9 8 3 4 2 5 6 7 
8 5 9 7 6 1 4 2 3 
4 2 6 8 5 3 7 9 1 
7 1 3 9 2 4 8 5 6 
9 6 1 5 3 7 2 8 4 
2 8 7 4 1 9 6 3 5 
3 4 5 2 8 6 1 7 9
```

### Among Constraint Demo

**Path:** `examples/among-demo/`

Demonstrates the Among global constraint that counts how many variables take values from a specified set.

**Run:**
```bash
cd examples/among-demo
go run main.go
```

**Features demonstrated:**
- **Among constraint**: Counting variables that take values from a set
- **Domain pruning**: Efficient propagation when count bounds are reached
- **Set membership**: Constraining variables to belong to specific value sets

**Sample Output:**
```
=== Among Constraint Demo (count in S) ===
x1: {1..2}
x2: {3}
x3: {3..4}
K: {2}
```

### Anytime Optimization

**Path:** `examples/anytime-optimization/`

Demonstrates **anytime optimization** where you can stop early and get valid (though possibly suboptimal) solutions.

**Run:**
```bash
cd examples/anytime-optimization
go run main.go
```

**Features demonstrated:**
- **Objective minimization**: Finding optimal solutions with limited resources
- **Node limits**: Stopping search early with valid solutions
- **Optimization comparison**: Comparing limited vs unlimited search

**Sample Output:**
```
=== Anytime Optimization Demo ===

Minimizing X + 2*Y with a node limit of 3...
✓ Optimal solution: objective = 3
  Solution: X=1, Y=1

Now solving without limit to confirm the true optimum...
✓ True optimum: objective = 3 (X=1, Y=1)

This demonstrates anytime optimization: you can stop early and
still get a valid (though possibly suboptimal) solution.
```

### Apartment Floor Puzzle (FD and Hybrid Variants)

**Path:** `examples/apartment_fd/` and `examples/apartment_hybrid/`

Alternative implementations of the apartment puzzle using pure FD solving and hybrid FD approaches.

**Run:**
```bash
cd examples/apartment_fd
go run main.go

# or hybrid version:
cd examples/apartment_hybrid  
go run main.go
```

**Features demonstrated:**
- **Pure FD constraints**: Modeling the apartment puzzle entirely with FD variables and constraints
- **Hybrid FD integration**: Combining multiple constraint solving approaches
- **Constraint equivalence**: Different modeling approaches for the same problem

**Sample Output:**
```
=== Apartment puzzle (FD variant) ===
baker: {1..4}
cooper: {2..4}
fletcher: {2..4}
miller: {3..5}
smith: {1..5}

Concrete assignment (after search):
Person    | Floor
----------|------
baker     | 1
cooper    | 3
fletcher  | 2
miller    | 4
smith     | 5
```

### Small TSP (Hamiltonian Cycle with Circuit)

**Path:** `examples/tsp-small/`

Demonstrates the new Circuit global constraint on a small symmetric TSP instance (n=5). Builds a successor permutation forming a single Hamiltonian cycle and enumerates tours, reporting the best cost found.

**Run:**
```bash
cd examples/tsp-small
go run main.go
```

**Features demonstrated:**
- Circuit global constraint: exactly-one successor and predecessor per node
- Subtour elimination using reified order constraints
- Enumerating and scoring Hamiltonian cycles; printing the best tour

**Sample Output:**
```
=== Small TSP with Circuit (n=5) ===
Found 12 unique tours. Best cost = 17
Best cycle: 1 -> 2 -> 5 -> 3 -> 4 -> 1
```

### Table Constraint Demo (Extensional Constraint)

**Path:** `examples/table-demo/`

Demonstrates the Table global constraint over enumerated variables by restricting (Color, Pet, Drink) tuples to a small allowed set and enumerating all satisfying assignments.

**Run:**
```bash
cd examples/table-demo
go run main.go
```

**Features demonstrated:**
- Table (extensional) constraint over multiple variables
- Enumerated domains with 1-based encoding
- Solving and printing all allowed tuples

**Sample Output:**
```
=== Table Constraint Demo (Color, Pet, Drink) ===
Found 4 solutions:
    Color=Red, Pet=Dog, Drink=Coffee
    Color=Green, Pet=Bird, Drink=Tea
    Color=Blue, Pet=Cat, Drink=Water
    Color=Blue, Pet=Dog, Drink=Tea
```

### Regular Constraint Demo (DFA pattern checker)

**Path:** `examples/regular-demo/`

Demonstrates the Regular global constraint using a simple DFA over the alphabet {A=1, B=2} that accepts exactly the strings that end with A. Shows strong pruning and enumerates all accepted sequences of fixed length.

**Run:**
```bash
cd examples/regular-demo
go run main.go
```

**Features demonstrated:**
- DFA-based Regular constraint with forward/backward filtering
- Enumerated symbols mapped to friendly names (A/B)
- Solving and printing all accepted sequences of a given length

**Sample Output:**
```
=== Regular Constraint Demo (DFA: ends with A) ===
Found 4 accepted sequences (length=3):
A A A
A B A
B A A
B B A
```

### Cumulative Constraint Demo (Resource scheduling)

**Path:** `examples/cumulative-demo/`

Demonstrates the Cumulative global constraint for renewable resource scheduling. Three tasks with fixed durations and demands share a resource with capacity 3; the demo enumerates feasible start-time assignments.

**Run:**
```bash
cd examples/cumulative-demo
go run main.go
```

**Features demonstrated:**
- Time-table filtering using compulsory parts
- Discrete time modeling with 1-based domains
- Enumerating and printing feasible schedules

**Sample Output:**
```
=== Cumulative Constraint Demo (capacity=3) ===
Found N feasible schedules (showing up to 50):
S1=1 S2=1 S3=1
...
```

### Global Cardinality (GCC) Demo

**Path:** `examples/gcc-demo/`

Demonstrates the GlobalCardinality constraint with per-value occurrence bounds. Three variables over {1,2,3}, with value 1 used exactly once and value 2 at most twice.

**Run:**
```bash
cd examples/gcc-demo
go run main.go
```

**Features demonstrated:**
- Per-value min/max occurrence bounds
- Pruning when a value's max count is saturated
- Enumerating and printing feasible assignments

**Sample Output:**
```
=== Global Cardinality Constraint Demo ===
Found N feasible assignments (showing up to 50):
X1=1 X2=2 X3=2
...
```

### Lexicographic Ordering Demo

**Path:** `examples/lex-demo/`

Demonstrates the Lexicographic ordering constraint with non-strict ordering X ≤lex Y over two-length vectors, showing bounds pruning on the first component.

**Run:**
```bash
cd examples/lex-demo
go run main.go
```

**Features demonstrated:**
- Bounds-consistent lexicographic pruning
- Symmetry breaking building block for sequences and permutations
- Prints pruned domains after propagation

**Sample Output:**
```
=== Lexicographic Constraint Demo (X ≤lex Y) ===
x1: {2..4}
x2: {1..3}
y1: {3..5}
y2: {2..4}
```

### Zebra Puzzle (Einstein's Riddle)

**Path:** `examples/zebra/`

A complete implementation of the famous logic puzzle demonstrating **idiomatic miniKanren** with relational helper functions.

**Run:**
```bash
cd examples/zebra
go run main.go
```

**Features demonstrated:**
- Complex constraint satisfaction with multiple variables
- Proper use of `Disj` (disjunction) for choice points
- Relational helper functions (`sameHouse`, `adjacent`, `leftOf`)
- **Idiomatic miniKanren**: constraints guide search rather than verify solutions
- `Neq` constraints for ensuring all values are different
- Solving a real-world logic puzzle efficiently

**Sample Output:**
```
House | Nationality | Color  | Pet    | Drink  | Smoke
------|-------------|--------|--------|--------|-------------
  1   | Norwegian   | yellow | cat    | water  | Dunhill
  2   | Dane        | blue   | horse  | tea    | Blend
  3   | English     | red    | bird   | milk   | Pall Mall
  4   | German      | green  | zebra  | coffee | Prince
  5   | Swede       | white  | dog    | beer   | Blue Master

🦓 Answer: The German owns the zebra!
```

### Apartment Floor Puzzle

**Path:** `examples/apartment/`

A constraint satisfaction problem about determining which floor each person lives on in a 5-floor apartment building.

**Run:**
```bash
cd examples/apartment
go run main.go
```

**The Puzzle:**
Five people (Baker, Cooper, Fletcher, Miller, and Smith) live on different floors of a 5-story building:
1. Baker does not live on the top floor
2. Cooper does not live on the bottom floor
3. Fletcher does not live on either the top or bottom floor
4. Miller lives on a higher floor than Cooper
5. Smith does not live on a floor adjacent to Fletcher's
6. Fletcher does not live on a floor adjacent to Cooper's

**Features demonstrated:**
- Floor assignment constraints with `validFloor`, `higherThan`, `notAdjacent`
- Using `Project` to extract values for **arithmetic comparisons**
- `allDiff` helper ensuring unique floor assignments
- Clear constraint modeling for spatial relationships
- **When to use Project**: extracting concrete values for host-language computation

**Sample Output:**
```
=== Solving the Apartment Floor Puzzle ===

✓ Solution found!

Person    | Floor
----------|------
Baker     | 3
Cooper    | 2
Fletcher  | 4
Miller    | 5
Smith     | 1

✅ All constraints satisfied!
```

### Twelve Statements Puzzle

**Path:** `examples/twelve-statements/`

A self-referential logic puzzle where twelve statements make claims about each other's truth values.

**Run:**
```bash
cd examples/twelve-statements
go run main.go
```

**The Puzzle:**
Given twelve statements that reference themselves and each other, determine which are true. For example:
- Statement 1: "This is a numbered list of twelve statements."
- Statement 2: "Exactly 3 of the last 6 statements are true."
- Statement 7: "Either statement 2 or 3 is true, but not both."

**Features demonstrated:**
- Self-referential constraint solving
- Using `Project` to verify complex interdependent constraints
- Boolean logic with implications and XOR
- Unique solution finding with `RunStar`

**Implementation note:**
This puzzle uses `Project` as a constraint verification oracle rather than pure relational programming. The self-referential nature and counting constraints ("exactly N are true") don't map naturally to miniKanren's relational model, so we enumerate the 2^12 boolean space and verify each assignment in Go. This demonstrates a pragmatic approach for constraint satisfaction problems that fall outside miniKanren's sweet spot. For more idiomatic relational examples, see the Zebra and Apartment puzzles.

**Sample Output:**
```
✓ Found 1 solution(s)!

TRUE statements:
 1. This is a numbered list of twelve statements.
 3. Exactly 2 of the even-numbered statements are true.
 4. If statement 5 is true, then statements 6 and 7 are both true.
 6. Exactly 4 of the odd-numbered statements are true.
 7. Either statement 2 or 3 is true, but not both.
11. Exactly 1 of statements 7, 8 and 9 are true.

✅ 6 true, 6 false
```

## Available Global Constraints

The gokanlogic library now includes a comprehensive set of global constraints that enable efficient solving of complex combinatorial problems:

### Implemented Global Constraints

- **Circuit**: Hamiltonian cycle constraints (single cycle visiting all nodes)
- **Table**: Extensional constraints (allowed tuples)
- **Regular**: DFA-based pattern constraints
- **Cumulative**: Resource scheduling constraints
- **GlobalCardinality (GCC)**: Per-value occurrence bounds
- **Among**: Count variables taking values from a set
- **Lexicographic**: Ordering constraints for sequences
- **AllDifferent**: Uniqueness constraints with efficient propagation
- **Element**: Indexing constraints (X[I] = V)
- **Sum**: Arithmetic sum constraints
- **Count**: Counting occurrences of values

### Constraint Composition

These constraints can be combined effectively to model complex problems. The examples demonstrate various combinations:
- **Sudoku**: Multiple AllDifferent constraints
- **TSP**: Circuit + arithmetic optimization
- **Scheduling**: Cumulative + precedence constraints
- **Assignment**: GCC + AllDifferent + capacity constraints

## Running the Examples

All examples are self-contained Go programs. To run any example:

1. Navigate to the example directory
2. Run with `go run main.go` or build with `go build`
3. The program will display the solution(s) found

### N-Queens Puzzle

**Path:** `examples/n-queens/`

The classic N-Queens problem: place N chess queens on an N×N board so no two queens attack each other.

**Run:**
```bash
cd examples/n-queens
go run main.go          # solves 6-queens (fast, ~228ms)
go run main.go 4        # solve for N=4
go run main.go 8        # solve for N=8 (slow, ~26s)
```

**The Problem:**
Place N queens on an N×N chessboard such that:
- No two queens share the same row (ensured by design)
- No two queens share the same column (enforced with `Neq`)
- No two queens share the same diagonal (checked with `Project`)

**Features demonstrated:**
- Backtracking search over combinatorial space
- Using `Neq` for column distinctness
- Using `Project` for diagonal checking (arithmetic)
- **Performance boundary**: Shows when miniKanren struggles without constraint propagation

**Performance:**
- N=4: ~1ms (fast)
- N=6: ~228ms (acceptable)
- N=8: ~26s (slow - demonstrates CLP(FD) would help)

**Sample Output:**
```
=== Solving the N-Queens Puzzle for N=6 ===

✓ Solution found in 228ms!

Board configuration:
  . . . Q . .   
  . Q . . . .   
  . . . . Q .   
  Q . . . . .   
  . . Q . . .   
  . . . . . Q   

Queens placed at columns: [4 2 5 1 3 6]
✅ All constraints satisfied!
```

## MiniKanren Idiomaticity Guide

Understanding when and why to deviate from pure relational programming helps you choose the right approach for your problem.

### What is "Idiomatic MiniKanren"?

Idiomatic miniKanren means writing **pure relational constraints** that:
- Use only declarative operators: `Eq`, `Neq`, `Disj`, `Conj`, `Fresh`
- State *what* the solution must satisfy, not *how* to find it
- Let constraints **guide the search** rather than **verify solutions**
- Avoid extracting values with `Project` for host-language computation

### Idiomaticity Spectrum

Our examples range from purely idiomatic to pragmatically mixed approaches:

#### ⭐⭐⭐⭐⭐ **Fully Idiomatic** (Graph Coloring, Zebra Puzzle)

**Graph Coloring** and **Zebra Puzzle** represent miniKanren at its best:
- **Pure relational constraints**: Only `Eq`, `Neq`, `Disj`, `Conj`
- **Constraint-driven**: `Neq(wa, nt)` means "WA and NT must differ" - the constraint IS the specification
- **No arithmetic**: All relationships are structural/symbolic
- **No verification**: Constraints actively guide search, not passively check results

**Why they're perfect for miniKanren:**
- Graph adjacency is naturally relational (symmetric, structural)
- Logic puzzle constraints are declarative statements
- No numeric computations required

#### ⭐⭐⭐ **Pragmatic** (Apartment Puzzle)

**Apartment** uses `Project` for arithmetic but stays mostly relational:

```go
higherThan := func(p1, p2 Term) Goal {
    return Project(List(p1, p2), func(vals []Term) Goal {
        floor1 := vals[0].(*Atom).Value().(int)
        floor2 := vals[1].(*Atom).Value().(int)
        if floor1 > floor2 {
            return Success
        }
        return Failure
    })
}
```

**Why deviate:** MiniKanren lacks built-in arithmetic constraints like `>`, `<`, `abs()`. Rather than enumerate all valid pairs, `Project` extracts values for a quick Go comparison.

**When this is acceptable:**
- The core problem is still relational (floor assignments, adjacency)
- Arithmetic is a small part of the constraint set
- Alternative would be verbose manual enumeration

#### ⭐⭐ **Mixed Approach** (N-Queens)

**N-Queens** uses relational constraints for columns but arithmetic for diagonals:

```go
// Idiomatic: column distinctness
Neq(col1, col2)

// Pragmatic: diagonal checking requires arithmetic
Project(List(col1, col2), func(vals []Term) Goal {
    c1, c2 := vals[0].(*Atom).Value().(int), vals[1].(*Atom).Value().(int)
    if abs(c1-c2) == abs(row1-row2) {
        return Failure  // On same diagonal
    }
    return Success
})
```

**Why deviate:** Diagonal constraints involve `abs()` and arithmetic differences. Pure relational encoding would require pre-computing all valid diagonal pairs.

**Trade-off:** Less idiomatic, but far more practical than encoding arithmetic relationally.

#### ⭐ **Verification Oracle** (Twelve Statements)

**Twelve Statements** uses miniKanren primarily for enumeration, with `Project` doing heavy verification:

```go
Project(s, func(vals []Term) Goal {
    // Extract all 12 boolean values
    // Verify complex interdependent logic in Go
    if statement1_implies_statement2 && exactly_N_true(...) {
        return Success
    }
    return Failure
})
```

**Why deviate:** Self-referential statements with counting constraints ("exactly 3 of the last 6 are true") don't map naturally to relational programming. The problem is inherently imperative.

**When this approach makes sense:**
- Small search space (2^12 = 4096 states)
- Constraints are deeply interdependent and numeric
- Verification logic is clearer in imperative code
- Alternative would be extremely verbose and unclear

### Choosing Your Approach

| Problem Type | Recommended Approach | Example |
|--------------|---------------------|---------|
| Symbolic/structural constraints | Pure relational (no `Project`) | Graph coloring, Zebra |
| Mostly symbolic + some arithmetic | Relational core + `Project` for math | Apartment |
| Mixed symbolic/numeric | Relational where possible, `Project` for computation | N-Queens |
| Numeric/counting/self-referential | `Project` verification oracle | Twelve Statements |
| Large combinatorial (no CLP) | Consider alternative tools | Sudoku (removed) |

### Performance Boundaries

MiniKanren excels at:
- ✅ Moderate search spaces with good constraint propagation
- ✅ Structural/symbolic relationships
- ✅ Problems where constraints prune the search effectively

FD Solver excels at:
- ✅ **Combinatorial optimization** with global constraints
- ✅ **Arithmetic constraint satisfaction** (sum, cardinality, scheduling)
- ✅ **Pattern matching** and sequence constraints
- ✅ **Resource allocation** problems
- ✅ **Graph problems** with specialized constraints (Circuit, Table)

Combined approaches struggle with:
- ❌ Very large combinatorial spaces without effective constraint propagation
- ❌ Problems requiring extensive floating-point computation
- ❌ Some highly specialized combinatorial problems (like complete knight's tours)

**Parallel Capabilities:**
- ✅ **Parallel search** with `ParallelRun` and `ParallelDisj` (see Graph Coloring)
- ✅ **Multi-core utilization** for constraint satisfaction problems
- ✅ **Parallel disjunction** for exploring multiple solution branches concurrently

**Performance Examples:**
- **Sudoku**: Solved in microseconds (~1.1µs)
- **Send+More=Money**: Fast solution finding
- **Magic Square**: Efficient 3x3 solution generation
- **N-Queens**: N=6 works well (~228ms), N=8 slower (~26s) due to combinatorial explosion
- **Graph Coloring**: Fast parallel solution finding
- **TSP Circuit**: Efficient Hamiltonian cycle enumeration

### Best Practices

1. **Start idiomatic**: Try pure relational first
2. **Use `Project` sparingly**: Only when arithmetic/imperative logic is truly needed
3. **Keep `Project` focused**: Extract minimal values, return quickly to relational constraints
4. **Document deviations**: Explain why `Project` is necessary (see Apartment, Twelve Statements examples)
5. **Consider alternatives**: If `Project` dominates, miniKanren may not be the right tool

The goal isn't purity for its own sake - it's using the right abstraction for your problem. Pure relational miniKanren is beautiful and powerful when it fits; pragmatic deviations are acceptable when they're the clearest solution.

## Creating Your Own Examples

Each example follows a similar pattern:

1. Import the miniKanren package
2. Define helper functions for constraints
3. Create a goal function that encodes the problem
4. Use `Run` or `RunStar` to find solutions
5. Display results in a user-friendly format

See the existing examples for reference implementations.
