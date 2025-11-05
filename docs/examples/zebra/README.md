# Zebra Puzzle (Einstein's Riddle)

The famous logic puzzle solved using gokanlogic's **relational solver** with symbolic reasoning.

## The Puzzle

There are five houses in a row, each with different attributes. Using the clues below, determine: **Who owns the zebra?**

### The 15 Clues

1. There are five houses
2. The English man lives in the red house
3. The Swede has a dog
4. The Dane drinks tea
5. The green house is immediately to the left of the white house
6. They drink coffee in the green house
7. The man who smokes Pall Mall has birds
8. In the yellow house they smoke Dunhill
9. In the middle house they drink milk
10. The Norwegian lives in the first house
11. The Blend smoker lives next to the house with cats
12. In the house next to the horse they smoke Dunhill
13. The Blue Master smoker drinks beer
14. The German smokes Prince
15. The Norwegian lives next to the blue house
16. They drink water next to the house where they smoke Blend

## Solution

**The German owns the zebra!**

### Complete Solution

| House | 1st      | 2nd     | 3rd    | 4th   | 5th    |
|-------|----------|---------|--------|-------|--------|
| Color | Yellow   | Blue    | Red    | Green | White  |
| Nation| Norwegian| Dane    | English| German| Swede  |
| Pet   | Cats     | Horse   | Birds  | Zebra | Dog    |
| Drink | Water    | Tea     | Milk   | Coffee| Beer   |
| Smoke | Dunhill  | Blend   | Pall Mall| Prince| Blue Master|

## Solution Approach

This example demonstrates:

- **Relational Reasoning** - Using logic variables for symbolic attributes
- **List Representation** - Each house is a list of 5 attributes
- **Membership Constraints** - Using `Membero` to constrain attribute values
- **Positional Constraints** - "next to", "left of", "at position X"
- **Run** - Finding solutions with the `Run(n, goal)` API

## Key Concepts

**Symbolic Reasoning**: Unlike FD solving with numbers, this uses atoms like `"red"`, `"english"`, `"dog"` as constraint values.

**Adjacency Constraints**: Helper functions like `nextTo()` and `toLeftOf()` encode spatial relationships between houses.

**Relational Power**: This puzzle is naturally expressed in miniKanren's relational style, making the code closely mirror the problem statement.

## Running

```bash
cd examples/zebra
go run main.go
```

**Expected Output:**
```
=== Solving the Zebra Puzzle with gokanlogic ===

✓ Solution found!

House 1: Yellow Norwegian Cats Water Dunhill
House 2: Blue Dane Horse Tea Blend
House 3: Red English Birds Milk Pall-Mall
House 4: Green German Zebra Coffee Prince
House 5: White Swede Dog Beer Blue-Master

🦓 The German owns the zebra!
```

## Performance

Typical solve time: **100-200ms**

The search space is large (5! choices for each of 5 attributes), but constraint propagation prunes it effectively.

## Files

- [main.go](main.md) - Complete relational implementation with helper predicates
