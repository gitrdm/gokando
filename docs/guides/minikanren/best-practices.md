# minikanren Best Practices

Best practices and recommended patterns for using the minikanren package effectively.

## Overview

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


## General Best Practices

### Import and Setup

```go
import "github.com/gitrdm/gokanlogic/pkg/minikanren"

// Always check for errors when initializing
config, err := minikanren.New()
if err != nil {
    log.Fatal(err)
}
```

### Error Handling

Always handle errors returned by minikanren functions:

```go
result, err := minikanren.DoSomething()
if err != nil {
    // Handle the error appropriately
    log.Printf("Error: %v", err)
    return err
}
```

### Resource Management

Ensure proper cleanup of resources:

```go
// Use defer for cleanup
defer resource.Close()

// Or use context for cancellation
ctx, cancel := context.WithCancel(context.Background())
defer cancel()
```

## Package-Specific Patterns

### minikanren Package

#### Using Types

**AbsenceConstraint**

AbsenceConstraint implements the absence constraint (absento). It ensures that a specific term does not occur anywhere within another term's structure, providing structural constraint checking. This constraint performs recursive structural inspection to detect the presence of the forbidden term at any level of nesting.

```go
// Example usage of AbsenceConstraint
// Create a new AbsenceConstraint
absenceconstraint := AbsenceConstraint{
    id: "example",
    absent: Term{},
    container: Term{},
    isLocal: true,
}
```

**AllDifferent**

Implementation uses Regin's arc-consistency algorithm based on maximum bipartite matching. This achieves stronger pruning than pairwise inequality: Example: X,Y,Z ∈ {1,2} with AllDifferent(X,Y,Z) - Matching algorithm detects impossibility (3 variables, 2 values) - Fails immediately without search - Pairwise X≠Y, Y≠Z, X≠Z would only fail after trying assignments Algorithm complexity: O(n²·d) where n = |variables|, d = max domain size Much more efficient than the exponential search that would be required otherwise.

```go
// Example usage of AllDifferent
// Create a new AllDifferent
alldifferent := AllDifferent{
    variables: [],
}
```

**AllDifferentConstraint**

AllDifferentConstraint is a custom version of the all-different constraint This demonstrates how built-in constraints can be reimplemented as custom constraints

```go
// Example usage of AllDifferentConstraint
// Create a new AllDifferentConstraint
alldifferentconstraint := AllDifferentConstraint{
    vars: [],
}
```

**Among**

Among is a global constraint that counts how many variables take values from S.

```go
// Example usage of Among
// Create a new Among
among := Among{
    vars: [],
    set: Domain{},
    k: &FDVariable{}{},
}
```

**AnswerIterator**

AnswerIterator iterates over answers in insertion order.

```go
// Example usage of AnswerIterator
// Create a new AnswerIterator
answeriterator := AnswerIterator{
    snapshot: [],
    idx: 42,
    mu: /* value */,
}
```

**AnswerRecord**

AnswerRecord bundles an answer's bindings with its WFS delay set. If Delay is empty, the answer is unconditional.

```go
// Example usage of AnswerRecord
// Create a new AnswerRecord
answerrecord := AnswerRecord{
    Bindings: map[],
    Delay: DelaySet{},
}
```

**AnswerRecordIterator**

AnswerRecordIterator is a metadata-aware iterator that wraps the existing AnswerIterator and pairs each binding with a DelaySet provided by a callback. The callback allows us to wire per-answer metadata later without changing the current AnswerTrie layout.

```go
// Example usage of AnswerRecordIterator
// Create a new AnswerRecordIterator
answerrecorditerator := AnswerRecordIterator{
    inner: &AnswerIterator{}{},
    startIndex: 42,
    delayProvider: /* value */,
    include: /* value */,
}
```

**AnswerTrie**

AnswerTrie represents a trie of answer substitutions for a tabled subgoal. Uses structural sharing to minimize memory overhead. Thread safety: The trie supports concurrent reads, and writes are coordinated via an internal mutex to ensure safety. Iteration returns copies of stored answers to prevent external mutation. In typical usage, writes are also coordinated at a higher level (e.g., by SubgoalEntry) to avoid unnecessary contention.

```go
// Example usage of AnswerTrie
// Create a new AnswerTrie
answertrie := AnswerTrie{
    root: &AnswerTrieNode{}{},
    answers: [],
    count: /* value */,
    nodePool: &/* value */{},
    mu: /* value */,
}
```

**AnswerTrieNode**

AnswerTrieNode represents a node in the answer trie. Thread safety: children map is protected by the trie's global mutex during writes, and is safe for concurrent reads after insertion since nodes are structurally shared.

```go
// Example usage of AnswerTrieNode
// Create a new AnswerTrieNode
answertrienode := AnswerTrieNode{
    varID: 42,
    value: Term{},
    children: map[],
    isAnswer: true,
    depth: 42,
}
```

**Arithmetic**

Provides bidirectional arc-consistency: - Forward: dst ∈ {src + offset | src ∈ Domain(src)} - Backward: src ∈ {dst - offset | dst ∈ Domain(dst)} Example: X + 3 = Y with X ∈ {1,2,5}, Y ∈ {1,2,3,4,5,6,7,8} - Forward prunes: Y restricted to {4,5,8} - Backward prunes: X restricted to {1,2,5} (no change, already consistent) Useful for modeling derived variables in problems like N-Queens where diagonal constraints are column ± row offset.

```go
// Example usage of Arithmetic
// Create a new Arithmetic
arithmetic := Arithmetic{
    src: &FDVariable{}{},
    dst: &FDVariable{}{},
    offset: 42,
}
```

**Atom**

Atom represents an atomic value (symbol, number, string, etc.). Atoms are immutable and represent themselves.

```go
// Example usage of Atom
// Create a new Atom
atom := Atom{
    value: /* value */,
}
```

**BinPacking**



