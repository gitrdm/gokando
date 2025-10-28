# parallel API

Complete API documentation for the parallel package.

**Import Path:** `github.com/gitrdm/gokando/internal/parallel`

## Package Documentation

Package parallel provides advanced parallel execution capabilities
for miniKanren goals. This package contains internal utilities
for managing concurrent goal evaluation with proper resource
management and backpressure control.


## Variables

### ErrLimiterShutdown

ErrLimiterShutdown is returned when trying to wait on a shutdown limiter.


```go
&{<nil> [ErrLimiterShutdown] <nil> [0xc000386bc0] <nil>}
```

### ErrPoolShutdown

ErrPoolShutdown is returned when trying to submit tasks to a shutdown pool.


```go
&{<nil> [ErrPoolShutdown] <nil> [0xc000375700] <nil>}
```

## Types

### BackpressureController
BackpressureController manages backpressure in the goal evaluation pipeline to prevent memory exhaustion during large or infinite search spaces.

#### Example Usage

```go
// Create a new BackpressureController
backpressurecontroller := BackpressureController{
    maxQueueSize: 42,
    currentLoad: 42,
    highWaterMark: 42,
    lowWaterMark: 42,
    paused: true,
    pauseChan: /* value */,
    resumeChan: /* value */,
    mu: /* value */,
}
```

#### Type Definition

