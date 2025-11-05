# pldb + Hybrid Solver Integration

This guide explains how to integrate gokanlogic's persistent logic database (pldb) with the Phase 3/4 hybrid solver, enabling interaction between relational facts and finite-domain (FD) constraints.

## Reality Check First

**What This Guide Delivers**:
- ✅ Working adapter for pldb + UnifiedStore
- ✅ Real bidirectional constraint propagation
- ✅ Production-quality code with comprehensive tests
- ⚠️ **Manual integration required** (not automatic)
- ⚠️ **Pattern boilerplate** (helper functions would reduce this)

**This is compositional, not seamless.** You get clean building blocks that work correctly, but you must explicitly compose them. See `TASK_6.6_REALITY_CHECK.md` for full assessment.

## Overview

The **UnifiedStoreAdapter** bridges pldb's `ConstraintStore` interface with the hybrid solver's `UnifiedStore`, enabling:

- **Relational queries** to retrieve facts from persistent databases
- **FD constraint propagation** on query results
- **Bidirectional integration** where database facts influence FD domains and vice versa
- **Thread-safe parallel search** with proper isolation

## Architecture

```
┌─────────────┐
│   pldb      │ ──→ Expects ConstraintStore interface
│  Database   │
└─────────────┘
       ↓
┌─────────────────────┐
│ UnifiedStoreAdapter │ ──→ Implements ConstraintStore
└─────────────────────┘     Wraps UnifiedStore
       ↓
┌─────────────────┐
│  UnifiedStore   │ ──→ Phase 3 hybrid solver state
└─────────────────┘     Relational bindings + FD domains
       ↓
┌─────────────────┐
│  HybridSolver   │ ──→ Coordinated propagation
└─────────────────┘     FDPlugin + RelationalPlugin
```

### Key Components

**UnifiedStoreAdapter** (`unified_store_adapter.go`):
- Wraps `UnifiedStore` to implement `ConstraintStore` interface
- Provides bidirectional access for hybrid solver integration
- Thread-safe with mutex protection for adapter state
- Copy-on-write semantics via `Clone()` for parallel search

**Integration Pattern**:
1. Create `UnifiedStore` with FD domains initialized
2. Wrap in `UnifiedStoreAdapter`
3. Pass adapter to `db.Query()`
4. Query results contain bindings in wrapped `UnifiedStore`
5. Run hybrid solver propagation on result stores

## Basic Usage

### Simple Query Without Constraints

```go
// Create database
person, _ := DbRel("person", 2, 0)
db := NewDatabase()
db, _ = db.AddFact(person, NewAtom("alice"), NewAtom(30))
db, _ = db.AddFact(person, NewAtom("bob"), NewAtom(25))

// Create adapter
store := NewUnifiedStore()
adapter := NewUnifiedStoreAdapter(store)

// Query
name := Fresh("name")
age := Fresh("age")
goal := db.Query(person, name, age)
stream := goal(context.Background(), adapter)

// Results contain bindings
results, _ := stream.Take(10)
for _, result := range results {
    nameBinding := result.GetBinding(name.ID())
    // ... use bindings
}
```

### Query with FD Constraints (Manual Filtering)

The adapter does **not** automatically filter query results by FD domains. This is intentional - it gives you explicit control over integration. To filter results:

```go
// Create FD domain for age
model := NewModel()
ageVar := model.NewVariableWithName(
    NewBitSetDomainFromValues(100, []int{25, 26, 27, 28, 29, 30}),
    "age",
)

// Initialize store with domain
store := NewUnifiedStore()
store, _ = store.SetDomain(ageVar.ID(), ageVar.Domain())
adapter := NewUnifiedStoreAdapter(store)

// Create hybrid query with manual filtering
hybridQuery := func(ctx context.Context, cstore ConstraintStore) *Stream {
    dbQuery := db.Query(person, name, age)
    dbStream := dbQuery(ctx, cstore)
    
    stream := NewStream()
    go func() {
        defer stream.Close()
        for {
            results, hasMore := dbStream.Take(1)
            if len(results) == 0 {
                if !hasMore {
                    break
                }
                continue
            }
            
            result := results[0]
            ageBinding := result.GetBinding(age.ID())
            
            // Filter by FD domain
            if ageAtom, ok := ageBinding.(*Atom); ok {
                if ageInt, ok := ageAtom.value.(int); ok {
                    if resAdapter, ok := result.(*UnifiedStoreAdapter); ok {
                        domain := resAdapter.GetDomain(ageVar.ID())
                        if domain != nil && domain.Has(ageInt) {
                            stream.Put(result) // Only emit if in domain
                        }
                    }
                }
            }
        }
    }()
    return stream
}

stream := hybridQuery(context.Background(), adapter)
results, _ := stream.Take(10) // Only ages 25-30
```

