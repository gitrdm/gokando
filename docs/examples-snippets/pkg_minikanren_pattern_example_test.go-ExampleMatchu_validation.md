```go
func ExampleMatchu_validation() {
	// Validate that a value matches exactly one category
	validate := func(val int) (string, bool) {
		// Alternative implementation for demonstration purposes
		// return Matchu(NewAtom(val),
		//		NewClause(NewAtom(1), Eq(q, NewAtom("category-A"))),
		//		NewClause(NewAtom(2), Eq(q, NewAtom("category-B"))),
		//		NewClause(NewAtom(3), Eq(q, NewAtom("category-C"))),
		result := Run(1, func(q *Var) Goal {
			return CaseIntMap(NewAtom(val), map[int]string{
				1: "category-A",
				2: "category-B",
				3: "category-C",
			}, q)
		})

		if len(result) == 0 {
			return "", false
		}

		if atom, ok := result[0].(*Atom); ok {
			if s, ok := atom.value.(string); ok {
				return s, true
			}
		}
		return "", false
	}

	// Valid values
	cat, ok := validate(1)
	fmt.Printf("Value 1: %s (valid: %t)\n", cat, ok)

	cat, ok = validate(2)
	fmt.Printf("Value 2: %s (valid: %t)\n", cat, ok)

	// Invalid value (no match)
	cat, ok = validate(99)
	fmt.Printf("Value 99: %s (valid: %t)\n", cat, ok)

	// Output:
	// Value 1: category-A (valid: true)
	// Value 2: category-B (valid: true)
	// Value 99:  (valid: false)
}

```


