```go
func ExampleMatchu() {
	// Classify numbers with mutually exclusive ranges
	classify := func(n int) string {
		result := Run(1, func(q *Var) Goal {
			return CaseIntMap(NewAtom(n), map[int]string{
				0: "zero",
				1: "one",
				2: "two",
			}, q)
		})

		if len(result) == 0 {
			return "unknown"
		}

		if atom, ok := result[0].(*Atom); ok {
			if s, ok := atom.value.(string); ok {
				return s
			}
		}
		return "error"
	}

	fmt.Println(classify(0))
	fmt.Println(classify(1))
	fmt.Println(classify(5))

	// Output:
	// zero
	// one
	// unknown
}

```


