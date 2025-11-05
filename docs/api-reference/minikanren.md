{% raw %}
# minikanren API

Complete API documentation for the minikanren package.

**Import Path:** `github.com/gitrdm/gokanlogic/pkg/minikanren`

## Package Documentation

Package minikanren adds an Among global constraint.

Among(vars, S, K) counts how many variables in vars take a value from the set S
and constrains that count to equal K (with the solver's positive-domain encoding).

Contract and encoding:
  - vars: non-empty slice of FD variables with positive integer domains (1..MaxValue)
  - S: finite set of allowed values (subset of 1..MaxValue); represented internally as a BitSetDomain
  - K: FD variable encoding the count using the solver's convention from Count:
    K ∈ [1 .. n+1] encodes actual count = K-1, where n = len(vars)

Propagation (bounds-consistent, O(n·d)):
  - Classify each variable Xi relative to S:
  - mandatory: domain(Xi) ⊆ S  (Xi must count toward K)
  - possible:  domain(Xi) ∩ S ≠ ∅ (Xi could count toward K)
  - disjoint:  domain(Xi) ∩ S = ∅ (Xi cannot count toward K)
  - Let m = |mandatory| and p = |possible|.
    This implies the count must satisfy m ≤ count ≤ p.
    Using the K-encoding, we prune K to [m+1 .. p+1].
  - Tight bounds enable useful domain pruning on Xi:
  - If m == maxCount (i.e., K.max-1), then all other variables that could be in S must be forced OUT of S: prune Xi := Xi \ S.
  - If p == minCount (i.e., K.min-1), then all variables that could be in S must be forced INTO S: prune Xi := Xi ∩ S.

This filtering is sound and efficient; it mirrors classical Among propagation used in CP.
Stronger propagation (e.g., generalized arc consistency using flows) is possible but beyond scope;
this implementation integrates cleanly with the solver's fixed-point loop and avoids technical debt.

Package minikanren provides a composition-based BinPacking global constraint.

BinPacking assigns each item i to one of m bins with capacity cap[k], and
enforces that, for every bin k, the total size of items assigned to k does
not exceed cap[k]. Items are represented by FD variables x[i] with domains
in {1..m} (bin indices). Sizes are positive integers.

Implementation uses reified assignment booleans and a weighted sum:
  - For each bin k, create booleans b[i,k] ↔ (x[i] == k)
  - For each bin k, compute: load_k = Σ size[i] * (b[i,k] - 1)
    Note: booleans are {1=false, 2=true}. (b-1) turns them into {0,1}
  - We implement Σ size[i]*b[i,k] as a LinearSum to a total variable sum_k,
    then tie sum_k and the encoded load LkPlus1 via Arithmetic:
    sum_k = LkPlus1 + (base_k - 1), where base_k = Σ size[i]
    and domain(LkPlus1) ⊆ [1..cap[k]+1]. This guarantees load ≤ cap[k].

This construction achieves safe bounds-consistent propagation using existing
primitives. Stronger propagation (e.g., subset sum reasoning) can be layered
later without changing the API.

Package minikanren provides global constraints - Circuit (single Hamiltonian cycle)

Circuit models a permutation of successors that forms a single cycle
visiting all nodes exactly once (a Hamiltonian circuit). It is a classic
global constraint used in routing and sequencing problems.

Interface and semantics
- Inputs: succ[1..n], where succ[i] is the successor node index of node i
- Domains: succ[i] ⊆ {1..n} for all i
- startIndex: distinguished start node in [1..n]

Enforced relations
 1. Exactly-one successor per node i (already implicit in a single-valued succ[i],
    but we encode with reified booleans for strong propagation):
    For each i, exactly one j has (succ[i] == j)
 2. Exactly-one predecessor per node j:
    For each j, exactly one i has (succ[i] == j)
 3. No self-loops: succ[i] ≠ i
 4. Subtour elimination via order variables u[1..n]:
    - u[start] = 1, and for all k ≠ start: u[k] ∈ [2..n]
    - For every arc (i -> j) with j ≠ start, if succ[i] == j then u[j] = u[i] + 1
    (reified arithmetic). We deliberately do NOT enforce order on arcs leading
    back to the start to avoid a wrap-around equality that would overconstrain
the Hamiltonian cycle.

Construction strategy
- Create boolean matrix b[i][j] reifying (succ[i] == j)
- Post row and column BoolSum constraints enforcing exactly one true in each
- Force b[i][i] = false to forbid self-loops
- Create order variables u with domains {1} for start and {2..n} for others
- For each (i, j ≠ start), post Reified(Arithmetic(u[i] + 1 = u[j]), b[i][j])

Notes
  - This approach uses O(n^2) auxiliary booleans and reified constraints,
    which is standard and provides robust propagation without bespoke graph
    algorithms. It integrates cleanly with the solver's immutable state.

Package minikanren provides constraint system infrastructure for order-independent
constraint logic programming. This file defines the core interfaces and types
for managing constraints in a hybrid local/global architecture.

The constraint system uses a two-tier approach:
  - Local constraints: managed within individual goal contexts for fast checking
  - Global constraints: coordinated across contexts when constraints span multiple stores

This design provides order-independent constraint semantics while maintaining
high performance for the common case of locally-scoped constraints.

Package minikanren provides concrete implementations of constraints
for the hybrid constraint system. These constraints implement the
Constraint interface and provide the core constraint logic for
disequality, absence, type checking, and other relational operations.

Each constraint implementation follows the same pattern:
  - Efficient local checking when all variables are bound
  - Graceful handling of unbound variables (returns ConstraintPending)
  - Thread-safe operations for concurrent constraint checking
  - Proper variable dependency tracking for optimization

The constraint implementations are designed to be:
  - Fast: Optimized for the common case of local constraint checking
  - Safe: Thread-safe and defensive against malformed input
  - Debuggable: Comprehensive error messages and string representations

Package minikanren provides a thread-safe, parallel implementation of miniKanren
in Go. This implementation follows the core principles of relational programming
while leveraging Go's concurrency primitives for parallel execution.

miniKanren is a domain-specific language for constraint logic programming.
It provides a minimal set of operators for building relational programs:
  - Unification (==): Constrains two terms to be equal
  - Fresh variables: Introduces new logic variables
  - Disjunction (conde): Represents choice points
  - Conjunction: Combines goals that must all succeed
  - Run: Executes a goal and returns solutions

This implementation is designed for production use with:
  - Thread-safe operations using sync package primitives
  - Parallel goal evaluation using goroutines and channels
  - Type-safe interfaces leveraging Go's type system
  - Comprehensive error handling and resource management

Package minikanren provides global constraints for constraint programming.

This file implements the Count constraint and related counting functionality
using reification to achieve arc-consistency.

Package minikanren implements global constraints for finite-domain CP.

This file provides a production-quality implementation of the Cumulative
constraint, a classic resource scheduling constraint. Given a set of tasks
with start-time variables, fixed durations and resource demands, and a fixed
resource capacity, Cumulative enforces that at every time unit the sum of
demands of tasks executing at that time does not exceed the capacity.

Contract (discrete time, 1-based domains):
  - For each task i:
    start[i] is an FD variable with integer domain of possible start times
    dur[i]   is a strictly positive integer duration (time units)
    dem[i]   is a non-negative integer resource demand
  - Capacity C is a strictly positive integer
  - A task scheduled at start s occupies the half-open interval [s, s+dur[i])
    which, with discrete 1-based times, we model as the inclusive range
    [s, s+dur[i]-1]. Two tasks overlap at time t if t is contained in both
    of their inclusive ranges.

Propagation strength: time-table filtering with compulsory parts.
  - We compute compulsory parts for each task from the current start windows:
    est = min(start[i]), lst = max(start[i])
    If lst <= est+dur[i]-1, the task must execute over the inclusive range
    [lst, est+dur[i]-1] regardless of the exact start.
  - We build a resource profile by summing demands over the union of all
    compulsory parts. If the profile ever exceeds capacity, we report
    inconsistency immediately.
  - For pruning, we remove any start value s for task i such that placing
    the task at [s, s+dur[i]-1] would push the profile above capacity at any
time t in that range (i.e., profile[t] + dem[i] > capacity).

This achieves sound bounds-consistent pruning commonly known as time-table
propagation. It is not as strong as edge-finding, but is fast, robust, and
catches many practical conflicts. The solver's fixed-point loop composes
this filtering with other constraints.

Package minikanren provides the Diffn (2D non-overlap) global constraint.

Differ from NoOverlap (1D disjunctive), Diffn enforces that axis-aligned
rectangles do not overlap in the plane. For each rectangle i we have two
finite-domain start variables X[i], Y[i] and fixed positive sizes W[i], H[i].
Rectangles are closed-open on both axes: [X[i], X[i]+W[i]) × [Y[i], Y[i]+H[i]).

Implementation strategy (production, composition-based):
  - For each pair (i, j), post a disjunction that at least one of these holds:
    1) X[i] + W[i] ≤ X[j]
    2) X[j] + W[j] ≤ X[i]
    3) Y[i] + H[i] ≤ Y[j]
    4) Y[j] + H[j] ≤ Y[i]
  - We construct each inequality using Arithmetic (offset helper) and
    Inequality, then reify the inequality into a boolean with the generic
    reifier. A BoolSum over the four booleans is constrained to have
