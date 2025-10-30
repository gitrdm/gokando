package minikanren

import (
	"context"
	"log"
	"os"
	"testing"
	"time"
)

// TestContextCancellation tests that context cancellation properly stops execution.
func TestContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a goal that produces results until cancelled
	cancelled := make(chan bool, 1)
	goal := func(ctx context.Context, store ConstraintStore) ResultStream {
		stream := NewStream()
		go func() {
			defer stream.Close()
			defer func() { cancelled <- true }()

			count := 0
			for {
				select {
				case <-ctx.Done():
					return
				default:
					if count >= 10 { // Prevent infinite loop in test
						return
					}
					newStore := store.Clone()
					stream.Put(ctx, newStore)
					count++
				}
			}
		}()
		return stream
	}

	// Start execution and cancel after first result
	done := make(chan bool, 1)
	go func() {
		defer func() { done <- true }()
		results := RunWithContext(ctx, 100, func(q *Var) Goal {
			return goal
		})
		// Should get some results before cancellation
		if len(results) == 0 {
			t.Error("Expected some results before cancellation")
		}
	}()

	// Cancel after a short delay
	time.Sleep(1 * time.Millisecond)
	cancel()

	// Wait for completion
	select {
	case <-done:
		// Good, completed
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Test did not complete within timeout")
	}

	// Verify the goal goroutine was cancelled
	select {
	case <-cancelled:
		// Good, goal was cancelled
	case <-time.After(10 * time.Millisecond):
		t.Error("Goal was not cancelled properly")
	}
}

// TestContextTimeout tests that context timeouts work correctly.
func TestContextTimeout(t *testing.T) {
	// Create a goal that takes some time to complete
	slowGoal := func(ctx context.Context, store ConstraintStore) ResultStream {
		stream := NewStream()
		go func() {
			defer stream.Close()
			select {
			case <-ctx.Done():
				return
			case <-time.After(50 * time.Millisecond): // Simulate slow operation
				stream.Put(ctx, store)
			}
		}()
		return stream
	}

	// Test with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	start := time.Now()
	results := RunWithContext(ctx, 1, func(q *Var) Goal {
		return slowGoal
	})
	elapsed := time.Since(start)

	// Should timeout quickly
	if elapsed > 25*time.Millisecond {
		t.Errorf("Expected timeout within 25ms, but took %v", elapsed)
	}

	// Should get no results due to timeout
	if len(results) != 0 {
		t.Errorf("Expected no results due to timeout, got %d", len(results))
	}
}

// TestContextDeadline tests context deadline handling.
func TestContextDeadline(t *testing.T) {
	// Test deadline that expires during execution
	deadline := time.Now().Add(10 * time.Millisecond)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	start := time.Now()
	results := RunWithContext(ctx, 100, func(q *Var) Goal {
		// Create a goal that would normally take longer than deadline
		return func(ctx context.Context, store ConstraintStore) ResultStream {
			stream := NewStream()
			go func() {
				defer stream.Close()
				for i := 0; i < 10; i++ {
					select {
					case <-ctx.Done():
						return
					case <-time.After(5 * time.Millisecond):
						stream.Put(ctx, store)
					}
				}
			}()
			return stream
		}
	})
	elapsed := time.Since(start)

	// Should respect deadline
	if elapsed > 30*time.Millisecond {
		t.Errorf("Expected completion within 30ms due to deadline, but took %v", elapsed)
	}

	// Should get partial results (deadline expired during execution)
	if len(results) == 0 {
		t.Error("Expected some partial results before deadline")
	}
}

// TestContextMonitor tests the context monitoring functionality.
func TestContextMonitor(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	monitor := NewContextMonitor("test-operation", logger)

	ctx, cancel := monitor.WithContextCancellation(context.Background())

	// Ensure context is not initially cancelled
	select {
	case <-ctx.Done():
		t.Fatal("Context should not be cancelled initially")
	default:
	}

	// Add a cleanup function with proper synchronization
	cleanupCalled := make(chan bool, 1)
	monitor.AddCleanup(func() {
		cleanupCalled <- true
	})

	// Start an operation
	tracker := monitor.StartOperation("test-goal")
	time.Sleep(1 * time.Millisecond)
	tracker.Complete()

	// Cancel the monitored context
	cancel()

	// Wait for cleanup to be called
	select {
	case <-cleanupCalled:
		// Good, cleanup was called
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected cleanup function to be called")
	}

	// Check metrics
	metrics := monitor.GetMetrics()
	if metrics.operationsCompleted != 1 {
		t.Errorf("Expected 1 completed operation, got %d", metrics.operationsCompleted)
	}
}

// TestContextAwareGoal tests wrapping goals with context monitoring.
func TestContextAwareGoal(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	monitor := NewContextMonitor("test-aware-goal", logger)

	x := Fresh("x")
	originalGoal := Eq(x, NewAtom("test"))
	awareGoal := ContextAwareGoal(originalGoal, monitor, "test-eq")

	ctx := context.Background()
	store := NewLocalConstraintStore(nil)

	stream := awareGoal(ctx, store)

	// Take results
	results, _, err := stream.Take(ctx, 1)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	// Check metrics
	metrics := monitor.GetMetrics()
	if metrics.operationsCompleted < 1 {
		t.Error("Expected at least 1 completed operation")
	}
}

