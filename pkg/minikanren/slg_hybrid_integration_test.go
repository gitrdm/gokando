package minikanren

import (
	"context"
	"testing"
)

// TestSLG_Hybrid_Integration_BindingToFD verifies that SLG-derived relational bindings
// flow into the UnifiedStore and trigger FD propagation via the HybridSolver.
//
// Scenario:
//   - FD model with variables X (id=0) and Y (id=1), domains 1..9
//   - Arithmetic constraint: Y = X + 1
//   - SLG evaluator derives relational binding X = 2
//
// Expectation after hybrid propagation:
//   - FD domain(X) = {2}
//   - FD domain(Y) = {3}
//   - Relational bindings contain X=2 and Y=3 (promotion of singleton)
func TestSLG_Hybrid_Integration_BindingToFD(t *testing.T) {
	// 1) Build a small FD model
	model := NewModel()
	x := model.NewVariableWithName(NewBitSetDomain(9), "X") // id = 0
	y := model.NewVariableWithName(NewBitSetDomain(9), "Y") // id = 1

	arith, err := NewArithmetic(x, y, 1) // Y = X + 1
	if err != nil {
		t.Fatalf("failed to create arithmetic constraint: %v", err)
	}
	model.AddConstraint(arith)

	// 2) Create Hybrid solver with both plugins
	fdPlugin := NewFDPlugin(model)
	relPlugin := NewRelationalPlugin()
	solver := NewHybridSolver(relPlugin, fdPlugin)

	// 3) Start from an empty UnifiedStore
	store := NewUnifiedStore()

	// 4) Use SLG to derive a relational binding X=2 (var id 0)
	engine := NewSLGEngine(nil)
	pattern := NewCallPattern("slg_bind", []Term{NewAtom("x")})

	evaluator := GoalEvaluator(func(ctx context.Context, answers chan<- map[int64]Term) error {
		answers <- map[int64]Term{int64(x.ID()): NewAtom(2)}
		return nil
	})

	ctx := context.Background()
	resultChan, err := engine.Evaluate(ctx, pattern, evaluator)
	if err != nil {
		t.Fatalf("SLG Evaluate error: %v", err)
	}

	// Apply SLG answers to the store as relational bindings
	for ans := range resultChan {
		for vid, term := range ans {
			var bindErr error
			store, bindErr = store.AddBinding(vid, term)
			if bindErr != nil {
				t.Fatalf("AddBinding failed: %v", bindErr)
			}
		}
	}

	// Sanity: relational binding for X should be present before propagation
	if got := store.GetBinding(int64(x.ID())); got == nil || !got.Equal(NewAtom(2)) {
		t.Fatalf("expected relational binding X=2 before propagation, got %v", got)
	}

	// 5) Run hybrid propagation to fixed point
	newStore, err := solver.Propagate(store)
	if err != nil {
		t.Fatalf("Hybrid propagation error: %v", err)
	}

	// 6) Check FD domains after propagation
	xDom := newStore.GetDomain(x.ID())
	yDom := newStore.GetDomain(y.ID())
	if xDom == nil || !xDom.IsSingleton() || xDom.SingletonValue() != 2 {
		t.Fatalf("expected X domain {2}, got %v", xDom)
	}
	if yDom == nil || !yDom.IsSingleton() || yDom.SingletonValue() != 3 {
		t.Fatalf("expected Y domain {3}, got %v", yDom)
	}

	// 7) Relational plugin should promote FD singleton Y=3 to a binding
	if got := newStore.GetBinding(int64(y.ID())); got == nil || !got.Equal(NewAtom(3)) {
		t.Fatalf("expected relational binding Y=3 after propagation, got %v", got)
	}
}