```go
type BackpressureController struct {
    maxQueueSize int
    currentLoad int64
    highWaterMark int
    lowWaterMark int
    paused bool
    pauseChan chan *ast.StructType
    resumeChan chan *ast.StructType
    mu sync.RWMutex
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| maxQueueSize | `int` |  |
| currentLoad | `int64` |  |
| highWaterMark | `int` |  |
| lowWaterMark | `int` |  |
| paused | `bool` |  |
| pauseChan | `chan *ast.StructType` |  |
| resumeChan | `chan *ast.StructType` |  |
| mu | `sync.RWMutex` |  |

### Constructor Functions

### NewBackpressureController

NewBackpressureController creates a new backpressure controller.

```go
func NewBackpressureController(maxQueueSize int) *BackpressureController
```

**Parameters:**
- `maxQueueSize` (int)

**Returns:**
- *BackpressureController

## Methods

### AddLoad

AddLoad adds to the current load and checks if backpressure should be applied.

```go
func (*BackpressureController) AddLoad(amount int)
```

**Parameters:**
- `amount` (int)

**Returns:**
  None

### CheckBackpressure

CheckBackpressure checks if backpressure should be applied. Returns true if the caller should pause, false otherwise.

```go
func (*BackpressureController) CheckBackpressure(ctx context.Context) error
```

**Parameters:**
- `ctx` (context.Context)

**Returns:**
- error

### CurrentLoad

CurrentLoad returns the current load level.

```go
func (*BackpressureController) CurrentLoad() int64
```

**Parameters:**
  None

**Returns:**
- int64

### RemoveLoad

RemoveLoad removes from the current load and checks if backpressure should be released.

```go
func (*BackpressureController) RemoveLoad(amount int)
```

**Parameters:**
- `amount` (int)

**Returns:**
  None

### LoadBalancer
LoadBalancer distributes work across multiple workers using a round-robin strategy to ensure fair distribution.

#### Example Usage

```go
// Create a new LoadBalancer
loadbalancer := LoadBalancer{
    workers: [],
    current: 42,
    mu: /* value */,
}
```

#### Type Definition

```go
type LoadBalancer struct {
    workers []Worker
    current int64
    mu sync.Mutex
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| workers | `[]Worker` |  |
| current | `int64` |  |
| mu | `sync.Mutex` |  |

### Constructor Functions

### NewLoadBalancer

NewLoadBalancer creates a new load balancer with the given workers.

```go
func NewLoadBalancer(workers []Worker) *LoadBalancer
```

**Parameters:**
- `workers` ([]Worker)

**Returns:**
- *LoadBalancer

## Methods

### Submit

Submit submits a task to the next available worker using round-robin.

```go
func (*LoadBalancer) Submit(ctx context.Context, task interface{}) error
```

**Parameters:**
- `ctx` (context.Context)
- `task` (interface{})

**Returns:**
- error

### RateLimiter
RateLimiter provides rate limiting functionality to prevent overwhelming downstream consumers during intensive goal evaluation.

#### Example Usage

```go
// Create a new RateLimiter
ratelimiter := RateLimiter{
    ticker: &/* value */{},
    tokens: /* value */,
    shutdown: /* value */,
    once: /* value */,
}
```

#### Type Definition

```go
type RateLimiter struct {
    ticker *time.Ticker
    tokens chan *ast.StructType
    shutdown chan *ast.StructType
    once sync.Once
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| ticker | `*time.Ticker` |  |
| tokens | `chan *ast.StructType` |  |
| shutdown | `chan *ast.StructType` |  |
| once | `sync.Once` |  |

### Constructor Functions

### NewRateLimiter

NewRateLimiter creates a new rate limiter that allows up to tokensPerSecond operations per second.

```go
func NewRateLimiter(tokensPerSecond int) *RateLimiter
```

**Parameters:**
- `tokensPerSecond` (int)

**Returns:**
- *RateLimiter

## Methods

### Close

Close shuts down the rate limiter and releases all resources.

```go
func (*RateLimiter) Close()
```

**Parameters:**
  None

**Returns:**
  None

### Wait

Wait blocks until a token is available or the context is cancelled.

```go
func (*RateLimiter) Wait(ctx context.Context) error
```

**Parameters:**
- `ctx` (context.Context)

**Returns:**
- error

### refillTokens

refillTokens continuously refills the token bucket at the specified rate.

```go
func (*RateLimiter) refillTokens()
```

**Parameters:**
  None

**Returns:**
  None

### StreamMerger
StreamMerger combines multiple streams into a single output stream while maintaining fairness and preventing any single stream from dominating the output.

#### Example Usage

```go
// Create a new StreamMerger
streammerger := StreamMerger{
    outputChan: /* value */,
    doneChan: /* value */,
    wg: /* value */,
    mu: /* value */,
    closed: true,
}
```

#### Type Definition

```go
type StreamMerger struct {
    outputChan chan interface{}
    doneChan chan *ast.StructType
    wg sync.WaitGroup
    mu sync.Mutex
    closed bool
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| outputChan | `chan interface{}` |  |
| doneChan | `chan *ast.StructType` |  |
| wg | `sync.WaitGroup` |  |
| mu | `sync.Mutex` |  |
| closed | `bool` |  |

### Constructor Functions

### NewStreamMerger

NewStreamMerger creates a new stream merger.

```go
func NewStreamMerger() *StreamMerger
```

**Parameters:**
  None

**Returns:**
- *StreamMerger

## Methods

### AddStream

AddStream adds an input stream to the merger. The merger will read from this stream and forward items to the output.

```go
func (*StreamMerger) AddStream(inputChan <-chan interface{})
```

**Parameters:**
- `inputChan` (<-chan interface{})

**Returns:**
  None

### Close

Close closes the merger and all associated resources. After calling Close, no more items will be output.

```go
func (*RateLimiter) Close()
```

**Parameters:**
  None

**Returns:**
  None

### Output

Output returns the output channel for reading merged items.

```go
func (*StreamMerger) Output() <-chan interface{}
```

**Parameters:**
  None

**Returns:**
- <-chan interface{}

### Worker
Worker represents a worker that can process tasks.

#### Example Usage

```go
// Example implementation of Worker
type MyWorker struct {
    // Add your fields here
}

func (m MyWorker) Process(param1 context.Context, param2 interface{}) error {
    // Implement your logic here
    return
}

func (m MyWorker) ID() string {
    // Implement your logic here
    return
}


```

#### Type Definition

```go
type Worker interface {
    Process(ctx context.Context, task interface{}) error
    ID() string
}
```

## Methods

| Method | Description |
| ------ | ----------- |

### WorkerPool
WorkerPool manages a pool of goroutines for parallel goal evaluation. It provides controlled concurrency with backpressure handling to prevent resource exhaustion during large search spaces.

#### Example Usage

```go
// Create a new WorkerPool
workerpool := WorkerPool{
    maxWorkers: 42,
    taskChan: /* value */,
    workerWg: /* value */,
    shutdownChan: /* value */,
    once: /* value */,
}
```

#### Type Definition

```go
type WorkerPool struct {
    maxWorkers int
    taskChan chan func()
    workerWg sync.WaitGroup
    shutdownChan chan *ast.StructType
    once sync.Once
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| maxWorkers | `int` |  |
| taskChan | `chan func()` |  |
| workerWg | `sync.WaitGroup` |  |
| shutdownChan | `chan *ast.StructType` |  |
| once | `sync.Once` |  |

### Constructor Functions

### NewWorkerPool

NewWorkerPool creates a new worker pool with the specified number of workers. If maxWorkers is 0 or negative, it defaults to the number of CPU cores.

```go
func NewWorkerPool(maxWorkers int) *WorkerPool
```

**Parameters:**
- `maxWorkers` (int)

**Returns:**
- *WorkerPool

## Methods

### Shutdown

Shutdown gracefully shuts down the worker pool, waiting for all currently executing tasks to complete.

```go
func (*WorkerPool) Shutdown()
```

**Parameters:**
  None

**Returns:**
  None

### Submit

Submit submits a task to the worker pool for execution. If the pool is full, this call will block until a worker becomes available.

```go
func (*LoadBalancer) Submit(ctx context.Context, task interface{}) error
```

**Parameters:**
- `ctx` (context.Context)
- `task` (interface{})

**Returns:**
- error

### worker

worker is the main worker loop that processes tasks from the channel.

```go
func (*WorkerPool) worker()
```

**Parameters:**
  None

**Returns:**
  None

## External Links

- [Package Overview](../packages/parallel.md)
- [pkg.go.dev Documentation](https://pkg.go.dev/github.com/gitrdm/gokando/internal/parallel)
- [Source Code](https://github.com/gitrdm/gokando/tree/master/internal/parallel)
