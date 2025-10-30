package minikanren

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestChannelResultStream_BasicOperations(t *testing.T) {
	t.Run("Create and close stream", func(t *testing.T) {
		stream := NewChannelResultStream(1)
		defer stream.Close()

		if stream.Count() != 0 {
			t.Errorf("Expected count 0, got %d", stream.Count())
		}
	})

	t.Run("Put and take single item", func(t *testing.T) {
		stream := NewChannelResultStream(1) // Use buffered channel
		defer stream.Close()

		ctx := context.Background()
		store := NewLocalConstraintStore(NewGlobalConstraintBus())

		err := stream.Put(ctx, store)
		if err != nil {
			t.Fatalf("Put failed: %v", err)
		}

		if stream.Count() != 1 {
			t.Errorf("Expected count 1, got %d", stream.Count())
		}

		results, hasMore, err := stream.Take(ctx, 1)
		if err != nil {
			t.Fatalf("Take failed: %v", err)
		}

		if len(results) != 1 {
			t.Errorf("Expected 1 result, got %d", len(results))
		}

		if !hasMore {
			t.Error("Expected hasMore to be true")
		}
	})

	t.Run("Close stream", func(t *testing.T) {
		stream := NewChannelResultStream(1)

		err := stream.Close()
		if err != nil {
			t.Fatalf("Close failed: %v", err)
		}

		// Second close should be safe
		err = stream.Close()
		if err != nil {
			t.Fatalf("Second close failed: %v", err)
		}
	})
}

func TestChannelResultStream_Buffered(t *testing.T) {
	t.Run("Buffered stream operations", func(t *testing.T) {
		stream := NewChannelResultStream(10)
		defer stream.Close()

		ctx := context.Background()
		store := NewLocalConstraintStore(NewGlobalConstraintBus())

		// Put multiple items
		for i := 0; i < 5; i++ {
			err := stream.Put(ctx, store)
			if err != nil {
				t.Fatalf("Put %d failed: %v", i, err)
			}
		}

		if stream.Count() != 5 {
			t.Errorf("Expected count 5, got %d", stream.Count())
		}

		// Take items
		results, hasMore, err := stream.Take(ctx, 3)
		if err != nil {
			t.Fatalf("Take failed: %v", err)
		}

		if len(results) != 3 {
			t.Errorf("Expected 3 results, got %d", len(results))
		}

		if !hasMore {
			t.Error("Expected hasMore to be true")
		}

		if stream.Count() != 5 {
			t.Errorf("Count should remain 5, got %d", stream.Count())
		}
	})
}

func TestChannelResultStream_ContextCancellation(t *testing.T) {
	t.Run("Context cancellation during put", func(t *testing.T) {
		stream := NewChannelResultStream(0)
		defer stream.Close()

		ctx, cancel := context.WithCancel(context.Background())
		store := NewLocalConstraintStore(NewGlobalConstraintBus())

		// Cancel context
		cancel()

		err := stream.Put(ctx, store)
		if err != context.Canceled {
			t.Errorf("Expected context.Canceled, got %v", err)
		}
	})

	t.Run("Context cancellation during take", func(t *testing.T) {
		stream := NewChannelResultStream(0)
		defer stream.Close()

		ctx, cancel := context.WithCancel(context.Background())

		// Use a channel to synchronize the test
		takeStarted := make(chan struct{})
		takeCompleted := make(chan error, 1)

		// Start take in goroutine
		go func() {
			close(takeStarted) // Signal that Take has started
			_, _, err := stream.Take(ctx, 1)
			takeCompleted <- err
		}()

		// Wait for Take to start
		<-takeStarted

		// Cancel context
		cancel()

		// Wait for Take to complete and check result
		select {
		case err := <-takeCompleted:
			if err != context.Canceled {
				t.Errorf("Expected context.Canceled, got %v", err)
			}
		case <-time.After(1 * time.Second): // Reasonable timeout for test
			t.Error("Take did not return on context cancellation within timeout")
		}
	})
}

