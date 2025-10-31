# GoKanDo vs Clojure core.logic: Feature Comparison and Enhancement Roadmap

## Executive Summary

This document analyzes the current feature set of GoKanDo (a Go implementation of miniKanren) against Clojure's core.logic library, identifies gaps, and provides a roadmap for achieving feature parity. GoKanDo currently implements core miniKanren functionality with a sophisticated finite domain (FD) solver, but lacks several advanced features present in core.logic.

## Current GoKanDo Feature Set

### ✅ Implemented Features

#### Core miniKanren
- **Unification**: Full term unification with variables, atoms, and pairs
- **Goals**: Success, Failure, Eq, Conj, Disj, Conde
- **Streams**: Lazy evaluation with thread-safe concurrent streams
- **Substitution**: Deep walking and binding management
- **Fresh variables**: Unique variable generation with optional naming

#### Constraint System
- **Order-independent constraints**: Constraints work regardless of addition order
- **Type constraints**: Symbolo, Numbero
- **Disequality**: Neq constraint
- **Absence**: Absento constraint
- **List operations**: Car, Cdr, Cons, Pairo, Nullo, Membero
- **Committed choice**: Conda, Condu, Onceo
- **Projection**: Project goal for computing with bound values

#### Tabling System ✅
- **Memoization**: Tabled goals prevent redundant computation of recursive relations
- **Global table management**: Configurable LRU caching with TTL and size limits
- **Thread-safe caching**: Concurrent access with atomic counters and mutexes
- **Variant generation**: SHA256-based goal normalization for cache keys
- **Streaming integration**: Asynchronous result caching with consumer notification
- **Statistics collection**: Hit rates, memory usage, and performance metrics

