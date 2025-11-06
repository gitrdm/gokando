package minikanren

import (
	"context"
	"testing"
)

// TestDirectTerminal tests Terminal pattern directly without DefineRule.
func TestDirectTerminal(t *testing.T) {
	input := makeList(NewAtom("1"))
	rest := Fresh("rest")

	// Test Terminal.Expand directly
	pattern := Terminal(NewAtom("1"))
	goal := pattern.Expand(input, rest)

	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	ctx := context.Background()
	stream := goal(ctx, store)

	stores, more := stream.Take(1)
	t.Logf("Direct terminal: got %d stores, more=%v", len(stores), more)

	if len(stores) != 1 {
		t.Fatalf("Expected 1 store, got %d", len(stores))
	}

	// Check that rest is bound to Nil
	binding := stores[0].GetBinding(rest.id)
	t.Logf("rest binding: %v", binding)
	if binding == nil {
		t.Fatal("rest should be bound")
	}
}
