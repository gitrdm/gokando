package minikanren

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestStressRaceConditions performs intensive stress testing for race conditions
// This test uses high concurrency and runs for longer duration to expose
// subtle timing-dependent race conditions that basic tests might miss.
func TestStressRaceConditions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	t.Run("Massive concurrent variable creation", func(t *testing.T) {
		const numGoroutines = 1000
		const variablesPerGoroutine = 100

		vars := make([][]*Var, numGoroutines)
		var wg sync.WaitGroup

		start := time.Now()
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				vars[index] = make([]*Var, variablesPerGoroutine)
				for j := 0; j < variablesPerGoroutine; j++ {
					vars[index][j] = Fresh("stress")
					// Add random tiny delays to increase race exposure
					if j%10 == 0 {
						runtime.Gosched()
					}
				}
			}(i)
		}

		wg.Wait()
		duration := time.Since(start)
		t.Logf("Created %d variables in %d goroutines in %v", numGoroutines*variablesPerGoroutine, numGoroutines, duration)

		// Verify all variables are unique
		ids := make(map[int64]bool)
		totalVars := 0
		for i := 0; i < numGoroutines; i++ {
			for j := 0; j < variablesPerGoroutine; j++ {
				v := vars[i][j]
				if v == nil {
					t.Errorf("Variable [%d][%d] should not be nil", i, j)
					continue
				}
				if ids[v.id] {
					t.Errorf("Duplicate variable ID %d found", v.id)
				}
				ids[v.id] = true
				totalVars++
			}
		}

		expectedVars := numGoroutines * variablesPerGoroutine
		if totalVars != expectedVars {
			t.Errorf("Expected %d unique variables, got %d", expectedVars, totalVars)
		}
	})

	t.Run("Concurrent stream operations stress test", func(t *testing.T) {
		const numProducers = 50
		const numConsumers = 20
		const itemsPerProducer = 20

		stream := NewStream()

		var produced int64
		var consumed int64
		var wg sync.WaitGroup

		// Start consumers first
		for i := 0; i < numConsumers; i++ {
			wg.Add(1)
			go func(consumerID int) {
				defer wg.Done()
				for {
					stores, hasMore := stream.Take(5)
					atomic.AddInt64(&consumed, int64(len(stores)))

					if !hasMore {
						break
					}

					// Random small delay to vary timing
					if consumerID%3 == 0 {
						time.Sleep(time.Microsecond)
					}
				}
			}(i)
		}

		// Start producers
		for i := 0; i < numProducers; i++ {
			wg.Add(1)
			go func(producerID int) {
				defer wg.Done()
				store := NewLocalConstraintStore(NewGlobalConstraintBus())

				for j := 0; j < itemsPerProducer; j++ {
					stream.Put(store)
					atomic.AddInt64(&produced, 1)

					// Vary timing to expose races
					if j%7 == 0 {
						runtime.Gosched()
					}
				}
			}(i)
		}

		// Wait for all producers to finish, then close stream
		go func() {
			time.Sleep(10 * time.Millisecond) // Let producers start
			for atomic.LoadInt64(&produced) < int64(numProducers*itemsPerProducer) {
				time.Sleep(time.Millisecond)
			}
			stream.Close()
		}()

		wg.Wait()

		totalProduced := atomic.LoadInt64(&produced)
		totalConsumed := atomic.LoadInt64(&consumed)

		t.Logf("Produced: %d, Consumed: %d", totalProduced, totalConsumed)

		if totalProduced != int64(numProducers*itemsPerProducer) {
			t.Errorf("Expected to produce %d items, produced %d", numProducers*itemsPerProducer, totalProduced)
		}

		if totalConsumed != totalProduced {
			t.Errorf("Expected to consume %d items, consumed %d", totalProduced, totalConsumed)
		}
	})

	t.Run("Parallel goal execution under stress", func(t *testing.T) {
		const numWorkers = 100
		const goalExecutionsPerWorker = 50

		executor := NewParallelExecutor(&ParallelConfig{
			MaxWorkers:   runtime.NumCPU() * 2,
			MaxQueueSize: 200,
		})
		defer executor.Shutdown()

		var wg sync.WaitGroup
		var successCount int64
		var errorCount int64

		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()

				for j := 0; j < goalExecutionsPerWorker; j++ {
					ctx := context.Background()
					store := NewLocalConstraintStore(NewGlobalConstraintBus())

					v1 := Fresh("x")
					v2 := Fresh("y")

					goal := executor.ParallelDisj(
						Eq(v1, NewAtom(workerID)),
						Eq(v2, NewAtom(j)),
						Conj(
							Eq(v1, NewAtom(workerID)),
							Eq(v2, NewAtom(j)),
						),
					)

					stream := goal(ctx, store)
					solutions, _ := stream.Take(10)

					if len(solutions) > 0 {
						atomic.AddInt64(&successCount, 1)
					} else {
						atomic.AddInt64(&errorCount, 1)
					}

					// Add some timing variation
					if (workerID+j)%13 == 0 {
						runtime.Gosched()
					}
				}
			}(i)
		}

		wg.Wait()

		totalExecutions := int64(numWorkers * goalExecutionsPerWorker)
		actualSuccess := atomic.LoadInt64(&successCount)
		actualErrors := atomic.LoadInt64(&errorCount)

		t.Logf("Total executions: %d, Successes: %d, Errors: %d",
			totalExecutions, actualSuccess, actualErrors)

		if actualSuccess+actualErrors != totalExecutions {
			t.Errorf("Success + Error count (%d) doesn't match total executions (%d)",
				actualSuccess+actualErrors, totalExecutions)
		}

		// We expect most executions to succeed
		if actualSuccess < totalExecutions/2 {
			t.Errorf("Too many failures: expected at least %d successes, got %d",
				totalExecutions/2, actualSuccess)
		}
	})

	t.Run("Constraint store concurrent access chaos test", func(t *testing.T) {
		const numReaders = 30
		const numWriters = 20
		const operationsPerGoroutine = 100

		bus := NewGlobalConstraintBus()
		defer bus.Shutdown()

		store := NewLocalConstraintStore(bus)
		var wg sync.WaitGroup
		var readOps int64
		var writeOps int64

		// Concurrent readers
		for i := 0; i < numReaders; i++ {
			wg.Add(1)
			go func(readerID int) {
				defer wg.Done()

				for j := 0; j < operationsPerGoroutine; j++ {
					// Try to read a binding that may or may not exist
					varID := int64(j % 50) // Limited range for collisions
					_ = store.GetBinding(varID)

					// Get substitution
					_ = store.GetSubstitution()

					// Get constraints
					_ = store.GetConstraints()

					atomic.AddInt64(&readOps, 3)

					// Vary timing
					if j%11 == 0 {
						time.Sleep(time.Nanosecond * 100)
					}
				}
			}(i)
		}

		// Concurrent writers
		for i := 0; i < numWriters; i++ {
			wg.Add(1)
			go func(writerID int) {
				defer wg.Done()

				for j := 0; j < operationsPerGoroutine; j++ {
					// Create a new store (which clones the current one)
					newStore := store.Clone()

					// Add a binding
					varID := int64(writerID*1000 + j) // Unique IDs to avoid conflicts
					term := NewAtom(writerID*1000 + j)
					_ = newStore.AddBinding(varID, term)

					atomic.AddInt64(&writeOps, 1)

					// Vary timing
					if j%7 == 0 {
						runtime.Gosched()
					}
				}
			}(i)
		}

		wg.Wait()

		totalReadOps := atomic.LoadInt64(&readOps)
		totalWriteOps := atomic.LoadInt64(&writeOps)

		expectedReads := int64(numReaders * operationsPerGoroutine * 3)
		expectedWrites := int64(numWriters * operationsPerGoroutine)

		t.Logf("Read operations: %d (expected %d), Write operations: %d (expected %d)",
			totalReadOps, expectedReads, totalWriteOps, expectedWrites)

		if totalReadOps != expectedReads {
			t.Errorf("Read operations mismatch: expected %d, got %d", expectedReads, totalReadOps)
		}

		if totalWriteOps != expectedWrites {
			t.Errorf("Write operations mismatch: expected %d, got %d", expectedWrites, totalWriteOps)
		}
	})
}

