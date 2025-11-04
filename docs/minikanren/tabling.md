# SLG Tabling and Well‑Founded Semantics

This page documents the production SLG/WFS tabling layer: what gets cached, how answers are minimized via subsumption, how finite‑domain (FD) changes invalidate cached answers, and how to use the lightweight public wrappers to run tabled predicates.

## Answer subsumption: general vs. specific

When multiple answers for the same subgoal are derivable, the engine keeps only the most general answers and automatically retracts more specific ones. Intuitively, an answer A is more general than B if A binds a subset of the variables that B binds, with the same values for those variables.

- Definition: A subsumes B iff for every (var → value) in A, B has the same binding. Equivalently, bindings(B) ⊇ bindings(A).
- Behavior on insertion:
  - If an existing, non‑retracted answer subsumes a new answer, the new one is dropped (no change).
  - Otherwise, any existing, non‑retracted answers that are subsumed by the new one are retracted, then the new answer is inserted.
- Determinism and visibility:
  - Retractions are logical only: existing answers remain in insertion order internally but are marked “retracted,” and WFS‑aware iterators skip them.
  - Retractions signal an event so live consumers proceed deterministically without polling or sleeps.
  - Delay‑set metadata (conditional answers) is preserved per remaining, visible answers. Retraction never mutates other answers’ metadata.

Examples (bindings shown as maps):
- General then specific: insert {X=1} then {X=1, Y=2} → the second is dropped because {X=1} already subsumes it.
- Specific then general: insert {X=1, Y=2} then {X=1} → the first is retracted; only {X=1} remains visible.
- Equal answers: duplicates are ignored (idempotent insert).

Why this matters:
- Smaller tables: fewer redundant answers to propagate/consume.
- Stronger compositionality: consumers of a subgoal get a logically minimal set of answers under subsumption, independent of derivation order.

## FD‑domain invalidation and tabled answers

The SLG engine prunes cached answers when FD domains shrink, keeping tabling consistent with the current constraint store.

- Trigger: whenever the FD solver prunes a variable’s domain, it notifies the SLG engine to invalidate answers that bind that variable to now‑impossible values.
- Semantics:
  - For a variable id v and domain D, any tabled answer whose binding for v is an integer atom not in D is retracted.
  - Answers that leave v unbound, or bind it to non‑integer atoms, are left untouched.
  - Retractions are evented and invisible to WFS iterators, just like subsumption.
- Interaction with subsumption:
  - If a specific answer is retracted, a more general existing answer may already cover the remaining space; nothing special is required.
  - If a general answer itself becomes inconsistent (e.g., v=7 is now impossible), it’s retracted even if it previously subsumed others; retracted specifics are not resurrected automatically—the engine will re‑derive new answers if and when they become valid again.
- Directionality: current invalidation is conservative and monotone—answers are retracted on domain shrink. Domain expansion does not auto‑restore previously retracted answers; new derivations will reinsert them if they become valid.

This yields deterministic, timer‑free synchronization between constraint propagation and tabling: FD changes immediately and deterministically reflect in the visible answer set.

## Public wrappers: TabledEvaluate and WithTabling

Two tiny helpers make it easy to use the tabling engine without boilerplate. They’re thin, synchronous wrappers over the production SLG engine and are safe for concurrent use.

### TabledEvaluate

Runs a tabled predicate using the global engine. Provide a predicate identifier, its arguments (Terms), and a GoalEvaluator that emits answers.

```go
package main

import (
    "context"
    "fmt"
    "github.com/gitrdm/gokando/pkg/minikanren"
)

func main() {
    // Reset the global engine for a clean run (optional).
    minikanren.ResetGlobalEngine()

    // An evaluator that emits exactly one answer.
    inner := minikanren.GoalEvaluator(func(ctx context.Context, answers chan<- map[int64]minikanren.Term) error {
        answers <- map[int64]minikanren.Term{42: minikanren.NewAtom(1)}
        return nil
    })

    ch, err := minikanren.TabledEvaluate(context.Background(), "demo", []minikanren.Term{minikanren.NewAtom("a")}, inner)
    if err != nil {
        fmt.Println("error:", err)
        return
    }
    for range ch { /* drain */ }

    fmt.Println("ok")
}
```

### WithTabling

Binds an SLG engine instance and returns a closure you can call repeatedly. Handy when you want per‑engine stats or non‑global isolation.

```go
package main

import (
    "context"
    "fmt"
    "github.com/gitrdm/gokando/pkg/minikanren"
)

func main() {
    engine := minikanren.NewSLGEngine(nil)
    eval := minikanren.WithTabling(engine)

    inner := minikanren.GoalEvaluator(func(ctx context.Context, answers chan<- map[int64]minikanren.Term) error {
        answers <- map[int64]minikanren.Term{1: minikanren.NewAtom("ok")}
        return nil
    })

    ch, err := eval(context.Background(), "test", []minikanren.Term{minikanren.NewAtom("x")}, inner)
    if err != nil {
        fmt.Println("error:", err)
        return
    }
    for range ch { /* drain */ }

    stats := engine.Stats()
    fmt.Printf("evaluations=%d cached=%d\n", stats.TotalEvaluations, stats.CachedSubgoals)
}
```

Notes:
- The channel emits answer bindings in insertion order and then closes. Consume it fully to allow the producer to finish.
- Answers are cached under the normalized CallPattern for (predicateID, args). Re‑evaluating the same call pattern returns from cache deterministically.
- Subsumption and FD‑domain invalidation apply to these cached answers automatically—no extra work is required by callers.

## Where this lives in the code

- Subsumption and invalidation are implemented in `pkg/minikanren/tabling.go` within `SubgoalEntry` (insertion with subsumption, per‑answer retraction, and FD‑domain invalidation).
- Engine‑level orchestration and evaluation live in `pkg/minikanren/slg_engine.go`.
- The FD plugin notifies the engine about domain changes; see `pkg/minikanren/hybrid_fd_plugin.go`.
- Public wrappers are in `pkg/minikanren/slg_wrappers.go` with runnable examples in `pkg/minikanren/slg_wrappers_example_test.go`.

If you’re new to SLG/WFS, start with the wrappers above and build up: they cover most everyday uses while keeping you on the deterministic, production paths.