```go
// Example usage of BinPacking
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

**BitSet**

Generic BitSet-backed Domain for FD variables. Values are 1-based indices.

```go
// Example usage of BitSet
// Create a new BitSet
bitset := BitSet{
    n: 42,
    words: [],
}
```

**BitSetDomain**

Values are 1-indexed in the range [1, maxValue]. Each value is represented by a single bit in a uint64 word array, providing O(1) membership testing and very fast set operations. Memory usage: (maxValue + 63) / 64 * 8 bytes Example: maxValue=100 uses 16 bytes (2 uint64 words) BitSetDomain is immutable - all operations return new instances rather than modifying in place. This enables efficient structural sharing and copy-on-write semantics for parallel search.

```go
// Example usage of BitSetDomain
// Create a new BitSetDomain
bitsetdomain := BitSetDomain{
    maxValue: 42,
    words: [],
}
```

**BoolSum**

Propagation: - Let lb = sum of per-var minimum contributions (1 if var must be true, else 0) - Let ub = sum of per-var maximum contributions (1 if var may be true, else 0) - Prune total to [lb+1, ub+1] - For each var, using otherLb = lb - varMin and otherUb = ub - varMax: - If (total.min-1) > otherUb  => var must be true (set to {2}) - If (total.max-1) < otherLb  => var must be false (set to {1}) This achieves bounds consistency for boolean sums and is sufficient for Count.

```go
// Example usage of BoolSum
// Create a new BoolSum
boolsum := BoolSum{
    vars: [],
    total: &FDVariable{}{},
}
```

**BoundsSum**

Constrains: sum(vars) = total Bounds propagation: - total.min >= sum(vars[i].min) - total.max <= sum(vars[i].max) - For each var[i]: var[i].min >= total.min - sum(vars[j!=i].max) - For each var[i]: var[i].max <= total.max - sum(vars[j!=i].min) This is a simplified version sufficient for counting with 0/1 variables. A full Sum constraint would support coefficients and inequalities.

```go
// Example usage of BoundsSum
// Create a new BoundsSum
boundssum := BoundsSum{
    vars: [],
    total: &FDVariable{}{},
}
```

**CallPattern**

CallPattern represents a normalized subgoal call for use as a tabling key. CallPatterns must be comparable and efficiently hashable. The pattern abstracts away specific variable identities, replacing them with canonical positions (e.g., "path(X0, X1)" instead of "path(_42, _73)"). This allows different calls with the same structure to share cached answers. Thread safety: CallPattern is immutable after creation.

```go
// Example usage of CallPattern
// Create a new CallPattern
callpattern := CallPattern{
    predicateID: "example",
    argStructure: "example",
    hashValue: 42,
}
```

**Circuit**

Circuit is a composite global constraint that owns auxiliary variables and reified constraints to enforce a single Hamiltonian circuit over successors. The Propagate method itself does no work; all pruning is done by the posted sub-constraints. This mirrors the Count and ElementValues pattern.

```go
// Example usage of Circuit
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

**Constraint**

Constraint represents a logical constraint that can be checked against variable bindings. Constraints are the core abstraction that enables order-independent constraint logic programming. Constraints must be thread-safe as they may be checked concurrently during parallel goal evaluation.

```go
// Example usage of Constraint
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

**ConstraintEvent**

ConstraintEvent represents a notification about constraint-related activities. Used for coordinating between local stores and the global constraint bus.

```go
// Example usage of ConstraintEvent
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

**ConstraintEventType**

ConstraintEventType categorizes different kinds of constraint events for efficient processing by the global constraint bus.

```go
// Example usage of ConstraintEventType
// Example usage of ConstraintEventType
var value ConstraintEventType
// Initialize with appropriate value
```

**ConstraintResult**

ConstraintResult represents the outcome of evaluating a constraint. Constraints can be satisfied (no violation), violated (goal should fail), or pending (waiting for more variable bindings).

```go
// Example usage of ConstraintResult
// Example usage of ConstraintResult
var value ConstraintResult
// Initialize with appropriate value
```

**ConstraintStore**

ConstraintStore represents a collection of constraints and variable bindings. This interface abstracts over both local and global constraint storage.

```go
// Example usage of ConstraintStore
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

**ConstraintViolationError**

ConstraintViolationError represents an error caused by constraint violations. It provides detailed information about which constraint was violated and why.

```go
// Example usage of ConstraintViolationError
// Create a new ConstraintViolationError
constraintviolationerror := ConstraintViolationError{
    Constraint: Constraint{},
    Bindings: map[],
    Message: "example",
}
```

**Count**

- Reified constraints prune variable domains based on boolean values - Sum constraint propagates bounds on countVar - Boolean domains drive further pruning on vars Example: Count([X,Y,Z], 5, N) with X,Y,Z ∈ {1..10}, N ∈ {0..3} - If X=5, Y=5 → N ∈ {2,3} (at least 2 equal 5) - If N=0 → X,Y,Z ≠ 5 - If N=3 → X=Y=Z=5 Complexity: O(n) propagation per variable domain change, where n = len(vars)

```go
// Example usage of Count
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

**Cumulative**

Cumulative models a single renewable resource with fixed capacity consumed by a set of tasks with fixed durations and demands.

```go
// Example usage of Cumulative
// Create a new Cumulative
cumulative := Cumulative{
    starts: [],
    durations: [],
    demands: [],
    capacity: 42,
}
```

**CustomConstraint**

fd_custom.go: custom constraint interfaces for FDStore CustomConstraint represents a user-defined constraint that can propagate

```go
// Example usage of CustomConstraint
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

**Database**

Database is an immutable collection of relations and their facts. Operations return new Database instances with copy-on-write semantics.

```go
// Example usage of Database
// Create a new Database
database := Database{
    relations: map[],
    mu: /* value */,
}
```

**DelaySet**

WFS scaffolding: types and iterators to support conditional answers with delay sets. This file introduces minimal, backwards-compatible structures to carry well-founded semantics (WFS) metadata alongside existing answer bindings. It does not change the storage layout of AnswerTrie; instead, it provides an optional metadata-aware iterator that can be wired to a delay provider. DelaySet represents the set of negatively depended-on subgoals (by key/hash) that must be resolved before an answer can be considered unconditional. Keys are the CallPattern hash values of the depended subgoals.

```go
// Example usage of DelaySet
// Example usage of DelaySet
var value DelaySet
// Initialize with appropriate value
```

**Diffn**

Diffn composes reified pairwise non-overlap disjunctions for rectangles.

```go
// Example usage of Diffn
// Create a new Diffn
diffn := Diffn{
    x: [],
    w: [],
    reifs: [],
}
```

**DisequalityConstraint**

DisequalityConstraint implements the disequality constraint (≠). It ensures that two terms are not equal, providing order-independent constraint semantics for the Neq operation. The constraint tracks two terms and checks that they never become equal through unification. If both terms are variables, the constraint remains pending until at least one is bound to a concrete value.

```go
// Example usage of DisequalityConstraint
// Create a new DisequalityConstraint
disequalityconstraint := DisequalityConstraint{
    id: "example",
    term1: Term{},
    isLocal: true,
}
```

**DistinctCount**

DistinctCount composes internal reified equalities and boolean sums to count distinct values among vars. The distinct count is exposed as a variable DPlus1 with the standard encoding: distinctCount = DPlus1 - 1.

```go
// Example usage of DistinctCount
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

**Domain**

Domains support efficient operations for: - Membership testing - Value removal (pruning) - Cardinality queries - Set operations (intersection, union, complement) - Iteration over values Thread safety: Domain implementations must be safe for concurrent read access. Write operations (which return new domains) are inherently safe as they don't modify existing domains.