domain [5..8] (since booleans are encoded {1=false,2=true}, a sum ≥5
    guarantees at least one true among four).

This decomposition favors correctness and integration with existing, well-
tested primitives. It achieves safe bounds-consistent pruning and is commonly
used as a baseline Diffn encoding. Stronger filtering (e.g., energy-based,
edge-finding) can be layered later without changing the API.

Package minikanren provides constraint programming abstractions.
This file defines the Domain interface for representing finite domains
over discrete values, enabling solver-agnostic constraint propagation.

Package minikanren: global constraint - ElementValues (table element)

ElementValues enforces: result = values[index]
- index: finite-domain variable whose values are 1-based indices into 'values'
- values: fixed slice of positive integers (acts like a constant array)
- result: finite-domain variable that must equal the value referenced by 'index'

Propagation (arc-consistent over the fixed table):
1) Index bounds pruning to valid range [1..len(values)].
2) From index to result: result ∈ { values[i] | i ∈ indexDomain }.
3) From result to index: index ∈ { i | values[i] ∈ resultDomain }.

Notes
- We allow duplicate entries in 'values'. The constraint naturally handles it.
- All domains are immutable; SetDomain returns a new state preserving copy-on-write semantics.
- If any domain becomes empty, propagation signals inconsistency via error.

