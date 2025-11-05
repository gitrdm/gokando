# Sudoku Solver

Solve 9×9 Sudoku puzzles using gokanlogic's **low-level FD solver** with AllDifferent constraints.

## The Problem

Fill a 9×9 grid so that each row, column, and 3×3 block contains the digits 1-9 exactly once.

### Example Puzzle

```
5 3 _ | _ 7 _ | _ _ _
6 _ _ | 1 9 5 | _ _ _
_ 9 8 | _ _ _ | _ 6 _
------+-------+------
8 _ _ | _ 6 _ | _ _ 3
4 _ _ | 8 _ 3 | _ _ 1
7 _ _ | _ 2 _ | _ _ 6
------+-------+------
_ 6 _ | _ _ _ | 2 8 _
_ _ _ | 4 1 9 | _ _ 5
_ _ _ | _ 8 _ | _ 7 9
```

## Solution Approach

This example demonstrates:

- **Low-Level FD API** - Direct use of `FDStore`, `FDVar`, and constraint primitives
- **Variable Creation** - 81 FD variables for the grid cells
- **Givens** - Pre-assigning known values with `Assign()`
- **AllDifferent Constraints** - 27 constraints (9 rows + 9 columns + 9 blocks)
- **Efficient Propagation** - The FD engine handles constraint propagation automatically

## Key Concepts

**AllDifferent Constraint**: The core of Sudoku solving. Each row, column, and 3×3 block gets an AllDifferent constraint ensuring all 9 values are unique.

**Low-Level vs High-Level**: This example uses the low-level `FDStore` API directly, showing the underlying mechanics without the `Model` abstraction.

**Constraint Propagation**: Once givens are assigned and constraints added, the FD solver uses propagation to reduce domains before search.

## Running

```bash
cd examples/sudoku
go run main.go
```

**Expected Output:**
```
Solved in 23.4ms, found 1 solutions

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

## Performance

Typical solve time: **20-30ms** for medium-difficulty puzzles.

Hard puzzles with fewer givens may take longer as they require more search.

## Puzzle Format

Puzzles are encoded as arrays of 81 integers:
- `0` = empty cell
- `1-9` = given digit

Row-major order: `[row0_col0, row0_col1, ..., row8_col8]`

## Files

- [main.go](main.md) - Low-level FD implementation showing constraint setup
