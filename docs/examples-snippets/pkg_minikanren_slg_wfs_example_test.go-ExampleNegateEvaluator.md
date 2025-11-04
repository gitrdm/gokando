```go
func ExampleNegateEvaluator() {
	engine := NewSLGEngine(nil)
	engine.SetStrata(map[string]int{
		"unreachable": 1,
		"path":        0,
	})

	// Simple graph: a->b, b->c, c->a (cycle). Reachable from a: {b, c, a}.
	edges := map[string][]string{
		"a": {"b"},
		"b": {"c"},
		"c": {"a"},
	}

	// path/2 evaluator (closed over start, end)
	var recPathEval func(start, goal string) GoalEvaluator
	recPathEval = func(start, goal string) GoalEvaluator {
		return func(ctx context.Context, answers chan<- map[int64]Term) error {
			// Base case: direct edge
			for _, to := range edges[start] {
				if to == goal {
					answers <- map[int64]Term{}
				}
				// Recursive case: path(start,to) && path(to,goal)
				if to != goal {
					// Evaluate recursively; this will register dependency via context.
					pat := NewCallPattern("path", []Term{NewAtom(to), NewAtom(goal)})
					_, _ = engine.Evaluate(ctx, pat, recPathEval(to, goal))
				}
			}
			return nil
		}
	}

	// unreachable/2(X,Y) :- not(path(X,Y))
	negPath := func(x, y string) GoalEvaluator {
		pat := NewCallPattern("path", []Term{NewAtom(x), NewAtom(y)})
		return NegateEvaluator(engine, "unreachable", pat, recPathEval(x, y))
	}

	ctx := context.Background()
	// Query: unreachable(a, d) where d is not in the graph should succeed; unreachable(a, b) should fail.
	res1, _ := engine.Evaluate(ctx, NewCallPattern("unreachable", []Term{NewAtom("a"), NewAtom("d")}), negPath("a", "d"))
	count1 := 0
	for range res1 {
		count1++
	}
	fmt.Printf("unreachable(a,d): %d answers\n", count1)

	res2, _ := engine.Evaluate(ctx, NewCallPattern("unreachable", []Term{NewAtom("a"), NewAtom("b")}), negPath("a", "b"))
	count2 := 0
	for range res2 {
		count2++
	}
	fmt.Printf("unreachable(a,b): %d answers\n", count2)

	// Output:
	// unreachable(a,d): 1 answers
	// unreachable(a,b): 0 answers
}

```


