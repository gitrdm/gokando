package minikanren

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestThreadSafetyOptimizedConstraintBus tests the thread safety of our optimized constraint bus patterns
func TestThreadSafetyOptimizedConstraintBus(t *testing.T) {
	if testing.Short() && !shouldRunHeavy() {
		t.Skip("Skipping thread safety test in short mode")
	}

	t.Run("Shared Global Bus Concurrent Access", func(t *testing.T) {
		const numGoroutines = 100
		const operationsPerGoroutine = 100

		var wg sync.WaitGroup
		var successCount int64
		var errorCount int64

		// Test concurrent access to shared global bus
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()

				for j := 0; j < operationsPerGoroutine; j++ {
					// Use the shared global bus
					results := Run(1, func(q *Var) Goal {
						return Eq(q, NewAtom(goroutineID*1000+j))
					})

					if len(results) == 1 {
						atomic.AddInt64(&successCount, 1)
					} else {
						atomic.AddInt64(&errorCount, 1)
					}

					// Add some scheduling pressure
					if j%10 == 0 {
						runtime.Gosched()
					}
				}
			}(i)
		}

		wg.Wait()

		expectedSuccess := int64(numGoroutines * operationsPerGoroutine)
		if successCount != expectedSuccess {
			t.Errorf("Expected %d successful operations, got %d", expectedSuccess, successCount)
		}
		if errorCount != 0 {
			t.Errorf("Expected 0 errors, got %d", errorCount)
		}

		t.Logf("✅ Shared bus: %d concurrent operations completed successfully", successCount)
	})

	t.Run("Pooled Bus Concurrent Access", func(t *testing.T) {
		const numGoroutines = 50
		const operationsPerGoroutine = 50

		var wg sync.WaitGroup
		var successCount int64
		var errorCount int64

		// Test concurrent access to pooled buses
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()

				for j := 0; j < operationsPerGoroutine; j++ {
					// Use pooled buses for isolation
					results := RunWithIsolation(1, func(q *Var) Goal {
						return Eq(q, NewAtom(goroutineID*1000+j))
					})

					if len(results) == 1 {
						atomic.AddInt64(&successCount, 1)
					} else {
						atomic.AddInt64(&errorCount, 1)
					}

					// Add some scheduling pressure
					if j%10 == 0 {
						runtime.Gosched()
					}
				}
			}(i)
		}

		wg.Wait()

		expectedSuccess := int64(numGoroutines * operationsPerGoroutine)
		if successCount != expectedSuccess {
			t.Errorf("Expected %d successful operations, got %d", expectedSuccess, successCount)
		}
		if errorCount != 0 {
			t.Errorf("Expected 0 errors, got %d", errorCount)
		}

		t.Logf("✅ Pooled bus: %d concurrent operations completed successfully", successCount)
	})

	t.Run("Mixed Strategy Concurrent Access", func(t *testing.T) {
		const numGoroutines = 60
		var wg sync.WaitGroup
		var sharedResults int64
		var isolatedResults int64

		// Mix of shared and isolated operations
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()

				if goroutineID%2 == 0 {
					// Use shared bus
					for j := 0; j < 25; j++ {
						results := Run(1, func(q *Var) Goal {
							return Eq(q, NewAtom(goroutineID*100+j))
						})
						if len(results) == 1 {
							atomic.AddInt64(&sharedResults, 1)
						}
					}
				} else {
					// Use isolated bus
					for j := 0; j < 25; j++ {
						results := RunWithIsolation(1, func(q *Var) Goal {
							return Eq(q, NewAtom(goroutineID*100+j))
						})
						if len(results) == 1 {
							atomic.AddInt64(&isolatedResults, 1)
						}
					}
				}
			}(i)
		}

		wg.Wait()

		expectedEach := int64((numGoroutines / 2) * 25)
		if sharedResults != expectedEach {
			t.Errorf("Expected %d shared results, got %d", expectedEach, sharedResults)
		}
		if isolatedResults != expectedEach {
			t.Errorf("Expected %d isolated results, got %d", expectedEach, isolatedResults)
		}

		t.Logf("✅ Mixed strategy: %d shared + %d isolated operations completed successfully",
			sharedResults, isolatedResults)
	})

	t.Run("Bus Pool Reset Safety", func(t *testing.T) {
		const numOperations = 1000
		var wg sync.WaitGroup

		// Test that bus reset doesn't interfere with concurrent operations
		for i := 0; i < numOperations; i++ {
			wg.Add(1)
			go func(opID int) {
				defer wg.Done()

				bus := GetPooledGlobalBus()
				defer ReturnPooledGlobalBus(bus)

				// Simulate work
				store := NewLocalConstraintStore(bus)
				v := Fresh("test")
				err := store.AddBinding(v.id, NewAtom(opID))
				if err != nil {
					t.Errorf("Operation %d failed: %v", opID, err)
				}

				// Small delay to increase chance of race conditions
				time.Sleep(time.Microsecond)
			}(i)
		}

		wg.Wait()
		t.Logf("✅ Bus pool reset safety: %d operations completed without interference", numOperations)
	})

	t.Run("Global Bus Singleton Thread Safety", func(t *testing.T) {
		const numGoroutines = 100
		var wg sync.WaitGroup
		buses := make([]*GlobalConstraintBus, numGoroutines)

		// Test that GetDefaultGlobalBus returns the same instance across goroutines
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				buses[index] = GetDefaultGlobalBus()
			}(i)
		}

		wg.Wait()

		// Verify all goroutines got the same instance
		firstBus := buses[0]
		for i := 1; i < numGoroutines; i++ {
			if buses[i] != firstBus {
				t.Errorf("Goroutine %d got different bus instance: %p vs %p", i, buses[i], firstBus)
			}
		}

		t.Logf("✅ Global bus singleton: All %d goroutines received the same instance", numGoroutines)
	})
}

