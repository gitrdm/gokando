package minikanren

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestFactStore_BasicOperations(t *testing.T) {
	store := NewFactStore()

	// Test asserting facts
	fact1 := &Fact{
		ID:    "fact1",
		Terms: []Term{NewAtom("parent"), NewAtom("alice"), NewAtom("bob")},
	}

	fact2 := &Fact{
		ID:    "fact2",
		Terms: []Term{NewAtom("parent"), NewAtom("bob"), NewAtom("charlie")},
	}

	err := store.Assert(fact1)
	if err != nil {
		t.Fatalf("Failed to assert fact1: %v", err)
	}

	err = store.Assert(fact2)
	if err != nil {
		t.Fatalf("Failed to assert fact2: %v", err)
	}

	// Test count
	if count := store.Count(); count != 2 {
		t.Errorf("Expected count 2, got %d", count)
	}

	// Test get
	retrieved, exists := store.Get("fact1")
	if !exists {
		t.Fatal("fact1 should exist")
	}
	if retrieved.ID != "fact1" {
		t.Errorf("Expected ID 'fact1', got '%s'", retrieved.ID)
	}

	// Test retract
	if !store.Retract("fact1") {
		t.Fatal("Failed to retract fact1")
	}
	if count := store.Count(); count != 1 {
		t.Errorf("Expected count 1 after retract, got %d", count)
	}
}

func TestFactStore_Query(t *testing.T) {
	store := NewFactStore()

	// Add some facts
	fact1 := &Fact{
		ID:    "fact1",
		Terms: []Term{NewAtom("parent"), NewAtom("alice"), NewAtom("bob")},
	}

	fact2 := &Fact{
		ID:    "fact2",
		Terms: []Term{NewAtom("parent"), NewAtom("bob"), NewAtom("charlie")},
	}

	fact3 := &Fact{
		ID:    "fact3",
		Terms: []Term{NewAtom("sibling"), NewAtom("alice"), NewAtom("david")},
	}

	store.Assert(fact1)
	store.Assert(fact2)
	store.Assert(fact3)

	// Query for all parents
	ctx := context.Background()
	results := store.Query(ctx, NewAtom("parent"), Fresh("x"), Fresh("y"))

	var count int
	for {
		stores, hasMore, err := results.Take(ctx, 10)
		if err != nil {
			t.Fatalf("Error taking results: %v", err)
		}
		count += len(stores)
		if !hasMore {
			break
		}
	}

	if count != 2 {
		t.Errorf("Expected 2 parent facts, got %d", count)
	}

	// Query for specific parent-child relationship
	results = store.Query(ctx, NewAtom("parent"), NewAtom("alice"), Fresh("child"))

	count = 0
	for {
		stores, hasMore, err := results.Take(ctx, 10)
		if err != nil {
			t.Fatalf("Error taking results: %v", err)
		}
		for _, store := range stores {
			count++
			// Should unify the child variable with "bob"
			childVar := Fresh("child")
			if _, success := unifyWithConstraints(NewAtom("bob"), childVar, store); !success {
				t.Error("Failed to unify expected child")
			}
		}
		if !hasMore {
			break
		}
	}

	if count != 1 {
		t.Errorf("Expected 1 alice parent fact, got %d", count)
	}
}

func TestFactStore_Indexing(t *testing.T) {
	store := NewFactStore()

	// Add facts
	for i := 0; i < 100; i++ {
		fact := &Fact{
			ID:    fmt.Sprintf("fact%d", i),
			Terms: []Term{NewAtom("data"), NewAtom(fmt.Sprintf("key%d", i)), NewAtom(fmt.Sprintf("value%d", i))},
		}
		store.Assert(fact)
	}

	// Query with indexing should be efficient
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	results := store.Query(ctx, NewAtom("data"), NewAtom("key50"), Fresh("value"))

	var count int
	for {
		stores, hasMore, err := results.Take(ctx, 10)
		if err != nil {
			if err == context.DeadlineExceeded {
				break // Timeout is expected for large result sets
			}
			t.Fatalf("Error taking results: %v", err)
		}
		count += len(stores)
		if !hasMore {
			break
		}
	}

	if count != 1 {
		t.Errorf("Expected 1 result for specific key, got %d", count)
	}
}

func TestFactStore_CustomIndex(t *testing.T) {
	store := NewFactStore()

	// Add custom index on position 2
	err := store.AddIndex("value_index", []int{2})
	if err != nil {
		t.Fatalf("Failed to add custom index: %v", err)
	}

	// Add facts
	fact1 := &Fact{
		ID:    "fact1",
		Terms: []Term{NewAtom("triple"), NewAtom("a"), NewAtom("x")},
	}

	fact2 := &Fact{
		ID:    "fact2",
		Terms: []Term{NewAtom("triple"), NewAtom("b"), NewAtom("x")},
	}

	store.Assert(fact1)
	store.Assert(fact2)

	// Query using the custom index
	ctx := context.Background()
	results := store.Query(ctx, NewAtom("triple"), Fresh("var"), NewAtom("x"))

	var count int
	for {
		stores, hasMore, err := results.Take(ctx, 10)
		if err != nil {
			t.Fatalf("Error taking results: %v", err)
		}
		count += len(stores)
		if !hasMore {
			break
		}
	}

	if count != 2 {
		t.Errorf("Expected 2 results for value 'x', got %d", count)
	}

	// Test index listing
	indexes := store.ListIndexes()
	if len(indexes) != 2 { // default + custom
		t.Errorf("Expected 2 indexes, got %d", len(indexes))
	}
}
