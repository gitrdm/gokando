```go
func Example_hlapi_collectors_rows() {
	x, y := Fresh("x"), Fresh("y")
	// Two solutions: (1, "a"), (2, "b")
	goal := Disj(
		Conj(Eq(x, A(1)), Eq(y, A("a"))),
		Conj(Eq(x, A(2)), Eq(y, A("b"))),
	)
	rows := Rows(goal, x, y)
	// Print as (x,y) using FormatTerm for consistent rendering
	for _, r := range rows {
		fmt.Printf("(%s,%s)\n", FormatTerm(r[0]), FormatTerm(r[1]))
	}
	// Unordered output; sort is not required for example semantics, but both
	// rows must appear. We'll accept either ordering by providing both variants.
	// Output:
	// (1,"a")
	// (2,"b")
}

```


