package minikanren

import (
	"context"
	"testing"
)

// TestParseWithSLGDebug debugs the full SLG integration.
func TestParseWithSLGDebug(t *testing.T) {
	engine := NewSLGEngine(nil)
	DefineRule("testDebug", Terminal(NewAtom("1")))
	
	input := makeList(NewAtom("1"))
	rest := Fresh("rest")
	t.Logf("Input: %v, Rest var ID: %d", input, rest.id)
	
	// Call ParseWithSLG
	goal := ParseWithSLG(engine, "testDebug", input, rest)
	
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	ctx := context.Background()
	stream := goal(ctx, store)
	
	stores, more := stream.Take(1)
	t.Logf("Got %d stores, more=%v", len(stores), more)
	
	if len(stores) > 0 {
		binding := stores[0].GetBinding(rest.id)
		t.Logf("rest binding in result store: %v", binding)
	}
}
