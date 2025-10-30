// Package minikanren provides context utilities for enhanced execution model.
// This file implements context-aware debugging, tracing, and monitoring
// capabilities for the miniKanren execution engine.
//
// The context utilities provide:
//   - Structured logging for context operations
//   - Context deadline and timeout monitoring
//   - Execution tracing with performance metrics
//   - Graceful cancellation with cleanup coordination
package minikanren

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// ContextMonitor provides monitoring and tracing capabilities for context operations.
// It tracks context lifecycle, cancellation events, and execution metrics.
type ContextMonitor struct {
	// operationID uniquely identifies this monitoring session
	operationID string

	// startTime tracks when monitoring began
	startTime time.Time

	// logger provides structured logging (can be nil for no logging)
	logger *log.Logger

	// metrics tracks execution statistics
	metrics *ContextMetrics

	// cleanupFuncs holds functions to call on cancellation
	cleanupFuncs []func()

	// mu protects concurrent access
	mu sync.RWMutex

	// cancelled tracks if context has been cancelled
	cancelled bool
}

// ContextMetrics tracks performance and execution statistics for context operations.
type ContextMetrics struct {
	// operationsStarted counts total operations initiated
	operationsStarted int64

	// operationsCompleted counts successfully completed operations
	operationsCompleted int64

	// operationsCancelled counts operations cancelled due to context
	operationsCancelled int64

	// totalExecutionTime tracks cumulative execution time
	totalExecutionTime time.Duration

	// averageExecutionTime caches computed average
	averageExecutionTime time.Duration

	// lastOperationTime tracks when the last operation completed
	lastOperationTime time.Time
}

// NewContextMonitor creates a new context monitor with optional logging.
func NewContextMonitor(operationID string, logger *log.Logger) *ContextMonitor {
	return &ContextMonitor{
		operationID: operationID,
		startTime:   time.Now(),
		logger:      logger,
		metrics:     &ContextMetrics{},
	}
}

// WithContextCancellation creates a context that monitors cancellation and cleanup.
func (cm *ContextMonitor) WithContextCancellation(ctx context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(ctx)

	// Monitor for cancellation in background
	go func() {
		<-ctx.Done()
		cm.mu.Lock()
		cm.cancelled = true
		cm.mu.Unlock()

		if cm.logger != nil {
			cm.logger.Printf("[ContextMonitor:%s] Context cancelled: %v", cm.operationID, ctx.Err())
		}

		// Execute cleanup functions
		cm.mu.RLock()
		cleanupFuncs := make([]func(), len(cm.cleanupFuncs))
		copy(cleanupFuncs, cm.cleanupFuncs)
		cm.mu.RUnlock()

		for _, cleanup := range cleanupFuncs {
			func() {
				defer func() {
					if r := recover(); r != nil && cm.logger != nil {
						cm.logger.Printf("[ContextMonitor:%s] Cleanup panic: %v", cm.operationID, r)
					}
				}()
				cleanup()
			}()
		}
	}()

	return ctx, cancel
}

// AddCleanup registers a cleanup function to be called on context cancellation.
func (cm *ContextMonitor) AddCleanup(cleanup func()) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if !cm.cancelled {
		cm.cleanupFuncs = append(cm.cleanupFuncs, cleanup)
	}
}

// StartOperation marks the beginning of an operation for monitoring.
func (cm *ContextMonitor) StartOperation(operationName string) *OperationTracker {
	if cm.logger != nil {
		cm.logger.Printf("[ContextMonitor:%s] Starting operation: %s", cm.operationID, operationName)
	}

	cm.mu.Lock()
	cm.metrics.operationsStarted++
	cm.mu.Unlock()

	return &OperationTracker{
		monitor:       cm,
		operationName: operationName,
		startTime:     time.Now(),
	}
}

// OperationTracker tracks individual operation execution.
type OperationTracker struct {
	monitor       *ContextMonitor
	operationName string
	startTime     time.Time
	completed     bool
}

// Complete marks the operation as successfully completed.
func (ot *OperationTracker) Complete() {
	if ot.completed {
		return
	}
	ot.completed = true

	duration := time.Since(ot.startTime)

	ot.monitor.mu.Lock()
	ot.monitor.metrics.operationsCompleted++
	ot.monitor.metrics.totalExecutionTime += duration
	ot.monitor.metrics.lastOperationTime = time.Now()

	// Update average
	totalOps := ot.monitor.metrics.operationsCompleted + ot.monitor.metrics.operationsCancelled
	if totalOps > 0 {
		ot.monitor.metrics.averageExecutionTime = ot.monitor.metrics.totalExecutionTime / time.Duration(totalOps)
	}
	ot.monitor.mu.Unlock()

	if ot.monitor.logger != nil {
		ot.monitor.logger.Printf("[ContextMonitor:%s] Operation completed: %s (duration: %v)",
			ot.monitor.operationID, ot.operationName, duration)
	}
}