```go
// Example usage of Domain
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

**ElementValues**

ElementValues is a constraint linking an index variable, a constant table of values, and a result variable such that result = values[index].

```go
// Example usage of ElementValues
// Create a new ElementValues
elementvalues := ElementValues{
    index: &FDVariable{}{},
    values: [],
    result: &FDVariable{}{},
}
```

**EqualityReified**

4. B becomes 1 → remove intersection from both domains (enforce X ≠ Y) This provides proper reification semantics for equality, handling both "constraint must be true" and "constraint must be false" cases correctly. Implementation achieves arc-consistency through: - When B=2: X.domain ← X.domain ∩ Y.domain (and vice versa) - When B=1: for each value v: if v ∈ X.domain and Y.domain={v}, remove v from X - Singleton detection: if X and Y are singletons, set B accordingly - Disjoint detection: if X.domain ∩ Y.domain = ∅, set B=1

```go
// Example usage of EqualityReified
// Create a new EqualityReified
equalityreified := EqualityReified{
    x: &FDVariable{}{},
    y: &FDVariable{}{},
    boolVar: &FDVariable{}{},
}
```

**FDChange**

Extend FDVar with offset links (placed here to avoid changing many other files) Note: we keep it unexported and simple; propagation logic in FDStore will consult these. We'll attach via a small map in FDStore to avoid changing serialized layout of FDVar across code paths. FDChange represents a single domain change for undo

```go
// Example usage of FDChange
// Create a new FDChange
fdchange := FDChange{
    vid: 42,
    domain: BitSet{},
}
```

**FDPlugin**

- PropagationConstraints: prune domains based on constraint semantics During propagation, the FDPlugin: 1. Extracts FD domains from the UnifiedStore 2. Builds a temporary SolverState representing those domains 3. Runs FD propagation constraints to fixed point 4. Extracts pruned domains back into a new UnifiedStore This allows the FD solver to participate in hybrid solving without modifying its core architecture.

```go
// Example usage of FDPlugin
// Create a new FDPlugin
fdplugin := FDPlugin{
    model: &Model{}{},
    solver: &Solver{}{},
}
```

**FDStore**

- Offset arithmetic constraints for modeling relationships - Iterative backtracking with dom/deg heuristics - Context-aware cancellation and timeouts Typical usage: store := NewFDStoreWithDomain(maxValue) vars := store.MakeFDVars(n) // Add constraints... solutions, err := store.Solve(ctx, limit)

```go
// Example usage of FDStore
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

**FDVar**

FDVar is a finite-domain variable

```go
// Example usage of FDVar
// Create a new FDVar
fdvar := FDVar{
    ID: 42,
    domain: BitSet{},
    peers: [],
}
```

**FDVariable**

FDVariable represents a finite-domain constraint variable. This is the standard variable type for finite-domain CSPs like Sudoku, N-Queens, scheduling, and resource allocation problems. FDVariable stores the initial domain. During solving, the Solver uses the variable's ID to track current domains in SolverState via copy-on-write. This separation enables: - Model immutability (can be shared by parallel workers) - Efficient O(1) state updates (only modified domains are tracked) - Lock-free parallel search (each worker has its own SolverState chain)

```go
// Example usage of FDVariable
// Create a new FDVariable
fdvariable := FDVariable{
    id: 42,
    domain: Domain{},
    name: "example",
}
```

**Fact**

Fact represents a single row in a relation. Facts must be ground (contain only atoms, no variables). Facts are immutable after creation.

```go
// Example usage of Fact
// Create a new Fact
fact := Fact{
    terms: [],
    hash: 42,
}
```

**FactsSpec**

FactsSpec describes facts for a relation for bulk loading.

```go
// Example usage of FactsSpec
// Create a new FactsSpec
factsspec := FactsSpec{
    Rel: &Relation{}{},
    Rows: [],
}
```

**GlobalCardinality**

GlobalCardinality constrains occurrence counts per value across variables.

```go
// Example usage of GlobalCardinality
// Create a new GlobalCardinality
globalcardinality := GlobalCardinality{
    vars: [],
    minCount: [],
    maxCount: [],
    maxValue: 42,
}
```

**GlobalConstraintBus**

GlobalConstraintBus coordinates constraint checking across multiple local constraint stores. It handles cross-store constraints and provides a coordination point for complex constraint interactions. The bus is designed to minimize coordination overhead - most constraints should be local and not require global coordination.

```go
// Example usage of GlobalConstraintBus
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

**GlobalConstraintBusPool**

GlobalConstraintBusPool manages a pool of reusable constraint buses

```go
// Example usage of GlobalConstraintBusPool
// Create a new GlobalConstraintBusPool
globalconstraintbuspool := GlobalConstraintBusPool{
    pool: /* value */,
}
```

**Goal**

Goal represents a constraint or a combination of constraints. Goals are functions that take a constraint store and return a stream of constraint stores representing all possible ways to satisfy the goal. Goals can be composed to build complex relational programs. The constraint store contains both variable bindings and active constraints, enabling order-independent constraint logic programming.

```go
// Example usage of Goal
// Example usage of Goal
var value Goal
// Initialize with appropriate value
```

**GoalEvaluator**

GoalEvaluator is a function that evaluates a goal and returns answer bindings. It's called by the SLG engine to produce answers for a tabled subgoal. The evaluator should: - Yield answer bindings via the channel - Close the channel when done - Respect context cancellation - Return any error encountered

```go
// Example usage of GoalEvaluator
// Example usage of GoalEvaluator
var value GoalEvaluator
// Initialize with appropriate value
```

**HybridRegistry**

variable spaces, eliminating boilerplate code in hybrid queries. Usage Pattern: 1. Create registry with NewHybridRegistry() 2. Register variable pairs with MapVars(relVar, fdVar) 3. Execute hybrid query producing bindings 4. Apply bindings with AutoBind(result, store) Thread Safety: Registry instances are immutable. All operations return new registry instances, making them safe for concurrent use.

```go
// Example usage of HybridRegistry
// Create a new HybridRegistry
hybridregistry := HybridRegistry{
    relToFD: map[],
    fdToRel: map[],
}
```

**HybridSolver**

3. The process repeats until no plugin makes further changes (fixed point) 4. If any plugin detects a conflict, solving backtracks Configuration options control: - Maximum propagation iterations (prevent infinite loops) - Plugin execution order (can affect performance) - Timeout and solution limits Thread safety: HybridSolver is safe for concurrent use. Multiple solvers can work on different search branches simultaneously.

```go
// Example usage of HybridSolver
// Create a new HybridSolver
hybridsolver := HybridSolver{
    plugins: [],
    config: &HybridSolverConfig{}{},
}
```

**HybridSolverConfig**

HybridSolverConfig configures the hybrid solver's behavior.

```go
// Example usage of HybridSolverConfig
// Create a new HybridSolverConfig
hybridsolverconfig := HybridSolverConfig{
    MaxPropagationIterations: 42,
    EnablePropagation: true,
}
```

**InSetReified**



```go
// Example usage of InSetReified
// Create a new InSetReified
insetreified := InSetReified{
    v: &FDVariable{}{},
    set: [],
    boolVar: &FDVariable{}{},
}
```

**Inequality**

But checking every X value against Y requires O(|X| × |Y|) operations When to use: - Ordering constraints in scheduling, resource allocation - Combined with search (which provides the final consistency check) - When domain sizes are large and efficiency matters When NOT to use: - When you need guaranteed arc-consistency (use AllDifferent or custom constraints) - When domains are tiny (arc-consistency overhead is negligible)

```go
// Example usage of Inequality
// Create a new Inequality
inequality := Inequality{
    x: &FDVariable{}{},
    y: &FDVariable{}{},
    kind: InequalityKind{},
}
```

**InequalityKind**

InequalityKind specifies the type of inequality.

```go
// Example usage of InequalityKind
// Example usage of InequalityKind
var value InequalityKind
// Initialize with appropriate value
```

**InequalityType**

fd_ineq.go: arithmetic inequality constraints for FDStore InequalityType represents the type of inequality constraint

```go
// Example usage of InequalityType
// Example usage of InequalityType
var value InequalityType
// Initialize with appropriate value
```

**LessEqualConstraint**

LessEqualConstraint represents a constraint that x <= y.

```go
// Example usage of LessEqualConstraint
// Create a new LessEqualConstraint
lessequalconstraint := LessEqualConstraint{
    id: "example",
    x: Term{},
    y: Term{},
}
```

**LessThanConstraint**

LessThanConstraint represents a constraint that x < y. It is evaluated whenever variables become bound.

```go
// Example usage of LessThanConstraint
// Create a new LessThanConstraint
lessthanconstraint := LessThanConstraint{
    id: "example",
    x: Term{},
    y: Term{},
}
```

**Lexicographic**

Lexicographic orders two equal-length vectors of variables.

```go
// Example usage of Lexicographic
// Create a new Lexicographic
lexicographic := Lexicographic{
    xs: [],
    ys: [],
    kind: lexKind{},
}
```

**LinearSum**

LinearSum is a bounds-consistent weighted sum constraint: Σ a[i]*x[i] = t

```go
// Example usage of LinearSum
// Create a new LinearSum
linearsum := LinearSum{
    vars: [],
    coeffs: [],
    total: &FDVariable{}{},
}
```

**LocalConstraintStore**

LocalConstraintStore interface defines the operations needed by the GlobalConstraintBus to coordinate with local stores.

```go
// Example usage of LocalConstraintStore
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

