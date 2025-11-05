package minikanren_test

import (
	"context"
	"fmt"
	"sort"
	"strings"

	. "github.com/gitrdm/gokanlogic/pkg/minikanren"
)

func runGoal(goal Goal, vars ...Term) []string {
	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)

	// Collect all results by taking one at a time until stream is closed
	var stores []ConstraintStore
	for {
		rs, more := stream.Take(1)
		if len(rs) > 0 {
			stores = append(stores, rs...)
		}
		if !more {
			break
		}
	}

	var out []string
	for _, st := range stores {
		if len(vars) == 0 {
			out = append(out, "")
			continue
		}
		parts := make([]string, 0, len(vars))
		for _, v := range vars {
			name := "q"
			if vv, ok := v.(*Var); ok && vv != nil {
				s := vv.String() // "_q_13" or "_13"
				// Try to extract the friendly name between underscores if present
				if strings.HasPrefix(s, "_") {
					segs := strings.Split(s, "_")
					if len(segs) >= 3 && segs[1] != "" {
						name = segs[1]
					}
				}
			}
			val := st.GetSubstitution().DeepWalk(v)
			parts = append(parts, fmt.Sprintf("%s: %s", name, prettyTerm(val)))
		}
		out = append(out, strings.Join(parts, ", "))
	}
	sort.Strings(out)
	return out
}

// prettyTerm renders terms in a user-friendly form:
// - Lists as (a b c), with strings quoted
// - Empty list as ()
// - Improper lists as (a b . tail)
func prettyTerm(t Term) string {
	// Empty list: Atom(nil)
	if a, ok := t.(*Atom); ok {
		if a.Value() == nil {
			return "()"
		}
		switch v := a.Value().(type) {
		case string:
			return fmt.Sprintf("%q", v)
		default:
			return fmt.Sprintf("%v", v)
		}
	}

	// Proper or improper list
	if p, ok := t.(*Pair); ok {
		elems := []string{}
		tail := Term(p)
		for {
			pr, ok := tail.(*Pair)
			if !ok {
				break
			}
			elems = append(elems, prettyTerm(pr.Car()))
			tail = pr.Cdr()
		}

		// Check tail
		if a, ok := tail.(*Atom); ok && a.Value() == nil {
			// Proper list
			return "(" + strings.Join(elems, " ") + ")"
		}
		// Improper list
		return "(" + strings.Join(elems, " ") + " . " + prettyTerm(tail) + ")"
	}

	// Variables or other terms - fall back to String()
	return t.String()
}

// ExampleRembero demonstrates removing duplicate elements from a list while
// preserving order.
//
// This example uses the relational predicate `Rembero` to enumerate all
// lists obtained from the input after removing duplicate elements. The
// helper `runGoal` collects and pretty-prints results deterministically
// so the `// Output:` block remains stable. Low-level list constructors
// are shown inline where helpful to illustrate construction of terms.
func ExampleRembero() {
	q := Fresh("q")
	goal := Rembero(NewAtom("a"), List(NewAtom("a"), NewAtom("b"), NewAtom("a")), q)
	for _, s := range runGoal(goal, q) {
		fmt.Println(s)
	}
	// Output:
	// q: ("a" "b")
	// q: ("b" "a")
}

// ExampleReverso demonstrates reversing a list using relational goals.
//
// The `Reverso` relation produces the reversed list as a result. We use
// the `runGoal` helper to execute the relation and print results in a
// deterministic order suitable for documentation extraction.
func ExampleReverso() {
	q := Fresh("q")
	goal := Reverso(List(NewAtom(1), NewAtom(2), NewAtom(3)), q)
	for _, s := range runGoal(goal, q) {
		fmt.Println(s)
	}
	// Output:
	// q: (3 2 1)
}

func ExamplePermuteo() {
	q := Fresh("q")
	goal := Permuteo(List(NewAtom(1), NewAtom(2), NewAtom(3)), q)
	results := runGoal(goal, q)
	fmt.Println(strings.Join(results, "\n"))
	// Output:
	// q: (1 2 3)
	// q: (1 3 2)
	// q: (2 1 3)
	// q: (2 3 1)
	// q: (3 1 2)
	// q: (3 2 1)
}

func ExampleSubseto() {
	q := Fresh("q")
	goal := Subseto(q, List(NewAtom(1), NewAtom(2)))
	results := runGoal(goal, q)
	fmt.Println(strings.Join(results, "\n"))
	// Output:
	// q: ()
	// q: (1 2)
	// q: (1)
	// q: (2)
}

func ExampleLengthoInt() {
	q := Fresh("q")
	goal := LengthoInt(List(NewAtom(1), NewAtom(2), NewAtom(3)), q)
	for _, s := range runGoal(goal, q) {
		fmt.Println(s)
	}
	// Output:
	// q: 3
}

func ExampleFlatteno() {
	q := Fresh("q")
	nested := List(List(NewAtom(1), NewAtom(2)), List(NewAtom(3), List(NewAtom(4), NewAtom(5))))
	goal := Flatteno(nested, q)
	for _, s := range runGoal(goal, q) {
		fmt.Println(s)
	}
	// Output:
	// q: (1 2 3 4 5)
}

func ExampleDistincto() {
	goalSuccess := Distincto(List(NewAtom(1), NewAtom(2), NewAtom(3)))
	resultsSuccess := runGoal(goalSuccess)
	fmt.Printf("Distinct list succeeds: %v\n", len(resultsSuccess) > 0)

	goalFail := Distincto(List(NewAtom(1), NewAtom(2), NewAtom(1)))
	resultsFail := runGoal(goalFail)
	fmt.Printf("Non-distinct list fails: %v\n", len(resultsFail) == 0)
	// Output:
	// Distinct list succeeds: true
	// Non-distinct list fails: true
}

func ExampleNoto() {
	// Noto succeeds because Membero(4, [1,2,3]) fails
	goalSuccess := Noto(Membero(NewAtom(4), List(NewAtom(1), NewAtom(2), NewAtom(3))))
	resultsSuccess := runGoal(goalSuccess)
	fmt.Printf("Noto(fail) succeeds: %v\n", len(resultsSuccess) > 0)

	// Noto fails because Membero(2, [1,2,3]) succeeds
	goalFail := Noto(Membero(NewAtom(2), List(NewAtom(1), NewAtom(2), NewAtom(3))))
	resultsFail := runGoal(goalFail)
	fmt.Printf("Noto(success) fails: %v\n", len(resultsFail) == 0)
	// Output:
	// Noto(fail) succeeds: true
	// Noto(success) fails: true
}
