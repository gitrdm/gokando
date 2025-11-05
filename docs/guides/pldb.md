# Relational Database (pldb) Guide

This guide covers gokanlogic's relational database system (pldb), which provides efficient in-memory fact storage and querying with indexed access. The pldb system integrates seamlessly with miniKanren goals and SLG tabling for recursive queries.

## Overview

The pldb (Prolog-like database) module enables logic programming over structured data. It's useful for:

- **Family trees and genealogy**: Define parent/child relationships, query for ancestors
- **Graph databases**: Store edges, query for reachable nodes and paths
- **Rule-based systems**: Define facts and derive conclusions through queries
- **Datalog-style joins**: Combine multiple relations with shared variables

## Core Concepts

### Relations

A **relation** defines a named predicate with fixed arity (number of arguments) and optional indexes:

```go
// Define a binary "parent" relation, indexed on both positions
parent, err := DbRel("parent", 2, 0, 1)
if err != nil {
    // Handle error (invalid arity or index)
}
```

**Indexing**: Specify which positions (0-based) should be indexed for fast lookups. Indexed positions enable O(1) hash-based lookups instead of O(n) scans.

### Database

A **Database** stores facts (tuples) for multiple relations:

```go
db := NewDatabase()
```

Databases are **immutable** (copy-on-write). Operations like `AddFact` and `RemoveFact` return new database instances:

```go
db2, err := db.AddFact(parent, NewAtom("alice"), NewAtom("bob"))
db3, err := db2.AddFact(parent, NewAtom("bob"), NewAtom("charlie"))
```

### Facts

Facts are ground (variable-free) tuples stored in relations. Each fact must:
- Match the relation's arity
- Contain only ground terms (no logic variables)

```go
// Valid facts
db, _ = db.AddFact(parent, NewAtom("alice"), NewAtom("bob"))
db, _ = db.AddFact(edge, NewAtom("a"), NewAtom("b"))

// Invalid: wrong arity
db, err = db.AddFact(parent, NewAtom("alice"))  // Error!

// Invalid: contains variable
x := Fresh("x")
db, err = db.AddFact(parent, NewAtom("alice"), x)  // Error!
```

## Basic Usage

### Defining Relations and Facts

```go
package main

import (
    "context"
    "fmt"
    . "github.com/gitrdm/gokanlogic/pkg/minikanren"
)

func main() {
    // Define binary parent relation with indexes on both positions
    parent, _ := DbRel("parent", 2, 0, 1)
    
    // Create database and add facts
    db := NewDatabase()
    db, _ = db.AddFact(parent, NewAtom("alice"), NewAtom("bob"))
    db, _ = db.AddFact(parent, NewAtom("alice"), NewAtom("charlie"))
    db, _ = db.AddFact(parent, NewAtom("bob"), NewAtom("diana"))
    
    // Query: who are Alice's children?
    child := Fresh("child")
    goal := db.Query(parent, NewAtom("alice"), child)
    
    ctx := context.Background()
    store := NewLocalConstraintStore(NewGlobalConstraintBus())
    stream := goal(ctx, store)
    
    results, _ := stream.Take(10)
    for _, s := range results {
        binding := s.GetBinding(child.ID())
        fmt.Printf("Alice's child: %v\n", binding)
    }
    // Output:
    // Alice's child: bob
    // Alice's child: charlie
}
```

### Query Patterns

Queries use a mix of ground terms and logic variables:

```go
// All children of Alice
db.Query(parent, NewAtom("alice"), child)

// All parents of Bob
db.Query(parent, p, NewAtom("bob"))

// All parent-child pairs
db.Query(parent, p, c)

// Check if specific fact exists
db.Query(parent, NewAtom("alice"), NewAtom("bob"))
```

### Repeated Variables

Variables appearing multiple times in a pattern enforce equality:

```go
// Self-loops: nodes with edges to themselves
db.Query(edge, x, x)

// This is equivalent to:
Conj(
    db.Query(edge, x, y),
    Eq(x, y),
)
```