**LocalConstraintStoreImpl**

LocalConstraintStoreImpl provides a concrete implementation of LocalConstraintStore for managing constraints and variable bindings within a single goal context. The store maintains two separate collections: - Local constraints: Checked quickly without global coordination - Local bindings: Variable-to-term mappings for this context When constraints or bindings are added, the store first checks all local constraints for immediate violations, then coordinates with the global bus if necessary for cross-store constraints.

```go
// Example usage of LocalConstraintStoreImpl
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

**MaxOfArray**

MaxOfArray enforces R = max(vars) with bounds-consistent pruning.

```go
// Example usage of MaxOfArray
// Create a new MaxOfArray
maxofarray := MaxOfArray{
    vars: [],
    r: &FDVariable{}{},
}
```

**MembershipConstraint**

MembershipConstraint implements the membership constraint (membero). It ensures that an element is a member of a list, providing relational list membership checking that can work in both directions.

```go
// Example usage of MembershipConstraint
// Create a new MembershipConstraint
membershipconstraint := MembershipConstraint{
    id: "example",
    element: Term{},
    list: Term{},
    isLocal: true,
}
```

**MinOfArray**

MinOfArray enforces R = min(vars) with bounds-consistent pruning.

```go
// Example usage of MinOfArray
// Create a new MinOfArray
minofarray := MinOfArray{
    vars: [],
    r: &FDVariable{}{},
}
```

**Model**

- Variables: decision variables with finite domains - Constraints: relationships that must hold among variables - Configuration: solver parameters and search heuristics Models are constructed incrementally by adding variables and constraints. Once constructed, models are immutable during solving, enabling safe concurrent access by parallel search workers. Thread safety: Models are safe for concurrent reads during solving, but must be constructed sequentially.

```go
// Example usage of Model
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

**ModelConstraint**

ModelConstraint represents a constraint within a model. Constraints restrict the values that variables can take simultaneously. Different constraint types provide different propagation strength: - AllDifferent: ensures variables take distinct values - Arithmetic: enforces arithmetic relationships (x + y = z) - Table: extensional constraints defined by allowed tuples - Global: specialized algorithms for common patterns ModelConstraints are immutable after creation and safe for concurrent access.

```go
// Example usage of ModelConstraint
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

**OptimizeOption**

OptimizeOption configures SolveOptimalWithOptions behavior. Use helpers like WithTimeLimit, WithNodeLimit, WithTargetObjective, WithParallelWorkers, and WithHeuristics to customize the search.

```go
// Example usage of OptimizeOption
// Example usage of OptimizeOption
var value OptimizeOption
// Initialize with appropriate value
```

**Pair**

Pair represents a cons cell (pair) in miniKanren. Pairs are used to build lists and other compound structures.

```go
// Example usage of Pair
// Create a new Pair
pair := Pair{
    car: Term{},
    cdr: Term{},
    mu: /* value */,
}
```

**ParallelConfig**

ParallelConfig holds configuration for parallel goal execution.

```go
// Example usage of ParallelConfig
// Create a new ParallelConfig
parallelconfig := ParallelConfig{
    MaxWorkers: 42,
    MaxQueueSize: 42,
    EnableBackpressure: true,
    RateLimit: 42,
}
```

**ParallelExecutor**

ParallelExecutor manages parallel execution of miniKanren goals.

```go
// Example usage of ParallelExecutor
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

**ParallelSearchConfig**

ParallelSearchConfig holds configuration for parallel backtracking search.

```go
// Example usage of ParallelSearchConfig
// Create a new ParallelSearchConfig
parallelsearchconfig := ParallelSearchConfig{
    NumWorkers: 42,
    WorkQueueSize: 42,
}
```

**ParallelStream**

ParallelStream represents a stream that can be evaluated in parallel. It wraps the standard Stream with additional parallel capabilities.

```go
// Example usage of ParallelStream
// Create a new ParallelStream
parallelstream := ParallelStream{
    executor: &ParallelExecutor{}{},
    ctx: /* value */,
}
```

**PatternClause**

PatternClause represents a single pattern matching clause. Each clause consists of a pattern term and a sequence of goals to execute if the pattern matches. The pattern is unified with the input term. If unification succeeds, the goals are executed in sequence (as if by Conj).

```go
// Example usage of PatternClause
// Create a new PatternClause
patternclause := PatternClause{
    Pattern: Term{},
    Goals: [],
}
```

**PropagationConstraint**

PropagationConstraint extends ModelConstraint with active domain pruning. This interface bridges the declarative ModelConstraint with the propagation engine. Propagation maintains copy-on-write semantics: constraints never modify state in-place but return a new state with pruned domains. This preserves the lock-free property critical for parallel search.

```go
// Example usage of PropagationConstraint
// Example implementation of PropagationConstraint
type MyPropagationConstraint struct {
    // Add your fields here
}

func (m MyPropagationConstraint) Propagate(param1 *Solver, param2 *SolverState) *SolverState {
    // Implement your logic here
    return
}


```

**Rational**

This enables exact representation of fractional coefficients without floating-point errors. Common irrational approximations: π ≈ 22/7 (Archimedes, error ~0.04%) π ≈ 355/113 (Zu Chongzhi, error ~0.000008%) √2 ≈ 99/70 (accurate to 4 decimals) √2 ≈ 1393/985 (accurate to 6 decimals) e ≈ 2721/1000 (accurate to 4 decimals) φ (golden ratio) ≈ 1618/1000 (accurate to 3 decimals)

```go
// Example usage of Rational
// Create a new Rational
rational := Rational{
    Num: 42,
    Den: 42,
}
```