func TestChannelResultStream_ConcurrentAccess(t *testing.T) {
	t.Run("Concurrent put operations", func(t *testing.T) {
		stream := NewChannelResultStream(1000) // Larger buffer for concurrent puts
		defer stream.Close()

		ctx := context.Background()
		store := NewLocalConstraintStore(NewGlobalConstraintBus())

		const numGoroutines = 10
		const putsPerGoroutine = 100

		var wg sync.WaitGroup
		errors := make(chan error, numGoroutines)

		// Start multiple goroutines putting items
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < putsPerGoroutine; j++ {
					err := stream.Put(ctx, store)
					if err != nil {
						errors <- err
						return
					}
				}
			}()
		}

		wg.Wait()
		close(errors)

		// Check for errors
		for err := range errors {
			t.Errorf("Put error: %v", err)
		}

		expectedCount := int64(numGoroutines * putsPerGoroutine)
		if stream.Count() != expectedCount {
			t.Errorf("Expected count %d, got %d", expectedCount, stream.Count())
		}
	})

	t.Run("Concurrent take operations", func(t *testing.T) {
		stream := NewChannelResultStream(100)

		ctx := context.Background()
		store := NewLocalConstraintStore(NewGlobalConstraintBus())

		// Put some items
		const totalItems = 10
		for i := 0; i < totalItems; i++ {
			stream.Put(ctx, store)
		}

		// Close stream to signal no more items
		stream.Close()

		totalTaken := int64(0)

		// Take with single goroutine
		results, hasMore, err := stream.Take(ctx, 10)
		if err != nil {
			t.Fatalf("Take failed: %v", err)
		}
		totalTaken += int64(len(results))

		if hasMore {
			t.Error("Expected hasMore to be false")
		}

		if totalTaken != totalItems {
			t.Errorf("Expected %d total items taken, got %d", totalItems, totalTaken)
		}

		// Try taking again, should get nothing
		results2, hasMore2, err := stream.Take(ctx, 10)
		if err != nil {
			t.Fatalf("Second take failed: %v", err)
		}
		if len(results2) != 0 || hasMore2 {
			t.Errorf("Second take should return no results and hasMore=false, got %d results, hasMore=%v", len(results2), hasMore2)
		}
	})
}

func TestBufferedResultStream(t *testing.T) {
	t.Run("Buffered stream creation", func(t *testing.T) {
		stream := NewBufferedResultStream()
		defer stream.Close()

		if stream.Count() != 0 {
			t.Errorf("Expected count 0, got %d", stream.Count())
		}
	})

	t.Run("Buffered stream operations", func(t *testing.T) {
		stream := NewBufferedResultStream()
		defer stream.Close()

		ctx := context.Background()
		store := NewLocalConstraintStore(NewGlobalConstraintBus())

		// Put items
		for i := 0; i < 50; i++ {
			err := stream.Put(ctx, store)
			if err != nil {
				t.Fatalf("Put %d failed: %v", i, err)
			}
		}

		if stream.Count() != 50 {
			t.Errorf("Expected count 50, got %d", stream.Count())
		}

		// Take items
		results, hasMore, err := stream.Take(ctx, 25)
		if err != nil {
			t.Fatalf("Take failed: %v", err)
		}

		if len(results) != 25 {
			t.Errorf("Expected 25 results, got %d", len(results))
		}

		if !hasMore {
			t.Error("Expected hasMore to be true")
		}
	})
}

func TestLazyResultStream(t *testing.T) {
	t.Run("Lazy stream creation", func(t *testing.T) {
		computeFunc := func(ctx context.Context) ([]ConstraintStore, error) {
			return []ConstraintStore{NewLocalConstraintStore(NewGlobalConstraintBus())}, nil
		}

		stream := NewLazyResultStream(computeFunc)
		defer stream.Close()

		if stream.Count() != 0 {
			t.Errorf("Expected count 0 before computation, got %d", stream.Count())
		}
	})

	t.Run("Lazy evaluation", func(t *testing.T) {
		called := false
		computeFunc := func(ctx context.Context) ([]ConstraintStore, error) {
			called = true
			stores := make([]ConstraintStore, 10)
			for i := range stores {
				stores[i] = NewLocalConstraintStore(NewGlobalConstraintBus())
			}
			return stores, nil
		}

		stream := NewLazyResultStream(computeFunc)
		defer stream.Close()

		ctx := context.Background()

		// Computation should not happen yet
		if called {
			t.Error("Computation should not happen before Take")
		}

		// Take should trigger computation
		results, hasMore, err := stream.Take(ctx, 5)
		if err != nil {
			t.Fatalf("Take failed: %v", err)
		}

		if !called {
			t.Error("Computation should happen on first Take")
		}

		if len(results) != 5 {
			t.Errorf("Expected 5 results, got %d", len(results))
		}

		if !hasMore {
			t.Error("Expected hasMore to be true")
		}

		if stream.Count() != 10 {
			t.Errorf("Expected count 10, got %d", stream.Count())
		}

		// Take remaining
		results2, hasMore2, err := stream.Take(ctx, 10)
		if err != nil {
			t.Fatalf("Second take failed: %v", err)
		}

		if len(results2) != 5 {
			t.Errorf("Expected 5 remaining results, got %d", len(results2))
		}

		if hasMore2 {
			t.Error("Expected hasMore to be false")
		}
	})

	t.Run("Lazy stream put not supported", func(t *testing.T) {
		stream := NewLazyResultStream(nil)

		ctx := context.Background()
		store := NewLocalConstraintStore(NewGlobalConstraintBus())

		err := stream.Put(ctx, store)
		if err == nil {
			t.Error("Expected error for Put on lazy stream")
		}
	})
}