## Integration with miniKanren

Database queries return `Goal` functions that compose with standard miniKanren operators:

### Conjunction (Joins)

```go
// Grandparent query
grandparent := Fresh("gp")
grandchild := Fresh("gc")
parent := Fresh("p")

goal := Conj(
    db.Query(parentRel, grandparent, parent),    // gp is parent of p
    db.Query(parentRel, parent, grandchild),     // p is parent of gc
)

// This effectively joins on the shared variable "parent"
```

### Disjunction (Union)

```go
// Either a parent or a sibling
goal := Disj(
    db.Query(parent, x, y),
    db.Query(sibling, x, y),
)
```

### Negation

```go
// Nodes with no outgoing edges (using WFS negation)
goal := Conj(
    db.Query(node, x),
    NegateEvaluator(engine, "no-edges", 
        edgePattern, 
        QueryEvaluator(db.Query(edge, x, Fresh("_")), ...)),
)
```

## SLG Tabling Integration

For recursive queries, combine pldb with SLG tabling to ensure termination and efficient fixpoint computation.

### TabledQuery

`TabledQuery` wraps a database query with tabling:

```go
// Define edge relation
edge, _ := DbRel("edge", 2, 0, 1)
db := NewDatabase()
db, _ = db.AddFact(edge, NewAtom("a"), NewAtom("b"))
db, _ = db.AddFact(edge, NewAtom("b"), NewAtom("c"))
db, _ = db.AddFact(edge, NewAtom("c"), NewAtom("d"))

// Non-recursive query (could use regular Query)
x := Fresh("x")
y := Fresh("y")
goal := TabledQuery(db, edge, "edge", x, y)
```

### RecursiveRule

`RecursiveRule` simplifies defining recursive predicates with base and recursive cases:

```go
// Transitive closure: path(X,Y) :- edge(X,Y) | edge(X,Z), path(Z,Y)
path := RecursiveRule(
    db,
    edge,                    // base relation
    "path",                  // predicate ID
    func(x, y Term) Goal {   // recursive case
        z := Fresh("z")
        return Conj(
            TabledQuery(db, edge, "path", x, z),
            TabledQuery(db, edge, "path", z, y),
        )
    },
)

// Query: all nodes reachable from "a"
goal := path(NewAtom("a"), Fresh("target"))
```

**Important**: Use the same predicate ID throughout the recursion (e.g., "path") to enable proper tabling and fixpoint computation.

### TabledDatabase Wrapper

For convenience, wrap an entire database to automatically table all queries:

```go
// Wrap database for automatic tabling
tabled := WithTabledDatabase(db)

// All queries now use tabling
goal := tabled.Query(edge, x, y)

// Mutations create new tabled database instances
tabled2, _ := tabled.AddFact(edge, NewAtom("d"), NewAtom("e"))

// Invalidate cache after mutations
InvalidateAll()  // or InvalidateRelation("edge")
```

## Performance Considerations

### Indexing Strategy

**When to index**:
- Index positions used in ground queries (e.g., `Query(parent, NewAtom("alice"), x)` benefits from index on position 0)
- Index all positions for symmetric relations (e.g., edges in undirected graphs)
- Skip indexes for write-heavy workloads (indexes add overhead on insertion)

**Index performance**:
- Indexed lookups: O(1) hash lookup + O(k) where k = matching facts
- Non-indexed lookups: O(n) linear scan over all facts

Example:
```go
// For queries like: parent(alice, X)
// Index position 0 (parent)
parent, _ := DbRel("parent", 2, 0)

// For queries like: parent(X, bob) or parent(alice, X)
// Index both positions
parent, _ := DbRel("parent", 2, 0, 1)
```

### Large Datasets

For 10k+ facts:
- **Always use indexes** on queried positions
- Consider selective indexing if only certain positions are queried
- Use `AllFacts()` sparingly (returns all facts, expensive for large relations)