// TestParallelContextCancellation tests context cancellation in parallel execution.
func TestParallelContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Track if cancellation was detected
	cancelled := make(chan bool, 1)

	// Create goals that check for cancellation
	slowGoal := func(id int) Goal {
		return func(ctx context.Context, store ConstraintStore) ResultStream {
			stream := NewStream()
			go func() {
				defer stream.Close()
				select {
				case <-ctx.Done():
					cancelled <- true
					return
				case <-time.After(50 * time.Millisecond):
					stream.Put(ctx, store)
				}
			}()
			return stream
		}
	}

	// Start parallel execution
	done := make(chan bool, 1)
	go func() {
		defer func() { done <- true }()
		results := ParallelRunWithContext(ctx, 5, func(q *Var) Goal {
			return NewParallelExecutor(DefaultParallelConfig()).ParallelDisj(
				slowGoal(1), slowGoal(2), slowGoal(3), slowGoal(4), slowGoal(5),
			)
		}, DefaultParallelConfig())

		// Should get no results due to cancellation
		if len(results) != 0 {
			t.Errorf("Expected no results due to cancellation, got %d", len(results))
		}
	}()

	// Cancel quickly
	time.Sleep(5 * time.Millisecond)
	cancel()

	// Wait for completion
	select {
	case <-done:
		// Good, completed
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Parallel execution did not complete within timeout")
	}

	// Verify cancellation was detected
	select {
	case <-cancelled:
		// Good, cancellation was detected
	case <-time.After(10 * time.Millisecond):
		t.Error("Cancellation was not detected in parallel execution")
	}
}

// TestContextPropagationInStreams tests that context is properly propagated through streams.
func TestContextPropagationInStreams(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream := NewStream()

	// Track producer cancellation
	producerCancelled := make(chan bool, 1)

	// Producer goroutine
	go func() {
		defer stream.Close()
		defer func() { producerCancelled <- true }()

		for i := 0; i < 100; i++ {
			select {
			case <-ctx.Done():
				return
			default:
				store := NewLocalConstraintStore(nil)
				if err := stream.Put(ctx, store); err != nil {
					if err == context.Canceled {
						return
					}
					t.Errorf("Unexpected error in producer: %v", err)
					return
				}
			}
		}
	}()

	// Consumer that cancels after getting some results
	results, hasMore, err := stream.Take(ctx, 3)

	// Should get exactly 3 results
	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !hasMore {
		t.Error("Expected more results available")
	}

	// Now cancel and verify propagation
	cancel()

	// Try to take more - should get cancellation error
	_, _, err = stream.Take(ctx, 1)
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}

	// Verify producer was cancelled
	select {
	case <-producerCancelled:
		// Good, producer was cancelled
	case <-time.After(10 * time.Millisecond):
		t.Error("Producer was not cancelled")
	}
}

// TestGoalContextPropagation tests that goals properly check context cancellation.
func TestGoalContextPropagation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// Create a goal that checks context
	goalExecuted := make(chan bool, 1)
	testGoal := func(ctx context.Context, store ConstraintStore) ResultStream {
		stream := NewStream()
		go func() {
			defer stream.Close()
			goalExecuted <- true

			// Check context immediately
			select {
			case <-ctx.Done():
				return
			default:
				stream.Put(ctx, store)
			}
		}()
		return stream
	}

	// Cancel before execution
	cancel()

	results := RunWithContext(ctx, 1, func(q *Var) Goal {
		return testGoal
	})

	// Goal should not have executed due to immediate cancellation check
	select {
	case <-goalExecuted:
		t.Error("Goal should not have executed due to cancelled context")
	case <-time.After(10 * time.Millisecond):
		// Good, goal didn't execute
	}

	if len(results) != 0 {
		t.Errorf("Expected no results due to cancellation, got %d", len(results))
	}
}

// TestStreamContextCancellation tests that streams respect context cancellation.
func TestStreamContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	stream := NewStream()

	// Put operation should fail with cancelled context
	cancel()

	err := stream.Put(ctx, NewLocalConstraintStore(nil))
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}

	// Take operation should also fail
	_, _, err = stream.Take(ctx, 1)
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}

// TestContextInheritance tests that context values are properly inherited.
func TestContextInheritance(t *testing.T) {
	type contextKey string
	testKey := contextKey("test")

	// Create context with value
	ctx := context.WithValue(context.Background(), testKey, "test-value")

	results := RunWithContext(ctx, 1, func(q *Var) Goal {
		return func(ctx context.Context, store ConstraintStore) ResultStream {
			stream := NewStream()
			go func() {
				defer stream.Close()

				// Verify context value is available
				if value := ctx.Value(testKey); value != "test-value" {
					t.Errorf("Expected context value 'test-value', got %v", value)
					return
				}

				stream.Put(ctx, store)
			}()
			return stream
		}
	})

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
}

// TestRunStarWithNilContext tests RunStar with nil context.
func TestRunStarWithNilContext(t *testing.T) {
	results := RunStarWithContext(context.TODO(), func(q *Var) Goal {
		return Disj(Eq(q, NewAtom(1)), Eq(q, NewAtom(2)))
	})

	if len(results) != 2 {
		t.Errorf("Expected 2 results with nil context, got %d", len(results))
	}
}

// BenchmarkContextPropagation benchmarks context propagation overhead.
func BenchmarkContextPropagation(b *testing.B) {
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		RunWithContext(ctx, 1, func(q *Var) Goal {
			return Eq(q, NewAtom(i))
		})
	}
}

// BenchmarkContextCancellation benchmarks context cancellation performance.
func BenchmarkContextCancellation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(1 * time.Microsecond)
			cancel()
		}()

		RunWithContext(ctx, 100, func(q *Var) Goal {
			return Eq(q, NewAtom("test"))
		})
	}
}