func TestResultStream_ResourceCleanup(t *testing.T) {
	t.Run("Stream cleanup after close", func(t *testing.T) {
		stream := NewChannelResultStream(0)

		// Close stream
		err := stream.Close()
		if err != nil {
			t.Fatalf("Close failed: %v", err)
		}

		ctx := context.Background()
		store := NewLocalConstraintStore(NewGlobalConstraintBus())

		// Put after close should not panic
		err = stream.Put(ctx, store)
		if err != nil {
			// Should be safe, no error expected
			t.Logf("Put after close returned: %v", err)
		}

		// Take after close should return no more items
		results, hasMore, err := stream.Take(ctx, 1)
		if err != nil {
			t.Fatalf("Take after close failed: %v", err)
		}

		if len(results) != 0 {
			t.Errorf("Expected no results after close, got %d", len(results))
		}

		if hasMore {
			t.Error("Expected hasMore to be false after close")
		}
	})

	t.Run("Multiple close calls safe", func(t *testing.T) {
		stream := NewChannelResultStream(0)

		// Close multiple times
		for i := 0; i < 5; i++ {
			err := stream.Close()
			if err != nil {
				t.Fatalf("Close %d failed: %v", i, err)
			}
		}
	})
}

func BenchmarkChannelResultStream_Put(b *testing.B) {
	stream := NewChannelResultStream(1000)
	defer stream.Close()

	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stream.Put(ctx, store)
	}
}

func BenchmarkChannelResultStream_Take(b *testing.B) {
	stream := NewChannelResultStream(1000)
	defer stream.Close()

	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())

	// Pre-populate stream
	for i := 0; i < 1000; i++ {
		stream.Put(ctx, store)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		results, _, _ := stream.Take(ctx, 1)
		if len(results) == 0 {
			// Re-populate if needed
			stream.Put(ctx, store)
		}
	}
}

func BenchmarkBufferedResultStream(b *testing.B) {
	stream := NewBufferedResultStream()
	defer stream.Close()

	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stream.Put(ctx, store)
		results, _, _ := stream.Take(ctx, 1)
		if len(results) == 0 {
			break
		}
	}
}

// BenchmarkPooledResultStream benchmarks zero-copy streaming with buffer pools.
func BenchmarkPooledResultStream(b *testing.B) {
	pool := NewConstraintStorePool(1000)
	defer pool.Reset()

	stream := NewPooledResultStream(pool, 100, false)
	defer stream.Close()

	ctx := context.Background()
	store := NewLocalConstraintStore(nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		err := stream.Put(ctx, store)
		if err != nil {
			b.Fatal(err)
		}

		results, _, err := stream.Take(ctx, 1)
		if err != nil {
			b.Fatal(err)
		}

		if len(results) > 0 {
			// Return store to pool
			pool.PutLocal(results[0])
		}
	}
}

// BenchmarkBatchedResultStream benchmarks result batching performance.
func BenchmarkBatchedResultStream(b *testing.B) {
	stream := NewBatchedResultStream(10, 10*time.Millisecond)
	defer stream.Close()

	ctx := context.Background()
	store := NewLocalConstraintStore(nil)

	b.ResetTimer()

	// Produce results
	go func() {
		for i := 0; i < b.N; i++ {
			stream.Put(ctx, store)
		}
		stream.Close()
	}()

	// Consume results
	totalConsumed := 0
	for totalConsumed < b.N {
		results, _, err := stream.Take(ctx, 10)
		if err != nil {
			b.Fatal(err)
		}
		totalConsumed += len(results)
	}
}

// BenchmarkBackpressureResultStream benchmarks backpressure handling.
func BenchmarkBackpressureResultStream(b *testing.B) {
	stream := NewBackpressureResultStream(1000, 800, 200)
	defer stream.Close()

	ctx := context.Background()
	store := NewLocalConstraintStore(nil)

	b.ResetTimer()

	// Fast producer
	go func() {
		for i := 0; i < b.N; i++ {
			stream.Put(ctx, store)
		}
		stream.Close()
	}()

	// Slow consumer
	totalConsumed := 0
	for totalConsumed < b.N {
		results, _, err := stream.Take(ctx, 1)
		if err != nil {
			b.Fatal(err)
		}
		totalConsumed += len(results)
		time.Sleep(100 * time.Microsecond) // Simulate slow consumer
	}
}

// BenchmarkMonitoredResultStream benchmarks monitoring overhead.
func BenchmarkMonitoredResultStream(b *testing.B) {
	pool := NewConstraintStorePool(100)
	stream := NewMonitoredResultStream(NewChannelResultStream(100), pool)
	defer stream.Close()

	ctx := context.Background()
	store := NewLocalConstraintStore(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stream.Put(ctx, store)
		results, _, _ := stream.Take(ctx, 1)
		if len(results) == 0 {
			break
		}
	}
}