Benchmarks (from tests):
- Indexed lookup on 10k facts: ~0.01ms per query
- Full scan on 10k facts: ~5ms per query
- 500x speedup with proper indexing

### Tabling Performance

Tabling adds overhead for the first query but caches results:
- **First query**: Evaluates and caches (~10-100µs overhead)
- **Subsequent queries**: Cache hit (~1µs, instant)
- **Recursive queries**: Avoids infinite loops, computes fixpoint efficiently

Use tabling when:
- Query is recursive (transitive closure, graph reachability)
- Same query is executed multiple times
- Query is expensive (large joins, deep recursion)

Don't use tabling when:
- Query is simple and only executed once
- Database is frequently mutated (cache invalidation overhead)
- Memory is constrained (tabling stores all answers)

## When to Use pldb vs. Constraints

### Use pldb when:

- **Working with discrete facts**: Parent-child relationships, graph edges, symbolic data
- **Queries dominate**: Read-heavy workloads with occasional updates
- **Recursive relationships**: Family trees, graphs, taxonomies
- **Datalog-style logic**: Rule-based reasoning, deductive databases

### Use FD constraints when:

- **Numeric domains**: Variables range over integers (e.g., 1..100)
- **Optimization**: Minimize/maximize objective functions
- **Propagation**: Constraint violations prune search space
- **Scheduling/planning**: Resource allocation, time-based reasoning

### Hybrid approach:

Combine both for maximum expressiveness:
```go
// Use pldb for graph structure
db.Query(edge, x, y)

// Use FD for numeric properties
InDomain(x, 1, 10)
LessThan(x, y)
```

## Common Patterns

### Family Tree

```go
parent, _ := DbRel("parent", 2, 0, 1)
male, _ := DbRel("male", 1, 0)
female, _ := DbRel("female", 1, 0)

db := NewDatabase()
db, _ = db.AddFact(parent, NewAtom("john"), NewAtom("mary"))
db, _ = db.AddFact(parent, NewAtom("john"), NewAtom("tom"))
db, _ = db.AddFact(male, NewAtom("john"))
db, _ = db.AddFact(male, NewAtom("tom"))
db, _ = db.AddFact(female, NewAtom("mary"))

// Father: parent + male
father := func(f, c Term) Goal {
    return Conj(
        db.Query(parent, f, c),
        db.Query(male, f),
    )
}

// Grandfather with tabling
grandfather := RecursiveRule(
    db, parent, "grandfather",
    func(gf, gc Term) Goal {
        p := Fresh("p")
        return Conj(
            father(gf, p),
            db.Query(parent, p, gc),
        )
    },
)
```

### Graph Reachability

```go
edge, _ := DbRel("edge", 2, 0, 1)
db := NewDatabase()
// Add edges...

// Reachable nodes (transitive closure)
reachable := RecursiveRule(
    db, edge, "reachable",
    func(from, to Term) Goal {
        mid := Fresh("mid")
        return Conj(
            TabledQuery(db, edge, "reachable", from, mid),
            TabledQuery(db, edge, "reachable", mid, to),
        )
    },
)

// Find all nodes reachable from "start"
goal := reachable(NewAtom("start"), Fresh("destination"))
```

### Datalog Rules

```go
// Base facts
employee, _ := DbRel("employee", 2, 0, 1)  // (person, department)
manager, _ := DbRel("manager", 2, 0, 1)    // (person, person)

// Derived: indirect_manager(X, Y) :- manager(X, Z), indirect_manager(Z, Y)
indirectManager := RecursiveRule(
    db, manager, "indirect_manager",
    func(x, y Term) Goal {
        z := Fresh("z")
        return Conj(
            TabledQuery(db, manager, "indirect_manager", x, z),
            TabledQuery(db, manager, "indirect_manager", z, y),
        )
    },
)

// Query: all indirect managers of alice
goal := indirectManager(Fresh("boss"), NewAtom("alice"))
```

