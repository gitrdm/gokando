# Apartment Floor Puzzle

A classic logic puzzle solved using gokanlogic's **finite domain (FD) constraint solver**.

## The Puzzle

Baker, Cooper, Fletcher, Miller, and Smith live on different floors of a five-story apartment building. Where does everyone live?

### Constraints

- Baker does not live on the top floor
- Cooper does not live on the bottom floor
- Fletcher does not live on either the top or the bottom floor
- Miller lives on a higher floor than Cooper
- Smith does not live on a floor adjacent to Fletcher's
- Fletcher does not live on a floor adjacent to Cooper's

## Solution Approach

This example demonstrates:

- **FD Variables** - Each person is assigned a variable with domain {1, 2, 3, 4, 5} representing floors
- **AllDifferent Constraint** - No two people live on the same floor
- **Inequality Constraints** - Enforcing relative floor positions
- **Composite Constraints** - Building adjacency constraints from primitives
- **High-Level API** - Using `Model`, `NewSolver()`, and `Solve()`

## Key Concepts

**Custom Constraint Building**: The `notAdjacentConstraint()` function shows how to compose multiple primitive constraints into a reusable pattern.

**FD vs Relational**: This puzzle is well-suited for FD solving because it involves numeric ranges and ordering constraints.

## Running

```bash
cd examples/apartment
go run main.go
```

**Expected Output:**
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

## Files

- [main.go](main.md) - Complete source code with detailed comments
