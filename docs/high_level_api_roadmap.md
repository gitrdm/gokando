# GoKanDo High‑Level API Roadmap

This document defines the plan to introduce an ergonomic, additive High‑Level API (HLAPI) for GoKanDo. It aims to remove boilerplate seen across examples and tests while preserving composability, performance, and the existing public API.

Quick recap (LLM-friendly):
- Goal: Reduce verbosity in common tasks (goal execution, FD modeling, pldb queries, tabling, hybrid glue).
- Approach: Add small, orthogonal wrappers; keep all low-level APIs intact.
- Success: Fewer lines for typical examples, unchanged semantics, all tests pass under -race.

---

## Principles

- Additive, not breaking: Existing APIs remain valid; HLAPI sits on top.
- Composable: All helpers return existing types (Goal, *Model, *FDVariable, etc.).
- Predictable defaults: Sensible context, buses, limits; options to override.
- Zero-cost abstractions: Thin wrappers around current implementations.
- Production-ready: Tested, documented, race-safe, benchmark-neutral.
- LLM-friendly docs: Clear sectioning, repeated key names, short examples; periodic recaps.

Context refresh: HLAPI exports small helpers for “Run/collect,” "Model/Domain sugar," "pldb sugar," "tabling wrappers," and "hybrid glue." Names are stable and repeated below to help retrieval.

---

## Current pain points (grounded in repo)

- Manual context/store/stream loops in many examples.
- Verbose domain constructors (NewBitSetDomain*, max calculation, value slices).
- Repeated modeling scaffolding: NewModel → NewVariable → constructor → AddConstraint → NewSolver → Solve.
- pldb friction: NewAtom everywhere, AddFact one-at-a-time, basic OR queries require manual Disj.
- Tabling: Explicit evaluators/patterns show up even for simple, repeatable use.
- Hybrid: Common glue (map relational result to FD var, filter by FD domain) repeats.
- Result rendering: Repeated prettyTerm/runGoal utilities.

LLM hint: Keywords to remember — Solutions, Reify, Format; DomainRange/DomainValues; Model.IntVar/IntVars; Model.AllDifferent/LinearSum; Solve/Optimize; db.Q/Facts; Tabled/TableStats; FilteredQ/AutoMap.

---

## Scope (MVP → Phases)

- MVP (v0.1):
  - Goal execution: Solutions, SolutionsN, SolutionsCtx; Format/FormatTerm; Reify helpers; A (atom) and L (list) sugar.
  - Domains & variables: DomainRange, DomainValues; Model.IntVar/IntVars/IntVarsWithValues; Model.AllDifferent; Model.LinearSum; Solve/Optimize shorthands with options.
  - pldb ergonomics: Facts (batch), AddFacts; db.Q sugar (auto-Atom; "_" → Fresh).
  - Tabling: Tabled, TabledFunc, WithTabling, AbolishTable, AbolishAllTables, TableStats (thin wrappers).
- Phase 2:
  - pldb DisjQ; RowsN/StringsN/IntsN collectors; more Model convenience (MinOf/MaxOf/Lex/Table) as shims.
  - Hybrid helpers: FilteredQ, AutoMap.
- Phase 3:
  - Typed ReifyAs[T]; optional result structs for popular patterns.

Non-goals:
- No changes to internal engines or semantics.
- No silent global state toggles; all options explicit.

---

## API outline (contracts)

For each function: Inputs; Returns; Errors; Notes on determinism.

### 1) Goal execution and results

- Solutions(goal Goal, vars ...Term) ([]map[*Var]Term, error)
  - Runs goal with defaults; reifies vars into a per-solution map. Returns all solutions (bounded by sensible default) or error if context canceled.
- SolutionsN(n int, goal Goal, vars ...Term) ([]map[*Var]Term, error)
  - Like Solutions, but returns up to n solutions (recommended default for finite problems).
- SolutionsCtx(ctx context.Context, n int, goal Goal, vars ...Term) ([]map[*Var]Term, error)
  - Context-aware execution (timeouts/cancel).
- Reify helpers: ReifyInt, ReifyString, ReifyList, ReifyAs[T any]
  - Extract typed values safely; (value, ok) semantics.
- Pretty/formatting: Format(store ConstraintStore, vars ...Term) string; FormatTerm(t Term) string
  - Canonical user-facing rendering (replaces repeated prettyTerm/runGoal code).
- Sugar: A(v interface{}) Term (NewAtom alias); L(vals ...interface{}) Term (list builder with auto-atoming of scalars).

Notes: Ordering of solutions not guaranteed (depends on goal/parallelism), matching current behavior.

### 2) Domains and variables (FD)

- DomainRange(min, max int) Domain
  - Inclusive [min..max]; optimized bitset build.
- DomainValues(values ...int) Domain
  - Construct domain from explicit set.
