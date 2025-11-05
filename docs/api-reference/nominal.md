# Nominal Logic (Phase 7.1)

This module adds nominal constructs to gokanlogic, enabling reasoning about binders and alpha-equivalence.

- Tie(name, body): represents a binding form (λ name . body). Use `Lambda(name, body)` as an alias for readability.
- Fresho(name, term): goal adding a freshness constraint that `name` does not occur free in `term`.
- AlphaEqo(left, right): goal adding an alpha-equivalence constraint between two terms, treating Tie binders modulo renaming.
- NomFresh(prefix): helper that produces fresh nominal name atoms with unique suffixes (e.g., `x#42`).
- Substo(term, name, replacement, out): goal relating `out` to the capture-avoiding substitution of all free occurrences of `name` in `term` with `replacement`.

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
