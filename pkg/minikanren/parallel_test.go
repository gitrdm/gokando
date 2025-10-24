package minikanren

import (
	"context"
	"sync"
	"testing"
	"time"
)

// TestParallelConfig tests parallel configuration.
func TestParallelConfig(t *testing.T) {
	t.Run("Default config", func(t *testing.T) {
		config := DefaultParallelConfig()

		if config.MaxWorkers <= 0 {
			t.Error("Default MaxWorkers should be positive")
		}

		if config.MaxQueueSize <= 0 {
			t.Error("Default MaxQueueSize should be positive")
		}

		if !config.EnableBackpressure {
			t.Error("Default should enable backpressure")
		}
	})
}

// TestParallelExecutor tests the parallel executor.
func TestParallelExecutor(t *testing.T) {
	t.Run("Create and shutdown executor", func(t *testing.T) {
		executor := NewParallelExecutor(nil)
		defer executor.Shutdown()

		if executor == nil {
			t.Error("Executor should not be nil")
		}

		if executor.config == nil {
			t.Error("Executor config should not be nil")
		}
	})

	t.Run("Executor with custom config", func(t *testing.T) {
		config := &ParallelConfig{
			MaxWorkers:         2,
			MaxQueueSize:       10,
			EnableBackpressure: false,
			RateLimit:          100,
		}

		executor := NewParallelExecutor(config)
		defer executor.Shutdown()

		if executor.config.MaxWorkers != 2 {
			t.Error("MaxWorkers should be 2")
		}

		if executor.rateLimiter == nil {
			t.Error("Rate limiter should be created when RateLimit > 0")
		}
	})

	t.Run("Multiple shutdown calls", func(t *testing.T) {
		executor := NewParallelExecutor(nil)

		// Should not panic on multiple shutdowns
		executor.Shutdown()
		executor.Shutdown()
		executor.Shutdown()
	})
}

// TestParallelDisj tests parallel disjunction.
func TestParallelDisj(t *testing.T) {
	t.Run("Empty parallel disjunction", func(t *testing.T) {
		executor := NewParallelExecutor(nil)
		defer executor.Shutdown()

		ctx := context.Background()
		sub := NewSubstitution()

		goal := executor.ParallelDisj()
		stream := goal(ctx, sub)
		solutions, _ := stream.Take(1)

		if len(solutions) != 0 {
			t.Error("Empty parallel disjunction should return no solutions")
		}
	})

	t.Run("Single goal parallel disjunction", func(t *testing.T) {
		executor := NewParallelExecutor(nil)
		defer executor.Shutdown()

		ctx := context.Background()
		sub := NewSubstitution()
		v := Fresh("x")
		a := NewAtom("hello")

		goal := executor.ParallelDisj(Eq(v, a))
		stream := goal(ctx, sub)
		solutions, _ := stream.Take(1)

		if len(solutions) != 1 {
			t.Fatal("Single goal parallel disjunction should return one solution")
		}

		result := solutions[0].Lookup(v)
		if !result.Equal(a) {
			t.Error("Variable should be bound correctly")
		}
	})

	t.Run("Multiple goal parallel disjunction", func(t *testing.T) {
		executor := NewParallelExecutor(nil)
		defer executor.Shutdown()

		ctx := context.Background()
		sub := NewSubstitution()
		v := Fresh("x")

		goal := executor.ParallelDisj(
			Eq(v, NewAtom(1)),
			Eq(v, NewAtom(2)),
			Eq(v, NewAtom(3)),
		)

		stream := goal(ctx, sub)
		solutions, _ := stream.Take(3)

		if len(solutions) != 3 {
			t.Fatalf("Expected 3 solutions, got %d", len(solutions))
		}

		// Verify we got all expected values
		values := make(map[int]bool)
		for _, sol := range solutions {
			val := sol.Lookup(v)
			if atom, ok := val.(*Atom); ok {
				if intVal, ok := atom.Value().(int); ok {
					values[intVal] = true
				}
			}
		}

		expected := []int{1, 2, 3}
		for _, exp := range expected {
			if !values[exp] {
				t.Errorf("Expected to find value %d", exp)
			}
		}
	})

	t.Run("Parallel disjunction with context cancellation", func(t *testing.T) {
		executor := NewParallelExecutor(nil)
		defer executor.Shutdown()

		ctx, cancel := context.WithCancel(context.Background())
		sub := NewSubstitution()
		v := Fresh("x")

		// Create a long-running goal
		slowGoal := func(ctx context.Context, sub *Substitution) *Stream {
			stream := NewStream()
			go func() {
				defer stream.Close()
				select {
				case <-time.After(100 * time.Millisecond):
					stream.Put(sub.Bind(v, NewAtom("slow")))
				case <-ctx.Done():
					return
				}
			}()
			return stream
		}

		goal := executor.ParallelDisj(
			Eq(v, NewAtom("fast")),
			slowGoal,
		)

		// Cancel context quickly
		go func() {
			time.Sleep(10 * time.Millisecond)
			cancel()
		}()

		stream := goal(ctx, sub)
		solutions, _ := stream.Take(2)

		// Should get at least the fast solution
		if len(solutions) == 0 {
			t.Error("Should get at least one solution before cancellation")
		}
	})
}