**RationalLinearSum**

Scaled: 2*x + 3*y = 6*z This enables exact rational coefficient constraints while leveraging existing integer domain infrastructure and propagation algorithms. Use cases: - Irrational approximations: π*diameter = circumference → (22/7)*d = c - Percentage calculations: 10% bonus → (1/10)*salary = bonus - Unit conversions with fractional ratios: (5/9)*(F-32) = C - Recipe scaling: (3/4)*flour + (1/2)*sugar = mixture

```go
// Example usage of RationalLinearSum
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

**Regular**

Regular is the DFA-based global constraint over a sequence of variables.

```go
// Example usage of Regular
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

**ReifiedConstraint**

4. When boolean = 1 → ensure constraint is violated (complex, often via search) For simplicity, this implementation focuses on cases 1–3. Case 4 (forcing a constraint to be false) is challenging and often requires specialized negation logic per constraint type. We handle it by: - If boolean is bound to 1 (false), we skip constraint propagation - The search will naturally find assignments that violate the constraint This is sound but may be weaker than full constraint negation. For many use cases (including Count built via equality reification), this is sufficient.

```go
// Example usage of ReifiedConstraint
// Create a new ReifiedConstraint
reifiedconstraint := ReifiedConstraint{
    constraint: PropagationConstraint{},
    boolVar: &FDVariable{}{},
}
```

**Relation**

Relation represents a named relation with a fixed arity and indexed columns. Relations are immutable after creation.

```go
// Example usage of Relation
// Create a new Relation
relation := Relation{
    name: "example",
    arity: 42,
    indexes: map[],
}
```

**RelationalPlugin**

1. Extracts relational bindings from the UnifiedStore 2. Checks each Constraint against those bindings 3. Returns error if any constraint is violated 4. Returns original store if all constraints are satisfied or pending The relational plugin doesn't typically modify the store (no pruning), it just validates that current bindings don't violate constraints. However, if FD domains narrow variables to singletons, those singleton values can be promoted to relational bindings, enabling cross-solver propagation.

```go
// Example usage of RelationalPlugin
// Create a new RelationalPlugin
relationalplugin := RelationalPlugin{

}
```

**SCC**

SCC represents a strongly connected component in the dependency graph. Used for cycle detection and fixpoint computation.

```go
// Example usage of SCC
// Create a new SCC
scc := SCC{
    nodes: [],
    index: 42,
}
```

**SLGConfig**

SLGConfig holds configuration for the SLG engine.

```go
// Example usage of SLGConfig
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

**SLGEngine**

SLGEngine coordinates tabled goal evaluation using SLG resolution. The engine maintains a global SubgoalTable shared across all evaluations, enabling answer reuse and cycle detection. Multiple goroutines can safely evaluate different goals concurrently. Thread safety: SLGEngine is safe for concurrent use by multiple goroutines.

```go
// Example usage of SLGEngine
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

**SLGStats**

SLGStats provides statistics about engine performance.

```go
// Example usage of SLGStats
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

**ScaledDivision**

- Backward propagation: dividend ⊆ {q*divisor...(q+1)*divisor-1 | q ∈ quotient.domain} This is arc-consistent propagation suitable for AC-3 and fixed-point iteration. Invariants: - divisor > 0 (enforced at construction) - All variables must have non-nil domains - Empty domain → immediate failure Thread Safety: Immutable after construction. Propagate() is safe for concurrent use.

```go
// Example usage of ScaledDivision
// Create a new ScaledDivision
scaleddivision := ScaledDivision{
    dividend: &FDVariable{}{},
    divisor: 42,
    quotient: &FDVariable{}{},
}
```

**Sequence**



```go
// Example usage of Sequence
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

**Solver**

- Smart backtracking with conflict-driven learning (future) The solver is designed for both sequential and parallel execution. State is immutable during search, with modifications creating lightweight derived states that share structure with their parent. Thread safety: Solver instances are NOT thread-safe. For parallel search, create multiple Solver instances that share the same immutable Model but maintain independent SolverState chains. This is zero-cost as the Model is read-only and domains are immutable.

```go
// Example usage of Solver
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

**SolverConfig**

SolverConfig holds configuration for the FD solver

```go
// Example usage of SolverConfig
// Create a new SolverConfig
solverconfig := SolverConfig{
    VariableHeuristic: VariableOrderingHeuristic{},
    ValueHeuristic: ValueOrderingHeuristic{},
    RandomSeed: 42,
}
```

**SolverMonitor**

SolverMonitor provides lock-free monitoring capabilities for the FD solver. All operations use atomic instructions for safe concurrent access without locks. Designed to match the lock-free copy-on-write architecture of the solver.

```go
// Example usage of SolverMonitor
// Create a new SolverMonitor
solvermonitor := SolverMonitor{
    stats: SolverStats{},
    startTime: /* value */,
    propStart: /* value */,
}
```

**SolverPlugin**

UnifiedStore containing both relational bindings and FD domains. Each plugin is responsible for: - Identifying which constraints it can handle - Propagating those constraints to prune the search space - Communicating changes through the UnifiedStore Plugins must be thread-safe as they may be called concurrently during parallel search. They must also maintain the copy-on-write semantics required for lock-free operation: all state changes return new store versions.

```go
// Example usage of SolverPlugin
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

**SolverState**

1. Constraint sees x={5} via GetDomain(State3, x.ID) 2. Constraint narrows y: y={2,3} (remove 5) 3. Creates State4: y={2,3} (parent: State3) 4. Constraint narrows z: z={1,2,3} (5 not present, no change) 5. Returns State4 (fixed point reached) Constraints "communicate" by reading current domains via GetDomain and creating new states via SetDomain. The state chain captures all changes. States are pooled and reused to minimize GC pressure.

```go
// Example usage of SolverState
// Create a new SolverState
solverstate := SolverState{
    parent: &SolverState{}{},
    modifiedVarID: 42,
    modifiedDomain: Domain{},
    depth: 42,
    refCount: /* value */,
}
```

**SolverStats**

SolverStats holds statistics about the FD solving process. All fields use atomic operations for lock-free updates.

```go
// Example usage of SolverStats
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

**Stream**

Stream represents a (potentially infinite) sequence of constraint stores. Streams are the core data structure for representing multiple solutions in miniKanren. Each constraint store contains variable bindings and active constraints representing a consistent logical state. This implementation uses channels for thread-safe concurrent access and supports parallel evaluation with proper constraint coordination.

```go
// Example usage of Stream
// Create a new Stream
stream := Stream{
    ch: /* value */,
    done: /* value */,
    mu: /* value */,
}
```

**Stretch**

Stretch is a thin wrapper around the constructed Regular constraint to expose the high-level intent and variables involved.

```go
// Example usage of Stretch
// Create a new Stretch
stretch := Stretch{
    vars: [],
    values: [],
    minByValue: map[],
    maxByValue: map[],
    dfa: &Regular{}{},
}
```

**SubgoalEntry**

SubgoalEntry represents a tabled subgoal with its cached answers. Thread safety: - Status is accessed atomically - Answer trie supports concurrent read/write - Dependencies protected by RWMutex - Condition variable for producer/consumer synchronization

```go
// Example usage of SubgoalEntry
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