### Full Hybrid Propagation

This pattern demonstrates bidirectional integration: database facts → FD domains, FD propagation → relational bindings.

```go
// Database + FD model
employee, _ := DbRel("employee", 3, 0)
db := NewDatabase()
db, _ = db.AddFact(employee, NewAtom("alice"), NewAtom(30), NewAtom("engineering"))

model := NewModel()
ageValues := make([]int, 101)
for i := range ageValues {
    ageValues[i] = i
}
ageVar := model.NewVariableWithName(NewBitSetDomainFromValues(101, ageValues), "age")

// Hybrid solver setup
fdPlugin := NewFDPlugin(model)
relPlugin := NewRelationalPlugin()
solver := NewHybridSolver(relPlugin, fdPlugin)

// Initialize store with FD domain
store := NewUnifiedStore()
store, _ = store.SetDomain(ageVar.ID(), ageVar.Domain())
adapter := NewUnifiedStoreAdapter(store)

// Query database
age := Fresh("age")
dept := Fresh("dept")
goal := db.Query(employee, NewAtom("alice"), age, dept)
stream := goal(context.Background(), adapter)

results, _ := stream.Take(1)
if len(results) > 0 {
    resultAdapter := results[0].(*UnifiedStoreAdapter)
    
    // Link logical variable to FD variable
    resultStore := resultAdapter.UnifiedStore()
    resultStore, _ = resultStore.AddBinding(int64(ageVar.ID()), NewAtom(30))
    resultAdapter.SetUnifiedStore(resultStore)
    
    // Run hybrid propagation
    propagated, err := solver.Propagate(resultAdapter.UnifiedStore())
    if err == nil {
        // FD domain pruned to singleton {30}
        ageDomain := propagated.GetDomain(ageVar.ID())
        fmt.Printf("Age domain: %v\n", ageDomain)
    }
}
```

## Design Rationale

### Why No Automatic FD Filtering?

The adapter intentionally does **not** automatically filter query results by FD domains because:

1. **Separation of concerns**: pldb handles relational queries, hybrid solver handles constraint propagation
2. **Performance**: Not all queries need FD integration; automatic checking would add overhead
3. **Flexibility**: Users can choose when/how to apply FD constraints
4. **Clarity**: Explicit integration makes data flow obvious

This follows the Unix philosophy: do one thing well, compose as needed.

### Thread Safety Model

**Adapter State**:
- Mutex protects the internal `UnifiedStore` pointer
- Read operations (GetBinding, etc.) acquire read lock
- Write operations (AddBinding, SetUnifiedStore) acquire write lock

**UnifiedStore Immutability**:
- UnifiedStore methods return new instances (copy-on-write)
- No shared mutable state between stores
- Cloning creates deep copies suitable for parallel search

**Race-Free ID Generation**:
- Adapter IDs use `atomic.AddInt64()` for thread-safe increments
- Each adapter has unique ID for debugging/logging

## Performance Considerations

### Indexed Queries

pldb maintains hash-based indexes on designated columns. The adapter preserves this performance:

```go
person, _ := DbRel("person", 3, 0, 1) // name and age indexed

// O(1) lookup on indexed field
goal := db.Query(person, NewAtom("alice"), age, dept)

// O(n) scan on non-indexed field
goal := db.Query(person, name, age, NewAtom("engineering"))
```

For large databases (1000+ facts), always index frequently-queried fields.

### Propagation Overhead

Hybrid solver propagation is O(variables × constraints). Minimize overhead by:

1. **Lazy propagation**: Only run solver when FD constraints are present
2. **Incremental propagation**: Propagate after each query result vs. batching
3. **Domain pre-pruning**: Initialize FD domains with tight bounds before querying

Example: querying 1000 facts with age FD domain [30,40] completes in <100ms.

## Integration with Tabling (SLG)

The adapter works with tabled queries for recursive datalog programs:

```go
ancestor, _ := DbRel("ancestor", 2)
parent, _ := DbRel("parent", 2)

// Populate database
db, _ = db.AddFact(parent, NewAtom("alice"), NewAtom("bob"))
db, _ = db.AddFact(parent, NewAtom("bob"), NewAtom("carol"))

// Tabled recursive rule
ancestorRule := func(x, y Term) Goal {
    return Disj(
        db.Query(parent, x, y),
        func(ctx context.Context, cstore ConstraintStore) *Stream {
            z := Fresh("z")
            return Conj(
                db.Query(parent, x, z),
                TabledQuery(ancestor, z, y),
            )(ctx, cstore)
        },
    )
}

// Use with adapter
store := NewUnifiedStore()
adapter := NewUnifiedStoreAdapter(store)

x := Fresh("x")
y := Fresh("y")
goal := TabledQuery(ancestor, x, y)
stream := goal(context.Background(), adapter)

results, _ := stream.Take(10)
// Results include transitive closure
```

