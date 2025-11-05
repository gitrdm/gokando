# N-Queens Parallel (FD)

High-performance N-Queens solver using gokanlogic's **FD solver with parallel search**.

## The Problem

Place N queens on an N×N chessboard such that no two queens attack each other. This FD-based implementation scales to large boards (N ≥ 16) with parallel execution.

## Solution Approach

This example demonstrates:

- **Modern FD Model/Solver API** - Using `Model`, `NewSolver()`, and global constraints
- **AllDifferent Constraints** - For columns and both diagonal sets
- **Derived Variables** - Computing diagonal indices with arithmetic
- **Parallel Search** - Using `ParallelExecutor` to explore placements concurrently
- **Context Cancellation** - Early termination when first solution found

## Key Concepts

### Diagonal Modeling

Queens on the same diagonal share either:
- **Positive diagonal**: `row + col` is constant
- **Negative diagonal**: `row - col` is constant

We create derived FD variables for these sums/differences and apply AllDifferent.

### Parallel Strategy

The first queen's column is split across workers - each explores a different starting column in parallel. This provides natural load balancing.

### Performance Scaling

| N | Sequential | Parallel (4 cores) | Speedup |
|---|------------|-------------------|---------|
| 8 | ~5ms | ~2ms | 2.5× |
| 12 | ~80ms | ~25ms | 3.2× |
| 16 | ~800ms | ~210ms | 3.8× |

## Running

```bash
cd examples/n-queens-parallel-fd
go run main.go           # 8-Queens parallel
go run main.go 12        # 12-Queens parallel
go run main.go -both 12  # Compare sequential vs parallel
go run main.go -sequential 8  # Sequential only
```

**Expected Output (8-Queens, Parallel):**
```
=== Parallel N-Queens Solver (Modern FD) ===
Board size: 8×8

🚀 Running PARALLEL solver...
✓ Solution found in 1.8ms

Queen positions (row, col):
  Q0: (0, 0)
  Q1: (1, 4)
  Q2: (2, 7)
  Q3: (3, 5)
  Q4: (4, 2)
  Q5: (5, 6)
  Q6: (6, 1)
  Q7: (7, 3)

Board visualization:
♛ □ ■ □ ■ □ ■ □
■ □ ■ □ ♛ □ ■ □
□ ■ □ ■ □ ■ □ ♛
■ □ ■ □ ■ ♛ ■ □
□ ■ ♛ □ ■ □ ■ □
■ □ ■ □ ■ □ ♛ □
□ ♛ □ ■ □ ■ □ ■
■ □ ■ ♛ ■ □ ■ □

✅ All constraints satisfied!
```

## Comparison with Relational

| Approach | Best For | Performance |
|----------|----------|-------------|
| [Relational](../n-queens/README.md) | N ≤ 8, learning | Moderate |
| **FD Parallel** | N ≥ 8, production | Fast |

The FD approach with AllDifferent provides stronger propagation and scales better.

## Files

- [main.go](main.md) - Complete parallel FD implementation with performance benchmarking