- (*Model) IntVar(min, max int, name ...string) *FDVariable
- (*Model) IntVars(n, min, max int, baseName ...string) []*FDVariable
- (*Model) IntVarsWithValues(values []int, baseName ...string) []*FDVariable
- (*Model) AllDifferent(vars ...*FDVariable) error
- (*Model) LinearSum(vars []*FDVariable, coeffs []int, total *FDVariable) error
- Solve/Optimize shorthands:
  - Solve(model *Model, opts ...SolveOption) (Solution, error)
  - Optimize(model *Model, obj *FDVariable, minimize bool, opts ...SolveOption) (Solution, best int, err error)
  - Options: WithParallelWorkers(int), WithTimeout(time.Duration), WithNodeLimit(int), etc.

Notes: All wrappers call existing constructors (e.g., NewLinearSum) and solver methods (Solve, SolveOptimal*).

### 3) pldb ergonomics

- NewDB() *Database (alias of NewDatabase for discoverability).
- (*Database) Facts(rel string, rows ...[]interface{}) (*Database, error)
  - Add many facts at once; auto-wrap Go scalars as Atoms.
- (*Database) AddFacts(rel *Relation, rows ...[]Term) (*Database, error)
  - Term-typed batch variant.
- (*Database) Q(rel *Relation, terms ...interface{}) Goal
  - Query sugar: auto-convert scalars to Atoms; use Fresh for "_" sentinel; returns Goal.
- DisjQ(db *Database, rel *Relation, variants [][]interface{}) Goal (Phase 2)
  - OR across pattern variants in one call.

### 4) Tabling (maps to roadmap Phase 5.5)

- Tabled(predicateID string, goalFn Goal) Goal
- TabledFunc(predicateID string, fn func(...Term) Goal) func(...Term) Goal
- WithTabling(config *SLGConfig, goal Goal) Goal
- AbolishTable(predicateID string) error; AbolishAllTables() error
- TableStats() *SLGStats

Notes: Thin wrappers over existing SLG engine and slg_wrappers.go; names mirror roadmap exemplars.

### 5) Hybrid glue (FD ↔ pldb)

- FilteredQ(db *Database, rel *Relation, fdVar *FDVariable, relVar *Var, terms ...interface{}) Goal
  - Thin wrapper calling FDFilteredQuery with auto-atom conversions.
- AutoMap(adapter *UnifiedStoreAdapter, result ConstraintStore, pairs ...struct{Rel *Var; FD *FDVariable}) (*UnifiedStore, error)
  - Batch version of MapQueryResult; no-ops when bindings missing.

---

## Before/after (illustrative snippets)

Optimization (minimize x + 2y over {1,2,3}):
- Before: NewModel → NewBitSetDomainFromValues → NewVariable → NewLinearSum → AddConstraint → NewSolver → SolveOptimal.
- After:
  - x, y := m.IntVarsWithValues([]int{1,2,3}, "x", "y")
  - t := m.IntVar(1, 20, "t"); _ = m.LinearSum([]*FDVariable{x, y}, []int{1,2}, t)
  - _, best, _ := Optimize(m, t, true)

pldb query (children of "alice"):
- Before: DbRel, NewDatabase, repeated AddFact(NewAtom(..)), Fresh, db.Query, manual stream handling.
- After:
  - rel, _ := DbRel("parent", 2, 0, 1); db := NewDB()
  - db, _ = db.Facts("parent", []interface{}{"alice","bob"}, []interface{}{"alice","charlie"})
  - child := Fresh("child"); rows, _ := SolutionsN(10, db.Q(rel, "alice", child), child)

Tabled join (grandparent):
- pq := db.TQ(parent, "parent")  // via TabledFunc under the hood
- gp, p, gc := Fresh("gp"), Fresh("p"), Fresh("gc")
- rows, _ := SolutionsN(10, Conj(pq(gp, p), pq(p, gc)), gp, gc)

List ops printing:
- Use L(...) and A(...) sugar; Solutions + Format to avoid custom pretty printers.

---

## Milestones and tasks

M0 — Document and skeleton (this file)
- [x] HLAPI roadmap completed and checked in.

M1 — MVP implementation
- [ ] Implement Solutions/SolutionsN/SolutionsCtx; Reify*, Format/FormatTerm; A, L.
- [ ] Implement DomainRange/DomainValues; Model.IntVar/IntVars/IntVarsWithValues; AllDifferent; LinearSum shims.
- [ ] Implement Solve/Optimize shorthands and options.
- [ ] Convert 2 examples to HLAPI equivalents (one optimization, one list/goal).

M2 — pldb convenience
- [ ] Implement Facts/AddFacts; db.Q sugar; DisjQ (optional in M2).
- [ ] Convert pldb example(s) to HLAPI.

M3 — Tabling wrappers
- [ ] Implement Tabled/TabledFunc/WithTabling; AbolishTable/AbolishAllTables; TableStats.
- [ ] Add example mirroring existing tabled examples with HLAPI.

M4 — Hybrid helpers
- [ ] Implement FilteredQ and AutoMap; convert one hybrid example.

