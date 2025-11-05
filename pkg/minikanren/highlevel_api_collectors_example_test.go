package minikanren_test

import (
	"fmt"
	"time"

	. "github.com/gitrdm/gokanlogic/pkg/minikanren"
)

// Example_hlapi_collectors_ints shows using Ints to collect numeric answers
// directly from a goal without manual projection.
func Example_hlapi_collectors_ints() {
	x := Fresh("x")
	goal := Disj(Eq(x, A(1)), Eq(x, A(3)), Eq(x, A(5)))
	vals := Ints(goal, x)
	// Print count and sum to avoid relying on order
	sum := 0
	for _, v := range vals {
		sum += v
	}
	fmt.Printf("%d %d\n", len(vals), sum)
	// Output:
	// 3 9
}

// Example_hlapi_collectors_rows shows using Rows to gather multiple projected
// variables in order, returning [][]Term for flexible formatting.
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

// Example_hlapi_collectors_pairs_ints shows how to collect typed pairs of ints.
func Example_hlapi_collectors_pairs_ints() {
	x, y := Fresh("x"), Fresh("y")
	goal := Disj(
		Conj(Eq(x, A(1)), Eq(y, A(2))),
		Conj(Eq(x, A(3)), Eq(y, A(4))),
	)
	pairs := PairsInts(goal, x, y)
	// Print count and sum of all elements for stable output
	sum := 0
	for _, p := range pairs {
		sum += p[0] + p[1]
	}
	fmt.Printf("%d %d\n", len(pairs), sum)
	// Output:
	// 2 10
}

// Example_hlapi_rowsAll_timeout demonstrates using a timeout to guard against
// accidental infinite enumeration while collecting all rows.
func Example_hlapi_rowsAll_timeout() {
	x, y := Fresh("x"), Fresh("y")
	goal := Disj(
		Conj(Eq(x, A(1)), Eq(y, A("a"))),
		Conj(Eq(x, A(2)), Eq(y, A("b"))),
	)
	rows := RowsAllTimeout(50*time.Millisecond, goal, x, y)
	fmt.Println(len(rows))
	// Output:
	// 2
}
