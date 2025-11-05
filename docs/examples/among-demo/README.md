# Among Constraint Demo

Demonstrates the **Among** global constraint for counting values in a set.

## What is Among?

The Among constraint counts how many variables from a list take values from a specified set `S`:

```
Among([x1, x2, x3], S={1, 2}, K) means:
  K = number of variables in {x1, x2, x3} with value in {1, 2}
```

## Example

Given:
- `x1 ∈ {1, 2}`
- `x2 ∈ {2, 3}`
- `x3 ∈ {3, 4}`
- `S = {1, 2}`
- `K = 1` (exactly 1 variable must be in S)

The constraint forces exactly one variable to take a value from S.

## Use Cases

- Resource allocation (count items meeting criteria)
- Scheduling (count events in time windows)
- Configuration (limit features enabled)

## Running

```bash
cd examples/among-demo
go run main.go
```

## Files

- [main.go](main.md) - Complete demonstration with domain propagation
