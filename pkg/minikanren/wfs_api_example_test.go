package minikanren

import (
	"context"
	"fmt"
)

// ExampleNegationTruth demonstrates computing the WFS truth of a negated subgoal
// without enumerating answers. It uses NegationTruth over a simple path relation.
func ExampleSLGEngine_NegationTruth() {
	engine := NewSLGEngine(nil)
	engine.SetStrata(map[string]int{"unreachable": 1, "path": 0})

	// small graph: a->b
	edges := map[string][]string{"a": {"b"}}

	// path/2 evaluator (existence of a direct edge only for brevity)
	pathEval := func(from, to string) GoalEvaluator {
		return func(ctx context.Context, answers chan<- map[int64]Term) error {
			for _, v := range edges[from] {
				if v == to {
					answers <- map[int64]Term{}
				}
			}
			return nil
		}
	}

	// Query not(path(a,c)) => true; not(path(a,b)) => false
	tv1, _ := engine.NegationTruth(context.Background(), "unreachable", NewCallPattern("path", []Term{NewAtom("a"), NewAtom("c")}), pathEval("a", "c"))
	fmt.Println("not(path(a,c)):", tv1)

	tv2, _ := engine.NegationTruth(context.Background(), "unreachable", NewCallPattern("path", []Term{NewAtom("a"), NewAtom("b")}), pathEval("a", "b"))
	fmt.Println("not(path(a,b)):", tv2)

	// Output:
	// not(path(a,c)): true
	// not(path(a,b)): false
}
