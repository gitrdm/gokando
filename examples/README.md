# GoKanDo Examples

This directory contains complete example programs demonstrating various features of the GoKanDo miniKanren implementation.

## Examples

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

## Example Categories

The examples demonstrate different approaches and capabilities of miniKanren:

1. **Graph Coloring** - Parallel search with `ParallelDisj` (demonstrates concurrency features)
2. **Zebra Puzzle** - Pure relational programming with helper functions (idiomatic miniKanren)
3. **Apartment Puzzle** - Using `Project` for arithmetic comparisons (pragmatic miniKanren)
4. **Twelve Statements** - Using `Project` for constraint verification (constraint satisfaction)
5. **N-Queens** - Backtracking search (shows performance boundaries without CLP(FD))

Each approach is suited to different problem types:
- **Best for miniKanren**: Relational/structural constraints (Zebra, Graph Coloring)
- **Pragmatic with Project**: Spatial/arithmetic comparisons (Apartment, N-Queens)
- **Verification approach**: Self-referential/counting constraints (Twelve Statements)
- **Performance limits**: Large combinatorial spaces without constraint propagation (N-Queens N>6)

## Creating Your Own Examples

Each example follows a similar pattern:

1. Import the miniKanren package
2. Define helper functions for constraints
3. Create a goal function that encodes the problem
4. Use `Run` or `RunStar` to find solutions
5. Display results in a user-friendly format

See the existing examples for reference implementations.