// Cancel marks the operation as cancelled.
func (ot *OperationTracker) Cancel() {
	if ot.completed {
		return
	}
	ot.completed = true

	duration := time.Since(ot.startTime)

	ot.monitor.mu.Lock()
	ot.monitor.metrics.operationsCancelled++
	ot.monitor.metrics.totalExecutionTime += duration
	ot.monitor.metrics.lastOperationTime = time.Now()

	// Update average
	totalOps := ot.monitor.metrics.operationsCompleted + ot.monitor.metrics.operationsCancelled
	if totalOps > 0 {
		ot.monitor.metrics.averageExecutionTime = ot.monitor.metrics.totalExecutionTime / time.Duration(totalOps)
	}
	ot.monitor.mu.Unlock()

	if ot.monitor.logger != nil {
		ot.monitor.logger.Printf("[ContextMonitor:%s] Operation cancelled: %s (duration: %v)",
			ot.monitor.operationID, ot.operationName, duration)
	}
}

// CheckContextCancellation checks if the context is cancelled and handles cleanup.
func CheckContextCancellation(ctx context.Context, monitor *ContextMonitor, operationName string) error {
	select {
	case <-ctx.Done():
		if monitor != nil {
			if monitor.logger != nil {
				monitor.logger.Printf("[ContextMonitor:%s] Context cancelled during %s: %v",
					monitor.operationID, operationName, ctx.Err())
			}
		}
		return ctx.Err()
	default:
		return nil
	}
}

// WithContextTimeout creates a context with timeout monitoring.
func WithContextTimeout(parent context.Context, timeout time.Duration, monitor *ContextMonitor) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(parent, timeout)

	if monitor != nil && monitor.logger != nil {
		deadline, ok := ctx.Deadline()
		if ok {
			monitor.logger.Printf("[ContextMonitor:%s] Context timeout set: %v (deadline: %v)",
				monitor.operationID, timeout, deadline)
		}
	}

	return ctx, cancel
}

// ContextAwareGoal wraps a goal with context monitoring and tracing.
func ContextAwareGoal(goal Goal, monitor *ContextMonitor, goalName string) Goal {
	return func(ctx context.Context, store ConstraintStore) ResultStream {
		tracker := monitor.StartOperation(fmt.Sprintf("goal:%s", goalName))
		defer tracker.Complete()

		// Check for immediate cancellation
		if err := CheckContextCancellation(ctx, monitor, goalName); err != nil {
			tracker.Cancel()
			stream := NewStream()
			stream.Close()
			return stream
		}

		// Execute the goal with monitoring
		stream := goal(ctx, store)

		// Wrap the stream to monitor operations
		return &MonitoredStream{
			ResultStream: stream,
			monitor:      monitor,
			operation:    goalName,
		}
	}
}

// MonitoredStream wraps a ResultStream with monitoring capabilities.
type MonitoredStream struct {
	ResultStream
	monitor   *ContextMonitor
	operation string
}

// Take wraps the Take operation with monitoring.
func (ms *MonitoredStream) Take(ctx context.Context, n int) ([]ConstraintStore, bool, error) {
	tracker := ms.monitor.StartOperation(fmt.Sprintf("stream-take:%s", ms.operation))
	defer tracker.Complete()

	stores, hasMore, err := ms.ResultStream.Take(ctx, n)

	if err != nil {
		tracker.Cancel()
		if ms.monitor.logger != nil {
			ms.monitor.logger.Printf("[ContextMonitor:%s] Stream take failed for %s: %v",
				ms.monitor.operationID, ms.operation, err)
		}
	}

	return stores, hasMore, err
}

// Put wraps the Put operation with monitoring.
func (ms *MonitoredStream) Put(ctx context.Context, store ConstraintStore) error {
	tracker := ms.monitor.StartOperation(fmt.Sprintf("stream-put:%s", ms.operation))
	defer tracker.Complete()

	err := ms.ResultStream.Put(ctx, store)

	if err != nil {
		tracker.Cancel()
		if ms.monitor.logger != nil {
			ms.monitor.logger.Printf("[ContextMonitor:%s] Stream put failed for %s: %v",
				ms.monitor.operationID, ms.operation, err)
		}
	}

	return err
}

// GetMetrics returns a copy of the current context metrics.
func (cm *ContextMonitor) GetMetrics() *ContextMetrics {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return &ContextMetrics{
		operationsStarted:    cm.metrics.operationsStarted,
		operationsCompleted:  cm.metrics.operationsCompleted,
		operationsCancelled:  cm.metrics.operationsCancelled,
		totalExecutionTime:   cm.metrics.totalExecutionTime,
		averageExecutionTime: cm.metrics.averageExecutionTime,
		lastOperationTime:    cm.metrics.lastOperationTime,
	}
}

// String returns a human-readable representation of the context monitor.
func (cm *ContextMonitor) String() string {
	metrics := cm.GetMetrics()
	return fmt.Sprintf("ContextMonitor{operationID: %s, started: %v, completed: %d, cancelled: %d, avg_time: %v}",
		cm.operationID, cm.startTime, metrics.operationsCompleted, metrics.operationsCancelled, metrics.averageExecutionTime)
}