// TestParallelRun tests parallel run functions.
func TestParallelRun(t *testing.T) {
	t.Run("Simple parallel run", func(t *testing.T) {
		results := ParallelRun(1, func(q *Var) Goal {
			return Eq(q, NewAtom("hello"))
		})

		if len(results) != 1 {
			t.Fatalf("Expected 1 result, got %d", len(results))
		}

		if !results[0].Equal(NewAtom("hello")) {
			t.Error("Result should be 'hello'")
		}
	})

	t.Run("Parallel run with custom config", func(t *testing.T) {
		config := &ParallelConfig{
			MaxWorkers:   1,
			MaxQueueSize: 5,
		}

		results := ParallelRunWithConfig(3, func(q *Var) Goal {
			return Disj(
				Eq(q, NewAtom(1)),
				Eq(q, NewAtom(2)),
				Eq(q, NewAtom(3)),
			)
		}, config)

		if len(results) != 3 {
			t.Fatalf("Expected 3 results, got %d", len(results))
		}
	})

	t.Run("Parallel run with context", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		// This would run longer without timeout
		results := ParallelRunWithContext(ctx, 100, func(q *Var) Goal {
			return Disj(Eq(q, NewAtom(1)), Eq(q, NewAtom(2)))
		}, nil)

		// Should get some results but not 100 due to timeout
		if len(results) > 50 {
			t.Error("Context timeout should limit results")
		}
	})
}

// TestParallelStream tests parallel stream operations.
func TestParallelStream(t *testing.T) {
	t.Run("Create parallel stream", func(t *testing.T) {
		executor := NewParallelExecutor(nil)
		defer executor.Shutdown()

		ctx := context.Background()
		stream := NewParallelStream(ctx, executor)

		if stream == nil {
			t.Error("Parallel stream should not be nil")
		}

		if stream.executor != executor {
			t.Error("Stream should reference the correct executor")
		}
	})

	t.Run("Parallel map operation", func(t *testing.T) {
		executor := NewParallelExecutor(nil)
		defer executor.Shutdown()

		ctx := context.Background()
		stream := NewParallelStream(ctx, executor)

		// Add some test substitutions
		go func() {
			defer stream.Close()
			for i := 0; i < 5; i++ {
				sub := NewSubstitution()
				v := Fresh("x")
				sub = sub.Bind(v, NewAtom(i))
				stream.Put(sub)
			}
		}()

		// Map each substitution to add a new binding
		mappedStream := stream.ParallelMap(func(sub *Substitution) *Substitution {
			v := Fresh("y")
			return sub.Bind(v, NewAtom("mapped"))
		})

		results := mappedStream.Collect()

		if len(results) != 5 {
			t.Errorf("Expected 5 results, got %d", len(results))
		}

		for _, result := range results {
			if result.Size() != 2 {
				t.Error("Each result should have 2 bindings")
			}
		}
	})

	t.Run("Parallel filter operation", func(t *testing.T) {
		executor := NewParallelExecutor(nil)
		defer executor.Shutdown()

		ctx := context.Background()
		stream := NewParallelStream(ctx, executor)

		// Add test substitutions with different sizes
		go func() {
			defer stream.Close()
			for i := 0; i < 10; i++ {
				sub := NewSubstitution()
				if i%2 == 0 {
					v := Fresh("x")
					sub = sub.Bind(v, NewAtom(i))
				}
				stream.Put(sub)
			}
		}()

		// Filter to only keep non-empty substitutions
		filteredStream := stream.ParallelFilter(func(sub *Substitution) bool {
			return sub.Size() > 0
		})

		results := filteredStream.Collect()

		// Should have 5 non-empty substitutions (even indices)
		if len(results) != 5 {
			t.Errorf("Expected 5 filtered results, got %d", len(results))
		}

		for _, result := range results {
			if result.Size() == 0 {
				t.Error("Filtered results should not be empty")
			}
		}
	})
}

