# High-Level API (HLAPI) v1.0 Scope

This page freezes the v1.0 scope and separates stretch ideas to avoid scope creep. Use this as the release checklist and as a reference for what’s considered “in” vs “later”.

## Frozen (v1.0)

Core term and domain helpers:
- A, L
- DomainRange, DomainValues

Model helpers:
- Model.IntVar, Model.IntVars, Model.IntVarsWithNames
- Model.AllDifferent, Model.LinearSum

Running goals and formatting results:
- Solve, SolveN
- Solutions, SolutionsN
- FormatSolutions and pretty printing

pldb + tabling sugar:
- MustRel, DB
- (db) Add, MustAdd, AddFacts, MustAddFacts
- (db) Q (accepts native values or Terms)
- TQ (tabled query), TablePred, TabledDB

Bulk loading:
- FactsSpec with Load, MustLoad
- NewDBFromMap

Recursive tabling:
- TabledRecursivePredicate (true recursive, tabled predicate helper)
- RecursiveTablePred (HLAPI wrapper accepting native values at call sites)

Extraction helpers for reified values:
- AsInt, MustInt; AsString, MustString; AsList
- ValuesInt, ValuesString

Examples/tests:
- HLAPI examples exercising joins, tabled recursion (ancestor/path), two-hop path, multi-relation loader, and values projection.
- Examples and tests are short, deterministic, and run in CI.

## Stretch (post v1.0)

- Additional HLAPI sugar for other FD/global constraints (e.g., cumulative, global-cardinality, lex order, regular, table).
- Convert more legacy examples to HLAPI style for parity (keep originals for contrast).
- Minor aliases/naming niceties (e.g., alternative helper names or thin wrappers) where discoverability helps.
- Expanded written guides and tutorials beyond the minimal getting-started note.

## Definition of Done (v1.0)

- API surface complete
  - All Frozen items above exist with literate inline docs.
  - No remaining TODOs for v1.0.
- Quality gates
  - Build: go test ./... passes.
  - Coverage: >= 70% (current ~75%).
  - Performance: examples run within seconds; no pathological long-running cases.
- Usability
  - At least one example per feature compiles and runs deterministically.
  - Minimal extraction helpers allow Solutions(...) to be consumed as typed values.
- Stability & docs
  - Public function names/signatures frozen for v1.0.
  - This page checked in and referenced from the Guides index.

## Notes

- Anything not explicitly listed under Frozen is considered Stretch by default. Promote to Frozen via a small change to this page and a quick review.
- Keep the HLAPI thin and additive: it should reduce boilerplate without changing core semantics.
