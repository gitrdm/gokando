package parallel

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestExecutionStats(t *testing.T) {
	stats := NewExecutionStats()

	// Test initial state
	if stats.TasksSubmitted != 0 {
		t.Errorf("Expected 0 tasks submitted initially, got %d", stats.TasksSubmitted)
	}

	// Test recording task submission
	stats.RecordTaskSubmitted()
	if stats.TasksSubmitted != 1 {
		t.Errorf("Expected 1 task submitted, got %d", stats.TasksSubmitted)
	}

	// Test recording task completion
	duration := 100 * time.Millisecond
	stats.RecordTaskCompleted(duration)
	if stats.TasksCompleted != 1 {
		t.Errorf("Expected 1 task completed, got %d", stats.TasksCompleted)
	}

	// Test recording task failure
	err := context.DeadlineExceeded
	stats.RecordTaskFailed(err)
	if stats.TasksFailed != 1 {
		t.Errorf("Expected 1 task failed, got %d", stats.TasksFailed)
	}
	if stats.LastError != err {
		t.Errorf("Expected last error to be %v, got %v", err, stats.LastError)
	}

	// Test recording worker count
	stats.RecordWorkerCount(5)
	if stats.PeakWorkerCount != 5 {
		t.Errorf("Expected peak worker count 5, got %d", stats.PeakWorkerCount)
	}

	// Test recording queue depth
	stats.RecordQueueDepth(10)
	if stats.PeakQueueDepth != 10 {
		t.Errorf("Expected peak queue depth 10, got %d", stats.PeakQueueDepth)
	}

	// Test finalization
	stats.Finalize()
	if stats.TotalExecutionTime <= 0 {
		t.Errorf("Expected positive total execution time, got %v", stats.TotalExecutionTime)
	}
}

func TestDeadlockDetector(t *testing.T) {
	dd := NewDeadlockDetector(100*time.Millisecond, 50*time.Millisecond)
	defer dd.Shutdown()

	// Test registering a task
	dd.RegisterTask("task1", "test task")
	if dd.GetActiveTaskCount() != 1 {
		t.Errorf("Expected 1 active task, got %d", dd.GetActiveTaskCount())
	}

	// Test updating a task
	dd.UpdateTask("task1")

	// Test unregistering a task
	dd.UnregisterTask("task1")
	if dd.GetActiveTaskCount() != 0 {
		t.Errorf("Expected 0 active tasks, got %d", dd.GetActiveTaskCount())
	}
}

func TestDeadlockDetectorTimeout(t *testing.T) {
	dd := NewDeadlockDetector(50*time.Millisecond, 25*time.Millisecond)
	defer dd.Shutdown()

	alerts := dd.GetAlerts()

	// Register a task and don't update it
	dd.RegisterTask("slow-task", "slow task")

	// Wait for timeout alert
	select {
	case alert := <-alerts:
		if alert.Type != AlertTaskTimeout {
			t.Errorf("Expected timeout alert, got %v", alert.Type)
		}
		if alert.TaskID != "slow-task" {
			t.Errorf("Expected task ID 'slow-task', got %s", alert.TaskID)
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("Expected timeout alert but none received")
	}
}

func TestWorkerPoolWithStats(t *testing.T) {
	pool := NewDynamicWorkerPoolWithConfig(4, 1, DynamicConfig{
		ScaleUpThreshold:   2,
		ScaleDownThreshold: 1,
		ScaleCheckInterval: 10 * time.Millisecond,
		ScaleCooldown:      5 * time.Millisecond,
	})
	defer pool.Shutdown()

	stats := pool.GetStats()
	if stats == nil {
		t.Error("Expected non-nil stats")
	}

	ctx := context.Background()

	// Submit some tasks
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		task := func() {
			defer wg.Done()
			time.Sleep(10 * time.Millisecond)
		}
		if err := pool.Submit(ctx, task); err != nil {
			t.Errorf("Failed to submit task: %v", err)
		}
	}

	wg.Wait()

	// Check stats after completion
	pool.Shutdown() // This will finalize stats

	finalStats := stats.GetStats()
	if finalStats.TasksSubmitted != 5 {
		t.Errorf("Expected 5 tasks submitted, got %d", finalStats.TasksSubmitted)
	}
	if finalStats.TasksCompleted != 5 {
		t.Errorf("Expected 5 tasks completed, got %d", finalStats.TasksCompleted)
	}
}

func TestWorkStealingPoolWithStats(t *testing.T) {
	pool := NewWorkStealingWorkerPoolWithConfig(4, 1, DynamicConfig{
		ScaleUpThreshold:   3,
		ScaleDownThreshold: 1,
		ScaleCheckInterval: 10 * time.Millisecond,
		ScaleCooldown:      5 * time.Millisecond,
	})
	defer pool.Shutdown()

	stats := pool.GetStats()
	if stats == nil {
		t.Error("Expected non-nil stats")
	}

	ctx := context.Background()

	// Submit tasks that will trigger work stealing
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		task := func() {
			defer wg.Done()
			time.Sleep(5 * time.Millisecond)
		}
		if err := pool.Submit(ctx, task); err != nil {
			t.Errorf("Failed to submit task: %v", err)
		}
	}

	wg.Wait()
	pool.Shutdown()

	finalStats := stats.GetStats()
	if finalStats.TasksCompleted != 10 {
		t.Errorf("Expected 10 tasks completed, got %d", finalStats.TasksCompleted)
	}
}

func TestParallelConstraintPropagator(t *testing.T) {
	t.Skip("ParallelConstraintPropagator test requires minikanren package integration")
}

func BenchmarkWorkerPool(b *testing.B) {
	pool := NewDynamicWorkerPool(4, 1)
	defer pool.Shutdown()

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			task := func() {
				// Simulate some work
				time.Sleep(1 * time.Millisecond)
			}
			pool.Submit(ctx, task)
		}
	})
}

func BenchmarkWorkStealingPool(b *testing.B) {
	pool := NewWorkStealingWorkerPool(4, 1)
	defer pool.Shutdown()

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			task := func() {
				// Simulate some work
				time.Sleep(1 * time.Millisecond)
			}
			pool.Submit(ctx, task)
		}
	})
}