Package minikanren provides specialized reified constraints.

This file implements EqualityReified, a constraint that links equality
between two variables to a boolean variable with full bidirectional propagation.

Package minikanren implements global constraints for finite-domain CP.

This file provides a production implementation of the Global Cardinality
Constraint (GCC). GCC bounds how many times each value can occur among a
collection of variables. It is commonly used for assignment and scheduling
models where per-value capacities must be respected.

Contract:
  - Variables X[0..n-1] each have a finite domain over positive integers
  - We consider value set V = {1..M}, where M = max domain value across X
  - For each v in V, we provide bounds minCount[v] and maxCount[v] with
    0 <= minCount[v] <= maxCount[v]
  - GCC enforces that, in any solution, the number of variables assigned
to value v lies within [minCount[v], maxCount[v]] for all v in V.

Propagation strength: bounds-consistent checks plus pruning for saturated values.
  - Compute fixedCount[v]: number of variables already fixed to v
  - Compute possibleCount[v]: number of variables whose domain contains v
  - Fail if fixedCount[v] > maxCount[v] or possibleCount[v] < minCount[v]
  - If fixedCount[v] == maxCount[v], remove v from all other variables

While stronger GAC can be achieved with flow-based algorithms, this
implementation is efficient, sound, and integrates cleanly with the solver's
fixed-point loop. It detects overloads early and applies useful pruning when
some values are saturated.

Package minikanren provides hybrid constraint solving by integrating
relational and finite-domain constraint solvers. This file defines the
plugin architecture that allows specialized solvers to cooperate on
problems requiring both types of reasoning.

The hybrid solver uses a plugin pattern where:
  - Each solver (relational, FD, etc.) implements the SolverPlugin interface
  - The HybridSolver dispatches constraints to appropriate plugins
  - The UnifiedStore maintains both relational bindings and FD domains
  - Plugins propagate changes bidirectionally through the store

This design enables:
  - Attributed variables: variables with both relational bindings and finite domains
  - Cross-solver propagation: FD pruning informs relational search and vice versa
  - Modular extension: new solver types can be added without modifying core infrastructure
  - Lock-free parallel search: UnifiedStore uses copy-on-write like SolverState