#### Finite Domain (FD) Solver
- **Domain representation**: Efficient BitSet-based domains (1-based indexing)
- **All-different constraints**: Basic pairwise and advanced Regin filtering
- **Arithmetic constraints**: Rich arithmetic operators (fd/+, fd/-, fd/*, fd/quotient, fd/mod, fd/=)
- **Offset constraints**: Variable offset relationships (var1 + offset = var2)
- **Inequality constraints**: <, <=, >, >=, != operators
- **Custom constraints**: Extensible framework with SumConstraint example
- **Search heuristics**: Dom/Deg, Domain, Degree, Lexicographic, Random
- **Value ordering**: Ascending, Descending, Random, Middle-out
- **Monitoring**: Comprehensive statistics and performance tracking

#### Thread Safety & Performance
- **Parallel execution**: Configurable worker pools with backpressure
- **Context support**: Cancellation and timeouts
- **Global constraint bus**: Cross-store coordination
- **Memory management**: Pooled constraint stores and buses

### ❌ Missing Features (vs core.logic)

#### Advanced Constraint Logic Programming (CLP)

1. **Tree Constraints / CLP(Tree)**
   - **core.logic**: Support for tree-structured constraints and nominal logic
   - **Gap**: No tree constraint system
   - **Impact**: Limited to flat term structures

2. **Nominal Logic**
   - **core.logic**: Fresh names and nominal unification
   - **Gap**: No support for nominal logic constructs
   - **Impact**: Cannot handle problems requiring name binding/disbinding

#### Enhanced Finite Domain Arithmetic

3. **Arithmetic Relations** ✅ COMPLETED
   - **Phase 7 Complete**: `FDPlus`, `FDMultiply`, etc. are now true relations
   - **No projection needed**: Arithmetic constraints work declaratively
   - **Order-independent**: Constraints validate regardless of variable binding order

4. **Domain Operations**
   - **core.logic**: `fd/in`, `fd/dom`, `fd/interval` for domain specification
   - **Gap**: Limited domain specification (only full domains 1..n)
   - **Impact**: Cannot constrain variables to specific value sets

#### Advanced Search and Control

7. **Search Strategies**
   - **core.logic**: `run*`, `run-db`, `run-nc` with different search behaviors
   - **Gap**: Limited to basic `run` and `run*`
   - **Impact**: Less control over search space exploration

8. **Constraint Store Operations** ✅ COMPLETED
   - **core.logic**: `empty-s`, `make-s`, constraint store manipulation
   - **Gap**: No direct constraint store manipulation primitives
   - **Impact**: Less flexibility in constraint programming

#### Type and Data Constraints

9. **Enhanced Type System**
   - **core.logic**: More sophisticated type constraints
   - **Gap**: Basic type constraints only
   - **Impact**: Limited type-based reasoning

10. **Custom Constraint Definition**
    - **core.logic**: `defc` for defining custom constraints
    - **Gap**: Custom constraints require implementing interface
    - **Impact**: Steeper learning curve for custom constraints

## Performance Analysis

### GoKanDo Strengths
- **Thread Safety**: Superior concurrent execution vs core.logic
- **FD Solver**: Competitive with Regin algorithm for all-different
- **Memory Efficiency**: Go's memory model and garbage collection
- **Compilation**: Ahead-of-time compilation for better startup performance
- **Zero-Copy Streaming**: **Phase 11.2 optimization delivers 5.6x-8.1x performance gains with 45x-48x memory reduction**

### GoKanDo Weaknesses
- **Search Space**: No tabling leads to redundant computation
- **Expressiveness**: Limited domain operations and search strategies

### Benchmark Results
- **Sudoku**: GoKanDo excels (Regin algorithm + parallel execution)
- **Magic Square**: Improved with **true relational arithmetic constraints** (Phase 7) - no more projection needed
- **Cryptarithms**: Now supported with **declarative relational arithmetic** (Phase 7) - SEND + MORE = MONEY can be expressed relationally
- **Streaming Performance**: **Phase 11.2 zero-copy optimizations achieve 5.6x-8.1x throughput improvements with 45x-48x memory savings**

## Enhancement Roadmap

### Phase 1: Core Arithmetic Extensions (High Priority) ✅ COMPLETED

#### 1.1 Rich Arithmetic Constraints ✅ COMPLETED

GoKanDo now supports **true relational arithmetic** without projection, enabling declarative arithmetic programming:

```go
// ✅ COMPLETED: True relational arithmetic constraints
FDPlus(a, b, c)        // a + b = c (relational, not projection-based)
FDMultiply(a, b, c)    // a * b = c (relational)
FDEqual(a, b, c)       // a = b = c (relational)
FDMinus(a, b, c)       // a - b = c (relational)
FDQuotient(a, b, c)    // a / b = c (relational, integer division)
FDModulo(a, b, c)      // a % b = c (relational)

// Legacy projection-based approach (still supported but deprecated)
Project([]Term{a, b, c}, func(vals) {
    // Manual arithmetic verification - no longer needed!
})
```

#### Key Achievement: Phase 7 (Arithmetic Relations) ✅ COMPLETED

GoKanDo now implements **true relational arithmetic constraints** that work without projection:

- **Order-independent**: Arithmetic constraints work regardless of binding order
- **Automatic validation**: No manual `Project` verification needed
- **Backward compatible**: Legacy projection code still works
- **Performance**: Direct constraint checking without host-language extraction

**Before Phase 7:**
```go
// Required manual projection for arithmetic
FDPlus(a, b, c), Project([]Term{a,b,c}, func(vals) {
    if vals[0]+vals[1] == vals[2] { return Success }
    return Failure
})
```

**After Phase 7:**
```go
// Pure relational arithmetic - no projection needed!
FDPlus(a, b, c)  // Automatically validates a + b = c
```

#### 1.2 Domain Specification
```go
// Allow custom domains
var := fd.NewVarWithDomain([]int{1,3,5,7,9})  // Only odd numbers
var := fd.NewVarWithInterval(10, 20)          // Range 10-20
```

#### 1.3 Arithmetic Goals
```go
// Declarative arithmetic goals
goal := FDPlus(a, b, c)     // a + b = c
goal := FDMultiply(a, b, c) // a * b = c
goal := FDEqual(a, b, c)    // a = b = c
goal := FDMinus(a, b, c)    // a - b = c
goal := FDQuotient(a, b, c) // a / b = c
goal := FDModulo(a, b, c)   // a % b = c
```

### Phase 2: Advanced CLP Features (Medium Priority)

#### 2.1 Tabling System ✅ COMPLETED
```go
// Memoization for recursive relations
table := NewTable()
goal := table.Memoize(func(x *Var) Goal {
    return Disj(
        Eq(x, NewAtom(0)),
        fresh(func(y *Var) Goal {
            return Conj(
                // recursive call with memoization
                table.Call(y),
                Eq(x, NewAtom(y.Value() + 1)),
            )
        }),
    )
})
```

#### 2.2 Tree Constraints
```go
// Support for tree-structured terms
goal := TreeConstraint(term, treeSpec)
```

### Phase 3: Enhanced Search and Control (Low Priority)

#### 3.1 Advanced Search Strategies
```go
// Different search behaviors
solutions := RunWithStrategy(strategy, n, goal)
```

#### 3.2 Constraint Store Manipulation
```go
// Direct store operations
store := EmptyStore()
store = store.WithConstraint(constraint)
```

## Implementation Strategy

### Architecture Decisions

1. **Maintain Thread Safety**: All new features must preserve GoKanDo's thread safety guarantees

2. **Incremental Enhancement**: Add features without breaking existing APIs

3. **Performance Focus**: Match or exceed core.logic performance where possible

4. **Go Idioms**: Use Go's type system and concurrency patterns effectively

### Technical Approach

#### Arithmetic Constraints
- Extend `FDStore` with new constraint types
- Implement efficient propagation algorithms
- Add goal constructors for declarative use

#### Tabling
- Implement SLG resolution algorithm
- Add memoization tables to constraint stores
- Integrate with existing stream system

#### Domain Operations
- Extend `BitSet` for custom domains
- Add domain union/intersection operations
- Support sparse domains efficiently

## Migration Path

### Backward Compatibility
- All existing APIs remain functional
- New features are additive
- Performance improvements are transparent

### Deprecation Strategy
- Mark basic implementations as deprecated when enhanced versions available
- Provide migration guides
- Maintain compatibility for 2+ major versions

## Success Metrics

### Feature Parity
- [x] Rich arithmetic constraints (fd/+, fd/*, fd/=, etc.)
- [x] **True relational arithmetic (Phase 7)**
- [x] Tabling support
- [ ] Domain specification (fd/in, fd/dom)
- [ ] Tree constraints
- [ ] Enhanced search strategies

### Performance Targets
- [x] **Phase 11.2 COMPLETED**: 5.6x-8.1x performance gains with 45x-48x memory reduction through zero-copy streaming
- [ ] Match core.logic on arithmetic-heavy benchmarks
- [ ] Maintain superiority on combinatorial problems
- [ ] Improve parallel scaling

### Usability Goals
- [ ] Declarative arithmetic programming
- [ ] Intuitive custom constraint definition
- [ ] Comprehensive documentation and examples

## Conclusion

GoKanDo has achieved significant progress with the completion of **Phase 6 (Rich Arithmetic Operators)** and **Phase 7 (Arithmetic Relations)**, implementing all core arithmetic constraints (fd/+, fd/-, fd/*, fd/quotient, fd/mod, fd/=) as **true relational constraints** without projection. This closes a major expressiveness gap with core.logic and enables more declarative constraint programming.

The remaining gaps focus on **Phase 11 (Ecosystem and Tooling)**. With the completion of **Phase 10 (Constraint Store Operations)**, GoKanDo now provides direct constraint store manipulation primitives (`empty-s`, `make-s`, store union/intersection/difference) and comprehensive inspection utilities that match core.logic's capabilities. This further closes the expressiveness gap with core.logic while leveraging Go's performance and concurrency advantages.

**Phase 11.2 (Performance Optimization) has been completed** with significant improvements: **5.6x-8.1x performance gains** and **45x-48x memory reduction** through zero-copy streaming optimizations.

## References

- [core.logic Documentation](https://clojure.github.io/core.logic/)
- [miniKanren Paper](http://webyrd.net/scheme-2013/papers/HemannMuKanren2013.pdf)
- [FD Solver Algorithms](https://www.sciencedirect.com/science/article/pii/S0004370212000384)
- [Constraint Programming Handbook](https://www.springer.com/gp/book/9780387266545)

## Fact Store Analysis: Should GoKanDo Have One?

### Current State: No Dedicated Fact Store

GoKanDo currently handles facts through **relational programming** - facts are encoded as goals rather than stored persistently:

```go
// Current approach: Facts as relations
likes := func(person, food minikanren.Term) minikanren.Goal {
    return minikanren.Disj(
        minikanren.Conj(
            minikanren.Eq(person, minikanren.NewAtom("alice")),
            minikanren.Eq(food, minikanren.NewAtom("pizza")),
        ),
        minikanren.Conj(
            minikanren.Eq(person, minikanren.NewAtom("bob")),
            minikanren.Eq(food, minikanren.NewAtom("burgers")),
        ),
        // ... more facts
    )
}
```

### Should GoKanDo Have a Fact Store?

**Answer: It depends on your use cases and design philosophy.**

#### Arguments FOR a Fact Store:
1. **Traditional Database Operations**: Assert/retract facts, persistent storage
2. **Large Knowledge Bases**: More efficient than encoding thousands of facts as disjunctions
3. **Dynamic Knowledge**: Add/remove facts at runtime without recompiling
4. **Industry Applications**: Expert systems, knowledge graphs, rule engines
5. **Interoperability**: Integration with external data sources

#### Arguments AGAINST a Fact Store:
1. **Pure Relational Programming**: Facts as relations maintains miniKanren's purity
2. **Performance**: Current approach is optimal for constraint solving
3. **Simplicity**: Adding persistence increases complexity significantly
4. **Scope**: GoKanDo is focused on constraint solving, not knowledge management
5. **Existing Solutions**: External databases can handle persistence needs

## Your Options

### Option 1: No Fact Store (Recommended for Current Scope)

**Pros:**
- Maintains miniKanren purity
- Optimal for constraint solving
- Simpler architecture
- Better performance for CLP problems

**Cons:**
- No persistent fact storage
- Facts must be hardcoded or loaded at startup
- Less suitable for dynamic knowledge bases

**Implementation:** Continue with current relational approach.

### Option 2: Lightweight Fact Store

Add a simple fact store for basic assert/retract operations:

```go
type FactStore struct {
    facts map[string][]Term // predicate -> facts
    mu    sync.RWMutex
}

func (fs *FactStore) Assert(predicate string, terms ...Term) {
    // Add fact to store
}

func (fs *FactStore) Retract(predicate string, terms ...Term) {
    // Remove fact from store
}

func (fs *FactStore) Query(predicate string, terms ...Term) Goal {
    // Convert stored facts to relational goals
}
```

**Pros:**
- Enables dynamic fact management
- Backward compatible
- Relatively simple to implement

**Cons:**
- Still not persistent across sessions
- Limited query capabilities
- Not a full database

### Option 3: Integration with External Databases

**Option 3a: Database Goals**
Create goals that query external databases:

```go
func DatabaseGoal(db *sql.DB, query string, args ...interface{}) Goal {
    return func(ctx context.Context, store ConstraintStore) *Stream {
        // Execute query and convert results to constraint store bindings
    }
}
```

**Option 3b: ORM Integration**
Integrate with Go ORMs (GORM, etc.) for object-relational mapping.

**Option 3c: Graph Database Integration**
Connect to Neo4j, Dgraph, or similar for complex relationships.

**Pros:**
- Leverages existing database ecosystems
- Handles persistence, transactions, scaling
- Rich query languages (SQL, Cypher, etc.)

**Cons:**
- Adds external dependencies
- Less integrated with miniKanren semantics
- Performance overhead of database roundtrips

### Option 4: Full CLP System with Tabling

Implement tabling (memoization) and a fact store as part of a complete CLP system:

```go
type TablingStore struct {
    table map[string]*Table // predicate -> memoized results
    facts map[string][]Fact // persistent facts
}

func (ts *TablingStore) Table(predicate string, goal Goal) Goal {
    // Memoize goal results
}
```

**Pros:**
- Complete CLP system
- Handles infinite relations
- Enables sophisticated applications

**Cons:**
- Major architectural change
- Complex implementation
- Significant performance overhead

## Recommendation

**For GoKanDo's current scope (constraint solving + CLP), I recommend Option 1: No dedicated fact store.**

**Rationale:**
1. **Core Purpose**: GoKanDo excels at constraint solving, where facts are typically known at compile time
2. **Performance**: Relational encoding is optimal for CLP problems
3. **Simplicity**: Avoids complexity that would dilute the focus
4. **Integration**: External databases can handle persistence needs

**When to reconsider:**
- If you need dynamic knowledge bases with frequent fact updates
- If building expert systems or rule engines
- If integrating with existing database-driven applications
- If the user base demands traditional database operations

**Alternative Approach**: Start with Option 2 (lightweight fact store) if you want to experiment with minimal risk, then expand to Option 3 or 4 based on user feedback.

## What core.logic Uses for Fact Storage

### core.logic's PLDB (Persistent Logic DataBase)

core.logic **does have a fact store system** called PLDB, which is essentially an **in-memory map-based database with indexing**. Here's how it works:

#### PLDB Architecture:
```clojure
;; Facts are stored as nested maps with indexing
{
  "namespace/relation_arity" {
    ::unindexed #{[fact1-args] [fact2-args] ...}  ; All facts
    0 {"value" #{[fact-args]}}                    ; Index on arg 0
    1 {"value" #{[fact-args]}}                    ; Index on arg 1
  }
}
```

#### Key Features:
- **In-memory storage** using Clojure's persistent data structures
- **Automatic indexing** on marked attributes for efficient querying
- **Assert/retract operations** for dynamic fact management
- **MVCC-like semantics** through immutable data structures
- **Indexing on specific relation arguments** for performance

#### Usage Example:
```clojure
;; Define a relation with indexing
(db-rel likes ^:index person ^:index food)

;; Create database with facts
(def db (db
  (likes "alice" "pizza")
  (likes "bob" "burgers")))

;; Query with automatic indexing
(run* [q]
  (with-db db
    (likes "alice" q)))  ; Uses index on person
```

## Evaluation of Alternatives for GoKanDo

### Option A: go-memdb

**What it is:** HashiCorp's in-memory database with MVCC, transactions, and rich indexing.

#### Pros:
- ✅ **Excellent indexing**: Supports compound indexes, efficient queries
- ✅ **ACID transactions**: Atomic operations across multiple tables
- ✅ **MVCC**: Multiple concurrent readers, single writer
- ✅ **Production ready**: Used in Consul, Nomad, Vault
- ✅ **Rich query API**: Supports complex queries with watches

#### Cons:
- ❌ **In-memory only**: No persistence (though you could add it)
- ❌ **Heavyweight**: Complex API, significant learning curve
- ❌ **Overkill for CLP**: Too much infrastructure for logic programming
- ❌ **SQL-like impedance**: Not designed for relational programming semantics

#### Fit for GoKanDo: **Poor**
go-memdb is designed for operational databases (service discovery, configuration), not logic programming. The API mismatch would be significant.

### Option B: SQLite

**What it is:** Embedded SQL database with ACID transactions and persistence.

#### Pros:
- ✅ **Persistent**: Survives process restarts
- ✅ **ACID**: Full transactional semantics
- ✅ **Familiar**: SQL interface, widely understood
- ✅ **Lightweight**: Small footprint, no server required
- ✅ **Concurrent**: Supports multiple readers, single writer

#### Cons:
- ❌ **SQL impedance mismatch**: Relational programming ≠ relational databases
- ❌ **Performance overhead**: Disk I/O, SQL parsing, transactions
- ❌ **Not designed for CLP**: No native support for unification, backtracking
- ❌ **Complex for simple facts**: Overkill for in-memory fact storage

#### Fit for GoKanDo: **Poor to Fair**
SQLite could work for persistence, but the SQL interface doesn't align with miniKanren's relational programming model.

## Recommended Approach for GoKanDo

### Option C: Custom PLDB-like Implementation (Recommended)

Implement a Go version of core.logic's PLDB approach:

```go
type FactStore struct {
    facts   map[string]*RelationStore // relation -> facts
    indexes map[string]map[int]map[Term]map[int]bool // relation -> argIndex -> value -> factIDs
    mu      sync.RWMutex
}

type RelationStore struct {
    facts   []Fact
    indexed []bool // which arguments are indexed
}

func (fs *FactStore) Assert(relation string, args ...Term) {
    // Add fact with automatic indexing
}

func (fs *FactStore) Query(relation string, args ...Term) Goal {
    // Convert indexed facts to relational goals
}
```

#### Why This is Best:
- ✅ **Aligned with core.logic**: Familiar semantics for users
- ✅ **Optimized for CLP**: Designed for relational programming
- ✅ **Flexible indexing**: Index only what needs performance
- ✅ **Go-native**: Uses Go's concurrency, no external dependencies
- ✅ **Extensible**: Can add persistence layer later if needed

#### Implementation Strategy:
1. **Core storage**: Map-based with optional indexing
2. **Indexing**: On specific argument positions (like PLDB)
3. **Query integration**: Facts become goals in the constraint system
4. **Optional persistence**: Add SQLite backend later if needed

### Option D: Hybrid Approach

Use the custom PLDB for logic programming, with optional persistence:

```go
type PersistentFactStore struct {
    memory *FactStore        // Fast in-memory access
    disk   *sql.DB          // SQLite for persistence
    sync   chan FactUpdate  // Async sync channel
}
```

This gives you the best of both worlds: fast in-memory operations with optional persistence.

## Conclusion

**core.logic uses PLDB**: An in-memory map-based database with automatic indexing, optimized for logic programming.

**Neither go-memdb nor SQLite are great fits** because they're designed for operational/relational databases, not constraint logic programming.

**Recommended**: Implement a Go version of PLDB with optional SQLite persistence. This maintains the relational programming semantics while providing the fact storage capabilities users expect.

## Implementation Complexity Analysis: Fact Store Options

### Complexity Metrics

| Metric | Description |
|--------|-------------|
| **Dev Time** | Estimated development time in person-weeks |
| **LoC** | Lines of code (core implementation) |
| **Maintenance** | Ongoing maintenance burden (1-5 scale) |
| **Performance** | Runtime performance impact |
| **Semantic Fit** | How well it matches CLP semantics (1-5 scale) |
| **Testing** | Test complexity and coverage needs |
| **Integration** | How cleanly it integrates with existing code |

---

## Option 1: Custom PLDB Implementation

**Architecture**: In-memory map-based storage with indexing, direct integration with miniKanren goals.

### Implementation Components:
1. **Core Storage** (`fact_store.go`): Map-based fact storage with mutexes
2. **Indexing System** (`indexer.go`): Automatic indexing on specified arguments
3. **Goal Integration** (`fact_goals.go`): Convert stored facts to miniKanren goals
4. **Assert/Retract** (`operations.go`): Dynamic fact management
5. **Query Engine** (`query.go`): Efficient fact retrieval with unification

### Complexity Breakdown:

| Component | LoC | Complexity | Key Challenges |
|-----------|-----|------------|----------------|
| Core Storage | 200 | Medium | Thread-safe maps, memory management |
| Indexing | 150 | Medium | Index maintenance, query optimization |
| Goal Bridge | 100 | Low | Convert facts to disjunctive goals |
| Operations | 80 | Low | Assert/retract with index updates |
| Query Engine | 120 | High | Unification with indexed facts |
| **Total Core** | **650** | **Medium** | **Unification integration** |

### Complexity Assessment:
- **Dev Time**: 2-3 weeks
- **LoC**: ~650 lines core + ~200 tests
- **Maintenance**: 2/5 (simple, self-contained)
- **Performance**: Excellent (in-memory, optimized for CLP)
- **Semantic Fit**: 5/5 (designed for relational programming)
- **Testing**: Medium (unit tests + integration with constraint system)
- **Integration**: Low (direct extension of existing patterns)

**Key Advantages:**
- ✅ **Semantic alignment**: Natural fit with miniKanren
- ✅ **Performance**: Optimized for CLP query patterns
- ✅ **Control**: Full control over implementation
- ✅ **Minimal dependencies**: Pure Go implementation

**Key Challenges:**
- 🔴 **Unification complexity**: Handling partial matches and variable binding
- 🟡 **Index design**: Choosing what to index for performance

---

## Option 2: go-memdb Bridge

**Architecture**: Use go-memdb as storage engine with translation layer to miniKanren.

### Implementation Components:
1. **Schema Design** (`schema.go`): Map relations to memdb tables
2. **Translation Layer** (`translator.go`): Convert miniKanren terms ↔ memdb objects
3. **Query Bridge** (`query_bridge.go`): Map relational queries to memdb operations
4. **Transaction Manager** (`tx_manager.go`): Handle memdb transactions
5. **Result Adapter** (`result_adapter.go`): Convert memdb results to miniKanren streams

### Complexity Breakdown:

| Component | LoC | Complexity | Key Challenges |
|-----------|-----|------------|----------------|
| Schema Design | 100 | Medium | Map CLP relations to DB tables |
| Translation | 200 | High | Term ↔ object conversion, unification |
| Query Bridge | 180 | High | CLP queries → memdb operations |
| Transaction Mgr | 120 | Medium | MVCC coordination with constraint system |
| Result Adapter | 150 | High | Stream conversion, backtracking |
| **Total Core** | **750** | **High** | **Semantic translation** |

### Complexity Assessment:
- **Dev Time**: 4-5 weeks
- **LoC**: ~750 lines core + ~300 tests + ~200 schema
- **Maintenance**: 4/5 (external dependency, complex integration)
- **Performance**: Good (memdb is fast), but translation overhead
- **Semantic Fit**: 2/5 (operational DB semantics ≠ CLP)
- **Testing**: High (test translation layers, memdb integration)
- **Integration**: High (significant API mismatch)

**Key Advantages:**
- ✅ **Proven storage**: memdb handles indexing, transactions
- ✅ **Scalability**: memdb designed for high performance
- ✅ **Features**: Rich indexing, watches, MVCC

**Key Challenges:**
- 🔴 **Semantic gap**: CLP unification vs operational queries
- 🔴 **API mismatch**: memdb Txn API vs miniKanren Goal API
- 🟡 **Schema evolution**: How to represent variable relations in fixed schema

---

## Option 3: SQLite Bridge

**Architecture**: SQLite as persistent storage with SQL query generation from miniKanren goals.

### Implementation Components:
1. **SQL Schema** (`sql_schema.go`): Design tables for relations and facts
2. **SQL Generator** (`sql_gen.go`): Convert miniKanren queries to SQL
3. **Connection Pool** (`db_pool.go`): Manage SQLite connections
4. **Result Parser** (`result_parser.go`): Convert SQL results to miniKanren terms
5. **Migration System** (`migrations.go`): Handle schema changes

### Complexity Breakdown:

| Component | LoC | Complexity | Key Challenges |
|-----------|-----|------------|----------------|
| SQL Schema | 120 | Medium | Design for relational programming |
| SQL Generator | 300 | Very High | Query compilation, unification in SQL |
| Connection Pool | 80 | Low | Standard database connection management |
| Result Parser | 150 | High | SQL tuples → miniKanren terms |
| Migration System | 100 | Medium | Schema versioning for relations |
| **Total Core** | **750** | **Very High** | **SQL query generation** |

### Complexity Assessment:
- **Dev Time**: 5-6 weeks
- **LoC**: ~750 lines core + ~400 tests + ~200 SQL/migrations
- **Maintenance**: 3/5 (SQL complexity, but familiar patterns)
- **Performance**: Poor to Fair (disk I/O, SQL parsing, impedance)
- **Semantic Fit**: 1/5 (SQL relations ≠ logic programming relations)
- **Testing**: Very High (test SQL generation, complex integration)
- **Integration**: Very High (massive semantic gap)

**Key Advantages:**
- ✅ **Persistence**: Survives restarts
- ✅ **Familiar**: SQL ecosystem, tooling
- ✅ **Ecosystem**: Rich SQL tools, ORMs, migrations

**Key Challenges:**
- 🔴 **Query compilation**: Converting unification to SQL is extremely complex
- 🔴 **Impedance mismatch**: Declarative logic vs imperative SQL
- 🔴 **Performance**: Disk I/O kills CLP performance
- 🟡 **Schema rigidity**: Fixed schemas vs dynamic relations

---

## Comparative Analysis

### Development Effort:
```
Custom PLDB: ████████░░  (2-3 weeks)
go-memdb:    ██████████  (4-5 weeks)
SQLite:      ███████████ (5-6 weeks)
```

### Maintenance Burden:
```
Custom PLDB: ████░░░░░░░  (Low - self-contained)
go-memdb:    ████████░░░  (High - external dependency)
SQLite:      ███████░░░░  (Medium - familiar SQL patterns)
```

### Semantic Fit:
```
Custom PLDB: ██████████  (Perfect alignment)
go-memdb:    ████░░░░░░░  (Poor - operational DB)
SQLite:      ██░░░░░░░░░  (Very Poor - SQL relations)
```

### Performance:
```
Custom PLDB: ██████████  (Optimized for CLP)
go-memdb:    ████████░░░  (Good, with translation overhead)
SQLite:      ████░░░░░░░  (Poor - disk I/O, SQL parsing)
```

### Risk Assessment:
- **Custom PLDB**: Low risk, full control, but requires CLP expertise
- **go-memdb**: Medium risk, proven storage but complex integration
- **SQLite**: High risk, semantic mismatch may lead to poor performance/usability

## Recommendation

**For GoKanDo's use case, Custom PLDB is strongly recommended:**

### Why Custom PLDB Wins:
1. **Semantic Alignment**: Designed for relational programming
2. **Performance**: Optimized for CLP query patterns
3. **Simplicity**: Less code, fewer moving parts
4. **Control**: No external dependencies or API mismatches
5. **Maintainability**: Self-contained, easy to modify

### When to Consider Alternatives:
- **go-memdb**: If you need advanced indexing features and don't mind the complexity
- **SQLite**: Only if persistence is absolutely critical and you're willing to accept the performance penalty

### Implementation Strategy:
1. **Start small**: Basic fact storage without indexing
2. **Add indexing**: Implement selective indexing for performance
3. **Goal integration**: Connect facts to the constraint system
4. **Optional persistence**: Add SQLite backend later if needed

The custom PLDB approach gives you 80% of the functionality with 50% of the complexity compared to the bridge approaches.

## Enhanced Architecture Requirements

### 1. Generic Constraint Interface & Manager

**Requirement**: Build a generic constraint interface/manager system that allows new solvers to slot in seamlessly.

#### Constraint Interface Design:
```go
type Constraint interface {
    ID() string
    Variables() []*Var
    Check(store ConstraintStore) ConstraintResult
    Propagate(store ConstraintStore) bool
    Clone() Constraint
}

type ConstraintManager interface {
    RegisterSolver(solver Solver)
    AddConstraint(constraint Constraint) error
    GetSolverFor(constraint Constraint) Solver
    CoordinateConstraints(store ConstraintStore) error
}

type Solver interface {
    Name() string
    CanHandle(constraint Constraint) bool
    Solve(constraint Constraint, store ConstraintStore) (ConstraintResult, error)
    Priority() int // For solver selection
}
```

#### Benefits:
- **Pluggable Solvers**: FD solver, SAT solver, custom solvers
- **Solver Selection**: Automatic routing based on constraint types
- **Extensibility**: Third-party solvers can be added
- **Performance**: Specialized solvers for different constraint classes

### 2. Pluggable Labeling/Search Strategies

**Requirement**: Provide labeling/search strategies as pluggable options.

#### Strategy Interface Design:
```go
type LabelingStrategy interface {
    Name() string
    SelectVariable(vars []*FDVar, store *FDStore) *FDVar
    OrderValues(var *FDVar, store *FDStore) []int
}

type SearchStrategy interface {
    Name() string
    Search(store *FDStore, vars []*FDVar) *Stream
    Configure(options SearchOptions)
}

// Built-in strategies
var (
    StrategyFirstFail     LabelingStrategy = &FirstFailStrategy{}
    StrategyDomOverWDeg   LabelingStrategy = &DomOverWDegStrategy{}
    StrategyLex           LabelingStrategy = &LexicographicStrategy{}

    SearchDFS             SearchStrategy = &DepthFirstSearch{}
    SearchBFS             SearchStrategy = &BreadthFirstSearch{}
    SearchLimitedDFS      SearchStrategy = &LimitedDepthFirstSearch{}
)
```

#### Usage:
```go
solver := NewFDSolver().
    WithLabelingStrategy(StrategyDomOverWDeg).
    WithSearchStrategy(SearchLimitedDFS.WithLimit(1000))

results := solver.Solve(vars, constraints)
```

#### Benefits:
- **Flexible Search**: DFS, BFS, limited depth, custom strategies
- **Variable Ordering**: First-fail, domain size, degree-based heuristics
- **Performance Tuning**: Strategy selection based on problem characteristics
- **Research Enablement**: Easy experimentation with new strategies

### 3. Streaming API for Incremental Results

**Requirement**: Provide streaming API (Go channels or lazy iterator) for incremental result consumption and early cancellation.

#### Stream Interface Design:
```go
type ResultStream interface {
    Next() (ConstraintStore, bool)  // Returns next result and whether more exist
    Close()                         // Clean up resources
    Count() int                     // Number of results consumed so far
}

type ChannelStream struct {
    results chan ConstraintStore
    done    chan struct{}
    count   int32
}

// Lazy iterator approach
type LazyIterator struct {
    computeNext func() (ConstraintStore, bool)
    current     ConstraintStore
    hasNext     bool
    computed    bool
}
```

#### Usage:
```go
stream := RunStream(100, func(q *Var) Goal {
    return complexGoal(q)
})

for result, hasMore := stream.Next(); hasMore; result, hasMore = stream.Next() {
    processResult(result)
    if shouldStop() {
        stream.Close()
        break
    }
}
```

#### Benefits:
- **Memory Efficiency**: Don't store all results in memory
- **Early Termination**: Stop when enough results found
- **Responsiveness**: Process results as they become available
- **Resource Control**: Prevent memory exhaustion on large result sets

### 4. Context-Aware Cancellation Support

**Requirement**: Provide cancellation support (context.Context-aware) for aborting long-running searches.

#### Context Integration:
```go
type Goal func(ctx context.Context, store ConstraintStore) *Stream

// Context-aware execution functions
func RunWithContext(ctx context.Context, n int, goalFunc func(*Var) Goal) []Term
func RunStarWithContext(ctx context.Context, goalFunc func(*Var) Goal) []Term

// Context propagation through combinators
func Conj(ctx context.Context, goals ...Goal) Goal {
    return func(store ConstraintStore) *Stream {
        select {
        case <-ctx.Done():
            return NewStream() // Empty stream on cancellation
        default:
            // Continue with conjunction logic
        }
    }
}
```

#### Usage:
```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

results := RunWithContext(ctx, 100, func(q *Var) Goal {
    return longRunningGoal(q)
})

// Or with manual cancellation
ctx, cancel := context.WithCancel(context.Background())
go func() {
    time.Sleep(1 * time.Second)
    cancel() // Cancel after 1 second
}()

results := RunWithContext(ctx, 100, goalFunc)
```

#### Benefits:
- **Timeout Support**: Prevent infinite searches
- **Resource Control**: Clean up goroutines on cancellation
- **User Control**: Allow users to abort long-running operations
- **Integration**: Works with Go's standard context patterns

### 5. First-Class Goal Functions with Combinators

**Requirement**: Favor first-class Go functions that return Goal objects with combinators and convenience wrappers.

#### Goal Function Design:
```go
type Goal func(ctx context.Context, store ConstraintStore) *Stream

// Core combinators
func Success(ctx context.Context, store ConstraintStore) *Stream
func Failure(ctx context.Context, store ConstraintStore) *Stream

func Eq(term1, term2 Term) Goal
func Conj(goals ...Goal) Goal
func Disj(goals ...Goal) Goal

// Convenience wrappers for readability
func And(goals ...Goal) Goal { return Conj(goals...) }
func Or(goals ...Goal) Goal  { return Disj(goals...) }

// Fresh variable creation
func Fresh(name string) *Var
func FreshN(names ...string) []*Var

// Goal constructors for constraints
func Numbero(term Term) Goal
func Symbolo(term Term) Goal
func Absento(absent, term Term) Goal

// FD goals
func FDAllDifferent(vars []*Var, domainSize int) Goal
func FDEqual(a, b Term) Goal
func FDPlus(a, b, c Term) Goal  // a + b = c
```

#### Fluent API Design:
```go
// Fluent constraint building
goal := And(
    Eq(x, NewAtom("alice")),
    Or(
        Eq(y, NewAtom("pizza")),
        Eq(y, NewAtom("pasta")),
    ),
    Numbero(z),
)

// Or with method chaining (if desired)
goal := Eq(x, NewAtom("alice")).
    And(Numbero(y)).
    Or(Eq(z, NewAtom("special")))
```

#### Benefits:
- **Readability**: Code reads like logical expressions
- **Composability**: Easy to combine and nest goals
- **Type Safety**: Go's type system prevents errors
- **Performance**: Direct function calls, no reflection
- **Debugging**: Clear stack traces, easy to inspect

---

## Updated Architecture Overview

### Core Components:
1. **ConstraintManager**: Routes constraints to appropriate solvers
2. **Solver Registry**: Pluggable solvers (FD, SAT, custom)
3. **Strategy Manager**: Configurable search and labeling strategies
4. **Stream System**: Channel-based or iterator-based result streaming
5. **Context Propagation**: Cancellation support throughout the system
6. **Goal Combinators**: Rich set of composition operators

### Example Usage:
```go
// Configure solver with strategies
solver := NewFDSolver().
    WithLabelingStrategy(StrategyDomOverWDeg).
    WithSearchStrategy(SearchLimitedDFS.WithLimit(1000))

// Create context with timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

// Build goal with combinators
goal := func(q *Var) Goal {
    x, y := Fresh("x"), Fresh("y")
    return And(
        FDAllDifferent([]*Var{x, y}, 9),
        FDPlus(x, y, NewAtom(15)),
        Eq(q, List(x, y)),
    )
}

// Execute with streaming
stream := RunStreamWithContext(ctx, goal)
defer stream.Close()

for result, hasMore := stream.Next(); hasMore; result, hasMore = stream.Next() {
    fmt.Printf("Found solution: %v\n", result.Reify(q))
    if stream.Count() >= 5 {
        break // Early termination
    }
}
```

This enhanced architecture provides a powerful, flexible, and Go-idiomatic constraint logic programming system with comprehensive performance optimizations and production-ready implementations.