**SubgoalStatus**

SubgoalStatus represents the evaluation state of a tabled subgoal.

```go
// Example usage of SubgoalStatus
// Example usage of SubgoalStatus
var value SubgoalStatus
// Initialize with appropriate value
```

**SubgoalTable**

SubgoalTable manages all tabled subgoals using a concurrent map. Thread safety: Uses sync.Map for lock-free concurrent access. The map is read-heavy (many lookups, few insertions), making sync.Map ideal.

```go
// Example usage of SubgoalTable
// Create a new SubgoalTable
subgoaltable := SubgoalTable{
    entries: /* value */,
    totalSubgoals: /* value */,
}
```

**Substitution**

Substitution represents a mapping from variables to terms. It's used to track bindings during unification and goal evaluation. The implementation is thread-safe and supports concurrent access.

```go
// Example usage of Substitution
// Create a new Substitution
substitution := Substitution{
    bindings: map[],
    mu: /* value */,
}
```

**SumConstraint**

Example custom constraint implementations SumConstraint enforces that the sum of variables equals a target value

```go
// Example usage of SumConstraint
// Create a new SumConstraint
sumconstraint := SumConstraint{
    vars: [],
    target: 42,
}
```

**Table**

Table is an extensional constraint over a fixed list of allowed tuples.

```go
// Example usage of Table
// Create a new Table
table := Table{
    vars: [],
    rows: [],
}
```

**TabledDatabase**

from a database. This is useful for applications where all queries should be cached. Example: db := NewDatabase() // ... add facts ... tdb := WithTabledDatabase(db, "mydb") // All queries are automatically tabled goal := tdb.Query(edge, x, y)

```go
// Example usage of TabledDatabase
// Create a new TabledDatabase
tableddatabase := TabledDatabase{
    db: &Database{}{},
    idPrefix: "example",
}
```

**Term**

Term represents any value in the miniKanren universe. Terms can be atoms, variables, compound structures, or any Go value. All Term implementations must be comparable and thread-safe.

```go
// Example usage of Term
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

**TruthValue**

TruthValue represents the three-valued logic outcomes under WFS. For negation-as-failure over a subgoal G, the truth of not(G) is: - True:     G completes with no answers - False:    G produces at least one answer - Undefined: G is incomplete (conditional)

```go
// Example usage of TruthValue
// Example usage of TruthValue
var value TruthValue
// Initialize with appropriate value
```

**TypeConstraint**

TypeConstraint implements type-based constraints (symbolo, numbero, etc.). It ensures that a term has a specific type, enabling type-safe relational programming patterns.

```go
// Example usage of TypeConstraint
// Create a new TypeConstraint
typeconstraint := TypeConstraint{
    id: "example",
    term: Term{},
    expectedType: TypeConstraintKind{},
    isLocal: true,
}
```

**TypeConstraintKind**

TypeConstraintKind represents the different types that can be constrained.

```go
// Example usage of TypeConstraintKind
// Example usage of TypeConstraintKind
var value TypeConstraintKind
// Initialize with appropriate value
```

**UnifiedStore**

- State branching for parallel workers is O(1) - Memory overhead is O(changes) not O(total state) Store operations: - Relational: AddBinding(), GetBinding(), GetSubstitution() - Finite-domain: SetDomain(), GetDomain() - Cross-solver: NotifyChange() for propagation triggering Thread safety: UnifiedStore is immutable. All modification methods return new instances, making concurrent reads safe without locks.

```go
// Example usage of UnifiedStore
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

**UnifiedStoreAdapter**

1. Create adapter wrapping a UnifiedStore 2. Use adapter as ConstraintStore in goals (pldb queries, unification, etc.) 3. Extract UnifiedStore for hybrid propagation 4. Update adapter with propagated store 5. Clone adapter for search branching Performance notes: - Adapter overhead is minimal (single pointer dereference + mutex in write path) - UnifiedStore's copy-on-write means cloning is O(1) - Constraint checking delegates to UnifiedStore's constraint system

```go
// Example usage of UnifiedStoreAdapter
// Create a new UnifiedStoreAdapter
unifiedstoreadapter := UnifiedStoreAdapter{
    store: &UnifiedStore{}{},
    mu: /* value */,
    id: "example",
}
```

**ValueEqualsReified**

ValueEqualsReified links a variable v and a boolean b such that b=2 iff v==target. Domain conventions: b ∈ {1=false, 2=true}

```go
// Example usage of ValueEqualsReified
// Create a new ValueEqualsReified
valueequalsreified := ValueEqualsReified{
    v: &FDVariable{}{},
    target: 42,
    boolVar: &FDVariable{}{},
}
```

**ValueOrderingHeuristic**

ValueOrderingHeuristic defines strategies for ordering values within a domain

```go
// Example usage of ValueOrderingHeuristic
// Example usage of ValueOrderingHeuristic
var value ValueOrderingHeuristic
// Initialize with appropriate value
```

**Var**

Var represents a logic variable in miniKanren. Variables can be bound to values through unification. Each variable has a unique identifier to distinguish it from others.

```go
// Example usage of Var
// Create a new Var
var := Var{
    id: 42,
    name: "example",
    mu: /* value */,
}
```

**Variable**

Variable represents a decision variable in a constraint satisfaction problem. Variables have identities, domains of possible values, and participate in constraints. The Variable abstraction allows the solver to be agnostic to the underlying domain representation, enabling different domain types (finite domains, intervals, sets, etc.) to coexist in the same model. Variables in the Model hold initial domains and are immutable once solving begins. During solving, the Solver tracks domain changes via SolverState using the variable's ID.