Package minikanren provides plugin implementations for the hybrid solver.
This file implements the FD (Finite Domain) plugin that wraps the
existing FD constraint propagation infrastructure from Phase 2.

Package minikanren provides hybrid integration between relational and
finite-domain constraint solving.

This file implements HybridRegistry for managing variable mappings between
pldb relational variables and FD constraint variables. The registry maintains
bidirectional mappings and provides automatic binding propagation.

Design Philosophy:
  - Explicit mappings: Users control which variables map to each other
  - Bidirectional: Maps both relational→FD and FD→relational
  - Immutable operations: All mapping operations return new registry instances
  - Type-safe: Validates mappings at registration time

The registry solves the "variable coordination problem" in hybrid systems:
when a database query binds a relational variable, how do we propagate that
binding to the corresponding FD variable for constraint propagation?

Package minikanren provides plugin implementations for the hybrid solver.
This file implements the Relational plugin that wraps the existing
miniKanren constraint system (disequality, type constraints, etc.).

Package minikanren provides reified set-membership for FD variables.

InSetReified links an integer variable v and a boolean b (1=false, 2=true)
such that b = 2 iff v ∈ S, where S is a fixed set of allowed values.

Propagation is bidirectional and safe:
  - If v's domain has no intersection with S → set b = 1
  - If v is singleton in S → set b = 2
  - If b = 2 → prune v to v∈S (intersect)
  - If b = 1 → prune v to v∉S (remove all S values)

This is used by higher-level globals like Sequence to create membership
booleans over a fixed set without resorting to large per-value tables.

Package minikanren adds a lexicographic ordering global constraint.

This file implements LexLess and LexLessEq over two equal-length vectors
of FD variables. These constraints are commonly used for symmetry breaking
and sequencing models.

Contract:
  - X = [x1..xn], Y = [y1..yn], n >= 1
  - Domains are positive integers as usual (1..MaxValue)
  - LexLess(X, Y)  enforces (x1, x2, ..., xn) <  (y1, y2, ..., yn)
  - LexLessEq(X, Y) enforces (x1, x2, ..., xn) <= (y1, y2, ..., yn)

Propagation (bounds-consistent, O(n)):
  - Maintain whether the prefix can still be equal: eqPrefix = true initially.
  - For i = 1..n while eqPrefix holds:
  - Prune xi > max(yi): xi ∈ (-∞ .. maxYi]
  - Prune yi < min(xi): yi ∈ [minXi .. +∞)
  - If max(xi) < min(yi), the constraint is already satisfied at i and
    later positions are unconstrained by Lex; we may stop.
  - Update eqPrefix := eqPrefix && (xi and yi have a non-empty intersection)
  - For strict LexLess, detect the all-equal tuple case early:
  - If for all i, dom(xi) and dom(yi) are singletons with the same value,
    the constraint is inconsistent.

This filtering is sound and inexpensive. Stronger propagation can be achieved
using reified decompositions, but this implementation integrates cleanly with
the solver's fixed-point propagation loop and avoids adding internal goals.

Package minikanren provides the LocalConstraintStore implementation for
managing constraints and variable bindings within individual goal contexts.

The LocalConstraintStore is the core component of the hybrid constraint system,
providing fast local constraint checking while coordinating with the global
constraint bus when necessary for cross-store constraints.

Key design principles:
  - Fast path: Local constraint checking without coordination overhead
  - Slow path: Global coordination only when cross-store constraints are involved
  - Thread-safe: Safe for concurrent access and parallel goal evaluation
  - Efficient cloning: Optimized for parallel execution where stores are frequently copied

Package minikanren provides Min/Max-of-array global constraints.

These constraints link a result variable R to the minimum or maximum
value among a list of FD variables X[1..n]. They implement safe, bounds-
consistent propagation without over-pruning:
  - Min(vars, R):
    R ∈ [min_i Min(Xi) .. min_i Max(Xi)]
    and for all i: Xi ≥ R (i.e., prune Xi below R.min)
  - Max(vars, R):
    R ∈ [max_i Min(Xi) .. max_i Max(Xi)]
    and for all i: Xi ≤ R (i.e., prune Xi above R.max)

This propagation is sound and inexpensive (O(n)) per call. Stronger
propagation (e.g., identifying unique carriers of the current extremum)
could prune more but is intentionally avoided here to keep the behavior
simple, predictable, and integration-friendly with the solver's fixed-point loop.