// TestMemoryPressureRaces tests race conditions under memory pressure
func TestMemoryPressureRaces(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory pressure test in short mode")
	}

	t.Run("Race detection under memory pressure", func(t *testing.T) {
		// Force GC to run more frequently to expose GC-related races
		oldGC := runtime.GOMAXPROCS(0)
		runtime.GOMAXPROCS(1) // Force more scheduling pressure
		defer runtime.GOMAXPROCS(oldGC)

		const numGoroutines = 200
		var wg sync.WaitGroup

		// Create memory pressure while testing concurrent operations
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				// Allocate memory to create pressure
				data := make([][]byte, 100)
				for j := range data {
					data[j] = make([]byte, 1024)
				}

				// Perform miniKanren operations under memory pressure
				ctx := context.Background()
				store := NewLocalConstraintStore(NewGlobalConstraintBus())

				v := Fresh("pressure")
				goal := Eq(v, NewAtom(id))
				stream := goal(ctx, store)
				solutions, _ := stream.Take(1)

				if len(solutions) != 1 {
					t.Errorf("Worker %d: expected 1 solution, got %d", id, len(solutions))
				}

				// Force GC
				if id%10 == 0 {
					runtime.GC()
				}

				// Use the data to prevent optimization
				_ = data[0][0]
			}(i)
		}

		wg.Wait()
	})
}
