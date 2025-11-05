# N-Queens Puzzle (Relational)

The classic N-Queens puzzle solved using gokanlogic's **relational (miniKanren) solver**.

## The Problem

Place N queens on an N×N chessboard such that no two queens can attack each other. Queens attack any piece on the same row, column, or diagonal.

### Constraints

- Each queen must be in a different column
- Each queen must be in a different row (implicit in representation)
- No two queens can be on the same diagonal

## Solution Approach

This example demonstrates:

- **Relational Solving** - Using logic variables and constraint propagation
- **Fresh Variables** - Creating logic variables for queen positions
- **Disjunction** - Enumerating possible column choices
- **Inequality** - Ensuring queens are in different columns (`Neq`)
- **Project** - Extracting and verifying diagonal constraints
- **High-Level API** - `Solutions()`, `A()`, `Fresh()`, `Eq()`, `Neq()`

## Key Concepts

**Representation**: Queens are represented as a list of column positions, where the index is the row number. For example, `[1, 3, 0, 2]` means:
- Row 0: Queen in column 1
- Row 1: Queen in column 3
- Row 2: Queen in column 0
- Row 3: Queen in column 2

**Project for Complex Constraints**: The diagonal check uses `Project()` to extract ground values and verify them with a custom function.

**Relational vs FD**: This relational approach works well for small N (≤8). For larger boards or parallel solving, see [n-queens-parallel-fd](../n-queens-parallel-fd/README.md).

## Running

```bash
cd examples/n-queens
go run main.go      # Solves 6-queens (default)
go run main.go 8    # Solves 8-queens
go run main.go 4    # Solves 4-queens
```

**Expected Output (6-Queens):**
```
=== Solving the 6-Queens Puzzle ===

✓ Solution found for 6 queens!

♛ ■ □ ■ □ ■
□ ■ □ ■ ♛ ■
■ □ ■ ♛ ■ □
■ ♛ ■ □ ■ □
□ ■ □ ■ □ ♛
□ ■ ♛ ■ □ ■

Queen positions (row, col): (0,1), (1,3), (2,5), (3,0), (4,2), (5,4)
```

## Performance

- **4-Queens**: ~1ms
- **6-Queens**: ~5ms
- **8-Queens**: ~50ms

For N > 8, consider the FD-based parallel version.

## Files

- [main.go](main.md) - Complete relational implementation with visual board display
