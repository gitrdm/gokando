package minikanren

import (
	"context"
	"fmt"
)

func ExampleWithTabling() {
	engine := NewSLGEngine(nil)
	eval := WithTabling(engine)

	// Simple evaluator that yields a single answer
	inner := GoalEvaluator(func(ctx context.Context, answers chan<- map[int64]Term) error {
		answers <- map[int64]Term{1: NewAtom("ok")}
		return nil
	})

	ch, err := eval(context.Background(), "demo", []Term{NewAtom("x")}, inner)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	for range ch { /* drain */
	}

	stats := engine.Stats()
	fmt.Printf("evaluations=%d cached=%d\n", stats.TotalEvaluations, stats.CachedSubgoals)
	// Output:
	// evaluations=1 cached=1
}

func ExampleTabledEvaluate() {
	// Use the global engine implicitly
	ResetGlobalEngine()
	inner := GoalEvaluator(func(ctx context.Context, answers chan<- map[int64]Term) error {
		answers <- map[int64]Term{42: NewAtom(1)}
		return nil
	})

	ch, err := TabledEvaluate(context.Background(), "test", []Term{NewAtom("a")}, inner)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	for range ch { /* drain */
	}

	fmt.Println("ok")
	// Output:
	// ok
}
