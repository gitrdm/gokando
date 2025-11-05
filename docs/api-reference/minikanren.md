---
title: minikanren API
render_with_liquid: false
---

# minikanren API

Complete API documentation for the minikanren package.

**Import Path:** `github.com/gitrdm/gokanlogic/pkg/minikanren`

## Package Documentation

Package minikanren provides finite domain constraint programming with MiniKanren-style logical variables.

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

Package minikanren: global constraints - Circuit (single Hamiltonian cycle)

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
Rectangles are closed-open on both axes: [X[i], X[i]+W[i)) × [Y[i], Y[i]+H[i)).

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

Package minikanren provides finite domain constraint programming with MiniKanren-style logical variables.

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

Package minikanren provides constraint propagation for finite-domain variables.

This file implements modulo constraints for integer arithmetic.
Modulo constraints enforce remainder relationships between variables
while maintaining pure integer domains and providing bidirectional propagation.

Design Philosophy:
  - Integer-only: All operations work with positive integer values (≥ 1)
  - Bidirectional: Propagates both forward (x→remainder) and backward (remainder→x)
  - AC-3 compatible: Implements standard arc-consistency propagation
  - Production-ready: Handles edge cases (modulo 1, bounds checking)

Example Use Case:
In scheduling problems where events repeat cyclically:

	day_of_week = day_number % 7
	time_slot = minute_offset % 30

The Modulo constraint maintains: x mod modulus = remainder

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
  - This mirrors a standard CP modeling technique and composes well with other
    constraints (precedences, objective variables, etc.).

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

These operators significantly reduce boilerplate compared to manual
combinations of Conde, Conj, and destructuring with Car/Cdr.

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

Package minikanren provides integration between pldb relational database
and SLG tabling for efficient recursive query evaluation.

# Integration Architecture

pldb queries normally return Goals that can be composed with Conj/Disj.
SLG tabling requires GoalEvaluators that yield answer bindings via channels.
This file bridges the two by providing:

  - TabledQuery: Wraps Database.Query for use with SLG tabling
  - RecursiveRule: Helper for defining recursive rules with pldb base cases
  - QueryEvaluator: Converts pldb queries to GoalEvaluator format

# Usage Pattern

	// Define base facts
	edge := DbRel("edge", 2, 0, 1)
	db := NewDatabase()
	db = db.AddFact(edge, NewAtom("a"), NewAtom("b"))

	// Define recursive rule with tabling
	path := func(x, y Term) Goal {
	    return TabledQuery(db, edge, x, y, "path", func() Goal {
	        z := Fresh("z")
	        return Conj(
	            TabledQuery(db, edge, x, z, "path"),
	            TabledQuery(db, edge, z, y, "path"),
	        )
	    })
	}

This enables terminating recursive queries over pldb relations using SLG's
fixpoint computation.

Package minikanren provides constraint propagation for finite-domain constraint programming.

This file implements concrete constraint types that integrate with the Phase 1
Model/Solver architecture. Constraints perform domain pruning by removing values
that cannot participate in any solution, providing stronger filtering than
simple backtracking search alone.

The propagation system follows these principles:
  - Constraints implement the ModelConstraint interface
  - Propagation is triggered after domain changes during search
  - The Solver runs constraints to a fixed-point (no more changes)
  - All operations maintain copy-on-write semantics for lock-free parallel search

Constraint algorithms:
  - AllDifferent: Regin's AC algorithm using maximum bipartite matching
  - Arithmetic: Bidirectional arc-consistency for X + c = Y
  - Inequality: Bounds propagation for <, ≤, >, ≥, ≠

Package minikanren: global constraint - Regular (DFA constraint)

Regular enforces that a sequence of FD variables (x1, x2, ..., xn)
forms a word accepted by a given deterministic finite automaton (DFA).

Contract (1-based, positive integers):
  - States are numbered 1..numStates. State 0 is reserved for "no transition".
  - Alphabet symbols are positive integers; a value v outside the transition
    table's width is treated as having no transition from any state.
  - delta is a transition table where delta[s][v] = t gives the next state t
    from state s consuming symbol v. A value of 0 denotes the absence of a
    transition.

Propagation (bounds/GAC over the DFA using forward/backward filtering):
 1. Forward pass: compute reachable states Fi after each position i using
    current domains. Early fail if Fi becomes empty.
 2. Backward pass: start from accepting states intersect Fi at i=n, then for
    i=n..1, compute predecessor states Bi-1 and, simultaneously, collect the
    set of supported symbols for xi using only transitions consistent with
    Fi-1 and Bi.
 3. Prune each xi to its supported symbols. If any domain empties, signal
    inconsistency.

This achieves strong pruning typical of the classic Regular constraint
(Pesant 2004) and composes well with other constraints in the solver's
fixed-point loop.

Package minikanren provides reification support for constraint programming.

Reification allows the truth value of a constraint to be reflected as a boolean
variable using 1-indexed domains: {1 = false, 2 = true}. This enables:
  - Conditional constraints: "if X > 5 then Y = 10"
  - Counting: "count how many variables equal a value"
  - Soft constraints: "maximize constraints satisfied"
  - Logical combinations: AND, OR, NOT over constraints

Reification is bidirectional:
  - Constraint → Boolean: When constraint becomes true/false, prune boolean domain
  - Boolean → Constraint: When boolean is bound, enforce or disable constraint

The reification architecture follows these principles:
  - ReifiedConstraint wraps any PropagationConstraint
  - Boolean variable must have domain subset of {1,2} (1=false, 2=true)
  - Maintains copy-on-write semantics for parallel search
  - Integrates seamlessly with existing constraint propagation

Package minikanren provides constraint propagation for finite-domain variables.

This file implements scaling constraints for integer arithmetic.
Scaling constraints enforce multiplicative relationships between variables
while maintaining pure integer domains and providing bidirectional propagation.

Design Philosophy:
  - Integer-only: All operations work with integer values
  - Bidirectional: Propagates both forward (x→result) and backward (result→x)
  - AC-3 compatible: Implements standard arc-consistency propagation
  - Production-ready: Handles edge cases (zero, negative, bounds checking)

Example Use Case:
In resource allocation problems where capacity scales linearly:

	worker_hours = 40
	total_cost = hourly_rate * worker_hours

The Scale constraint maintains: total_cost = hourly_rate * 40

Package minikanren provides constraint propagation for finite-domain variables.

This file implements scaled division constraints for integer arithmetic.
Scaled division allows division-like reasoning while maintaining pure integer
domains, following the PicoLisp pattern of global scale factors.

Design Philosophy:
  - Integer-only: All operations work with scaled integer values
  - Bidirectional: Propagates both forward (dividend→quotient) and backward (quotient→dividend)
  - AC-3 compatible: Implements standard arc-consistency propagation
  - Production-ready: Handles edge cases (zero, negative, bounds checking)

Example Use Case:
If all monetary values are scaled by 100 (cents), then:

	salary_cents = 5000000 (representing $50,000.00)
	bonus_cents = salary_cents / 100 (representing 10% bonus)

The ScaledDivision constraint maintains: bonus * 100 ⊆ [salary, salary+99]

Package minikanren provides the Sequence global constraint.

Sequence(vars, S, k, minCount, maxCount) enforces that in every sliding
window of length k over vars, the number of variables taking a value in S
is between minCount and maxCount (inclusive).

Implementation uses composition over existing primitives:
  - For each Xi, create a boolean bi reifying Xi ∈ S via InSetReified
  - For each window i..i+k-1, post BoolSum(b[i..i+k-1], totalWin)
    with totalWin domain set to [minCount+1 .. maxCount+1]

This achieves safe bounds-consistent propagation. Stronger filters (e.g.,
sequential counters) can be layered later without API changes.

Package minikanren provides SLG (Linear resolution with Selection function for General logic programs)
resolution engine for tabled evaluation of recursive queries.

# SLG Resolution

SLG resolution extends standard SLD resolution (Prolog/miniKanren) with tabling to:
  - Detect and resolve cycles in recursive predicates
  - Compute fixpoints for mutually recursive relations
  - Cache intermediate results for reuse
  - Guarantee termination for a broad class of programs

# Architecture

The SLG engine coordinates:
  - Producer goroutines that evaluate goals and derive new answers
  - Consumer goroutines that read cached answers as they become available
  - Cycle detection using Tarjan's SCC algorithm on the dependency graph
  - Fixpoint computation for strongly connected components

# Thread Safety

The engine is designed for concurrent access:
  - SubgoalTable uses sync.Map for lock-free lookups
  - Answer insertion is synchronized via mutex in AnswerTrie
  - Producer/consumer coordination uses sync.Cond for efficient signaling
  - Context cancellation propagates cleanly to all goroutines

Package minikanren adds stratified negation (WFS) helpers on top of the SLG engine.

This file provides production-quality Well-Founded Semantics (WFS) implementation
for stratified and general negation with conditional answers, delay sets, and
completion. It builds on the existing SLG Evaluate API and dependency tracking.

Synchronization approach (no sleeps/timers):
  - Non-blocking fast path: we first drain innerCh if it's already closed or has a
    buffered answer to catch immediate outcomes with zero wait.
  - Race-free subscription: we use a versioned event sequence (EventSeq/WaitChangeSince)
    to avoid missing just-fired events.
  - Engine handshake: we obtain a pre-start sequence and a Started() signal from the
    engine to deterministically handle the "inner completes immediately with no answers"
    case without any timers. We also prioritize real change events over the started signal.
  - Reverse-dependency propagation ensures conditional answers are simplified or
    retracted as soon as child outcomes are known.

Package minikanren provides constraint solving infrastructure.
This file implements the core solver with efficient copy-on-write state management
for lock-free parallel search.

# Architecture Overview

The solver separates immutable problem definition from mutable solving state:

	Model (immutable during solving):
	  - Variables with initial domains
	  - Constraints that reference variables
	  - Configuration (heuristics, etc.)
	  - Shared by all parallel workers (zero copy cost)

	SolverState (mutable, copy-on-write):
	  - Sparse chain of domain modifications
	  - Each worker maintains its own independent chain
	  - O(1) cost to create new state node
	  - Pooled for zero GC pressure

# How Constraint Propagation Works

Constraints need to communicate domain changes. This happens via the SolverState:

 1. Constraint reads current domains: GetDomain(state, varID)
 2. Constraint computes domain reduction
 3. Constraint creates new state: SetDomain(state, varID, newDomain)
 4. Process repeats until fixed point

Example with AllDifferent(x, y, z):

	Initial:  x={1,2,3}, y={1,2,3}, z={1,2,3}
	Assign:   x := 1  → State1: x={1}
	Propagate: Remove 1 from y → State2: y={2,3} (parent: State1)
	Propagate: Remove 1 from z → State3: z={2,3} (parent: State2)
	Fixed point reached

Each state node is tiny (40 bytes) and creation is O(1). Backtracking just
discards state nodes. Parallel workers share the Model but have independent
state chains, enabling lock-free search.

Package minikanren provides constraint solving infrastructure.
This file defines additional Solver API methods.

Package minikanren provides the Stretch global constraint.

Stretch(vars, values, minLen, maxLen) constrains run lengths of values along
a sequence of FD variables. For each value v in values, every maximal run of
consecutive occurrences of v must have a length between minLen[v] and
maxLen[v] (inclusive). Values not listed in 'values' are unconstrained by
default (equivalent to minLen=1, maxLen=len(vars)).

Implementation strategy: DFA via Regular
  - Build a deterministic finite automaton whose states encode
    "currently in a run of value v of length c" for c ∈ [1..maxLen[v]].
  - Transitions:
  - From start: on symbol v → state (v,1)
  - From (v,c) on symbol v:
    if c < maxLen[v], go to (v,c+1); else no transition (forbid > max)
  - From (v,c) on symbol w ≠ v:
    allowed iff c ≥ minLen[v], then go to (w,1); else no transition
  - Accepting states are exactly those (v,c) with c ≥ minLen[v], ensuring that
    the final run also satisfies its minimum length.

This reduction achieves strong propagation using the existing Regular
constraint (forward/backward DFA filtering), composes cleanly with other
constraints, and avoids technical debt.

Package minikanren: global constraints - LinearSum (bounds propagation)

LinearSum enforces an equality between a weighted sum of FD variables and
an FD "total" variable using bounds-consistent propagation. This is a
production-ready constraint for modeling many arithmetic relations
(e.g., resource limits, cost-benefit models, profit maximization) while
preserving the solver's immutable, lock-free semantics.

Design
- Variables: x[0..n-1] with domains over positive integers (1..Max)
- Coefficients: arbitrary integers a[i] (positive, negative, or zero)
- Total: t with domain over positive integers (1..Max)
- Relation: sum(i) a[i]*x[i] = t

Propagation (bounds consistency):
  - Prune t to [SumMin..SumMax], where
    SumMin = Σ (a[i]>0 ? a[i]*min(x[i]) : a[i]*max(x[i]))
    SumMax = Σ (a[i]>0 ? a[i]*max(x[i]) : a[i]*min(x[i]))
  - For each x[k], derive admissible interval:
    a[k]*x[k] ∈ [t.min - OtherMax, t.max - OtherMin]
    Convert to bounds on x[k] using sign-aware ceil/floor division and prune.

Notes
  - Mixed-sign coefficients are fully supported; negative coefficients enable
    profit maximization, cost-benefit analysis, and offset modeling.
  - If any variable or total is empty, the solver will detect via domain checks
    and return an error (inconsistency).
  - This constraint is intentionally bounds-only (interval reasoning). It is
    fast and safe; value-level pruning would require heavier algorithms.

Package minikanren: global constraint - Table (extensional constraint)

Table enforces that the n-tuple of FD variables (vars[0],...,vars[n-1])
must be exactly equal to one of the rows in a fixed table of allowed tuples.

Propagation (generalized arc consistency over the fixed table in one pass):
 1. Discard any table row that is incompatible with current domains.
 2. For each variable i, collect the set of values that appear at column i in
    at least one remaining compatible row (a support).
 3. Prune each variable's domain to the supported set.

Notes
  - Tuples must be positive integers to respect Domain invariants (1-based).
  - Rows may contain repeated values; rows may be duplicated; both are handled.
  - Propagation is monotonic; if pruning happens, the solver will call this
    constraint again during the fixed-point loop for further pruning.
  - If no compatible rows remain, the constraint signals inconsistency.

Package minikanren provides SLG (Linear resolution with Selection function for General logic programs)
tabling infrastructure for terminating recursive queries and improving performance through memoization.

# What is Tabling?

Tabling (also called tabulation or memoization for logic programs) is a technique that:
  - Prevents infinite loops in recursive relations by detecting and resolving cycles
  - Improves performance by caching and reusing intermediate results
  - Enables negation through stratification and well-founded semantics
  - Guarantees termination for a broad class of programs

# SLG Resolution

SLG combines:
  - SLD resolution (standard Prolog/miniKanren evaluation)
  - Tabling to handle recursion through fixpoint computation
  - Well-Founded Semantics for stratified negation

# Architecture

The tabling infrastructure uses lock-free data structures for parallel evaluation:
  - AnswerTrie: Stores answer substitutions with structural sharing
  - SubgoalTable: Maps call patterns to cached results using sync.Map
  - CallPattern: Normalized representation of subgoal calls for efficient lookup

All data structures are designed for concurrent access and follow the same
copy-on-write and pooling patterns as the core solver (Phase 1-4).

Package minikanren provides adapters for integrating UnifiedStore
with the ConstraintStore interface, enabling hybrid pldb queries.

The UnifiedStoreAdapter bridges between the UnifiedStore (Phase 3 hybrid solver)
and the ConstraintStore interface used by miniKanren goals. This adapter enables
pldb queries to work seamlessly with FD constraints and bidirectional propagation.

Design rationale:
  - UnifiedStore has methods that return (*UnifiedStore, error) for immutability
  - ConstraintStore interface expects methods that return error for in-place modification
  - Adapter maintains a reference to current store version and updates on mutations
  - Thread-safe through UnifiedStore's immutability and adapter's synchronization

Usage pattern:

	store := NewUnifiedStore()
	adapter := NewUnifiedStoreAdapter(store)

	// Use with pldb queries
	stream := db.Query(person, name, age)(ctx, adapter)

	// Access underlying store for hybrid solver propagation
	hybridStore := adapter.UnifiedStore()
	propagatedStore, err := solver.Propagate(hybridStore)
	adapter.SetUnifiedStore(propagatedStore)

Package minikanren provides constraint programming abstractions.
This file defines the Variable interface for constraint variables
that can hold domains and participate in constraints.

Package minikanren provides a thread-safe parallel implementation of miniKanren in Go.

Version: 1.0.1

This package offers a complete set of miniKanren operators with high-performance
concurrent execution capabilities, designed for production use.


## Constants

### Infinity

Infinity provides a sentinel large positive value for tests/examples when initializing bounds.


```go
&{<nil> [Infinity] <nil> [0xc0001405d0] <nil>}
```

### Version

Version represents the current version of the gokanlogic miniKanren implementation.


```go
&{<nil> [Version] <nil> [0xc000947d40] <nil>}
```

## Variables

### ErrInconsistent, ErrInvalidValue, ErrDomainEmpty, ErrInvalidArgument

FD errors


```go
&{<nil> [ErrInconsistent] <nil> [0xc000695440] <nil>}&{<nil> [ErrInvalidValue] <nil> [0xc000695480] <nil>}&{<nil> [ErrDomainEmpty] <nil> [0xc0006954c0] <nil>}&{<nil> [ErrInvalidArgument] <nil> [0xc000695540] <nil>}
```

### CommonIrrationals

CommonIrrationals provides pre-computed rational approximations for common irrational constants.


```go
&{<nil> [CommonIrrationals] <nil> [0xc000742c80] <nil>}
```

### ErrSearchLimitReached

ErrSearchLimitReached indicates an optimization run terminated due to a configured search limit
(e.g., node limit). The returned incumbent is valid but optimality may not be proven.


```go
&{<nil> [ErrSearchLimitReached] <nil> [0xc0005fa2c0] <nil>}
```

### Nil

Nil represents the empty list


```go
&{<nil> [Nil] <nil> [0xc00039cb40] <nil>}
```

## Types

### AbsenceConstraint
AbsenceConstraint implements the absence constraint (absento). It ensures that a specific term does not occur anywhere within another term's structure, providing structural constraint checking. This constraint performs recursive structural inspection to detect the presence of the forbidden term at any level of nesting.

#### Example Usage

```go
// Create a new AbsenceConstraint
absenceconstraint := AbsenceConstraint{
    id: "example",
    absent: Term{},
    container: Term{},
    isLocal: true,
}
```

#### Type Definition

```go
type AbsenceConstraint struct {
    id string
    absent Term
    container Term
    isLocal bool
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| id | `string` | id uniquely identifies this constraint instance |
| absent | `Term` | absent is the term that must not occur |
| container | `Term` | container is the term that must not contain the absent term |
| isLocal | `bool` | isLocal indicates whether this constraint can be checked locally |

### Constructor Functions

### NewAbsenceConstraint

NewAbsenceConstraint creates a new absence constraint.

```go
func NewAbsenceConstraint(absent, container Term) *AbsenceConstraint
```

**Parameters:**
- `absent` (Term)
- `container` (Term)

**Returns:**
- *AbsenceConstraint

## Methods

### Check

Check evaluates the absence constraint against current bindings. Returns ConstraintViolated if the absent term is found in the container, ConstraintPending if variables are unbound, or ConstraintSatisfied otherwise. Implements the Constraint interface.

```go
func (*AlphaEqConstraint) Check(bindings map[int64]Term) ConstraintResult
```

**Parameters:**
- `bindings` (map[int64]Term)

**Returns:**
- ConstraintResult

### Clone

Clone creates a deep copy of the constraint for parallel execution. Implements the Constraint interface.

```go
func (*MembershipConstraint) Clone() Constraint
```

**Parameters:**
  None

**Returns:**
- Constraint

### ID

ID returns the unique identifier for this constraint instance. Implements the Constraint interface.

```go
func (*MembershipConstraint) ID() string
```

**Parameters:**
  None

**Returns:**
- string

### IsLocal

IsLocal returns true if this constraint can be evaluated locally. Implements the Constraint interface.

```go
func (*MembershipConstraint) IsLocal() bool
```

**Parameters:**
  None

**Returns:**
- bool

### String

String returns a human-readable representation of the constraint. Implements the Constraint interface.

```go
func (*Lexicographic) String() string
```

**Parameters:**
  None

**Returns:**
- string

### Variables

Variables returns the logic variables this constraint depends on. Implements the Constraint interface.

```go
func (*IntervalArithmetic) Variables() []*FDVariable
```

**Parameters:**
  None

**Returns:**
- []*FDVariable

### Absolute
- Both variables must be initialized with proper offset-encoded domains - abs_value domain contains only positive results (≥ 1) Mathematical Properties: - |x| ≥ 0 for all real x, but BitSetDomain requires ≥ 1 - |0| = 0 is represented as offset value in the encoding - |-x| = |x| creates symmetry in backward propagation - Self-reference |x| = x implies x ≥ 0 Thread Safety: Immutable after construction. Propagate() is safe for concurrent use.

#### Example Usage

```go
// Create a new Absolute
absolute := Absolute{
    x: &FDVariable{}{},
    absValue: &FDVariable{}{},
    offset: 42,
}
```

#### Type Definition

```go
type Absolute struct {
    x *FDVariable
    absValue *FDVariable
    offset int
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| x | `*FDVariable` | The input value (offset-encoded for negative support) |
| absValue | `*FDVariable` | The absolute value result (always positive ≥ 1) |
| offset | `int` | Offset used to encode negative values as positive |

### Constructor Functions

### NewAbsolute

NewAbsolute creates a new absolute value constraint: abs_value = |x|. The constraint uses offset encoding to represent negative numbers within BitSetDomain constraints. For an offset O, the encoding is: - Negative value -k is encoded as O - k - Zero is encoded as O - Positive value k is encoded as O + k

```go
func NewAbsolute(x *FDVariable, offset int, absValue *FDVariable) (*Absolute, error)
```

**Parameters:**

- `x` (*FDVariable) - The FD variable representing the input value (offset-encoded)

- `offset` (int) - The offset used for encoding negative values (must be > 0)

- `absValue` (*FDVariable) - The FD variable representing the absolute value (always ≥ 1)

**Returns:**
- *Absolute
- error

## Methods

### Clone

Clone creates an independent copy of this constraint.

```go
func (*Absolute) Clone() PropagationConstraint
```

**Parameters:**
  None

**Returns:**
- PropagationConstraint

### Propagate

Propagate performs bidirectional arc-consistency enforcement for the absolute value constraint. Algorithm: 1. Check for self-reference (x = absValue) and handle specially 2. Forward propagation: Compute |x| values and prune absValue domain 3. Backward propagation: For each |x| value, find corresponding x values 4. Apply domain changes and detect failures The constraint maintains: absValue = |decode(x)| where decode(x) = x - offset. Returns the updated solver state, or error if the constraint is unsatisfiable.

```go
func (*Inequality) Propagate(solver *Solver, state *SolverState) (*SolverState, error)
```

**Parameters:**
- `solver` (*Solver)
- `state` (*SolverState)

**Returns:**
- *SolverState
- error

### String

String returns a human-readable representation of the constraint.

```go
func (*DistinctCount) String() string
```

**Parameters:**
  None

**Returns:**
- string

### Type

Type returns the constraint type name.

```go
func (*Circuit) Type() string
```

**Parameters:**
  None

**Returns:**
- string

### Variables

Variables returns the FD variables involved in this constraint.

```go
func (*GlobalCardinality) Variables() []*FDVariable
```

**Parameters:**
  None

**Returns:**
- []*FDVariable

### backwardPropagate

backwardPropagate prunes the x domain based on absValue values. For each value a in absValue.domain: - Compute the original values that produce |x| = a - These are: x = a and x = -a (in offset encoding) - Offset encoding: +a → offset + a, -a → offset - a - Add valid encoded values to possible x values Returns a new domain with only feasible x values.

```go
func (*Scale) backwardPropagate(resultDomain, xDomain Domain) Domain
```

**Parameters:**
- `resultDomain` (Domain)
- `xDomain` (Domain)

**Returns:**
- Domain

### computeAbsolute

computeAbsolute computes the absolute value of an offset-encoded x value. Algorithm: - Decode: actual_value = x - offset - Compute: abs_value = |actual_value| - Handle BitSetDomain constraint: if abs_value = 0, return 1 This handles the BitSetDomain requirement that all values ≥ 1.

```go
func (*Absolute) computeAbsolute(x int) int
```

**Parameters:**
- `x` (int)

**Returns:**
- int

### forwardPropagate

forwardPropagate prunes the absValue domain based on x values. For each value v in x.domain: - Decode: actual_value = v - offset - Compute: abs_actual = |actual_value| - Encode: abs_encoded = abs_actual (but ensure ≥ 1 for BitSetDomain) - Add abs_encoded to possible absValue values Returns a new domain with only feasible absolute values.

```go
func (*Absolute) forwardPropagate(xDomain, absValueDomain Domain) Domain
```

**Parameters:**
- `xDomain` (Domain)
- `absValueDomain` (Domain)

**Returns:**
- Domain

### handleSelfReference

handleSelfReference handles the special case where |x| = x. This is only valid when x ≥ 0 (in offset encoding: x ≥ offset).

```go
func (*Modulo) handleSelfReference(solver *Solver, state *SolverState, xDomain Domain) (*SolverState, error)
```

**Parameters:**
- `solver` (*Solver)
- `state` (*SolverState)
- `xDomain` (Domain)

**Returns:**
- *SolverState
- error

### AllDifferent
Implementation uses Regin's arc-consistency algorithm based on maximum bipartite matching. This achieves stronger pruning than pairwise inequality: Example: X,Y,Z ∈ {1,2} with AllDifferent(X,Y,Z) - Matching algorithm detects impossibility (3 variables, 2 values) - Fails immediately without search - Pairwise X≠Y, Y≠Z, X≠Z would only fail after trying assignments Algorithm complexity: O(n²·d) where n = |variables|, d = max domain size Much more efficient than the exponential search that would be required otherwise.

#### Example Usage

```go
// Create a new AllDifferent
alldifferent := AllDifferent{
    variables: [],
}
```

#### Type Definition

```go
type AllDifferent struct {
    variables []*FDVariable
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| variables | `[]*FDVariable` |  |

### Constructor Functions

### NewAllDifferent

NewAllDifferent creates an AllDifferent constraint over the given variables. Returns error if variables is nil or empty.

```go
func NewAllDifferent(variables []*FDVariable) (*AllDifferent, error)
```

**Parameters:**
- `variables` ([]*FDVariable)

**Returns:**
- *AllDifferent
- error

## Methods

### Propagate

Propagate applies Regin's AllDifferent filtering algorithm. Implements PropagationConstraint.

```go
func (*ReifiedConstraint) Propagate(solver *Solver, state *SolverState) (*SolverState, error)
```

**Parameters:**
- `solver` (*Solver)
- `state` (*SolverState)

**Returns:**
- *SolverState
- error

### String

String returns a human-readable representation. Implements ModelConstraint.

```go
func (*RationalLinearSum) String() string
```

**Parameters:**
  None

**Returns:**
- string

### Type

Type returns the constraint type identifier. Implements ModelConstraint.

```go
func (*BoolSum) Type() string
```

**Parameters:**
  None

**Returns:**
- string

### Variables

Variables returns the variables involved in this constraint. Implements ModelConstraint.

```go
func (*BinPacking) Variables() []*FDVariable
```

**Parameters:**
  None

**Returns:**
- []*FDVariable

### augment

augment finds augmenting path for variable vi using DFS.

```go
func (*AllDifferent) augment(vi int, domains []Domain, matchVal, matchVar []int, visited []bool, maxVal int) bool
```

**Parameters:**
- `vi` (int)
- `domains` ([]Domain)
- `matchVal` ([]int)
- `matchVar` ([]int)
- `visited` ([]bool)
- `maxVal` (int)

**Returns:**
- bool

### buildValueGraph



```go
func (*AllDifferent) buildValueGraph(domains []Domain, matching map[int]int, n, maxVal int) *valueGraph
```

**Parameters:**
- `domains` ([]Domain)
- `matching` (map[int]int)
- `n` (int)
- `maxVal` (int)

**Returns:**
- *valueGraph

### computeSCCs

computeSCCs computes strongly connected components using Tarjan's algorithm. Returns scc[node] = component ID for each node.

```go
func (*AllDifferent) computeSCCs(g *valueGraph, n, maxVal int) []int
```

**Parameters:**
- `g` (*valueGraph)
- `n` (int)
- `maxVal` (int)

**Returns:**
- []int

### maxMatching

Returns mapping from value to variable index, and matching size.

```go
func (*AllDifferent) maxMatching(domains []Domain, maxVal int) (map[int]int, int)
```

**Parameters:**
- `domains` ([]Domain)
- `maxVal` (int)

**Returns:**
- map[int]int
- int

### AllDifferentConstraint
AllDifferentConstraint is a custom version of the all-different constraint This demonstrates how built-in constraints can be reimplemented as custom constraints

#### Example Usage

```go
// Create a new AllDifferentConstraint
alldifferentconstraint := AllDifferentConstraint{
    vars: [],
}
```

#### Type Definition

```go
type AllDifferentConstraint struct {
    vars []*FDVar
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| vars | `[]*FDVar` |  |

### Constructor Functions

### NewAllDifferentConstraint

NewAllDifferentConstraint creates a new all-different constraint

```go
func NewAllDifferentConstraint(vars []*FDVar) *AllDifferentConstraint
```

**Parameters:**
- `vars` ([]*FDVar)

**Returns:**
- *AllDifferentConstraint

## Methods

### IsSatisfied

IsSatisfied checks if all variables have distinct values

```go
func (*AllDifferentConstraint) IsSatisfied() bool
```

**Parameters:**
  None

**Returns:**
- bool

### Propagate

Propagate performs constraint propagation for all-different

```go
func (*Among) Propagate(solver *Solver, state *SolverState) (*SolverState, error)
```

**Parameters:**
- `solver` (*Solver)
- `state` (*SolverState)

**Returns:**
- *SolverState
- error

### Variables

Variables returns the variables involved in this constraint

```go
func (*Regular) Variables() []*FDVariable
```

**Parameters:**
  None

**Returns:**
- []*FDVariable

### AlphaEqConstraint
AlphaEqConstraint checks alpha-equivalence between two terms (Tie-aware).

#### Example Usage

```go
// Create a new AlphaEqConstraint
alphaeqconstraint := AlphaEqConstraint{
    id: "example",
    left: Term{},
    right: Term{},
}
```

#### Type Definition

```go
type AlphaEqConstraint struct {
    id string
    left Term
    right Term
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| id | `string` |  |
| left | `Term` |  |
| right | `Term` |  |

### Constructor Functions

### NewAlphaEqConstraint

NewAlphaEqConstraint constructs the constraint object.

```go
func NewAlphaEqConstraint(left, right Term) *AlphaEqConstraint
```

**Parameters:**
- `left` (Term)
- `right` (Term)

**Returns:**
- *AlphaEqConstraint

## Methods

### Check



```go
func (*LessEqualConstraint) Check(bindings map[int64]Term) ConstraintResult
```

**Parameters:**
- `bindings` (map[int64]Term)

**Returns:**
- ConstraintResult

### Clone



```go
func (*RationalLinearSum) Clone() ModelConstraint
```

**Parameters:**
  None

**Returns:**
- ModelConstraint

### ID



```go
func (*LocalConstraintStoreImpl) ID() string
```

**Parameters:**
  None

**Returns:**
- string

### IsLocal



```go
func (*MembershipConstraint) IsLocal() bool
```

**Parameters:**
  None

**Returns:**
- bool

### String



```go
func (*Lexicographic) String() string
```

**Parameters:**
  None

**Returns:**
- string

### Variables



```go
func (*MembershipConstraint) Variables() []*Var
```

**Parameters:**
  None

**Returns:**
- []*Var

### Among
Among is a global constraint that counts how many variables take values from S.

#### Example Usage

```go
// Create a new Among
among := Among{
    vars: [],
    set: Domain{},
    k: &FDVariable{}{},
}
```

#### Type Definition

```go
type Among struct {
    vars []*FDVariable
    set Domain
    k *FDVariable
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| vars | `[]*FDVariable` |  |
| set | `Domain` | bitset mask for S over [1..maxV] |
| k | `*FDVariable` | encoded count: value = count+1 |

## Methods

### Propagate

Propagate enforces bounds-consistent pruning for Among.

```go
func (*EqualityReified) Propagate(solver *Solver, state *SolverState) (*SolverState, error)
```

**Parameters:**
- `solver` (*Solver)
- `state` (*SolverState)

**Returns:**
- *SolverState
- error

### String

String returns a human-readable description.

```go
func (*GlobalCardinality) String() string
```

**Parameters:**
  None

**Returns:**
- string

### Type

Type names the constraint.

```go
func (*DistinctCount) Type() string
```

**Parameters:**
  None

**Returns:**
- string

### Variables

Variables returns all variables involved (vars plus K).

```go
func (*Among) Variables() []*FDVariable
```

**Parameters:**
  None

**Returns:**
- []*FDVariable

### AnswerIterator
AnswerIterator iterates over answers in insertion order.

#### Example Usage

```go
// Create a new AnswerIterator
answeriterator := AnswerIterator{
    snapshot: [],
    idx: 42,
    mu: /* value */,
}
```

#### Type Definition

```go
type AnswerIterator struct {
    snapshot []map[int64]Term
    idx int
    mu sync.Mutex
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| snapshot | `[]map[int64]Term` | snapshot holds a point-in-time copy of the trie's answer slice headers. Individual answers are still copied on return by Next() to prevent external mutation. To observe new answers appended after iterator creation, construct a new iterator. |
| idx | `int` |  |
| mu | `sync.Mutex` | Protects idx |

## Methods

### Next

Next returns the next answer or nil if exhausted. Thread safety: This method uses internal locks and is safe to call from multiple goroutines, but using a single goroutine per iterator preserves deterministic ordering and minimizes contention.

```go
func (*AnswerIterator) Next() (map[int64]Term, bool)
```

**Parameters:**
  None

**Returns:**
- map[int64]Term
- bool

### AnswerRecord
AnswerRecord bundles an answer's bindings with its WFS delay set. If Delay is empty, the answer is unconditional.

#### Example Usage

```go
// Create a new AnswerRecord
answerrecord := AnswerRecord{
    Bindings: map[],
    Delay: DelaySet{},
}
```

#### Type Definition

```go
type AnswerRecord struct {
    Bindings map[int64]Term
    Delay DelaySet
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| Bindings | `map[int64]Term` |  |
| Delay | `DelaySet` |  |

### AnswerRecordIterator
AnswerRecordIterator is a metadata-aware iterator that wraps the existing AnswerIterator and pairs each binding with a DelaySet provided by a callback. The callback allows us to wire per-answer metadata later without changing the current AnswerTrie layout.

#### Example Usage

```go
// Create a new AnswerRecordIterator
answerrecorditerator := AnswerRecordIterator{
    inner: &AnswerIterator{}{},
    startIndex: 42,
    delayProvider: /* value */,
    include: /* value */,
}
```

#### Type Definition

```go
type AnswerRecordIterator struct {
    inner *AnswerIterator
    startIndex int
    delayProvider func(index int) DelaySet
    include func(index int) bool
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| inner | `*AnswerIterator` |  |
| startIndex | `int` |  |
| delayProvider | `func(index int) DelaySet` | nil provider yields empty delay sets |
| include | `func(index int) bool` | optional visibility filter; nil => include all |

### Constructor Functions

### NewAnswerRecordIterator

NewAnswerRecordIterator constructs a metadata-aware iterator over the given trie starting at index 0. Delay metadata is supplied by delayProvider; pass nil to provide empty delay sets (unconditional semantics).

```go
func NewAnswerRecordIterator(trie *AnswerTrie, delayProvider func(index int) DelaySet) *AnswerRecordIterator
```

**Parameters:**
- `trie` (*AnswerTrie)
- `delayProvider` (func(index int) DelaySet)

**Returns:**
- *AnswerRecordIterator

### NewAnswerRecordIteratorFrom

NewAnswerRecordIteratorFrom constructs a metadata-aware iterator starting at start.

```go
func NewAnswerRecordIteratorFrom(trie *AnswerTrie, start int, delayProvider func(index int) DelaySet) *AnswerRecordIterator
```

**Parameters:**
- `trie` (*AnswerTrie)
- `start` (int)
- `delayProvider` (func(index int) DelaySet)

**Returns:**
- *AnswerRecordIterator

## Methods

### Next

Next returns the next AnswerRecord or ok=false when exhausted.

```go
func (*AnswerIterator) Next() (map[int64]Term, bool)
```

**Parameters:**
  None

**Returns:**
- map[int64]Term
- bool

### WithInclude

WithInclude sets a visibility predicate for the iterator.

```go
func (*AnswerRecordIterator) WithInclude(include func(index int) bool) *AnswerRecordIterator
```

**Parameters:**
- `include` (func(index int) bool)

**Returns:**
- *AnswerRecordIterator

### AnswerTrie
AnswerTrie represents a trie of answer substitutions for a tabled subgoal. Uses structural sharing to minimize memory overhead. Thread safety: The trie supports concurrent reads, and writes are coordinated via an internal mutex to ensure safety. Iteration returns copies of stored answers to prevent external mutation. In typical usage, writes are also coordinated at a higher level (e.g., by SubgoalEntry) to avoid unnecessary contention.

#### Example Usage

```go
// Create a new AnswerTrie
answertrie := AnswerTrie{
    root: &AnswerTrieNode{}{},
    answers: [],
    count: /* value */,
    nodePool: &/* value */{},
    mu: /* value */,
}
```

#### Type Definition

```go
type AnswerTrie struct {
    root *AnswerTrieNode
    answers []map[int64]Term
    count atomic.Int64
    nodePool *sync.Pool
    mu sync.Mutex
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| root | `*AnswerTrieNode` | Root node of the trie |
| answers | `[]map[int64]Term` | Ordered list of answers for deterministic iteration |
| count | `atomic.Int64` | Cached answer count for O(1) size queries |
| nodePool | `*sync.Pool` | Pool for trie nodes (zero-allocation reuse) |
| mu | `sync.Mutex` | Mutex for coordinating insertions |

### Constructor Functions

### NewAnswerTrie

NewAnswerTrie creates an empty answer trie.

```go
func NewAnswerTrie() *AnswerTrie
```

**Parameters:**
  None

**Returns:**
- *AnswerTrie

## Methods

### Count

Count returns the number of answers in the trie.

```go
func (BitSet) Count() int
```

**Parameters:**
  None

**Returns:**
- int

### Insert

Insert adds an answer to the trie. Returns true if the answer was new, false if it was a duplicate. Answers are represented as variable bindings. The trie is organized by variable ID, with each path representing a complete answer.

```go
func (*AnswerTrie) Insert(bindings map[int64]Term) bool
```

**Parameters:**
- `bindings` (map[int64]Term)

**Returns:**
- bool

### Iterator

Iterator returns an iterator over all answers in the trie. Answers are returned in insertion order for deterministic iteration. The iterator creates a snapshot of the answer list to avoid concurrent modification issues during iteration.

```go
func (*AnswerTrie) Iterator() *AnswerIterator
```

**Parameters:**
  None

**Returns:**
- *AnswerIterator

### IteratorFrom

IteratorFrom returns an iterator starting at the given index over a snapshot of the current answers. If start >= len(snapshot), the iterator is exhausted. Use this to resume iteration without re-reading already-consumed answers when new answers may have been appended concurrently.

```go
func (*AnswerTrie) IteratorFrom(start int) *AnswerIterator
```

**Parameters:**
- `start` (int)

**Returns:**
- *AnswerIterator

### AnswerTrieNode
AnswerTrieNode represents a node in the answer trie. Thread safety: children map is protected by the trie's global mutex during writes, and is safe for concurrent reads after insertion since nodes are structurally shared.

#### Example Usage

```go
// Create a new AnswerTrieNode
answertrienode := AnswerTrieNode{
    varID: 42,
    value: Term{},
    children: map[],
    isAnswer: true,
    depth: 42,
}
```

#### Type Definition

```go
type AnswerTrieNode struct {
    varID int64
    value Term
    children map[nodeKey]*AnswerTrieNode
    isAnswer bool
    depth int
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| varID | `int64` | Variable ID at this level (-1 for root) |
| value | `Term` | Bound value at this node (nil if unbound) |
| children | `map[nodeKey]*AnswerTrieNode` | Children indexed by (varID, valueHash) pairs Protected by trie-level mutex during modifications |
| isAnswer | `bool` | Marks this as a complete answer (leaf node) |
| depth | `int` | Depth in trie (for debugging) |

### Arithmetic
Provides bidirectional arc-consistency: - Forward: dst ∈ {src + offset | src ∈ Domain(src)} - Backward: src ∈ {dst - offset | dst ∈ Domain(dst)} Example: X + 3 = Y with X ∈ {1,2,5}, Y ∈ {1,2,3,4,5,6,7,8} - Forward prunes: Y restricted to {4,5,8} - Backward prunes: X restricted to {1,2,5} (no change, already consistent) Useful for modeling derived variables in problems like N-Queens where diagonal constraints are column ± row offset.

#### Example Usage

```go
// Create a new Arithmetic
arithmetic := Arithmetic{
    src: &FDVariable{}{},
    dst: &FDVariable{}{},
    offset: 42,
}
```

#### Type Definition

```go
type Arithmetic struct {
    src *FDVariable
    dst *FDVariable
    offset int
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| src | `*FDVariable` |  |
| dst | `*FDVariable` |  |
| offset | `int` |  |

### Constructor Functions

### NewArithmetic

NewArithmetic creates dst = src + offset constraint. Returns error if src or dst is nil.

```go
func NewArithmetic(src, dst *FDVariable, offset int) (*Arithmetic, error)
```

**Parameters:**
- `src` (*FDVariable)
- `dst` (*FDVariable)
- `offset` (int)

**Returns:**
- *Arithmetic
- error

## Methods

### Propagate

Propagate applies bidirectional arc-consistency. Implements PropagationConstraint.

```go
func (*RationalLinearSum) Propagate(solver *Solver, state *SolverState) (*SolverState, error)
```

**Parameters:**
- `solver` (*Solver)
- `state` (*SolverState)

**Returns:**
- *SolverState
- error

### String

String returns human-readable representation. Implements ModelConstraint.

```go
func (*Among) String() string
```

**Parameters:**
  None

**Returns:**
- string

### Type

Type returns "Arithmetic". Implements ModelConstraint.

```go
func (*Lexicographic) Type() string
```

**Parameters:**
  None

**Returns:**
- string

### Variables

Variables returns [src, dst]. Implements ModelConstraint.

```go
func (*Absolute) Variables() []*FDVariable
```

**Parameters:**
  None

**Returns:**
- []*FDVariable

### eq

eq checks domain equality by comparing values.

```go
func (*Arithmetic) eq(d1, d2 Domain) bool
```

**Parameters:**
- `d1` (Domain)
- `d2` (Domain)

**Returns:**
- bool

### imageForTarget

imageForTarget computes {v + offset | v ∈ dom} with the result having targetMaxValue. This ensures the result can be intersected with the target domain.

```go
func (*Arithmetic) imageForTarget(dom Domain, offset, targetMaxValue int) Domain
```

**Parameters:**
- `dom` (Domain)
- `offset` (int)
- `targetMaxValue` (int)

**Returns:**
- Domain

### Atom
Atom represents an atomic value (symbol, number, string, etc.). Atoms are immutable and represent themselves.

#### Example Usage

```go
// Create a new Atom
atom := Atom{
    value: /* value */,
}
```

#### Type Definition

```go
type Atom struct {
    value interface{}
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| value | `interface{}` | The underlying Go value |

### Constructor Functions

### AtomFromValue

AtomFromValue creates a new atomic term from any Go value. This is a convenience function that's equivalent to NewAtom.

```go
func AtomFromValue(value interface{}) *Atom
```

**Parameters:**
- `value` (interface{})

**Returns:**
- *Atom

### NewAtom

NewAtom creates a new atom from any Go value.

```go
func NewAtom(value interface{}) *Atom
```

**Parameters:**
- `value` (interface{})

**Returns:**
- *Atom

### NomFresh

NomFresh generates fresh nominal name atoms with unique suffixes to avoid accidental clashes. If names are provided, they're used as prefixes; otherwise "n" is used.

```go
func NomFresh(prefix string) *Atom
```

**Parameters:**
- `prefix` (string)

**Returns:**
- *Atom

### freeNamesDet

freeNamesDet computes the set of free nominal names in term. Returns a sorted slice of *Atom and ok=false if pending due to unknown vars.

```go
func freeNamesDet(term Term) ([]*Atom, bool)
```

**Parameters:**
- `term` (Term)

**Returns:**
- []*Atom
- bool

## Methods

### Clone

Clone creates a copy of the atom.

```go
func (*LocalConstraintStoreImpl) Clone() ConstraintStore
```

**Parameters:**
  None

**Returns:**
- ConstraintStore

### Equal

Equal checks if two atoms have the same value.

```go
func (*BitSetDomain) Equal(other Domain) bool
```

**Parameters:**
- `other` (Domain)

**Returns:**
- bool

### IsVar

IsVar always returns false for atoms.

```go
func (*Pair) IsVar() bool
```

**Parameters:**
  None

**Returns:**
- bool

### String

String returns a string representation of the atom.

```go
func (*MembershipConstraint) String() string
```

**Parameters:**
  None

**Returns:**
- string

### Value

Value returns the underlying Go value.

```go
func (*FDVariable) Value() int
```

**Parameters:**
  None

**Returns:**
- int

### BinPacking
_No documentation available_

#### Example Usage

```go
// Create a new BinPacking
binpacking := BinPacking{
    items: [],
    sizes: [],
    capacities: [],
    m: 42,
    binBools: [],
    binSums: [],
    binLoads: [],
    reifs: [],
    sums: [],
    ties: [],
}
```

#### Type Definition

```go
type BinPacking struct {
    items []*FDVariable
    sizes []int
    capacities []int
    m int
    binBools [][]*FDVariable
    binSums []*FDVariable
    binLoads []*FDVariable
    reifs [][]PropagationConstraint
    sums []PropagationConstraint
    ties []PropagationConstraint
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| items | `[]*FDVariable` |  |
| sizes | `[]int` |  |
| capacities | `[]int` |  |
| m | `int` |  |
| binBools | `[][]*FDVariable` | per-bin artifacts (for introspection) |
| binSums | `[]*FDVariable` | sum_k variables (Σ size[i]*b[i,k]) |
| binLoads | `[]*FDVariable` | LkPlus1 variables encoding load+1 ≤ cap+1 |
| reifs | `[][]PropagationConstraint` |  |
| sums | `[]PropagationConstraint` |  |
| ties | `[]PropagationConstraint` | Arithmetic links load to sum |

### Constructor Functions

### NewBinPacking

NewBinPacking constructs the capacity constraints for m bins.

```go
func NewBinPacking(model *Model, items []*FDVariable, sizes []int, capacities []int) (*BinPacking, error)
```

**Parameters:**

- `model` (*Model) - hosting model

- `items` ([]*FDVariable) - variables with domains ⊆ {1..m}

- `sizes` ([]int) - positive integers (len = len(items))

- `capacities` ([]int) - positive integers (len = m)

**Returns:**
- *BinPacking
- error

## Methods

### Propagate



```go
func (*Lexicographic) Propagate(solver *Solver, state *SolverState) (*SolverState, error)
```

**Parameters:**
- `solver` (*Solver)
- `state` (*SolverState)

**Returns:**
- *SolverState
- error

### String



```go
func (*Inequality) String() string
```

**Parameters:**
  None

**Returns:**
- string

### Type



```go
func (*IntervalArithmetic) Type() string
```

**Parameters:**
  None

**Returns:**
- string

### Variables



```go
func (*Lexicographic) Variables() []*FDVariable
```

**Parameters:**
  None

**Returns:**
- []*FDVariable

### BitSet
Generic BitSet-backed Domain for FD variables. Values are 1-based indices.

#### Example Usage

```go
// Create a new BitSet
bitset := BitSet{
    n: 42,
    words: [],
}
```

#### Type Definition

```go
type BitSet struct {
    n int
    words []uint64
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| n | `int` |  |
| words | `[]uint64` |  |

### Constructor Functions

### NewBitSet



```go
func NewBitSet(n int) BitSet
```

**Parameters:**
- `n` (int)

**Returns:**
- BitSet

### imageOfDomain

imageOfDomain returns a BitSet representing {v+offset | v in dom} intersected with 1..n

```go
func imageOfDomain(dom BitSet, offset int, n int) BitSet
```

**Parameters:**
- `dom` (BitSet)
- `offset` (int)
- `n` (int)

**Returns:**
- BitSet

### intersectBitSet



```go
func intersectBitSet(a, b BitSet) BitSet
```

**Parameters:**
- `a` (BitSet)
- `b` (BitSet)

**Returns:**
- BitSet

## Methods

### Clone



```go
func (*Substitution) Clone() *Substitution
```

**Parameters:**
  None

**Returns:**
- *Substitution

### Complement

Complement returns a new BitSet containing all values NOT in this BitSet within the domain 1..n

```go
func (BitSet) Complement() BitSet
```

**Parameters:**
  None

**Returns:**
- BitSet

### Count



```go
func (BitSet) Count() int
```

**Parameters:**
  None

**Returns:**
- int

### Has



```go
func (BitSet) Has(v int) bool
```

**Parameters:**
- `v` (int)

**Returns:**
- bool

### Intersect

Intersect returns a new BitSet containing values present in both this and other BitSet

```go
func (BitSet) Intersect(other BitSet) BitSet
```

**Parameters:**
- `other` (BitSet)

**Returns:**
- BitSet

### IsSingleton



```go
func (*FDVar) IsSingleton() bool
```

**Parameters:**
  None

**Returns:**
- bool

### IterateValues



```go
func (BitSet) IterateValues(f func(v int))
```

**Parameters:**
- `f` (func(v int))

**Returns:**
  None

### RemoveValue



```go
func (BitSet) RemoveValue(v int) BitSet
```

**Parameters:**
- `v` (int)

**Returns:**
- BitSet

### SingletonValue



```go
func (*FDVar) SingletonValue() int
```

**Parameters:**
  None

**Returns:**
- int

### ToSlice

ToSlice returns all values in the domain as a pre-allocated slice. This is more efficient than IterateValues when you need all values at once.

```go
func (*BitSetDomain) ToSlice() []int
```

**Parameters:**
  None

**Returns:**
- []int

### Union

Union returns a new BitSet containing values present in either this or other BitSet

```go
func (BitSet) Union(other BitSet) BitSet
```

**Parameters:**
- `other` (BitSet)

**Returns:**
- BitSet

### BitSetDomain
Values are 1-indexed in the range [1, maxValue]. Each value is represented by a single bit in a uint64 word array, providing O(1) membership testing and very fast set operations. Memory usage: (maxValue + 63) / 64 * 8 bytes Example: maxValue=100 uses 16 bytes (2 uint64 words) BitSetDomain is immutable - all operations return new instances rather than modifying in place. This enables efficient structural sharing and copy-on-write semantics for parallel search.

#### Example Usage

```go
// Create a new BitSetDomain
bitsetdomain := BitSetDomain{
    maxValue: 42,
    words: [],
}
```

#### Type Definition

```go
type BitSetDomain struct {
    maxValue int
    words []uint64
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| maxValue | `int` | Maximum value (inclusive), typically 9 for Sudoku, higher for other problems |
| words | `[]uint64` | Bit array: bit i represents value i+1 |

### Constructor Functions

### NewBitSetDomain

NewBitSetDomain creates a new domain containing all values from 1 to maxValue (inclusive). maxValue must be positive. Uses object pooling for common domain sizes.

```go
func NewBitSetDomain(maxValue int) *BitSetDomain
```

**Parameters:**
- `maxValue` (int)

**Returns:**
- *BitSetDomain

### NewBitSetDomainFromValues

NewBitSetDomainFromValues creates a domain containing only the specified values. Values outside [1, maxValue] are ignored. Uses object pooling for common domain sizes.

```go
func NewBitSetDomainFromValues(maxValue int, values []int) *BitSetDomain
```

**Parameters:**
- `maxValue` (int)
- `values` ([]int)

**Returns:**
- *BitSetDomain

### getDomainFromPool

getDomainFromPool retrieves a BitSetDomain from the appropriate pool. Returns nil if domain would be too large for pooling.

```go
func getDomainFromPool(maxValue int) *BitSetDomain
```

**Parameters:**
- `maxValue` (int)

**Returns:**
- *BitSetDomain

## Methods

### Clone

Clone returns a copy of the domain. O(number of words) operation. Uses object pooling for common domain sizes.

```go
func (*RationalLinearSum) Clone() ModelConstraint
```

**Parameters:**
  None

**Returns:**
- ModelConstraint

### Complement

Complement returns a new domain with all values NOT in this domain. Values are within the range [1, maxValue]. O(number of words) operation.

```go
func (BitSet) Complement() BitSet
```

**Parameters:**
  None

**Returns:**
- BitSet

### Count

Count returns the number of values in the domain. Uses hardware popcount instructions for efficiency (O(number of words)).

```go
func (*AnswerTrie) Count() int64
```

**Parameters:**
  None

**Returns:**
- int64

### Equal

Equal returns true if this domain contains exactly the same values as other. O(number of words) operation.

```go
func (*Pair) Equal(other Term) bool
```

**Parameters:**
- `other` (Term)

**Returns:**
- bool

### Has

Has returns true if the domain contains the value. Values are 1-indexed. O(1) operation.

```go
func (*BitSetDomain) Has(value int) bool
```

**Parameters:**
- `value` (int)

**Returns:**
- bool

### Intersect

Intersect returns a new domain containing values in both this and other. This is the core operation for constraint propagation. O(number of words) operation.

```go
func (BitSet) Intersect(other BitSet) BitSet
```

**Parameters:**
- `other` (BitSet)

**Returns:**
- BitSet

### IsSingleton

IsSingleton returns true if the domain contains exactly one value. O(number of words) operation.

```go
func (*FDVar) IsSingleton() bool
```

**Parameters:**
  None

**Returns:**
- bool

### IterateValues

IterateValues calls f for each value in the domain in ascending order. The function must not retain references to mutable state during iteration.

```go
func (BitSet) IterateValues(f func(v int))
```

**Parameters:**
- `f` (func(v int))

**Returns:**
  None

### Max

Max returns the maximum value in the domain. Returns 0 if domain is empty. O(words) in worst case, but typically O(1) as maximum is in last word.

```go
func (*BitSetDomain) Max() int
```

**Parameters:**
  None

**Returns:**
- int

### MaxValue

MaxValue returns the maximum value that can be in this domain.

```go
func (*BitSetDomain) MaxValue() int
```

**Parameters:**
  None

**Returns:**
- int

### Min

Min returns the minimum value in the domain. Returns 0 if domain is empty. O(words) in worst case, but typically O(1) as minimum is in first word.

```go
func (*BitSetDomain) Min() int
```

**Parameters:**
  None

**Returns:**
- int

### Remove

Remove returns a new domain without the specified value. If the value is not present, returns an equivalent domain. O(number of words) due to array copy.

```go
func (*BitSetDomain) Remove(value int) Domain
```

**Parameters:**
- `value` (int)

**Returns:**
- Domain

### RemoveAbove

RemoveAbove returns a new domain with all values > threshold removed. Uses efficient bit masking - O(words) not O(domain_size). Example: {1,2,3,4,5}.RemoveAbove(3) = {1,2,3}

```go
func (*BitSetDomain) RemoveAbove(threshold int) Domain
```

**Parameters:**
- `threshold` (int)

**Returns:**
- Domain

### RemoveAtOrAbove

RemoveAtOrAbove returns a new domain with all values >= threshold removed. Uses efficient bit masking - O(words) not O(domain_size). Example: {1,2,3,4,5}.RemoveAtOrAbove(3) = {1,2}

```go
func (*BitSetDomain) RemoveAtOrAbove(threshold int) Domain
```

**Parameters:**
- `threshold` (int)

**Returns:**
- Domain

### RemoveAtOrBelow

RemoveAtOrBelow returns a new domain with all values <= threshold removed. Uses efficient bit masking - O(words) not O(domain_size). Example: {1,2,3,4,5}.RemoveAtOrBelow(3) = {4,5}

```go
func (*BitSetDomain) RemoveAtOrBelow(threshold int) Domain
```

**Parameters:**
- `threshold` (int)

**Returns:**
- Domain

### RemoveBelow

RemoveBelow returns a new domain with all values < threshold removed. Uses efficient bit masking - O(words) not O(domain_size). Example: {1,2,3,4,5}.RemoveBelow(3) = {3,4,5}

```go
func (*BitSetDomain) RemoveBelow(threshold int) Domain
```

**Parameters:**
- `threshold` (int)

**Returns:**
- Domain

### SingletonValue

SingletonValue returns the single value in the domain. Panics if the domain is not a singleton. O(number of words) operation.

```go
func (*BitSetDomain) SingletonValue() int
```

**Parameters:**
  None

**Returns:**
- int

### String

String returns a human-readable representation of the domain. Example: "{1,3,5,7,9}" or "{1..100}" for ranges.

```go
func (*Lexicographic) String() string
```

**Parameters:**
  None

**Returns:**
- string

### ToSlice

ToSlice returns all values in the domain as a pre-allocated slice. This is more efficient than IterateValues when you need all values at once.

```go
func (*BitSetDomain) ToSlice() []int
```

**Parameters:**
  None

**Returns:**
- []int

### Union

Union returns a new domain containing values from both this and other. O(number of words) operation.

```go
func (BitSet) Union(other BitSet) BitSet
```

**Parameters:**
- `other` (BitSet)

**Returns:**
- BitSet

### isConsecutiveRange

isConsecutiveRange checks if values form a consecutive range.

```go
func (*BitSetDomain) isConsecutiveRange(values []int) bool
```

**Parameters:**
- `values` ([]int)

**Returns:**
- bool

### BoolSum
Propagation: - Let lb = sum of per-var minimum contributions (1 if var must be true, else 0) - Let ub = sum of per-var maximum contributions (1 if var may be true, else 0) - Prune total to [lb+1, ub+1] - For each var, using otherLb = lb - varMin and otherUb = ub - varMax: - If (total.min-1) > otherUb  => var must be true (set to {2}) - If (total.max-1) < otherLb  => var must be false (set to {1}) This achieves bounds consistency for boolean sums and is sufficient for Count.

#### Example Usage

```go
// Create a new BoolSum
boolsum := BoolSum{
    vars: [],
    total: &FDVariable{}{},
}
```

#### Type Definition

```go
type BoolSum struct {
    vars []*FDVariable
    total *FDVariable
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| vars | `[]*FDVariable` |  |
| total | `*FDVariable` | domain [1..n+1], representing count+1 |

### Constructor Functions

### NewBoolSum

NewBoolSum creates a BoolSum constraint over boolean variables {1,2} and a total in [1..n+1].

```go
func NewBoolSum(vars []*FDVariable, total *FDVariable) (*BoolSum, error)
```

**Parameters:**
- `vars` ([]*FDVariable)
- `total` (*FDVariable)

**Returns:**
- *BoolSum
- error

## Methods

### Propagate

Propagate enforces bounds consistency on the sum of boolean vars.

```go
func (*BinPacking) Propagate(solver *Solver, state *SolverState) (*SolverState, error)
```

**Parameters:**
- `solver` (*Solver)
- `state` (*SolverState)

**Returns:**
- *SolverState
- error

### String

String returns a human-readable representation.

```go
func (*LinearSum) String() string
```

**Parameters:**
  None

**Returns:**
- string

### Type

Type returns the constraint type identifier.

```go
func (*EqualityReified) Type() string
```

**Parameters:**
  None

**Returns:**
- string

### Variables

Variables returns all variables in the BoolSum constraint.

```go
func (*BoolSum) Variables() []*FDVariable
```

**Parameters:**
  None

**Returns:**
- []*FDVariable

### BoundsSum
Constrains: sum(vars) = total Bounds propagation: - total.min >= sum(vars[i].min) - total.max <= sum(vars[i].max) - For each var[i]: var[i].min >= total.min - sum(vars[j!=i].max) - For each var[i]: var[i].max <= total.max - sum(vars[j!=i].min) This is a simplified version sufficient for counting with 0/1 variables. A full Sum constraint would support coefficients and inequalities.

#### Example Usage

```go
// Create a new BoundsSum
boundssum := BoundsSum{
    vars: [],
    total: &FDVariable{}{},
}
```

#### Type Definition

```go
type BoundsSum struct {
    vars []*FDVariable
    total *FDVariable
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| vars | `[]*FDVariable` |  |
| total | `*FDVariable` |  |

### Constructor Functions

### NewBoundsSum

NewBoundsSum creates a bounds-propagating sum constraint.

```go
func NewBoundsSum(vars []*FDVariable, total *FDVariable) (*BoundsSum, error)
```

**Parameters:**

- `vars` ([]*FDVariable) - variables to sum (must not be nil or empty)

- `total` (*FDVariable) - variable representing the sum

**Returns:**
- *BoundsSum
- error

## Methods

### Propagate

Propagate applies bounds propagation for sum constraint. Implements PropagationConstraint.

```go
func (*Inequality) Propagate(solver *Solver, state *SolverState) (*SolverState, error)
```

**Parameters:**
- `solver` (*Solver)
- `state` (*SolverState)

**Returns:**
- *SolverState
- error

### String

String returns a human-readable representation. Implements ModelConstraint.

```go
func (*RationalLinearSum) String() string
```

**Parameters:**
  None

**Returns:**
- string

### Type

Type returns the constraint type identifier. Implements ModelConstraint.

```go
func (*ScaledDivision) Type() string
```

**Parameters:**
  None

**Returns:**
- string

### Variables

Variables returns the variables involved in this constraint. Implements ModelConstraint.

```go
func (*ElementValues) Variables() []*FDVariable
```

**Parameters:**
  None

**Returns:**
- []*FDVariable

### CallPattern
CallPattern represents a normalized subgoal call for use as a tabling key. CallPatterns must be comparable and efficiently hashable. The pattern abstracts away specific variable identities, replacing them with canonical positions (e.g., "path(X0, X1)" instead of "path(_42, _73)"). This allows different calls with the same structure to share cached answers. Thread safety: CallPattern is immutable after creation.

#### Example Usage

```go
// Create a new CallPattern
callpattern := CallPattern{
    predicateID: "example",
    argStructure: "example",
    hashValue: 42,
}
```

#### Type Definition

```go
type CallPattern struct {
    predicateID string
    argStructure string
    hashValue uint64
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| predicateID | `string` | Predicate identifier (name or unique ID) |
| argStructure | `string` | Canonical argument structure with variables abstracted to positions Example: "X0,atom(a),X1" for args [Var(42), Atom("a"), Var(73)] |
| hashValue | `uint64` | Pre-computed hash for O(1) map lookup |

### Constructor Functions

### NewCallPattern

NewCallPattern creates a normalized call pattern from a predicate name and arguments. Variables are abstracted to canonical positions (X0, X1, ...) based on first occurrence. Example: args := []Term{NewVar(42, "x"), NewAtom("a"), NewVar(42, "x")} pattern := NewCallPattern("path", args) // pattern.argStructure == "X0,atom(a),X0"

```go
func NewCallPattern(predicateID string, args []Term) *CallPattern
```

**Parameters:**
- `predicateID` (string)
- `args` ([]Term)

**Returns:**
- *CallPattern

## Methods

### ArgStructure

ArgStructure returns the canonical argument structure.

```go
func (*CallPattern) ArgStructure() string
```

**Parameters:**
  None

**Returns:**
- string

### Equal

Equal checks if two call patterns are structurally equal.

```go
func (*BitSetDomain) Equal(other Domain) bool
```

**Parameters:**
- `other` (Domain)

**Returns:**
- bool

### Hash

Hash returns the pre-computed hash value for efficient map lookup.

```go
func (*CallPattern) Hash() uint64
```

**Parameters:**
  None

**Returns:**
- uint64

### PredicateID

PredicateID returns the predicate identifier.

```go
func (*CallPattern) PredicateID() string
```

**Parameters:**
  None

**Returns:**
- string

### String

String returns a human-readable representation of the call pattern.

```go
func (*Model) String() string
```

**Parameters:**
  None

**Returns:**
- string

### Circuit
Circuit is a composite global constraint that owns auxiliary variables and reified constraints to enforce a single Hamiltonian circuit over successors. The Propagate method itself does no work; all pruning is done by the posted sub-constraints. This mirrors the Count and ElementValues pattern.

#### Example Usage

```go
// Create a new Circuit
circuit := Circuit{
    succ: [],
    startIndex: 42,
    bools: [],
    rowSums: [],
    colSums: [],
    eqReifs: [],
    orderVars: [],
    orderReifs: [],
}
```

#### Type Definition

```go
type Circuit struct {
    succ []*FDVariable
    startIndex int
    bools [][]*FDVariable
    rowSums []PropagationConstraint
    colSums []PropagationConstraint
    eqReifs [][]PropagationConstraint
    orderVars []*FDVariable
    orderReifs []PropagationConstraint
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| succ | `[]*FDVariable` |  |
| startIndex | `int` |  |
| bools | `[][]*FDVariable` | b[i][j] ∈ {1=false,2=true} |
| rowSums | `[]PropagationConstraint` | exactly-one per row |
| colSums | `[]PropagationConstraint` | exactly-one per column |
| eqReifs | `[][]PropagationConstraint` | b[i][j] ↔ succ[i]==j |
| orderVars | `[]*FDVariable` | u[1..n] |
| orderReifs | `[]PropagationConstraint` | Reified(Arithmetic(u[i]+1=u[j]), b[i][j]) for j!=start |

### Constructor Functions

### NewCircuit

NewCircuit constructs a Circuit global constraint and posts all auxiliary variables and constraints into the model. Contract: - model != nil - len(succ) = n >= 2 - startIndex in [1..n]

```go
func NewCircuit(model *Model, succ []*FDVariable, startIndex int) (*Circuit, error)
```

**Parameters:**
- `model` (*Model)
- `succ` ([]*FDVariable)
- `startIndex` (int)

**Returns:**
- *Circuit
- error

## Methods

### Propagate

Propagate is a no-op: all pruning is handled by posted sub-constraints. Implements PropagationConstraint.

```go
func (*RationalLinearSum) Propagate(solver *Solver, state *SolverState) (*SolverState, error)
```

**Parameters:**
- `solver` (*Solver)
- `state` (*SolverState)

**Returns:**
- *SolverState
- error

### String

String returns a human-readable description. Implements ModelConstraint.

```go
func (*Absolute) String() string
```

**Parameters:**
  None

**Returns:**
- string

### Type

Type returns the constraint type identifier. Implements ModelConstraint.

```go
func (*Lexicographic) Type() string
```

**Parameters:**
  None

**Returns:**
- string

### Variables

Variables returns the primary decision variables for this global constraint. Implements ModelConstraint.

```go
func (*InSetReified) Variables() []*FDVariable
```

**Parameters:**
  None

**Returns:**
- []*FDVariable

### Constraint
Constraint represents a logical constraint that can be checked against variable bindings. Constraints are the core abstraction that enables order-independent constraint logic programming. Constraints must be thread-safe as they may be checked concurrently during parallel goal evaluation.

#### Example Usage

```go
// Example implementation of Constraint
type MyConstraint struct {
    // Add your fields here
}

func (m MyConstraint) ID() string {
    // Implement your logic here
    return
}

func (m MyConstraint) IsLocal() bool {
    // Implement your logic here
    return
}

func (m MyConstraint) Variables() []*Var {
    // Implement your logic here
    return
}

func (m MyConstraint) Check(param1 map[int64]Term) ConstraintResult {
    // Implement your logic here
    return
}

func (m MyConstraint) String() string {
    // Implement your logic here
    return
}

func (m MyConstraint) Clone() Constraint {
    // Implement your logic here
    return
}


```

#### Type Definition

```go
type Constraint interface {
    ID() string
    IsLocal() bool
    Variables() []*Var
    Check(bindings map[int64]Term) ConstraintResult
    String() string
    Clone() Constraint
}
```

## Methods

| Method | Description |
| ------ | ----------- |

### ConstraintEvent
ConstraintEvent represents a notification about constraint-related activities. Used for coordinating between local stores and the global constraint bus.

#### Example Usage

```go
// Create a new ConstraintEvent
constraintevent := ConstraintEvent{
    Type: ConstraintEventType{},
    StoreID: "example",
    VarID: 42,
    Term: Term{},
    Constraint: Constraint{},
    Timestamp: 42,
}
```

#### Type Definition

```go
type ConstraintEvent struct {
    Type ConstraintEventType
    StoreID string
    VarID int64
    Term Term
    Constraint Constraint
    Timestamp int64
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| Type | `ConstraintEventType` | Type indicates the kind of event (constraint added, variable bound, etc.) |
| StoreID | `string` | StoreID identifies which local constraint store generated this event |
| VarID | `int64` | VarID is the variable ID involved in the event (for binding events) |
| Term | `Term` | Term is the term being bound to the variable (for binding events) |
| Constraint | `Constraint` | Constraint is the constraint involved in the event (for constraint events) |
| Timestamp | `int64` | Timestamp helps with debugging and event ordering |

### ConstraintEventType
ConstraintEventType categorizes different kinds of constraint events for efficient processing by the global constraint bus.

#### Example Usage

```go
// Example usage of ConstraintEventType
var value ConstraintEventType
// Initialize with appropriate value
```

#### Type Definition

```go
type ConstraintEventType int
```

## Methods

### String

String returns a human-readable representation of the constraint event type.

```go
func (*ElementValues) String() string
```

**Parameters:**
  None

**Returns:**
- string

### ConstraintResult
ConstraintResult represents the outcome of evaluating a constraint. Constraints can be satisfied (no violation), violated (goal should fail), or pending (waiting for more variable bindings).

#### Example Usage

```go
// Example usage of ConstraintResult
var value ConstraintResult
// Initialize with appropriate value
```

#### Type Definition

```go
type ConstraintResult int
```

## Methods

### String

String returns a human-readable representation of the constraint result.

```go
func (*Modulo) String() string
```

**Parameters:**
  None

**Returns:**
- string

### ConstraintStore
ConstraintStore represents a collection of constraints and variable bindings. This interface abstracts over both local and global constraint storage.

#### Example Usage

```go
// Example implementation of ConstraintStore
type MyConstraintStore struct {
    // Add your fields here
}

func (m MyConstraintStore) AddConstraint(param1 Constraint) error {
    // Implement your logic here
    return
}

func (m MyConstraintStore) AddBinding(param1 int64, param2 Term) error {
    // Implement your logic here
    return
}

func (m MyConstraintStore) GetBinding(param1 int64) Term {
    // Implement your logic here
    return
}

func (m MyConstraintStore) GetSubstitution() *Substitution {
    // Implement your logic here
    return
}

func (m MyConstraintStore) GetConstraints() []Constraint {
    // Implement your logic here
    return
}

func (m MyConstraintStore) Clone() ConstraintStore {
    // Implement your logic here
    return
}

func (m MyConstraintStore) String() string {
    // Implement your logic here
    return
}


```

#### Type Definition

```go
type ConstraintStore interface {
    AddConstraint(constraint Constraint) error
    AddBinding(varID int64, term Term) error
    GetBinding(varID int64) Term
    GetSubstitution() *Substitution
    GetConstraints() []Constraint
    Clone() ConstraintStore
    String() string
}
```

## Methods

| Method | Description |
| ------ | ----------- |

### Constructor Functions

### unifyWithConstraints

unifyWithConstraints performs unification using the constraint store system. Returns a new constraint store if unification succeeds, and a boolean indicating success. This replaces the old unify function to work with the order-independent constraint system.

```go
func unifyWithConstraints(term1, term2 Term, store ConstraintStore) (ConstraintStore, bool)
```

**Parameters:**
- `term1` (Term)
- `term2` (Term)
- `store` (ConstraintStore)

**Returns:**
- ConstraintStore
- bool

### ConstraintViolationError
ConstraintViolationError represents an error caused by constraint violations. It provides detailed information about which constraint was violated and why.

#### Example Usage

```go
// Create a new ConstraintViolationError
constraintviolationerror := ConstraintViolationError{
    Constraint: Constraint{},
    Bindings: map[],
    Message: "example",
}
```

#### Type Definition

```go
type ConstraintViolationError struct {
    Constraint Constraint
    Bindings map[int64]Term
    Message string
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| Constraint | `Constraint` |  |
| Bindings | `map[int64]Term` |  |
| Message | `string` |  |

### Constructor Functions

### NewConstraintViolationError

NewConstraintViolationError creates a new constraint violation error.

```go
func NewConstraintViolationError(constraint Constraint, bindings map[int64]Term, message string) *ConstraintViolationError
```

**Parameters:**
- `constraint` (Constraint)
- `bindings` (map[int64]Term)
- `message` (string)

**Returns:**
- *ConstraintViolationError

## Methods

### Error

Error returns a detailed error message about the constraint violation.

```go
func (*ConstraintViolationError) Error() string
```

**Parameters:**
  None

**Returns:**
- string

### Count
- Reified constraints prune variable domains based on boolean values - Sum constraint propagates bounds on countVar - Boolean domains drive further pruning on vars Example: Count([X,Y,Z], 5, N) with X,Y,Z ∈ {1..10}, N ∈ {0..3} - If X=5, Y=5 → N ∈ {2,3} (at least 2 equal 5) - If N=0 → X,Y,Z ≠ 5 - If N=3 → X=Y=Z=5 Complexity: O(n) propagation per variable domain change, where n = len(vars)

#### Example Usage

```go
// Create a new Count
count := Count{
    vars: [],
    targetValue: 42,
    countVar: &FDVariable{}{},
    boolVars: [],
    eqConstraints: [],
    sumConstraint: PropagationConstraint{},
}
```

#### Type Definition

```go
type Count struct {
    vars []*FDVariable
    targetValue int
    countVar *FDVariable
    boolVars []*FDVariable
    eqConstraints []PropagationConstraint
    sumConstraint PropagationConstraint
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| vars | `[]*FDVariable` | Variables to count over |
| targetValue | `int` | Value to count occurrences of |
| countVar | `*FDVariable` | Variable representing the count |
| boolVars | `[]*FDVariable` | Internal structures for propagation |
| eqConstraints | `[]PropagationConstraint` | Equality-reified constraints |
| sumConstraint | `PropagationConstraint` | Sum of (b[i]==2) equals countVar-1 |

### Constructor Functions

### NewCount

NewCount creates a Count constraint.

```go
func NewCount(model *Model, vars []*FDVariable, targetValue int, countVar *FDVariable) (*Count, error)
```

**Parameters:**

- `model` (*Model) - the model to add boolean variables to

- `vars` ([]*FDVariable) - variables to count (must not be nil or empty)

- `targetValue` (int) - the value to count occurrences of

- `countVar` (*FDVariable) - variable to hold the count, encoded as [1..len(vars)+1]

**Returns:**
- *Count
- error

## Methods

### Propagate

Propagate applies the Count constraint's propagation. The Count constraint itself doesn't need to do propagation because the reified constraints and sum constraint handle it. However, we implement Propagate to satisfy the PropagationConstraint interface and potentially add Count-specific optimizations. Implements PropagationConstraint.

```go
func (*BinPacking) Propagate(solver *Solver, state *SolverState) (*SolverState, error)
```

**Parameters:**
- `solver` (*Solver)
- `state` (*SolverState)

**Returns:**
- *SolverState
- error

### String

String returns a human-readable representation. Implements ModelConstraint.

```go
func (*Modulo) String() string
```

**Parameters:**
  None

**Returns:**
- string

### Type

Type returns the constraint type identifier. Implements ModelConstraint.

```go
func (*EqualityReified) Type() string
```

**Parameters:**
  None

**Returns:**
- string

### Variables

Variables returns all variables involved in this constraint. Includes the input variables, count variable, and auxiliary boolean variables. Implements ModelConstraint.

```go
func (*ElementValues) Variables() []*FDVariable
```

**Parameters:**
  None

**Returns:**
- []*FDVariable

### Cumulative
Cumulative models a single renewable resource with fixed capacity consumed by a set of tasks with fixed durations and demands.

#### Example Usage

```go
// Create a new Cumulative
cumulative := Cumulative{
    starts: [],
    durations: [],
    demands: [],
    capacity: 42,
}
```

#### Type Definition

```go
type Cumulative struct {
    starts []*FDVariable
    durations []int
    demands []int
    capacity int
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| starts | `[]*FDVariable` | start-time variables (1-based discrete time) |
| durations | `[]int` | strictly positive |
| demands | `[]int` | non-negative |
| capacity | `int` | strictly positive |

## Methods

### Propagate

Propagate performs time-table filtering using compulsory parts. See the file header for algorithmic notes.

```go
func (*RationalLinearSum) Propagate(solver *Solver, state *SolverState) (*SolverState, error)
```

**Parameters:**
- `solver` (*Solver)
- `state` (*SolverState)

**Returns:**
- *SolverState
- error

### String

String returns a readable description.

```go
func (*BoolSum) String() string
```

**Parameters:**
  None

**Returns:**
- string

### Type

Type returns the constraint identifier.

```go
func (*Lexicographic) Type() string
```

**Parameters:**
  None

**Returns:**
- string

### Variables

Variables returns the variables involved in this constraint.

```go
func (*BinPacking) Variables() []*FDVariable
```

**Parameters:**
  None

**Returns:**
- []*FDVariable

### CustomConstraint
fd_custom.go: custom constraint interfaces for FDStore CustomConstraint represents a user-defined constraint that can propagate

#### Example Usage

```go
// Example implementation of CustomConstraint
type MyCustomConstraint struct {
    // Add your fields here
}

func (m MyCustomConstraint) Variables() []*FDVar {
    // Implement your logic here
    return
}

func (m MyCustomConstraint) Propagate(param1 *FDStore) bool {
    // Implement your logic here
    return
}

func (m MyCustomConstraint) IsSatisfied() bool {
    // Implement your logic here
    return
}


```

#### Type Definition

```go
type CustomConstraint interface {
    Variables() []*FDVar
    Propagate(store *FDStore) (bool, error)
    IsSatisfied() bool
}
```

## Methods

| Method | Description |
| ------ | ----------- |

### Database
Database is an immutable collection of relations and their facts. Operations return new Database instances with copy-on-write semantics.

#### Example Usage

```go
// Create a new Database
database := Database{
    relations: map[],
    mu: /* value */,
}
```

#### Type Definition

```go
type Database struct {
    relations map[string]*relationData
    mu sync.RWMutex
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| relations | `map[string]*relationData` |  |
| mu | `sync.RWMutex` | protects read/write for concurrent queries |

### Constructor Functions

### DB

DB returns a new, empty Database.

```go
func DB() *Database
```

**Parameters:**
  None

**Returns:**
- *Database

### Load

Load inserts facts for multiple relations in sequence and returns the new DB.

```go
func Load(db *Database, specs ...FactsSpec) (*Database, error)
```

**Parameters:**
- `db` (*Database)
- `specs` (...FactsSpec)

**Returns:**
- *Database
- error

### MustLoad

MustLoad is Load but panics on error.

```go
func MustLoad(db *Database, specs ...FactsSpec) *Database
```

**Parameters:**
- `db` (*Database)
- `specs` (...FactsSpec)

**Returns:**
- *Database

### NewDBFromMap

NewDBFromMap loads facts from a map keyed by relation name using the provided relation registry. This is convenient for multi-relation setups where data is produced as JSON-like maps. Example: rels := map[string]*Relation{"employee": emp, "manager": mgr} data := map[string][][]interface{}{ "employee": {{"alice","eng"}, {"bob","eng"}}, "manager":  {{"bob","alice"}}, } db, _ := NewDBFromMap(rels, data)

```go
func NewDBFromMap(relations map[string]*Relation, data map[string][][]interface{}) (*Database, error)
```

**Parameters:**
- `relations` (map[string]*Relation)
- `data` (map[string][][]interface{})

**Returns:**
- *Database
- error

### NewDatabase

NewDatabase creates an empty database.

```go
func NewDatabase() *Database
```

**Parameters:**
  None

**Returns:**
- *Database

## Methods

### Add

Add inserts a single fact converting non-Term arguments to Atoms. It returns the new immutable Database instance.

```go
func (DelaySet) Add(dep uint64)
```

**Parameters:**
- `dep` (uint64)

**Returns:**
  None

### AddFact

AddFact adds a ground fact to the relation, returning a new Database. Facts are deduplicated; adding the same fact twice is idempotent. Example: db = db.AddFact(parent, NewAtom("alice"), NewAtom("bob")) Returns an error if: - The relation is nil - The number of terms doesn't match the relation's arity - Any term is not ground (contains variables)

```go
func (*Database) AddFact(rel *Relation, terms ...Term) (*Database, error)
```

**Parameters:**
- `rel` (*Relation)
- `terms` (...Term)

**Returns:**
- *Database
- error

### AddFacts

AddFacts inserts many facts at once. Each row must have length = arity. Elements are converted to Terms (Term as-is; otherwise wrapped as Atom).

```go
func (*Database) AddFacts(rel *Relation, rows ...[]interface{}) (*Database, error)
```

**Parameters:**
- `rel` (*Relation)
- `rows` (...[]interface{})

**Returns:**
- *Database
- error

### AllFacts

AllFacts returns all non-deleted facts for a relation as a slice of term slices. Returns nil if the relation has no facts.

```go
func (*TabledDatabase) AllFacts(rel *Relation) [][]Term
```

**Parameters:**
- `rel` (*Relation)

**Returns:**
- [][]Term

### FactCount

FactCount returns the number of non-deleted facts in the given relation.

```go
func (*TabledDatabase) FactCount(rel *Relation) int
```

**Parameters:**
- `rel` (*Relation)

**Returns:**
- int

### MustAdd

MustAdd is Add but panics on error. Convenient for compact examples.

```go
func (*Database) MustAdd(rel *Relation, values ...interface{}) *Database
```

**Parameters:**
- `rel` (*Relation)
- `values` (...interface{})

**Returns:**
- *Database

### MustAddFacts

MustAddFacts is AddFacts but panics on error.

```go
func (*Database) MustAddFacts(rel *Relation, rows ...[]interface{}) *Database
```

**Parameters:**
- `rel` (*Relation)
- `rows` (...[]interface{})

**Returns:**
- *Database

### Q

Q queries a relation, accepting native values or Terms. It converts non-Terms to Atoms before delegating to Database.Query.

```go
func (*TabledDatabase) Q(rel *Relation, args ...interface{}) Goal
```

**Parameters:**
- `rel` (*Relation)
- `args` (...interface{})

**Returns:**
- Goal

### Query

Query returns a Goal that unifies the given pattern with all matching facts. The pattern may contain variables, which will be unified with fact values. Query uses index selection heuristics: - If any term is ground and indexed, use that index for O(1) lookup - Otherwise, scan all facts (O(n)) - Repeated variables are checked for consistency Example: // Find all of alice's children goal := db.Query(parent, NewAtom("alice"), Fresh("child")) // Find all parent-child pairs goal := db.Query(parent, Fresh("p"), Fresh("c")) // Find self-loops (repeated variable) goal := db.Query(edge, Fresh("x"), Fresh("x"))

```go
func (*TabledDatabase) Query(rel *Relation, args ...Term) Goal
```

**Parameters:**
- `rel` (*Relation)
- `args` (...Term)

**Returns:**
- Goal

### RemoveFact

RemoveFact removes a fact from the relation, returning a new Database. If the fact doesn't exist, returns the database unchanged. Uses tombstone marking for O(1) removal with stable fact IDs. Indexes remain valid as fact positions don't change.

```go
func (*TabledDatabase) RemoveFact(rel *Relation, terms ...Term) (*TabledDatabase, error)
```

**Parameters:**
- `rel` (*Relation)
- `terms` (...Term)

**Returns:**
- *TabledDatabase
- error

### DelaySet
WFS scaffolding: types and iterators to support conditional answers with delay sets. This file introduces minimal, backwards-compatible structures to carry well-founded semantics (WFS) metadata alongside existing answer bindings. It does not change the storage layout of AnswerTrie; instead, it provides an optional metadata-aware iterator that can be wired to a delay provider. DelaySet represents the set of negatively depended-on subgoals (by key/hash) that must be resolved before an answer can be considered unconditional. Keys are the CallPattern hash values of the depended subgoals.

#### Example Usage

```go
// Example usage of DelaySet
var value DelaySet
// Initialize with appropriate value
```

#### Type Definition

```go
type DelaySet map[uint64]*ast.StructType
```

### Constructor Functions

### NewDelaySet

NewDelaySet creates an empty delay set.

```go
func NewDelaySet() DelaySet
```

**Parameters:**
  None

**Returns:**
- DelaySet

## Methods

### Add

Add inserts a dependency into the set.

```go
func (Rational) Add(other Rational) Rational
```

**Parameters:**
- `other` (Rational)

**Returns:**
- Rational

### Empty

Empty reports whether the set is empty.

```go
func (DelaySet) Empty() bool
```

**Parameters:**
  None

**Returns:**
- bool

### Has

Has checks membership.

```go
func (BitSet) Has(v int) bool
```

**Parameters:**
- `v` (int)

**Returns:**
- bool

### Merge

Merge unions other into ds in-place.

```go
func (DelaySet) Merge(other DelaySet)
```

**Parameters:**
- `other` (DelaySet)

**Returns:**
  None

### Diffn
Diffn composes reified pairwise non-overlap disjunctions for rectangles.

#### Example Usage

```go
// Create a new Diffn
diffn := Diffn{
    x: [],
    w: [],
    reifs: [],
}
```

#### Type Definition

```go
type Diffn struct {
    x []*FDVariable
    y []*FDVariable
    w []int
    h []int
    reifs [][]*ReifiedConstraint
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| x | `[]*FDVariable` |  |
| y | `[]*FDVariable` |  |
| w | `[]int` |  |
| h | `[]int` |  |
| reifs | `[][]*ReifiedConstraint` | per-pair, four reified inequalities |

### Constructor Functions

### NewDiffn

NewDiffn posts a 2D non-overlap constraint over rectangles defined by positions (x[i], y[i]) and fixed sizes (w[i], h[i]). All sizes must be ≥1.

```go
func NewDiffn(model *Model, x, y []*FDVariable, w, h []int) (*Diffn, error)
```

**Parameters:**
- `model` (*Model)
- `x` ([]*FDVariable)
- `y` ([]*FDVariable)
- `w` ([]int)
- `h` ([]int)

**Returns:**
- *Diffn
- error

## Methods

### Propagate

Propagate is a no-op: pruning is performed by the internal reified inequalities and their BoolSum disjunctions.

```go
func (*EqualityReified) Propagate(solver *Solver, state *SolverState) (*SolverState, error)
```

**Parameters:**
- `solver` (*Solver)
- `state` (*SolverState)

**Returns:**
- *SolverState
- error

### String



```go
func (*Regular) String() string
```

**Parameters:**
  None

**Returns:**
- string

### Type



```go
func (*RationalLinearSum) Type() string
```

**Parameters:**
  None

**Returns:**
- string

### Variables



```go
func (*MembershipConstraint) Variables() []*Var
```

**Parameters:**
  None

**Returns:**
- []*Var

### DisequalityConstraint
DisequalityConstraint implements the disequality constraint (≠). It ensures that two terms are not equal, providing order-independent constraint semantics for the Neq operation. The constraint tracks two terms and checks that they never become equal through unification. If both terms are variables, the constraint remains pending until at least one is bound to a concrete value.

#### Example Usage

```go
// Create a new DisequalityConstraint
disequalityconstraint := DisequalityConstraint{
    id: "example",
    term1: Term{},
    isLocal: true,
}
```

#### Type Definition

```go
type DisequalityConstraint struct {
    id string
    term1 Term
    term2 Term
    isLocal bool
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| id | `string` | id uniquely identifies this constraint instance |
| term1 | `Term` | term1 and term2 are the terms that must not be equal |
| term2 | `Term` | term1 and term2 are the terms that must not be equal |
| isLocal | `bool` | isLocal indicates whether this constraint can be checked locally |

### Constructor Functions

### NewDisequalityConstraint

NewDisequalityConstraint creates a new disequality constraint. The constraint is considered local if both terms are in the same constraint store context, enabling fast local checking.

```go
func NewDisequalityConstraint(term1, term2 Term) *DisequalityConstraint
```

**Parameters:**
- `term1` (Term)
- `term2` (Term)

**Returns:**
- *DisequalityConstraint

## Methods

### Check

Check evaluates the disequality constraint against current variable bindings. Returns ConstraintViolated if the terms are equal, ConstraintPending if variables are unbound, or ConstraintSatisfied if terms are provably unequal. Implements the Constraint interface.

```go
func (*LessEqualConstraint) Check(bindings map[int64]Term) ConstraintResult
```

**Parameters:**
- `bindings` (map[int64]Term)

**Returns:**
- ConstraintResult

### Clone

Clone creates a deep copy of the constraint for parallel execution. Implements the Constraint interface.

```go
func (*Absolute) Clone() PropagationConstraint
```

**Parameters:**
  None

**Returns:**
- PropagationConstraint

### ID

ID returns the unique identifier for this constraint instance. Implements the Constraint interface.

```go
func (*FDVariable) ID() int
```

**Parameters:**
  None

**Returns:**
- int

### IsLocal

IsLocal returns true if this constraint can be evaluated purely within a local constraint store. Implements the Constraint interface.

```go
func (*LessEqualConstraint) IsLocal() bool
```

**Parameters:**
  None

**Returns:**
- bool

### String

String returns a human-readable representation of the constraint. Implements the Constraint interface.

```go
func (*Among) String() string
```

**Parameters:**
  None

**Returns:**
- string

### Variables

Variables returns the logic variables that this constraint depends on. Used to determine when the constraint needs to be re-evaluated. Implements the Constraint interface.

```go
func (*ElementValues) Variables() []*FDVariable
```

**Parameters:**
  None

**Returns:**
- []*FDVariable

### DistinctCount
DistinctCount composes internal reified equalities and boolean sums to count distinct values among vars. The distinct count is exposed as a variable DPlus1 with the standard encoding: distinctCount = DPlus1 - 1.

#### Example Usage

```go
// Create a new DistinctCount
distinctcount := DistinctCount{
    vars: [],
    dPlus1: &FDVariable{}{},
    values: [],
    usedBools: [],
    tTotals: [],
    zeroBools: [],
    eqReified: [],
    perValSums: [],
    xorConstraints: [],
    totalSum: PropagationConstraint{},
}
```

#### Type Definition

```go
type DistinctCount struct {
    vars []*FDVariable
    dPlus1 *FDVariable
    values []int
    usedBools []*FDVariable
    tTotals []*FDVariable
    zeroBools []*FDVariable
    eqReified [][]PropagationConstraint
    perValSums []PropagationConstraint
    xorConstraints []PropagationConstraint
    totalSum PropagationConstraint
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| vars | `[]*FDVariable` |  |
| dPlus1 | `*FDVariable` |  |
| values | `[]int` | union of candidate values |
| usedBools | `[]*FDVariable` | used_v booleans per value v |
| tTotals | `[]*FDVariable` | T_v totals (count+1) per value v |
| zeroBools | `[]*FDVariable` | b_zero_v reifying T_v == 1 |
| eqReified | `[][]PropagationConstraint` | b_iv reifications |
| perValSums | `[]PropagationConstraint` | BoolSum(b_iv, T_v) |
| xorConstraints | `[]PropagationConstraint` | XOR(used_v, b_zero_v) via BoolSum |
| totalSum | `PropagationConstraint` | BoolSum(used_v, dPlus1) |

### Constructor Functions

### NewAtLeastNValues

NewAtLeastNValues enforces that the number of distinct values is ≥ N. Provide minPlus1 with domain [N+1..|U|+1]; the BoolSum over used_bools ties the count to minPlus1, so the ≥ is enforced via the lower bound on minPlus1.

```go
func NewAtLeastNValues(model *Model, vars []*FDVariable, minPlus1 *FDVariable) (*DistinctCount, error)
```

**Parameters:**
- `model` (*Model)
- `vars` ([]*FDVariable)
- `minPlus1` (*FDVariable)

**Returns:**
- *DistinctCount
- error

### NewAtMostNValues

NewAtMostNValues enforces that the number of distinct values is ≤ N. The caller should provide limitPlus1 with domain [1..N+1]; the BoolSum over used_bools ties the count to limitPlus1, so the ≤ is enforced via the upper bound on limitPlus1.

```go
func NewAtMostNValues(model *Model, vars []*FDVariable, limitPlus1 *FDVariable) (*DistinctCount, error)
```

**Parameters:**
- `model` (*Model)
- `vars` ([]*FDVariable)
- `limitPlus1` (*FDVariable)

**Returns:**
- *DistinctCount
- error

### NewDistinctCount

NewDistinctCount builds the distinct-count composition and posts the internal constraints to the provided model.

```go
func NewDistinctCount(model *Model, vars []*FDVariable, dPlus1 *FDVariable) (*DistinctCount, error)
```

**Parameters:**

- `model` (*Model) - the model to host auxiliary variables and constraints

- `vars` ([]*FDVariable) - non-empty slice of FD variables

- `dPlus1` (*FDVariable) - FD variable encoding distinctCount+1 in [1..len(U)+1]

**Returns:**
- *DistinctCount
- error

### NewNValue

NewNValue creates an exact NValue: number of distinct values equals N where the encoding is NPlus1 = N + 1.

```go
func NewNValue(model *Model, vars []*FDVariable, nPlus1 *FDVariable) (*DistinctCount, error)
```

**Parameters:**
- `model` (*Model)
- `vars` ([]*FDVariable)
- `nPlus1` (*FDVariable)

**Returns:**
- *DistinctCount
- error

## Methods

### Propagate

Propagate is a no-op: all pruning is performed by internal constraints.

```go
func (*EqualityReified) Propagate(solver *Solver, state *SolverState) (*SolverState, error)
```

**Parameters:**
- `solver` (*Solver)
- `state` (*SolverState)

**Returns:**
- *SolverState
- error

### String



```go
func (*Lexicographic) String() string
```

**Parameters:**
  None

**Returns:**
- string

### Type



```go
func (*Lexicographic) Type() string
```

**Parameters:**
  None

**Returns:**
- string

### Variables

Variables returns all public-facing variables (vars + dPlus1).

```go
func (*Regular) Variables() []*FDVariable
```

**Parameters:**
  None

**Returns:**
- []*FDVariable

### Domain
Domains support efficient operations for: - Membership testing - Value removal (pruning) - Cardinality queries - Set operations (intersection, union, complement) - Iteration over values Thread safety: Domain implementations must be safe for concurrent read access. Write operations (which return new domains) are inherently safe as they don't modify existing domains.

#### Example Usage

```go
// Example implementation of Domain
type MyDomain struct {
    // Add your fields here
}

func (m MyDomain) Count() int {
    // Implement your logic here
    return
}

func (m MyDomain) Has(param1 int) bool {
    // Implement your logic here
    return
}

func (m MyDomain) Remove(param1 int) Domain {
    // Implement your logic here
    return
}

func (m MyDomain) IsSingleton() bool {
    // Implement your logic here
    return
}

func (m MyDomain) SingletonValue() int {
    // Implement your logic here
    return
}

func (m MyDomain) IterateValues(param1 func(value int))  {
    // Implement your logic here
    return
}

func (m MyDomain) ToSlice() []int {
    // Implement your logic here
    return
}

func (m MyDomain) Intersect(param1 Domain) Domain {
    // Implement your logic here
    return
}

func (m MyDomain) Union(param1 Domain) Domain {
    // Implement your logic here
    return
}

func (m MyDomain) Complement() Domain {
    // Implement your logic here
    return
}

func (m MyDomain) Clone() Domain {
    // Implement your logic here
    return
}

func (m MyDomain) Equal(param1 Domain) bool {
    // Implement your logic here
    return
}

func (m MyDomain) MaxValue() int {
    // Implement your logic here
    return
}

func (m MyDomain) RemoveAbove(param1 int) Domain {
    // Implement your logic here
    return
}

func (m MyDomain) RemoveBelow(param1 int) Domain {
    // Implement your logic here
    return
}

func (m MyDomain) RemoveAtOrAbove(param1 int) Domain {
    // Implement your logic here
    return
}

func (m MyDomain) RemoveAtOrBelow(param1 int) Domain {
    // Implement your logic here
    return
}

func (m MyDomain) Min() int {
    // Implement your logic here
    return
}

func (m MyDomain) Max() int {
    // Implement your logic here
    return
}

func (m MyDomain) String() string {
    // Implement your logic here
    return
}


```

#### Type Definition

```go
type Domain interface {
    Count() int
    Has(value int) bool
    Remove(value int) Domain
    IsSingleton() bool
    SingletonValue() int
    IterateValues(f func(value int))
    ToSlice() []int
    Intersect(other Domain) Domain
    Union(other Domain) Domain
    Complement() Domain
    Clone() Domain
    Equal(other Domain) bool
    MaxValue() int
    RemoveAbove(threshold int) Domain
    RemoveBelow(threshold int) Domain
    RemoveAtOrAbove(threshold int) Domain
    RemoveAtOrBelow(threshold int) Domain
    Min() int
    Max() int
    String() string
}
```

## Methods

| Method | Description |
| ------ | ----------- |

### Constructor Functions

### DomainRange

DomainRange returns a domain representing the inclusive range [min..max]. If min <= 1, this is equivalent to NewBitSetDomain(max). For min>1, values outside the range are removed in one bulk operation. Empty ranges return an empty domain.

```go
func DomainRange(min, max int) Domain
```

**Parameters:**
- `min` (int)
- `max` (int)

**Returns:**
- Domain

### DomainValues

DomainValues returns a domain containing only the provided values. Values out of range are ignored. Empty input yields an empty domain.

```go
func DomainValues(vals ...int) Domain
```

**Parameters:**
- `vals` (...int)

**Returns:**
- Domain

### ElementValues
ElementValues is a constraint linking an index variable, a constant table of values, and a result variable such that result = values[index].

#### Example Usage

```go
// Create a new ElementValues
elementvalues := ElementValues{
    index: &FDVariable{}{},
    values: [],
    result: &FDVariable{}{},
}
```

#### Type Definition

```go
type ElementValues struct {
    index *FDVariable
    values []int
    result *FDVariable
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| index | `*FDVariable` |  |
| values | `[]int` |  |
| result | `*FDVariable` |  |

### Constructor Functions

### NewElementValues

NewElementValues constructs a new ElementValues constraint. Contract: - index != nil, result != nil - len(values) > 0

```go
func NewElementValues(index *FDVariable, values []int, result *FDVariable) (*ElementValues, error)
```

**Parameters:**
- `index` (*FDVariable)
- `values` ([]int)
- `result` (*FDVariable)

**Returns:**
- *ElementValues
- error

## Methods

### Propagate

Propagate enforces result = values[index] bidirectionally. Implements PropagationConstraint.

```go
func (*Sequence) Propagate(solver *Solver, state *SolverState) (*SolverState, error)
```

**Parameters:**
- `solver` (*Solver)
- `state` (*SolverState)

**Returns:**
- *SolverState
- error

### String

String returns a human-readable description. Implements ModelConstraint.

```go
func (*LinearSum) String() string
```

**Parameters:**
  None

**Returns:**
- string

### Type

Type returns the constraint identifier. Implements ModelConstraint.

```go
func (*Modulo) Type() string
```

**Parameters:**
  None

**Returns:**
- string

### Variables

Variables returns the involved variables. Implements ModelConstraint.

```go
func (*Among) Variables() []*FDVariable
```

**Parameters:**
  None

**Returns:**
- []*FDVariable

### EqualityReified
4. B becomes 1 → remove intersection from both domains (enforce X ≠ Y) This provides proper reification semantics for equality, handling both "constraint must be true" and "constraint must be false" cases correctly. Implementation achieves arc-consistency through: - When B=2: X.domain ← X.domain ∩ Y.domain (and vice versa) - When B=1: for each value v: if v ∈ X.domain and Y.domain={v}, remove v from X - Singleton detection: if X and Y are singletons, set B accordingly - Disjoint detection: if X.domain ∩ Y.domain = ∅, set B=1

#### Example Usage

```go
// Create a new EqualityReified
equalityreified := EqualityReified{
    x: &FDVariable{}{},
    y: &FDVariable{}{},
    boolVar: &FDVariable{}{},
}
```

#### Type Definition

```go
type EqualityReified struct {
    x *FDVariable
    y *FDVariable
    boolVar *FDVariable
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| x | `*FDVariable` | First variable |
| y | `*FDVariable` | Second variable |
| boolVar | `*FDVariable` | Boolean variable (domain {1,2}) |

### Constructor Functions

### NewEqualityReified

NewEqualityReified creates an equality-reified constraint.

```go
func NewEqualityReified(x, y, boolVar *FDVariable) (*EqualityReified, error)
```

**Parameters:**
- `x` (*FDVariable)

- `y` (*FDVariable) - variables whose equality is being reified

- `boolVar` (*FDVariable) - boolean variable with domain {1,2} (1=false, 2=true)

**Returns:**
- *EqualityReified
- error

## Methods

### Propagate

Propagate applies the equality-reified constraint's propagation. Implements PropagationConstraint.

```go
func (*Table) Propagate(solver *Solver, state *SolverState) (*SolverState, error)
```

**Parameters:**
- `solver` (*Solver)
- `state` (*SolverState)

**Returns:**
- *SolverState
- error

### String

String returns a human-readable representation. Implements ModelConstraint.

```go
func (*AlphaEqConstraint) String() string
```

**Parameters:**
  None

**Returns:**
- string

### Type

Type returns the constraint type identifier. Implements ModelConstraint.

```go
func (*IntervalArithmetic) Type() string
```

**Parameters:**
  None

**Returns:**
- string

### Variables

Variables returns the variables involved in this constraint. Implements ModelConstraint.

```go
func (*Lexicographic) Variables() []*FDVariable
```

**Parameters:**
  None

**Returns:**
- []*FDVariable

### FDChange
Extend FDVar with offset links (placed here to avoid changing many other files) Note: we keep it unexported and simple; propagation logic in FDStore will consult these. We'll attach via a small map in FDStore to avoid changing serialized layout of FDVar across code paths. FDChange represents a single domain change for undo

#### Example Usage

```go
// Create a new FDChange
fdchange := FDChange{
    vid: 42,
    domain: BitSet{},
}
```

#### Type Definition

```go
type FDChange struct {
    vid int
    domain BitSet
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| vid | `int` |  |
| domain | `BitSet` |  |

### FDPlugin
- PropagationConstraints: prune domains based on constraint semantics During propagation, the FDPlugin: 1. Extracts FD domains from the UnifiedStore 2. Builds a temporary SolverState representing those domains 3. Runs FD propagation constraints to fixed point 4. Extracts pruned domains back into a new UnifiedStore This allows the FD solver to participate in hybrid solving without modifying its core architecture.

#### Example Usage

```go
// Create a new FDPlugin
fdplugin := FDPlugin{
    model: &Model{}{},
    solver: &Solver{}{},
}
```

#### Type Definition

```go
type FDPlugin struct {
    model *Model
    solver *Solver
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| model | `*Model` | model holds the FD variables and constraints |
| solver | `*Solver` | solver performs FD constraint propagation |

### Constructor Functions

### NewFDPlugin

NewFDPlugin creates an FD plugin for the given model. The model should contain all FD variables and PropagationConstraints.

```go
func NewFDPlugin(model *Model) *FDPlugin
```

**Parameters:**
- `model` (*Model)

**Returns:**
- *FDPlugin

## Methods

### CanHandle

CanHandle returns true if the constraint is an FD constraint. Implements SolverPlugin.

```go
func (*FDPlugin) CanHandle(constraint interface{}) bool
```

**Parameters:**
- `constraint` (interface{})

**Returns:**
- bool

### GetModel

GetModel returns the FD model used by this plugin. Useful for debugging and testing.

```go
func (*FDPlugin) GetModel() *Model
```

**Parameters:**
  None

**Returns:**
- *Model

### GetSolver

GetSolver returns the FD solver used by this plugin. Useful for debugging and testing.

```go
func (*FDPlugin) GetSolver() *Solver
```

**Parameters:**
  None

**Returns:**
- *Solver

### Name

Name returns the plugin identifier. Implements SolverPlugin.

```go
func (*FDVariable) Name() string
```

**Parameters:**
  None

**Returns:**
- string

### Propagate

Propagate runs FD constraint propagation on the unified store. Implements SolverPlugin.

```go
func (*BoolSum) Propagate(solver *Solver, state *SolverState) (*SolverState, error)
```

**Parameters:**
- `solver` (*Solver)
- `state` (*SolverState)

**Returns:**
- *SolverState
- error

### stateToStore

stateToStore converts a SolverState back into a UnifiedStore. Extracts all domains from the state and updates the store.

```go
func (*FDPlugin) stateToStore(state *SolverState, originalStore *UnifiedStore) (*UnifiedStore, error)
```

**Parameters:**
- `state` (*SolverState)
- `originalStore` (*UnifiedStore)

**Returns:**
- *UnifiedStore
- error

### storeToState

storeToState builds a SolverState from UnifiedStore FD domains. This allows the FD Solver to work with domains from the hybrid store.

```go
func (*FDPlugin) storeToState(store *UnifiedStore) *SolverState
```

**Parameters:**
- `store` (*UnifiedStore)

**Returns:**
- *SolverState

### FDStore
- Offset arithmetic constraints for modeling relationships - Iterative backtracking with dom/deg heuristics - Context-aware cancellation and timeouts Typical usage: store := NewFDStoreWithDomain(maxValue) vars := store.MakeFDVars(n) // Add constraints... solutions, err := store.Solve(ctx, limit)

#### Example Usage

```go
// Create a new FDStore
fdstore := FDStore{
    mu: /* value */,
    vars: [],
    idToVar: map[],
    queue: [],
    trail: [],
    domainSize: 42,
    offsetLinks: map[],
    ineqLinks: map[],
    customConstraints: [],
    config: &SolverConfig{}{},
    monitor: &SolverMonitor{}{},
}
```

#### Type Definition

```go
type FDStore struct {
    mu sync.Mutex
    vars []*FDVar
    idToVar map[int]*FDVar
    queue []int
    trail []FDChange
    domainSize int
    offsetLinks map[int][]offsetLink
    ineqLinks map[int][]ineqLink
    customConstraints []CustomConstraint
    config *SolverConfig
    monitor *SolverMonitor
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| mu | `sync.Mutex` |  |
| vars | `[]*FDVar` |  |
| idToVar | `map[int]*FDVar` |  |
| queue | `[]int` | variable ids to propagate |
| trail | `[]FDChange` | undo trail |
| domainSize | `int` |  |
| offsetLinks | `map[int][]offsetLink` | offsetLinks maps a variable id to offset links used for arithmetic propagation |
| ineqLinks | `map[int][]ineqLink` | ineqLinks maps a variable id to inequality links used for inequality propagation |
| customConstraints | `[]CustomConstraint` | customConstraints holds user-defined constraints |
| config | `*SolverConfig` | config holds solver configuration including heuristics |
| monitor | `*SolverMonitor` | monitor tracks solving statistics (optional) |

### Constructor Functions

### NewFDStore

NewFDStore creates a store with default domain size 9 (1..9)

```go
func NewFDStore() *FDStore
```

**Parameters:**
  None

**Returns:**
- *FDStore

### NewFDStoreWithConfig

NewFDStoreWithConfig creates a store with custom solver configuration

```go
func NewFDStoreWithConfig(n int, config *SolverConfig) *FDStore
```

**Parameters:**
- `n` (int)
- `config` (*SolverConfig)

**Returns:**
- *FDStore

### NewFDStoreWithDomain

NewFDStoreWithDomain creates a store with domain values 1..n

```go
func NewFDStoreWithDomain(n int) *FDStore
```

**Parameters:**
- `n` (int)

**Returns:**
- *FDStore

## Methods

### AddAllDifferent

AddAllDifferent registers pairwise peers and enqueues initial propagation

```go
func (*FDStore) AddAllDifferent(vars []*FDVar)
```

**Parameters:**
- `vars` ([]*FDVar)

**Returns:**
  None

### AddAllDifferentRegin

AddAllDifferentRegin registers an AllDifferent constraint and applies Regin filtering.

```go
func (*FDStore) AddAllDifferentRegin(vars []*FDVar) error
```

**Parameters:**
- `vars` ([]*FDVar)

**Returns:**
- error

### AddCustomConstraint

AddCustomConstraint adds a user-defined custom constraint to the store

```go
func (*FDStore) AddCustomConstraint(constraint CustomConstraint) error
```

**Parameters:**
- `constraint` (CustomConstraint)

**Returns:**
- error

### AddInequalityConstraint

AddInequalityConstraint adds an inequality constraint between two variables. The constraint enforces the relationship specified by the inequality type.

```go
func (*FDStore) AddInequalityConstraint(x, y *FDVar, typ InequalityType) error
```

**Parameters:**
- `x` (*FDVar)
- `y` (*FDVar)
- `typ` (InequalityType)

**Returns:**
- error

### AddOffsetConstraint

AddOffsetConstraint enforces dst = src + offset (integer constant). Domains are 1..domainSize. It installs bidirectional propagation so changes to either variable restrict the other.

```go
func (*FDStore) AddOffsetConstraint(src *FDVar, offset int, dst *FDVar) error
```

**Parameters:**
- `src` (*FDVar)
- `offset` (int)
- `dst` (*FDVar)

**Returns:**
- error

### AddOffsetLink

AddOffsetLink adds an offset constraint: dst = src + offset This establishes a bidirectional relationship where changes to either variable propagate to restrict the other's domain. Useful for modeling arithmetic relationships like diagonals in N-Queens or temporal constraints.

```go
func (*FDStore) AddOffsetLink(src *FDVar, offset int, dst *FDVar) error
```

**Parameters:**
- `src` (*FDVar)
- `offset` (int)
- `dst` (*FDVar)

**Returns:**
- error

### ApplyAllDifferentRegin

ApplyAllDifferentRegin applies the Regin AllDifferent constraint to the variables. This ensures all variables take distinct values, using efficient bipartite matching to prune domains beyond basic pairwise propagation. Essential for permutation problems like Sudoku rows/columns or N-Queens columns.

```go
func (*FDStore) ApplyAllDifferentRegin(vars []*FDVar) error
```

**Parameters:**
- `vars` ([]*FDVar)

**Returns:**
- error

### Assign

assign domain to singleton value v, returns error on contradiction

```go
func (*FDStore) Assign(v *FDVar, value int) error
```

**Parameters:**
- `v` (*FDVar)
- `value` (int)

**Returns:**
- error

### ComplementDomain

ComplementDomain replaces the domain of v with its complement

```go
func (*FDStore) ComplementDomain(v *FDVar) error
```

**Parameters:**
- `v` (*FDVar)

**Returns:**
- error

### GetDomain

GetDomain returns a copy of the variable's current domain

```go
func (*UnifiedStoreAdapter) GetDomain(varID int) Domain
```

**Parameters:**
- `varID` (int)

**Returns:**
- Domain

### GetMonitor

GetMonitor returns the current monitor, or nil if monitoring is disabled

```go
func (*FDStore) GetMonitor() *SolverMonitor
```

**Parameters:**
  None

**Returns:**
- *SolverMonitor

### GetStats

GetStats returns current solving statistics, or nil if monitoring is disabled

```go
func (*FDStore) GetStats() *SolverStats
```

**Parameters:**
  None

**Returns:**
- *SolverStats

### IntersectDomains

IntersectDomains intersects the domain of v with the given BitSet

```go
func (*FDStore) IntersectDomains(v *FDVar, other BitSet) error
```

**Parameters:**
- `v` (*FDVar)
- `other` (BitSet)

**Returns:**
- error

### MakeFDVars

MakeFDVars creates n new FD variables with the store's default domain. The variables are initialized with full domains (1..domainSize). Returns a slice of *FDVar ready for constraint application.

```go
func (*FDStore) MakeFDVars(n int) []*FDVar
```

**Parameters:**
- `n` (int)

**Returns:**
- []*FDVar

### NewVar



```go
func (*FDStore) NewVar() *FDVar
```

**Parameters:**
  None

**Returns:**
- *FDVar

### ReginFilterLocked



```go
func (*FDStore) ReginFilterLocked(vars []*FDVar) error
```

**Parameters:**
- `vars` ([]*FDVar)

**Returns:**
- error

### Remove

Remove removes a value from a variable's domain

```go
func (*BitSetDomain) Remove(value int) Domain
```

**Parameters:**
- `value` (int)

**Returns:**
- Domain

### SetMonitor

SetMonitor enables statistics collection for this store

```go
func (*FDStore) SetMonitor(monitor *SolverMonitor)
```

**Parameters:**
- `monitor` (*SolverMonitor)

**Returns:**
  None

### Solve

Solve using iterative backtracking with MRV heuristic

```go
func Solve(m *Model, maxSolutions int) ([][]int, error)
```

**Parameters:**
- `m` (*Model)
- `maxSolutions` (int)

**Returns:**
- [][]int
- error

### UnionDomains

UnionDomains unions the domain of v with the given BitSet

```go
func (*FDStore) UnionDomains(v *FDVar, other BitSet) error
```

**Parameters:**
- `v` (*FDVar)
- `other` (BitSet)

**Returns:**
- error

### enqueue



```go
func (*FDStore) enqueue(vid int)
```

**Parameters:**
- `vid` (int)

**Returns:**
  None

### propagateCustomConstraintsLocked

propagateCustomConstraintsLocked runs propagation for all custom constraints

```go
func (*FDStore) propagateCustomConstraintsLocked() error
```

**Parameters:**
  None

**Returns:**
- error

### propagateGreaterEqual

propagateGreaterEqual prunes domains for X >= Y constraint

```go
func (*FDStore) propagateGreaterEqual(x, y *FDVar) error
```

**Parameters:**
- `x` (*FDVar)
- `y` (*FDVar)

**Returns:**
- error

### propagateGreaterThan

propagateGreaterThan prunes domains for X > Y constraint

```go
func (*FDStore) propagateGreaterThan(x, y *FDVar) error
```

**Parameters:**
- `x` (*FDVar)
- `y` (*FDVar)

**Returns:**
- error

### propagateInequalityLocked

propagateInequalityLocked performs initial pruning for an inequality constraint

```go
func (*FDStore) propagateInequalityLocked(x, y *FDVar, typ InequalityType) error
```

**Parameters:**
- `x` (*FDVar)
- `y` (*FDVar)
- `typ` (InequalityType)

**Returns:**
- error

### propagateLessEqual

propagateLessEqual prunes domains for X <= Y constraint

```go
func (*FDStore) propagateLessEqual(x, y *FDVar) error
```

**Parameters:**
- `x` (*FDVar)
- `y` (*FDVar)

**Returns:**
- error

### propagateLessThan

propagateLessThan prunes domains for X < Y constraint

```go
func (*FDStore) propagateLessThan(x, y *FDVar) error
```

**Parameters:**
- `x` (*FDVar)
- `y` (*FDVar)

**Returns:**
- error

### propagateLocked

propagateLocked runs a simple AC-3 style propagation loop (requires lock)

```go
func (*FDStore) propagateLocked() error
```

**Parameters:**
  None

**Returns:**
- error

### propagateNotEqual

propagateNotEqual prunes domains for X != Y constraint

```go
func (*FDStore) propagateNotEqual(x, y *FDVar) error
```

**Parameters:**
- `x` (*FDVar)
- `y` (*FDVar)

**Returns:**
- error

### selectNextVariableAdvanced

selectNextVariableAdvanced selects the next variable using the configured heuristic

```go
func (*FDStore) selectNextVariableAdvanced(config *SolverConfig) (int, []int)
```

**Parameters:**
- `config` (*SolverConfig)

**Returns:**
- int
- []int

### selectNextVariableDeg

selectNextVariableDeg selects variable with highest degree (most constraints)

```go
func (*FDStore) selectNextVariableDeg() (int, []int)
```

**Parameters:**
  None

**Returns:**
- int
- []int

### selectNextVariableDom

selectNextVariableDom selects variable with smallest domain

```go
func (*FDStore) selectNextVariableDom() (int, []int)
```

**Parameters:**
  None

**Returns:**
- int
- []int

### selectNextVariableDomDeg

selectNextVariableDomDeg implements the original dom/deg heuristic

```go
func (*FDStore) selectNextVariableDomDeg() (int, []int)
```

**Parameters:**
  None

**Returns:**
- int
- []int

### selectNextVariableLex

selectNextVariableLex selects the first variable by ID

```go
func (*FDStore) selectNextVariableLex() (int, []int)
```

**Parameters:**
  None

**Returns:**
- int
- []int

### selectNextVariableRandom

selectNextVariableRandom selects a random unassigned variable

```go
func (*FDStore) selectNextVariableRandom(seed int64) (int, []int)
```

**Parameters:**
- `seed` (int64)

**Returns:**
- int
- []int

### setDomainLocked



```go
func (*FDStore) setDomainLocked(v *FDVar, newDom BitSet)
```

**Parameters:**
- `v` (*FDVar)
- `newDom` (BitSet)

**Returns:**
  None

### snapshot

snapshot returns current trail size for backtracking

```go
func (*parentSet) snapshot() []*SubgoalEntry
```

**Parameters:**
  None

**Returns:**
- []*SubgoalEntry

### undo

undo to snapshot

```go
func (*FDStore) undo(to int)
```

**Parameters:**
- `to` (int)

**Returns:**
  None

### variableDegree

variableDegree returns the degree (number of constraints) for a variable

```go
func (*FDStore) variableDegree(v *FDVar) int
```

**Parameters:**
- `v` (*FDVar)

**Returns:**
- int

### FDVar
FDVar is a finite-domain variable

#### Example Usage

```go
// Create a new FDVar
fdvar := FDVar{
    ID: 42,
    domain: BitSet{},
    peers: [],
}
```

#### Type Definition

```go
type FDVar struct {
    ID int
    domain BitSet
    peers []*FDVar
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| ID | `int` |  |
| domain | `BitSet` |  |
| peers | `[]*FDVar` |  |

## Methods

### Domain

Domain returns a copy of the variable's current domain (thread-safe)

```go
func (*FDVar) Domain() BitSet
```

**Parameters:**
  None

**Returns:**
- BitSet

### IsSingleton

IsSingleton returns true if the variable's domain contains exactly one value

```go
func (*FDVar) IsSingleton() bool
```

**Parameters:**
  None

**Returns:**
- bool

### SingletonValue

SingletonValue returns the single value if the domain is singleton, panics otherwise

```go
func (*BitSetDomain) SingletonValue() int
```

**Parameters:**
  None

**Returns:**
- int

### FDVariable
FDVariable represents a finite-domain constraint variable. This is the standard variable type for finite-domain CSPs like Sudoku, N-Queens, scheduling, and resource allocation problems. FDVariable stores the initial domain. During solving, the Solver uses the variable's ID to track current domains in SolverState via copy-on-write. This separation enables: - Model immutability (can be shared by parallel workers) - Efficient O(1) state updates (only modified domains are tracked) - Lock-free parallel search (each worker has its own SolverState chain)

#### Example Usage

```go
// Create a new FDVariable
fdvariable := FDVariable{
    id: 42,
    domain: Domain{},
    name: "example",
}
```

#### Type Definition

```go
type FDVariable struct {
    id int
    domain Domain
    name string
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| id | `int` | Unique identifier within the model |
| domain | `Domain` | Current domain of possible values |
| name | `string` | Optional name for debugging |

### Constructor Functions

### NewFDVariable

NewFDVariable creates a new finite-domain variable with the given ID and domain. The variable is initially unbound (domain may contain multiple values).

```go
func NewFDVariable(id int, domain Domain) *FDVariable
```

**Parameters:**
- `id` (int)
- `domain` (Domain)

**Returns:**
- *FDVariable

### NewFDVariableWithName

NewFDVariableWithName creates a named finite-domain variable for easier debugging.

```go
func NewFDVariableWithName(id int, domain Domain, name string) *FDVariable
```

**Parameters:**
- `id` (int)
- `domain` (Domain)
- `name` (string)

**Returns:**
- *FDVariable

## Methods

### Domain

Domain returns the current domain of possible values.

```go
func (*FDVar) Domain() BitSet
```

**Parameters:**
  None

**Returns:**
- BitSet

### ID

ID returns the unique identifier of this variable.

```go
func (*MembershipConstraint) ID() string
```

**Parameters:**
  None

**Returns:**
- string

### IsBound

IsBound returns true if the variable has a single value in its domain.

```go
func (*FDVariable) IsBound() bool
```

**Parameters:**
  None

**Returns:**
- bool

### Name

Name returns the variable's name for debugging.

```go
func (*FDPlugin) Name() string
```

**Parameters:**
  None

**Returns:**
- string

### SetDomain

SetDomain updates the variable's domain during model construction. This method must NOT be called during solving. During solving, domain changes are tracked via SolverState, not by modifying the variable directly.

```go
func (*Solver) SetDomain(state *SolverState, varID int, domain Domain) (*SolverState, bool)
```

**Parameters:**
- `state` (*SolverState)
- `varID` (int)
- `domain` (Domain)

**Returns:**
- *SolverState
- bool

### String

String returns a human-readable representation.

```go
func (Rational) String() string
```

**Parameters:**
  None

**Returns:**
- string

### TryValue

TryValue returns the variable's value if it is bound; otherwise it returns 0 together with a descriptive error. This provides a safe alternative to Value() for callers that prefer not to recover panics.

```go
func (*FDVariable) TryValue() (int, error)
```

**Parameters:**
  None

**Returns:**
- int
- error

### Value

Value returns the bound value if the variable is bound. Panics if the variable is not bound.

```go
func (*FDVariable) Value() int
```

**Parameters:**
  None

**Returns:**
- int

### Fact
Fact represents a single row in a relation. Facts must be ground (contain only atoms, no variables). Facts are immutable after creation.

#### Example Usage

```go
// Create a new Fact
fact := Fact{
    terms: [],
    hash: 42,
}
```

#### Type Definition

```go
type Fact struct {
    terms []Term
    hash uint64
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| terms | `[]Term` |  |
| hash | `uint64` |  |

### Constructor Functions

### newFact

newFact creates a fact from ground terms, computing its hash for deduplication.

```go
func newFact(terms []Term) (*Fact, error)
```

**Parameters:**
- `terms` ([]Term)

**Returns:**
- *Fact
- error

### selectFacts

selectFacts chooses facts to scan based on index availability and pattern. Skips tombstoned (deleted) facts.

```go
func selectFacts(rd *relationData, rel *Relation, pattern []Term) []*Fact
```

**Parameters:**
- `rd` (*relationData)
- `rel` (*Relation)
- `pattern` ([]Term)

**Returns:**
- []*Fact

## Methods

### Equal

Equal returns true if two facts have identical terms.

```go
func (*CallPattern) Equal(other *CallPattern) bool
```

**Parameters:**
- `other` (*CallPattern)

**Returns:**
- bool

### FactsSpec
FactsSpec describes facts for a relation for bulk loading.

#### Example Usage

```go
// Create a new FactsSpec
factsspec := FactsSpec{
    Rel: &Relation{}{},
    Rows: [],
}
```

#### Type Definition

```go
type FactsSpec struct {
    Rel *Relation
    Rows [][]interface{}
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| Rel | `*Relation` |  |
| Rows | `[][]interface{}` |  |

### FreshnessConstraint
FreshnessConstraint enforces that a nominal name does not occur free in a term. The constraint is local and re-evaluates when any variable inside the term binds. Note: LocalConstraintStore validates constraints on AddConstraint; if this freshness is already violated under current bindings, the add will be rejected with an error and the constraint will not be stored.

#### Example Usage

```go
// Create a new FreshnessConstraint
freshnessconstraint := FreshnessConstraint{
    id: "example",
    name: &Atom{}{},
    term: Term{},
}
```

#### Type Definition

```go
type FreshnessConstraint struct {
    id string
    name *Atom
    term Term
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| id | `string` |  |
| name | `*Atom` |  |
| term | `Term` |  |

### Constructor Functions

### NewFreshnessConstraint

NewFreshnessConstraint constructs a freshness constraint a # term.

```go
func NewFreshnessConstraint(name *Atom, term Term) *FreshnessConstraint
```

**Parameters:**
- `name` (*Atom)
- `term` (Term)

**Returns:**
- *FreshnessConstraint

## Methods

### Check

Check evaluates the freshness constraint against current bindings.

```go
func (*AlphaEqConstraint) Check(bindings map[int64]Term) ConstraintResult
```

**Parameters:**
- `bindings` (map[int64]Term)

**Returns:**
- ConstraintResult

### Clone

Clone implements deep copy.

```go
func (*Absolute) Clone() PropagationConstraint
```

**Parameters:**
  None

**Returns:**
- PropagationConstraint

### ID

ID implements Constraint.

```go
func (*LocalConstraintStoreImpl) ID() string
```

**Parameters:**
  None

**Returns:**
- string

### IsLocal

IsLocal implements Constraint (freshness is checked locally).

```go
func (*LessEqualConstraint) IsLocal() bool
```

**Parameters:**
  None

**Returns:**
- bool

### String

String implements Constraint formatting.

```go
func (*HybridRegistry) String() string
```

**Parameters:**
  None

**Returns:**
- string

### Variables

Variables returns variables that can affect the freshness decision (all vars in term).

```go
func (*Model) Variables() []*FDVariable
```

**Parameters:**
  None

**Returns:**
- []*FDVariable

### GlobalCardinality
GlobalCardinality constrains occurrence counts per value across variables.

#### Example Usage

```go
// Create a new GlobalCardinality
globalcardinality := GlobalCardinality{
    vars: [],
    minCount: [],
    maxCount: [],
    maxValue: 42,
}
```

#### Type Definition

```go
type GlobalCardinality struct {
    vars []*FDVariable
    minCount []int
    maxCount []int
    maxValue int
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| vars | `[]*FDVariable` |  |
| minCount | `[]int` | indexed by value (1..M); index 0 unused |
| maxCount | `[]int` | indexed by value (1..M); index 0 unused |
| maxValue | `int` | M |

## Methods

### Propagate

Propagate performs bounds checks and removes saturated values from other domains.

```go
func (*NominalPlugin) Propagate(store *UnifiedStore) (*UnifiedStore, error)
```

**Parameters:**
- `store` (*UnifiedStore)

**Returns:**
- *UnifiedStore
- error

### String

String returns a readable description.

```go
func (*RationalLinearSum) String() string
```

**Parameters:**
  None

**Returns:**
- string

### Type

Type returns the constraint identifier.

```go
func (*BinPacking) Type() string
```

**Parameters:**
  None

**Returns:**
- string

### Variables

Variables returns variables constrained by GCC.

```go
func (*Circuit) Variables() []*FDVariable
```

**Parameters:**
  None

**Returns:**
- []*FDVariable

### GlobalConstraintBus
GlobalConstraintBus coordinates constraint checking across multiple local constraint stores. It handles cross-store constraints and provides a coordination point for complex constraint interactions. The bus is designed to minimize coordination overhead - most constraints should be local and not require global coordination.

#### Example Usage

```go
// Create a new GlobalConstraintBus
globalconstraintbus := GlobalConstraintBus{
    crossStoreConstraints: map[],
    storeRegistry: map[],
    events: /* value */,
    eventCounter: 42,
    mu: /* value */,
    shutdown: true,
    shutdownCh: /* value */,
    refCount: 42,
}
```

#### Type Definition

```go
type GlobalConstraintBus struct {
    crossStoreConstraints map[string]Constraint
    storeRegistry map[string]LocalConstraintStore
    events chan ConstraintEvent
    eventCounter int64
    mu sync.RWMutex
    shutdown bool
    shutdownCh chan *ast.StructType
    refCount int64
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| crossStoreConstraints | `map[string]Constraint` | crossStoreConstraints holds constraints that span multiple stores |
| storeRegistry | `map[string]LocalConstraintStore` | storeRegistry tracks all active local constraint stores |
| events | `chan ConstraintEvent` | events is the channel for constraint events requiring global coordination |
| eventCounter | `int64` | eventCounter provides unique timestamps for events |
| mu | `sync.RWMutex` | mu protects concurrent access to bus state |
| shutdown | `bool` | shutdown indicates if the bus is shutting down |
| shutdownCh | `chan *ast.StructType` | shutdownCh is closed when the bus shuts down |
| refCount | `int64` | refCount tracks active references to this bus for automatic cleanup |

### Constructor Functions

### GetDefaultGlobalBus

GetDefaultGlobalBus returns a shared global constraint bus instance Use this for operations that don't require constraint isolation between goals

```go
func GetDefaultGlobalBus() *GlobalConstraintBus
```

**Parameters:**
  None

**Returns:**
- *GlobalConstraintBus

### GetPooledGlobalBus

GetPooledGlobalBus gets a constraint bus from the pool for operations that need isolation but can reuse cleaned instances

```go
func GetPooledGlobalBus() *GlobalConstraintBus
```

**Parameters:**
  None

**Returns:**
- *GlobalConstraintBus

### NewGlobalConstraintBus

NewGlobalConstraintBus creates a new global constraint bus for coordinating constraint checking across multiple local stores.

```go
func NewGlobalConstraintBus() *GlobalConstraintBus
```

**Parameters:**
  None

**Returns:**
- *GlobalConstraintBus

## Methods

### AddCrossStoreConstraint

AddCrossStoreConstraint registers a constraint that requires global coordination. Such constraints are checked whenever any relevant variable is bound in any store.

```go
func (*GlobalConstraintBus) AddCrossStoreConstraint(constraint Constraint) error
```

**Parameters:**
- `constraint` (Constraint)

**Returns:**
- error

### CoordinateBinding

CoordinateBinding attempts to bind a variable across all relevant stores. This is used when a binding might affect cross-store constraints.

```go
func (*GlobalConstraintBus) CoordinateBinding(varID int64, term Term, originStoreID string) error
```

**Parameters:**
- `varID` (int64)
- `term` (Term)
- `originStoreID` (string)

**Returns:**
- error

### RegisterStore

RegisterStore adds a local constraint store to the global registry. This enables the bus to coordinate constraints across the store.

```go
func (*GlobalConstraintBus) RegisterStore(store LocalConstraintStore) error
```

**Parameters:**
- `store` (LocalConstraintStore)

**Returns:**
- error

### Reset

Reset clears the constraint bus state for reuse in a pool. This method prepares the bus for safe reuse by clearing all state while keeping the goroutine and channels alive.

```go
func (*GlobalConstraintBus) Reset()
```

**Parameters:**
  None

**Returns:**
  None

### Shutdown

Shutdown gracefully shuts down the global constraint bus. Should be called when constraint processing is complete.

```go
func (*ParallelExecutor) Shutdown()
```

**Parameters:**
  None

**Returns:**
  None

### UnregisterStore

UnregisterStore removes a local constraint store from the global registry. Automatically shuts down the bus when no stores remain (reference counting).

```go
func (*GlobalConstraintBus) UnregisterStore(storeID string)
```

**Parameters:**
- `storeID` (string)

**Returns:**
  None

### handleConstraintAdded

handleConstraintAdded processes events when new constraints are added.

```go
func (*GlobalConstraintBus) handleConstraintAdded(event ConstraintEvent)
```

**Parameters:**
- `event` (ConstraintEvent)

**Returns:**
  None

### handleConstraintViolated

handleConstraintViolated processes constraint violation events.

```go
func (*GlobalConstraintBus) handleConstraintViolated(event ConstraintEvent)
```

**Parameters:**
- `event` (ConstraintEvent)

**Returns:**
  None

### handleStoreCloned

handleStoreCloned processes store cloning events for parallel execution.

```go
func (*GlobalConstraintBus) handleStoreCloned(event ConstraintEvent)
```

**Parameters:**
- `event` (ConstraintEvent)

**Returns:**
  None

### handleVariableBound

handleVariableBound processes events when variables are bound.

```go
func (*GlobalConstraintBus) handleVariableBound(event ConstraintEvent)
```

**Parameters:**
- `event` (ConstraintEvent)

**Returns:**
  None

### processEvents

processEvents handles constraint events in a dedicated goroutine. This provides asynchronous processing of cross-store constraint coordination.

```go
func (*GlobalConstraintBus) processEvents()
```

**Parameters:**
  None

**Returns:**
  None

### trySend

trySend attempts a non-blocking send of an event. It is safe to call concurrently with Shutdown/close; if the channel is closed, the panic is recovered and the send is treated as a no-op (returns false).

```go
func (*GlobalConstraintBus) trySend(event ConstraintEvent) ok bool
```

**Parameters:**
- `event` (ConstraintEvent)

**Returns:**
- bool

### wouldBindingViolateConstraint

wouldBindingViolateConstraint checks if a proposed variable binding would violate a cross-store constraint by examining the combined state of all registered stores.

```go
func (*GlobalConstraintBus) wouldBindingViolateConstraint(constraint Constraint, varID int64, term Term) bool
```

**Parameters:**
- `constraint` (Constraint)
- `varID` (int64)
- `term` (Term)

**Returns:**
- bool

### GlobalConstraintBusPool
GlobalConstraintBusPool manages a pool of reusable constraint buses

#### Example Usage

```go
// Create a new GlobalConstraintBusPool
globalconstraintbuspool := GlobalConstraintBusPool{
    pool: /* value */,
}
```

#### Type Definition

```go
type GlobalConstraintBusPool struct {
    pool sync.Pool
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| pool | `sync.Pool` |  |

### Constructor Functions

### NewGlobalConstraintBusPool

NewGlobalConstraintBusPool creates a new pool of constraint buses

```go
func NewGlobalConstraintBusPool() *GlobalConstraintBusPool
```

**Parameters:**
  None

**Returns:**
- *GlobalConstraintBusPool

## Methods

### Get

Get retrieves a constraint bus from the pool

```go
func (*GlobalConstraintBusPool) Get() *GlobalConstraintBus
```

**Parameters:**
  None

**Returns:**
- *GlobalConstraintBus

### Put

Put returns a constraint bus to the pool after cleaning it

```go
func (*GlobalConstraintBusPool) Put(bus *GlobalConstraintBus)
```

**Parameters:**
- `bus` (*GlobalConstraintBus)

**Returns:**
  None

### Goal
Goal represents a constraint or a combination of constraints. Goals are functions that take a constraint store and return a stream of constraint stores representing all possible ways to satisfy the goal. Goals can be composed to build complex relational programs. The constraint store contains both variable bindings and active constraints, enabling order-independent constraint logic programming.

#### Example Usage

```go
// Example usage of Goal
var value Goal
// Initialize with appropriate value
```

#### Type Definition

```go
type Goal func(ctx context.Context, store ConstraintStore) *Stream
```

### Constructor Functions

### Absento

Absento creates a constraint ensuring that a term does not appear anywhere within another term (at any level of structure). Example: x := Fresh("x") goal := Conj(Absento(NewAtom("bad"), x), Eq(x, List(NewAtom("good"))))

```go
func Absento(absent, term Term) Goal
```

**Parameters:**
- `absent` (Term)
- `term` (Term)

**Returns:**
- Goal

### AlphaEqo

AlphaEqo adds an alpha-equivalence constraint between two terms. It succeeds when the terms are structurally equal modulo renaming of bound names.

```go
func AlphaEqo(left, right Term) Goal
```

**Parameters:**
- `left` (Term)
- `right` (Term)

**Returns:**
- Goal

### Appendo

Appendo creates a goal that relates three lists where the third list is the result of appending the first two lists. This is a classic example of a relational operation in miniKanren. Example: x := Fresh("x") goal := Appendo(List(Atom(1), Atom(2)), List(Atom(3)), x) // x will be bound to (1 2 3)

```go
func Appendo(l1, l2, l3 Term) Goal
```

**Parameters:**
- `l1` (Term)
- `l2` (Term)
- `l3` (Term)

**Returns:**
- Goal

### Arityo

Arityo creates a goal that relates a term to its arity. For pairs/lists, the arity is the length of the list. For atoms, the arity is 0. For variables, the goal fails (cannot determine arity of unbound variable). This is useful for meta-programming and validating term structure. Example: pair := NewPair(NewAtom("a"), NewPair(NewAtom("b"), Nil)) result := Run(1, func(arity *Var) Goal { return Arityo(pair, arity) }) // Result: [2]

```go
func Arityo(term, arity Term) Goal
```

**Parameters:**
- `term` (Term)
- `arity` (Term)

**Returns:**
- Goal

### BetaNormalizeo

BetaNormalizeo relates out to the normal form obtained by repeatedly applying leftmost-outermost beta-reduction. If any decision depends on unresolved logic variables, the goal yields no solution until enough information is available.

```go
func BetaNormalizeo(term Term, out Term) Goal
```

**Parameters:**
- `term` (Term)
- `out` (Term)

**Returns:**
- Goal

### BetaReduceo

BetaReduceo relates out to the result of a single leftmost-outermost beta-reduction step performed on term. If term contains no beta-redex, the goal fails (produces no solutions). Reduction is capture-avoiding and uses Substo to substitute the argument into the lambda body.

```go
func BetaReduceo(term Term, out Term) Goal
```

**Parameters:**
- `term` (Term)
- `out` (Term)

**Returns:**
- Goal

### Booleano

Booleano constrains a term to be a boolean value (true/false). This is useful for boolean logic and conditional processing. Example: x := Fresh("x") goal := Conj(Booleano(x), Eq(x, NewAtom(true)))

```go
func Booleano(term Term) Goal
```

**Parameters:**
- `term` (Term)

**Returns:**
- Goal

### Car

Car extracts the first element of a pair/list. Example: goal := Car(List(NewAtom(1), NewAtom(2)), x) // x = 1

```go
func (*Pair) Car() Term
```

**Parameters:**
  None

**Returns:**
- Term

### CaseIntMap

CaseIntMap builds a ready-to-use Goal that maps integer values to string atoms according to the provided mapping. The helper creates a deterministic sequence of pattern clauses (sorted by key) and returns a Goal equivalent to calling Matche(term, clauses...). Example usage: goal := CaseIntMap(valueTerm, map[int]string{0: "zero", 1: "one"}, q) // then run or combine `goal` with other goals

```go
func CaseIntMap(term Term, mapping map[int]string, q *Var) Goal
```

**Parameters:**
- `term` (Term)
- `mapping` (map[int]string)
- `q` (*Var)

**Returns:**
- Goal

### Cdr

Cdr extracts the rest of a pair/list. Example: goal := Cdr(List(NewAtom(1), NewAtom(2)), x) // x = List(NewAtom(2))

```go
func (*Pair) Cdr() Term
```

**Parameters:**
  None

**Returns:**
- Term

### CompoundTermo

CompoundTermo creates a goal that succeeds only if the term is a compound term (a pair). This is useful for validating term structure before attempting to decompose it. Example: result := Run(1, func(q *Var) Goal { pair := NewPair(NewAtom("a"), NewAtom("b")) return Conj( CompoundTermo(pair), Eq(q, NewAtom("is-compound")), ) }) // Result: ["is-compound"]

```go
func CompoundTermo(term Term) Goal
```

**Parameters:**
- `term` (Term)

**Returns:**
- Goal

### Conda

Conda implements committed choice (if-then-else with cut). Takes pairs of condition-goal clauses and commits to the first condition that succeeds. Example: goal := Conda( []Goal{condition1, thenGoal1}, []Goal{condition2, thenGoal2}, []Goal{Success, elseGoal}, // default case )

```go
func Conda(clauses ...[]Goal) Goal
```

**Parameters:**
- `clauses` (...[]Goal)

**Returns:**
- Goal

### Conde

Conde creates a disjunction goal with lazy interleaving evaluation. Unlike Disj which eagerly evaluates all branches in parallel, Conde interleaves results from branches, pulling from each branch on demand. This enables efficient stream consumption when only a few solutions are needed. "conde" represents "count" in Spanish, indicating enumeration of choices. This is the standard miniKanren conde with fair interleaving. Example: x := Fresh("x") goal := Conde(Eq(x, NewAtom(1)), Eq(x, NewAtom(2)))  // x can be 1 or 2

```go
func Conde(goals ...Goal) Goal
```

**Parameters:**
- `goals` (...Goal)

**Returns:**
- Goal

### Condu

Condu implements committed choice with a unique solution requirement. Like Conda but only commits if the condition has exactly one solution. Example: goal := Condu( []Goal{uniqueCondition, thenGoal}, []Goal{Success, elseGoal}, )

```go
func Condu(clauses ...[]Goal) Goal
```

**Parameters:**
- `clauses` (...[]Goal)

**Returns:**
- Goal

### Conj

Conj creates a conjunction goal that requires all goals to succeed. The goals are evaluated sequentially, with each goal operating on the constraint stores produced by the previous goal. Example: x := Fresh("x") y := Fresh("y") goal := Conj(Eq(x, NewAtom(1)), Eq(y, NewAtom(2)))

```go
func Conj(goals ...Goal) Goal
```

**Parameters:**
- `goals` (...Goal)

**Returns:**
- Goal

### Cons

Cons creates a pair/list construction goal. Example: goal := Cons(NewAtom(1), Nil, x) // x = List(NewAtom(1))

```go
func Cons(car, cdr, pair Term) Goal
```

**Parameters:**
- `car` (Term)
- `cdr` (Term)
- `pair` (Term)

**Returns:**
- Goal

### CopyTerm

CopyTerm creates a goal that unifies copy with a structurally identical version of original, but with all variables replaced by fresh variables. This is essential for meta-programming tasks and implementing certain tabling patterns. The copy preserves the structure of the original term: - Atoms are copied as-is (they're immutable) - Variables are replaced with fresh variables - Pairs are recursively copied Example: x := Fresh("x") original := List(x, NewAtom("hello"), x)  // [x, "hello", x] result := Run(1, func(copy *Var) Goal { return CopyTerm(original, copy) }) // Result will be a list with TWO fresh variables (preserving sharing): // [_fresh1, "hello", _fresh1]

```go
func CopyTerm(original, copy Term) Goal
```

**Parameters:**
- `original` (Term)
- `copy` (Term)

**Returns:**
- Goal

### Disj

Disj creates a disjunction goal that succeeds if any of the goals succeed. This represents choice points in the search space. All solutions from all goals are included in the result stream. This implementation evaluates goals eagerly in parallel for maximum throughput. For lazy interleaving evaluation, use Conde instead. Example: x := Fresh("x") goal := Disj(Eq(x, NewAtom(1)), Eq(x, NewAtom(2)))  // x can be 1 or 2

```go
func Disj(goals ...Goal) Goal
```

**Parameters:**
- `goals` (...Goal)

**Returns:**
- Goal

### DisjQ

DisjQ builds a disjunction (logical OR) of multiple relation queries. Each variant is a row of arguments (native values or Terms) with arity matching the relation. It returns Failure if no variants are provided. Example: // parent(gp, p) OR parent(gp, gc) goal := DisjQ(db, parent, []interface{}{gp, p}, []interface{}{gp, gc})

```go
func DisjQ(db *Database, rel *Relation, variants ...[]interface{}) Goal
```

**Parameters:**
- `db` (*Database)
- `rel` (*Relation)
- `variants` (...[]interface{})

**Returns:**
- Goal

### Distincto

Distincto creates a goal that succeeds if all elements in a list are distinct. This is useful for constraint problems where uniqueness is required. Example: // Verify [1,2,3] has all distinct elements Distincto(List(NewAtom(1), NewAtom(2), NewAtom(3))) // Verify [1,2,1] fails (duplicate 1) Distincto(List(NewAtom(1), NewAtom(2), NewAtom(1))) // fails

```go
func Distincto(list Term) Goal
```

**Parameters:**
- `list` (Term)

**Returns:**
- Goal

### Divo

Divo creates a relational division goal: x / y = z (integer division). Works bidirectionally when possible. Modes: - (x, y, ?) → z = x / y - (x, ?, z) → y = x / z - (?, y, z) → x = y * z Example: result := Run(1, func(q *Var) Goal { return Divo(NewAtom(15), NewAtom(3), q)  // 15 / 3 = ? }) // Result: [5]

```go
func Divo(x, y, z Term) Goal
```

**Parameters:**
- `x` (Term)
- `y` (Term)
- `z` (Term)

**Returns:**
- Goal

### Eq

Eq creates a unification goal that constrains two terms to be equal. This is the fundamental operation in miniKanren - it attempts to make two terms identical by binding variables as needed. The new implementation works with constraint stores to provide order-independent constraint semantics. Variable bindings are checked against all active constraints before being accepted. Unification Rules: - Atom == Atom: succeeds if atoms have the same value - Var == Term: binds the variable to the term (subject to constraints) - Pair == Pair: recursively unifies car and cdr - Otherwise: fails Example: x := Fresh("x") goal := Eq(x, NewAtom("hello"))  // Binds x to "hello"

```go
func Eq(term1, term2 Term) Goal
```

**Parameters:**
- `term1` (Term)
- `term2` (Term)

**Returns:**
- Goal

### Expo

Expo creates a relational exponentiation goal: base^exp = result. Supports multiple modes: - (base, exp, ?) → result = base^exp (forward) - (base, exp, result) → verify base^exp = result - (?, exp, result) → solve for base (integer root) - (base, ?, result) → solve for exp (logarithm) Example: result := Run(1, func(q *Var) Goal { return Expo(NewAtom(2), NewAtom(10), q)  // 2^10 = ? }) // Result: [1024]

```go
func Expo(base, exp, result Term) Goal
```

**Parameters:**
- `base` (Term)
- `exp` (Term)
- `result` (Term)

**Returns:**
- Goal

### FDAllDifferentGoal

FDAllDifferentGoal creates a Goal that enforces an all-different constraint over the provided logic variables. domainSize specifies the integer domain (values 1..domainSize). The goal, when executed, will enumerate all assignments that satisfy the AllDifferent constraint and existing bindings in the provided ConstraintStore.

```go
func FDAllDifferentGoal(vars []*Var, domainSize int) Goal
```

**Parameters:**
- `vars` ([]*Var)
- `domainSize` (int)

**Returns:**
- Goal

### FDCustomGoal

FDCustomGoal creates a goal that enforces a custom constraint

```go
func FDCustomGoal(vars []*Var, constraint CustomConstraint) Goal
```

**Parameters:**
- `vars` ([]*Var)
- `constraint` (CustomConstraint)

**Returns:**
- Goal

### FDFilteredQuery

FDFilteredQuery creates a Goal that queries a database and automatically filters results based on an FD variable's domain. This is the recommended way to integrate pldb with FD constraints. The function builds a query over the relation, then filters those results to include only values where filterVar's binding is present in fdVar's domain. This implements the "FD domains filter database queries" pattern.

```go
func FDFilteredQuery(db *Database, rel *Relation, fdVar *FDVariable, filterVar *Var, queryTerms ...Term) Goal
```

**Parameters:**

- `db` (*Database) - The database to query

- `rel` (*Relation) - The relation to query

- `fdVar` (*FDVariable) - The FD variable whose domain constrains the results

- `filterVar` (*Var) - The relational variable to filter (must appear in queryTerms)

- `queryTerms` (...Term) - The complete list of terms for the query (must include filterVar)

**Returns:**
- Goal

### FDInequalityGoal

FDInequalityGoal creates a goal that enforces an inequality constraint between two variables

```go
func FDInequalityGoal(x, y *Var, typ InequalityType) Goal
```

**Parameters:**
- `x` (*Var)
- `y` (*Var)
- `typ` (InequalityType)

**Returns:**
- Goal

### FDQueensGoal

FDQueensGoal models N-Queens using the FD engine idiomatically: - column variables range 1..n - derived diagonal variables are created as offsets of columns - AllDifferent is applied to columns and both diagonal sets

```go
func FDQueensGoal(vars []*Var, n int) Goal
```

**Parameters:**
- `vars` ([]*Var)
- `n` (int)

**Returns:**
- Goal

### Flatteno

Flatteno creates a goal that relates a nested list structure to its flattened form. This operation converts a tree-like structure of nested lists into a flat list. Atoms are preserved as singleton elements in the result. Example: // Flatten [[1,2],[3,[4,5]]] to [1,2,3,4,5] nested := List(List(NewAtom(1), NewAtom(2)), List(NewAtom(3), List(NewAtom(4), NewAtom(5)))) Flatteno(nested, flat)

```go
func Flatteno(nested, flat Term) Goal
```

**Parameters:**
- `nested` (Term)
- `flat` (Term)

**Returns:**
- Goal

### FreeNameso

FreeNameso relates out to a list (proper list using Pair/Nil) of nominal Atoms that occur free in term. The list is sorted lexicographically by the Atom's string value for determinism and ease of testing.

```go
func FreeNameso(term Term, out Term) Goal
```

**Parameters:**
- `term` (Term)
- `out` (Term)

**Returns:**
- Goal

### Fresho

Fresho adds a freshness constraint asserting that name is fresh for term. Intuition: name does not occur free in term; occurrences bound by inner Tie(name, ...) are allowed.

```go
func Fresho(name *Atom, term Term) Goal
```

**Parameters:**
- `name` (*Atom)
- `term` (Term)

**Returns:**
- Goal

### Functoro

Functoro creates a goal that relates a pair to its "functor" (the car). This is useful for working with compound terms in Prolog-like patterns. For a pair (a . b), the functor is a. For atoms and variables, the goal fails. Example: pair := NewPair(NewAtom("foo"), List(NewAtom(1), NewAtom(2))) result := Run(1, func(functor *Var) Goal { return Functoro(pair, functor) }) // Result: ["foo"]

```go
func Functoro(term, functor Term) Goal
```

**Parameters:**
- `term` (Term)
- `functor` (Term)

**Returns:**
- Goal

### GreaterEqualo

GreaterEqualo creates a relational greater-than-or-equal goal: x ≥ y.

```go
func GreaterEqualo(x, y Term) Goal
```

**Parameters:**
- `x` (Term)
- `y` (Term)

**Returns:**
- Goal

### GreaterThano

GreaterThano creates a relational greater-than goal: x > y. Example: result := Run(1, func(q *Var) Goal { return Conj( GreaterThano(NewAtom(10), NewAtom(5)), Eq(q, NewAtom("yes")), ) }) // Result: ["yes"]

```go
func GreaterThano(x, y Term) Goal
```

**Parameters:**
- `x` (Term)
- `y` (Term)

**Returns:**
- Goal

### Ground

Ground creates a goal that succeeds only if the given term is fully ground (contains no unbound variables). This is useful for validation and ensuring that a term is fully instantiated before performing certain operations. A term is considered ground if: - It's an atom (atoms have no variables) - It's a variable that's bound to a ground term - It's a pair where both car and cdr are ground Example: x := Fresh("x") result := Run(1, func(q *Var) Goal { return Conj( Eq(x, NewAtom("hello")), Ground(x),  // Succeeds because x is now bound Eq(q, NewAtom("success")), ) }) // Result: ["success"]

```go
func Ground(term Term) Goal
```

**Parameters:**
- `term` (Term)

**Returns:**
- Goal

### HybridConj

HybridConj creates a Goal that combines multiple FD-filtered queries with conjunction. This is useful when multiple database facts need to be checked against different FD constraints. Each query is executed and filtered independently, then results are combined via conjunction (all queries must succeed for the overall goal to succeed).

```go
func HybridConj(goals ...Goal) Goal
```

**Parameters:**

- `goals` (...Goal) - Variable number of Goals to execute in conjunction

**Returns:**
- Goal

### HybridDisj

HybridDisj creates a Goal that combines multiple FD-filtered queries with disjunction. This is useful when any of several database facts can satisfy the constraint. Each query is executed and filtered independently, then results are combined via disjunction (any query succeeding makes the overall goal succeed).

```go
func HybridDisj(goals ...Goal) Goal
```

**Parameters:**

- `goals` (...Goal) - Variable number of Goals to execute in disjunction

**Returns:**
- Goal

### Lengtho

Lengtho creates a goal that relates a list to its length. The length is represented as a Peano number (nested pairs): 0 = nil, S(n) = (s . n) This operation works bidirectionally: - Given list, computes length - Given length, can verify if a list has that length - Can generate lists of a specific length (with unbound elements) For working with integer lengths, use LengthoInt instead. Example: // Get length of [1,2,3] as Peano number Lengtho(List(NewAtom(1), NewAtom(2), NewAtom(3)), length) // length will be (s . (s . (s . nil))) // Verify a list has length 3 three := NewPair(NewAtom("s"), NewPair(NewAtom("s"), NewPair(NewAtom("s"), Nil))) Lengtho(someList, three)

```go
func Lengtho(list, length Term) Goal
```

**Parameters:**
- `list` (Term)
- `length` (Term)

**Returns:**
- Goal

### LengthoInt

LengthoInt creates a goal that relates a list to its length as an integer. This is a convenience wrapper around Lengtho that works with Go integers instead of Peano numbers. Example: // Get length of [1,2,3] as integer LengthoInt(List(NewAtom(1), NewAtom(2), NewAtom(3)), length) // length will be NewAtom(3) // Verify a list has length 3 LengthoInt(someList, NewAtom(3))

```go
func LengthoInt(list, length Term) Goal
```

**Parameters:**
- `list` (Term)
- `length` (Term)

**Returns:**
- Goal

### LessEqualo

LessEqualo creates a relational less-than-or-equal goal: x ≤ y. This constraint is reified and evaluated when variables become bound. Example: result := Run(1, func(q *Var) Goal { return Conj( LessEqualo(NewAtom(5), NewAtom(5)), Eq(q, NewAtom("yes")), ) }) // Result: ["yes"]

```go
func LessEqualo(x, y Term) Goal
```

**Parameters:**
- `x` (Term)
- `y` (Term)

**Returns:**
- Goal

### LessThano

LessThano creates a relational less-than goal: x < y. This constraint is reified - it's added to the constraint store and evaluated whenever variables become bound, ensuring goal order independence. Example: result := Run(3, func(q *Var) Goal { return Conj( Membero(q, List(NewAtom(1), NewAtom(3), NewAtom(7))), LessThano(q, NewAtom(5)), ) }) // Result: [1, 3]

```go
func LessThano(x, y Term) Goal
```

**Parameters:**
- `x` (Term)
- `y` (Term)

**Returns:**
- Goal

### Logo

Logo creates a relational logarithm goal: log_base(value) = result. Supports multiple modes: - (base, value, ?) → result = log_base(value) (forward) - (base, value, result) → verify log_base(value) = result - (base, ?, result) → solve for value (exponential: value = base^result) - (?, value, result) → solve for base (inverse logarithm) Example: result := Run(1, func(q *Var) Goal { return Logo(NewAtom(2), NewAtom(1024), q)  // log2(1024) = ? }) // Result: [10]

```go
func Logo(base, value, result Term) Goal
```

**Parameters:**
- `base` (Term)
- `value` (Term)
- `result` (Term)

**Returns:**
- Goal

### Matcha

Matcha performs committed choice pattern matching. It tries each clause in order and commits to the first matching pattern. Once a pattern matches, subsequent clauses are not tried, even if the committed clause fails during goal execution. This is more efficient than Matche when you know patterns are mutually exclusive or you want deterministic pattern selection. Semantics: - Try clauses in order - Commit to first matching pattern - Execute that clause's goals - Do NOT try subsequent clauses even if goals fail Example: // Safe head of list (deterministic) Matcha(list, NewClause(Nil, Eq(result, NewAtom("error"))), NewClause(NewPair(Fresh("h"), Fresh("_")), Eq(result, Fresh("h"))), )

```go
func Matcha(term Term, clauses ...PatternClause) Goal
```

**Parameters:**
- `term` (Term)
- `clauses` (...PatternClause)

**Returns:**
- Goal

### Matche

Matche performs exhaustive pattern matching over multiple clauses. It tries to match the input term against each clause's pattern, executing the corresponding goals for ALL matching clauses. This is similar to Conde - multiple clauses can match and produce solutions. Each matching clause generates a separate branch in the search tree. Semantics: - For each clause, unify term with clause.Pattern - If unification succeeds, execute clause.Goals - Combine all successful branches with Disj (disjunction) Example: // Classify list length Matche(list, NewClause(Nil, Eq(result, NewAtom("empty"))), NewClause(NewPair(Fresh("_"), Nil), Eq(result, NewAtom("singleton"))), NewClause(NewPair(Fresh("_"), NewPair(Fresh("_"), Fresh("_"))), Eq(result, NewAtom("multiple"))), )

```go
func Matche(term Term, clauses ...PatternClause) Goal
```

**Parameters:**
- `term` (Term)
- `clauses` (...PatternClause)

**Returns:**
- Goal

### MatcheList

MatcheList is a convenience wrapper for matching lists with specific patterns. It handles common list matching scenarios with cleaner syntax. Clauses are specified as: - Empty list: NewClause(Nil, goals...) - Singleton: NewClause(NewPair(element, Nil), goals...) - Cons: NewClause(NewPair(head, tail), goals...) Example: MatcheList(list, NewClause(Nil, Eq(sum, NewAtom(0))), NewClause(NewPair(x, rest), Conj( SumList(rest, restSum), Eq(sum, Add(x, restSum)), )), )

```go
func MatcheList(list Term, clauses ...PatternClause) Goal
```

**Parameters:**
- `list` (Term)
- `clauses` (...PatternClause)

**Returns:**
- Goal

### Matchu

Matchu performs unique pattern matching. It requires that exactly one clause matches. If zero or multiple clauses match, the goal fails. This is useful for enforcing pattern exclusivity and catching ambiguous cases during development. Semantics: - Try to match each clause's pattern (without executing goals yet) - If zero matches: fail - If multiple matches: fail - If exactly one match: execute that clause's goals Example: // Enforce unique classification Matchu(value, NewClause(NewAtom("small"), LessThan(value, 10)), NewClause(NewAtom("medium"), Conj(GreaterThanEq(value, 10), LessThan(value, 100))), NewClause(NewAtom("large"), GreaterThanEq(value, 100)), )

```go
func Matchu(term Term, clauses ...PatternClause) Goal
```

**Parameters:**
- `term` (Term)
- `clauses` (...PatternClause)

**Returns:**
- Goal

### Membero

Membero creates a goal that relates an element to a list it's a member of. This is the relational membership predicate. Example: x := Fresh("x") list := List(NewAtom(1), NewAtom(2), NewAtom(3)) goal := Membero(x, list) // x can be 1, 2, or 3

```go
func Membero(element, list Term) Goal
```

**Parameters:**
- `element` (Term)
- `list` (Term)

**Returns:**
- Goal

### Minuso

Minuso creates a relational subtraction goal: x - y = z. Works bidirectionally like Pluso. Modes: - (x, y, ?) → z = x - y - (x, ?, z) → y = x - z - (?, y, z) → x = y + z Example: result := Run(1, func(q *Var) Goal { return Minuso(NewAtom(5), NewAtom(3), q)  // 5 - 3 = ? }) // Result: [2]

```go
func Minuso(x, y, z Term) Goal
```

**Parameters:**
- `x` (Term)
- `y` (Term)
- `z` (Term)

**Returns:**
- Goal

### Neq

Neq creates a disequality constraint that ensures two terms are NOT equal. This is a constraint that's checked during unification and can cause goals to fail if the constraint would be violated. Example: x := Fresh("x") goal := Conj(Neq(x, NewAtom("forbidden")), Eq(x, NewAtom("allowed"))) Neq implements the disequality constraint. It ensures that two terms are not equal.

```go
func Neq(t1, t2 Term) Goal
```

**Parameters:**
- `t1` (Term)
- `t2` (Term)

**Returns:**
- Goal

### Noto

Noto creates a goal that succeeds if the given goal fails. This is the negation operator for goals. Note: This uses negation-as-failure, which is not purely relational. The goal must be ground (fully instantiated) for negation to be sound.

```go
func Noto(goal Goal) Goal
```

**Parameters:**
- `goal` (Goal)

**Returns:**
- Goal

### Nullo

Nullo checks if a term is the empty list (nil). Example: goal := Nullo(x) // x must be nil

```go
func Nullo(term Term) Goal
```

**Parameters:**
- `term` (Term)

**Returns:**
- Goal

### Numbero

Numbero constrains a term to be a number. Example: x := Fresh("x") goal := Conj(Numbero(x), Eq(x, NewAtom(42)))

```go
func Numbero(term Term) Goal
```

**Parameters:**
- `term` (Term)

**Returns:**
- Goal

### Onceo

Onceo ensures a goal succeeds at most once (cuts choice points). Example: goal := Onceo(Disj(Eq(x, NewAtom(1)), Eq(x, NewAtom(2)))) // Will only return the first solution

```go
func Onceo(goal Goal) Goal
```

**Parameters:**
- `goal` (Goal)

**Returns:**
- Goal

### Pairo

Pairo checks if a term is a pair (non-empty list). Example: goal := Pairo(x) // x must be a pair

```go
func Pairo(term Term) Goal
```

**Parameters:**
- `term` (Term)

**Returns:**
- Goal

### Permuteo

Permuteo creates a goal that relates a list to one of its permutations. This operation generates all permutations when 'permutation' is a variable, or verifies if 'permutation' is a valid permutation of 'list'. Note: This generates n! permutations for a list of length n. Use with caution for lists longer than ~8-10 elements. Uses lazy evaluation (Conde) for efficient stream consumption. Example: // Generate all permutations of [1,2,3] Permuteo(List(NewAtom(1), NewAtom(2), NewAtom(3)), perm) // Verify [3,1,2] is a permutation of [1,2,3] Permuteo(List(NewAtom(1), NewAtom(2), NewAtom(3)), List(NewAtom(3), NewAtom(1), NewAtom(2)))

```go
func Permuteo(list, permutation Term) Goal
```

**Parameters:**
- `list` (Term)
- `permutation` (Term)

**Returns:**
- Goal

### Pluso

Pluso creates a relational addition goal: x + y = z. This operator works bidirectionally - it can solve for any of the three arguments given the other two. Modes of operation: - (x, y, ?) → z = x + y (forward) - (x, ?, z) → y = z - x (backward) - (?, y, z) → x = z - y (backward) - (?, ?, z) → generate pairs that sum to z Example: x := Fresh("x") result := Run(1, func(q *Var) Goal { return Conj( Pluso(NewAtom(2), NewAtom(3), q),  // 2 + 3 = ? ) }) // Result: [5]

```go
func Pluso(x, y, z Term) Goal
```

**Parameters:**
- `x` (Term)
- `y` (Term)
- `z` (Term)

**Returns:**
- Goal

### Project

Project extracts the values of variables from the current substitution and passes them to a function that creates a new goal. Example: goal := Project([]Term{x, y}, func(values []Term) Goal { // values[0] is the value of x, values[1] is the value of y return someGoalUsing(values) })

```go
func Project(vars []Term, goalFunc func([]Term) Goal) Goal
```

**Parameters:**
- `vars` ([]Term)
- `goalFunc` (func([]Term) Goal)

**Returns:**
- Goal

### RecursiveRule

RecursiveRule defines a recursive pldb query rule with tabling support. This helper simplifies common patterns like transitive closure. The rule combines: - Base case: Direct facts from the database - Recursive case: User-defined recursive logic

```go
func RecursiveRule(db *Database, baseRel *Relation, predicateID string, args []Term, recursiveGoal func() Goal) Goal
```

**Parameters:**

- `db` (*Database) - The pldb database

- `baseRel` (*Relation) - The base relation (e.g., "edge")

- `predicateID` (string) - Unique ID for the recursive predicate (e.g., "path")

- `args` ([]Term) - Query arguments (variables or ground terms)

- `recursiveGoal` (func() Goal) - Function that builds the recursive case

**Returns:**
- Goal

### Rembero

Rembero creates a goal that relates an element to input and output lists, where the output list is the input list with the first occurrence of the element removed. This operation works bidirectionally: - Given element and inputList, computes outputList - Given element and outputList, can generate possible inputLists - Given inputList and outputList, can determine what element was removed Uses lazy evaluation (Conde) for efficient stream consumption. Example: // Remove 2 from [1,2,3]: output is [1,3] Rembero(NewAtom(2), List(NewAtom(1), NewAtom(2), NewAtom(3)), output) // Generate lists that when 2 is removed give [1,3] Rembero(NewAtom(2), input, List(NewAtom(1), NewAtom(3)))

```go
func Rembero(element, inputList, outputList Term) Goal
```

**Parameters:**
- `element` (Term)
- `inputList` (Term)
- `outputList` (Term)

**Returns:**
- Goal

### Reverso

Reverso creates a goal that relates a list to its reverse. This operation works bidirectionally and terminates in all modes by first constraining both lists to have the same length (preventing Appendo from diverging). Implementation follows the StackOverflow solution for core.logic's reverso: https://stackoverflow.com/questions/70159176/non-termination-when-query-variable-is-on-a-specific-position Example: // Reverse [1,2,3] to get [3,2,1] Reverso(List(NewAtom(1), NewAtom(2), NewAtom(3)), reversed) // Verify [1,2,3] and [3,2,1] are reverses Reverso(List(NewAtom(1), NewAtom(2), NewAtom(3)), List(NewAtom(3), NewAtom(2), NewAtom(1))) // Works in backward mode: find list that reverses to [3,2,1] Reverso(list, List(NewAtom(3), NewAtom(2), NewAtom(1)))

```go
func Reverso(list, reversed Term) Goal
```

**Parameters:**
- `list` (Term)
- `reversed` (Term)

**Returns:**
- Goal

### SameLengtho

SameLengtho creates a goal that succeeds if two lists have the same length. This is used to constrain search and prevent divergence in relations like Reverso where Appendo could otherwise generate arbitrarily long lists. This relation is bidirectional: it can verify equality of lengths or constrain one list's length based on another's.

```go
func SameLengtho(xs, ys Term) Goal
```

**Parameters:**
- `xs` (Term)
- `ys` (Term)

**Returns:**
- Goal

### SimpleTermo

SimpleTermo creates a goal that succeeds only if the term is simple (an atom or a fully ground term with no compound structure). Example: result := Run(1, func(q *Var) Goal { return Conj( SimpleTermo(NewAtom(42)), Eq(q, NewAtom("is-simple")), ) }) // Result: ["is-simple"]

```go
func SimpleTermo(term Term) Goal
```

**Parameters:**
- `term` (Term)

**Returns:**
- Goal

### Stringo

Stringo constrains a term to be a string (string atom). This is distinct from Symbolo in that it specifically checks for Go string type, whereas Symbolo accepts any string-like symbol. Example: x := Fresh("x") goal := Conj(Stringo(x), Eq(x, NewAtom("hello")))

```go
func Stringo(term Term) Goal
```

**Parameters:**
- `term` (Term)

**Returns:**
- Goal

### Subseto

Subseto creates a goal that relates two lists where the first is a subset of the second. For subset generation, each element from the superset appears at most once in any subset. For subset verification, checks if all elements in subset appear in superset. Note: When generating subsets, produces 2^n subsets for a list of length n. Example: // Verify [1,3] is a subset of [1,2,3,4] Subseto(List(NewAtom(1), NewAtom(3)), List(NewAtom(1), NewAtom(2), NewAtom(3), NewAtom(4))) // Generate all subsets of [1,2,3] (produces 8 subsets: [], [1], [2], [3], [1,2], [1,3], [2,3], [1,2,3]) Subseto(subset, List(NewAtom(1), NewAtom(2), NewAtom(3)))

```go
func Subseto(subset, superset Term) Goal
```

**Parameters:**
- `subset` (Term)
- `superset` (Term)

**Returns:**
- Goal

### Substo

Substo relates out to the result of capture-avoiding substitution of all free occurrences of the nominal name `name` in `term` with `replacement`. Contract (deterministic core, relational wrapper): - Inputs: term (Term possibly containing Tie binders), name (*Atom), replacement (Term) - Output: out (Term) such that out ≡ term[name := replacement] with capture avoidance - If the decision depends on unresolved logic variables in term/replacement, the goal yields no solutions until they become instantiated enough. Binder cases (λ-calculus intuition with Tie(name, body)): - If the binder equals `name`, occurrences are bound; substitution does not enter the body. - Else, if binder is fresh for `replacement` (no free occurrence inside replacement), substitute in the body under the same binder. - Else, pick a fresh nominal name a' (NomFresh("n")), alpha-rename the binder in the body to a' (avoiding inner shadowing), then substitute under the renamed binder.

```go
func Substo(term Term, name *Atom, replacement Term, out Term) Goal
```

**Parameters:**
- `term` (Term)
- `name` (*Atom)
- `replacement` (Term)
- `out` (Term)

**Returns:**
- Goal

### Symbolo

Symbolo constrains a term to be a symbol (string atom). Example: x := Fresh("x") goal := Conj(Symbolo(x), Eq(x, NewAtom("symbol")))

```go
func Symbolo(term Term) Goal
```

**Parameters:**
- `term` (Term)

**Returns:**
- Goal

### TQ

TQ performs a tabled query using rel.Name() as the predicate identifier. Accepts native values or Terms and converts as needed.

```go
func TQ(db *Database, rel *Relation, args ...interface{}) Goal
```

**Parameters:**
- `db` (*Database)
- `rel` (*Relation)
- `args` (...interface{})

**Returns:**
- Goal

### TabledQuery

TabledQuery wraps a pldb query with SLG tabling for recursive evaluation. This is the primary integration point between pldb and the SLG engine. TabledQuery properly composes with Conj/Disj by: - Walking variables in the incoming ConstraintStore to get current bindings - Using bound values as ground terms in the tabled query - Only caching based on the effective query pattern after instantiation - Unifying remaining unbound variables with tabled results This enables tabled queries to work correctly in joins: Conj( TabledQuery(db, parent, "parent", gp, p),    // p unbound, will be bound by results TabledQuery(db, parent, "parent", p, gc),    // p now bound, used as ground term ) The function: 1. Walks all argument variables to get current bindings from store 2. Constructs a CallPattern from the instantiated arguments 3. Creates a GoalEvaluator from the pldb query 4. Evaluates via the global SLG engine with caching 5. Returns a Goal that unifies results with remaining unbound variables

```go
func TabledQuery(db *Database, rel *Relation, predicateID string, args ...Term) Goal
```

**Parameters:**

- `db` (*Database) - The pldb database to query

- `rel` (*Relation) - The relation to query

- `predicateID` (string) - Unique identifier for tabling (e.g., "edge", "path")

- `args` (...Term) - Query pattern (may contain variables or ground terms)

**Returns:**
- Goal

### Timeso

Timeso creates a relational multiplication goal: x * y = z. Works bidirectionally when possible. Modes: - (x, y, ?) → z = x * y - (x, ?, z) → y = z / x (if z divisible by x) - (?, y, z) → x = z / y (if z divisible by y) Example: result := Run(1, func(q *Var) Goal { return Timeso(NewAtom(4), NewAtom(5), q)  // 4 * 5 = ? }) // Result: [20]

```go
func Timeso(x, y, z Term) Goal
```

**Parameters:**
- `x` (Term)
- `y` (Term)
- `z` (Term)

**Returns:**
- Goal

### TypeChecko

TypeChecko checks that term has type "typ" under environment env. Environment is an association list of (name . type) pairs: ((x . T1) (y . T2) ...) Supported term forms: variables (Atoms), application (Pair(fun,arg) via App), and lambda (Tie). Typing rules (simply-typed λ-calculus): - Var: type from env - App: if fun : A->B and arg : A then (fun arg) : B - Lam: if typ is of the form A->B, then under env[x:=A] body : B This is a checker (expects typ shape for lambdas); with logic variables inside typ, it can infer A/B.

```go
func TypeChecko(term Term, env Term, typ Term) Goal
```

**Parameters:**
- `term` (Term)
- `env` (Term)
- `typ` (Term)

**Returns:**
- Goal

### Vectoro

Vectoro constrains a term to be a slice or array. This is useful for working with Go slices in relational programs. Example: x := Fresh("x") goal := Conj(Vectoro(x), Eq(x, NewAtom([]int{1, 2, 3})))

```go
func Vectoro(term Term) Goal
```

**Parameters:**
- `term` (Term)

**Returns:**
- Goal

### plusoGenerate

plusoGenerate generates pairs (x, y) that sum to z. Used when z is known but x and y are not.

```go
func plusoGenerate(x, y Term, z int) Goal
```

**Parameters:**
- `x` (Term)
- `y` (Term)
- `z` (int)

**Returns:**
- Goal

### reversoCore

reversoCore implements the core reversal logic without length constraints. This is separated to allow Reverso to impose length equality first.

```go
func reversoCore(list, reversed Term) Goal
```

**Parameters:**
- `list` (Term)
- `reversed` (Term)

**Returns:**
- Goal

### unifyFactGoal

unifyFactGoal returns a goal that unifies a fact's terms with a pattern. Handles repeated variables (e.g., edge(X, X) requires same value in both positions).

```go
func unifyFactGoal(fact *Fact, pattern []Term) Goal
```

**Parameters:**
- `fact` (*Fact)
- `pattern` ([]Term)

**Returns:**
- Goal

### GoalEvaluator
GoalEvaluator is a function that evaluates a goal and returns answer bindings. It's called by the SLG engine to produce answers for a tabled subgoal. The evaluator should: - Yield answer bindings via the channel - Close the channel when done - Respect context cancellation - Return any error encountered

#### Example Usage

```go
// Example usage of GoalEvaluator
var value GoalEvaluator
// Initialize with appropriate value
```

#### Type Definition

```go
type GoalEvaluator func(ctx context.Context, answers chan<- map[int64]Term) error
```

### Constructor Functions

### NegateEvaluator

NegateEvaluator returns a GoalEvaluator that succeeds with an empty binding iff the inner tabled goal produces no answers. It enforces stratification by requiring that the current predicate's stratum is strictly greater than the inner predicate's stratum. When this condition is violated, it returns an error. WFS Semantics: - If the inner subgoal is complete and has no answers: emit unconditional success (empty binding). - If the inner subgoal is complete and has answers: fail (emit nothing). - If the inner subgoal is incomplete (still evaluating): emit a conditional answer delayed on the completion of the inner subgoal. Usage pattern: outerEval := NegateEvaluator(engine, currentPredID, innerPattern, innerEval) engine.Evaluate(ctx, NewCallPattern(currentPredID, args), outerEval) Note: For conditional answers to work correctly, this evaluator must be called within an SLG evaluation context where the parent subgoal entry is accessible. The engine automatically provides this context via produceAndConsume.

```go
func NegateEvaluator(engine *SLGEngine, currentPredicateID string, innerPattern *CallPattern, innerEvaluator GoalEvaluator) GoalEvaluator
```

**Parameters:**
- `engine` (*SLGEngine)
- `currentPredicateID` (string)
- `innerPattern` (*CallPattern)
- `innerEvaluator` (GoalEvaluator)

**Returns:**
- GoalEvaluator

### QueryEvaluator

QueryEvaluator converts a pldb query Goal into a GoalEvaluator for SLG tabling. It evaluates the query goal and extracts bindings for the specified variables, yielding them as answer substitutions via the channel.

```go
func QueryEvaluator(query Goal, varIDs ...int64) GoalEvaluator
```

**Parameters:**

- `query` (Goal) - The pldb query goal (from Database.Query)

- `varIDs` (...int64) - Variable IDs to extract from each answer

**Returns:**
- GoalEvaluator

### HybridRegistry
variable spaces, eliminating boilerplate code in hybrid queries. Usage Pattern: 1. Create registry with NewHybridRegistry() 2. Register variable pairs with MapVars(relVar, fdVar) 3. Execute hybrid query producing bindings 4. Apply bindings with AutoBind(result, store) Thread Safety: Registry instances are immutable. All operations return new registry instances, making them safe for concurrent use.

#### Example Usage

```go
// Create a new HybridRegistry
hybridregistry := HybridRegistry{
    relToFD: map[],
    fdToRel: map[],
}
```

#### Type Definition

```go
type HybridRegistry struct {
    relToFD map[int64]int
    fdToRel map[int]int64
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| relToFD | `map[int64]int` | relToFD maps relational variable IDs to FD variable IDs |
| fdToRel | `map[int]int64` | fdToRel maps FD variable IDs to relational variable IDs |

### Constructor Functions

### NewHybridRegistry

NewHybridRegistry creates an empty variable mapping registry. Returns a registry with no mappings. Use MapVars() to register variable relationships. Example: registry := NewHybridRegistry() registry, _ = registry.MapVars(ageRelVar, ageFDVar) registry, _ = registry.MapVars(nameRelVar, nameFDVar)

```go
func NewHybridRegistry() *HybridRegistry
```

**Parameters:**
  None

**Returns:**
- *HybridRegistry

## Methods

### AutoBind

AutoBind automatically transfers bindings from a query result to the UnifiedStore based on registered variable mappings. For each mapped variable: 1. Extract binding from query result 2. Apply binding to corresponding FD variable in store 3. Return updated store with all mapped bindings

```go
func (*HybridRegistry) AutoBind(result ConstraintStore, store *UnifiedStore) (*UnifiedStore, error)
```

**Parameters:**

- `result` (ConstraintStore) - Query result containing relational variable bindings

- `store` (*UnifiedStore) - UnifiedStore to update with FD variable bindings

**Returns:**
- *UnifiedStore
- error

### Clone

Clone creates a copy of the registry with the same mappings. Since registries are immutable, this returns a new instance with independent map storage but identical content. Useful when you need to create independent registry branches for different query contexts. Example: baseRegistry := NewHybridRegistry() baseRegistry, _ = baseRegistry.MapVars(age, ageVar) // Create specialized registries for different queries query1Registry, _ = baseRegistry.Clone().MapVars(name, nameVar) query2Registry, _ = baseRegistry.Clone().MapVars(salary, salaryVar)

```go
func (*Absolute) Clone() PropagationConstraint
```

**Parameters:**
  None

**Returns:**
- PropagationConstraint

### GetFDVariable

GetFDVariable returns the FD variable ID mapped to the given relational variable, or -1 if no mapping exists.

```go
func (*HybridRegistry) GetFDVariable(relVar *Var) int
```

**Parameters:**

- `relVar` (*Var) - The relational variable to look up

**Returns:**
- int

### GetRelVariable

GetRelVariable returns the relational variable ID mapped to the given FD variable, or -1 if no mapping exists.

```go
func (*HybridRegistry) GetRelVariable(fdVar *FDVariable) int64
```

**Parameters:**

- `fdVar` (*FDVariable) - The FD variable to look up

**Returns:**
- int64

### HasMapping

HasMapping returns true if a mapping exists for the given relational variable.

```go
func (*HybridRegistry) HasMapping(relVar *Var) bool
```

**Parameters:**

- `relVar` (*Var) - The relational variable to check

**Returns:**
- bool

### MapVars

MapVars registers a bidirectional mapping between a relational variable and an FD variable.

```go
func (*HybridRegistry) MapVars(relVar *Var, fdVar *FDVariable) (*HybridRegistry, error)
```

**Parameters:**

- `relVar` (*Var) - The relational logic variable (from Fresh())

- `fdVar` (*FDVariable) - The FD constraint variable (from model.NewVariable())

**Returns:**
- *HybridRegistry
- error

### MappingCount

MappingCount returns the number of variable mappings in the registry. Useful for debugging and testing to verify registration succeeded. Example: registry, _ = registry.MapVars(age, ageVar) registry, _ = registry.MapVars(name, nameVar) if registry.MappingCount() != 2 { panic("expected 2 mappings") }

```go
func (*HybridRegistry) MappingCount() int
```

**Parameters:**
  None

**Returns:**
- int

### String

String returns a human-readable representation of the registry. Shows all registered mappings in the format: HybridRegistry{rel_id → fd_id, ...} Useful for debugging and logging.

```go
func (*EqualityReified) String() string
```

**Parameters:**
  None

**Returns:**
- string

### HybridSolver
3. The process repeats until no plugin makes further changes (fixed point) 4. If any plugin detects a conflict, solving backtracks Configuration options control: - Maximum propagation iterations (prevent infinite loops) - Plugin execution order (can affect performance) - Timeout and solution limits Thread safety: HybridSolver is safe for concurrent use. Multiple solvers can work on different search branches simultaneously.

#### Example Usage

```go
// Create a new HybridSolver
hybridsolver := HybridSolver{
    plugins: [],
    config: &HybridSolverConfig{}{},
}
```

#### Type Definition

```go
type HybridSolver struct {
    plugins []SolverPlugin
    config *HybridSolverConfig
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| plugins | `[]SolverPlugin` | plugins holds all registered solver plugins in execution order |
| config | `*HybridSolverConfig` | config holds solver parameters |

### Constructor Functions

### NewHybridSolver

NewHybridSolver creates a hybrid solver with the given plugins. Plugins are executed in the order provided, which can affect performance. Typically, you'll register both a RelationalPlugin and an FDPlugin: solver := NewHybridSolver( NewRelationalPlugin(), NewFDPlugin(model), )

```go
func NewHybridSolver(plugins ...SolverPlugin) *HybridSolver
```

**Parameters:**
- `plugins` (...SolverPlugin)

**Returns:**
- *HybridSolver

### NewHybridSolverWithConfig

NewHybridSolverWithConfig creates a hybrid solver with custom configuration.

```go
func NewHybridSolverWithConfig(config *HybridSolverConfig, plugins ...SolverPlugin) *HybridSolver
```

**Parameters:**
- `config` (*HybridSolverConfig)
- `plugins` (...SolverPlugin)

**Returns:**
- *HybridSolver

## Methods

### CanHandle

CanHandle returns a list of plugins that can handle the given constraint. Used for debugging and understanding constraint routing.

```go
func (*RelationalPlugin) CanHandle(constraint interface{}) bool
```

**Parameters:**
- `constraint` (interface{})

**Returns:**
- bool

### GetPlugins

GetPlugins returns all registered plugins. The returned slice should not be modified.

```go
func (*HybridSolver) GetPlugins() []SolverPlugin
```

**Parameters:**
  None

**Returns:**
- []SolverPlugin

### Propagate

Propagate runs all registered plugins to a fixed point on the given store. Returns a new store with all propagations applied, or an error if a conflict is detected. The propagation algorithm: 1. Run each plugin in sequence on the current store 2. If any plugin returns a new store (changes made), record it 3. After all plugins run, if changes occurred, repeat from step 1 4. Stop when no plugin makes changes (fixed point) or max iterations reached 5. Return error if any plugin detects a conflict This implements the "chaotic iteration" algorithm standard in constraint programming.

```go
func (*EqualityReified) Propagate(solver *Solver, state *SolverState) (*SolverState, error)
```

**Parameters:**
- `solver` (*Solver)
- `state` (*SolverState)

**Returns:**
- *SolverState
- error

### PropagateWithConstraints

PropagateWithConstraints runs propagation after adding new constraints. This is a convenience method that combines constraint addition with propagation.

```go
func (*HybridSolver) PropagateWithConstraints(store *UnifiedStore, constraints ...interface{}) (*UnifiedStore, error)
```

**Parameters:**
- `store` (*UnifiedStore)
- `constraints` (...interface{})

**Returns:**
- *UnifiedStore
- error

### RegisterPlugin

RegisterPlugin adds a plugin to the solver. Plugins are executed in registration order.

```go
func (*HybridSolver) RegisterPlugin(plugin SolverPlugin)
```

**Parameters:**
- `plugin` (SolverPlugin)

**Returns:**
  None

### SetConfig

SetConfig updates the solver configuration.

```go
func (*HybridSolver) SetConfig(config *HybridSolverConfig)
```

**Parameters:**
- `config` (*HybridSolverConfig)

**Returns:**
  None

### String

String returns a human-readable representation of the solver.

```go
func (*BoolSum) String() string
```

**Parameters:**
  None

**Returns:**
- string

### HybridSolverConfig
HybridSolverConfig configures the hybrid solver's behavior.

#### Example Usage

```go
// Create a new HybridSolverConfig
hybridsolverconfig := HybridSolverConfig{
    MaxPropagationIterations: 42,
    EnablePropagation: true,
}
```

#### Type Definition

```go
type HybridSolverConfig struct {
    MaxPropagationIterations int
    EnablePropagation bool
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| MaxPropagationIterations | `int` | MaxPropagationIterations limits how many times the solver will iterate through all plugins before declaring a fixed point. Prevents infinite loops from buggy constraint implementations. |
| EnablePropagation | `bool` | EnablePropagation controls whether constraint propagation runs. Can be disabled for pure backtracking search. |

### Constructor Functions

### DefaultHybridSolverConfig

DefaultHybridSolverConfig returns sensible default configuration.

```go
func DefaultHybridSolverConfig() *HybridSolverConfig
```

**Parameters:**
  None

**Returns:**
- *HybridSolverConfig

### InSetReified
_No documentation available_

#### Example Usage

```go
// Create a new InSetReified
insetreified := InSetReified{
    v: &FDVariable{}{},
    set: [],
    boolVar: &FDVariable{}{},
}
```

#### Type Definition

```go
type InSetReified struct {
    v *FDVariable
    set []int
    boolVar *FDVariable
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| v | `*FDVariable` |  |
| set | `[]int` |  |
| boolVar | `*FDVariable` | domain subset of {1,2} |

### Constructor Functions

### NewInSetReified

NewInSetReified creates a reified membership constraint b ↔ (v ∈ setValues).

```go
func NewInSetReified(v *FDVariable, setValues []int, boolVar *FDVariable) (*InSetReified, error)
```

**Parameters:**
- `v` (*FDVariable)
- `setValues` ([]int)
- `boolVar` (*FDVariable)

**Returns:**
- *InSetReified
- error

## Methods

### Propagate

Propagate enforces b ↔ (v ∈ S) with bidirectional pruning.

```go
func (*RelationalPlugin) Propagate(store *UnifiedStore) (*UnifiedStore, error)
```

**Parameters:**
- `store` (*UnifiedStore)

**Returns:**
- *UnifiedStore
- error

### String



```go
func (*Circuit) String() string
```

**Parameters:**
  None

**Returns:**
- string

### Type



```go
func (*Diffn) Type() string
```

**Parameters:**
  None

**Returns:**
- string

### Variables



```go
func (*Modulo) Variables() []*FDVariable
```

**Parameters:**
  None

**Returns:**
- []*FDVariable

### Inequality
But checking every X value against Y requires O(|X| × |Y|) operations When to use: - Ordering constraints in scheduling, resource allocation - Combined with search (which provides the final consistency check) - When domain sizes are large and efficiency matters When NOT to use: - When you need guaranteed arc-consistency (use AllDifferent or custom constraints) - When domains are tiny (arc-consistency overhead is negligible)

#### Example Usage

```go
// Create a new Inequality
inequality := Inequality{
    x: &FDVariable{}{},
    y: &FDVariable{}{},
    kind: InequalityKind{},
}
```

#### Type Definition

```go
type Inequality struct {
    x *FDVariable
    y *FDVariable
    kind InequalityKind
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| x | `*FDVariable` |  |
| y | `*FDVariable` |  |
| kind | `InequalityKind` |  |

### Constructor Functions

### NewInequality

NewInequality creates X op Y constraint. Returns error if x or y is nil.

```go
func NewInequality(x, y *FDVariable, kind InequalityKind) (*Inequality, error)
```

**Parameters:**
- `x` (*FDVariable)
- `y` (*FDVariable)
- `kind` (InequalityKind)

**Returns:**
- *Inequality
- error

## Methods

### Propagate

Propagate applies bounds propagation. Implements PropagationConstraint.

```go
func (*NominalPlugin) Propagate(store *UnifiedStore) (*UnifiedStore, error)
```

**Parameters:**
- `store` (*UnifiedStore)

**Returns:**
- *UnifiedStore
- error

### String

String returns human-readable representation. Implements ModelConstraint.

```go
func (*MembershipConstraint) String() string
```

**Parameters:**
  None

**Returns:**
- string

### Type

Type returns "Inequality". Implements ModelConstraint.

```go
func (*BinPacking) Type() string
```

**Parameters:**
  None

**Returns:**
- string

### Variables

Variables returns [x, y]. Implements ModelConstraint.

```go
func (*Lexicographic) Variables() []*FDVariable
```

**Parameters:**
  None

**Returns:**
- []*FDVariable

### eqDom

eqDom checks domain equality.

```go
func (*Inequality) eqDom(d1, d2 Domain) bool
```

**Parameters:**
- `d1` (Domain)
- `d2` (Domain)

**Returns:**
- bool

### propGE

propGE propagates X ≥ Y. Bounds propagation: X must be ≥ some Y value, Y must be ≤ some X value - Remove from X: all values < min(Y) - Remove from Y: all values > max(X)

```go
func (*Inequality) propGE(solver *Solver, state *SolverState, xDom, yDom Domain) (*SolverState, error)
```

**Parameters:**
- `solver` (*Solver)
- `state` (*SolverState)
- `xDom` (Domain)
- `yDom` (Domain)

**Returns:**
- *SolverState
- error

### propGT

propGT propagates X > Y. Bounds propagation: X must be > some Y value, Y must be < some X value - Remove from X: all values <= min(Y) - Remove from Y: all values >= max(X)

```go
func (*Inequality) propGT(solver *Solver, state *SolverState, xDom, yDom Domain) (*SolverState, error)
```

**Parameters:**
- `solver` (*Solver)
- `state` (*SolverState)
- `xDom` (Domain)
- `yDom` (Domain)

**Returns:**
- *SolverState
- error

### propLE

propLE propagates X ≤ Y. Bounds propagation: X must be ≤ some Y value, Y must be ≥ some X value - Remove from X: all values > max(Y) - Remove from Y: all values < min(X)

```go
func (*Inequality) propLE(solver *Solver, state *SolverState, xDom, yDom Domain) (*SolverState, error)
```

**Parameters:**
- `solver` (*Solver)
- `state` (*SolverState)
- `xDom` (Domain)
- `yDom` (Domain)

**Returns:**
- *SolverState
- error

### propLT

propLT propagates X < Y. Bounds propagation: X must be < some Y value, Y must be > some X value - Remove from X: all values >= max(Y) - Remove from Y: all values <= min(X)

```go
func (*Inequality) propLT(solver *Solver, state *SolverState, xDom, yDom Domain) (*SolverState, error)
```

**Parameters:**
- `solver` (*Solver)
- `state` (*SolverState)
- `xDom` (Domain)
- `yDom` (Domain)

**Returns:**
- *SolverState
- error

### propNE

propNE propagates X ≠ Y.

```go
func (*Inequality) propNE(solver *Solver, state *SolverState, xDom, yDom Domain) (*SolverState, error)
```

**Parameters:**
- `solver` (*Solver)
- `state` (*SolverState)
- `xDom` (Domain)
- `yDom` (Domain)

**Returns:**
- *SolverState
- error

### InequalityKind
InequalityKind specifies the type of inequality.

#### Example Usage

```go
// Example usage of InequalityKind
var value InequalityKind
// Initialize with appropriate value
```

#### Type Definition

```go
type InequalityKind int
```

## Methods

### String

String returns operator symbol.

```go
func (*Absolute) String() string
```

**Parameters:**
  None

**Returns:**
- string

### InequalityType
fd_ineq.go: arithmetic inequality constraints for FDStore InequalityType represents the type of inequality constraint

#### Example Usage

```go
// Example usage of InequalityType
var value InequalityType
// Initialize with appropriate value
```

#### Type Definition

```go
type InequalityType int
```

### Constructor Functions

### reverseInequalityType



```go
func reverseInequalityType(typ InequalityType) InequalityType
```

**Parameters:**
- `typ` (InequalityType)

**Returns:**
- InequalityType

### IntervalArithmetic
- Operations maintain mathematical interval arithmetic properties Mathematical Properties: - Containment: x ∈ [min, max] → domain(x) ⊆ [min, max] - Intersection: [a,b] ∩ [c,d] = [max(a,c), min(b,d)] - Union: [a,b] ∪ [c,d] = [min(a,c), max(b,d)] (convex hull) - Sum: [a,b] + [c,d] = [a+c, b+d] - Difference: [a,b] - [c,d] = [a-d, b-c] Thread Safety: Immutable after construction. Propagate() is safe for concurrent use.

#### Example Usage

```go
// Create a new IntervalArithmetic
intervalarithmetic := IntervalArithmetic{
    variable: &FDVariable{}{},
    minBound: 42,
    maxBound: 42,
    operation: IntervalOperation{},
    result: &FDVariable{}{},
}
```

#### Type Definition

```go
type IntervalArithmetic struct {
    variable *FDVariable
    minBound int
    maxBound int
    operation IntervalOperation
    result *FDVariable
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| variable | `*FDVariable` | The primary variable being constrained |
| minBound | `int` | Minimum bound of the interval (≥ 1 for BitSetDomain) |
| maxBound | `int` | Maximum bound of the interval |
| operation | `IntervalOperation` | The interval operation to perform |
| result | `*FDVariable` | Optional result variable for binary operations (can be nil for containment) |

### Constructor Functions

### NewIntervalArithmetic

NewIntervalArithmetic creates a new interval arithmetic constraint. For containment operations, only variable, minBound, maxBound are used (result should be nil). For binary operations (intersection, union, sum, difference), both variable and result are used.

```go
func NewIntervalArithmetic(variable *FDVariable, minBound, maxBound int, operation IntervalOperation, result *FDVariable) (*IntervalArithmetic, error)
```

**Parameters:**

- `variable` (*FDVariable) - The FD variable to be constrained or first operand

- `minBound` (int) - Minimum bound of the interval (must be ≥ 1)

- `maxBound` (int) - Maximum bound of the interval (must be ≥ minBound)

- `operation` (IntervalOperation) - The interval operation to perform

- `result` (*FDVariable) - The result variable for binary operations (nil for containment)

**Returns:**
- *IntervalArithmetic
- error

## Methods

### Clone

Clone creates an independent copy of this constraint.

```go
func (*UnifiedStore) Clone() *UnifiedStore
```

**Parameters:**
  None

**Returns:**
- *UnifiedStore

### Propagate

Propagate performs interval arithmetic constraint propagation. Algorithm: 1. For containment: Intersect variable domain with [minBound, maxBound] 2. For binary operations: Compute interval arithmetic and propagate to result 3. Bidirectional propagation for binary operations when possible 4. Apply domain changes and detect failures Returns the updated solver state, or error if the constraint is unsatisfiable.

```go
func (*BoolSum) Propagate(solver *Solver, state *SolverState) (*SolverState, error)
```

**Parameters:**
- `solver` (*Solver)
- `state` (*SolverState)

**Returns:**
- *SolverState
- error

### String

String returns a human-readable representation of the constraint.

```go
func (*Lexicographic) String() string
```

**Parameters:**
  None

**Returns:**
- string

### Type

Type returns the constraint type name.

```go
func (*RationalLinearSum) Type() string
```

**Parameters:**
  None

**Returns:**
- string

### Variables

Variables returns the FD variables involved in this constraint.

```go
func (*RationalLinearSum) Variables() []*FDVariable
```

**Parameters:**
  None

**Returns:**
- []*FDVariable

### intersectDomainWithInterval

intersectDomainWithInterval creates a new domain containing only values within [minVal, maxVal].

```go
func (*IntervalArithmetic) intersectDomainWithInterval(domain Domain, minVal, maxVal int) Domain
```

**Parameters:**
- `domain` (Domain)
- `minVal` (int)
- `maxVal` (int)

**Returns:**
- Domain

### propagateContainment

propagateContainment ensures the variable domain is contained within [minBound, maxBound].

```go
func (*IntervalArithmetic) propagateContainment(solver *Solver, state *SolverState, variableDomain Domain) (*SolverState, error)
```

**Parameters:**
- `solver` (*Solver)
- `state` (*SolverState)
- `variableDomain` (Domain)

**Returns:**
- *SolverState
- error

### propagateDifference

propagateDifference computes interval difference: variable_interval - [minBound, maxBound] = result_interval.

```go
func (*IntervalArithmetic) propagateDifference(solver *Solver, state *SolverState, variableDomain Domain) (*SolverState, error)
```

**Parameters:**
- `solver` (*Solver)
- `state` (*SolverState)
- `variableDomain` (Domain)

**Returns:**
- *SolverState
- error

### propagateIntersection

propagateIntersection computes interval intersection between variable interval and [minBound, maxBound].

```go
func (*IntervalArithmetic) propagateIntersection(solver *Solver, state *SolverState, variableDomain Domain) (*SolverState, error)
```

**Parameters:**
- `solver` (*Solver)
- `state` (*SolverState)
- `variableDomain` (Domain)

**Returns:**
- *SolverState
- error

### propagateSum

propagateSum computes interval sum: variable_interval + [minBound, maxBound] = result_interval.

```go
func (*IntervalArithmetic) propagateSum(solver *Solver, state *SolverState, variableDomain Domain) (*SolverState, error)
```

**Parameters:**
- `solver` (*Solver)
- `state` (*SolverState)
- `variableDomain` (Domain)

**Returns:**
- *SolverState
- error

### propagateUnion

propagateUnion computes interval union (convex hull) between variable interval and [minBound, maxBound].

```go
func (*IntervalArithmetic) propagateUnion(solver *Solver, state *SolverState, variableDomain Domain) (*SolverState, error)
```

**Parameters:**
- `solver` (*Solver)
- `state` (*SolverState)
- `variableDomain` (Domain)

**Returns:**
- *SolverState
- error

### IntervalOperation
IntervalOperation represents the type of interval arithmetic operation to perform.

#### Example Usage

```go
// Example usage of IntervalOperation
var value IntervalOperation
// Initialize with appropriate value
```

#### Type Definition

```go
type IntervalOperation int
```

## Methods

### String

String returns a human-readable representation of the interval operation.

```go
func (*Lexicographic) String() string
```

**Parameters:**
  None

**Returns:**
- string

### LessEqualConstraint
LessEqualConstraint represents a constraint that x <= y.

#### Example Usage

```go
// Create a new LessEqualConstraint
lessequalconstraint := LessEqualConstraint{
    id: "example",
    x: Term{},
    y: Term{},
}
```

#### Type Definition

```go
type LessEqualConstraint struct {
    id string
    x Term
    y Term
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| id | `string` |  |
| x | `Term` |  |
| y | `Term` |  |

## Methods

### Check

Check evaluates the less-equal constraint against current bindings.

```go
func (*AlphaEqConstraint) Check(bindings map[int64]Term) ConstraintResult
```

**Parameters:**
- `bindings` (map[int64]Term)

**Returns:**
- ConstraintResult

### Clone

Clone creates a copy of this constraint.

```go
func (*Substitution) Clone() *Substitution
```

**Parameters:**
  None

**Returns:**
- *Substitution

### ID

ID returns the unique identifier for this constraint.

```go
func (*LocalConstraintStoreImpl) ID() string
```

**Parameters:**
  None

**Returns:**
- string

### IsLocal

IsLocal returns true since this constraint can be evaluated locally.

```go
func (*LessEqualConstraint) IsLocal() bool
```

**Parameters:**
  None

**Returns:**
- bool

### String

String returns a human-readable representation.

```go
func (TruthValue) String() string
```

**Parameters:**
  None

**Returns:**
- string

### Variables

Variables returns the logic variables involved in this constraint.

```go
func (*BinPacking) Variables() []*FDVariable
```

**Parameters:**
  None

**Returns:**
- []*FDVariable

### LessThanConstraint
LessThanConstraint represents a constraint that x < y. It is evaluated whenever variables become bound.

#### Example Usage

```go
// Create a new LessThanConstraint
lessthanconstraint := LessThanConstraint{
    id: "example",
    x: Term{},
    y: Term{},
}
```

#### Type Definition

```go
type LessThanConstraint struct {
    id string
    x Term
    y Term
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| id | `string` |  |
| x | `Term` |  |
| y | `Term` |  |

## Methods

### Check

Check evaluates the less-than constraint against current bindings.

```go
func (*MembershipConstraint) Check(bindings map[int64]Term) ConstraintResult
```

**Parameters:**
- `bindings` (map[int64]Term)

**Returns:**
- ConstraintResult

### Clone

Clone creates a copy of this constraint.

```go
func (*RationalLinearSum) Clone() ModelConstraint
```

**Parameters:**
  None

**Returns:**
- ModelConstraint

### ID

ID returns the unique identifier for this constraint.

```go
func (*FDVariable) ID() int
```

**Parameters:**
  None

**Returns:**
- int

### IsLocal

IsLocal returns true since this constraint can be evaluated locally.

```go
func (*AlphaEqConstraint) IsLocal() bool
```

**Parameters:**
  None

**Returns:**
- bool

### String

String returns a human-readable representation.

```go
func (*ScaledDivision) String() string
```

**Parameters:**
  None

**Returns:**
- string

### Variables

Variables returns the logic variables involved in this constraint.

```go
func (*RationalLinearSum) Variables() []*FDVariable
```

**Parameters:**
  None

**Returns:**
- []*FDVariable

### Lexicographic
Lexicographic orders two equal-length vectors of variables.

#### Example Usage

```go
// Create a new Lexicographic
lexicographic := Lexicographic{
    xs: [],
    ys: [],
    kind: lexKind{},
}
```

#### Type Definition

```go
type Lexicographic struct {
    xs []*FDVariable
    ys []*FDVariable
    kind lexKind
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| xs | `[]*FDVariable` |  |
| ys | `[]*FDVariable` |  |
| kind | `lexKind` |  |

## Methods

### Propagate

Propagate enforces bounds-consistent pruning for lexicographic ordering.

```go
func (*Lexicographic) Propagate(solver *Solver, state *SolverState) (*SolverState, error)
```

**Parameters:**
- `solver` (*Solver)
- `state` (*SolverState)

**Returns:**
- *SolverState
- error

### String

String returns a readable description.

```go
func (*Modulo) String() string
```

**Parameters:**
  None

**Returns:**
- string

### Type

Type names the constraint.

```go
func (*Regular) Type() string
```

**Parameters:**
  None

**Returns:**
- string

### Variables

Variables returns all variables in X followed by Y.

```go
func (*Lexicographic) Variables() []*FDVariable
```

**Parameters:**
  None

**Returns:**
- []*FDVariable

### LinearSum
LinearSum is a bounds-consistent weighted sum constraint: Σ a[i]*x[i] = t

#### Example Usage

```go
// Create a new LinearSum
linearsum := LinearSum{
    vars: [],
    coeffs: [],
    total: &FDVariable{}{},
}
```

#### Type Definition

```go
type LinearSum struct {
    vars []*FDVariable
    coeffs []int
    total *FDVariable
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| vars | `[]*FDVariable` |  |
| coeffs | `[]int` |  |
| total | `*FDVariable` |  |

### Constructor Functions

### NewLinearSum

NewLinearSum constructs a new LinearSum constraint. Contract: - len(vars) > 0, len(vars) == len(coeffs) - coeffs[i] can be positive, negative, or zero - total != nil

```go
func NewLinearSum(vars []*FDVariable, coeffs []int, total *FDVariable) (*LinearSum, error)
```

**Parameters:**
- `vars` ([]*FDVariable)
- `coeffs` ([]int)
- `total` (*FDVariable)

**Returns:**
- *LinearSum
- error

## Methods

### Propagate

Propagate applies bounds-consistent pruning. Implements PropagationConstraint.

```go
func (*Circuit) Propagate(solver *Solver, state *SolverState) (*SolverState, error)
```

**Parameters:**
- `solver` (*Solver)
- `state` (*SolverState)

**Returns:**
- *SolverState
- error

### String

String implements ModelConstraint.

```go
func (*Regular) String() string
```

**Parameters:**
  None

**Returns:**
- string

### Type

Type implements ModelConstraint.

```go
func (*Lexicographic) Type() string
```

**Parameters:**
  None

**Returns:**
- string

### Variables

Variables implements ModelConstraint.

```go
func (*Regular) Variables() []*FDVariable
```

**Parameters:**
  None

**Returns:**
- []*FDVariable

### LocalConstraintStore
LocalConstraintStore interface defines the operations needed by the GlobalConstraintBus to coordinate with local stores.

#### Example Usage

```go
// Example implementation of LocalConstraintStore
type MyLocalConstraintStore struct {
    // Add your fields here
}

func (m MyLocalConstraintStore) ID() string {
    // Implement your logic here
    return
}

func (m MyLocalConstraintStore) getAllBindings() map[int64]Term {
    // Implement your logic here
    return
}


```

#### Type Definition

```go
type LocalConstraintStore interface {
    ID() string
    getAllBindings() map[int64]Term
}
```

## Methods

| Method | Description |
| ------ | ----------- |

### LocalConstraintStoreImpl
LocalConstraintStoreImpl provides a concrete implementation of LocalConstraintStore for managing constraints and variable bindings within a single goal context. The store maintains two separate collections: - Local constraints: Checked quickly without global coordination - Local bindings: Variable-to-term mappings for this context When constraints or bindings are added, the store first checks all local constraints for immediate violations, then coordinates with the global bus if necessary for cross-store constraints.

#### Example Usage

```go
// Create a new LocalConstraintStoreImpl
localconstraintstoreimpl := LocalConstraintStoreImpl{
    id: "example",
    constraints: [],
    bindings: map[],
    globalBus: &GlobalConstraintBus{}{},
    generation: 42,
    mu: /* value */,
}
```

#### Type Definition

```go
type LocalConstraintStoreImpl struct {
    id string
    constraints []Constraint
    bindings map[int64]Term
    globalBus *GlobalConstraintBus
    generation int64
    mu sync.RWMutex
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| id | `string` | id uniquely identifies this store instance |
| constraints | `[]Constraint` | constraints holds all local constraints for this store |
| bindings | `map[int64]Term` | bindings maps variable IDs to their bound terms |
| globalBus | `*GlobalConstraintBus` | globalBus coordinates cross-store constraints (optional) |
| generation | `int64` | generation tracks the number of modifications for efficient cloning |
| mu | `sync.RWMutex` | mu protects concurrent access to store state |

### Constructor Functions

### NewLocalConstraintStore

NewLocalConstraintStore creates a new local constraint store with optional global constraint bus integration. If globalBus is nil, the store operates in local-only mode with no cross-store constraint coordination. This is suitable for simple use cases where all constraints are local.

```go
func NewLocalConstraintStore(globalBus *GlobalConstraintBus) *LocalConstraintStoreImpl
```

**Parameters:**
- `globalBus` (*GlobalConstraintBus)

**Returns:**
- *LocalConstraintStoreImpl

## Methods

### AddBinding

AddBinding attempts to bind a variable to a term, checking all relevant constraints for violations. The binding process follows these steps: 1. Check all local constraints against the proposed binding 2. If any local constraint is violated, reject the binding 3. If the binding affects cross-store constraints, coordinate with global bus 4. If all checks pass, add the binding to the store Returns an error if the binding would violate any constraint.

```go
func (*UnifiedStore) AddBinding(varID int64, term Term) (*UnifiedStore, error)
```

**Parameters:**
- `varID` (int64)
- `term` (Term)

**Returns:**
- *UnifiedStore
- error

### AddConstraint

AddConstraint adds a new constraint to the store and checks it against current bindings for immediate violations. The constraint is first checked locally for immediate violations. If the constraint is not local (requires global coordination), it is also registered with the global constraint bus. Returns an error if the constraint is immediately violated.

```go
func (*UnifiedStore) AddConstraint(constraint interface{}) *UnifiedStore
```

**Parameters:**
- `constraint` (interface{})

**Returns:**
- *UnifiedStore

### Clone

Clone creates a deep copy of the constraint store for parallel execution. The clone shares no mutable state with the original store, making it safe for concurrent use in parallel goal evaluation. Cloning is optimized for performance as it's used frequently in parallel execution contexts. The clone initially shares constraint references with the original but will copy-on-write if modified. Implements the ConstraintStore interface.

```go
func (*Scale) Clone() PropagationConstraint
```

**Parameters:**
  None

**Returns:**
- PropagationConstraint

### Generation

Generation returns the current generation number of the store. The generation increments with each modification, enabling efficient change detection and caching strategies.

```go
func (*LocalConstraintStoreImpl) Generation() int64
```

**Parameters:**
  None

**Returns:**
- int64

### GetBinding

GetBinding retrieves the current binding for a variable. Returns nil if the variable is unbound. Implements the ConstraintStore interface.

```go
func (*LocalConstraintStoreImpl) GetBinding(varID int64) Term
```

**Parameters:**
- `varID` (int64)

**Returns:**
- Term

### GetConstraints

GetConstraints returns a copy of all constraints in the store. Used for debugging and testing purposes.

```go
func (*LocalConstraintStoreImpl) GetConstraints() []Constraint
```

**Parameters:**
  None

**Returns:**
- []Constraint

### GetSubstitution

GetSubstitution returns a substitution representing all current bindings. This bridges between the constraint store system and the existing miniKanren substitution-based APIs. Implements the ConstraintStore interface.

```go
func (*UnifiedStoreAdapter) GetSubstitution() *Substitution
```

**Parameters:**
  None

**Returns:**
- *Substitution

### ID

ID returns the unique identifier for this constraint store. Implements the LocalConstraintStore interface.

```go
func (*LocalConstraintStoreImpl) ID() string
```

**Parameters:**
  None

**Returns:**
- string

### IsEmpty

IsEmpty returns true if the store has no constraints or bindings. Useful for optimization and testing.

```go
func (*LocalConstraintStoreImpl) IsEmpty() bool
```

**Parameters:**
  None

**Returns:**
- bool

### Shutdown

Shutdown cleanly shuts down the store and unregisters it from the global constraint bus. Should be called when the store is no longer needed to prevent memory leaks.

```go
func (*LocalConstraintStoreImpl) Shutdown()
```

**Parameters:**
  None

**Returns:**
  None

### String

String returns a human-readable representation of the constraint store for debugging and error reporting. Implements the ConstraintStore interface.

```go
func (*BinPacking) String() string
```

**Parameters:**
  None

**Returns:**
- string

### getAllBindings

getAllBindings returns a copy of all current bindings. Used by the global constraint bus for cross-store constraint checking. Implements the LocalConstraintStore interface.

```go
func (*UnifiedStore) getAllBindings() map[int64]Term
```

**Parameters:**
  None

**Returns:**
- map[int64]Term

### MaxOfArray
MaxOfArray enforces R = max(vars) with bounds-consistent pruning.

#### Example Usage

```go
// Create a new MaxOfArray
maxofarray := MaxOfArray{
    vars: [],
    r: &FDVariable{}{},
}
```

#### Type Definition

```go
type MaxOfArray struct {
    vars []*FDVariable
    r *FDVariable
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| vars | `[]*FDVariable` |  |
| r | `*FDVariable` |  |

## Methods

### Propagate

Propagate clamps r to feasible [max_i Min(Xi) .. max_i Max(Xi)] and enforces Xi <= r.max.

```go
func (*RationalLinearSum) Propagate(solver *Solver, state *SolverState) (*SolverState, error)
```

**Parameters:**
- `solver` (*Solver)
- `state` (*SolverState)

**Returns:**
- *SolverState
- error

### String



```go
func (ConstraintEventType) String() string
```

**Parameters:**
  None

**Returns:**
- string

### Type



```go
func (*Regular) Type() string
```

**Parameters:**
  None

**Returns:**
- string

### Variables



```go
func (*LessEqualConstraint) Variables() []*Var
```

**Parameters:**
  None

**Returns:**
- []*Var

### MembershipConstraint
MembershipConstraint implements the membership constraint (membero). It ensures that an element is a member of a list, providing relational list membership checking that can work in both directions.

#### Example Usage

```go
// Create a new MembershipConstraint
membershipconstraint := MembershipConstraint{
    id: "example",
    element: Term{},
    list: Term{},
    isLocal: true,
}
```

#### Type Definition

```go
type MembershipConstraint struct {
    id string
    element Term
    list Term
    isLocal bool
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| id | `string` | id uniquely identifies this constraint instance |
| element | `Term` | element is the term that should be a member of the list |
| list | `Term` | list is the list that should contain the element |
| isLocal | `bool` | isLocal indicates whether this constraint can be checked locally |

### Constructor Functions

### NewMembershipConstraint

NewMembershipConstraint creates a new membership constraint.

```go
func NewMembershipConstraint(element, list Term) *MembershipConstraint
```

**Parameters:**
- `element` (Term)
- `list` (Term)

**Returns:**
- *MembershipConstraint

## Methods

### Check

Check evaluates the membership constraint against current bindings. Note: This is a simplified implementation. The full membero relation is typically implemented as a recursive goal rather than a simple constraint. Implements the Constraint interface.

```go
func (*LessEqualConstraint) Check(bindings map[int64]Term) ConstraintResult
```

**Parameters:**
- `bindings` (map[int64]Term)

**Returns:**
- ConstraintResult

### Clone

Clone creates a deep copy of the constraint for parallel execution. Implements the Constraint interface.

```go
func (*Substitution) Clone() *Substitution
```

**Parameters:**
  None

**Returns:**
- *Substitution

### ID

ID returns the unique identifier for this constraint instance. Implements the Constraint interface.

```go
func (*LessEqualConstraint) ID() string
```

**Parameters:**
  None

**Returns:**
- string

### IsLocal

IsLocal returns true if this constraint can be evaluated locally. Implements the Constraint interface.

```go
func (*AlphaEqConstraint) IsLocal() bool
```

**Parameters:**
  None

**Returns:**
- bool

### String

String returns a human-readable representation of the constraint. Implements the Constraint interface.

```go
func (*EqualityReified) String() string
```

**Parameters:**
  None

**Returns:**
- string

### Variables

Variables returns the logic variables this constraint depends on. Implements the Constraint interface.

```go
func (*Regular) Variables() []*FDVariable
```

**Parameters:**
  None

**Returns:**
- []*FDVariable

### MinOfArray
MinOfArray enforces R = min(vars) with bounds-consistent pruning.

#### Example Usage

```go
// Create a new MinOfArray
minofarray := MinOfArray{
    vars: [],
    r: &FDVariable{}{},
}
```

#### Type Definition

```go
type MinOfArray struct {
    vars []*FDVariable
    r *FDVariable
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| vars | `[]*FDVariable` |  |
| r | `*FDVariable` |  |

## Methods

### Propagate

Propagate clamps r to feasible [min_i Min(Xi) .. min_i Max(Xi)] and enforces Xi >= r.min.

```go
func (*Among) Propagate(solver *Solver, state *SolverState) (*SolverState, error)
```

**Parameters:**
- `solver` (*Solver)
- `state` (*SolverState)

**Returns:**
- *SolverState
- error

### String



```go
func (*Absolute) String() string
```

**Parameters:**
  None

**Returns:**
- string

### Type



```go
func (*Inequality) Type() string
```

**Parameters:**
  None

**Returns:**
- string

### Variables



```go
func (*AllDifferentConstraint) Variables() []*FDVar
```

**Parameters:**
  None

**Returns:**
- []*FDVar

### Model
- Variables: decision variables with finite domains - Constraints: relationships that must hold among variables - Configuration: solver parameters and search heuristics Models are constructed incrementally by adding variables and constraints. Once constructed, models are immutable during solving, enabling safe concurrent access by parallel search workers. Thread safety: Models are safe for concurrent reads during solving, but must be constructed sequentially.

#### Example Usage

```go
// Create a new Model
model := Model{
    variables: [],
    constraints: [],
    variableIndex: map[],
    maxDomainSize: 42,
    config: &SolverConfig{}{},
    mu: /* value */,
}
```

#### Type Definition

```go
type Model struct {
    variables []*FDVariable
    constraints []ModelConstraint
    variableIndex map[int]*FDVariable
    maxDomainSize int
    config *SolverConfig
    mu sync.RWMutex
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| variables | `[]*FDVariable` | variables holds all decision variables in order of creation |
| constraints | `[]ModelConstraint` | constraints holds all constraints posted to the model |
| variableIndex | `map[int]*FDVariable` | variableIndex maps variable IDs to variable pointers for fast lookup |
| maxDomainSize | `int` | maxDomainSize is the largest domain size across all variables |
| config | `*SolverConfig` | config holds solver configuration (heuristics, limits, etc.) |
| mu | `sync.RWMutex` | mu protects model during construction |

### Constructor Functions

### NewModel

NewModel creates a new empty constraint model with default configuration.

```go
func NewModel() *Model
```

**Parameters:**
  None

**Returns:**
- *Model

### NewModelWithConfig

NewModelWithConfig creates a model with custom solver configuration.

```go
func NewModelWithConfig(config *SolverConfig) *Model
```

**Parameters:**
- `config` (*SolverConfig)

**Returns:**
- *Model

## Methods

### AddConstraint

AddConstraint adds a constraint to the model. Constraints are enforced during solving via propagation and search.

```go
func (*LocalConstraintStoreImpl) AddConstraint(constraint Constraint) error
```

**Parameters:**
- `constraint` (Constraint)

**Returns:**
- error

### AllDifferent

AllDifferent posts an AllDifferent constraint over vars.

```go
func (*Model) AllDifferent(vars ...*FDVariable) error
```

**Parameters:**
- `vars` (...*FDVariable)

**Returns:**
- error

### Among

Among posts an Among(vars, S, K) constraint to the model. It counts how many variables in vars take a value from the set S and encodes the count into K (see NewAmong for encoding details).

```go
func (*Model) Among(vars []*FDVariable, values []int, k *FDVariable) error
```

**Parameters:**
- `vars` ([]*FDVariable)
- `values` ([]int)
- `k` (*FDVariable)

**Returns:**
- error

### BinPacking

BinPacking posts a bin-packing constraint over items with given sizes and bin capacities. It's a thin wrapper around NewBinPacking.

```go
func (*Model) BinPacking(items []*FDVariable, sizes []int, capacities []int) error
```

**Parameters:**
- `items` ([]*FDVariable)
- `sizes` ([]int)
- `capacities` ([]int)

**Returns:**
- error

### Config

Config returns the solver configuration for this model.

```go
func (*Model) Config() *SolverConfig
```

**Parameters:**
  None

**Returns:**
- *SolverConfig

### ConstraintCount

ConstraintCount returns the number of constraints in the model.

```go
func (*Model) ConstraintCount() int
```

**Parameters:**
  None

**Returns:**
- int

### Constraints

Constraints returns all constraints in the model. The returned slice should not be modified.

```go
func (*Model) Constraints() []ModelConstraint
```

**Parameters:**
  None

**Returns:**
- []ModelConstraint

### Cumulative

Cumulative posts a Cumulative(starts, durations, demands, capacity) global constraint to the model. See NewCumulative for contract and semantics.

```go
func (*Model) Cumulative(starts []*FDVariable, durations, demands []int, capacity int) error
```

**Parameters:**
- `starts` ([]*FDVariable)
- `durations` ([]int)
- `demands` ([]int)
- `capacity` (int)

**Returns:**
- error

### GetVariable

GetVariable retrieves a variable by its ID. Returns nil if the ID doesn't exist.

```go
func (*Model) GetVariable(id int) *FDVariable
```

**Parameters:**
- `id` (int)

**Returns:**
- *FDVariable

### GlobalCardinality

GlobalCardinality posts a GCC over vars with per-value min/max occurrence bounds. See NewGlobalCardinality for requirements regarding slice lengths and indexing.

```go
func (*Model) GlobalCardinality(vars []*FDVariable, minCount, maxCount []int) error
```

**Parameters:**
- `vars` ([]*FDVariable)
- `minCount` ([]int)
- `maxCount` ([]int)

**Returns:**
- error

### IntVar

IntVar creates a new FD variable with integer domain [min..max]. If name is non-empty a named variable is created (useful in debugging and formatted output).

```go
func (*Model) IntVar(min, max int, name string) *FDVariable
```

**Parameters:**
- `min` (int)
- `max` (int)
- `name` (string)

**Returns:**
- *FDVariable

### IntVarValues

IntVarValues creates a new FD variable whose domain is exactly the provided non-contiguous set of values. If name is non-empty, the variable is named. Duplicate values are ignored. Empty or all non-positive values yield an empty domain which will cause the model to be immediately infeasible.

```go
func (*Model) IntVarValues(values []int, name string) *FDVariable
```

**Parameters:**
- `values` ([]int)
- `name` (string)

**Returns:**
- *FDVariable

### IntVars

IntVars creates count FD variables with domain [min..max]. If baseName is non-empty, variables are named baseName1, baseName2, ... baseNameN; otherwise anonymous variables are created.

```go
func (*Model) IntVars(count, min, max int, baseName string) []*FDVariable
```

**Parameters:**
- `count` (int)
- `min` (int)
- `max` (int)
- `baseName` (string)

**Returns:**
- []*FDVariable

### IntVarsWithNames

IntVarsWithNames creates FD variables with domain [min..max] using the given names. Handy for small models that benefit from explicit names.

```go
func (*Model) IntVarsWithNames(names []string, min, max int) []*FDVariable
```

**Parameters:**
- `names` ([]string)
- `min` (int)
- `max` (int)

**Returns:**
- []*FDVariable

### LexLess

LexLess posts a strict lexicographic ordering constraint X < Y.

```go
func (*Model) LexLess(xs, ys []*FDVariable) error
```

**Parameters:**
- `xs` ([]*FDVariable)
- `ys` ([]*FDVariable)

**Returns:**
- error

### LexLessEq

LexLessEq posts a non-strict lexicographic ordering X <= Y.

```go
func (*Model) LexLessEq(xs, ys []*FDVariable) error
```

**Parameters:**
- `xs` ([]*FDVariable)
- `ys` ([]*FDVariable)

**Returns:**
- error

### LinearSum

LinearSum posts Σ coeffs[i]*vars[i] = total, using bounds-consistent propagation.

```go
func (*Model) LinearSum(vars []*FDVariable, coeffs []int, total *FDVariable) error
```

**Parameters:**
- `vars` ([]*FDVariable)
- `coeffs` ([]int)
- `total` (*FDVariable)

**Returns:**
- error

### MaxDomainSize

MaxDomainSize returns the largest domain size in the model.

```go
func (*Model) MaxDomainSize() int
```

**Parameters:**
  None

**Returns:**
- int

### NewVariable

NewVariable creates and adds a new variable to the model with the specified domain. Returns the created variable which can be used to post constraints.

```go
func (*Model) NewVariable(domain Domain) *FDVariable
```

**Parameters:**
- `domain` (Domain)

**Returns:**
- *FDVariable

### NewVariableWithName

NewVariableWithName creates a named variable for easier debugging.

```go
func (*Model) NewVariableWithName(domain Domain, name string) *FDVariable
```

**Parameters:**
- `domain` (Domain)
- `name` (string)

**Returns:**
- *FDVariable

### NewVariables

NewVariables creates multiple variables with the same domain. Returns a slice of variables for convenient constraint posting.

```go
func (*Model) NewVariables(count int, domain Domain) []*FDVariable
```

**Parameters:**
- `count` (int)
- `domain` (Domain)

**Returns:**
- []*FDVariable

### NewVariablesWithNames

NewVariablesWithNames creates multiple named variables with the same domain.

```go
func (*Model) NewVariablesWithNames(names []string, domain Domain) []*FDVariable
```

**Parameters:**
- `names` ([]string)
- `domain` (Domain)

**Returns:**
- []*FDVariable

### NoOverlap

NoOverlap posts a NoOverlap(starts, durations) global constraint to the model. This is modeled via Cumulative with unit demands and capacity 1.

```go
func (*Model) NoOverlap(starts []*FDVariable, durations []int) error
```

**Parameters:**
- `starts` ([]*FDVariable)
- `durations` ([]int)

**Returns:**
- error

### Regular

Regular posts a Regular(vars, numStates, start, acceptStates, delta) DFA constraint.

```go
func (*Model) Regular(vars []*FDVariable, numStates, start int, acceptStates []int, delta [][]int) error
```

**Parameters:**
- `vars` ([]*FDVariable)
- `numStates` (int)
- `start` (int)
- `acceptStates` ([]int)
- `delta` ([][]int)

**Returns:**
- error

### SetConfig

SetConfig updates the solver configuration. Should be called before solving begins.

```go
func (*HybridSolver) SetConfig(config *HybridSolverConfig)
```

**Parameters:**
- `config` (*HybridSolverConfig)

**Returns:**
  None

### String

String returns a human-readable representation of the model.

```go
func (*Regular) String() string
```

**Parameters:**
  None

**Returns:**
- string

### Table

Table posts an extensional Table constraint over the given variables and rows.

```go
func (*Model) Table(vars []*FDVariable, rows [][]int) error
```

**Parameters:**
- `vars` ([]*FDVariable)
- `rows` ([][]int)

**Returns:**
- error

### Validate

Validate checks if the model is well-formed and ready for solving. Returns an error if: - Any variable has an empty domain - Any constraint references unknown variables - Configuration is invalid

```go
func (*Model) Validate() error
```

**Parameters:**
  None

**Returns:**
- error

### VariableCount

VariableCount returns the number of variables in the model.

```go
func (*Model) VariableCount() int
```

**Parameters:**
  None

**Returns:**
- int

### Variables

Variables returns all variables in the model. The returned slice should not be modified.

```go
func (*BoolSum) Variables() []*FDVariable
```

**Parameters:**
  None

**Returns:**
- []*FDVariable

### ModelConstraint
ModelConstraint represents a constraint within a model. Constraints restrict the values that variables can take simultaneously. Different constraint types provide different propagation strength: - AllDifferent: ensures variables take distinct values - Arithmetic: enforces arithmetic relationships (x + y = z) - Table: extensional constraints defined by allowed tuples - Global: specialized algorithms for common patterns ModelConstraints are immutable after creation and safe for concurrent access.

#### Example Usage

```go
// Example implementation of ModelConstraint
type MyModelConstraint struct {
    // Add your fields here
}

func (m MyModelConstraint) Variables() []*FDVariable {
    // Implement your logic here
    return
}

func (m MyModelConstraint) Type() string {
    // Implement your logic here
    return
}

func (m MyModelConstraint) String() string {
    // Implement your logic here
    return
}


```

#### Type Definition

```go
type ModelConstraint interface {
    Variables() []*FDVariable
    Type() string
    String() string
}
```

## Methods

| Method | Description |
| ------ | ----------- |

### Modulo
- Backward propagation: x ⊆ {q*modulus + remainder | q ≥ 0, remainder ∈ remainder.domain} This is arc-consistent propagation suitable for AC-3 and fixed-point iteration. Invariants: - modulus > 0 (enforced at construction) - All variables must have non-nil domains with positive integer values - Empty domain → immediate failure Thread Safety: Immutable after construction. Propagate() is safe for concurrent use.

#### Example Usage

```go
// Create a new Modulo
modulo := Modulo{
    x: &FDVariable{}{},
    modulus: 42,
    remainder: &FDVariable{}{},
}
```

#### Type Definition

```go
type Modulo struct {
    x *FDVariable
    modulus int
    remainder *FDVariable
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| x | `*FDVariable` | The value being divided |
| modulus | `int` | The constant modulus (must be > 0) |
| remainder | `*FDVariable` | The remainder (x mod modulus) |

### Constructor Functions

### NewModulo

NewModulo creates a new modulo constraint: remainder = x mod modulus.

```go
func NewModulo(x *FDVariable, modulus int, remainder *FDVariable) (*Modulo, error)
```

**Parameters:**

- `x` (*FDVariable) - The FD variable representing the input value

- `modulus` (int) - The constant integer modulus (must be > 0)

- `remainder` (*FDVariable) - The FD variable representing the remainder

**Returns:**
- *Modulo
- error

## Methods

### Clone

Clone creates a copy of the constraint with the same modulus. The variable references are shared (constraints are immutable).

```go
func (*Absolute) Clone() PropagationConstraint
```

**Parameters:**
  None

**Returns:**
- PropagationConstraint

### Propagate

Propagate applies bidirectional arc-consistency. Performs bidirectional arc-consistent propagation: 1. Forward: Prune remainder based on possible x mod modulus values 2. Backward: Prune x based on possible values that yield valid remainders 3. Detect conflicts: Empty domain after propagation → failure

```go
func (*Absolute) Propagate(solver *Solver, state *SolverState) (*SolverState, error)
```

**Parameters:**
- `solver` (*Solver)
- `state` (*SolverState)

**Returns:**
- *SolverState
- error

### String

String returns a human-readable representation of the constraint. Useful for debugging and logging. Implements ModelConstraint.

```go
func (*BoolSum) String() string
```

**Parameters:**
  None

**Returns:**
- string

### Type

Type returns the constraint type identifier. Implements ModelConstraint.

```go
func (*Among) Type() string
```

**Parameters:**
  None

**Returns:**
- string

### Variables

Variables returns the variables involved in this constraint. Used for dependency tracking and constraint graph construction. Implements ModelConstraint.

```go
func (*EqualityReified) Variables() []*FDVariable
```

**Parameters:**
  None

**Returns:**
- []*FDVariable

### backwardPropagate

backwardPropagate prunes the x domain based on remainder values. For each value r in remainder.domain: - Find all x values where x mod modulus = r - Keep x values that are in the original x domain - This generates: x ∈ {r, r+modulus, r+2*modulus, ...} ∩ x.domain Returns a new domain with only feasible x values.

```go
func (*Modulo) backwardPropagate(remainderDomain, xDomain Domain) Domain
```

**Parameters:**
- `remainderDomain` (Domain)
- `xDomain` (Domain)

**Returns:**
- Domain

### computeModulo

computeModulo computes x mod modulus, handling the constraint that BitSetDomain only supports positive integers (≥ 1). Since we're working with positive integers, this is straightforward.

```go
func (*Modulo) computeModulo(x int) int
```

**Parameters:**
- `x` (int)

**Returns:**
- int

### forwardPropagate

forwardPropagate prunes the remainder domain based on x values. For each value v in x.domain: - Compute r = v mod modulus - Keep r in remainder.domain if already present - Remove from remainder.domain if no x value can produce it Returns a new domain with only feasible remainder values.

```go
func (*Absolute) forwardPropagate(xDomain, absValueDomain Domain) Domain
```

**Parameters:**
- `xDomain` (Domain)
- `absValueDomain` (Domain)

**Returns:**
- Domain

### handleSelfReference

handleSelfReference handles the special case where X mod modulus = X. This is only valid when X < modulus.

```go
func (*Modulo) handleSelfReference(solver *Solver, state *SolverState, xDomain Domain) (*SolverState, error)
```

**Parameters:**
- `solver` (*Solver)
- `state` (*SolverState)
- `xDomain` (Domain)

**Returns:**
- *SolverState
- error

### NominalPlugin
NominalPlugin handles nominal logic constraints (freshness, alpha-equality) within the HybridSolver. Currently, it validates FreshnessConstraint instances against the UnifiedStore's relational bindings.

#### Example Usage

```go
// Create a new NominalPlugin
nominalplugin := NominalPlugin{

}
```

#### Type Definition

```go
type NominalPlugin struct {
}
```

### Constructor Functions

### NewNominalPlugin

NewNominalPlugin creates a new nominal plugin instance.

```go
func NewNominalPlugin() *NominalPlugin
```

**Parameters:**
  None

**Returns:**
- *NominalPlugin

## Methods

### CanHandle

CanHandle implements SolverPlugin. Returns true for nominal constraints we recognize.

```go
func (*FDPlugin) CanHandle(constraint interface{}) bool
```

**Parameters:**
- `constraint` (interface{})

**Returns:**
- bool

### Name

Name implements SolverPlugin.

```go
func (*FDVariable) Name() string
```

**Parameters:**
  None

**Returns:**
- string

### Propagate

Propagate implements SolverPlugin. Validates nominal constraints; returns error on violation. Note: This plugin currently does not modify the UnifiedStore. Future enhancements may include alpha-equivalence-aware normalization and derived constraints.

```go
func (*RationalLinearSum) Propagate(solver *Solver, state *SolverState) (*SolverState, error)
```

**Parameters:**
- `solver` (*Solver)
- `state` (*SolverState)

**Returns:**
- *SolverState
- error

### OptimizeOption
OptimizeOption configures SolveOptimalWithOptions behavior. Use helpers like WithTimeLimit, WithNodeLimit, WithTargetObjective, WithParallelWorkers, and WithHeuristics to customize the search.

#### Example Usage

```go
// Example usage of OptimizeOption
var value OptimizeOption
// Initialize with appropriate value
```

#### Type Definition

```go
type OptimizeOption func(*optConfig)
```

### Constructor Functions

### WithHeuristics

WithHeuristics overrides variable/value ordering heuristics for this solve call only.

```go
func WithHeuristics(v VariableOrderingHeuristic, val ValueOrderingHeuristic, seed int64) OptimizeOption
```

**Parameters:**
- `v` (VariableOrderingHeuristic)
- `val` (ValueOrderingHeuristic)
- `seed` (int64)

**Returns:**
- OptimizeOption

### WithNodeLimit

WithNodeLimit limits the number of search node expansions. When reached, the best incumbent is returned together with ErrSearchLimitReached.

```go
func WithNodeLimit(n int) OptimizeOption
```

**Parameters:**
- `n` (int)

**Returns:**
- OptimizeOption

### WithParallelWorkers

WithParallelWorkers enables parallel branch-and-bound using the shared work-queue infrastructure. Values <= 1 select sequential mode.

```go
func WithParallelWorkers(workers int) OptimizeOption
```

**Parameters:**
- `workers` (int)

**Returns:**
- OptimizeOption

### WithTargetObjective

WithTargetObjective requests early exit as soon as a solution with objective == target is found.

```go
func WithTargetObjective(target int) OptimizeOption
```

**Parameters:**
- `target` (int)

**Returns:**
- OptimizeOption

### WithTimeLimit

WithTimeLimit sets a hard time limit for the optimization. When reached, the best incumbent is returned together with context.DeadlineExceeded.

```go
func WithTimeLimit(d time.Duration) OptimizeOption
```

**Parameters:**
- `d` (time.Duration)

**Returns:**
- OptimizeOption

### Pair
Pair represents a cons cell (pair) in miniKanren. Pairs are used to build lists and other compound structures.

#### Example Usage

```go
// Create a new Pair
pair := Pair{
    car: Term{},
    cdr: Term{},
    mu: /* value */,
}
```

#### Type Definition

```go
type Pair struct {
    car Term
    cdr Term
    mu sync.RWMutex
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| car | `Term` | First element |
| cdr | `Term` | Rest of the structure |
| mu | `sync.RWMutex` | Protects concurrent access |

### Constructor Functions

### App

App constructs an application term using Pair(fun, arg). In this library, lambda application is represented as a Pair where Car is the function and Cdr is the argument. This aligns with s-expression conventions and interoperates with existing Pair-based traversal.

```go
func App(fun, arg Term) *Pair
```

**Parameters:**
- `fun` (Term)
- `arg` (Term)

**Returns:**
- *Pair

### NewPair

NewPair creates a new pair with the given car and cdr.

```go
func NewPair(car, cdr Term) *Pair
```

**Parameters:**
- `car` (Term)
- `cdr` (Term)

**Returns:**
- *Pair

## Methods

### Car

Car returns the first element of the pair.

```go
func (*Pair) Car() Term
```

**Parameters:**
  None

**Returns:**
- Term

### Cdr

Cdr returns the rest of the pair.

```go
func Cdr(pair, cdr Term) Goal
```

**Parameters:**
- `pair` (Term)
- `cdr` (Term)

**Returns:**
- Goal

### Clone

Clone creates a deep copy of the pair.

```go
func (*MembershipConstraint) Clone() Constraint
```

**Parameters:**
  None

**Returns:**
- Constraint

### Equal

Equal checks if two pairs are structurally equal.

```go
func (*Fact) Equal(other *Fact) bool
```

**Parameters:**
- `other` (*Fact)

**Returns:**
- bool

### IsVar

IsVar always returns false for pairs.

```go
func (*Pair) IsVar() bool
```

**Parameters:**
  None

**Returns:**
- bool

### String

String returns a string representation of the pair.

```go
func (*Regular) String() string
```

**Parameters:**
  None

**Returns:**
- string

### ParallelConfig
ParallelConfig holds configuration for parallel goal execution.

#### Example Usage

```go
// Create a new ParallelConfig
parallelconfig := ParallelConfig{
    MaxWorkers: 42,
    MaxQueueSize: 42,
    EnableBackpressure: true,
    RateLimit: 42,
}
```

#### Type Definition

```go
type ParallelConfig struct {
    MaxWorkers int
    MaxQueueSize int
    EnableBackpressure bool
    RateLimit int
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| MaxWorkers | `int` | MaxWorkers is the maximum number of concurrent workers. If 0, defaults to runtime.NumCPU(). |
| MaxQueueSize | `int` | MaxQueueSize is the maximum number of pending tasks. If 0, defaults to MaxWorkers * 10. |
| EnableBackpressure | `bool` | EnableBackpressure enables backpressure control to prevent memory exhaustion during large search spaces. |
| RateLimit | `int` | RateLimit sets the maximum operations per second. If 0, no rate limiting is applied. |

### Constructor Functions

### DefaultParallelConfig

DefaultParallelConfig returns a default configuration for parallel execution.

```go
func DefaultParallelConfig() *ParallelConfig
```

**Parameters:**
  None

**Returns:**
- *ParallelConfig

### ParallelExecutor
ParallelExecutor manages parallel execution of miniKanren goals.

#### Example Usage

```go
// Create a new ParallelExecutor
parallelexecutor := ParallelExecutor{
    config: &ParallelConfig{}{},
    workerPool: &/* value */{},
    backpressureCtrl: &/* value */{},
    rateLimiter: &/* value */{},
    mu: /* value */,
    shutdown: true,
}
```

#### Type Definition

```go
type ParallelExecutor struct {
    config *ParallelConfig
    workerPool *parallel.WorkerPool
    backpressureCtrl *parallel.BackpressureController
    rateLimiter *parallel.RateLimiter
    mu sync.RWMutex
    shutdown bool
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| config | `*ParallelConfig` |  |
| workerPool | `*parallel.WorkerPool` |  |
| backpressureCtrl | `*parallel.BackpressureController` |  |
| rateLimiter | `*parallel.RateLimiter` |  |
| mu | `sync.RWMutex` |  |
| shutdown | `bool` |  |

### Constructor Functions

### NewParallelExecutor

NewParallelExecutor creates a new parallel executor with the given configuration.

```go
func NewParallelExecutor(config *ParallelConfig) *ParallelExecutor
```

**Parameters:**
- `config` (*ParallelConfig)

**Returns:**
- *ParallelExecutor

## Methods

### ParallelDisj

ParallelDisj creates a disjunction goal that evaluates all sub-goals in parallel using the parallel executor. This can significantly improve performance when dealing with computationally intensive goals or large search spaces.

```go
func (*ParallelExecutor) ParallelDisj(goals ...Goal) Goal
```

**Parameters:**
- `goals` (...Goal)

**Returns:**
- Goal

### Shutdown

Shutdown gracefully shuts down the parallel executor.

```go
func (*ParallelExecutor) Shutdown()
```

**Parameters:**
  None

**Returns:**
  None

### ParallelSearchConfig
ParallelSearchConfig holds configuration for parallel backtracking search.

#### Example Usage

```go
// Create a new ParallelSearchConfig
parallelsearchconfig := ParallelSearchConfig{
    NumWorkers: 42,
    WorkQueueSize: 42,
}
```

#### Type Definition

```go
type ParallelSearchConfig struct {
    NumWorkers int
    WorkQueueSize int
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| NumWorkers | `int` | NumWorkers is the number of parallel worker goroutines. If 0 or negative, defaults to runtime.NumCPU(). |
| WorkQueueSize | `int` | WorkQueueSize is the buffer size for the work channel. Larger values allow more work to be queued, potentially improving load balancing at the cost of memory. |

### Constructor Functions

### DefaultParallelSearchConfig

DefaultParallelSearchConfig returns the default parallel search configuration.

```go
func DefaultParallelSearchConfig() *ParallelSearchConfig
```

**Parameters:**
  None

**Returns:**
- *ParallelSearchConfig

### ParallelStream
ParallelStream represents a stream that can be evaluated in parallel. It wraps the standard Stream with additional parallel capabilities.

#### Example Usage

```go
// Create a new ParallelStream
parallelstream := ParallelStream{
    executor: &ParallelExecutor{}{},
    ctx: /* value */,
}
```

#### Type Definition

```go
type ParallelStream struct {
    *Stream
    executor *ParallelExecutor
    ctx context.Context
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| **Stream | `*Stream` |  |
| executor | `*ParallelExecutor` |  |
| ctx | `context.Context` |  |

### Constructor Functions

### NewParallelStream

NewParallelStream creates a new parallel stream with the given executor.

```go
func NewParallelStream(ctx context.Context, executor *ParallelExecutor) *ParallelStream
```

**Parameters:**
- `ctx` (context.Context)
- `executor` (*ParallelExecutor)

**Returns:**
- *ParallelStream

## Methods

### Collect

Collect gathers all constraint stores from the parallel stream.

```go
func (*ParallelStream) Collect() []ConstraintStore
```

**Parameters:**
  None

**Returns:**
- []ConstraintStore

### ParallelFilter

ParallelFilter filters constraint stores in the stream in parallel.

```go
func (*ParallelStream) ParallelFilter(predicate func(ConstraintStore) bool) *ParallelStream
```

**Parameters:**
- `predicate` (func(ConstraintStore) bool)

**Returns:**
- *ParallelStream

### ParallelMap

ParallelMap applies a function to each constraint store in the stream in parallel.

```go
func (*ParallelStream) ParallelMap(fn func(ConstraintStore) ConstraintStore) *ParallelStream
```

**Parameters:**
- `fn` (func(ConstraintStore) ConstraintStore)

**Returns:**
- *ParallelStream

### PatternClause
PatternClause represents a single pattern matching clause. Each clause consists of a pattern term and a sequence of goals to execute if the pattern matches. The pattern is unified with the input term. If unification succeeds, the goals are executed in sequence (as if by Conj).

#### Example Usage

```go
// Create a new PatternClause
patternclause := PatternClause{
    Pattern: Term{},
    Goals: [],
}
```

#### Type Definition

```go
type PatternClause struct {
    Pattern Term
    Goals []Goal
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| Pattern | `Term` | Pattern is the term to match against. Can contain fresh variables that will be bound if the pattern matches. |
| Goals | `[]Goal` | Goals are executed in sequence if the pattern matches. An empty goal list succeeds immediately (useful for filtering). |

### Constructor Functions

### NewClause

NewClause creates a pattern matching clause from a pattern and goals. This is a convenience constructor for PatternClause. Example: clause := NewClause(Nil, Eq(result, NewAtom(0)))

```go
func NewClause(pattern Term, goals ...Goal) PatternClause
```

**Parameters:**
- `pattern` (Term)
- `goals` (...Goal)

**Returns:**
- PatternClause

### PropagationConstraint
PropagationConstraint extends ModelConstraint with active domain pruning. This interface bridges the declarative ModelConstraint with the propagation engine. Propagation maintains copy-on-write semantics: constraints never modify state in-place but return a new state with pruned domains. This preserves the lock-free property critical for parallel search.

#### Example Usage

```go
// Example implementation of PropagationConstraint
type MyPropagationConstraint struct {
    // Add your fields here
}

func (m MyPropagationConstraint) Propagate(param1 *Solver, param2 *SolverState) *SolverState {
    // Implement your logic here
    return
}


```

#### Type Definition

```go
type PropagationConstraint interface {
    ModelConstraint
    Propagate(solver *Solver, state *SolverState) (*SolverState, error)
}
```

## Methods

| Method | Description |
| ------ | ----------- |

### Constructor Functions

### NewAmong

NewAmong creates an Among(vars, S, K) constraint.

```go
func NewAmong(vars []*FDVariable, values []int, k *FDVariable) (PropagationConstraint, error)
```

**Parameters:**

- `vars` ([]*FDVariable) - non-empty list of variables

- `values` ([]int) - the explicit set S of allowed values (each in [1..maxValue]); duplicates are ignored

- `k` (*FDVariable) - the encoded count variable with domain in [1..len(vars)+1]

**Returns:**
- PropagationConstraint
- error

### NewCumulative

NewCumulative constructs a Cumulative constraint.

```go
func NewCumulative(starts []*FDVariable, durations, demands []int, capacity int) (PropagationConstraint, error)
```

**Parameters:**

- `starts` ([]*FDVariable) - start-time variables (length n > 0)

- `durations` ([]int) - positive durations (length n; each > 0)

- `demands` ([]int) - non-negative demands (length n; each >= 0)

- `capacity` (int) - total resource capacity (must be > 0)

**Returns:**
- PropagationConstraint
- error

### NewGlobalCardinality

NewGlobalCardinality constructs a GCC over vars with per-value min/max bounds. minCount and maxCount must be length >= M+1 where M is the maximum domain value across vars; indexes 1..M are used. For values not present, bounds may be zero.

```go
func NewGlobalCardinality(vars []*FDVariable, minCount, maxCount []int) (PropagationConstraint, error)
```

**Parameters:**
- `vars` ([]*FDVariable)
- `minCount` ([]int)
- `maxCount` ([]int)

**Returns:**
- PropagationConstraint
- error

### NewLexLess

NewLexLess creates a strict lexicographic ordering constraint X < Y.

```go
func NewLexLess(xs, ys []*FDVariable) (PropagationConstraint, error)
```

**Parameters:**
- `xs` ([]*FDVariable)
- `ys` ([]*FDVariable)

**Returns:**
- PropagationConstraint
- error

### NewLexLessEq

NewLexLessEq creates a non-strict lexicographic ordering constraint X ≤ Y.

```go
func NewLexLessEq(xs, ys []*FDVariable) (PropagationConstraint, error)
```

**Parameters:**
- `xs` ([]*FDVariable)
- `ys` ([]*FDVariable)

**Returns:**
- PropagationConstraint
- error

### NewMax

NewMax creates a MaxOfArray constraint with result variable r. Contract: - vars: non-empty slice; each variable must have a positive domain (1..MaxValue) - r: non-nil result variable with a positive domain

```go
func NewMax(vars []*FDVariable, r *FDVariable) (PropagationConstraint, error)
```

**Parameters:**

- `vars` ([]*FDVariable) - non-empty slice; each variable must have a positive domain (1..MaxValue)

- `r` (*FDVariable) - non-nil result variable with a positive domain

**Returns:**
- PropagationConstraint
- error

### NewMin

NewMin creates a MinOfArray constraint with result variable r. Contract: - vars: non-empty slice; each variable must have a positive domain (1..MaxValue) - r: non-nil result variable with a positive domain

```go
func NewMin(vars []*FDVariable, r *FDVariable) (PropagationConstraint, error)
```

**Parameters:**

- `vars` ([]*FDVariable) - non-empty slice; each variable must have a positive domain (1..MaxValue)

- `r` (*FDVariable) - non-nil result variable with a positive domain

**Returns:**
- PropagationConstraint
- error

### NewNoOverlap

NewNoOverlap constructs a NoOverlap (disjunctive) constraint over tasks.

```go
func NewNoOverlap(starts []*FDVariable, durations []int) (PropagationConstraint, error)
```

**Parameters:**

- `starts` ([]*FDVariable) - start-time FD variables (len n > 0)

- `durations` ([]int) - strictly positive integer durations (len n; each > 0)

**Returns:**
- PropagationConstraint
- error

### newLex



```go
func newLex(xs, ys []*FDVariable, k lexKind) (PropagationConstraint, error)
```

**Parameters:**
- `xs` ([]*FDVariable)
- `ys` ([]*FDVariable)
- `k` (lexKind)

**Returns:**
- PropagationConstraint
- error

### Rational
This enables exact representation of fractional coefficients without floating-point errors. Common irrational approximations: π ≈ 22/7 (Archimedes, error ~0.04%) π ≈ 355/113 (Zu Chongzhi, error ~0.000008%) √2 ≈ 99/70 (accurate to 4 decimals) √2 ≈ 1393/985 (accurate to 6 decimals) e ≈ 2721/1000 (accurate to 4 decimals) φ (golden ratio) ≈ 1618/1000 (accurate to 3 decimals)

#### Example Usage

```go
// Create a new Rational
rational := Rational{
    Num: 42,
    Den: 42,
}
```

#### Type Definition

```go
type Rational struct {
    Num int
    Den int
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| Num | `int` | numerator |
| Den | `int` | denominator (always > 0 after normalization) |

### Constructor Functions

### ApproximateIrrational

ApproximateIrrational provides rational approximations for common irrational values. Returns a rational with the requested precision (number of decimal places). Supported values: "pi", "sqrt2", "e", "phi" Uses continued fraction approximations for higher precision. For simplicity, this implementation provides fixed precision levels.

```go
func ApproximateIrrational(name string, precision int) (Rational, error)
```

**Parameters:**
- `name` (string)
- `precision` (int)

**Returns:**
- Rational
- error

### FromFloat

FromFloat creates a rational approximation of a floating-point number. Uses continued fraction algorithm with the specified maximum denominator. Warning: This is an approximation. For known constants like π, use ApproximateIrrational instead. Example: FromFloat(3.14159, 1000) ≈ 355/113 (close to π)

```go
func FromFloat(f float64, maxDenominator int) Rational
```

**Parameters:**
- `f` (float64)
- `maxDenominator` (int)

**Returns:**
- Rational

### NewRational

NewRational creates a rational number num/den in normalized form. Panics if denominator is zero. Normalization ensures: - GCD(num, den) = 1 (reduced to lowest terms) - den > 0 (sign stored in numerator) Examples: NewRational(6, 8) → 3/4 NewRational(-6, 8) → -3/4 NewRational(6, -8) → -3/4 NewRational(0, 5) → 0/1

```go
func NewRational(num, den int) Rational
```

**Parameters:**
- `num` (int)
- `den` (int)

**Returns:**
- Rational

## Methods

### Add

Add returns the sum of two rational numbers: r + other. Algorithm: a/b + c/d = (a*d + b*c) / (b*d), then normalize. Example: (1/2) + (1/3) = 3/6 + 2/6 = 5/6

```go
func (Rational) Add(other Rational) Rational
```

**Parameters:**
- `other` (Rational)

**Returns:**
- Rational

### Div

Div returns the quotient of two rational numbers: r / other. Panics if other is zero. Algorithm: (a/b) / (c/d) = (a/b) * (d/c) = (a*d) / (b*c), then normalize. Example: (3/4) / (2/3) = (3/4) * (3/2) = 9/8

```go
func (Rational) Div(other Rational) Rational
```

**Parameters:**
- `other` (Rational)

**Returns:**
- Rational

### Equals

Equals returns true if two rational numbers are equal. Since rationals are normalized, structural equality is sufficient.

```go
func (Rational) Equals(other Rational) bool
```

**Parameters:**
- `other` (Rational)

**Returns:**
- bool

### IsNegative

IsNegative returns true if the rational number is less than zero.

```go
func (Rational) IsNegative() bool
```

**Parameters:**
  None

**Returns:**
- bool

### IsPositive

IsPositive returns true if the rational number is greater than zero.

```go
func (Rational) IsPositive() bool
```

**Parameters:**
  None

**Returns:**
- bool

### IsZero

IsZero returns true if the rational number is zero.

```go
func (Rational) IsZero() bool
```

**Parameters:**
  None

**Returns:**
- bool

### Mul

Mul returns the product of two rational numbers: r * other. Algorithm: (a/b) * (c/d) = (a*c) / (b*d), then normalize. Example: (2/3) * (3/4) = 6/12 = 1/2

```go
func (Rational) Mul(other Rational) Rational
```

**Parameters:**
- `other` (Rational)

**Returns:**
- Rational

### Neg

Neg returns the negation of the rational number: -r. Example: -(3/4) = -3/4

```go
func (Rational) Neg() Rational
```

**Parameters:**
  None

**Returns:**
- Rational

### String

String returns a string representation of the rational number. Format: "num/den" for non-integers, "num" for integers (den=1). Examples: Rational{3, 4}.String() → "3/4" Rational{6, 1}.String() → "6" Rational{-5, 2}.String() → "-5/2"

```go
func (*BinPacking) String() string
```

**Parameters:**
  None

**Returns:**
- string

### Sub

Sub returns the difference of two rational numbers: r - other. Example: (3/4) - (1/2) = 3/4 - 2/4 = 1/4

```go
func (Rational) Sub(other Rational) Rational
```

**Parameters:**
- `other` (Rational)

**Returns:**
- Rational

### ToFloat

ToFloat returns the floating-point approximation of the rational number. Useful for debugging or when exact precision is not required. Example: Rational{22, 7}.ToFloat() ≈ 3.142857...

```go
func (Rational) ToFloat() float64
```

**Parameters:**
  None

**Returns:**
- float64

### RationalLinearSum
Scaled: 2*x + 3*y = 6*z This enables exact rational coefficient constraints while leveraging existing integer domain infrastructure and propagation algorithms. Use cases: - Irrational approximations: π*diameter = circumference → (22/7)*d = c - Percentage calculations: 10% bonus → (1/10)*salary = bonus - Unit conversions with fractional ratios: (5/9)*(F-32) = C - Recipe scaling: (3/4)*flour + (1/2)*sugar = mixture

#### Example Usage

```go
// Create a new RationalLinearSum
rationallinearsum := RationalLinearSum{
    vars: [],
    coeffs: [],
    result: &FDVariable{}{},
    scale: 42,
    intCoeffs: [],
    underlying: &LinearSum{}{},
}
```

#### Type Definition

```go
type RationalLinearSum struct {
    vars []*FDVariable
    coeffs []Rational
    result *FDVariable
    scale int
    intCoeffs []int
    underlying *LinearSum
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| vars | `[]*FDVariable` | variables with rational coefficients |
| coeffs | `[]Rational` | rational coefficients |
| result | `*FDVariable` | result variable |
| scale | `int` | LCM of all denominators (scaling factor) |
| intCoeffs | `[]int` | scaled integer coefficients |
| underlying | `*LinearSum` | delegated integer constraint |

### Constructor Functions

### NewRationalLinearSum

NewRationalLinearSum creates a rational linear sum constraint. Requires that all variables and coefficients have matching lengths. The constraint is automatically converted to integer form via LCM scaling: scale = LCM(all denominators including result's implicit 1) intCoeffs[i] = coeffs[i].Num * (scale / coeffs[i].Den) Then creates underlying constraint: intCoeffs[0]*vars[0] + ... = result IMPORTANT: When scale > 1, the result variable's domain must be pre-scaled. For example, if scale = 6: - User constraint: (1/3)*x + (1/2)*y = z - Internal: 2*x + 3*y = 6*z - If x∈[3,9] and y∈[2,4], then z's domain should be pre-divided by 6 - Or use ScaledDivision to handle the scaling For coefficients that share denominators with result's implicit 1, scale will be 1. Example: (1/1)*x + (2/1)*y = z → scale=1 → x + 2*y = z (direct mapping) Panics if: - Any variable is nil - Result is nil - Length of vars and coeffs don't match - Any coefficient is zero (use fewer variables instead) Example (scale = 1): // Constraint: 2*x + 3*y = z (integer coefficients) c, _ := NewRationalLinearSum( []*FDVariable{x, y}, []Rational{NewRational(2,1), NewRational(3,1)}, z, ) Example (scale = 6, requires pre-scaled result): // Constraint: (1/3)*x + (1/2)*y = z // Internal: 2*x + 3*y = 6*z // User must ensure z's domain accounts for factor of 6 c, _ := NewRationalLinearSum( []*FDVariable{x, y}, []Rational{NewRational(1,3), NewRational(1,2)}, z, // z's domain should be ⌊original_range / 6⌋ )

```go
func NewRationalLinearSum(vars []*FDVariable, coeffs []Rational, result *FDVariable) (*RationalLinearSum, error)
```

**Parameters:**
- `vars` ([]*FDVariable)
- `coeffs` ([]Rational)
- `result` (*FDVariable)

**Returns:**
- *RationalLinearSum
- error

## Methods

### Clone

Clone implements ModelConstraint.

```go
func (*UnifiedStore) Clone() *UnifiedStore
```

**Parameters:**
  None

**Returns:**
- *UnifiedStore

### GetIntCoeffs

GetIntCoeffs returns the scaled integer coefficients used internally. These are the numerators after multiplying each coefficient by (scale / denominator).

```go
func (*RationalLinearSum) GetIntCoeffs() []int
```

**Parameters:**
  None

**Returns:**
- []int

### GetScale

GetScale returns the LCM scaling factor used to convert rational coefficients to integers. Useful for debugging or understanding the internal representation.

```go
func (*RationalLinearSum) GetScale() int
```

**Parameters:**
  None

**Returns:**
- int

### Propagate

Propagate implements PropagationConstraint by delegating to underlying integer LinearSum.

```go
func (*ElementValues) Propagate(solver *Solver, state *SolverState) (*SolverState, error)
```

**Parameters:**
- `solver` (*Solver)
- `state` (*SolverState)

**Returns:**
- *SolverState
- error

### String

String implements ModelConstraint.

```go
func (*Absolute) String() string
```

**Parameters:**
  None

**Returns:**
- string

### Type

Type implements ModelConstraint.

```go
func (*Regular) Type() string
```

**Parameters:**
  None

**Returns:**
- string

### Variables

Variables implements ModelConstraint.

```go
func (*EqualityReified) Variables() []*FDVariable
```

**Parameters:**
  None

**Returns:**
- []*FDVariable

### Regular
Regular is the DFA-based global constraint over a sequence of variables.

#### Example Usage

```go
// Create a new Regular
regular := Regular{
    vars: [],
    numStates: 42,
    start: 42,
    accept: [],
    delta: [],
    alphabetMax: 42,
}
```

#### Type Definition

```go
type Regular struct {
    vars []*FDVariable
    numStates int
    start int
    accept []bool
    delta [][]int
    alphabetMax int
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| vars | `[]*FDVariable` |  |
| numStates | `int` |  |
| start | `int` |  |
| accept | `[]bool` | length = numStates+1, 1-based indexing |
| delta | `[][]int` | 1-based: delta[s][v] -> next state in [1..numStates] or 0 |
| alphabetMax | `int` | maximum symbol index covered by delta rows |

### Constructor Functions

### NewRegular

NewRegular constructs a Regular constraint over vars with a DFA.

```go
func NewRegular(vars []*FDVariable, numStates, start int, acceptStates []int, delta [][]int) (*Regular, error)
```

**Parameters:**

- `vars` ([]*FDVariable) - non-empty slice of FD variables (non-nil), the sequence x1..xn

- `numStates` (int) - number of DFA states (>=1), states are 1..numStates

- `start` (int) - start state in [1..numStates]

- `acceptStates` ([]int) - list of accepting states (each in [1..numStates], may repeat)

- `delta` ([][]int) - transition table; must have numStates rows; each row length must be

**Returns:**
- *Regular
- error

## Methods

### Propagate

Propagate applies forward/backward DFA filtering to prune variable domains. Implements PropagationConstraint.

```go
func (*Stretch) Propagate(solver *Solver, state *SolverState) (*SolverState, error)
```

**Parameters:**
- `solver` (*Solver)
- `state` (*SolverState)

**Returns:**
- *SolverState
- error

### String

String implements ModelConstraint.

```go
func (*BinPacking) String() string
```

**Parameters:**
  None

**Returns:**
- string

### Type

Type implements ModelConstraint.

```go
func (*EqualityReified) Type() string
```

**Parameters:**
  None

**Returns:**
- string

### Variables

Variables implements ModelConstraint.

```go
func (*EqualityReified) Variables() []*FDVariable
```

**Parameters:**
  None

**Returns:**
- []*FDVariable

### ReifiedConstraint
4. When boolean = 1 → ensure constraint is violated (complex, often via search) For simplicity, this implementation focuses on cases 1–3. Case 4 (forcing a constraint to be false) is challenging and often requires specialized negation logic per constraint type. We handle it by: - If boolean is bound to 1 (false), we skip constraint propagation - The search will naturally find assignments that violate the constraint This is sound but may be weaker than full constraint negation. For many use cases (including Count built via equality reification), this is sufficient.

#### Example Usage

```go
// Create a new ReifiedConstraint
reifiedconstraint := ReifiedConstraint{
    constraint: PropagationConstraint{},
    boolVar: &FDVariable{}{},
}
```

#### Type Definition

```go
type ReifiedConstraint struct {
    constraint PropagationConstraint
    boolVar *FDVariable
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| constraint | `PropagationConstraint` | Underlying constraint to reify |
| boolVar | `*FDVariable` | Boolean variable (domain must be {0,1}) |

### Constructor Functions

### NewReifiedConstraint

NewReifiedConstraint creates a reified constraint.

```go
func NewReifiedConstraint(constraint PropagationConstraint, boolVar *FDVariable) (*ReifiedConstraint, error)
```

**Parameters:**

- `constraint` (PropagationConstraint) - the constraint to reify (must not be nil)

- `boolVar` (*FDVariable) - boolean variable with domain subset of {1,2} reflecting truth value

**Returns:**
- *ReifiedConstraint
- error

## Methods

### BoolVar

BoolVar returns the boolean variable associated with this reified constraint. Useful for accessing the truth value during or after solving.

```go
func (*ReifiedConstraint) BoolVar() *FDVariable
```

**Parameters:**
  None

**Returns:**
- *FDVariable

### Constraint

Constraint returns the underlying constraint being reified. Useful for introspection and debugging.

```go
func (*ReifiedConstraint) Constraint() PropagationConstraint
```

**Parameters:**
  None

**Returns:**
- PropagationConstraint

### Propagate

Propagate applies reification logic with bidirectional propagation. Algorithm: 1. Check boolean variable's domain: - If bound to 1: propagate underlying constraint normally - If bound to 0: constraint is disabled (we don't enforce violation) - If {0,1}: attempt propagation and check if constraint is determined 2. If boolean is not yet bound and we propagate: - Try propagating the constraint - If constraint leads to failure → set boolean to 0 - If constraint is trivially satisfied → set boolean to 1 - Otherwise, boolean remains {0,1} Implements PropagationConstraint.

```go
func (*AllDifferentConstraint) Propagate(store *FDStore) (bool, error)
```

**Parameters:**
- `store` (*FDStore)

**Returns:**
- bool
- error

### String

String returns a human-readable representation. Implements ModelConstraint.

```go
func (*EqualityReified) String() string
```

**Parameters:**
  None

**Returns:**
- string

### Type

Type returns the constraint type identifier. Implements ModelConstraint.

```go
func (*Cumulative) Type() string
```

**Parameters:**
  None

**Returns:**
- string

### Variables

Variables returns all variables involved in this reified constraint. Includes both the constraint's variables and the boolean variable. Implements ModelConstraint.

```go
func (*AllDifferentConstraint) Variables() []*FDVar
```

**Parameters:**
  None

**Returns:**
- []*FDVar

### enforceNegation

enforceNegation applies the logical negation of the underlying constraint as much as is practical without introducing new variables. The intent is to prevent solutions that would make the constraint true when the boolean is 1. Strategy by constraint type: - Arithmetic (dst = src + k): - If both bound and satisfy equality → conflict - If one side bound → remove the matching value from the other - Inequality: - LessThan    false → enforce X ≥ Y via bounds - LessEqual   false → enforce X > Y via bounds - GreaterThan false → enforce X ≤ Y via bounds - GreaterEqual false→ enforce X < Y via bounds - NotEqual    false → enforce X = Y by intersecting domains - AllDifferent: - If all vars bound and all distinct → conflict (since NOT AllDifferent must hold) - Otherwise, no pruning (would require disjunction of equalities)

```go
func (*ReifiedConstraint) enforceNegation(solver *Solver, state *SolverState) (*SolverState, error)
```

**Parameters:**
- `solver` (*Solver)
- `state` (*SolverState)

**Returns:**
- *SolverState
- error

### isConstraintDetermined

isConstraintDetermined checks if the constraint's satisfaction is determined.

```go
func (*ReifiedConstraint) isConstraintDetermined(solver *Solver, state *SolverState) (bool, bool, error)
```

**Parameters:**
- `solver` (*Solver)
- `state` (*SolverState)

**Returns:**

- bool - true if we can definitively say if constraint is sat/unsat

- bool - if isDetermined, whether constraint is satisfied

- error - if check fails

### Relation
Relation represents a named relation with a fixed arity and indexed columns. Relations are immutable after creation.

#### Example Usage

```go
// Create a new Relation
relation := Relation{
    name: "example",
    arity: 42,
    indexes: map[],
}
```

#### Type Definition

```go
type Relation struct {
    name string
    arity int
    indexes map[int]bool
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| name | `string` |  |
| arity | `int` |  |
| indexes | `map[int]bool` | set of indexed column positions (0-based) |

### Constructor Functions

### DbRel

DbRel creates a new relation with the given name, arity, and optional indexed columns. Column indexes are 0-based. Indexing a column enables O(1) lookups for queries with ground terms in that position. Example: parent := DbRel("parent", 2, 0, 1)  // Both columns indexed edge := DbRel("edge", 2, 0)         // Only source column indexed Returns an error if arity is <= 0 or if any index is out of range.

```go
func DbRel(name string, arity int, indexedCols ...int) (*Relation, error)
```

**Parameters:**
- `name` (string)
- `arity` (int)
- `indexedCols` (...int)

**Returns:**
- *Relation
- error

### MustRel

MustRel creates a Relation and panics on error. Useful in examples/tests where arity and indexes are known constants.

```go
func MustRel(name string, arity int, indexedCols ...int) *Relation
```

**Parameters:**
- `name` (string)
- `arity` (int)
- `indexedCols` (...int)

**Returns:**
- *Relation

## Methods

### Arity

Arity returns the relation's arity (number of columns).

```go
func (*Relation) Arity() int
```

**Parameters:**
  None

**Returns:**
- int

### IsIndexed

IsIndexed returns true if the given column is indexed.

```go
func (*Relation) IsIndexed(col int) bool
```

**Parameters:**
- `col` (int)

**Returns:**
- bool

### Name

Name returns the relation's name.

```go
func (*RelationalPlugin) Name() string
```

**Parameters:**
  None

**Returns:**
- string

### RelationalPlugin
1. Extracts relational bindings from the UnifiedStore 2. Checks each Constraint against those bindings 3. Returns error if any constraint is violated 4. Returns original store if all constraints are satisfied or pending The relational plugin doesn't typically modify the store (no pruning), it just validates that current bindings don't violate constraints. However, if FD domains narrow variables to singletons, those singleton values can be promoted to relational bindings, enabling cross-solver propagation.

#### Example Usage

```go
// Create a new RelationalPlugin
relationalplugin := RelationalPlugin{

}
```

#### Type Definition

```go
type RelationalPlugin struct {
}
```

### Constructor Functions

### NewRelationalPlugin

NewRelationalPlugin creates a new relational constraint plugin.

```go
func NewRelationalPlugin() *RelationalPlugin
```

**Parameters:**
  None

**Returns:**
- *RelationalPlugin

## Methods

### CanHandle

CanHandle returns true if the constraint is a relational constraint. Implements SolverPlugin.

```go
func (*RelationalPlugin) CanHandle(constraint interface{}) bool
```

**Parameters:**
- `constraint` (interface{})

**Returns:**
- bool

### Name

Name returns the plugin identifier. Implements SolverPlugin.

```go
func (*FDVariable) Name() string
```

**Parameters:**
  None

**Returns:**
- string

### Propagate

Propagate checks all relational constraints in the store. Returns error if any constraint is violated, otherwise returns the store unchanged. Implements SolverPlugin.

```go
func (*BinPacking) Propagate(solver *Solver, state *SolverState) (*SolverState, error)
```

**Parameters:**
- `solver` (*Solver)
- `state` (*SolverState)

**Returns:**
- *SolverState
- error

### promoteSingletons

promoteSingletons checks FD domains for singleton values and promotes them to relational bindings. This enables the relational solver to use information from FD propagation. Example: If FD propagation narrows X's domain to {5}, we can add the relational binding X=5, allowing relational constraints to fire.

```go
func (*RelationalPlugin) promoteSingletons(store *UnifiedStore) (*UnifiedStore, error)
```

**Parameters:**
- `store` (*UnifiedStore)

**Returns:**
- *UnifiedStore
- error

### propagateBindingsToDomains

propagateBindingsToDomains synchronizes relational bindings to FD domains. When a variable is bound relationally (x=5), and it has an FD domain, we prune the FD domain to contain only the bound value. This enables bidirectional propagation: - Relational says x=5 → FD domain becomes {5} - Relational says x≠3 → 3 is removed from FD domain (future enhancement) This ensures attributed variables (with both bindings and domains) remain consistent across solver boundaries.

```go
func (*RelationalPlugin) propagateBindingsToDomains(store *UnifiedStore) (*UnifiedStore, error)
```

**Parameters:**
- `store` (*UnifiedStore)

**Returns:**
- *UnifiedStore
- error

### SCC
SCC represents a strongly connected component in the dependency graph. Used for cycle detection and fixpoint computation.

#### Example Usage

```go
// Create a new SCC
scc := SCC{
    nodes: [],
    index: 42,
}
```

#### Type Definition

```go
type SCC struct {
    nodes []*SubgoalEntry
    index int
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| nodes | `[]*SubgoalEntry` | Nodes in this SCC |
| index | `int` | SCC index (for topological ordering) |

## Methods

### AnswerCount

AnswerCount returns the total number of answers across all nodes in the SCC.

```go
func (*SCC) AnswerCount() int64
```

**Parameters:**
  None

**Returns:**
- int64

### Contains

Contains checks if the SCC contains the given entry.

```go
func (*SCC) Contains(entry *SubgoalEntry) bool
```

**Parameters:**
- `entry` (*SubgoalEntry)

**Returns:**
- bool

### SLGConfig
SLGConfig holds configuration for the SLG engine.

#### Example Usage

```go
// Create a new SLGConfig
slgconfig := SLGConfig{
    MaxTableSize: 42,
    MaxAnswersPerSubgoal: 42,
    MaxFixpointIterations: 42,
    EnableParallelProducers: true,
    EnableSubsumptionChecking: true,
    EnforceStratification: true,
    DebugWFS: true,
    NegationPeekTimeout: /* value */,
}
```

#### Type Definition

```go
type SLGConfig struct {
    MaxTableSize int64
    MaxAnswersPerSubgoal int64
    MaxFixpointIterations int
    EnableParallelProducers bool
    EnableSubsumptionChecking bool
    EnforceStratification bool
    DebugWFS bool
    NegationPeekTimeout time.Duration
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| MaxTableSize | `int64` | MaxTableSize limits the total number of subgoals (0 = unlimited) |
| MaxAnswersPerSubgoal | `int64` | MaxAnswersPerSubgoal limits answers per subgoal (0 = unlimited) |
| MaxFixpointIterations | `int` | MaxFixpointIterations limits iterations for cyclic computations |
| EnableParallelProducers | `bool` | EnableParallelProducers allows multiple producers per subgoal |
| EnableSubsumptionChecking | `bool` | EnableSubsumptionChecking enables answer subsumption (future enhancement) |
| EnforceStratification | `bool` | EnforceStratification controls whether negation is restricted by strata. When true (default), a predicate may only negate predicates in the same or lower stratum; negating a higher stratum is a violation and yields no answers. When false, general WFS with unfounded-set handling applies. |
| DebugWFS | `bool` | DebugWFS enables verbose tracing for WFS/negation synchronization paths. Prefer enabling via environment variable gokanlogic_WFS_TRACE=1 when possible. |
| NegationPeekTimeout | `time.Duration` | NegationPeekTimeout is deprecated and ignored. Negation now uses a timing-free, race-free event sequence + handshake, so no peek window is needed. This field is retained for backward compatibility and will be removed in a future major version. |

### Constructor Functions

### DefaultSLGConfig

DefaultSLGConfig returns the default SLG configuration.

```go
func DefaultSLGConfig() *SLGConfig
```

**Parameters:**
  None

**Returns:**
- *SLGConfig

### SLGEngine
SLGEngine coordinates tabled goal evaluation using SLG resolution. The engine maintains a global SubgoalTable shared across all evaluations, enabling answer reuse and cycle detection. Multiple goroutines can safely evaluate different goals concurrently. Thread safety: SLGEngine is safe for concurrent use by multiple goroutines.

#### Example Usage

```go
// Create a new SLGEngine
slgengine := SLGEngine{
    subgoals: &SubgoalTable{}{},
    config: &SLGConfig{}{},
    totalEvaluations: /* value */,
    totalAnswers: /* value */,
    cacheHits: /* value */,
    cacheMisses: /* value */,
    mu: /* value */,
    strataMu: /* value */,
    strata: map[],
    reverseDeps: /* value */,
    depMu: /* value */,
    depAdj: map[],
    negMu: /* value */,
    negUndefined: map[],
    predicateMu: /* value */,
    predicateEntries: map[],
}
```

#### Type Definition

```go
type SLGEngine struct {
    subgoals *SubgoalTable
    config *SLGConfig
    totalEvaluations atomic.Int64
    totalAnswers atomic.Int64
    cacheHits atomic.Int64
    cacheMisses atomic.Int64
    mu sync.RWMutex
    strataMu sync.RWMutex
    strata map[string]int
    reverseDeps sync.Map
    depMu sync.RWMutex
    depAdj map[uint64]map[uint64]*edgePolarity
    negMu sync.RWMutex
    negUndefined map[uint64]bool
    predicateMu sync.RWMutex
    predicateEntries map[string]map[uint64]*ast.StructType
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| subgoals | `*SubgoalTable` | Global subgoal table (shared across all evaluations) |
| config | `*SLGConfig` | Configuration |
| totalEvaluations | `atomic.Int64` | Statistics (atomic counters) |
| totalAnswers | `atomic.Int64` |  |
| cacheHits | `atomic.Int64` |  |
| cacheMisses | `atomic.Int64` |  |
| mu | `sync.RWMutex` | Mutex for engine-level operations |
| strataMu | `sync.RWMutex` | Optional stratification map: predicateID -> stratum (0 = base) |
| strata | `map[string]int` |  |
| reverseDeps | `sync.Map` | Reverse dependency index: child pattern hash -> set of parent entries |
| depMu | `sync.RWMutex` | Dependency graph (for unfounded set detection): parent -> child edges with polarity. |
| depAdj | `map[uint64]map[uint64]*edgePolarity` |  |
| negMu | `sync.RWMutex` | Cached set of nodes detected as part of SCCs containing a negative edge |
| negUndefined | `map[uint64]bool` |  |
| predicateMu | `sync.RWMutex` | Predicate tracking: maps predicateID to set of subgoal entry hashes |
| predicateEntries | `map[string]map[uint64]*ast.StructType` |  |

### Constructor Functions

### GlobalEngine

GlobalEngine returns the global SLG engine, creating it if necessary.

```go
func GlobalEngine() *SLGEngine
```

**Parameters:**
  None

**Returns:**
- *SLGEngine

### NewSLGEngine

NewSLGEngine creates a new SLG engine with the given configuration.

```go
func NewSLGEngine(config *SLGConfig) *SLGEngine
```

**Parameters:**
- `config` (*SLGConfig)

**Returns:**
- *SLGEngine

## Methods

### Clear

Clear removes all cached subgoals and resets statistics.

```go
func (*SubgoalTable) Clear()
```

**Parameters:**
  None

**Returns:**
  None

### ClearPredicate

ClearPredicate removes all cached subgoals for a specific predicate. This enables fine-grained invalidation when a relation's facts change. Returns the number of subgoal entries that were invalidated.

```go
func (*SLGEngine) ClearPredicate(predicateID string) int
```

**Parameters:**
- `predicateID` (string)

**Returns:**
- int

### ComputeFixpoint

ComputeFixpoint computes the least fixpoint for a strongly connected component. This is used when a cycle is detected in the dependency graph. The algorithm: 1. Iteratively re-evaluate all subgoals in the SCC 2. Check if new answers were derived 3. Repeat until no new answers (fixpoint reached) or max iterations exceeded Returns error if max iterations exceeded without convergence.

```go
func (*SLGEngine) ComputeFixpoint(ctx context.Context, scc *SCC) error
```

**Parameters:**
- `ctx` (context.Context)
- `scc` (*SCC)

**Returns:**
- error

### DetectCycles

DetectCycles finds strongly connected components in the dependency graph using Tarjan's algorithm. Returns all SCCs in reverse topological order.

```go
func (*SLGEngine) DetectCycles() []*SCC
```

**Parameters:**
  None

**Returns:**
- []*SCC

### Evaluate

Evaluate evaluates a tabled goal using SLG resolution. The process: 1. Normalize the call pattern 2. Check if subgoal exists in table (cache hit/miss) 3. If new: start producer to evaluate goal 4. If existing: consume from answer trie 5. Handle cycles via dependency tracking Returns a channel that yields answer bindings as they become available. The channel is closed when evaluation completes or context is cancelled. Thread safety: Safe for concurrent calls with different patterns.

```go
func (*SLGEngine) Evaluate(ctx context.Context, pattern *CallPattern, evaluator GoalEvaluator) (<-chan map[int64]Term, error)
```

**Parameters:**
- `ctx` (context.Context)
- `pattern` (*CallPattern)
- `evaluator` (GoalEvaluator)

**Returns:**
- <-chan map[int64]Term
- error

### InvalidateByDomain

InvalidateByDomain notifies the engine that the FD domain for a variable has changed. It retracts any tabled answers across all subgoals that bind varID to an integer value not contained in the provided domain. Returns the total number of answers retracted across all subgoals. Thread-safety: safe for concurrent use. Iterates over a snapshot of all entries and delegates to entry-level invalidation which handles its own synchronization.

```go
func (*SubgoalEntry) InvalidateByDomain(varID int64, dom Domain) int
```

**Parameters:**
- `varID` (int64)
- `dom` (Domain)

**Returns:**
- int

### IsCyclic

IsCyclic checks if the dependency graph contains any cycles.

```go
func (*SLGEngine) IsCyclic() bool
```

**Parameters:**
  None

**Returns:**
- bool

### NegationTruth

NegationTruth evaluates not(innerPattern) using the provided inner evaluator and reports the WFS truth value. It does not enumerate all answers; it only determines whether the negation holds (true), fails (false), or is currently undefined (conditional due to active dependencies). Contract: - Returns (TruthTrue, nil) if an unconditional empty binding is produced. - Returns (TruthFalse, nil) if no binding is produced because the inner has answers. - Returns (TruthUndefined, nil) if a conditional binding is produced (delayed). - Returns (TruthUndefined, ctx.Err()) if the context is canceled before a decision.

```go
func (*SLGEngine) NegationTruth(ctx context.Context, currentPredicateID string, innerPattern *CallPattern, innerEvaluator GoalEvaluator) (TruthValue, error)
```

**Parameters:**
- `ctx` (context.Context)
- `currentPredicateID` (string)
- `innerPattern` (*CallPattern)
- `innerEvaluator` (GoalEvaluator)

**Returns:**
- TruthValue
- error

### SetStrata

stratification accessors SetStrata sets fixed predicate strata for WFS enforcement where lower strata must not depend negatively on same-or-higher strata. Keys are predicate IDs as used by CallPattern.PredicateID(). Missing keys default to stratum 0.

```go
func (*SLGEngine) SetStrata(strata map[string]int)
```

**Parameters:**
- `strata` (map[string]int)

**Returns:**
  None

### Stats

Stats returns current engine statistics.

```go
func (*SLGEngine) Stats() *SLGStats
```

**Parameters:**
  None

**Returns:**
- *SLGStats

### Stratum

Stratum returns the configured stratum for a predicate, or 0 if unspecified.

```go
func (*SLGEngine) Stratum(predicateID string) int
```

**Parameters:**
- `predicateID` (string)

**Returns:**
- int

### addNegativeEdge

addNegativeEdge records a negative dependency parent->child for unfounded set analysis.

```go
func (*SLGEngine) addNegativeEdge(parent, child uint64)
```

**Parameters:**
- `parent` (uint64)
- `child` (uint64)

**Returns:**
  None

### addPositiveEdge

addPositiveEdge records a positive dependency parent->child for unfounded set analysis.

```go
func (*SLGEngine) addPositiveEdge(parent, child uint64)
```

**Parameters:**
- `parent` (uint64)
- `child` (uint64)

**Returns:**
  None

### addReverseDependency



```go
func (*SLGEngine) addReverseDependency(child uint64, parent *SubgoalEntry)
```

**Parameters:**
- `child` (uint64)
- `parent` (*SubgoalEntry)

**Returns:**
  None

### computeUndefinedSCCs

computeUndefinedSCCs runs Tarjan's SCC and marks subgoals in SCCs containing at least one negative edge as WFS undefined.

```go
func (*SLGEngine) computeUndefinedSCCs()
```

**Parameters:**
  None

**Returns:**
  None

### consumeAnswers

consumeAnswers creates a consumer for an existing subgoal's answers.

```go
func (*SLGEngine) consumeAnswers(ctx context.Context, entry *SubgoalEntry) <-chan map[int64]Term
```

**Parameters:**
- `ctx` (context.Context)
- `entry` (*SubgoalEntry)

**Returns:**
- <-chan map[int64]Term

### evaluateWithHandshake

evaluateWithHandshake evaluates a subgoal and returns: - the answer channel - the subgoal entry - the pre-start event sequence (captured before starting a new producer) - the producer started channel This enables race-free initial-shape decisions without timers.

```go
func (*SLGEngine) evaluateWithHandshake(ctx context.Context, pattern *CallPattern, evaluator GoalEvaluator) (<-chan map[int64]Term, *SubgoalEntry, uint64, <-chan *ast.StructType, error)
```

**Parameters:**
- `ctx` (context.Context)
- `pattern` (*CallPattern)
- `evaluator` (GoalEvaluator)

**Returns:**
- <-chan map[int64]Term
- *SubgoalEntry
- uint64
- <-chan *ast.StructType
- error

### getReverseParents



```go
func (*SLGEngine) getReverseParents(child uint64) []*SubgoalEntry
```

**Parameters:**
- `child` (uint64)

**Returns:**
- []*SubgoalEntry

### hasNegEdgeReachableFrom

hasNegEdgeReachableFrom reports whether starting from 'hash' and following only positive edges, we can reach any negative edge (u -neg-> v). This conservatively detects potential unfounded-set cycles involving negation reachable from the inner goal.

```go
func (*SLGEngine) hasNegEdgeReachableFrom(hash uint64) bool
```

**Parameters:**
- `hash` (uint64)

**Returns:**
- bool

### hasNegativeIncoming

hasNegativeIncoming reports whether any parent has a negative edge to this node. This is a conservative heuristic useful when SCC computation hasn't yet converged.

```go
func (*SLGEngine) hasNegativeIncoming(hash uint64) bool
```

**Parameters:**
- `hash` (uint64)

**Returns:**
- bool

### isInNegativeSCC

isInNegativeSCC reports whether the given subgoal hash is currently known to be in an SCC that contains at least one negative edge.

```go
func (*SLGEngine) isInNegativeSCC(hash uint64) bool
```

**Parameters:**
- `hash` (uint64)

**Returns:**
- bool

### onChildCompletedNoAnswers

onChildCompletedNoAnswers simplifies delay sets in dependent parents.

```go
func (*SLGEngine) onChildCompletedNoAnswers(child *SubgoalEntry)
```

**Parameters:**
- `child` (*SubgoalEntry)

**Returns:**
  None

### onChildHasAnswers

onChildHasAnswers is invoked when a child subgoal derives its first answer. It retracts all conditional answers in parents that depend on this child.

```go
func (*SLGEngine) onChildHasAnswers(child *SubgoalEntry)
```

**Parameters:**
- `child` (*SubgoalEntry)

**Returns:**
  None

### produceAndConsume

produceAndConsume starts producer and consumer goroutines for a new subgoal. Producer: evaluates goal and inserts answers into trie Consumer: reads from trie and streams to output channel

```go
func (*SLGEngine) produceAndConsume(ctx context.Context, entry *SubgoalEntry, evaluator GoalEvaluator) <-chan map[int64]Term
```

**Parameters:**
- `ctx` (context.Context)
- `entry` (*SubgoalEntry)
- `evaluator` (GoalEvaluator)

**Returns:**
- <-chan map[int64]Term

### registerPredicate

registerPredicate registers a subgoal entry with the predicate tracking system. This should be called when a new entry is created.

```go
func (*SLGEngine) registerPredicate(pattern *CallPattern, hash uint64)
```

**Parameters:**
- `pattern` (*CallPattern)
- `hash` (uint64)

**Returns:**
  None

### removeReverseDependency



```go
func (*SLGEngine) removeReverseDependency(child uint64, parent *SubgoalEntry)
```

**Parameters:**
- `child` (uint64)
- `parent` (*SubgoalEntry)

**Returns:**
  None

### SLGStats
SLGStats provides statistics about engine performance.

#### Example Usage

```go
// Create a new SLGStats
slgstats := SLGStats{
    TotalEvaluations: 42,
    TotalAnswers: 42,
    CacheHits: 42,
    CacheMisses: 42,
    CachedSubgoals: 42,
    HitRatio: 3.14,
}
```

#### Type Definition

```go
type SLGStats struct {
    TotalEvaluations int64
    TotalAnswers int64
    CacheHits int64
    CacheMisses int64
    CachedSubgoals int64
    HitRatio float64
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| TotalEvaluations | `int64` | Total number of subgoals evaluated |
| TotalAnswers | `int64` | Total answers derived across all subgoals |
| CacheHits | `int64` | Number of cache hits (subgoal already evaluated) |
| CacheMisses | `int64` | Number of cache misses (new subgoal evaluation) |
| CachedSubgoals | `int64` | Current number of cached subgoals |
| HitRatio | `float64` | Cache hit ratio (hits / (hits + misses)) |

### Scale
- Backward propagation: x ⊆ {result / multiplier | result ∈ result.domain, result % multiplier == 0} This is arc-consistent propagation suitable for AC-3 and fixed-point iteration. Invariants: - multiplier > 0 (enforced at construction) - All variables must have non-nil domains with positive integer values - Empty domain → immediate failure Thread Safety: Immutable after construction. Propagate() is safe for concurrent use.

#### Example Usage

```go
// Create a new Scale
scale := Scale{
    x: &FDVariable{}{},
    multiplier: 42,
    result: &FDVariable{}{},
}
```

#### Type Definition

```go
type Scale struct {
    x *FDVariable
    multiplier int
    result *FDVariable
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| x | `*FDVariable` | The value being scaled |
| multiplier | `int` | The constant multiplier (must be > 0) |
| result | `*FDVariable` | The result of scaling (x * multiplier) |

### Constructor Functions

### NewScale

NewScale creates a new scaling constraint: result = x * multiplier.

```go
func NewScale(x *FDVariable, multiplier int, result *FDVariable) (*Scale, error)
```

**Parameters:**

- `x` (*FDVariable) - The FD variable representing the input value

- `multiplier` (int) - The constant integer multiplier (must be > 0)

- `result` (*FDVariable) - The FD variable representing the scaled result

**Returns:**
- *Scale
- error

## Methods

### Clone

Clone creates a copy of the constraint with the same multiplier. The variable references are shared (constraints are immutable).

```go
func (*Modulo) Clone() PropagationConstraint
```

**Parameters:**
  None

**Returns:**
- PropagationConstraint

### Propagate

Propagate applies bidirectional arc-consistency. Performs bidirectional arc-consistent propagation: 1. Forward: Prune result based on possible x * multiplier values 2. Backward: Prune x based on possible result / multiplier values (where result % multiplier == 0) 3. Detect conflicts: Empty domain after propagation → failure

```go
func (*Lexicographic) Propagate(solver *Solver, state *SolverState) (*SolverState, error)
```

**Parameters:**
- `solver` (*Solver)
- `state` (*SolverState)

**Returns:**
- *SolverState
- error

### String

String returns a human-readable representation of the constraint. Useful for debugging and logging. Implements ModelConstraint.

```go
func (*Regular) String() string
```

**Parameters:**
  None

**Returns:**
- string

### Type

Type returns the constraint type identifier. Implements ModelConstraint.

```go
func (*Regular) Type() string
```

**Parameters:**
  None

**Returns:**
- string

### Variables

Variables returns the variables involved in this constraint. Used for dependency tracking and constraint graph construction. Implements ModelConstraint.

```go
func (*RationalLinearSum) Variables() []*FDVariable
```

**Parameters:**
  None

**Returns:**
- []*FDVariable

### backwardPropagate

backwardPropagate prunes the x domain based on result values. For each value r in result.domain: - If r % multiplier == 0, compute x = r / multiplier - Keep x in x.domain if already present - Remove from x.domain if no result value can be produced by it Returns a new domain with only feasible x values.

```go
func (*Absolute) backwardPropagate(absValueDomain, xDomain Domain) Domain
```

**Parameters:**
- `absValueDomain` (Domain)
- `xDomain` (Domain)

**Returns:**
- Domain

### forwardPropagate

forwardPropagate prunes the result domain based on x values. For each value v in x.domain: - Compute r = v * multiplier - Keep r in result.domain if already present - Remove from result.domain if no x value can produce it Returns a new domain with only feasible result values.

```go
func (*Modulo) forwardPropagate(xDomain, remainderDomain Domain) Domain
```

**Parameters:**
- `xDomain` (Domain)
- `remainderDomain` (Domain)

**Returns:**
- Domain

### ScaledDivision
- Backward propagation: dividend ⊆ {q*divisor...(q+1)*divisor-1 | q ∈ quotient.domain} This is arc-consistent propagation suitable for AC-3 and fixed-point iteration. Invariants: - divisor > 0 (enforced at construction) - All variables must have non-nil domains - Empty domain → immediate failure Thread Safety: Immutable after construction. Propagate() is safe for concurrent use.

#### Example Usage

```go
// Create a new ScaledDivision
scaleddivision := ScaledDivision{
    dividend: &FDVariable{}{},
    divisor: 42,
    quotient: &FDVariable{}{},
}
```

#### Type Definition

```go
type ScaledDivision struct {
    dividend *FDVariable
    divisor int
    quotient *FDVariable
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| dividend | `*FDVariable` | The value being divided |
| divisor | `int` | The constant divisor (must be > 0) |
| quotient | `*FDVariable` | The result of division |

### Constructor Functions

### NewScaledDivision

NewScaledDivision creates a new scaled division constraint.

```go
func NewScaledDivision(dividend *FDVariable, divisor int, quotient *FDVariable) (*ScaledDivision, error)
```

**Parameters:**

- `dividend` (*FDVariable) - The FD variable representing the numerator

- `divisor` (int) - The constant integer divisor (must be > 0)

- `quotient` (*FDVariable) - The FD variable representing the result

**Returns:**
- *ScaledDivision
- error

## Methods

### Clone

Clone creates a copy of the constraint with the same divisor. The variables references are shared (constraints are immutable).

```go
func (*AlphaEqConstraint) Clone() Constraint
```

**Parameters:**
  None

**Returns:**
- Constraint

### Propagate

Propagate implements the PropagationConstraint interface. Performs bidirectional arc-consistent propagation: 1. Forward: Prune quotient based on possible dividend/divisor values 2. Backward: Prune dividend based on possible quotient*divisor ranges 3. Detect conflicts: Empty domain after propagation → failure

```go
func (*EqualityReified) Propagate(solver *Solver, state *SolverState) (*SolverState, error)
```

**Parameters:**
- `solver` (*Solver)
- `state` (*SolverState)

**Returns:**
- *SolverState
- error

### String

String returns a human-readable representation of the constraint. Useful for debugging and logging. Implements ModelConstraint.

```go
func (*GlobalCardinality) String() string
```

**Parameters:**
  None

**Returns:**
- string

### Type

Type returns the constraint type identifier. Implements ModelConstraint.

```go
func (*GlobalCardinality) Type() string
```

**Parameters:**
  None

**Returns:**
- string

### Variables

Vars returns the variables involved in this constraint. Used for dependency tracking and constraint graph construction. Implements ModelConstraint.

```go
func (*Modulo) Variables() []*FDVariable
```

**Parameters:**
  None

**Returns:**
- []*FDVariable

### backwardPropagate

backwardPropagate prunes the dividend domain based on quotient values. For each value q in quotient.domain: - Compute range [q*divisor, (q+1)*divisor - 1] - Keep dividend values in this range - Remove dividend values outside all ranges Returns a new domain with only feasible dividend values.

```go
func (*Scale) backwardPropagate(resultDomain, xDomain Domain) Domain
```

**Parameters:**
- `resultDomain` (Domain)
- `xDomain` (Domain)

**Returns:**
- Domain

### forwardPropagate

forwardPropagate prunes the quotient domain based on dividend values. For each value d in dividend.domain: - Compute q = ⌊d/divisor⌋ - Keep q in quotient.domain if already present - Remove from quotient.domain if no dividend value can produce it Returns a new domain with only feasible quotient values.

```go
func (*Modulo) forwardPropagate(xDomain, remainderDomain Domain) Domain
```

**Parameters:**
- `xDomain` (Domain)
- `remainderDomain` (Domain)

**Returns:**
- Domain

### Sequence
_No documentation available_

#### Example Usage

```go
// Create a new Sequence
sequence := Sequence{
    vars: [],
    set: [],
    k: 42,
    minCount: 42,
    maxCount: 42,
    b: [],
    reifs: [],
    windows: [],
}
```

#### Type Definition

```go
type Sequence struct {
    vars []*FDVariable
    set []int
    k int
    minCount int
    maxCount int
    b []*FDVariable
    reifs []PropagationConstraint
    windows []PropagationConstraint
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| vars | `[]*FDVariable` |  |
| set | `[]int` |  |
| k | `int` |  |
| minCount | `int` |  |
| maxCount | `int` |  |
| b | `[]*FDVariable` |  |
| reifs | `[]PropagationConstraint` |  |
| windows | `[]PropagationConstraint` |  |

### Constructor Functions

### NewSequence

NewSequence constructs the Sequence constraint.

```go
func NewSequence(model *Model, vars []*FDVariable, setValues []int, windowLen, minCount, maxCount int) (*Sequence, error)
```

**Parameters:**
- `model` (*Model)
- `vars` ([]*FDVariable)
- `setValues` ([]int)
- `windowLen` (int)
- `minCount` (int)
- `maxCount` (int)

**Returns:**
- *Sequence
- error

## Methods

### Propagate



```go
func (*Absolute) Propagate(solver *Solver, state *SolverState) (*SolverState, error)
```

**Parameters:**
- `solver` (*Solver)
- `state` (*SolverState)

**Returns:**
- *SolverState
- error

### String



```go
func (*Inequality) String() string
```

**Parameters:**
  None

**Returns:**
- string

### Type



```go
func (*BinPacking) Type() string
```

**Parameters:**
  None

**Returns:**
- string

### Variables



```go
func (*Diffn) Variables() []*FDVariable
```

**Parameters:**
  None

**Returns:**
- []*FDVariable

### Solver
- Smart backtracking with conflict-driven learning (future) The solver is designed for both sequential and parallel execution. State is immutable during search, with modifications creating lightweight derived states that share structure with their parent. Thread safety: Solver instances are NOT thread-safe. For parallel search, create multiple Solver instances that share the same immutable Model but maintain independent SolverState chains. This is zero-cost as the Model is read-only and domains are immutable.

#### Example Usage

```go
// Create a new Solver
solver := Solver{
    model: &Model{}{},
    config: &SolverConfig{}{},
    statePool: &/* value */{},
    monitor: &SolverMonitor{}{},
    baseState: &SolverState{}{},
    optContext: &optimizationContext{}{},
}
```

#### Type Definition

```go
type Solver struct {
    model *Model
    config *SolverConfig
    statePool *sync.Pool
    monitor *SolverMonitor
    baseState *SolverState
    optContext *optimizationContext
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| model | `*Model` | model is the CSP being solved (read-only during search, shared by all workers) |
| config | `*SolverConfig` | config holds solver configuration and heuristics |
| statePool | `*sync.Pool` | statePool manages allocation of solver states for reuse |
| monitor | `*SolverMonitor` | monitor tracks solving statistics (optional) |
| baseState | `*SolverState` | baseState caches the last root-level propagated state from Solve. When present, GetDomain(nil, varID) will read domains from this state rather than the model's initial domains, allowing callers to inspect propagation effects without threading SolverState explicitly. |
| optContext | `*optimizationContext` | optContext holds optimization-specific state during SolveOptimal calls |

### Constructor Functions

### NewSolver

NewSolver creates a solver for the given model. The model should be fully constructed before creating the solver.

```go
func NewSolver(model *Model) *Solver
```

**Parameters:**
- `model` (*Model)

**Returns:**
- *Solver

### NewSolverWithConfig

NewSolverWithConfig creates a solver with custom configuration that overrides model config.

```go
func NewSolverWithConfig(model *Model, config *SolverConfig) *Solver
```

**Parameters:**
- `model` (*Model)
- `config` (*SolverConfig)

**Returns:**
- *Solver

## Methods

### BestObjectiveValue

BestObjectiveValue computes a trivial admissible bound for the objective in the current state. It is a helper primarily for testing and documentation.

```go
func (*Solver) BestObjectiveValue(state *SolverState, obj *FDVariable, minimize bool) (int, bool)
```

**Parameters:**
- `state` (*SolverState)
- `obj` (*FDVariable)
- `minimize` (bool)

**Returns:**
- int
- bool

### GetDomain

GetDomain returns the current domain of a variable in the given state. Walks the state chain to find the most recent domain for the variable. This is O(depth) in the worst case, but typically O(1) due to locality.

```go
func (*Solver) GetDomain(state *SolverState, varID int) Domain
```

**Parameters:**
- `state` (*SolverState)
- `varID` (int)

**Returns:**
- Domain

### Model

Model returns the model being solved. The model is read-only during solving and safe for concurrent access by multiple solver instances.

```go
func (*Solver) Model() *Model
```

**Parameters:**
  None

**Returns:**
- *Model

### ReleaseState

ReleaseState returns a state to the pool for reuse. Should be called when backtracking to free memory. Only the state itself is pooled, not domains (they are immutable and shared).

```go
func (*Solver) ReleaseState(state *SolverState)
```

**Parameters:**
- `state` (*SolverState)

**Returns:**
  None

### SetDomain

SetDomain creates a new state with an updated domain for the specified variable. Returns the new state and a boolean indicating if the domain actually changed. If the domain is identical to the current domain, returns the original state and false to avoid unnecessary propagation. This is an O(1) operation for the state update, plus O(domain size) for equality check. The returned state should replace the current state in the search.

```go
func (*FDVariable) SetDomain(domain Domain)
```

**Parameters:**
- `domain` (Domain)

**Returns:**
  None

### SetMonitor

SetMonitor enables statistics collection during solving.

```go
func (*Solver) SetMonitor(monitor *SolverMonitor)
```

**Parameters:**
- `monitor` (*SolverMonitor)

**Returns:**
  None

### Solve

Solve finds solutions to the constraint satisfaction problem. Returns up to maxSolutions solutions, or all solutions if maxSolutions <= 0. Solutions are returned as slices of integers, one per variable in order. The search can be cancelled via the context, enabling timeouts and cancellation.

```go
func Solve(m *Model, maxSolutions int) ([][]int, error)
```

**Parameters:**
- `m` (*Model)
- `maxSolutions` (int)

**Returns:**
- [][]int
- error

### SolveOptimal

SolveOptimal finds a solution that optimizes the given objective variable. Contract: - obj is an FD variable participating in the model. Its domain encodes the objective value (smaller is better when minimize=true). - minimize selects the direction (true: minimize, false: maximize). - On success, returns the best solution found (values for all model variables in model order) and the objective value. If the model is infeasible, returns (nil, 0, nil). If ctx is cancelled, returns the best incumbent if any together with ctx.Err(). Implementation notes: - This is a native branch-and-bound layered on the existing FD solver. It reuses propagation and branching; adds a fast admissible bound check and an incumbent cutoff applied as a dynamic constraint on the objective domain. - Lower bound (LB) for minimize is obj.Min() from the current state; for maximize, the symmetric upper bound is used. - Incumbent cutoff is injected by tightening the objective domain at nodes: minimize: obj ≤ (best-1)  via RemoveAtOrAbove(best) maximize: obj ≥ (best+1)  via RemoveAtOrBelow(best)

```go
func (*Solver) SolveOptimal(ctx context.Context, obj *FDVariable, minimize bool) ([]int, int, error)
```

**Parameters:**
- `ctx` (context.Context)
- `obj` (*FDVariable)

- `minimize` (bool) - obj ≤ (best-1)  via RemoveAtOrAbove(best)

**Returns:**
- []int
- int
- error

### SolveOptimalWithOptions

SolveOptimalWithOptions is like SolveOptimal but supports options (time/node limits, target objective, heuristic overrides, and parallel workers).

```go
func (*Solver) SolveOptimalWithOptions(ctx context.Context, obj *FDVariable, minimize bool, opts ...OptimizeOption) ([]int, int, error)
```

**Parameters:**
- `ctx` (context.Context)
- `obj` (*FDVariable)
- `minimize` (bool)
- `opts` (...OptimizeOption)

**Returns:**
- []int
- int
- error

### SolveParallel

SolveParallel performs parallel backtracking search to find solutions. Uses multiple workers sharing a work queue via a buffered channel.

```go
func (*Solver) SolveParallel(ctx context.Context, numWorkers, maxSolutions int) ([][]int, error)
```

**Parameters:**

- `ctx` (context.Context) - Context for cancellation

- `numWorkers` (int) - Number of parallel workers (0 = runtime.NumCPU())

- `maxSolutions` (int) - Maximum solutions to find (0 = find all)

**Returns:**
- [][]int
- error

### computeObjectiveBound

computeObjectiveBound computes a safe admissible bound for the objective based on the current state and known structural constraints. It falls back to the objective variable's domain when no better structural bound is available.

```go
func (*Solver) computeObjectiveBound(state *SolverState, obj *FDVariable, minimize bool) (int, bool)
```

**Parameters:**
- `state` (*SolverState)
- `obj` (*FDVariable)
- `minimize` (bool)

**Returns:**
- int
- bool

### computeVariableDegree

computeVariableDegree returns the number of constraints involving the variable.

```go
func (*Solver) computeVariableDegree(varID int) int
```

**Parameters:**
- `varID` (int)

**Returns:**
- int

### computeVariableScore

computeVariableScore computes a score for variable selection heuristics. Lower scores are better (selected first).

```go
func (*Solver) computeVariableScore(varID int, domain Domain) float64
```

**Parameters:**
- `varID` (int)
- `domain` (Domain)

**Returns:**
- float64

### extractSolution

extractSolution extracts the variable assignments from a complete state.

```go
func (*Solver) extractSolution(state *SolverState) []int
```

**Parameters:**
- `state` (*SolverState)

**Returns:**
- []int

### isComplete

isComplete returns true if all variables are bound (singleton domains).

```go
func (*Solver) isComplete(state *SolverState) bool
```

**Parameters:**
- `state` (*SolverState)

**Returns:**
- bool

### orderValues

orderValues orders domain values according to the configured heuristic.

```go
func (*Solver) orderValues(values []int) []int
```

**Parameters:**
- `values` ([]int)

**Returns:**
- []int

### parallelWorker

parallelWorker processes work items from the shared work channel.

```go
func (*Solver) parallelWorker(ctx context.Context, cancel context.CancelFunc, workerID int, workChan chan *workItem, solutionChan chan []int, tasksWG *sync.WaitGroup, solutionsFound *atomic.Int64, maxSolutions int)
```

**Parameters:**
- `ctx` (context.Context)
- `cancel` (context.CancelFunc)
- `workerID` (int)
- `workChan` (chan *workItem)
- `solutionChan` (chan []int)
- `tasksWG` (*sync.WaitGroup)
- `solutionsFound` (*atomic.Int64)
- `maxSolutions` (int)

**Returns:**
  None

### processWork

processWork processes a single work item, trying all values for the variable. Does NOT release work.state - caller is responsible.

```go
func (*Solver) processWork(ctx context.Context, work *workItem, workChan chan *workItem, solutionChan chan []int, solutionsFound *atomic.Int64, tasksWG *sync.WaitGroup, maxSolutions int)
```

**Parameters:**
- `ctx` (context.Context)
- `work` (*workItem)
- `workChan` (chan *workItem)
- `solutionChan` (chan []int)
- `solutionsFound` (*atomic.Int64)
- `tasksWG` (*sync.WaitGroup)
- `maxSolutions` (int)

**Returns:**
  None

### propagate

propagate runs all propagation constraints to a fixed-point. Returns a new state with pruned domains, or error if inconsistency detected. The propagation loop: 1. Collect all PropagationConstraints from the model 2. Run each constraint once 3. If any constraint modified domains, repeat from step 2 4. Stop when no changes occur (fixed-point reached) This maintains copy-on-write semantics: each constraint returns a new state, preserving the lock-free property for parallel search.

```go
func (*Solver) propagate(state *SolverState) (*SolverState, error)
```

**Parameters:**
- `state` (*SolverState)

**Returns:**
- *SolverState
- error

### search

search performs iterative deepening backtracking search. Uses an explicit stack to avoid deep recursion and enable better control.

```go
func (*Solver) search(ctx context.Context, state *SolverState, solutions *[][]int, maxSolutions int)
```

**Parameters:**
- `ctx` (context.Context)
- `state` (*SolverState)
- `solutions` (*[][]int)
- `maxSolutions` (int)

**Returns:**
  None

### selectVariable

selectVariable chooses the next variable to branch on using the configured heuristic. Returns the variable ID and the ordered list of values to try. Returns (-1, nil) if all variables are bound.

```go
func (*Solver) selectVariable(state *SolverState) (int, []int)
```

**Parameters:**
- `state` (*SolverState)

**Returns:**
- int
- []int

### solveOptimalParallel

solveOptimalParallel runs branch-and-bound optimization using the shared work-queue parallel search infrastructure. It shares the incumbent objective across workers via atomics and applies dynamic objective cutoffs at each node to prune subtrees.

```go
func (*Solver) solveOptimalParallel(ctx context.Context, obj *FDVariable, minimize bool, cfg *optConfig) ([]int, int, error)
```

**Parameters:**
- `ctx` (context.Context)
- `obj` (*FDVariable)
- `minimize` (bool)
- `cfg` (*optConfig)

**Returns:**
- []int
- int
- error

### SolverConfig
SolverConfig holds configuration for the FD solver

#### Example Usage

```go
// Create a new SolverConfig
solverconfig := SolverConfig{
    VariableHeuristic: VariableOrderingHeuristic{},
    ValueHeuristic: ValueOrderingHeuristic{},
    RandomSeed: 42,
}
```

#### Type Definition

```go
type SolverConfig struct {
    VariableHeuristic VariableOrderingHeuristic
    ValueHeuristic ValueOrderingHeuristic
    RandomSeed int64
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| VariableHeuristic | `VariableOrderingHeuristic` |  |
| ValueHeuristic | `ValueOrderingHeuristic` |  |
| RandomSeed | `int64` | for reproducible random heuristics |

### Constructor Functions

### DefaultSolverConfig

DefaultSolverConfig returns a default solver configuration

```go
func DefaultSolverConfig() *SolverConfig
```

**Parameters:**
  None

**Returns:**
- *SolverConfig

### SolverMonitor
SolverMonitor provides lock-free monitoring capabilities for the FD solver. All operations use atomic instructions for safe concurrent access without locks. Designed to match the lock-free copy-on-write architecture of the solver.

#### Example Usage

```go
// Create a new SolverMonitor
solvermonitor := SolverMonitor{
    stats: SolverStats{},
    startTime: /* value */,
    propStart: /* value */,
}
```

#### Type Definition

```go
type SolverMonitor struct {
    stats SolverStats
    startTime time.Time
    propStart atomic.Int64
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| stats | `SolverStats` |  |
| startTime | `time.Time` |  |
| propStart | `atomic.Int64` | Propagation start time in nanoseconds (0 = not started) |

### Constructor Functions

### NewSolverMonitor

NewSolverMonitor creates a new solver monitor. Uses atomic operations for lock-free statistics collection.

```go
func NewSolverMonitor() *SolverMonitor
```

**Parameters:**
  None

**Returns:**
- *SolverMonitor

## Methods

### CaptureFinalDomains

CaptureFinalDomains captures the final domain state and computes reductions. Safe to call on nil monitor. Phase 2 implementation - currently a no-op.

```go
func (*SolverMonitor) CaptureFinalDomains(store *FDStore)
```

**Parameters:**
- `store` (*FDStore)

**Returns:**
  None

### CaptureInitialDomains

CaptureInitialDomains captures the initial domain state. Safe to call on nil monitor. Phase 2 implementation - currently a no-op.

```go
func (*SolverMonitor) CaptureInitialDomains(store *FDStore)
```

**Parameters:**
- `store` (*FDStore)

**Returns:**
  None

### EndPropagation

EndPropagation marks the end of a propagation operation. Safe to call on nil monitor. Lock-free.

```go
func (*SolverMonitor) EndPropagation()
```

**Parameters:**
  None

**Returns:**
  None

### FinishSearch

FinishSearch marks the end of the search process. Safe to call on nil monitor. Only called once at end, no synchronization needed.

```go
func (*SolverMonitor) FinishSearch()
```

**Parameters:**
  None

**Returns:**
  None

### GetStats

GetStats returns a snapshot of the current statistics. Returns nil if the monitor is nil. Safe to call concurrently from multiple goroutines.

```go
func (*FDStore) GetStats() *SolverStats
```

**Parameters:**
  None

**Returns:**
- *SolverStats

### RecordBacktrack

RecordBacktrack records a backtrack operation. Safe to call on nil monitor. Lock-free.

```go
func (*SolverMonitor) RecordBacktrack()
```

**Parameters:**
  None

**Returns:**
  None

### RecordConstraint

RecordConstraint records adding a constraint. Safe to call on nil monitor. Lock-free.

```go
func (*SolverMonitor) RecordConstraint()
```

**Parameters:**
  None

**Returns:**
  None

### RecordDepth

RecordDepth records the current search depth. Safe to call on nil monitor. Lock-free using compare-and-swap.

```go
func (*SolverMonitor) RecordDepth(depth int)
```

**Parameters:**
- `depth` (int)

**Returns:**
  None

### RecordNode

RecordNode records exploring a search node. Safe to call on nil monitor. Lock-free.

```go
func (*SolverMonitor) RecordNode()
```

**Parameters:**
  None

**Returns:**
  None

### RecordQueueSize

RecordQueueSize records the current queue size. Safe to call on nil monitor. Lock-free using compare-and-swap.

```go
func (*SolverMonitor) RecordQueueSize(size int)
```

**Parameters:**
- `size` (int)

**Returns:**
  None

### RecordSolution

RecordSolution records finding a solution. Safe to call on nil monitor. Lock-free.

```go
func (*SolverMonitor) RecordSolution()
```

**Parameters:**
  None

**Returns:**
  None

### RecordTrailSize

RecordTrailSize records the current trail size. Safe to call on nil monitor. Lock-free using compare-and-swap.

```go
func (*SolverMonitor) RecordTrailSize(size int)
```

**Parameters:**
- `size` (int)

**Returns:**
  None

### StartPropagation

StartPropagation marks the beginning of a propagation operation. Safe to call on nil monitor. Lock-free.

```go
func (*SolverMonitor) StartPropagation()
```

**Parameters:**
  None

**Returns:**
  None

### SolverPlugin
UnifiedStore containing both relational bindings and FD domains. Each plugin is responsible for: - Identifying which constraints it can handle - Propagating those constraints to prune the search space - Communicating changes through the UnifiedStore Plugins must be thread-safe as they may be called concurrently during parallel search. They must also maintain the copy-on-write semantics required for lock-free operation: all state changes return new store versions.

#### Example Usage

```go
// Example implementation of SolverPlugin
type MySolverPlugin struct {
    // Add your fields here
}

func (m MySolverPlugin) Name() string {
    // Implement your logic here
    return
}

func (m MySolverPlugin) CanHandle(param1 interface{}) bool {
    // Implement your logic here
    return
}

func (m MySolverPlugin) Propagate(param1 *UnifiedStore) *UnifiedStore {
    // Implement your logic here
    return
}


```

#### Type Definition

```go
type SolverPlugin interface {
    Name() string
    CanHandle(constraint interface{}) bool
    Propagate(store *UnifiedStore) (*UnifiedStore, error)
}
```

## Methods

| Method | Description |
| ------ | ----------- |

### SolverState
1. Constraint sees x={5} via GetDomain(State3, x.ID) 2. Constraint narrows y: y={2,3} (remove 5) 3. Creates State4: y={2,3} (parent: State3) 4. Constraint narrows z: z={1,2,3} (5 not present, no change) 5. Returns State4 (fixed point reached) Constraints "communicate" by reading current domains via GetDomain and creating new states via SetDomain. The state chain captures all changes. States are pooled and reused to minimize GC pressure.

#### Example Usage

```go
// Create a new SolverState
solverstate := SolverState{
    parent: &SolverState{}{},
    modifiedVarID: 42,
    modifiedDomain: Domain{},
    depth: 42,
    refCount: /* value */,
}
```

#### Type Definition

```go
type SolverState struct {
    parent *SolverState
    modifiedVarID int
    modifiedDomain Domain
    depth int
    refCount atomic.Int64
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| parent | `*SolverState` | parent points to the previous state (nil for root) |
| modifiedVarID | `int` | modifiedVarID is the ID of the variable whose domain changed |
| modifiedDomain | `Domain` | modifiedDomain is the new domain for the modified variable |
| depth | `int` | depth tracks the depth in the search tree for heuristics |
| refCount | `atomic.Int64` | refCount tracks the number of active references to this state node. In sequential search this is typically 1 and ReleaseState will cascade, in parallel search multiple workers may hold references simultaneously. When the count drops to zero, the node can be safely returned to the pool. |

### SolverStats
SolverStats holds statistics about the FD solving process. All fields use atomic operations for lock-free updates.

#### Example Usage

```go
// Create a new SolverStats
solverstats := SolverStats{
    NodesExplored: 42,
    Backtracks: 42,
    SolutionsFound: 42,
    SearchTime: /* value */,
    MaxDepth: 42,
    PropagationCount: 42,
    PropagationTime: 42,
    ConstraintsAdded: 42,
    PeakTrailSize: 42,
    PeakQueueSize: 42,
}
```

#### Type Definition

```go
type SolverStats struct {
    NodesExplored int64
    Backtracks int64
    SolutionsFound int64
    SearchTime time.Duration
    MaxDepth int64
    PropagationCount int64
    PropagationTime int64
    ConstraintsAdded int64
    PeakTrailSize int64
    PeakQueueSize int64
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| NodesExplored | `int64` | Search statistics |
| Backtracks | `int64` | Number of backtracks performed |
| SolutionsFound | `int64` | Number of solutions found |
| SearchTime | `time.Duration` | Time spent in search |
| MaxDepth | `int64` | Maximum search depth reached |
| PropagationCount | `int64` | Propagation statistics |
| PropagationTime | `int64` | Time spent in propagation (nanoseconds) |
| ConstraintsAdded | `int64` | Number of constraints added |
| PeakTrailSize | `int64` | Memory statistics |
| PeakQueueSize | `int64` | Peak size of the propagation queue |

## Methods

### String



```go
func (*BoolSum) String() string
```

**Parameters:**
  None

**Returns:**
- string

### Stream
Stream represents a (potentially infinite) sequence of constraint stores. Streams are the core data structure for representing multiple solutions in miniKanren. Each constraint store contains variable bindings and active constraints representing a consistent logical state. This implementation uses channels for thread-safe concurrent access and supports parallel evaluation with proper constraint coordination.

#### Example Usage

```go
// Create a new Stream
stream := Stream{
    ch: /* value */,
    done: /* value */,
    mu: /* value */,
}
```

#### Type Definition

```go
type Stream struct {
    ch chan ConstraintStore
    done chan *ast.StructType
    mu sync.Mutex
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| ch | `chan ConstraintStore` | Channel for streaming constraint stores |
| done | `chan *ast.StructType` | Channel to signal completion |
| mu | `sync.Mutex` | Protects stream state |

### Constructor Functions

### NewStream

NewStream creates a new empty stream.

```go
func NewStream() *Stream
```

**Parameters:**
  None

**Returns:**
- *Stream

### conjHelper

conjHelper recursively evaluates conjunction goals

```go
func conjHelper(ctx context.Context, goals []Goal, store ConstraintStore) *Stream
```

**Parameters:**
- `ctx` (context.Context)
- `goals` ([]Goal)
- `store` (ConstraintStore)

**Returns:**
- *Stream

### streamFromAnswers

streamFromAnswers converts a channel of SLG answer substitutions into a miniKanren Stream. It unifies each answer with the original query pattern variables using Eq goals.

```go
func streamFromAnswers(ctx context.Context, store ConstraintStore, answers <-chan map[int64]Term, pattern []Term) *Stream
```

**Parameters:**
- `ctx` (context.Context)
- `store` (ConstraintStore)
- `answers` (<-chan map[int64]Term)
- `pattern` ([]Term)

**Returns:**
- *Stream

## Methods

### Close

Close closes the stream, indicating no more substitutions will be added.

```go
func (*Stream) Close()
```

**Parameters:**
  None

**Returns:**
  None

### Put



```go
func (*GlobalConstraintBusPool) Put(bus *GlobalConstraintBus)
```

**Parameters:**
- `bus` (*GlobalConstraintBus)

**Returns:**
  None

### Take

Take retrieves up to n constraint stores from the stream. Returns a slice of constraint stores and a boolean indicating if more stores might be available.

```go
func (*Stream) Take(n int) ([]ConstraintStore, bool)
```

**Parameters:**
- `n` (int)

**Returns:**
- []ConstraintStore
- bool

### Stretch
Stretch is a thin wrapper around the constructed Regular constraint to expose the high-level intent and variables involved.

#### Example Usage

```go
// Create a new Stretch
stretch := Stretch{
    vars: [],
    values: [],
    minByValue: map[],
    maxByValue: map[],
    dfa: &Regular{}{},
}
```

#### Type Definition

```go
type Stretch struct {
    vars []*FDVariable
    values []int
    minByValue map[int]int
    maxByValue map[int]int
    dfa *Regular
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| vars | `[]*FDVariable` |  |
| values | `[]int` | values explicitly parameterized |
| minByValue | `map[int]int` |  |
| maxByValue | `map[int]int` |  |
| dfa | `*Regular` | underlying DFA constraint |

### Constructor Functions

### NewStretch

NewStretch constructs Stretch(vars, values, minLen, maxLen).

```go
func NewStretch(model *Model, vars []*FDVariable, values []int, minLen []int, maxLen []int) (*Stretch, error)
```

**Parameters:**

- `model` (*Model) - hosting model (non-nil)

- `vars` ([]*FDVariable) - non-empty sequence of variables (positive domains)

- `values` ([]int) - distinct positive values to constrain explicitly
- `minLen` ([]int)

- `maxLen` ([]int) - same length as values; for each i, enforce

**Returns:**
- *Stretch
- error

## Methods

### Propagate

Propagate is a no-op for the wrapper; pruning is performed by the Regular DFA.

```go
func (*Lexicographic) Propagate(solver *Solver, state *SolverState) (*SolverState, error)
```

**Parameters:**
- `solver` (*Solver)
- `state` (*SolverState)

**Returns:**
- *SolverState
- error

### String

String returns a human-readable description.

```go
func (Rational) String() string
```

**Parameters:**
  None

**Returns:**
- string

### Type

Type returns the constraint type name.

```go
func (*Circuit) Type() string
```

**Parameters:**
  None

**Returns:**
- string

### Variables

Variables returns the sequence variables.

```go
func (*IntervalArithmetic) Variables() []*FDVariable
```

**Parameters:**
  None

**Returns:**
- []*FDVariable

### SubgoalEntry
SubgoalEntry represents a tabled subgoal with its cached answers. Thread safety: - Status is accessed atomically - Answer trie supports concurrent read/write - Dependencies protected by RWMutex - Condition variable for producer/consumer synchronization

#### Example Usage

```go
// Create a new SubgoalEntry
subgoalentry := SubgoalEntry{
    pattern: &CallPattern{}{},
    answers: &AnswerTrie{}{},
    evaluator: GoalEvaluator{},
    status: /* value */,
    dependencies: [],
    dependencyMu: /* value */,
    stratum: 42,
    answerCond: &/* value */{},
    answerMu: /* value */,
    consumptionCount: /* value */,
    derivationCount: /* value */,
    refCount: /* value */,
    answerMetadata: map[],
    metadataMu: /* value */,
    pendingDelaySet: DelaySet{},
    pendingMu: /* value */,
    eventMu: /* value */,
    eventCh: /* value */,
    changeSeq: /* value */,
    startMu: /* value */,
    startedCh: /* value */,
    startFired: true,
    retracted: map[],
    wfsTruth: /* value */,
}
```

#### Type Definition

```go
type SubgoalEntry struct {
    pattern *CallPattern
    answers *AnswerTrie
    evaluator GoalEvaluator
    status atomic.Int32
    dependencies []*SubgoalEntry
    dependencyMu sync.RWMutex
    stratum int
    answerCond *sync.Cond
    answerMu sync.Mutex
    consumptionCount atomic.Int64
    derivationCount atomic.Int64
    refCount atomic.Int64
    answerMetadata map[int]DelaySet
    metadataMu sync.RWMutex
    pendingDelaySet DelaySet
    pendingMu sync.Mutex
    eventMu sync.Mutex
    eventCh chan *ast.StructType
    changeSeq atomic.Uint64
    startMu sync.Mutex
    startedCh chan *ast.StructType
    startFired bool
    retracted map[int]*ast.StructType
    wfsTruth atomic.Int32
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| pattern | `*CallPattern` | Call pattern (immutable) |
| answers | `*AnswerTrie` | Answer trie containing all derived answers |
| evaluator | `GoalEvaluator` | Evaluator for re-evaluation during fixpoint computation Stored to enable re-derivation when dependencies change |
| status | `atomic.Int32` | Current evaluation status (atomic access) |
| dependencies | `[]*SubgoalEntry` | Dependencies for cycle detection and fixpoint computation |
| dependencyMu | `sync.RWMutex` |  |
| stratum | `int` | Stratification level for negation (0 = base stratum) |
| answerCond | `*sync.Cond` | Condition variable for answer availability signaling |
| answerMu | `sync.Mutex` |  |
| consumptionCount | `atomic.Int64` | Statistics (atomic counters) |
| derivationCount | `atomic.Int64` | Times new answers were added |
| refCount | `atomic.Int64` | Reference count for memory management |
| answerMetadata | `map[int]DelaySet` | WFS metadata: delay sets per answer index (conditional answers) Maps answer index (insertion order) to DelaySet Protected by metadataMu for thread-safe access |
| metadataMu | `sync.RWMutex` |  |
| pendingDelaySet | `DelaySet` | Pending metadata for the next answer to be inserted Evaluators queue metadata here before emitting answers |
| pendingMu | `sync.Mutex` |  |
| eventMu | `sync.Mutex` | Event channel: closed on any answer insertion or status change, then replaced with a new channel. Enables non-blocking observers to detect immediate changes without polling or sleeps. |
| eventCh | `chan *ast.StructType` |  |
| changeSeq | `atomic.Uint64` | Monotonic change sequence incremented on every signalEvent. Used with WaitChangeSince for race-free subscription. |
| startMu | `sync.Mutex` | Producer start signal: closed exactly once when the producer goroutine for this entry has started and is ready to emit answers or status changes. |
| startedCh | `chan *ast.StructType` |  |
| startFired | `bool` |  |
| retracted | `map[int]*ast.StructType` | Retracted answers (by index). Retracted answers are hidden from WFS-aware iterators but remain in the underlying trie to preserve insertion order and avoid structural mutation. |
| wfsTruth | `atomic.Int32` | WFS truth state for this subgoal (optional metadata). Defaults to TruthUndefined. This is separate from evaluation Status to avoid mixing lifecycle with semantics. |

### Constructor Functions

### NewSubgoalEntry

NewSubgoalEntry creates a new subgoal entry with the given call pattern.

```go
func NewSubgoalEntry(pattern *CallPattern) *SubgoalEntry
```

**Parameters:**
- `pattern` (*CallPattern)

**Returns:**
- *SubgoalEntry

## Methods

### AddDependency

AddDependency records that this subgoal depends on another.

```go
func (*SubgoalEntry) AddDependency(other *SubgoalEntry)
```

**Parameters:**
- `other` (*SubgoalEntry)

**Returns:**
  None

### AnswerRecords

AnswerRecords returns an iterator over AnswerRecord (bindings + delay sets). This is the WFS-aware iterator that provides metadata for conditional answers.

```go
func (*SubgoalEntry) AnswerRecords() *AnswerRecordIterator
```

**Parameters:**
  None

**Returns:**
- *AnswerRecordIterator

### AnswerRecordsFrom

AnswerRecordsFrom returns a WFS-aware iterator starting at the given index.

```go
func (*SubgoalEntry) AnswerRecordsFrom(start int) *AnswerRecordIterator
```

**Parameters:**
- `start` (int)

**Returns:**
- *AnswerRecordIterator

### Answers

Answers returns the answer trie.

```go
func (*SubgoalEntry) Answers() *AnswerTrie
```

**Parameters:**
  None

**Returns:**
- *AnswerTrie

### AttachDelaySet

AttachDelaySet associates a DelaySet with the answer at the given index. This marks the answer as conditional on the resolution of the dependencies in the delay set. Thread-safe for concurrent access. If ds is nil or empty, the answer remains unconditional.

```go
func (*SubgoalEntry) AttachDelaySet(answerIndex int, ds DelaySet)
```

**Parameters:**
- `answerIndex` (int)
- `ds` (DelaySet)

**Returns:**
  None

### ConsumptionCount

ConsumptionCount returns the number of times answers were consumed.

```go
func (*SubgoalEntry) ConsumptionCount() int64
```

**Parameters:**
  None

**Returns:**
- int64

### DelaySetFor

DelaySetFor retrieves the DelaySet for the answer at the given index. Returns nil if the answer is unconditional or the index is out of range. Thread-safe for concurrent access.

```go
func (*SubgoalEntry) DelaySetFor(answerIndex int) DelaySet
```

**Parameters:**
- `answerIndex` (int)

**Returns:**
- DelaySet

### Dependencies

Dependencies returns a snapshot of current dependencies.

```go
func (*SubgoalEntry) Dependencies() []*SubgoalEntry
```

**Parameters:**
  None

**Returns:**
- []*SubgoalEntry

### DerivationCount

DerivationCount returns the number of answers derived.

```go
func (*SubgoalEntry) DerivationCount() int64
```

**Parameters:**
  None

**Returns:**
- int64

### Event

Event returns a read-only channel that will be closed upon the next answer insertion or status change. After being closed, a new channel will be created for subsequent events.

```go
func (*SubgoalEntry) Event() <-chan *ast.StructType
```

**Parameters:**
  None

**Returns:**
- <-chan *ast.StructType

### EventSeq

EventSeq returns the current change sequence. Each call to signalEvent() increments this value.

```go
func (*SubgoalEntry) EventSeq() uint64
```

**Parameters:**
  None

**Returns:**
- uint64

### InsertAnswerWithSubsumption

InsertAnswerWithSubsumption inserts an answer with logical subsumption. - If an existing, non-retracted answer subsumes the new answer, do nothing (return false,-1). - Otherwise, retract any existing, non-retracted answers that are subsumed by the new answer, then insert the new answer into the trie and attach any pending delay set. Returns (wasNew, newIndex). When wasNew is false, newIndex is -1. Events for retractions are signaled by RetractAnswer; the caller should still signal for the insertion to preserve existing sequencing semantics.

```go
func (*SubgoalEntry) InsertAnswerWithSubsumption(bindings map[int64]Term) (bool, int)
```

**Parameters:**
- `bindings` (map[int64]Term)

**Returns:**
- bool
- int

### InvalidateByDomain

InvalidateByDomain retracts answers whose binding for varID is a concrete atom not compatible with the provided finite domain. Answers binding varID to a non-integer atom or leaving it unbound are left untouched (only integer atoms are interpreted as FD values). Returns the number of answers retracted.

```go
func (*SubgoalEntry) InvalidateByDomain(varID int64, dom Domain) int
```

**Parameters:**
- `varID` (int64)
- `dom` (Domain)

**Returns:**
- int

### IsRetracted

IsRetracted reports whether the answer at the given index is retracted.

```go
func (*SubgoalEntry) IsRetracted(index int) bool
```

**Parameters:**
- `index` (int)

**Returns:**
- bool

### Pattern

Pattern returns the call pattern for this subgoal.

```go
func (*SubgoalEntry) Pattern() *CallPattern
```

**Parameters:**
  None

**Returns:**
- *CallPattern

### QueueDelaySetForNextAnswer

QueueDelaySetForNextAnswer queues a DelaySet to be attached to the next answer inserted into this entry's answer trie. This allows evaluators to associate metadata with answers they are about to emit. Thread-safe for concurrent access.

```go
func (*SubgoalEntry) QueueDelaySetForNextAnswer(ds DelaySet)
```

**Parameters:**
- `ds` (DelaySet)

**Returns:**
  None

### Release

Release decrements the reference count and returns true if it reaches zero.

```go
func (*SubgoalEntry) Release() bool
```

**Parameters:**
  None

**Returns:**
- bool

### Retain

Retain increments the reference count.

```go
func (*SubgoalEntry) Retain()
```

**Parameters:**
  None

**Returns:**
  None

### RetractAnswer

RetractAnswer marks the answer at the given index as retracted (invisible). Thread-safe for concurrent access with metadata operations.

```go
func (*SubgoalEntry) RetractAnswer(index int)
```

**Parameters:**
- `index` (int)

**Returns:**
  None

### RetractByChild

RetractByChild retracts all answers whose delay set contains the given child. Returns the count of answers retracted.

```go
func (*SubgoalEntry) RetractByChild(child uint64) int
```

**Parameters:**
- `child` (uint64)

**Returns:**
- int

### SetStatus

SetStatus updates the evaluation status.

```go
func (*SubgoalEntry) SetStatus(status SubgoalStatus)
```

**Parameters:**
- `status` (SubgoalStatus)

**Returns:**
  None

### SetWfsTruth

SetWfsTruth sets the WFS truth value for this subgoal.

```go
func (*SubgoalEntry) SetWfsTruth(tv TruthValue)
```

**Parameters:**
- `tv` (TruthValue)

**Returns:**
  None

### SimplifyDelaySets

SimplifyDelaySets removes the provided child dependency from all delay sets in this entry. Returns two booleans: anyChanged indicates any DS was modified; stillDepends indicates whether any delay set in this entry still references child.

```go
func (*SubgoalEntry) SimplifyDelaySets(child uint64) (anyChanged bool, stillDepends bool)
```

**Parameters:**
- `child` (uint64)

**Returns:**
- bool
- bool

### Started

Started returns a channel that is closed when the producer goroutine for this subgoal has started. It is closed exactly once.

```go
func (*SubgoalEntry) Started() <-chan *ast.StructType
```

**Parameters:**
  None

**Returns:**
- <-chan *ast.StructType

### Status

Status returns the current evaluation status.

```go
func (*SubgoalEntry) Status() SubgoalStatus
```

**Parameters:**
  None

**Returns:**
- SubgoalStatus

### WaitChangeSince

WaitChangeSince returns a channel that will be closed when a change occurs with sequence strictly greater than 'since'. This method is race-free: it registers for the current event channel under lock and re-checks the sequence while still holding the lock to avoid missing an event between registration and check. If a change has already occurred (EventSeq() > since), it returns an already-closed channel.

```go
func (*SubgoalEntry) WaitChangeSince(since uint64) <-chan *ast.StructType
```

**Parameters:**
- `since` (uint64)

**Returns:**
- <-chan *ast.StructType

### WfsTruth

WfsTruth returns the current WFS truth value for this subgoal.

```go
func (*SubgoalEntry) WfsTruth() TruthValue
```

**Parameters:**
  None

**Returns:**
- TruthValue

### consumePendingDelaySet

consumePendingDelaySet retrieves and clears any queued DelaySet. Called by the producer after inserting an answer. Thread-safe for concurrent access.

```go
func (*SubgoalEntry) consumePendingDelaySet() DelaySet
```

**Parameters:**
  None

**Returns:**
- DelaySet

### signalEvent

signalEvent closes the current event channel (if not already closed) and replaces it with a new channel to signal future events.

```go
func (*SubgoalEntry) signalEvent()
```

**Parameters:**
  None

**Returns:**
  None

### signalStarted

signalStarted closes the startedCh if not already closed.

```go
func (*SubgoalEntry) signalStarted()
```

**Parameters:**
  None

**Returns:**
  None

### SubgoalStatus
SubgoalStatus represents the evaluation state of a tabled subgoal.

#### Example Usage

```go
// Example usage of SubgoalStatus
var value SubgoalStatus
// Initialize with appropriate value
```

#### Type Definition

```go
type SubgoalStatus int32
```

## Methods

### String

String returns a human-readable representation of the status.

```go
func (*LinearSum) String() string
```

**Parameters:**
  None

**Returns:**
- string

### SubgoalTable
SubgoalTable manages all tabled subgoals using a concurrent map. Thread safety: Uses sync.Map for lock-free concurrent access. The map is read-heavy (many lookups, few insertions), making sync.Map ideal.

#### Example Usage

```go
// Create a new SubgoalTable
subgoaltable := SubgoalTable{
    entries: /* value */,
    totalSubgoals: /* value */,
}
```

#### Type Definition

```go
type SubgoalTable struct {
    entries sync.Map
    totalSubgoals atomic.Int64
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| entries | `sync.Map` | Maps call pattern hash to SubgoalEntry |
| totalSubgoals | `atomic.Int64` | Total subgoals created (for statistics) |

### Constructor Functions

### NewSubgoalTable

NewSubgoalTable creates an empty subgoal table.

```go
func NewSubgoalTable() *SubgoalTable
```

**Parameters:**
  None

**Returns:**
- *SubgoalTable

## Methods

### AllEntries

AllEntries returns a snapshot of all subgoal entries. This is an O(n) operation used for debugging and statistics.

```go
func (*SubgoalTable) AllEntries() []*SubgoalEntry
```

**Parameters:**
  None

**Returns:**
- []*SubgoalEntry

### Clear

Clear removes all entries from the table.

```go
func (*SubgoalTable) Clear()
```

**Parameters:**
  None

**Returns:**
  None

### Delete

Delete removes a specific entry from the table by hash. Returns true if the entry was found and deleted.

```go
func (*SubgoalTable) Delete(hash uint64) bool
```

**Parameters:**
- `hash` (uint64)

**Returns:**
- bool

### Get

Get retrieves an existing subgoal entry by call pattern. Returns nil if not found.

```go
func (*GlobalConstraintBusPool) Get() *GlobalConstraintBus
```

**Parameters:**
  None

**Returns:**
- *GlobalConstraintBus

### GetByHash

GetByHash retrieves an existing subgoal entry by hash. Returns nil if not found.

```go
func (*SubgoalTable) GetByHash(hash uint64) *SubgoalEntry
```

**Parameters:**
- `hash` (uint64)

**Returns:**
- *SubgoalEntry

### GetOrCreate

GetOrCreate retrieves an existing subgoal entry or creates a new one. Returns the entry and a boolean indicating if it was newly created. Thread safety: Uses sync.Map.LoadOrStore for atomic get-or-create.

```go
func (*SubgoalTable) GetOrCreate(pattern *CallPattern) (*SubgoalEntry, bool)
```

**Parameters:**
- `pattern` (*CallPattern)

**Returns:**
- *SubgoalEntry
- bool

### TotalSubgoals

TotalSubgoals returns the total number of subgoals created.

```go
func (*SubgoalTable) TotalSubgoals() int64
```

**Parameters:**
  None

**Returns:**
- int64

### Substitution
Substitution represents a mapping from variables to terms. It's used to track bindings during unification and goal evaluation. The implementation is thread-safe and supports concurrent access.

#### Example Usage

```go
// Create a new Substitution
substitution := Substitution{
    bindings: map[],
    mu: /* value */,
}
```

#### Type Definition

```go
type Substitution struct {
    bindings map[int64]Term
    mu sync.RWMutex
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| bindings | `map[int64]Term` | Maps variable IDs to terms |
| mu | `sync.RWMutex` | Protects concurrent access |

### Constructor Functions

### NewSubstitution

NewSubstitution creates an empty substitution.

```go
func NewSubstitution() *Substitution
```

**Parameters:**
  None

**Returns:**
- *Substitution

### unify

unify performs the unification algorithm. Returns a new substitution if unification succeeds, nil if it fails.

```go
func unify(term1, term2 Term, sub *Substitution) *Substitution
```

**Parameters:**
- `term1` (Term)
- `term2` (Term)
- `sub` (*Substitution)

**Returns:**
- *Substitution

## Methods

### Bind

Bind creates a new substitution with an additional binding. Returns nil if the binding would create an inconsistency.

```go
func (*Substitution) Bind(v *Var, term Term) *Substitution
```

**Parameters:**
- `v` (*Var)
- `term` (Term)

**Returns:**
- *Substitution

### Clone

Clone creates a deep copy of the substitution.

```go
func (*Scale) Clone() PropagationConstraint
```

**Parameters:**
  None

**Returns:**
- PropagationConstraint

### DeepWalk

DeepWalk recursively walks a term, resolving variables in compound structures. This is essential for reifying solutions that contain nested structures.

```go
func (*Substitution) DeepWalk(term Term) Term
```

**Parameters:**
- `term` (Term)

**Returns:**
- Term

### Lookup

Lookup returns the term bound to a variable, or nil if unbound.

```go
func (*Substitution) Lookup(v *Var) Term
```

**Parameters:**
- `v` (*Var)

**Returns:**
- Term

### Size

Size returns the number of bindings in the substitution.

```go
func (*Substitution) Size() int
```

**Parameters:**
  None

**Returns:**
- int

### String

String returns a string representation of the substitution.

```go
func (*MembershipConstraint) String() string
```

**Parameters:**
  None

**Returns:**
- string

### Walk

Walk traverses a term following variable bindings in the substitution.

```go
func (*Substitution) Walk(term Term) Term
```

**Parameters:**
- `term` (Term)

**Returns:**
- Term

### SumConstraint
Example custom constraint implementations SumConstraint enforces that the sum of variables equals a target value

#### Example Usage

```go
// Create a new SumConstraint
sumconstraint := SumConstraint{
    vars: [],
    target: 42,
}
```

#### Type Definition

```go
type SumConstraint struct {
    vars []*FDVar
    target int
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| vars | `[]*FDVar` |  |
| target | `int` |  |

### Constructor Functions

### NewSumConstraint

NewSumConstraint creates a new sum constraint

```go
func NewSumConstraint(vars []*FDVar, target int) *SumConstraint
```

**Parameters:**
- `vars` ([]*FDVar)
- `target` (int)

**Returns:**
- *SumConstraint

## Methods

### IsSatisfied

IsSatisfied checks if the sum constraint is satisfied

```go
func (*AllDifferentConstraint) IsSatisfied() bool
```

**Parameters:**
  None

**Returns:**
- bool

### Propagate

Propagate performs constraint propagation for the sum constraint

```go
func (*InSetReified) Propagate(solver *Solver, state *SolverState) (*SolverState, error)
```

**Parameters:**
- `solver` (*Solver)
- `state` (*SolverState)

**Returns:**
- *SolverState
- error

### Variables

Variables returns the variables involved in this constraint

```go
func (*Lexicographic) Variables() []*FDVariable
```

**Parameters:**
  None

**Returns:**
- []*FDVariable

### Table
Table is an extensional constraint over a fixed list of allowed tuples.

#### Example Usage

```go
// Create a new Table
table := Table{
    vars: [],
    rows: [],
}
```

#### Type Definition

```go
type Table struct {
    vars []*FDVariable
    rows [][]int
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| vars | `[]*FDVariable` |  |
| rows | `[][]int` | len of each row equals len(vars) |

### Constructor Functions

### NewTable

NewTable constructs a new Table constraint given variables and allowed rows. Contract: - len(vars) > 0, all vars non-nil - len(rows) > 0, each row has exactly len(vars) entries - All row values are >= 1

```go
func NewTable(vars []*FDVariable, rows [][]int) (*Table, error)
```

**Parameters:**
- `vars` ([]*FDVariable)
- `rows` ([][]int)

**Returns:**
- *Table
- error

## Methods

### Propagate

Propagate enforces generalized arc consistency against the extensional table. Implements PropagationConstraint.

```go
func (*BoolSum) Propagate(solver *Solver, state *SolverState) (*SolverState, error)
```

**Parameters:**
- `solver` (*Solver)
- `state` (*SolverState)

**Returns:**
- *SolverState
- error

### String

String returns a human-readable description. Implements ModelConstraint.

```go
func (*Regular) String() string
```

**Parameters:**
  None

**Returns:**
- string

### Type

Type returns the constraint identifier. Implements ModelConstraint.

```go
func (*BinPacking) Type() string
```

**Parameters:**
  None

**Returns:**
- string

### Variables

Variables returns the involved variables. Implements ModelConstraint.

```go
func (*Diffn) Variables() []*FDVariable
```

**Parameters:**
  None

**Returns:**
- []*FDVariable

### TabledDatabase
from a database. This is useful for applications where all queries should be cached. Example: db := NewDatabase() // ... add facts ... tdb := WithTabledDatabase(db, "mydb") // All queries are automatically tabled goal := tdb.Query(edge, x, y)

#### Example Usage

```go
// Create a new TabledDatabase
tableddatabase := TabledDatabase{
    db: &Database{}{},
    idPrefix: "example",
}
```

#### Type Definition

```go
type TabledDatabase struct {
    db *Database
    idPrefix string
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| db | `*Database` |  |
| idPrefix | `string` |  |

### Constructor Functions

### TabledDB

TabledDB is a convenience wrapper for WithTabledDatabase.

```go
func TabledDB(db *Database, idPrefix string) *TabledDatabase
```

**Parameters:**
- `db` (*Database)
- `idPrefix` (string)

**Returns:**
- *TabledDatabase

### WithTabledDatabase

WithTabledDatabase creates a database wrapper that tables all queries.

```go
func WithTabledDatabase(db *Database, idPrefix string) *TabledDatabase
```

**Parameters:**
- `db` (*Database)
- `idPrefix` (string)

**Returns:**
- *TabledDatabase

## Methods

### AddFact

AddFact delegates to the underlying database and invalidates caches.

```go
func (*TabledDatabase) AddFact(rel *Relation, terms ...Term) (*TabledDatabase, error)
```

**Parameters:**
- `rel` (*Relation)
- `terms` (...Term)

**Returns:**
- *TabledDatabase
- error

### AllFacts

AllFacts delegates to the underlying database.

```go
func (*Database) AllFacts(rel *Relation) [][]Term
```

**Parameters:**
- `rel` (*Relation)

**Returns:**
- [][]Term

### FactCount

FactCount delegates to the underlying database.

```go
func (*TabledDatabase) FactCount(rel *Relation) int
```

**Parameters:**
- `rel` (*Relation)

**Returns:**
- int

### Q

Q on a TabledDatabase: same as Database.Q but tabled automatically.

```go
func (*TabledDatabase) Q(rel *Relation, args ...interface{}) Goal
```

**Parameters:**
- `rel` (*Relation)
- `args` (...interface{})

**Returns:**
- Goal

### Query

Query wraps Database.Query with automatic tabling.

```go
func (*TabledDatabase) Query(rel *Relation, args ...Term) Goal
```

**Parameters:**
- `rel` (*Relation)
- `args` (...Term)

**Returns:**
- Goal

### RemoveFact

RemoveFact delegates to the underlying database and invalidates caches.

```go
func (*TabledDatabase) RemoveFact(rel *Relation, terms ...Term) (*TabledDatabase, error)
```

**Parameters:**
- `rel` (*Relation)
- `terms` (...Term)

**Returns:**
- *TabledDatabase
- error

### Unwrap

Unwrap returns the underlying Database for operations that don't need tabling.

```go
func (*TabledDatabase) Unwrap() *Database
```

**Parameters:**
  None

**Returns:**
- *Database

### Term
Term represents any value in the miniKanren universe. Terms can be atoms, variables, compound structures, or any Go value. All Term implementations must be comparable and thread-safe.

#### Example Usage

```go
// Example implementation of Term
type MyTerm struct {
    // Add your fields here
}

func (m MyTerm) String() string {
    // Implement your logic here
    return
}

func (m MyTerm) Equal(param1 Term) bool {
    // Implement your logic here
    return
}

func (m MyTerm) IsVar() bool {
    // Implement your logic here
    return
}

func (m MyTerm) Clone() Term {
    // Implement your logic here
    return
}


```

#### Type Definition

```go
type Term interface {
    String() string
    Equal(other Term) bool
    IsVar() bool
    Clone() Term
}
```

## Methods

| Method | Description |
| ------ | ----------- |

### Constructor Functions

### A

A creates an Atom term from any Go value. Shorthand for NewAtom/AtomFromValue. Examples: A(1), A("hello"), A(true)

```go
func A(value interface{}) Term
```

**Parameters:**
- `value` (interface{})

**Returns:**
- Term

### ArrType

ArrType constructs an arrow type term (t1 -> t2) encoded as Pair(Atom("->"), Pair(t1, Pair(t2, Nil))).

```go
func ArrType(t1, t2 Term) Term
```

**Parameters:**
- `t1` (Term)
- `t2` (Term)

**Returns:**
- Term

### AsList

AsList collects a proper Scheme-like list into a Go slice of Terms. Returns false for non-list or improper lists.

```go
func AsList(t Term) ([]Term, bool)
```

**Parameters:**
- `t` (Term)

**Returns:**
- []Term
- bool

### EnvExtend

EnvExtend returns a new env mapping name->typ as Pair(Pair(name, typ), env).

```go
func EnvExtend(env Term, name *Atom, typ Term) Term
```

**Parameters:**
- `env` (Term)
- `name` (*Atom)
- `typ` (Term)

**Returns:**
- Term

### L

L builds a miniKanren list from values. Each element is converted to a Term: - Term values are used as-is - Other values are wrapped via A(...) Example: L(1, 2, 3) → (1 2 3)

```go
func L(values ...interface{}) Term
```

**Parameters:**
- `values` (...interface{})

**Returns:**
- Term

### List

List creates a list (chain of pairs) from a slice of terms. The list is terminated with nil (empty list). Example: lst := List(NewAtom(1), NewAtom(2), NewAtom(3)) // Creates: (1 . (2 . (3 . nil)))

```go
func List(terms ...Term) Term
```

**Parameters:**
- `terms` (...Term)

**Returns:**
- Term

### NewVarOr

NewVarOr returns typ if not nil, otherwise a fresh logic var. Helper to guide inference.

```go
func NewVarOr(typ Term) Term
```

**Parameters:**
- `typ` (Term)

**Returns:**
- Term

### ParallelRun

ParallelRun executes a goal in parallel and returns up to n solutions. This function creates a parallel executor, runs the goal, and cleans up.

```go
func ParallelRun(n int, goalFunc func(*Var) Goal) []Term
```

**Parameters:**
- `n` (int)
- `goalFunc` (func(*Var) Goal)

**Returns:**
- []Term

### ParallelRunWithConfig

ParallelRunWithConfig executes a goal in parallel with custom configuration.

```go
func ParallelRunWithConfig(n int, goalFunc func(*Var) Goal, config *ParallelConfig) []Term
```

**Parameters:**
- `n` (int)
- `goalFunc` (func(*Var) Goal)
- `config` (*ParallelConfig)

**Returns:**
- []Term

### ParallelRunWithContext

ParallelRunWithContext executes a goal in parallel with context and configuration.

```go
func ParallelRunWithContext(ctx context.Context, n int, goalFunc func(*Var) Goal, config *ParallelConfig) []Term
```

**Parameters:**
- `ctx` (context.Context)
- `n` (int)
- `goalFunc` (func(*Var) Goal)
- `config` (*ParallelConfig)

**Returns:**
- []Term

### Run

Run executes a goal and returns up to n solutions. This is the main entry point for executing miniKanren programs. It takes a goal that introduces one or more fresh variables and returns the values those variables can take. Example: solutions := Run(5, func(q *Var) Goal { return Eq(q, NewAtom("hello")) }) // Returns: [hello]

```go
func Run(n int, goalFunc func(*Var) Goal) []Term
```

**Parameters:**
- `n` (int)
- `goalFunc` (func(*Var) Goal)

**Returns:**
- []Term

### RunStar

RunStar executes a goal and returns all solutions. WARNING: This can run forever if the goal has infinite solutions. Use RunWithContext with a timeout for safer execution. Example: solutions := RunStar(func(q *Var) Goal { return Disj(Eq(q, NewAtom(1)), Eq(q, NewAtom(2))) }) // Returns: [1, 2]

```go
func RunStar(goalFunc func(*Var) Goal) []Term
```

**Parameters:**
- `goalFunc` (func(*Var) Goal)

**Returns:**
- []Term

### RunStarWithContext

RunStarWithContext executes a goal and returns all solutions with context support.

```go
func RunStarWithContext(ctx context.Context, goalFunc func(*Var) Goal) []Term
```

**Parameters:**
- `ctx` (context.Context)
- `goalFunc` (func(*Var) Goal)

**Returns:**
- []Term

### RunWithContext

RunWithContext executes a goal with a context for cancellation and timeouts. This allows for better control over long-running or infinite searches. Example: ctx, cancel := context.WithTimeout(context.Background(), time.Second) defer cancel() solutions := RunWithContext(ctx, 100, func(q *Var) Goal { return someLongRunningGoal(q) })

```go
func RunWithContext(ctx context.Context, n int, goalFunc func(*Var) Goal) []Term
```

**Parameters:**
- `ctx` (context.Context)
- `n` (int)
- `goalFunc` (func(*Var) Goal)

**Returns:**
- []Term

### RunWithIsolation

RunWithIsolation is like Run but uses an isolated constraint bus. Use this when you need complete constraint isolation between goals. Slightly slower than Run() but provides stronger isolation guarantees.

```go
func RunWithIsolation(n int, goalFunc func(*Var) Goal) []Term
```

**Parameters:**
- `n` (int)
- `goalFunc` (func(*Var) Goal)

**Returns:**
- []Term

### RunWithIsolationContext

RunWithIsolationContext is like RunWithContext but uses an isolated constraint bus.

```go
func RunWithIsolationContext(ctx context.Context, n int, goalFunc func(*Var) Goal) []Term
```

**Parameters:**
- `ctx` (context.Context)
- `n` (int)
- `goalFunc` (func(*Var) Goal)

**Returns:**
- []Term

### betaNormalizeDet

betaNormalizeDet reduces a term to normal form by repeatedly applying leftmost-outermost beta-reduction until no change occurs. Returns (normalForm, ok) where ok=false indicates pending due to unknown vars.

```go
func betaNormalizeDet(term Term) (Term, bool)
```

**Parameters:**
- `term` (Term)

**Returns:**
- Term
- bool

### betaReduceDet

betaReduceDet performs one leftmost-outermost beta-reduction step.

```go
func betaReduceDet(term Term) (Term, bool, bool)
```

**Parameters:**
- `term` (Term)

**Returns:**
- Term
- bool
- bool

### copyTermRecursive

copyTermRecursive performs the actual copying with variable tracking. The varMap ensures that shared variables in the original remain shared in the copy (with fresh variables).

```go
func copyTermRecursive(term Term, varMap map[int64]*Var) Term
```

**Parameters:**
- `term` (Term)
- `varMap` (map[int64]*Var)

**Returns:**
- Term

### envLookupDet

envLookupDet finds the type bound to name in env (alist). Returns (type, ok).

```go
func envLookupDet(env Term, name *Atom) (Term, bool)
```

**Parameters:**
- `env` (Term)
- `name` (*Atom)

**Returns:**
- Term
- bool

### intToPeano

intToPeano converts a non-negative integer to a Peano number. 0 -> Nil n -> (s . intToPeano(n-1))

```go
func intToPeano(n int) Term
```

**Parameters:**
- `n` (int)

**Returns:**
- Term

### renameBound

renameBound renames bound occurrences of `oldName` to `newName` within `term`. It assumes it's called on the body of a Tie(oldName, body). It does not descend into inner Tie that also bind oldName (to respect shadowing). For other inner binders, it recurses normally.

```go
func renameBound(term Term, oldName, newName *Atom) Term
```

**Parameters:**
- `term` (Term)
- `oldName` (*Atom)
- `newName` (*Atom)

**Returns:**
- Term

### substoDet

substoDet performs capture-avoiding substitution deterministically on a ground-enough term. Returns (result, true) if computed; (nil, false) if pending due to unresolved vars.

```go
func substoDet(term Term, name *Atom, replacement Term) (Term, bool)
```

**Parameters:**
- `term` (Term)
- `name` (*Atom)
- `replacement` (Term)

**Returns:**
- Term
- bool

### typeCheckDet

typeCheckDet attempts to infer the type of term under env, possibly guided by expected typ. Returns (inferredType, ok). ok=false indicates pending due to insufficient structure.

```go
func typeCheckDet(term Term, env Term, typ Term) (Term, bool)
```

**Parameters:**
- `term` (Term)
- `env` (Term)
- `typ` (Term)

**Returns:**
- Term
- bool

### walkTerm

walkTerm follows variable bindings to find the final value of a term.

```go
func walkTerm(term Term, bindings map[int64]Term) Term
```

**Parameters:**
- `term` (Term)
- `bindings` (map[int64]Term)

**Returns:**
- Term

### TieTerm
Nominal names are represented as atoms (e.g., NewAtom("a")). TieTerm encodes a binding form that binds a nominal name within body. Semantics: Tie(name, body) roughly corresponds to λ name . body This structure is used by freshness constraints and alpha-aware operations.

#### Example Usage

```go
// Create a new TieTerm
tieterm := TieTerm{
    name: &Atom{}{},
    body: Term{},
}
```

#### Type Definition

```go
type TieTerm struct {
    name *Atom
    body Term
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| name | `*Atom` |  |
| body | `Term` |  |

### Constructor Functions

### Lambda

Lambda is an alias for Tie to emphasize binder semantics.

```go
func Lambda(name *Atom, body Term) *TieTerm
```

**Parameters:**
- `name` (*Atom)
- `body` (Term)

**Returns:**
- *TieTerm

### Tie

Tie creates a binding form for a nominal name within a term body.

```go
func Tie(name *Atom, body Term) *TieTerm
```

**Parameters:**
- `name` (*Atom)
- `body` (Term)

**Returns:**
- *TieTerm

## Methods

### Clone

Clone makes a deep copy of the tie term.

```go
func (*UnifiedStore) Clone() *UnifiedStore
```

**Parameters:**
  None

**Returns:**
- *UnifiedStore

### Equal

Equal performs structural equality (NOT alpha-equivalence). Alpha-equivalence-aware equality will be provided by a separate goal/constraint.

```go
func (*BitSetDomain) Equal(other Domain) bool
```

**Parameters:**
- `other` (Domain)

**Returns:**
- bool

### IsVar

IsVar indicates this is not a logic variable.

```go
func (*Pair) IsVar() bool
```

**Parameters:**
  None

**Returns:**
- bool

### String

String renders the tie term in a readable form.

```go
func (*BoolSum) String() string
```

**Parameters:**
  None

**Returns:**
- string

### TruthValue
TruthValue represents the three-valued logic outcomes under WFS. For negation-as-failure over a subgoal G, the truth of not(G) is: - True:     G completes with no answers - False:    G produces at least one answer - Undefined: G is incomplete (conditional)

#### Example Usage

```go
// Example usage of TruthValue
var value TruthValue
// Initialize with appropriate value
```

#### Type Definition

```go
type TruthValue int
```

## Methods

### String



```go
func (*BinPacking) String() string
```

**Parameters:**
  None

**Returns:**
- string

### TypeConstraint
TypeConstraint implements type-based constraints (symbolo, numbero, etc.). It ensures that a term has a specific type, enabling type-safe relational programming patterns.

#### Example Usage

```go
// Create a new TypeConstraint
typeconstraint := TypeConstraint{
    id: "example",
    term: Term{},
    expectedType: TypeConstraintKind{},
    isLocal: true,
}
```

#### Type Definition

```go
type TypeConstraint struct {
    id string
    term Term
    expectedType TypeConstraintKind
    isLocal bool
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| id | `string` | id uniquely identifies this constraint instance |
| term | `Term` | term is the term that must have the specified type |
| expectedType | `TypeConstraintKind` | expectedType specifies what type the term must have |
| isLocal | `bool` | isLocal indicates whether this constraint can be checked locally |

### Constructor Functions

### NewTypeConstraint

NewTypeConstraint creates a new type constraint.

```go
func NewTypeConstraint(term Term, expectedType TypeConstraintKind) *TypeConstraint
```

**Parameters:**
- `term` (Term)
- `expectedType` (TypeConstraintKind)

**Returns:**
- *TypeConstraint

## Methods

### Check

Check evaluates the type constraint against current bindings. Returns ConstraintViolated if the term has the wrong type, ConstraintPending if the term is unbound, or ConstraintSatisfied if the term has the correct type. Implements the Constraint interface.

```go
func (*AlphaEqConstraint) Check(bindings map[int64]Term) ConstraintResult
```

**Parameters:**
- `bindings` (map[int64]Term)

**Returns:**
- ConstraintResult

### Clone

Clone creates a deep copy of the constraint for parallel execution. Implements the Constraint interface.

```go
func (*Scale) Clone() PropagationConstraint
```

**Parameters:**
  None

**Returns:**
- PropagationConstraint

### ID

ID returns the unique identifier for this constraint instance. Implements the Constraint interface.

```go
func (*MembershipConstraint) ID() string
```

**Parameters:**
  None

**Returns:**
- string

### IsLocal

IsLocal returns true if this constraint can be evaluated locally. Implements the Constraint interface.

```go
func (*AlphaEqConstraint) IsLocal() bool
```

**Parameters:**
  None

**Returns:**
- bool

### String

String returns a human-readable representation of the constraint. Implements the Constraint interface.

```go
func (*ElementValues) String() string
```

**Parameters:**
  None

**Returns:**
- string

### Variables

Variables returns the logic variables this constraint depends on. Implements the Constraint interface.

```go
func (*LessEqualConstraint) Variables() []*Var
```

**Parameters:**
  None

**Returns:**
- []*Var

### hasExpectedType

hasExpectedType checks if a term has the type expected by this constraint.

```go
func (*TypeConstraint) hasExpectedType(term Term) bool
```

**Parameters:**
- `term` (Term)

**Returns:**
- bool

### TypeConstraintKind
TypeConstraintKind represents the different types that can be constrained.

#### Example Usage

```go
// Example usage of TypeConstraintKind
var value TypeConstraintKind
// Initialize with appropriate value
```

#### Type Definition

```go
type TypeConstraintKind int
```

## Methods

### String

String returns a human-readable representation of the type constraint kind.

```go
func (*Regular) String() string
```

**Parameters:**
  None

**Returns:**
- string

### UnifiedStore
- State branching for parallel workers is O(1) - Memory overhead is O(changes) not O(total state) Store operations: - Relational: AddBinding(), GetBinding(), GetSubstitution() - Finite-domain: SetDomain(), GetDomain() - Cross-solver: NotifyChange() for propagation triggering Thread safety: UnifiedStore is immutable. All modification methods return new instances, making concurrent reads safe without locks.

#### Example Usage

```go
// Create a new UnifiedStore
unifiedstore := UnifiedStore{
    parent: &UnifiedStore{}{},
    relationalBindings: map[],
    fdDomains: map[],
    constraints: [],
    depth: 42,
    changedVars: map[],
}
```

#### Type Definition

```go
type UnifiedStore struct {
    parent *UnifiedStore
    relationalBindings map[int64]Term
    fdDomains map[int]Domain
    constraints []interface{}
    depth int
    changedVars map[int64]bool
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| parent | `*UnifiedStore` | parent points to the previous store version in the chain. nil for the root store. |
| relationalBindings | `map[int64]Term` | relationalBindings holds variable bindings from unification. Maps variable ID -> Term (Atom, Var, Pair) Uses copy-on-write: only modified bindings are stored, others are inherited from parent chain. |
| fdDomains | `map[int]Domain` | fdDomains holds finite domains for FD variables. Maps variable ID -> Domain Uses copy-on-write: only modified domains are stored. |
| constraints | `[]interface{}` | constraints holds all active constraints (both relational and FD). Inherited from parent unless explicitly modified. |
| depth | `int` | depth tracks the depth in the search tree. Used for heuristics and debugging. |
| changedVars | `map[int64]bool` | changedVars tracks which variables were modified in this version. Used to optimize propagation: only re-check constraints involving changed variables. |

### Constructor Functions

### MapQueryResult

MapQueryResult extracts a binding from a query result and maps it to an FD variable in the UnifiedStore. This encapsulates the manual mapping pattern used when propagating database facts to FD domains. This is a convenience function for the common operation: binding := result.GetBinding(relVar.ID()) store, err = store.AddBinding(int64(fdVar.ID()), binding)

```go
func MapQueryResult(result ConstraintStore, relVar *Var, fdVar *FDVariable, store *UnifiedStore) (*UnifiedStore, error)
```

**Parameters:**

- `result` (ConstraintStore) - The query result containing bindings

- `relVar` (*Var) - The relational variable to extract from result

- `fdVar` (*FDVariable) - The FD variable to bind in the store

- `store` (*UnifiedStore) - The current UnifiedStore

**Returns:**
- *UnifiedStore
- error

### NewUnifiedStore

NewUnifiedStore creates a new empty unified store. This is the root of the store chain for a new search.

```go
func NewUnifiedStore() *UnifiedStore
```

**Parameters:**
  None

**Returns:**
- *UnifiedStore

### NewUnifiedStoreFromModel

NewUnifiedStoreFromModel creates a fresh UnifiedStore populated with the model's FD domains and registered model constraints. This is a convenience helper for constructing the hybrid starting store used by the HybridSolver. The function validates the model, copies each variable's initial domain into the store, and adds model constraints so that plugins (e.g. FDPlugin) can discover them during propagation.

```go
func NewUnifiedStoreFromModel(m *Model) (*UnifiedStore, error)
```

**Parameters:**
- `m` (*Model)

**Returns:**
- *UnifiedStore
- error

## Methods

### AddBinding

AddBinding creates a new store with an additional relational binding. This is used by the relational solver during unification. Returns a new store with the binding, or an error if the binding would violate constraints.

```go
func (*UnifiedStoreAdapter) AddBinding(varID int64, term Term) error
```

**Parameters:**
- `varID` (int64)
- `term` (Term)

**Returns:**
- error

### AddConstraint

AddConstraint creates a new store with an additional constraint. Constraints are checked during propagation, not immediately.

```go
func (*LocalConstraintStoreImpl) AddConstraint(constraint Constraint) error
```

**Parameters:**
- `constraint` (Constraint)

**Returns:**
- error

### ChangedVariables

ChangedVariables returns the set of variables modified in this store version. Used to optimize propagation by only re-checking affected constraints.

```go
func (*UnifiedStore) ChangedVariables() map[int64]bool
```

**Parameters:**
  None

**Returns:**
- map[int64]bool

### Clone

Clone creates a shallow copy of the store for branching search paths. The new store shares most data with the parent via structural sharing.

```go
func (*RationalLinearSum) Clone() ModelConstraint
```

**Parameters:**
  None

**Returns:**
- ModelConstraint

### Depth

Depth returns the depth of this store in the search tree. Used for heuristics and debugging.

```go
func (*UnifiedStore) Depth() int
```

**Parameters:**
  None

**Returns:**
- int

### GetBinding

GetBinding retrieves the relational binding for a variable. Walks the parent chain to find the most recent binding. Returns nil if the variable is unbound.

```go
func (*UnifiedStore) GetBinding(varID int64) Term
```

**Parameters:**
- `varID` (int64)

**Returns:**
- Term

### GetConstraints

GetConstraints returns all active constraints in the store. This includes constraints from the entire parent chain.

```go
func (*UnifiedStore) GetConstraints() []interface{}
```

**Parameters:**
  None

**Returns:**
- []interface{}

### GetDomain

GetDomain retrieves the finite domain for an FD variable. Walks the parent chain to find the most recent domain. Returns nil if the variable has no FD domain (relational-only variable).

```go
func (*UnifiedStoreAdapter) GetDomain(varID int) Domain
```

**Parameters:**
- `varID` (int)

**Returns:**
- Domain

### GetSubstitution

GetSubstitution returns a Substitution representing all relational bindings. This bridges the UnifiedStore to miniKanren's substitution-based APIs.

```go
func (*UnifiedStoreAdapter) GetSubstitution() *Substitution
```

**Parameters:**
  None

**Returns:**
- *Substitution

### SetDomain

SetDomain creates a new store with an updated finite domain. This is used by the FD solver during propagation. Returns a new store with the domain change, or an error if the domain is empty (conflict detected).

```go
func (*UnifiedStore) SetDomain(varID int, domain Domain) (*UnifiedStore, error)
```

**Parameters:**
- `varID` (int)
- `domain` (Domain)

**Returns:**
- *UnifiedStore
- error

### String

String returns a human-readable representation of the store for debugging.

```go
func (*Lexicographic) String() string
```

**Parameters:**
  None

**Returns:**
- string

### collectBindings

collectBindings recursively collects bindings from the parent chain.

```go
func (*UnifiedStore) collectBindings(bindings map[int64]Term)
```

**Parameters:**
- `bindings` (map[int64]Term)

**Returns:**
  None

### collectDomains

collectDomains recursively collects domains from the parent chain.

```go
func (*UnifiedStore) collectDomains(domains map[int]Domain)
```

**Parameters:**
- `domains` (map[int]Domain)

**Returns:**
  None

### getAllBindings

getAllBindings walks the parent chain and collects all relational bindings. Used internally to avoid repeatedly walking the chain.

```go
func (*LocalConstraintStoreImpl) getAllBindings() map[int64]Term
```

**Parameters:**
  None

**Returns:**
- map[int64]Term

### getAllDomains

getAllDomains walks the parent chain and collects all FD domains.

```go
func (*UnifiedStore) getAllDomains() map[int]Domain
```

**Parameters:**
  None

**Returns:**
- map[int]Domain

### UnifiedStoreAdapter
1. Create adapter wrapping a UnifiedStore 2. Use adapter as ConstraintStore in goals (pldb queries, unification, etc.) 3. Extract UnifiedStore for hybrid propagation 4. Update adapter with propagated store 5. Clone adapter for search branching Performance notes: - Adapter overhead is minimal (single pointer dereference + mutex in write path) - UnifiedStore's copy-on-write means cloning is O(1) - Constraint checking delegates to UnifiedStore's constraint system

#### Example Usage

```go
// Create a new UnifiedStoreAdapter
unifiedstoreadapter := UnifiedStoreAdapter{
    store: &UnifiedStore{}{},
    mu: /* value */,
    id: "example",
}
```

#### Type Definition

```go
type UnifiedStoreAdapter struct {
    store *UnifiedStore
    mu sync.RWMutex
    id string
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| store | `*UnifiedStore` | store holds the current version of the UnifiedStore. Updated atomically on mutations. |
| mu | `sync.RWMutex` | mu protects concurrent access to store pointer updates. Read operations on UnifiedStore don't need locks (immutable data). Write operations (mutations that create new store versions) do. |
| id | `string` | idCounter generates unique IDs for adapter instances. Used for debugging and identifying store lineage. |

### Constructor Functions

### NewUnifiedStoreAdapter

NewUnifiedStoreAdapter creates a ConstraintStore adapter wrapping the given UnifiedStore. The adapter takes ownership of the store reference and will update it on mutations. Example: store := NewUnifiedStore() adapter := NewUnifiedStoreAdapter(store) goal := db.Query(person, Fresh("name"), Fresh("age")) stream := goal(ctx, adapter)

```go
func NewUnifiedStoreAdapter(store *UnifiedStore) *UnifiedStoreAdapter
```

**Parameters:**
- `store` (*UnifiedStore)

**Returns:**
- *UnifiedStoreAdapter

## Methods

### AddBinding

AddBinding binds a variable to a term in the underlying UnifiedStore. Implements ConstraintStore interface. Returns error if the binding would violate constraints. In the UnifiedStore model, binding errors typically come from the hybrid solver during propagation, but we return any error from AddBinding for interface compliance.

```go
func (*UnifiedStoreAdapter) AddBinding(varID int64, term Term) error
```

**Parameters:**
- `varID` (int64)
- `term` (Term)

**Returns:**
- error

### AddConstraint

AddConstraint adds a constraint to the underlying UnifiedStore. Implements ConstraintStore interface. Note: UnifiedStore uses interface{} for constraints (not typed Constraint), allowing both relational constraints and FD constraints. The hybrid solver's plugins determine how to handle each constraint type during propagation. Returns nil always - constraint violations are detected during propagation, not at constraint addition time. This matches UnifiedStore's batched constraint checking philosophy.

```go
func (*UnifiedStore) AddConstraint(constraint interface{}) *UnifiedStore
```

**Parameters:**
- `constraint` (interface{})

**Returns:**
- *UnifiedStore

### Clone

Clone creates a deep copy of the adapter with an independent UnifiedStore. Implements ConstraintStore interface. The cloned adapter starts with a copy of the current store (via UnifiedStore.Clone), enabling parallel search where each branch has its own constraint evolution. Cloning is cheap (O(1)) due to UnifiedStore's copy-on-write semantics with structural sharing. Most data is shared until modified.

```go
func (*AlphaEqConstraint) Clone() Constraint
```

**Parameters:**
  None

**Returns:**
- Constraint

### Depth

Depth returns the depth of the underlying store in the search tree. Used for heuristics and debugging. Not part of ConstraintStore interface.

```go
func (*UnifiedStoreAdapter) Depth() int
```

**Parameters:**
  None

**Returns:**
- int

### GetBinding

GetBinding retrieves the relational binding for a variable. Implements ConstraintStore interface. Returns nil if the variable is unbound. Thread-safe due to UnifiedStore immutability.

```go
func (*UnifiedStoreAdapter) GetBinding(varID int64) Term
```

**Parameters:**
- `varID` (int64)

**Returns:**
- Term

### GetConstraints

GetConstraints returns all active constraints in the underlying store. Implements ConstraintStore interface. Note: Returns []Constraint but UnifiedStore stores []interface{}. This assumes all constraints added implement the Constraint interface. If non-Constraint objects are added (e.g., FD-specific constraints), they'll be filtered out.

```go
func (*UnifiedStore) GetConstraints() []interface{}
```

**Parameters:**
  None

**Returns:**
- []interface{}

### GetDomain

GetDomain retrieves the FD domain for a variable from the underlying UnifiedStore. This is not part of the ConstraintStore interface but provides access to FD domains for hybrid solving scenarios. Returns nil if the variable has no FD domain (relational-only variable).

```go
func (*UnifiedStore) GetDomain(varID int) Domain
```

**Parameters:**
- `varID` (int)

**Returns:**
- Domain

### GetSubstitution

GetSubstitution returns a Substitution representing all relational bindings. Implements ConstraintStore interface. This bridges UnifiedStore to miniKanren's substitution-based APIs. The substitution is a snapshot at call time; subsequent mutations won't affect it.

```go
func (*UnifiedStore) GetSubstitution() *Substitution
```

**Parameters:**
  None

**Returns:**
- *Substitution

### SetDomain

SetDomain updates the FD domain for a variable in the underlying UnifiedStore. This is not part of the ConstraintStore interface but provides FD domain updates for hybrid solving scenarios. Returns error if the domain is empty (conflict detected).

```go
func (*UnifiedStoreAdapter) SetDomain(varID int, domain Domain) error
```

**Parameters:**
- `varID` (int)
- `domain` (Domain)

**Returns:**
- error

### SetUnifiedStore

SetUnifiedStore updates the adapter's underlying store. Used after hybrid solver propagation to install the propagated store. This method should be used with care: it replaces the entire store, so any bindings/constraints added directly to the adapter (bypassing the hybrid solver) will be overwritten. Typical usage: store := adapter.UnifiedStore() propagated, err := hybridSolver.Propagate(store) if err != nil { // Conflict detected, backtrack return } adapter.SetUnifiedStore(propagated)

```go
func (*UnifiedStoreAdapter) SetUnifiedStore(store *UnifiedStore)
```

**Parameters:**
- `store` (*UnifiedStore)

**Returns:**
  None

### String

String returns a human-readable representation for debugging. Implements ConstraintStore interface.

```go
func (*BinPacking) String() string
```

**Parameters:**
  None

**Returns:**
- string

### UnifiedStore

UnifiedStore returns the underlying UnifiedStore for hybrid solver operations. This allows extracting the store for propagation with HybridSolver. Example usage pattern: adapter := NewUnifiedStoreAdapter(store) // ... use adapter with goals/pldb ... hybridStore := adapter.UnifiedStore() propagated, err := hybridSolver.Propagate(hybridStore) if err == nil { adapter.SetUnifiedStore(propagated) }

```go
func (*UnifiedStoreAdapter) UnifiedStore() *UnifiedStore
```

**Parameters:**
  None

**Returns:**
- *UnifiedStore

### ValueEqualsReified
ValueEqualsReified links a variable v and a boolean b such that b=2 iff v==target. Domain conventions: b ∈ {1=false, 2=true}

#### Example Usage

```go
// Create a new ValueEqualsReified
valueequalsreified := ValueEqualsReified{
    v: &FDVariable{}{},
    target: 42,
    boolVar: &FDVariable{}{},
}
```

#### Type Definition

```go
type ValueEqualsReified struct {
    v *FDVariable
    target int
    boolVar *FDVariable
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| v | `*FDVariable` |  |
| target | `int` |  |
| boolVar | `*FDVariable` |  |

### Constructor Functions

### NewValueEqualsReified

NewValueEqualsReified creates a reified equality to a constant target.

```go
func NewValueEqualsReified(v *FDVariable, target int, boolVar *FDVariable) (*ValueEqualsReified, error)
```

**Parameters:**
- `v` (*FDVariable)
- `target` (int)
- `boolVar` (*FDVariable)

**Returns:**
- *ValueEqualsReified
- error

## Methods

### Propagate

Propagate enforces b ↔ (v == target) with bidirectional pruning.

```go
func (*Cumulative) Propagate(solver *Solver, state *SolverState) (*SolverState, error)
```

**Parameters:**
- `solver` (*Solver)
- `state` (*SolverState)

**Returns:**
- *SolverState
- error

### String



```go
func (*FDVariable) String() string
```

**Parameters:**
  None

**Returns:**
- string

### Type



```go
func (*RationalLinearSum) Type() string
```

**Parameters:**
  None

**Returns:**
- string

### Variables



```go
func (*EqualityReified) Variables() []*FDVariable
```

**Parameters:**
  None

**Returns:**
- []*FDVariable

### ValueOrderingHeuristic
ValueOrderingHeuristic defines strategies for ordering values within a domain

#### Example Usage

```go
// Example usage of ValueOrderingHeuristic
var value ValueOrderingHeuristic
// Initialize with appropriate value
```

#### Type Definition

```go
type ValueOrderingHeuristic int
```

### Var
Var represents a logic variable in miniKanren. Variables can be bound to values through unification. Each variable has a unique identifier to distinguish it from others.

#### Example Usage

```go
// Create a new Var
var := Var{
    id: 42,
    name: "example",
    mu: /* value */,
}
```

#### Type Definition

```go
type Var struct {
    id int64
    name string
    mu sync.RWMutex
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| id | `int64` | Unique identifier |
| name | `string` | Optional name for debugging |
| mu | `sync.RWMutex` | Protects concurrent access |

### Constructor Functions

### Fresh

Fresh creates a new logic variable with an optional name for debugging. Each call to Fresh generates a variable with a globally unique ID, ensuring no variable conflicts even in concurrent environments. Example: x := Fresh("x")  // Creates a variable named x y := Fresh("")   // Creates an anonymous variable

```go
func Fresh(name string) *Var
```

**Parameters:**
- `name` (string)

**Returns:**
- *Var

### extractVariables

extractVariables recursively extracts all variables from a term.

```go
func extractVariables(term Term) []*Var
```

**Parameters:**
- `term` (Term)

**Returns:**
- []*Var

## Methods

### Clone

Clone creates a copy of the variable with the same identity.

```go
func (*Substitution) Clone() *Substitution
```

**Parameters:**
  None

**Returns:**
- *Substitution

### Equal

Equal checks if two variables are the same variable.

```go
func (*BitSetDomain) Equal(other Domain) bool
```

**Parameters:**
- `other` (Domain)

**Returns:**
- bool

### ID

ID returns the unique identifier of the variable.

```go
func (*LocalConstraintStoreImpl) ID() string
```

**Parameters:**
  None

**Returns:**
- string

### IsVar

IsVar always returns true for variables.

```go
func (*Pair) IsVar() bool
```

**Parameters:**
  None

**Returns:**
- bool

### String

String returns a string representation of the variable.

```go
func (*Inequality) String() string
```

**Parameters:**
  None

**Returns:**
- string

### Variable
Variable represents a decision variable in a constraint satisfaction problem. Variables have identities, domains of possible values, and participate in constraints. The Variable abstraction allows the solver to be agnostic to the underlying domain representation, enabling different domain types (finite domains, intervals, sets, etc.) to coexist in the same model. Variables in the Model hold initial domains and are immutable once solving begins. During solving, the Solver tracks domain changes via SolverState using the variable's ID.

#### Example Usage

```go
// Example implementation of Variable
type MyVariable struct {
    // Add your fields here
}

func (m MyVariable) ID() int {
    // Implement your logic here
    return
}

func (m MyVariable) Domain() Domain {
    // Implement your logic here
    return
}

func (m MyVariable) IsBound() bool {
    // Implement your logic here
    return
}

func (m MyVariable) Value() int {
    // Implement your logic here
    return
}

func (m MyVariable) String() string {
    // Implement your logic here
    return
}


```

#### Type Definition

```go
type Variable interface {
    ID() int
    Domain() Domain
    IsBound() bool
    Value() int
    String() string
}
```

## Methods

| Method | Description |
| ------ | ----------- |

### VariableOrderingHeuristic
VariableOrderingHeuristic defines strategies for selecting the next variable to assign

#### Example Usage

```go
// Example usage of VariableOrderingHeuristic
var value VariableOrderingHeuristic
// Initialize with appropriate value
```

#### Type Definition

```go
type VariableOrderingHeuristic int
```

### VersionInfo
VersionInfo provides detailed version information.

#### Example Usage

```go
// Create a new VersionInfo
versioninfo := VersionInfo{
    Version: "example",
    GoVersion: "example",
    GitCommit: "example",
    BuildDate: "example",
}
```

#### Type Definition

```go
type VersionInfo struct {
    Version string `json:"version"`
    GoVersion string `json:"go_version"`
    GitCommit string `json:"git_commit,omitempty"`
    BuildDate string `json:"build_date,omitempty"`
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| Version | `string` |  |
| GoVersion | `string` |  |
| GitCommit | `string` |  |
| BuildDate | `string` |  |

### Constructor Functions

### GetVersionInfo

GetVersionInfo returns detailed version information.

```go
func GetVersionInfo() VersionInfo
```

**Parameters:**
  None

**Returns:**
- VersionInfo

## Functions

### AsInt
AsInt attempts to extract an int from a reified Term (Atom). Returns false on mismatch.

```go
func AsInt(t Term) (int, bool)
```

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `t` | `Term` | |

**Returns:**
| Type | Description |
|------|-------------|
| `int` | |
| `bool` | |

**Example:**

```go
// Example usage of AsInt
result := AsInt(/* parameters */)
```

### AsString
AsString attempts to extract a string from a reified Term (Atom).

```go
func AsString(t Term) (string, bool)
```

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `t` | `Term` | |

**Returns:**
| Type | Description |
|------|-------------|
| `string` | |
| `bool` | |

**Example:**

```go
// Example usage of AsString
result := AsString(/* parameters */)
```

### FormatSolutions
FormatSolutions pretty-prints a slice of solutions for human-friendly output. Each solution is rendered as "name: value, name2: value2" with lists and strings formatted pleasantly. Output is sorted for stable tests.

```go
func FormatSolutions(solutions []map[string]Term) []string
```

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `solutions` | `[]map[string]Term` | |

**Returns:**
| Type | Description |
|------|-------------|
| `[]string` | |

**Example:**

```go
// Example usage of FormatSolutions
result := FormatSolutions(/* parameters */)
```

### FormatTerm
FormatTerm returns the canonical human-friendly string for a reified Term. It mirrors the formatting used by FormatSolutions: - Empty list: () - Proper lists: (a b c) - Improper lists: (a b . tail) - Strings quoted; other atoms via fmt %%v

```go
func FormatTerm(t Term) string
```

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `t` | `Term` | |

**Returns:**
| Type | Description |
|------|-------------|
| `string` | |

**Example:**

```go
// Example usage of FormatTerm
result := FormatTerm(/* parameters */)
```

### GetVersion
GetVersion returns the current version string.

```go
func GetVersion() string
```

**Parameters:**
None

**Returns:**
| Type | Description |
|------|-------------|
| `string` | |

**Example:**

```go
// Example usage of GetVersion
result := GetVersion(/* parameters */)
```

### Ints
Ints is IntsN with n<=0 (all results).

```go
func Ints(goal Goal, v *Var) []int
```

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `goal` | `Goal` | |
| `v` | `*Var` | |

**Returns:**
| Type | Description |
|------|-------------|
| `[]int` | |

**Example:**

```go
// Example usage of Ints
result := Ints(/* parameters */)
```

### IntsN
IntsN solves for a single variable and returns up to n integer values. Non-int bindings are skipped. When n<=0, all results are returned.

```go
func IntsN(ctx context.Context, n int, goal Goal, v *Var) []int
```

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `ctx` | `context.Context` | |
| `n` | `int` | |
| `goal` | `Goal` | |
| `v` | `*Var` | |

**Returns:**
| Type | Description |
|------|-------------|
| `[]int` | |

**Example:**

```go
// Example usage of IntsN
result := IntsN(/* parameters */)
```

### InvalidateAll
InvalidateAll clears the entire SLG answer table. Use this after major database changes when fine-grained invalidation is impractical.

```go
func InvalidateAll()
```

**Parameters:**
None

**Returns:**
None

**Example:**

```go
// Example usage of InvalidateAll
result := InvalidateAll(/* parameters */)
```

### InvalidateRelation
InvalidateRelation removes all cached answers for queries involving a specific relation. This should be called when the relation's facts change (AddFact/RemoveFact). The SLG engine now provides fine-grained predicate-level invalidation, removing only the cached answers for the specified predicateID while preserving unrelated tabled predicates. This is more efficient than clearing the entire table.

```go
func InvalidateRelation(predicateID string)
```

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `predicateID` | `string` | The predicate identifier used in TabledQuery calls |

**Returns:**
None

**Example:**

```go
// Example usage of InvalidateRelation
result := InvalidateRelation(/* parameters */)
```

### MustInt
MustInt extracts an int from a Term or panics. Intended for examples/tests.

```go
func MustInt(t Term) int
```

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `t` | `Term` | |

**Returns:**
| Type | Description |
|------|-------------|
| `int` | |

**Example:**

```go
// Example usage of MustInt
result := MustInt(/* parameters */)
```

### MustString
MustString extracts a string from a Term or panics.

```go
func MustString(t Term) string
```

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `t` | `Term` | |

**Returns:**
| Type | Description |
|------|-------------|
| `string` | |

**Example:**

```go
// Example usage of MustString
result := MustString(/* parameters */)
```

### NewHybridSolverFromModel
NewHybridSolverFromModel builds a HybridSolver wired for the given model and returns it along with a UnifiedStore pre-populated from the model. The returned solver registers both the Relational and FD plugins in that order which is the common configuration for hybrid solving.

```go
func NewHybridSolverFromModel(m *Model) (*HybridSolver, *UnifiedStore, error)
```

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `m` | `*Model` | |

**Returns:**
| Type | Description |
|------|-------------|
| `*HybridSolver` | |
| `*UnifiedStore` | |
| `error` | |

**Example:**

```go
// Example usage of NewHybridSolverFromModel
result := NewHybridSolverFromModel(/* parameters */)
```

### NewRationalLinearSumWithScaling
NewRationalLinearSumWithScaling creates a RationalLinearSum and handles result scaling automatically. This is a convenience wrapper that uses ScaledDivision when needed (scale > 1). Returns the RationalLinearSum constraint plus an optional ScaledDivision constraint that must also be added to the model. Usage: rls, scaledDiv, err := NewRationalLinearSumWithScaling(vars, coeffs, result, model) model.AddConstraint(rls) if scaledDiv != nil { model.AddConstraint(scaledDiv) } When scale == 1: Returns only RationalLinearSum, scaledDiv is nil When scale > 1: Returns RationalLinearSum with scaled intermediate variable, plus ScaledDivision constraint linking intermediate to result

```go
func NewRationalLinearSumWithScaling(vars []*FDVariable, coeffs []Rational, result *FDVariable, model *Model) (*RationalLinearSum, *ScaledDivision, error)
```

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `vars` | `[]*FDVariable` | |
| `coeffs` | `[]Rational` | |
| `result` | `*FDVariable` | |
| `model` | `*Model` | |

**Returns:**
| Type | Description |
|------|-------------|
| `*RationalLinearSum` | |
| `*ScaledDivision` | |
| `error` | |

**Example:**

```go
// Example usage of NewRationalLinearSumWithScaling
result := NewRationalLinearSumWithScaling(/* parameters */)
```

### Optimize
Optimize finds a solution that optimizes the objective variable. It is a thin wrapper around Solver.SolveOptimal with context.Background().

```go
func Optimize(m *Model, obj *FDVariable, minimize bool) ([]int, int, error)
```

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `m` | `*Model` | |
| `obj` | `*FDVariable` | FD variable representing the objective (e.g., a LinearSum total) |
| `minimize` | `bool` | true to minimize, false to maximize |

**Returns:**
| Type | Description |
|------|-------------|
| `[]int` | |
| `int` | |
| `error` | |

**Example:**

```go
// Example usage of Optimize
result := Optimize(/* parameters */)
```

### OptimizeWithOptions
OptimizeWithOptions is like Optimize but accepts a context and solver options for time/node limits or parallel workers. See WithParallelWorkers, WithNodeLimit, and other OptimizeOption helpers.

```go
func OptimizeWithOptions(ctx context.Context, m *Model, obj *FDVariable, minimize bool, opts ...OptimizeOption) ([]int, int, error)
```

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `ctx` | `context.Context` | |
| `m` | `*Model` | |
| `obj` | `*FDVariable` | |
| `minimize` | `bool` | |
| `opts` | `...OptimizeOption` | |

**Returns:**
| Type | Description |
|------|-------------|
| `[]int` | |
| `int` | |
| `error` | |

**Example:**

```go
// Example usage of OptimizeWithOptions
result := OptimizeWithOptions(/* parameters */)
```

### PairsInts
PairsInts is PairsIntsN with n<=0 (all results).

```go
func PairsInts(goal Goal, x, y *Var) [][*ast.BasicLit]int
```

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `goal` | `Goal` | |
| `x` | `*Var` | |
| `y` | `*Var` | |

**Returns:**
| Type | Description |
|------|-------------|
| `[][*ast.BasicLit]int` | |

**Example:**

```go
// Example usage of PairsInts
result := PairsInts(/* parameters */)
```

### PairsIntsN
PairsIntsN returns up to n pairs of ints for two projected variables. Rows with non-int bindings are skipped.

```go
func PairsIntsN(ctx context.Context, n int, goal Goal, x, y *Var) [][*ast.BasicLit]int
```

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `ctx` | `context.Context` | |
| `n` | `int` | |
| `goal` | `Goal` | |
| `x` | `*Var` | |
| `y` | `*Var` | |

**Returns:**
| Type | Description |
|------|-------------|
| `[][*ast.BasicLit]int` | |

**Example:**

```go
// Example usage of PairsIntsN
result := PairsIntsN(/* parameters */)
```

### PairsStrings
PairsStrings is PairsStringsN with n<=0 (all results).

```go
func PairsStrings(goal Goal, x, y *Var) [][*ast.BasicLit]string
```

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `goal` | `Goal` | |
| `x` | `*Var` | |
| `y` | `*Var` | |

**Returns:**
| Type | Description |
|------|-------------|
| `[][*ast.BasicLit]string` | |

**Example:**

```go
// Example usage of PairsStrings
result := PairsStrings(/* parameters */)
```

### PairsStringsN
PairsStringsN returns up to n pairs of strings for two projected variables. Rows with non-string bindings are skipped.

```go
func PairsStringsN(ctx context.Context, n int, goal Goal, x, y *Var) [][*ast.BasicLit]string
```

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `ctx` | `context.Context` | |
| `n` | `int` | |
| `goal` | `Goal` | |
| `x` | `*Var` | |
| `y` | `*Var` | |

**Returns:**
| Type | Description |
|------|-------------|
| `[][*ast.BasicLit]string` | |

**Example:**

```go
// Example usage of PairsStringsN
result := PairsStringsN(/* parameters */)
```

### RecursiveTablePred
RecursiveTablePred provides a thin HLAPI wrapper around TabledRecursivePredicate. It returns a predicate constructor that accepts native values or Terms when called, converting non-Terms to Atoms automatically. The recursive definition uses the same signature as TabledRecursivePredicate: a callback that receives a self predicate (for recursive calls) and the instantiated call arguments as Terms, and must return the recursive case Goal. The base case over baseRel is handled automatically by the underlying helper. Example: ancestor := RecursiveTablePred(db, parent, "ancestor2", func(self func(...Term) Goal, args ...Term) Goal { x, y := args[0], args[1] z := Fresh("z") return Conj( db.Query(parent, x, z), // base facts used in recursive step self(z, y),              // recursive call to tabled predicate ) }) // Use native values or Terms at call sites goal := ancestor(Fresh("x"), "carol")

```go
func RecursiveTablePred(db *Database, baseRel *Relation, predicateID string, recursive func(self func(...Term) Goal, args ...Term) Goal) func(...interface{}) Goal
```

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `db` | `*Database` | |
| `baseRel` | `*Relation` | |
| `predicateID` | `string` | |
| `recursive` | `func(self func(...Term) Goal, args ...Term) Goal` | |

**Returns:**
| Type | Description |
|------|-------------|
| `func(...interface{}) Goal` | |

**Example:**

```go
// Example usage of RecursiveTablePred
result := RecursiveTablePred(/* parameters */)
```

### ResetGlobalEngine
ResetGlobalEngine clears the global engine's cache and resets it.

```go
func ResetGlobalEngine()
```

**Parameters:**
None

**Returns:**
None

**Example:**

```go
// Example usage of ResetGlobalEngine
result := ResetGlobalEngine(/* parameters */)
```

### ReturnPooledGlobalBus
ReturnPooledGlobalBus returns a bus to the pool

```go
func ReturnPooledGlobalBus(bus *GlobalConstraintBus)
```

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `bus` | `*GlobalConstraintBus` | |

**Returns:**
None

**Example:**

```go
// Example usage of ReturnPooledGlobalBus
result := ReturnPooledGlobalBus(/* parameters */)
```

### Rows
Rows is RowsN with n<=0 (all results). WARNING: may not terminate on goals with infinite streams.

```go
func Rows(goal Goal, vars ...*Var) [][]Term
```

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `goal` | `Goal` | |
| `vars` | `...*Var` | |

**Returns:**
| Type | Description |
|------|-------------|
| `[][]Term` | |

**Example:**

```go
// Example usage of Rows
result := Rows(/* parameters */)
```

### RowsAllCtx
RowsAllCtx returns all rows using the provided context. Use a timeout/cancel to avoid infinite enumeration.

```go
func RowsAllCtx(ctx context.Context, goal Goal, vars ...*Var) [][]Term
```

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `ctx` | `context.Context` | |
| `goal` | `Goal` | |
| `vars` | `...*Var` | |

**Returns:**
| Type | Description |
|------|-------------|
| `[][]Term` | |

**Example:**

```go
// Example usage of RowsAllCtx
result := RowsAllCtx(/* parameters */)
```

### RowsAllTimeout
RowsAllTimeout returns all rows but aborts enumeration after the given timeout.

```go
func RowsAllTimeout(timeout time.Duration, goal Goal, vars ...*Var) [][]Term
```

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `timeout` | `time.Duration` | |
| `goal` | `Goal` | |
| `vars` | `...*Var` | |

**Returns:**
| Type | Description |
|------|-------------|
| `[][]Term` | |

**Example:**

```go
// Example usage of RowsAllTimeout
result := RowsAllTimeout(/* parameters */)
```

### RowsAsInts
RowsAsInts converts [][]Term rows into [][]int, keeping only rows where all entries are int Atoms. Rows with any non-int terms are skipped.

```go
func RowsAsInts(rows [][]Term) [][]int
```

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `rows` | `[][]Term` | |

**Returns:**
| Type | Description |
|------|-------------|
| `[][]int` | |

**Example:**

```go
// Example usage of RowsAsInts
result := RowsAsInts(/* parameters */)
```

### RowsAsStrings
RowsAsStrings converts [][]Term rows into [][]string, keeping only rows where all entries are string Atoms. Rows with any non-string terms are skipped.

```go
func RowsAsStrings(rows [][]Term) [][]string
```

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `rows` | `[][]Term` | |

**Returns:**
| Type | Description |
|------|-------------|
| `[][]string` | |

**Example:**

```go
// Example usage of RowsAsStrings
result := RowsAsStrings(/* parameters */)
```

### RowsN
RowsN runs a goal and returns up to n rows of reified Terms projected in the order of vars provided. Each row corresponds to one solution. If no vars are provided, each row contains a single Atom(nil) to preserve cardinality. When n<=0, all solutions are returned (which may not terminate for infinite goals).

```go
func RowsN(ctx context.Context, n int, goal Goal, vars ...*Var) [][]Term
```

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `ctx` | `context.Context` | |
| `n` | `int` | |
| `goal` | `Goal` | |
| `vars` | `...*Var` | |

**Returns:**
| Type | Description |
|------|-------------|
| `[][]Term` | |

**Example:**

```go
// Example usage of RowsN
result := RowsN(/* parameters */)
```

### SetGlobalEngine
SetGlobalEngine sets the global SLG engine. This is useful for testing or custom configurations.

```go
func SetGlobalEngine(engine *SLGEngine)
```

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `engine` | `*SLGEngine` | |

**Returns:**
None

**Example:**

```go
// Example usage of SetGlobalEngine
result := SetGlobalEngine(/* parameters */)
```

### Solutions
Solutions is SolutionsN with n<=0 (all results). WARNING: may not terminate on goals with infinite streams.

```go
func Solutions(goal Goal, vars ...*Var) []map[string]Term
```

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `goal` | `Goal` | |
| `vars` | `...*Var` | |

**Returns:**
| Type | Description |
|------|-------------|
| `[]map[string]Term` | |

**Example:**

```go
// Example usage of Solutions
result := Solutions(/* parameters */)
```

### SolutionsAllCtx
SolutionsAllCtx returns all solutions (unbounded) using the provided context. Use a context with timeout/cancel to avoid infinite enumeration.

```go
func SolutionsAllCtx(ctx context.Context, goal Goal, vars ...*Var) []map[string]Term
```

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `ctx` | `context.Context` | |
| `goal` | `Goal` | |
| `vars` | `...*Var` | |

**Returns:**
| Type | Description |
|------|-------------|
| `[]map[string]Term` | |

**Example:**

```go
// Example usage of SolutionsAllCtx
result := SolutionsAllCtx(/* parameters */)
```

### SolutionsAllTimeout
SolutionsAllTimeout returns all solutions but aborts enumeration after the given timeout.

```go
func SolutionsAllTimeout(timeout time.Duration, goal Goal, vars ...*Var) []map[string]Term
```

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `timeout` | `time.Duration` | |
| `goal` | `Goal` | |
| `vars` | `...*Var` | |

**Returns:**
| Type | Description |
|------|-------------|
| `[]map[string]Term` | |

**Example:**

```go
// Example usage of SolutionsAllTimeout
result := SolutionsAllTimeout(/* parameters */)
```

### SolutionsCtx
SolutionsCtx is an alias for SolutionsN that improves discoverability when passing an explicit context and solution cap together. It returns up to n solutions (n<=0 for all solutions, which may not terminate).

```go
func SolutionsCtx(ctx context.Context, n int, goal Goal, vars ...*Var) []map[string]Term
```

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `ctx` | `context.Context` | |
| `n` | `int` | |
| `goal` | `Goal` | |
| `vars` | `...*Var` | |

**Returns:**
| Type | Description |
|------|-------------|
| `[]map[string]Term` | |

**Example:**

```go
// Example usage of SolutionsCtx
result := SolutionsCtx(/* parameters */)
```

### SolutionsN
SolutionsN runs a goal against a fresh local store and returns up to n solutions projected onto the provided variables. Each solution is a map from variable name to the reified value term. If no vars are provided, an empty string key is used for each result to preserve cardinality.

```go
func SolutionsN(ctx context.Context, n int, goal Goal, vars ...*Var) []map[string]Term
```

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `ctx` | `context.Context` | |
| `n` | `int` | |
| `goal` | `Goal` | |
| `vars` | `...*Var` | |

**Returns:**
| Type | Description |
|------|-------------|
| `[]map[string]Term` | |

**Example:**

```go
// Example usage of SolutionsN
result := SolutionsN(/* parameters */)
```

### Solve
Solve is SolveN with context.Background().

```go
func (*Solver) Solve(ctx context.Context, maxSolutions int) ([][]int, error)
```

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `ctx` | `context.Context` | |
| `maxSolutions` | `int` | |

**Returns:**
| Type | Description |
|------|-------------|
| `[][]int` | |
| `error` | |

**Example:**

```go
// Example usage of Solve
result := Solve(/* parameters */)
```

### SolveN
SolveN solves the model and returns up to maxSolutions solutions using the default sequential solver. For advanced control, use NewSolver(m) directly.

```go
func SolveN(ctx context.Context, m *Model, maxSolutions int) ([][]int, error)
```

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `ctx` | `context.Context` | |
| `m` | `*Model` | |
| `maxSolutions` | `int` | |

**Returns:**
| Type | Description |
|------|-------------|
| `[][]int` | |
| `error` | |

**Example:**

```go
// Example usage of SolveN
result := SolveN(/* parameters */)
```

### Strings
Strings is StringsN with n<=0 (all results).

```go
func Strings(goal Goal, v *Var) []string
```

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `goal` | `Goal` | |
| `v` | `*Var` | |

**Returns:**
| Type | Description |
|------|-------------|
| `[]string` | |

**Example:**

```go
// Example usage of Strings
result := Strings(/* parameters */)
```

### StringsN
StringsN solves for a single variable and returns up to n string values. Non-string bindings are skipped. When n<=0, all results are returned.

```go
func StringsN(ctx context.Context, n int, goal Goal, v *Var) []string
```

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `ctx` | `context.Context` | |
| `n` | `int` | |
| `goal` | `Goal` | |
| `v` | `*Var` | |

**Returns:**
| Type | Description |
|------|-------------|
| `[]string` | |

**Example:**

```go
// Example usage of StringsN
result := StringsN(/* parameters */)
```

### TablePred
TablePred returns a function that builds tabled goals for the given predicateID while accepting native values or Terms.

```go
func TablePred(db *Database, rel *Relation, predicateID string) func(args ...interface{}) Goal
```

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `db` | `*Database` | |
| `rel` | `*Relation` | |
| `predicateID` | `string` | |

**Returns:**
| Type | Description |
|------|-------------|
| `func(args ...interface{}) Goal` | |

**Example:**

```go
// Example usage of TablePred
result := TablePred(/* parameters */)
```

### TabledEvaluate
TabledEvaluate is a convenience wrapper that evaluates a tabled predicate using the global SLG engine. It constructs the CallPattern from the provided predicate identifier and arguments, and runs the supplied evaluator to produce answers that will be cached by the engine.

```go
func TabledEvaluate(ctx context.Context, predicateID string, args []Term, evaluator GoalEvaluator) (<-chan map[int64]Term, error)
```

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `ctx` | `context.Context` | |
| `predicateID` | `string` | |
| `args` | `[]Term` | |
| `evaluator` | `GoalEvaluator` | |

**Returns:**
| Type | Description |
|------|-------------|
| `<-chan map[int64]Term` | |
| `error` | |

**Example:**

```go
// Example usage of TabledEvaluate
result := TabledEvaluate(/* parameters */)
```

### TabledRecursivePredicate
TabledRecursivePredicate builds a true recursive, tabled predicate over a base relation. It returns a predicate constructor that can be called with arguments to form a Goal.

```go
func TabledRecursivePredicate(db *Database, baseRel *Relation, predicateID string, recursive func(self func(...Term) Goal, args ...Term) Goal) func(...Term) Goal
```

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `db` | `*Database` | pldb database |
| `baseRel` | `*Relation` | base relation providing direct facts (e.g., parent for ancestor) |
| `predicateID` | `string` | unique predicate name for tabling (e.g., "ancestor") |
| `recursive` | `func(self func(...Term) Goal, args ...Term) Goal` | function that, given a self predicate for recursive calls and |

**Returns:**
| Type | Description |
|------|-------------|
| `func(...Term) Goal` | |

**Example:**

```go
// Example usage of TabledRecursivePredicate
result := TabledRecursivePredicate(/* parameters */)
```

### TabledRelation
TabledRelation provides a convenient wrapper for creating tabled predicates over pldb relations. It returns a constructor function that builds tabled goals. Example: edge := DbRel("edge", 2, 0, 1) db := NewDatabase() db = db.AddFact(edge, NewAtom("a"), NewAtom("b")) // Create tabled predicate pathPred := TabledRelation(db, edge, "path") // Use it in queries x, y := Fresh("x"), Fresh("y") goal := pathPred(x, y)  // Automatically tabled

```go
func TabledRelation(db *Database, rel *Relation, predicateID string) func(...Term) Goal
```

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `db` | `*Database` | |
| `rel` | `*Relation` | |
| `predicateID` | `string` | |

**Returns:**
| Type | Description |
|------|-------------|
| `func(...Term) Goal` | |

**Example:**

```go
// Example usage of TabledRelation
result := TabledRelation(/* parameters */)
```

### TriplesInts
TriplesInts is TriplesIntsN with n<=0 (all results).

```go
func TriplesInts(goal Goal, x, y, z *Var) [][*ast.BasicLit]int
```

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `goal` | `Goal` | |
| `x` | `*Var` | |
| `y` | `*Var` | |
| `z` | `*Var` | |

**Returns:**
| Type | Description |
|------|-------------|
| `[][*ast.BasicLit]int` | |

**Example:**

```go
// Example usage of TriplesInts
result := TriplesInts(/* parameters */)
```

### TriplesIntsN
TriplesIntsN returns up to n triples of ints for three projected variables. Rows with non-int bindings are skipped.

```go
func TriplesIntsN(ctx context.Context, n int, goal Goal, x, y, z *Var) [][*ast.BasicLit]int
```

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `ctx` | `context.Context` | |
| `n` | `int` | |
| `goal` | `Goal` | |
| `x` | `*Var` | |
| `y` | `*Var` | |
| `z` | `*Var` | |

**Returns:**
| Type | Description |
|------|-------------|
| `[][*ast.BasicLit]int` | |

**Example:**

```go
// Example usage of TriplesIntsN
result := TriplesIntsN(/* parameters */)
```

### TriplesStrings
TriplesStrings is TriplesStringsN with n<=0 (all results).

```go
func TriplesStrings(goal Goal, x, y, z *Var) [][*ast.BasicLit]string
```

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `goal` | `Goal` | |
| `x` | `*Var` | |
| `y` | `*Var` | |
| `z` | `*Var` | |

**Returns:**
| Type | Description |
|------|-------------|
| `[][*ast.BasicLit]string` | |

**Example:**

```go
// Example usage of TriplesStrings
result := TriplesStrings(/* parameters */)
```

### TriplesStringsN
TriplesStringsN returns up to n triples of strings for three projected variables. Rows with non-string bindings are skipped.

```go
func TriplesStringsN(ctx context.Context, n int, goal Goal, x, y, z *Var) [][*ast.BasicLit]string
```

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `ctx` | `context.Context` | |
| `n` | `int` | |
| `goal` | `Goal` | |
| `x` | `*Var` | |
| `y` | `*Var` | |
| `z` | `*Var` | |

**Returns:**
| Type | Description |
|------|-------------|
| `[][*ast.BasicLit]string` | |

**Example:**

```go
// Example usage of TriplesStringsN
result := TriplesStringsN(/* parameters */)
```

### ValuesInt
ValuesInt projects a named value from Solutions(...) into a slice of ints. Missing or non-int entries are skipped.

```go
func ValuesInt(results []map[string]Term, name string) []int
```

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `results` | `[]map[string]Term` | |
| `name` | `string` | |

**Returns:**
| Type | Description |
|------|-------------|
| `[]int` | |

**Example:**

```go
// Example usage of ValuesInt
result := ValuesInt(/* parameters */)
```

### ValuesString
ValuesString projects a named value from Solutions(...) into a slice of strings. Missing or non-string entries are skipped.

```go
func ValuesString(results []map[string]Term, name string) []string
```

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `results` | `[]map[string]Term` | |
| `name` | `string` | |

**Returns:**
| Type | Description |
|------|-------------|
| `[]string` | |

**Example:**

```go
// Example usage of ValuesString
result := ValuesString(/* parameters */)
```

### WithTabling
WithTabling returns a convenience closure bound to the given SLG engine that can be used to evaluate tabled predicates without referencing the engine directly. Example: eval := WithTabling(NewSLGEngine(nil)) ch, err := eval(ctx, "path", []Term{NewAtom("a"), NewAtom("b")}, myEval)

```go
func WithTabling(engine *SLGEngine) func(ctx context.Context, predicateID string, args []Term, evaluator GoalEvaluator) (<-chan map[int64]Term, error)
```

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `engine` | `*SLGEngine` | |

**Returns:**
| Type | Description |
|------|-------------|
| `func(ctx context.Context, predicateID string, args []Term, evaluator GoalEvaluator) (<-chan map[int64]Term, error)` | |

**Example:**

```go
// Example usage of WithTabling
result := WithTabling(/* parameters */)
```

## External Links

- [Package Overview](../packages/minikanren.md)
- [pkg.go.dev Documentation](https://pkg.go.dev/github.com/gitrdm/gokanlogic/pkg/minikanren)
- [Source Code](https://github.com/gitrdm/gokanlogic/tree/master/pkg/minikanren)