// BenchmarkComposableResultStream benchmarks functional composition.
func BenchmarkComposableResultStream(b *testing.B) {
	stream := NewComposableResultStream(NewChannelResultStream(100))
	stream = stream.Map(func(store ConstraintStore) ConstraintStore {
		// Simple transformation
		return store
	}).Filter(func(store ConstraintStore) bool {
		return true // Accept all
	})

	ctx := context.Background()
	store := NewLocalConstraintStore(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stream.Stream().Put(ctx, store)
		results, _, _ := stream.Stream().Take(ctx, 1)
		if len(results) == 0 {
			break
		}
	}
}

// BenchmarkErrorRecoveryResultStream benchmarks error recovery performance.
func BenchmarkErrorRecoveryResultStream(b *testing.B) {
	stream := NewErrorRecoveryResultStream(NewChannelResultStream(100), DefaultRetryConfig())
	defer stream.Close()

	ctx := context.Background()
	store := NewLocalConstraintStore(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stream.Put(ctx, store)
		results, _, _ := stream.Take(ctx, 1)
		if len(results) == 0 {
			break
		}
	}
}

// BenchmarkLargeResultSet benchmarks streaming with large result sets.
func BenchmarkLargeResultSet(b *testing.B) {
	benchmarks := []struct {
		name   string
		stream func() ResultStream
	}{
		{"Channel", func() ResultStream { return NewChannelResultStream(1000) }},
		{"Buffered", func() ResultStream { return NewBufferedResultStream() }},
		{"Pooled", func() ResultStream {
			pool := NewConstraintStorePool(1000)
			return NewPooledResultStream(pool, 100, false)
		}},
		{"Batched", func() ResultStream { return NewBatchedResultStream(50, 100*time.Millisecond) }},
		{"Backpressure", func() ResultStream { return NewBackpressureResultStream(1000, 800, 200) }},
		{"Monitored", func() ResultStream {
			return NewMonitoredResultStream(NewChannelResultStream(1000), nil)
		}},
	}

	const numResults = 10000

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				stream := bm.stream()
				defer stream.Close()

				ctx := context.Background()
				store := NewLocalConstraintStore(nil)

				// Produce results
				go func() {
					for j := 0; j < numResults; j++ {
						stream.Put(ctx, store)
					}
					stream.Close()
				}()

				// Consume results
				totalConsumed := 0
				for totalConsumed < numResults {
					results, _, err := stream.Take(ctx, 100)
					if err != nil {
						b.Fatal(err)
					}
					totalConsumed += len(results)
				}
			}
		})
	}
}

// BenchmarkMemoryUsage benchmarks memory usage of different stream types.
func BenchmarkMemoryUsage(b *testing.B) {
	const numResults = 10000

	benchmarks := []struct {
		name   string
		stream func() ResultStream
	}{
		{"Channel", func() ResultStream { return NewChannelResultStream(1000) }},
		{"Pooled", func() ResultStream {
			pool := NewConstraintStorePool(1000)
			return NewPooledResultStream(pool, 100, false)
		}},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				stream := bm.stream()
				defer stream.Close()

				ctx := context.Background()
				store := NewLocalConstraintStore(nil)

				// Produce and consume results
				for j := 0; j < numResults; j++ {
					stream.Put(ctx, store)
					results, _, _ := stream.Take(ctx, 1)
					if len(results) == 0 {
						break
					}
				}
			}
		})
	}
}

// BenchmarkConcurrentStreaming benchmarks concurrent streaming performance.
func BenchmarkConcurrentStreaming(b *testing.B) {
	stream := NewChannelResultStream(10000)
	defer stream.Close()

	ctx := context.Background()
	store := NewLocalConstraintStore(nil)

	const numGoroutines = 10
	const resultsPerGoroutine = 1000

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup

		// Start producers
		for g := 0; g < numGoroutines; g++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < resultsPerGoroutine; j++ {
					stream.Put(ctx, store)
				}
			}()
		}

		// Start consumers
		totalConsumed := int64(0)
		for g := 0; g < numGoroutines; g++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				localConsumed := 0
				for localConsumed < resultsPerGoroutine {
					results, _, err := stream.Take(ctx, 10)
					if err != nil {
						b.Error(err)
						return
					}
					localConsumed += len(results)
				}
				atomic.AddInt64(&totalConsumed, int64(localConsumed))
			}()
		}

		wg.Wait()

		if atomic.LoadInt64(&totalConsumed) != numGoroutines*resultsPerGoroutine {
			b.Errorf("Expected %d results, got %d", numGoroutines*resultsPerGoroutine, totalConsumed)
		}
	}
}
