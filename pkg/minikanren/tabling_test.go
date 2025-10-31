package minikanren

import (
	"context"
	"testing"
	"time"
)

func TestTabledGoal_BasicCaching(t *testing.T) {
	// Test that tabled goals cache results correctly
	ctx := context.Background()
	manager := NewTableManager()

	// Create a simple goal that succeeds once
	callCount := 0
	goal := func(ctx context.Context, store ConstraintStore) ResultStream {
		callCount++
		stream := NewStream()
		go func() {
			defer stream.Close()
			stream.Put(ctx, store)
		}()
		return stream
	}

	tabledGoal := NewTabledGoal(goal, manager)

	// First execution should call the goal
	results, _, err := tabledGoal.Execute(ctx, NewLocalConstraintStore(nil)).Take(ctx, 10)
	if err != nil {
		t.Fatalf("Error taking results: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
	if callCount != 1 {
		t.Errorf("Expected goal to be called 1 time, got %d", callCount)
	}

	// Second execution should use cached result (goal not called again)
	results, _, err = tabledGoal.Execute(ctx, NewLocalConstraintStore(nil)).Take(ctx, 10)
	if err != nil {
		t.Fatalf("Error taking results: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
	if callCount != 1 {
		t.Errorf("Expected goal to still be called 1 time, got %d", callCount)
	}
}

func TestTableGoal(t *testing.T) {
	// Test the TableGoal convenience function
	ctx := context.Background()

	callCount := 0
	baseGoal := func(ctx context.Context, store ConstraintStore) ResultStream {
		callCount++
		stream := NewStream()
		go func() {
			defer stream.Close()
			stream.Put(ctx, store)
		}()
		return stream
	}

	tabledGoal := TableGoal(baseGoal)

	// Execute twice
	results, _, err := tabledGoal(ctx, NewLocalConstraintStore(nil)).Take(ctx, 10)
	if err != nil {
		t.Fatalf("Error taking results: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	results, _, err = tabledGoal(ctx, NewLocalConstraintStore(nil)).Take(ctx, 10)
	if err != nil {
		t.Fatalf("Error taking results: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	if callCount != 1 {
		t.Errorf("Expected goal to be called only once, got %d", callCount)
	}
}

func TestTableManager_GetOrCreateTable(t *testing.T) {
	manager := NewTableManager()

	// Simple goal for testing
	goal := func(ctx context.Context, store ConstraintStore) ResultStream {
		stream := NewStream()
		go func() {
			defer stream.Close()
			stream.Put(ctx, store)
		}()
		return stream
	}

	store := NewLocalConstraintStore(nil)

	// First call should create a new table
	table1 := manager.GetOrCreateTable(goal, store)
	if table1 == nil {
		t.Fatal("Expected table to be created")
	}

	// Second call should return the same table
	table2 := manager.GetOrCreateTable(goal, store)
	if table1 != table2 {
		t.Error("Expected same table to be returned")
	}

	// Check statistics
	stats := manager.GetStats()
	if stats.TotalTables != 1 {
		t.Errorf("Expected 1 table, got %d", stats.TotalTables)
	}
	if stats.TotalHits != 1 {
		t.Errorf("Expected 1 hit, got %d", stats.TotalHits)
	}
	if stats.TotalMisses != 1 {
		t.Errorf("Expected 1 miss, got %d", stats.TotalMisses)
	}
}

func TestTableManager_Limits(t *testing.T) {
	// Test table limits
	manager := NewTableManagerWithConfig(2, 1000, time.Hour) // Max 2 tables

	goal1 := func(ctx context.Context, store ConstraintStore) ResultStream {
		stream := NewStream()
		go func() {
			defer stream.Close()
			stream.Put(ctx, store)
		}()
		return stream
	}

	goal2 := func(ctx context.Context, store ConstraintStore) ResultStream {
		stream := NewStream()
		go func() {
			defer stream.Close()
			stream.Put(ctx, store)
		}()
		return stream
	}

	goal3 := func(ctx context.Context, store ConstraintStore) ResultStream {
		stream := NewStream()
		go func() {
			defer stream.Close()
			stream.Put(ctx, store)
		}()
		return stream
	}

	store := NewLocalConstraintStore(nil)

	// Create first two tables
	manager.GetOrCreateTable(goal1, store)
	manager.GetOrCreateTable(goal2, store)

	stats := manager.GetStats()
	if stats.TotalTables != 2 {
		t.Errorf("Expected 2 tables, got %d", stats.TotalTables)
	}

	// Third table should trigger eviction
	manager.GetOrCreateTable(goal3, store)

	stats = manager.GetStats()
	if stats.TotalTables != 2 {
		t.Errorf("Expected still 2 tables after eviction, got %d", stats.TotalTables)
	}
	if stats.TablesEvicted != 1 {
		t.Errorf("Expected 1 table evicted, got %d", stats.TablesEvicted)
	}
}

func TestGlobalTableManager(t *testing.T) {
	// Test global table manager
	manager1 := GetGlobalTableManager()
	manager2 := GetGlobalTableManager()

	if manager1 != manager2 {
		t.Error("Expected same global manager instance")
	}

	// Test setting global manager
	customManager := NewTableManager()
	SetGlobalTableManager(customManager)

	manager3 := GetGlobalTableManager()
	if customManager != manager3 {
		t.Error("Expected custom manager to be set")
	}
}
