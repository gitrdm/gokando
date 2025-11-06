# API Reference

Complete API documentation for gokanlogic.

## Overview

This section contains detailed API documentation for all packages. For package overviews and getting started guides, see the [Packages](../packages/README.md) section.

## Package APIs

### [parallel](parallel.md)

Package parallel provides advanced parallel execution capabilities
for miniKanren goals. This package contains internal utilities
for managing concurrent goal evaluation with proper resource
management and backpressure control.


**[→ Full API Documentation](parallel.md)**

Key APIs:

- Types and interfaces
- Functions and methods
- Constants and variables
- Detailed usage examples

### [minikanren](minikanren.md)

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

Package minikanren provides advanced control flow operators that extend
the core conjunction (Conj) and disjunction (Disj/Conde) primitives.

# Control Flow Operators

This package implements four fundamental control flow operators inspired by
Prolog and advanced logic programming systems:

  - Ifa: If-then-else with backtracking through all condition solutions
  - Ifte: If-then-else with commitment to first condition solution
  - SoftCut: Prolog-style soft cut (*->) for conditional commitment
  - CallGoal: Meta-call for indirect goal invocation

# Design Philosophy

These operators are implemented using the existing Goal/Stream and
ConstraintStore interfaces with no special runtime support. They respect
context cancellation and integrate seamlessly with the SLG tabling system.

# Variable Scoping

CRITICAL: All variables used in control flow goals must be created inside
the Run closure to ensure proper projection and substitution:

	// CORRECT - variables inside closure
	Run(5, func(q *Var) Goal {
	    x := Fresh("x")
	    return Ifa(Eq(x, NewAtom(1)), Eq(q, x), Eq(q, NewAtom("none")))
	})

	// WRONG - variables outside closure (will return unbound)
	x := Fresh("x")
	Run(5, func(q *Var) Goal {
	    return Ifa(Eq(x, NewAtom(1)), Eq(q, x), Eq(q, NewAtom("none")))
	})

# Search Behavior

The operators differ in how they handle multiple solutions from the condition:

  - Ifa: Evaluates thenGoal for EACH solution of condition; if condition fails, evaluates elseGoal
  - Ifte: Commits to FIRST solution of condition and evaluates thenGoal; if condition fails, evaluates elseGoal
  - SoftCut: Synonym for Ifte with Prolog-compatible semantics

# Integration with SLG Tabling

These operators are compatible with SLG/WFS tabling. They do not execute
goals during pattern construction, avoiding circular dependencies. All goal
evaluation happens lazily during stream consumption.

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

Package minikanren provides DCG (Definite Clause Grammar) support with
SLG resolution and pattern-based evaluation.

# Pattern-Based DCG Architecture

DCGs in this package implement a pattern-based architecture where grammar
rules return DESCRIPTIONS of goals rather than executing them directly.
This design enables:
  - Clause-order independence (declarative semantics)
  - Left recursion via SLG fixpoint iteration
  - Clean separation between grammar construction and evaluation

# Difference Lists

DCGs use difference lists to represent sequences:
  - Input list s0, output list s1
  - Sequence [a,b,c] represented as: s0 = [a,b,c|s1]
  - Empty sequence: s0 = s1

# Pattern Types

DCG patterns construct goal descriptions without executing them:
  - Terminal(t): Matches single token (s0=[t|s1])
  - Seq(p1, p2): Sequential composition
  - Alternation(p1, p2, ...): Choice (declarative, order-independent)
  - NonTerminal(engine, name): Reference to defined rule

# SLG Integration

When evaluating rules, the SLG engine orchestrates pattern expansion:
 1. Rule bodies return GoalPattern descriptions
 2. SLG expands patterns to concrete Goals
 3. Recursive NonTerminal calls route through SLG (cycle detection, caching)
 4. No circular execution chains within pattern constructors

# Example: Left-Recursive Grammar

	engine := NewSLGEngine(nil)
	DefineRule("expr", Alternation(
	    NonTerminal(engine, "term"),
	    Seq(NonTerminal(engine, "expr"), Terminal(NewAtom("+")), NonTerminal(engine, "term")),
	))
	DefineRule("term", Terminal(NewAtom("1")))

	// Parse with SLG tabling
	results := Run(5, func(q *Var) Goal {
	    input := MakeList(NewAtom("1"), NewAtom("+"), NewAtom("1"))
	    rest := Fresh("rest")
	    return Conj(
	        ParseWithSLG(engine, "expr", input, rest),
	        Eq(q, rest),
	    )
	})

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

Version: 1.2.0

This package offers a complete set of miniKanren operators with high-performance
concurrent execution capabilities, designed for production use.


**[→ Full API Documentation](minikanren.md)**

Key APIs:

- Types and interfaces
- Functions and methods
- Constants and variables
- Detailed usage examples

## Navigation

- **[Packages](../packages/README.md)** - Package overviews and installation
- **[Examples](../examples/README.md)** - Working code examples
- **[Guides](../guides/README.md)** - Best practices and patterns

## External References

- [pkg.go.dev Documentation](https://pkg.go.dev/github.com/gitrdm/gokanlogic) - Go module documentation
- [GitHub Repository](https://github.com/gitrdm/gokanlogic) - Source code and issues
