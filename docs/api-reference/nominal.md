# Nominal Logic (Phase 7.1)

This module adds nominal constructs to gokanlogic, enabling reasoning about binders and alpha-equivalence.

- Tie(name, body): represents a binding form (λ name . body). Use `Lambda(name, body)` as an alias for readability.
- Fresho(name, term): goal adding a freshness constraint that `name` does not occur free in `term`.
- AlphaEqo(left, right): goal adding an alpha-equivalence constraint between two terms, treating Tie binders modulo renaming.
- NomFresh(prefix): helper that produces fresh nominal name atoms with unique suffixes (e.g., `x#42`).

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