## Advanced Features

### Copy-on-Write Semantics

Databases are immutable. Mutations create new instances:

```go
db1 := NewDatabase()
db2, _ := db1.AddFact(parent, NewAtom("alice"), NewAtom("bob"))
db3, _ := db2.AddFact(parent, NewAtom("bob"), NewAtom("charlie"))

// db1, db2, db3 are all independent
// db1 has 0 facts, db2 has 1, db3 has 2
```

### Tombstone Semantics

Removed facts are marked as tombstones (not deleted):

```go
db, _ = db.AddFact(edge, NewAtom("a"), NewAtom("b"))
db, _ = db.RemoveFact(edge, NewAtom("a"), NewAtom("b"))
db, _ = db.AddFact(edge, NewAtom("a"), NewAtom("b"))  // Re-add is OK
```

Tombstones are filtered from query results but remain in internal storage for versioning.

### Tabling Cache Invalidation

After database mutations, invalidate the tabling cache:

```go
db, _ = db.AddFact(edge, NewAtom("x"), NewAtom("y"))
InvalidateRelation("path")  // Invalidate specific predicate
// or
InvalidateAll()  // Invalidate all tabled queries
```

## Error Handling

Common errors:

```go
// Arity mismatch
rel, err := DbRel("parent", 2, 0, 1)
db, err = db.AddFact(rel, NewAtom("alice"))  // Error: expected 2 args

// Index out of range
rel, err := DbRel("parent", 2, 0, 5)  // Error: index 5 >= arity 2

// Non-ground term
x := Fresh("x")
db, err = db.AddFact(rel, NewAtom("alice"), x)  // Error: variables not allowed

// Nil relation
db, err = db.AddFact(nil, NewAtom("a"))  // Error: nil relation
```

## API Reference

### Types

```go
type Relation struct { /* name, arity, indexes */ }
type Database struct { /* immutable fact store */ }
```

### Functions

```go
// Create relation with indexes
func DbRel(name string, arity int, indexes ...int) (*Relation, error)

// Create database
func NewDatabase() *Database

// Database operations (return new instance)
func (db *Database) AddFact(rel *Relation, terms ...Term) (*Database, error)
func (db *Database) RemoveFact(rel *Relation, terms ...Term) (*Database, error)
func (db *Database) AllFacts(rel *Relation) []Fact

// Query (returns Goal)
func (db *Database) Query(rel *Relation, pattern ...Term) Goal

// Tabling integration
func TabledQuery(db *Database, rel *Relation, predicateID string, args ...Term) Goal
func RecursiveRule(db *Database, baseRel *Relation, predicateID string, recursiveCase func(Term, Term) Goal) func(Term, Term) Goal
func QueryEvaluator(query Goal, varIDs ...int64) GoalEvaluator

// Database wrapper with automatic tabling
func WithTabledDatabase(db *Database) *TabledDatabase
func (tdb *TabledDatabase) Query(rel *Relation, pattern ...Term) Goal

// Cache invalidation
func InvalidateAll()
func InvalidateRelation(predicateID string)
```

## Examples

See package examples:
- `ExampleDatabase_Query_simple` - Basic queries
- `ExampleDatabase_Query_join` - Multi-relation joins
- `ExampleTabledQuery` - Tabling integration
- `ExampleRecursiveRule` - Transitive closure
- `ExampleDatabase_Query_datalog` - Datalog-style rules

## Further Reading

- **SLG Tabling**: See `docs/minikanren/tabling.md` for details on answer subsumption, WFS, and cache invalidation
- **API Reference**: See `docs/api-reference/minikanren.md` for complete API documentation
- **Source Code**: 
  - `pkg/minikanren/pldb.go` - Core database implementation
  - `pkg/minikanren/pldb_slg.go` - Tabling integration
  - `pkg/minikanren/pldb_test.go` - Comprehensive tests
