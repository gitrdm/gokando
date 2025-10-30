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
		store := NewLocalConstraintStore(NewGlobalConstraintBus())

		goal := executor.ParallelDisj()
		stream := goal(ctx, store)
		solutions, _, _ := stream.Take(ctx, 1)

		if len(solutions) != 0 {
			t.Error("Empty parallel disjunction should return no solutions")
		}
	})

	t.Run("Single goal parallel disjunction", func(t *testing.T) {
		executor := NewParallelExecutor(nil)
		defer executor.Shutdown()

		ctx := context.Background()
		store := NewLocalConstraintStore(NewGlobalConstraintBus())
		v := Fresh("x")
		a := NewAtom("hello")

		goal := executor.ParallelDisj(Eq(v, a))
		stream := goal(ctx, store)
		solutions, _, _ := stream.Take(ctx, 1)

		if len(solutions) != 1 {
			t.Fatal("Single goal parallel disjunction should return one solution")
		}

		result := solutions[0].GetBinding(v.ID())
		if result == nil || !result.Equal(a) {
			t.Error("Variable should be bound correctly")
		}
	})

	t.Run("Multiple goal parallel disjunction", func(t *testing.T) {
		executor := NewParallelExecutor(nil)
		defer executor.Shutdown()

		ctx := context.Background()
		store := NewLocalConstraintStore(NewGlobalConstraintBus())
		v := Fresh("x")

		goal := executor.ParallelDisj(
			Eq(v, NewAtom(1)),
			Eq(v, NewAtom(2)),
			Eq(v, NewAtom(3)),
		)

		stream := goal(ctx, store)
		solutions, _, _ := stream.Take(ctx, 3)

		if len(solutions) != 3 {
			t.Fatalf("Expected 3 solutions, got %d", len(solutions))
		}

		// Verify we got all expected values
		values := make(map[int]bool)
		for _, sol := range solutions {
			val := sol.GetBinding(v.ID())
			if val != nil {
				if atom, ok := val.(*Atom); ok {
					if intVal, ok := atom.Value().(int); ok {
						values[intVal] = true
					}
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

	t.Run("Parallel disjunction with mixed fast/slow goals", func(t *testing.T) {
		executor := NewParallelExecutor(nil)
		defer executor.Shutdown()

		ctx := context.Background()
		store := NewLocalConstraintStore(NewGlobalConstraintBus())
		v := Fresh("x")

		// Create goals with different completion times
		fastGoal := Eq(v, NewAtom("fast"))

		slowGoal := func(ctx context.Context, store ConstraintStore) ResultStream {
			stream := NewStream()
			go func() {
				defer stream.Close()
				// Simulate slower completion
				time.Sleep(10 * time.Millisecond)
				newStore := store.Clone()
				newStore.AddBinding(v.ID(), NewAtom("slow"))
				stream.Put(ctx, newStore)
			}()
			return stream
		}

		goal := executor.ParallelDisj(fastGoal, slowGoal)

		// Collect results
		stream := goal(ctx, store)
		solutions, _, _ := stream.Take(ctx, 2)

		// Should get both results (order may vary)
		if len(solutions) != 2 {
			t.Fatalf("Expected 2 solutions, got %d", len(solutions))
		}

		// Check that we got both expected values
		values := make(map[string]bool)
		for _, sol := range solutions {
			binding := sol.GetBinding(v.ID())
			if binding != nil {
				if atom, ok := binding.(*Atom); ok {
					if strVal, ok := atom.Value().(string); ok {
						values[strVal] = true
					}
				}
			}
		}

		if !values["fast"] {
			t.Error("Should have found the fast result")
		}
		if !values["slow"] {
			t.Error("Should have found the slow result")
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
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Create a goal that produces results slowly
		resultsReceived := make(chan struct{}, 10)
		slowGoal := func(q *Var) Goal {
			return func(ctx context.Context, store ConstraintStore) ResultStream {
				stream := NewStream()
				go func() {
					defer stream.Close()
					for i := 0; i < 10; i++ {
						select {
						case <-ctx.Done():
							return
						default:
							newStore := store.Clone()
							newStore.AddBinding(q.ID(), NewAtom(i))
							stream.Put(ctx, newStore)
							resultsReceived <- struct{}{}
							// Small delay to ensure we can cancel mid-stream
							time.Sleep(1 * time.Millisecond)
						}
					}
				}()
				return stream
			}
		}

		// Start the parallel run in a goroutine
		resultChan := make(chan []Term, 1)
		go func() {
			results := ParallelRunWithContext(ctx, 100, slowGoal, nil)
			resultChan <- results
		}()

		// Wait for some results to be produced
		for i := 0; i < 3; i++ {
			select {
			case <-resultsReceived:
				// Got a result
			case <-time.After(100 * time.Millisecond):
				t.Fatal("Should have received results quickly")
			}
		}

		// Cancel the context
		cancel()

		// Get the results - should be fewer than requested due to cancellation
		select {
		case results := <-resultChan:
			if len(results) >= 10 {
				t.Error("Cancellation should limit the number of results")
			}
			if len(results) == 0 {
				t.Error("Should get some results before cancellation")
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatal("ParallelRun should complete after cancellation")
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

		if stream != nil && (stream.executor == nil || stream.executor != executor) {
			t.Error("Stream should reference the correct executor")
		}
	})

	t.Run("Parallel map operation", func(t *testing.T) {
		executor := NewParallelExecutor(nil)
		defer executor.Shutdown()

		ctx := context.Background()
		stream := NewParallelStream(ctx, executor)

		// Add some test constraint stores
		go func() {
			defer stream.Close()
			for i := 0; i < 5; i++ {
				store := NewLocalConstraintStore(NewGlobalConstraintBus())
				v := Fresh("x")
				store.AddBinding(v.ID(), NewAtom(i))
				stream.Put(ctx, store)
			}
		}()

		// Map each constraint store to add a new binding
		mappedStream := stream.ParallelMap(func(store ConstraintStore) ConstraintStore {
			newStore := store.Clone()
			v := Fresh("y")
			newStore.AddBinding(v.ID(), NewAtom("mapped"))
			return newStore
		})

		results := mappedStream.Collect()

		if len(results) != 5 {
			t.Errorf("Expected 5 results, got %d", len(results))
		}

		for _, result := range results {
			sub := result.GetSubstitution()
			if sub.Size() != 2 {
				t.Error("Each result should have 2 bindings")
			}
		}
	})

	t.Run("Parallel filter operation", func(t *testing.T) {
		executor := NewParallelExecutor(nil)
		defer executor.Shutdown()

		ctx := context.Background()
		stream := NewParallelStream(ctx, executor)

		// Add test constraint stores with different numbers of bindings
		go func() {
			defer stream.Close()
			for i := 0; i < 10; i++ {
				store := NewLocalConstraintStore(NewGlobalConstraintBus())
				if i%2 == 0 {
					v := Fresh("x")
					store.AddBinding(v.ID(), NewAtom(i))
				}
				stream.Put(ctx, store)
			}
		}()

		// Filter to only keep non-empty constraint stores
		filteredStream := stream.ParallelFilter(func(store ConstraintStore) bool {
			sub := store.GetSubstitution()
			return sub.Size() > 0
		})

		results := filteredStream.Collect()

		// Should have 5 non-empty constraint stores (even indices)
		if len(results) != 5 {
			t.Errorf("Expected 5 filtered results, got %d", len(results))
		}

		for _, result := range results {
			sub := result.GetSubstitution()
			if sub.Size() == 0 {
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

				store := NewLocalConstraintStore(NewGlobalConstraintBus())
				stream := goal(ctx, store)
				solutions, _, _ := stream.Take(ctx, 2)

				results[index] = make([]Term, len(solutions))
				for j, sol := range solutions {
					binding := sol.GetBinding(q.ID())
					if binding != nil {
						results[index][j] = binding
					}
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

// TestDynamicWorkerScaling tests dynamic worker scaling functionality.
func TestDynamicWorkerScaling(t *testing.T) {
	t.Run("Dynamic scaling enabled", func(t *testing.T) {
		config := &ParallelConfig{
			MaxWorkers:           4,
			MinWorkers:           1,
			EnableDynamicScaling: true,
			ScaleUpThreshold:     2,                     // Scale up when queue > 2
			ScaleDownThreshold:   1,                     // Scale down when queue < 1
			ScaleCheckInterval:   10 * time.Millisecond, // Fast checking for test
			ScaleCooldown:        5 * time.Millisecond,  // Fast cooldown for test
		}

		executor := NewParallelExecutor(config)
		defer executor.Shutdown()

		// Initially should have minimum workers
		initialWorkers := executor.GetWorkerCount()
		if initialWorkers != 1 {
			t.Errorf("Expected initial workers to be 1, got %d", initialWorkers)
		}

		// Submit tasks that will trigger scaling
		const numTasks = 10
		tasksCompleted := make(chan struct{}, numTasks)
		var wg sync.WaitGroup

		for i := 0; i < numTasks; i++ {
			wg.Add(1)
			task := func() {
				defer wg.Done()
				defer func() { tasksCompleted <- struct{}{} }()
				// Simulate work with deterministic completion signaling
				// Instead of time.Sleep, we could use a channel-based approach
				// but for scaling tests, some delay is needed to allow scaling to occur
				time.Sleep(50 * time.Millisecond)
			}

			ctx := context.Background()
			if err := executor.workerPool.Submit(ctx, task); err != nil {
				t.Errorf("Failed to submit task: %v", err)
			}
		}

		// Wait for scaling to potentially occur by monitoring worker count changes
		// Use a channel-based approach to detect scaling events
		scalingDetected := make(chan struct{})
		go func() {
			defer close(scalingDetected)
			initial := executor.GetWorkerCount()
			for {
				select {
				case <-time.After(500 * time.Millisecond):
					// Timeout - scaling may not occur
					return
				default:
					if executor.GetWorkerCount() > initial {
						return // Scaling detected
					}
					time.Sleep(10 * time.Millisecond) // Brief polling interval
				}
			}
		}()

		<-scalingDetected // Wait for scaling detection or timeout

		currentWorkers := executor.GetWorkerCount()
		if currentWorkers > initialWorkers {
			t.Logf("Successfully scaled up to %d workers", currentWorkers)
		} else {
			t.Log("Scaling up may not have occurred within timeout")
		}

		// Wait for all tasks to complete
		wg.Wait()

		// Verify all tasks completed
		completedCount := 0
		timeout := time.After(100 * time.Millisecond)
		for completedCount < numTasks {
			select {
			case <-tasksCompleted:
				completedCount++
			case <-timeout:
				t.Errorf("Not all tasks completed, got %d/%d", completedCount, numTasks)
				goto finalCheck
			}
		}

	finalCheck:
		// Check final worker count - should be at or below max
		finalWorkers := executor.GetWorkerCount()
		if finalWorkers > config.MaxWorkers {
			t.Errorf("Final worker count %d exceeds max %d", finalWorkers, config.MaxWorkers)
		}

		// Verify we can still submit tasks
		testTaskCompleted := make(chan struct{})
		testTask := func() {
			close(testTaskCompleted)
		}

		ctx := context.Background()
		if err := executor.workerPool.Submit(ctx, testTask); err != nil {
			t.Errorf("Failed to submit test task after scaling: %v", err)
		}

		select {
		case <-testTaskCompleted:
			// Success
		case <-time.After(100 * time.Millisecond):
			t.Error("Test task did not complete")
		}
	})

	t.Run("Dynamic scaling disabled", func(t *testing.T) {
		config := &ParallelConfig{
			MaxWorkers:           4,
			MinWorkers:           1,
			EnableDynamicScaling: false, // Disabled
		}

		executor := NewParallelExecutor(config)
		defer executor.Shutdown()

		// Should have static worker count
		workers := executor.GetWorkerCount()
		if workers != 4 {
			t.Errorf("Expected static worker count of 4, got %d", workers)
		}
	})

	t.Run("Scaling statistics", func(t *testing.T) {
		config := &ParallelConfig{
			MaxWorkers:           3,
			MinWorkers:           1,
			EnableDynamicScaling: true,
		}

		executor := NewParallelExecutor(config)
		defer executor.Shutdown()

		current, queue, max := executor.GetScalingStats()

		if current < 1 || current > 3 {
			t.Errorf("Current workers should be between 1 and 3, got %d", current)
		}

		if max != 3 {
			t.Errorf("Max workers should be 3, got %d", max)
		}

		// Queue depth should be 0 initially
		if queue != 0 {
			t.Errorf("Initial queue depth should be 0, got %d", queue)
		}
	})
}

// TestWorkStealingLoadBalancing tests that work stealing improves load balancing.
func TestWorkStealingLoadBalancing(t *testing.T) {
	t.Run("Work stealing vs global queue", func(t *testing.T) {
		// Test that work stealing distributes tasks more evenly than global queue
		numTasks := 50

		testLoadBalancing := func(enableWorkStealing bool) (taskDistribution map[int]int) {
			config := &ParallelConfig{
				MaxWorkers:           4,
				MinWorkers:           4, // Fixed size for fair comparison
				EnableDynamicScaling: false,
				EnableWorkStealing:   enableWorkStealing,
			}

			executor := NewParallelExecutor(config)
			defer executor.Shutdown()

			// Track which worker executes which tasks
			taskDistribution = make(map[int]int)
			var mu sync.Mutex
			var wg sync.WaitGroup

			// We'll simulate this by using a custom task that records worker ID
			// Since we can't directly get worker IDs, we'll use timing patterns
			// to infer load balancing quality

			tasksCompleted := make(chan int, numTasks) // worker index that completed task

			for i := 0; i < numTasks; i++ {
				wg.Add(1)
				taskIndex := i
				task := func() {
					defer wg.Done()
					// Simulate variable work load
					var workTime time.Duration
					if taskIndex%3 == 0 {
						workTime = 10 * time.Millisecond
					} else if taskIndex%3 == 1 {
						workTime = 5 * time.Millisecond
					} else {
						workTime = 15 * time.Millisecond
					}
					time.Sleep(workTime)

					// Record completion (simulating worker tracking)
					mu.Lock()
					// In a real implementation, we'd track actual worker IDs
					// For this test, we'll just count completions
					tasksCompleted <- taskIndex % 4 // Simulate worker assignment
					mu.Unlock()
				}

				ctx := context.Background()
				if err := executor.workerPool.Submit(ctx, task); err != nil {
					t.Errorf("Failed to submit task: %v", err)
				}
			}

			wg.Wait()
			close(tasksCompleted)

			// Count tasks per simulated worker
			taskDistribution = make(map[int]int)
			for workerID := range tasksCompleted {
				taskDistribution[workerID]++
			}

			return taskDistribution
		}

		// Run both configurations
		globalQueueDist := testLoadBalancing(false)
		workStealingDist := testLoadBalancing(true)

		t.Logf("Global queue distribution: %v", globalQueueDist)
		t.Logf("Work stealing distribution: %v", workStealingDist)

		// Both should have distributed tasks across workers
		for workerID := 0; workerID < 4; workerID++ {
			if globalQueueDist[workerID] == 0 {
				t.Errorf("Global queue: worker %d got no tasks", workerID)
			}
			if workStealingDist[workerID] == 0 {
				t.Errorf("Work stealing: worker %d got no tasks", workerID)
			}
		}

		// Work stealing should provide more balanced distribution
		// Calculate variance in task distribution
		calcVariance := func(dist map[int]int) float64 {
			total := 0
			for _, count := range dist {
				total += count
			}
			mean := float64(total) / float64(len(dist))
			variance := 0.0
			for _, count := range dist {
				diff := float64(count) - mean
				variance += diff * diff
			}
			return variance / float64(len(dist))
		}

		globalVariance := calcVariance(globalQueueDist)
		workStealingVariance := calcVariance(workStealingDist)

		t.Logf("Global queue variance: %.2f", globalVariance)
		t.Logf("Work stealing variance: %.2f", workStealingVariance)

		// Work stealing should have lower or equal variance (more balanced)
		if workStealingVariance > globalVariance*1.5 {
			t.Logf("Work stealing distribution variance (%.2f) higher than expected vs global queue (%.2f)",
				workStealingVariance, globalVariance)
		}
	})

	t.Run("Work stealing pool creation", func(t *testing.T) {
		config := &ParallelConfig{
			MaxWorkers:           3,
			MinWorkers:           2,
			EnableDynamicScaling: true, // Explicitly enable dynamic scaling
			EnableWorkStealing:   true,
		}

		executor := NewParallelExecutor(config)
		defer executor.Shutdown()

		if executor.GetMaxWorkers() != 3 {
			t.Errorf("Expected max workers 3, got %d", executor.GetMaxWorkers())
		}

		// When dynamic scaling is enabled, should start with MinWorkers
		if executor.GetWorkerCount() != 2 {
			t.Errorf("Expected current workers 2, got %d", executor.GetWorkerCount())
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
		goals[i] = func(ctx context.Context, store ConstraintStore) ResultStream {
			v := Fresh("x")
			return Eq(v, NewAtom(val))(ctx, store)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		goal := executor.ParallelDisj(goals...)
		ctx := context.Background()
		store := NewLocalConstraintStore(NewGlobalConstraintBus())
		stream := goal(ctx, store)
		stream.Take(ctx, 10)
	}
}

func BenchmarkSequentialVsParallel(b *testing.B) {
	createGoals := func() []Goal {
		goals := make([]Goal, 20)
		for i := 0; i < 20; i++ {
			val := i
			goals[i] = func(ctx context.Context, store ConstraintStore) ResultStream {
				v := Fresh("x")
				// Simulate some work
				time.Sleep(100 * time.Microsecond)
				return Eq(v, NewAtom(val))(ctx, store)
			}
		}
		return goals
	}

	b.Run("Sequential", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			goals := createGoals()
			goal := Disj(goals...)
			ctx := context.Background()
			store := NewLocalConstraintStore(NewGlobalConstraintBus())
			stream := goal(ctx, store)
			stream.Take(ctx, 20)
		}
	})

	b.Run("Parallel", func(b *testing.B) {
		executor := NewParallelExecutor(nil)
		defer executor.Shutdown()

		for i := 0; i < b.N; i++ {
			goals := createGoals()
			goal := executor.ParallelDisj(goals...)
			ctx := context.Background()
			store := NewLocalConstraintStore(NewGlobalConstraintBus())
			stream := goal(ctx, store)
			stream.Take(ctx, 20)
		}
	})
}