// TestConcurrentParallelExecution tests concurrent use of parallel features.
func TestConcurrentParallelExecution(t *testing.T) {
	t.Run("Concurrent parallel runs", func(t *testing.T) {
		const numGoroutines = 10
		results := make([][]Term, numGoroutines)
		var wg sync.WaitGroup

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()

				results[index] = ParallelRun(2, func(q *Var) Goal {
					return Disj(
						Eq(q, NewAtom(index*2)),
						Eq(q, NewAtom(index*2+1)),
					)
				})
			}(i)
		}

		wg.Wait()

		// Verify all goroutines got their expected results
		for i, result := range results {
			if len(result) != 2 {
				t.Errorf("Goroutine %d should get 2 results, got %d", i, len(result))
			}
		}
	})

	t.Run("Shared executor across goroutines", func(t *testing.T) {
		executor := NewParallelExecutor(&ParallelConfig{
			MaxWorkers:   4,
			MaxQueueSize: 20,
		})
		defer executor.Shutdown()

		const numGoroutines = 5
		results := make([][]Term, numGoroutines)
		var wg sync.WaitGroup

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()

				ctx := context.Background()
				q := Fresh("q")
				goal := executor.ParallelDisj(
					Eq(q, NewAtom(index)),
					Eq(q, NewAtom(index+100)),
				)

				sub := NewSubstitution()
				stream := goal(ctx, sub)
				solutions, _ := stream.Take(2)

				results[index] = make([]Term, len(solutions))
				for j, sol := range solutions {
					results[index][j] = sol.Walk(q)
				}
			}(i)
		}

		wg.Wait()

		// Verify all goroutines completed successfully
		for i, result := range results {
			if len(result) != 2 {
				t.Errorf("Goroutine %d should get 2 results, got %d", i, len(result))
			}
		}
	})
}

// Benchmark tests for parallel performance.
func BenchmarkParallelRun(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ParallelRun(1, func(q *Var) Goal {
			return Eq(q, NewAtom(i))
		})
	}
}

func BenchmarkParallelDisjunction(b *testing.B) {
	executor := NewParallelExecutor(nil)
	defer executor.Shutdown()

	goals := make([]Goal, 10)
	for i := 0; i < 10; i++ {
		val := i
		goals[i] = func(ctx context.Context, sub *Substitution) *Stream {
			v := Fresh("x")
			return Eq(v, NewAtom(val))(ctx, sub)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		goal := executor.ParallelDisj(goals...)
		ctx := context.Background()
		sub := NewSubstitution()
		stream := goal(ctx, sub)
		stream.Take(10)
	}
}

func BenchmarkSequentialVsParallel(b *testing.B) {
	createGoals := func() []Goal {
		goals := make([]Goal, 20)
		for i := 0; i < 20; i++ {
			val := i
			goals[i] = func(ctx context.Context, sub *Substitution) *Stream {
				v := Fresh("x")
				// Simulate some work
				time.Sleep(100 * time.Microsecond)
				return Eq(v, NewAtom(val))(ctx, sub)
			}
		}
		return goals
	}

	b.Run("Sequential", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			goals := createGoals()
			goal := Disj(goals...)
			ctx := context.Background()
			sub := NewSubstitution()
			stream := goal(ctx, sub)
			stream.Take(20)
		}
	})

	b.Run("Parallel", func(b *testing.B) {
		executor := NewParallelExecutor(nil)
		defer executor.Shutdown()

		for i := 0; i < b.N; i++ {
			goals := createGoals()
			goal := executor.ParallelDisj(goals...)
			ctx := context.Background()
			sub := NewSubstitution()
			stream := goal(ctx, sub)
			stream.Take(20)
		}
	})
}