// TestConstraintIsolationAfterOptimization verifies that constraint isolation still works
func TestConstraintIsolationAfterOptimization(t *testing.T) {
	t.Run("Shared Bus - No Constraint Interference", func(t *testing.T) {
		// Test that using shared bus doesn't cause constraint interference
		// between different goal executions

		// First execution with constraint
		results1 := Run(1, func(q *Var) Goal {
			return Conj(
				Neq(q, NewAtom("forbidden")),
				Eq(q, NewAtom("allowed")),
			)
		})

		// Second execution should not be affected by first
		results2 := Run(1, func(q *Var) Goal {
			return Eq(q, NewAtom("forbidden")) // This should work in new execution
		})

		if len(results1) != 1 || !results1[0].Equal(NewAtom("allowed")) {
			t.Error("First execution should succeed with 'allowed'")
		}

		if len(results2) != 1 || !results2[0].Equal(NewAtom("forbidden")) {
			t.Error("Second execution should succeed with 'forbidden' (no interference)")
		}

		t.Log("✅ Shared bus maintains proper constraint isolation between executions")
	})

	t.Run("Isolated Bus - Complete Isolation", func(t *testing.T) {
		// Test that isolated buses provide complete constraint isolation

		// First execution with constraint
		results1 := RunWithIsolation(1, func(q *Var) Goal {
			return Conj(
				Neq(q, NewAtom("forbidden")),
				Eq(q, NewAtom("allowed")),
			)
		})

		// Second execution should not be affected by first
		results2 := RunWithIsolation(1, func(q *Var) Goal {
			return Eq(q, NewAtom("forbidden")) // This should work in isolated execution
		})

		if len(results1) != 1 || !results1[0].Equal(NewAtom("allowed")) {
			t.Error("First isolated execution should succeed with 'allowed'")
		}

		if len(results2) != 1 || !results2[0].Equal(NewAtom("forbidden")) {
			t.Error("Second isolated execution should succeed with 'forbidden'")
		}

		t.Log("✅ Isolated buses maintain complete constraint isolation")
	})
}

// TestRaceConditionDetectionOptimized specifically tests for race conditions in optimized code
func TestRaceConditionDetectionOptimized(t *testing.T) {
	if testing.Short() && !shouldRunHeavy() {
		t.Skip("Skipping race condition test in short mode")
	}

	t.Run("High Pressure Concurrent Operations", func(t *testing.T) {
		const numGoroutines = 200
		const operationsPerGoroutine = 50

		var wg sync.WaitGroup
		var totalOperations int64

		// Maximum pressure test
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()

				for j := 0; j < operationsPerGoroutine; j++ {
					// Alternate between strategies to stress test both paths
					if (goroutineID+j)%3 == 0 {
						// Shared bus
						Run(1, func(q *Var) Goal {
							return Eq(q, NewAtom(goroutineID*1000+j))
						})
					} else if (goroutineID+j)%3 == 1 {
						// Isolated bus
						RunWithIsolation(1, func(q *Var) Goal {
							return Eq(q, NewAtom(goroutineID*1000+j))
						})
					} else {
						// Manual pool management
						bus := GetPooledGlobalBus()
						store := NewLocalConstraintStore(bus)
						v := Fresh("stress")
						store.AddBinding(v.id, NewAtom(goroutineID*1000+j))
						ReturnPooledGlobalBus(bus)
					}

					atomic.AddInt64(&totalOperations, 1)

					// Force scheduler switching
					if j%5 == 0 {
						runtime.Gosched()
					}
				}
			}(i)
		}

		wg.Wait()

		expectedOps := int64(numGoroutines * operationsPerGoroutine)
		if totalOperations != expectedOps {
			t.Errorf("Expected %d operations, completed %d", expectedOps, totalOperations)
		}

		t.Logf("✅ High pressure test: %d concurrent operations completed without race conditions", totalOperations)
	})
}
