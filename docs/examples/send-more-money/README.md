# SEND + MORE = MONEY Cryptarithm

A classic cryptarithmetic puzzle solved using gokanlogic's **FD solver with Table constraints**.

## The Puzzle

```
    S E N D
  + M O R E
  ---------
  M O N E Y
```

Each letter represents a unique digit (0-9). Find the digit assignment that makes the addition correct.

### Constraints

- All letters represent different digits
- S and M cannot be 0 (leading digits)
- The arithmetic must be valid: SEND + MORE = MONEY

## Solution

```
    9 5 6 7
  + 1 0 8 5
  ---------
  1 0 6 5 2
```

**Answer**: S=9, E=5, N=6, D=7, M=1, O=0, R=8, Y=2

## Solution Approach

This example demonstrates:

- **FD Encoding** - Digits 0-9 mapped to domain values 1-10
- **AllDifferent** - Ensuring each letter gets a unique digit
- **Table Constraints** - Modeling column addition with carry logic
- **Domain Restrictions** - S and M cannot be 0 (domains 2-10)
- **Carry Variables** - Intermediate boolean variables for arithmetic

## Key Concepts

**Table Constraints for Arithmetic**: Each column addition (x + y + carry_in = result + 10*carry_out) is encoded as a table of valid tuples. This is more efficient than decomposing into primitive constraints.

**Encoding Trick**: Digits 0-9 are encoded as FD values 1-10 to avoid special handling of 0. The mapping is `FD_value = digit + 1`.

**Leading Digit Optimization**: Since M must be 1 (the only way SEND + MORE can produce a 5-digit result), we can fix M=1 upfront.

## Running

```bash
cd examples/send-more-money
go run main.go
```

**Expected Output:**
```
=== FD SEND + MORE = MONEY ===

✓ Solved in 45ms

S=9 E=5 N=6 D=7
M=1 O=0 R=8 Y=2

Verification:
    9567
  + 1085
  -------
   10652 ✓

✅ Solution correct!
```

## Performance

Typical solve time: **40-50ms** on modern hardware.

The table-based approach is significantly faster than decomposing into arithmetic primitives.

## Files

- [main.go](main.md) - Complete FD implementation with Table constraints
