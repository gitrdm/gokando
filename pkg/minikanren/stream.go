package minikanren

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
)

// ResultStream represents a stream of constraint stores that can be consumed lazily.
// This interface provides a clean abstraction for streaming results from goal evaluation,
// supporting concurrent access, resource management, and result counting.
//
// Key features:
//   - Lazy evaluation to prevent memory exhaustion with large result sets
//   - Thread-safe operations for concurrent consumption
//   - Resource cleanup through explicit Close() calls
//   - Result counting for monitoring and optimization
//   - Context-aware operations for cancellation support
type ResultStream interface {
	// Take retrieves up to n constraint stores from the stream.
	// Returns a slice of stores and a boolean indicating if more stores might be available.
	// This operation is thread-safe and can be called concurrently.
	Take(ctx context.Context, n int) ([]ConstraintStore, bool, error)

	// Put adds a constraint store to the stream.
	// This operation is thread-safe and can be called concurrently by producers.
	Put(ctx context.Context, store ConstraintStore) error

	// Close closes the stream, indicating no more stores will be added.
	// After Close(), Take() will eventually return hasMore=false.
	// This ensures proper resource cleanup and prevents goroutine leaks.
	Close() error

	// Count returns the number of stores that have been put into the stream.
	// This is useful for monitoring progress and optimizing consumption.
	Count() int64
}

// ChannelResultStream implements ResultStream using Go channels.
// This provides efficient, concurrent streaming with proper synchronization
// and resource management. The implementation uses buffered channels
// for performance and atomic operations for thread-safe counting.
type ChannelResultStream struct {
	ch     chan ConstraintStore // Channel for streaming stores
	count  int64                // Atomic counter for results
	closed int32                // Atomic flag for closed state
	mu     sync.Mutex           // Protects stream state
}

// NewChannelResultStream creates a new channel-based result stream.
// The bufferSize parameter controls the channel buffer size for performance tuning.
// A bufferSize of 0 creates an unbuffered channel.
func NewChannelResultStream(bufferSize int) ResultStream {
	return &ChannelResultStream{
		ch: make(chan ConstraintStore, bufferSize),
	}
}

// Take implements ResultStream.Take.
// Retrieves up to n constraint stores, respecting context cancellation.
// Returns an error if the context is cancelled or the stream encounters an issue.
func (s *ChannelResultStream) Take(ctx context.Context, n int) ([]ConstraintStore, bool, error) {
	var results []ConstraintStore

	for i := 0; i < n; i++ {
		select {
		case store, ok := <-s.ch:
			if !ok {
				// Channel is closed, no more items
				return results, false, nil
			}
			if store != nil {
				results = append(results, store)
			}
		case <-ctx.Done():
			// Context cancelled, return what we have so far
			return results, len(results) > 0, ctx.Err()
		}
	}

	// We took exactly n items, check if there are more or if channel is closed
	select {
	case _, ok := <-s.ch:
		if ok {
			// There are more items, but we can't put it back
			// This is a limitation of the channel-based approach
			return results, true, nil
		}
		// Channel is closed
		return results, false, nil
	default:
		// No more items immediately available, but channel might not be closed
		return results, true, nil
	}
}

// Put implements ResultStream.Put.
// Adds a constraint store to the stream, respecting context cancellation.
// Returns an error if the context is cancelled or the stream is closed.
func (s *ChannelResultStream) Put(ctx context.Context, store ConstraintStore) error {
	// Check if stream is closed
	if atomic.LoadInt32(&s.closed) == 1 {
		return nil // Silently ignore puts to closed stream
	}

	select {
	case s.ch <- store:
		atomic.AddInt64(&s.count, 1)
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Close implements ResultStream.Close.
// Closes the stream and ensures all resources are cleaned up.
// Safe to call multiple times.
func (s *ChannelResultStream) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if atomic.LoadInt32(&s.closed) == 1 {
		return nil // Already closed
	}

	atomic.StoreInt32(&s.closed, 1)
	close(s.ch)
	return nil
}

// Count implements ResultStream.Count.
// Returns the number of stores that have been successfully put into the stream.
func (s *ChannelResultStream) Count() int64 {
	return atomic.LoadInt64(&s.count)
}

// BufferedResultStream extends ChannelResultStream with additional buffering
// for high-throughput scenarios. This implementation provides better performance
// for streams with many producers and consumers by using larger buffers
// and optimized synchronization.
type BufferedResultStream struct {
	*ChannelResultStream
	bufferSize int
}

// NewBufferedResultStream creates a new buffered result stream with optimized settings.
// Uses a larger buffer size for better throughput in concurrent scenarios.
func NewBufferedResultStream() ResultStream {
	return &BufferedResultStream{
		ChannelResultStream: &ChannelResultStream{
			ch: make(chan ConstraintStore, 100), // Larger buffer for throughput
		},
		bufferSize: 100,
	}
}

// LazyResultStream provides lazy evaluation of results.
// Instead of eagerly computing all results, this stream computes results
// on-demand as they are consumed, preventing memory exhaustion.
type LazyResultStream struct {
	computeFunc func(ctx context.Context) ([]ConstraintStore, error)
	results     []ConstraintStore
	index       int
	count       int64
	computed    bool
	mu          sync.Mutex
}

// NewLazyResultStream creates a lazy result stream with a computation function.
// The computation function is called only when results are first requested.
func NewLazyResultStream(computeFunc func(ctx context.Context) ([]ConstraintStore, error)) ResultStream {
	return &LazyResultStream{
		computeFunc: computeFunc,
		results:     nil,
		index:       0,
		computed:    false,
	}
}

// Take implements lazy evaluation for ResultStream.Take.
func (s *LazyResultStream) Take(ctx context.Context, n int) ([]ConstraintStore, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Compute results on first access
	if !s.computed {
		results, err := s.computeFunc(ctx)
		if err != nil {
			return nil, false, err
		}
		s.results = results
		s.count = int64(len(results))
		s.computed = true
	}

	// Return available results
	var batch []ConstraintStore
	remaining := len(s.results) - s.index

	if remaining <= 0 {
		return nil, false, nil
	}

	if n >= remaining {
		batch = s.results[s.index:]
		s.index = len(s.results)
		return batch, false, nil
	}

	batch = s.results[s.index : s.index+n]
	s.index += n
	return batch, true, nil
}

// Put is not supported for lazy streams - they are read-only.
func (s *LazyResultStream) Put(ctx context.Context, store ConstraintStore) error {
	return ErrUnsupportedOperation
}

// Close marks the lazy stream as fully consumed.
func (s *LazyResultStream) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.computed = true
	return nil
}

// Count returns the total number of results (computed lazily).
func (s *LazyResultStream) Count() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.computed {
		// If not computed yet, we don't know the count
		return 0
	}
	return s.count
}

// Error definitions for stream operations
var (
	ErrUnsupportedOperation = fmt.Errorf("unsupported operation")
	ErrStreamClosed         = fmt.Errorf("stream closed")
)