Package minikanren provides constraint programming infrastructure.
This file defines the Model abstraction for declaratively building
constraint satisfaction problems.

Package minikanren provides global constraints for finite-domain CP.

This file defines a production-quality NoOverlap (a.k.a. Disjunctive)
constraint constructor built on top of the Cumulative global.

NoOverlap models a set of non-preemptive tasks on a single machine (capacity 1):
no two tasks may execute at the same time. Each task i has a start-time
variable start[i] and a fixed positive duration dur[i].

Implementation strategy:
  - NoOverlap(starts, durations) is modeled as Cumulative with capacity=1,
    unit demands for all tasks, and the given durations.
  - Propagation strength is that of the Cumulative implementation: time-table
    filtering with compulsory parts, which is sound and effective for many
    scheduling problems.
    
Package minikanren provides NValue-style global constraints.

DistinctCount (aka NValue) constrains the number of distinct values taken
by a list of variables. This file provides a composition-based, production
implementation using existing, well-tested primitives (reification and
BoolSum) to achieve safe bounds-consistent propagation without bespoke
graph algorithms.

Design overview
----------------
Given variables X[1..n] with discrete domains, let U be the union of values
present in their domains. For each value v in U, we create:
  - Booleans b_iv reifying (X_i == v)
  - A total T_v that counts how many X_i equal v via BoolSum(b_iv, T_v)
    where T_v encodes count+1 in [1..n+1]
  - A boolean used_v that is true iff some variable takes value v.
    We implement used_v ↔ (T_v >= 2), which is equivalent to T_v ≠ 1.
    To avoid introducing a general inequality reifier, we use a small gadget:
  - Reify (T_v == 1) into b_zero_v
  - Enforce XOR(used_v, b_zero_v) via BoolSum([used_v, b_zero_v], total={2})

Finally, the number of distinct values equals the number of used_v that are
true. We connect that with a BoolSum over all used_v to a caller-provided
variable DPlus1 that encodes distinctCount+1.

With this composition, standard propagation flows through the existing
constraints and achieves sound bounds-consistent pruning. For example,
when DPlus1 is fixed to 2 (distinctCount=1) and one X_i becomes bound to
value a, all other values w≠a get used_w=false, which forces all b_jw=false
and removes w from other X_j domains. This matches the typical AtMostNValues=1
behavior without bespoke code paths.

Package minikanren provides pattern matching operators for miniKanren.

Pattern matching is a fundamental operation in logic programming that
allows matching terms against multiple patterns and executing corresponding
goals. This module provides three pattern matching primitives following
core.logic conventions:

  - Matche: Exhaustive pattern matching (tries all matching clauses)
  - Matcha: Committed choice pattern matching (first match wins)
  - Matchu: Unique pattern matching (requires exactly one match)

Package minikanren provides pldb, an in-memory relational database for logic programming.

pldb enables efficient storage and querying of ground facts with indexed access.
Relations are defined with a name, arity, and optional column indexes.
The Database is a persistent data structure using copy-on-write semantics,
enabling cheap snapshots for backtracking search.

Example usage:

	parent := DbRel("parent", 2, 0, 1)  // Index both columns
	db := NewDatabase()
	db = db.AddFact(parent, NewAtom("alice"), NewAtom("bob"))
	db = db.AddFact(parent, NewAtom("bob"), NewAtom("charlie"))

	// Query: who are alice's children?
	goal := db.Query(parent, NewAtom("alice"), Fresh("child"))

Package minikanren provides hybrid integration helpers for combining pldb
relational queries with finite-domain constraint solving.

This file implements convenience functions that reduce boilerplate when
working with both pldb databases and FD constraints. The helpers maintain
the compositional design while making common patterns more ergonomic.

Design Philosophy:
  - Explicit over implicit: Users control when FD filtering happens
  - Compositional: Helpers wrap existing primitives without magic
  - Thread-safe: All operations safe for concurrent use
  - Zero overhead: No performance penalty vs manual implementation

The key insight is that pldb queries and FD constraints are separate
concerns that can be composed at the Goal level. These helpers encapsulate
proven patterns from the test suite into reusable library functions.

The rest of the file remains unchanged (truncated for brevity in this commit).

{% endraw %}