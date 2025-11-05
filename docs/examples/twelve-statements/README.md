# Twelve Statements Puzzle

A self-referential logic puzzle solved using gokanlogic's **FD solver with reification and Table constraints**.

## The Puzzle

Given twelve statements about themselves, determine which statements are true:

1. This is a numbered list of twelve statements
2. Exactly 3 of the last 6 statements are true
3. Exactly 2 of the even-numbered statements are true
4. If statement 5 is true, then statements 6 and 7 are both true
5. The 3 preceding statements are all false
6. Exactly 4 of the odd-numbered statements are true
7. Either statement 2 or 3 is true, but not both
8. If statement 7 is true, then 5 and 6 are both true
9. Exactly 3 of the first 6 statements are true
10. The next two statements are both true
11. Exactly 1 of statements 7, 8 and 9 are true
12. Exactly 4 of the preceding statements are true

## Solution

**True statements**: 1, 3, 4, 6, 7, 11
**False statements**: 2, 5, 8, 9, 10, 12

## Solution Approach

This example demonstrates advanced FD techniques:

- **Boolean Encoding** - Statements as FD variables with domain {1=false, 2=true}
- **BoolSum Constraint** - Counting true statements efficiently
- **Reification** - Linking statement truth to constraint satisfaction
- **Table Constraints** - Encoding logical operators (implication, XOR, AND)
- **Self-Reference** - Statements that refer to other statements

## Key Concepts

### Reification

Reification links a boolean variable to whether a constraint holds:
```go
// S2 is true iff exactly 3 of last 6 statements are true
S2 ↔ (count(S7..S12) == 3)
```

### Counting with BoolSum

`BoolSum` efficiently counts how many variables are set to `true` (encoded as 2):
```go
boolSum, _ := mk.NewBoolSum([]*FDVariable{S7, S8, S9, S10, S11, S12}, count)
```

### Logical Operators via Table

Implication, XOR, and AND are encoded as small truth tables:
```go
// S4: If S5 then (S6 and S7)
// Table: [(s5, s6, s7, s4), ...]
table := [][]int{
    {1, 1, 1, 2},  // false → _ = true (vacuous truth)
    {1, 1, 2, 2},
    {1, 2, 1, 2},
    {1, 2, 2, 2},
    {2, 2, 2, 2},  // true → true ∧ true = true
    {2, 1, _, 1},  // true → false = false
    {2, _, 1, 1},
}
```

## Running

```bash
cd examples/twelve-statements
go run main.go
```

**Expected Output:**
```
=== Solving the Twelve Statements Puzzle ===

✓ Solution found in 12ms!

Statement Results:
  1: ✓ TRUE  (This is a numbered list of twelve statements)
  2: ✗ FALSE (Exactly 3 of the last 6 statements are true)
  3: ✓ TRUE  (Exactly 2 of the even-numbered statements are true)
  4: ✓ TRUE  (If statement 5 is true, then 6 and 7 are both true)
  5: ✗ FALSE (The 3 preceding statements are all false)
  6: ✓ TRUE  (Exactly 4 of the odd-numbered statements are true)
  7: ✓ TRUE  (Either statement 2 or 3 is true, but not both)
  8: ✗ FALSE (If statement 7 is true, then 5 and 6 are both true)
  9: ✗ FALSE (Exactly 3 of the first 6 statements are true)
 10: ✗ FALSE (The next two statements are both true)
 11: ✓ TRUE  (Exactly 1 of statements 7, 8 and 9 are true)
 12: ✗ FALSE (Exactly 4 of the preceding statements are true)

✅ All constraints satisfied!
True statements: 1, 3, 4, 6, 7, 11
```

## Performance

Typical solve time: **10-15ms**

The small search space (2^12 possibilities pruned heavily by propagation) makes this very fast.

## Files

- [main.go](main.md) - FD implementation with reification and Table constraints