M5 — Adoption and polish
- [ ] Convert at least 5 examples; write migration guide.
- [ ] Benchmarks to confirm no regression vs low-level equivalents.

---

## Acceptance criteria

Functional:
- HLAPI reproduces results of equivalent low-level code paths across examples and unit tests.
- No public breaking changes; low-level APIs remain available and documented.

Ergonomics:
- ≥30% LOC reduction in three representative examples (optimization, pldb join, list ops/tabled).
- Replace ≥70% of NewBitSetDomain* calls in examples with DomainRange/DomainValues or Model.IntVar helpers.
- Replace ≥80% of manual stream/context plumbing in examples with Solutions/SolutionsN.

Quality gates:
- Build: PASS; Lint/Typecheck: PASS; Tests: PASS; -race: PASS.
- Performance: No statistically significant slowdowns in example benchmarks (target ±2%).

Docs:
- New guide section: “High‑Level API” with side-by-side classic vs HLAPI.
- API reference godocs for all HLAPI functions.

---

## Coding standards and quality gates (HLAPI)

Production-only code (no placeholders):
- No TODOs, placeholders, stubs, or commented-out code in committed HLAPI.
- No dead code; no half-implemented branches or feature flags for incomplete work.
- Additive changes only; do not break or weaken existing public APIs.

Testing (regression-quality, not smoke):
- Real implementations only; no mocks/stubs for HLAPI behavior paths.
- Deterministic tests/examples; fix seeds/orderings where relevant; avoid timeouts as correctness oracles.
- Race detector mandatory on suites that exercise HLAPI paths (go test -race).
- Coverage: maintain or increase repository coverage; new HLAPI code targets ≥90% coverage with focused unit and example tests.
- Examples as tests: every exported HLAPI function has a corresponding Go Example in a *_example_test.go that compiles and asserts output via // Output:.

Documentation (Go-literate style):
- Godoc comments start with a clear, single-sentence summary, followed by usage notes, options, and behavior.
- Include concurrency/thread-safety notes, error modes, determinism/ordering, and performance characteristics.
- Keep names and terms consistent with existing package docs (Goal, ConstraintStore, Domain, FDVariable, etc.).

Performance and concurrency:
- HLAPI wrappers must be zero-cost abstractions over existing implementations (no hidden goroutines, no extra buffering beyond what underlying APIs do).
- Avoid allocations on hot paths; prefer passing through to existing efficient code.
- Benchmarks (where practical) should show no statistically significant regression versus equivalent low-level code.

Tooling and CI:
- Build: PASS; Lint/Typecheck: PASS; Vet/Staticcheck clean; gofmt/go mod tidy enforced.
- Tests: PASS locally and in CI under -race; example tests run as part of CI.

Backwards compatibility and deprecation:
- HLAPI is strictly additive; low-level APIs remain supported and documented.
- If a future deprecation is required, provide a compatibility shim and migration notes; do not remove functionality without a major version signal.

Security and side effects:
- No network calls or environment-dependent behavior in HLAPI.
- No global mutable state beyond documented singletons already present; maintain thread safety.

This section mirrors the repository’s existing standards (production-ready, zero technical debt) and makes them explicit for all HLAPI additions.

---

## Risks and mitigations

- Ordering expectations: HLAPI does not enforce deterministic ordering beyond what underlying goals provide. Document explicitly.
- Over-sugar: Keep wrappers thin; expose options to fall back to low-level control.
- Concept drift: Periodic review to ensure wrappers stay close to core semantics.

---

## Mapping to existing roadmap

- Phase 4 (Search/Optimization): HLAPI Solve/Optimize shorthands align with production-ready solver.
- Phase 5 (SLG/WFS): HLAPI tabling wrappers (Tabled, WithTabling, TableStats) align with “Public API and UX” goals.
- Phase 6 (pldb): HLAPI pldb sugar and hybrid helpers build on existing adapters and helpers.

---

## Glossary (names repeated for retrieval)

HLAPI function index:
- Solutions, SolutionsN, SolutionsCtx, ReifyInt, ReifyString, ReifyList, ReifyAs, Format, FormatTerm, A, L.
- DomainRange, DomainValues; (*Model) IntVar, IntVars, IntVarsWithValues, AllDifferent, LinearSum; Solve, Optimize; WithParallelWorkers, WithTimeout, WithNodeLimit.
- (*Database) Facts, AddFacts, Q; DisjQ.
- Tabled, TabledFunc, WithTabling, AbolishTable, AbolishAllTables, TableStats.
- FilteredQ, AutoMap.

These names are intentional duplicates of the section headings to improve LLM recall across context windows.

---

## Decision log (summary)

- Use additive wrappers only; do not change semantics.
- Prefer minimal, memorable names mirroring existing concepts.
- Provide typed reification helpers for safety and clarity.
- Provide batch/builder ergonomics for pldb.
- Document non-determinism and defaults clearly.

End of document.