Tabling + hybrid solver enables:
- **Well-founded semantics (WFS)** for negation
- **FD constraints** on recursive predicates
- **Efficient** termination detection

## Common Patterns

### Age Range Query

```go
// Find employees aged 25-35
ageValues := []int{}
for i := 25; i <= 35; i++ {
    ageValues = append(ageValues, i)
}
ageVar := model.NewVariableWithName(
    NewBitSetDomainFromValues(100, ageValues),
    "age",
)
store, _ = store.SetDomain(ageVar.ID(), ageVar.Domain())

// Query + manual filter (see "Query with FD Constraints" above)
```

### Join with FD Constraints

```go
// Find parent-child pairs where child age > 20
parent, _ := DbRel("parent", 2, 0, 1)
person, _ := DbRel("person", 2, 0)

childAgeVar := model.NewVariableWithName(
    NewBitSetDomainFromValues(100, makeRange(21, 100)),
    "child_age",
)

parentName := Fresh("parent")
childName := Fresh("child")
childAge := Fresh("age")

goal := Conj(
    db.Query(parent, parentName, childName),
    db.Query(person, childName, childAge),
    // Add manual FD filter here
)
```

### Optimization Queries

```go
// Find youngest employee in department
deptEmployees := db.Query(employee, name, age, NewAtom("engineering"))

minAge := 100
var youngestName string

results, _ := deptEmployees(context.Background(), adapter).Take(100)
for _, result := range results {
    ageBinding := result.GetBinding(age.ID())
    if ageAtom, ok := ageBinding.(*Atom); ok {
        if ageInt, ok := ageAtom.value.(int); ok {
            if ageInt < minAge {
                minAge = ageInt
                nameBinding := result.GetBinding(name.ID())
                if nameAtom, ok := nameBinding.(*Atom); ok {
                    youngestName = nameAtom.value.(string)
                }
            }
        }
    }
}
```

## Testing

The implementation includes comprehensive tests:

- **Basic integration**: Adapter works with pldb queries ✓
- **Adapter cloning**: Parallel search independence ✓
- **Unidirectional propagation**: Facts → FD, FD → facts ✓
- **Bidirectional propagation**: Full round-trip ✓
- **Performance**: 1000-fact database queries ✓
- **Edge cases**: Empty domain conflicts ✓
- **Race detection**: All tests pass with `-race` flag ✓

See `pldb_hybrid_test.go` for details.

## Examples

See `pldb_hybrid_example_test.go` for runnable examples:

- `ExampleUnifiedStoreAdapter_basicQuery` - Simple query
- `ExampleUnifiedStoreAdapter_fdConstrainedQuery` - Manual FD filtering
- `ExampleUnifiedStoreAdapter_hybridPropagation` - Full propagation
- `ExampleUnifiedStoreAdapter_parallelSearch` - Concurrent usage
- `ExampleUnifiedStoreAdapter_performance` - Large database handling

## Limitations

### Current Limitations

1. **No automatic FD filtering**: Must manually check domain membership (by design)
2. **No built-in helper functions**: Manual filtering requires boilerplate (future improvement)
3. **No pldb-FD variable mapping**: User must explicitly link logical vars to FD vars
4. **Performance**: Full domain iteration for filtering (could use interval arithmetic)

### Future Enhancements

Potential additions (not in current scope):

- **HybridQuery() helper**: Automatic FD filtering for common patterns
- **Indexed domain checks**: O(log n) membership tests for large domains
- **Lazy propagation triggers**: Only propagate when FD vars are bound
- **Query planning integration**: Use FD bounds for query optimization

## See Also

- **[pldb Guide](../pldb.md)**: Persistent logic database basics
- **[Hybrid Solver Guide](../../minikanren/hybrid_solver.md)**: Phase 3/4 architecture
- **[FD Constraints](../../minikanren/finite_domains.md)**: FD constraint reference
- **[Tabling (SLG)](../../minikanren/tabling.md)**: Recursive query memoization

## References

- Task 6.6 specification in `docs/implementation_roadmap_v3.md`
- `unified_store_adapter.go` - Adapter implementation
- `pldb_hybrid_test.go` - Integration tests
- `pldb_hybrid_example_test.go` - Usage examples