```go
// Example usage of Variable
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

**VariableOrderingHeuristic**

VariableOrderingHeuristic defines strategies for selecting the next variable to assign

```go
// Example usage of VariableOrderingHeuristic
// Example usage of VariableOrderingHeuristic
var value VariableOrderingHeuristic
// Initialize with appropriate value
```

**VersionInfo**

VersionInfo provides detailed version information.

```go
// Example usage of VersionInfo
// Create a new VersionInfo
versioninfo := VersionInfo{
    Version: "example",
    GoVersion: "example",
    GitCommit: "example",
    BuildDate: "example",
}
```

#### Using Functions

**AsInt**

AsInt attempts to extract an int from a reified Term (Atom). Returns false on mismatch.

```go
// Example usage of AsInt
result := AsInt(/* parameters */)
```

**AsString**

AsString attempts to extract a string from a reified Term (Atom).

```go
// Example usage of AsString
result := AsString(/* parameters */)
```

**FormatSolutions**

FormatSolutions pretty-prints a slice of solutions for human-friendly output. Each solution is rendered as "name: value, name2: value2" with lists and strings formatted pleasantly. Output is sorted for stable tests.

```go
// Example usage of FormatSolutions
result := FormatSolutions(/* parameters */)
```

**FormatTerm**

FormatTerm returns the canonical human-friendly string for a reified Term. It mirrors the formatting used by FormatSolutions: - Empty list: () - Proper lists: (a b c) - Improper lists: (a b . tail) - Strings quoted; other atoms via fmt %%v

```go
// Example usage of FormatTerm
result := FormatTerm(/* parameters */)
```

**GetVersion**

GetVersion returns the current version string.

```go
// Example usage of GetVersion
result := GetVersion(/* parameters */)
```

**Ints**

Ints is IntsN with n<=0 (all results).

```go
// Example usage of Ints
result := Ints(/* parameters */)
```

**IntsN**

IntsN solves for a single variable and returns up to n integer values. Non-int bindings are skipped. When n<=0, all results are returned.

```go
// Example usage of IntsN
result := IntsN(/* parameters */)
```

**InvalidateAll**

InvalidateAll clears the entire SLG answer table. Use this after major database changes when fine-grained invalidation is impractical.

```go
// Example usage of InvalidateAll
result := InvalidateAll(/* parameters */)
```

**InvalidateRelation**

InvalidateRelation removes all cached answers for queries involving a specific relation. This should be called when the relation's facts change (AddFact/RemoveFact). The SLG engine now provides fine-grained predicate-level invalidation, removing only the cached answers for the specified predicateID while preserving unrelated tabled predicates. This is more efficient than clearing the entire table.

```go
// Example usage of InvalidateRelation
result := InvalidateRelation(/* parameters */)
```

**MustInt**

MustInt extracts an int from a Term or panics. Intended for examples/tests.

```go
// Example usage of MustInt
result := MustInt(/* parameters */)
```

**MustString**

MustString extracts a string from a Term or panics.

```go
// Example usage of MustString
result := MustString(/* parameters */)
```

**NewHybridSolverFromModel**

NewHybridSolverFromModel builds a HybridSolver wired for the given model and returns it along with a UnifiedStore pre-populated from the model. The returned solver registers both the Relational and FD plugins in that order which is the common configuration for hybrid solving.

```go
// Example usage of NewHybridSolverFromModel
result := NewHybridSolverFromModel(/* parameters */)
```

**NewRationalLinearSumWithScaling**

NewRationalLinearSumWithScaling creates a RationalLinearSum and handles result scaling automatically. This is a convenience wrapper that uses ScaledDivision when needed (scale > 1). Returns the RationalLinearSum constraint plus an optional ScaledDivision constraint that must also be added to the model. Usage: rls, scaledDiv, err := NewRationalLinearSumWithScaling(vars, coeffs, result, model) model.AddConstraint(rls) if scaledDiv != nil { model.AddConstraint(scaledDiv) } When scale == 1: Returns only RationalLinearSum, scaledDiv is nil When scale > 1: Returns RationalLinearSum with scaled intermediate variable, plus ScaledDivision constraint linking intermediate to result

```go
// Example usage of NewRationalLinearSumWithScaling
result := NewRationalLinearSumWithScaling(/* parameters */)
```

**Optimize**

Optimize finds a solution that optimizes the objective variable. It is a thin wrapper around Solver.SolveOptimal with context.Background().

```go
// Example usage of Optimize
result := Optimize(/* parameters */)
```

**OptimizeWithOptions**

OptimizeWithOptions is like Optimize but accepts a context and solver options for time/node limits or parallel workers. See WithParallelWorkers, WithNodeLimit, and other OptimizeOption helpers.

```go
// Example usage of OptimizeWithOptions
result := OptimizeWithOptions(/* parameters */)
```

**PairsInts**

PairsInts is PairsIntsN with n<=0 (all results).

```go
// Example usage of PairsInts
result := PairsInts(/* parameters */)
```

**PairsIntsN**

PairsIntsN returns up to n pairs of ints for two projected variables. Rows with non-int bindings are skipped.

```go
// Example usage of PairsIntsN
result := PairsIntsN(/* parameters */)
```

**PairsStrings**

PairsStrings is PairsStringsN with n<=0 (all results).

```go
// Example usage of PairsStrings
result := PairsStrings(/* parameters */)
```

**PairsStringsN**

PairsStringsN returns up to n pairs of strings for two projected variables. Rows with non-string bindings are skipped.

```go
// Example usage of PairsStringsN
result := PairsStringsN(/* parameters */)
```

**RecursiveTablePred**

RecursiveTablePred provides a thin HLAPI wrapper around TabledRecursivePredicate. It returns a predicate constructor that accepts native values or Terms when called, converting non-Terms to Atoms automatically. The recursive definition uses the same signature as TabledRecursivePredicate: a callback that receives a self predicate (for recursive calls) and the instantiated call arguments as Terms, and must return the recursive case Goal. The base case over baseRel is handled automatically by the underlying helper. Example: ancestor := RecursiveTablePred(db, parent, "ancestor2", func(self func(...Term) Goal, args ...Term) Goal { x, y := args[0], args[1] z := Fresh("z") return Conj( db.Query(parent, x, z), // base facts used in recursive step self(z, y),              // recursive call to tabled predicate ) }) // Use native values or Terms at call sites goal := ancestor(Fresh("x"), "carol")

```go
// Example usage of RecursiveTablePred
result := RecursiveTablePred(/* parameters */)
```

**ResetGlobalEngine**

ResetGlobalEngine clears the global engine's cache and resets it.

```go
// Example usage of ResetGlobalEngine
result := ResetGlobalEngine(/* parameters */)
```

**ReturnPooledGlobalBus**

ReturnPooledGlobalBus returns a bus to the pool

```go
// Example usage of ReturnPooledGlobalBus
result := ReturnPooledGlobalBus(/* parameters */)
```

**Rows**

Rows is RowsN with n<=0 (all results). WARNING: may not terminate on goals with infinite streams.

```go
// Example usage of Rows
result := Rows(/* parameters */)
```

**RowsAllCtx**

RowsAllCtx returns all rows using the provided context. Use a timeout/cancel to avoid infinite enumeration.

```go
// Example usage of RowsAllCtx
result := RowsAllCtx(/* parameters */)
```

**RowsAllTimeout**

RowsAllTimeout returns all rows but aborts enumeration after the given timeout.

```go
// Example usage of RowsAllTimeout
result := RowsAllTimeout(/* parameters */)
```

**RowsAsInts**

RowsAsInts converts [][]Term rows into [][]int, keeping only rows where all entries are int Atoms. Rows with any non-int terms are skipped.

```go
// Example usage of RowsAsInts
result := RowsAsInts(/* parameters */)
```

**RowsAsStrings**

RowsAsStrings converts [][]Term rows into [][]string, keeping only rows where all entries are string Atoms. Rows with any non-string terms are skipped.

```go
// Example usage of RowsAsStrings
result := RowsAsStrings(/* parameters */)
```

**RowsN**

RowsN runs a goal and returns up to n rows of reified Terms projected in the order of vars provided. Each row corresponds to one solution. If no vars are provided, each row contains a single Atom(nil) to preserve cardinality. When n<=0, all solutions are returned (which may not terminate for infinite goals).

```go
// Example usage of RowsN
result := RowsN(/* parameters */)
```

**SetGlobalEngine**

SetGlobalEngine sets the global SLG engine. This is useful for testing or custom configurations.

```go
// Example usage of SetGlobalEngine
result := SetGlobalEngine(/* parameters */)
```

**Solutions**

Solutions is SolutionsN with n<=0 (all results). WARNING: may not terminate on goals with infinite streams.

```go
// Example usage of Solutions
result := Solutions(/* parameters */)
```

**SolutionsAllCtx**

SolutionsAllCtx returns all solutions (unbounded) using the provided context. Use a context with timeout/cancel to avoid infinite enumeration.

```go
// Example usage of SolutionsAllCtx
result := SolutionsAllCtx(/* parameters */)
```

**SolutionsAllTimeout**

SolutionsAllTimeout returns all solutions but aborts enumeration after the given timeout.

```go
// Example usage of SolutionsAllTimeout
result := SolutionsAllTimeout(/* parameters */)
```

**SolutionsCtx**

SolutionsCtx is an alias for SolutionsN that improves discoverability when passing an explicit context and solution cap together. It returns up to n solutions (n<=0 for all solutions, which may not terminate).

```go
// Example usage of SolutionsCtx
result := SolutionsCtx(/* parameters */)
```

**SolutionsN**

SolutionsN runs a goal against a fresh local store and returns up to n solutions projected onto the provided variables. Each solution is a map from variable name to the reified value term. If no vars are provided, an empty string key is used for each result to preserve cardinality.

```go
// Example usage of SolutionsN
result := SolutionsN(/* parameters */)
```

**Solve**

Solve is SolveN with context.Background().

```go
// Example usage of Solve
result := Solve(/* parameters */)
```

**SolveN**

SolveN solves the model and returns up to maxSolutions solutions using the default sequential solver. For advanced control, use NewSolver(m) directly.

```go
// Example usage of SolveN
result := SolveN(/* parameters */)
```

**Strings**

Strings is StringsN with n<=0 (all results).

```go
// Example usage of Strings
result := Strings(/* parameters */)
```

**StringsN**

StringsN solves for a single variable and returns up to n string values. Non-string bindings are skipped. When n<=0, all results are returned.

```go
// Example usage of StringsN
result := StringsN(/* parameters */)
```

**TablePred**

TablePred returns a function that builds tabled goals for the given predicateID while accepting native values or Terms.

```go
// Example usage of TablePred
result := TablePred(/* parameters */)
```

**TabledEvaluate**

TabledEvaluate is a convenience wrapper that evaluates a tabled predicate using the global SLG engine. It constructs the CallPattern from the provided predicate identifier and arguments, and runs the supplied evaluator to produce answers that will be cached by the engine.

```go
// Example usage of TabledEvaluate
result := TabledEvaluate(/* parameters */)
```

**TabledRecursivePredicate**

TabledRecursivePredicate builds a true recursive, tabled predicate over a base relation. It returns a predicate constructor that can be called with arguments to form a Goal.

```go
// Example usage of TabledRecursivePredicate
result := TabledRecursivePredicate(/* parameters */)
```

**TabledRelation**

TabledRelation provides a convenient wrapper for creating tabled predicates over pldb relations. It returns a constructor function that builds tabled goals. Example: edge := DbRel("edge", 2, 0, 1) db := NewDatabase() db = db.AddFact(edge, NewAtom("a"), NewAtom("b")) // Create tabled predicate pathPred := TabledRelation(db, edge, "path") // Use it in queries x, y := Fresh("x"), Fresh("y") goal := pathPred(x, y)  // Automatically tabled

```go
// Example usage of TabledRelation
result := TabledRelation(/* parameters */)
```

**TriplesInts**

TriplesInts is TriplesIntsN with n<=0 (all results).

```go
// Example usage of TriplesInts
result := TriplesInts(/* parameters */)
```

**TriplesIntsN**

TriplesIntsN returns up to n triples of ints for three projected variables. Rows with non-int bindings are skipped.

```go
// Example usage of TriplesIntsN
result := TriplesIntsN(/* parameters */)
```

**TriplesStrings**

TriplesStrings is TriplesStringsN with n<=0 (all results).

```go
// Example usage of TriplesStrings
result := TriplesStrings(/* parameters */)
```

**TriplesStringsN**

TriplesStringsN returns up to n triples of strings for three projected variables. Rows with non-string bindings are skipped.

```go
// Example usage of TriplesStringsN
result := TriplesStringsN(/* parameters */)
```

**ValuesInt**

ValuesInt projects a named value from Solutions(...) into a slice of ints. Missing or non-int entries are skipped.

```go
// Example usage of ValuesInt
result := ValuesInt(/* parameters */)
```

**ValuesString**

ValuesString projects a named value from Solutions(...) into a slice of strings. Missing or non-string entries are skipped.

```go
// Example usage of ValuesString
result := ValuesString(/* parameters */)
```

**WithTabling**

WithTabling returns a convenience closure bound to the given SLG engine that can be used to evaluate tabled predicates without referencing the engine directly. Example: eval := WithTabling(NewSLGEngine(nil)) ch, err := eval(ctx, "path", []Term{NewAtom("a"), NewAtom("b")}, myEval)

```go
// Example usage of WithTabling
result := WithTabling(/* parameters */)
```

## Performance Considerations

### Optimization Tips

- Use appropriate data structures for your use case
- Consider memory usage for large datasets
- Profile your code to identify bottlenecks

### Caching

When appropriate, implement caching to improve performance:

```go
// Example caching pattern
var cache = make(map[string]interface{})

