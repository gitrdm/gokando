package minikanren

import "context"

// TabledEvaluate is a convenience wrapper that evaluates a tabled predicate
// using the global SLG engine. It constructs the CallPattern from the provided
// predicate identifier and arguments, and runs the supplied evaluator to produce
// answers that will be cached by the engine.
func TabledEvaluate(ctx context.Context, predicateID string, args []Term, evaluator GoalEvaluator) (<-chan map[int64]Term, error) {
	engine := GlobalEngine()
	pattern := NewCallPattern(predicateID, args)
	return engine.Evaluate(ctx, pattern, evaluator)
}

// WithTabling returns a convenience closure bound to the given SLG engine that
// can be used to evaluate tabled predicates without referencing the engine directly.
//
// Example:
//
//	eval := WithTabling(NewSLGEngine(nil))
//	ch, err := eval(ctx, "path", []Term{NewAtom("a"), NewAtom("b")}, myEval)
func WithTabling(engine *SLGEngine) func(ctx context.Context, predicateID string, args []Term, evaluator GoalEvaluator) (<-chan map[int64]Term, error) {
	if engine == nil {
		engine = GlobalEngine()
	}
	return func(ctx context.Context, predicateID string, args []Term, evaluator GoalEvaluator) (<-chan map[int64]Term, error) {
		pattern := NewCallPattern(predicateID, args)
		return engine.Evaluate(ctx, pattern, evaluator)
	}
}
