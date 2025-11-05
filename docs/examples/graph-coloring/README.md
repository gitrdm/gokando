# Graph Coloring - Map of Australia

A classic constraint satisfaction problem demonstrating **relational solving** with **parallel search** in gokanlogic.

## The Problem

Color the regions of a map such that no two adjacent regions share the same color, using the minimum number of colors possible.

### Map of Australia

- **7 Regions**: WA, NT, SA, Q, NSW, V, T (Tasmania)
- **Goal**: 3-color the map (red, green, blue)
- **Challenge**: South Australia (SA) touches 5 other regions - a constraint bottleneck

### Adjacencies

```
WA:  NT, SA
NT:  WA, SA, Q
SA:  WA, NT, Q, NSW, V  (most connected!)
Q:   NT, SA, NSW
NSW: Q, SA, V
V:   SA, NSW
T:   (island - no adjacencies)
```

## Solution Approach

This example demonstrates:

- **Relational HLAPI** - Using `Fresh()`, `A()`, `Eq()`, `Neq()`, `Disj()`, `Conj()`
- **Parallel Search** - `ParallelDisj()` and `ParallelRun()` for concurrent exploration
- **Performance Comparison** - Sequential vs parallel solving
- **Term Sugar** - `A("red")` for atoms, `L()` for lists
- **Structured Results** - `Rows()` to extract solutions as tables

## Key Concepts

**Parallel vs Sequential**: Run with different modes to compare performance:
- **Parallel mode** (default): Uses `ParallelDisj()` to explore color choices concurrently
- **Sequential mode**: Standard depth-first search

**Relational Constraints**: Unlike FD solving, this uses pure logic programming with symbolic values and inequality constraints.

## Running

```bash
# Parallel search (default)
cd examples/graph-coloring
go run main.go

# Sequential search for comparison
go run main.go seq
```

**Expected Output:**
```
=== Graph Coloring: Map of Australia ===

Regions to color:
  WA (Western Australia)
  NT (Northern Territory)
  ...

🚀 Using PARALLEL search with ParallelRun and ParallelDisj

✓ Solution found in 12.3ms!

Region Coloring:
  WA   : 🔴 red
  NT   : 🟢 green
  SA   : 🔵 blue
  Q    : 🔴 red
  NSW  : 🟢 green
  V    : 🔴 red
  T    : 🔴 red

✅ Valid 3-coloring found
```

## Files

- [main.go](main.md) - Complete source with both sequential and parallel implementations