func getCachedValue(key string) (interface{}, bool) {
    return cache[key], true
}
```

## Security Best Practices

### Input Validation

Always validate inputs:

```go
func processInput(input string) error {
    if input == "" {
        return errors.New("input cannot be empty")
    }
    // Process the input
    return nil
}
```

### Error Information

Be careful not to expose sensitive information in error messages:

```go
// Good: Generic error message
return errors.New("authentication failed")

// Bad: Exposing internal details
return fmt.Errorf("authentication failed: invalid token %s", token)
```

## Testing Best Practices

### Unit Tests

Write comprehensive unit tests:

```go
func TestminikanrenFunction(t *testing.T) {
    // Test setup
    input := "test input"

    // Execute function
    result, err := minikanren.Function(input)

    // Assertions
    if err != nil {
        t.Errorf("Expected no error, got %v", err)
    }

    if result == nil {
        t.Error("Expected non-nil result")
    }
}
```

### Integration Tests

Test integration with other components:

```go
func TestminikanrenIntegration(t *testing.T) {
    // Setup integration test environment
    // Run integration tests
    // Cleanup
}
```

## Common Pitfalls

### What to Avoid

1. **Ignoring errors**: Always check returned errors
2. **Not cleaning up resources**: Use defer or context cancellation
3. **Hardcoding values**: Use configuration instead
4. **Not testing edge cases**: Test boundary conditions

### Debugging Tips

1. Use logging to trace execution flow
2. Add debug prints for troubleshooting
3. Use Go's built-in profiling tools
4. Check the [FAQ](../faq.md) for common issues

## Migration and Upgrades

### Version Compatibility

When upgrading minikanren:

1. Check the changelog for breaking changes
2. Update your code to use new APIs
3. Test thoroughly after upgrades
4. Review deprecated functions and types

## Additional Resources

- [API Reference](../../api-reference/minikanren.md)
