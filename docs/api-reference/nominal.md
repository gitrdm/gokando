---
title: Nominal Logic (Phase 7.1)
render_with_liquid: false
---

# Nominal Logic (Phase 7.1)

This module adds nominal constructs to gokanlogic, enabling reasoning about binders and alpha-equivalence.

- Tie(name, body): represents a binding form (λ name . body). Use `Lambda(name, body)` as an alias for readability.
- Fresho(name, term): goal adding a freshness constraint that `name` does not occur free in `term`.
- AlphaEqo(left, right): goal adding an alpha-equivalence constraint between two terms, treating Tie binders modulo renaming.
- NomFresh(prefix): helper that produces fresh nominal name atoms with unique suffixes (e.g., `x#42`).
- Substo(term, name, replacement, out): goal relating `out` to the capture-avoiding substitution of all free occurrences of `name` in `term` with `replacement`.
- App(fun, arg): helper to construct application terms as `Pair(fun, arg)`.
- BetaReduceo(term, out): goal performing one leftmost-outermost beta-reduction step; fails if no redex exists.
- BetaNormalizeo(term, out): goal reducing a term to beta-normal form using leftmost-outermost strategy.
- FreeNameso(term, outList): goal producing the proper list of free nominal names (Atoms) in `term` (sorted deterministically).
- TypeChecko(term, env, type): simply-typed λ-calculus type checker over Atoms, application (Pair), and Tie binders; `env` is an association list of (name . type) pairs.

Constraints implement the project-wide `Constraint` interface and are processed by the `NominalPlugin` in the HybridSolver. They are purely local and thread-safe.

Note: When using LocalConstraintStore, constraints are validated at add-time. A `Fresho` goal that creates a freshness constraint already violated by current bindings will cause `AddConstraint` to return an error; such a constraint is not recorded as pending.

## Examples

- Freshness under binders:

```go
func ExampleFresho_basic() {
    a := NewAtom("a")
    term := Lambda(a, a) // λa.a
    solutions := Run(1, func(q *Var) Goal {
        return Conj(
            Fresho(a, term),
            Eq(q, NewAtom("ok")),
        )
    })
    fmt.Println(solutions)
    // Output: [ok]
}
```

- Alpha-equivalence:

```go
func ExampleAlphaEqo_basic() {
    a := NewAtom("a")
    b := NewAtom("b")
    t1 := Lambda(a, a)
    t2 := Lambda(b, b)
    results := Run(1, func(q *Var) Goal {
        return Conj(
            AlphaEqo(t1, t2),
            Eq(q, NewAtom(true)),
        )
    })
    fmt.Println(results)
    // Output: [true]
}
```

See `pkg/minikanren/nominal_example_test.go` for more examples.

## Substitution without capture (Substo)

Substo performs λ-calculus style substitution respecting binders encoded with `Tie`/`Lambda`:

- If the binder equals the target name, substitution does not enter the body.
- If the binder is fresh for the replacement, substitution proceeds under the same binder.
- Otherwise, the binder is alpha-renamed to a fresh nominal atom (via `NomFresh`) before substitution to avoid capture.

Notes:
- Substo is a Goal that computes deterministically with current bindings. If its decision depends on unresolved logic variables (e.g., freshness cannot yet be decided), it yields no solution until more information becomes available.
- For nominal freshness checks, the same add-time validation rule applies: `Fresho` used internally in derivation is validated immediately in the LocalConstraintStore.

Example:

```go
func ExampleSubsto_avoidCapture() {
    a := NewAtom("a")
    b := NewAtom("b")
    term := Lambda(b, a) // λb.a

    results := Run(1, func(q *Var) Goal {
        return Substo(term, a, b, q)
    })

    tie := results[0].(*TieTerm)
    // binderIsB:false, bodyIsB:true — binder was renamed, body became b
    fmt.Printf("binderIsB:%v bodyIsB:%v\n", tie.name.Equal(b), tie.body.Equal(b))
    // Output: binderIsB:false bodyIsB:true
}
```

## Beta reduction and normalization

Application is represented by `App(fun, arg)` which is a shorthand for `Pair(fun, arg)`.

Leftmost-outermost beta reduction (one step):

```go
func ExampleBetaReduce_basic() {
    a := NewAtom("a")
    b := NewAtom("b")
    term := App(Lambda(a, a), b) // (λa.a) b
    results := Run(1, func(q *Var) Goal { return BetaReduceo(term, q) })
    fmt.Println(results[0])
    // Output: b
}
```

Normalization to beta-normal form (deterministic; pending if variables prevent decisions):

```go
func ExampleBetaNormalize_basic() {
    a := NewAtom("a")
    x := NewAtom("x")
    y := NewAtom("y")
    term := App(Lambda(a, Lambda(x, a)), y)
    results := Run(1, func(q *Var) Goal { return BetaNormalizeo(term, q) })
    fmt.Println(results[0])
    // Output: (tie x . y)
}
```

## Free names

Compute the set of free nominal names (as a proper list) with stable ordering:

```go
func ExampleFreeNames_basic() {
    a := NewAtom("a")
    b := NewAtom("b")
    term := Lambda(a, App(a, b)) // free(b)
    results := Run(1, func(q *Var) Goal { return FreeNameso(term, q) })
    fmt.Println(results[0])
    // Output: (b . <nil>)
}
```

## Simple type checking

Types are terms; arrow types use `ArrType(t1, t2)`. Environments are proper lists of pairs `(name . type)`. Checking is deterministic and may involve logic variables inside the expected type.

```go
func ExampleTypeCheck_lambda() {
    a := NewAtom("a")
    T := Fresh("T")
    term := Lambda(a, a)
    ty := ArrType(T, T) // expect λa.a : T->T
    results := Run(1, func(q *Var) Goal { return TypeChecko(term, Nil, ty) })
    fmt.Println(len(results) > 0)
    // Output: true
}

func ExampleTypeCheck_app() {
    b := NewAtom("b")
    intT := NewAtom("Int")
    id := NewAtom("id")
    env := EnvExtend(EnvExtend(Nil, b, intT), id, ArrType(intT, intT))
    term := App(id, b)
    results := Run(1, func(q *Var) Goal { return TypeChecko(term, env, intT) })
    fmt.Println(len(results) > 0)
    // Output: true
}
```
