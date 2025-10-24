# GoKanren - Thread-Safe Parallel miniKanren in Go

GoKanren is a production-quality implementation of miniKanren in Go, designed with thread-safety and parallel execution as first-class concerns. This implementation provides a complete set of miniKanren operators with high-performance concurrent execution.

## Features

### Core miniKanren
- **Complete Operator Set**: All standard miniKanren operators including constraints
- **Thread-Safe**: All operations are safe for concurrent use across goroutines
- **Parallel Execution**: Goals can be evaluated in parallel using configurable worker pools
- **Type-Safe**: Leverages Go's type system for safe relational programming
- **Stream-Based**: Lazy evaluation with channel-based streaming for scalability

### Advanced Features  
- **Constraint System**: Full constraint support (disequality, absence, type constraints)
- **Parallel Goal Evaluation**: Concurrent disjunction and stream processing
- **Backpressure Control**: Automatic rate limiting and resource management
- **Context Support**: Timeout and cancellation support for all operations
- **Production Ready**: Extensive testing, benchmarking, and documentation

## Requirements

### System Requirements
- **Go Version**: Go 1.18+ (uses generics and modern concurrency patterns)
- **Memory**: Sufficient RAM for concurrent goal evaluation (scales with parallelism level)
- **OS**: Any platform supporting Go (Linux, macOS, Windows)

### Development Requirements
```bash
# Install dependencies
go mod tidy

# Run tests
go test ./...

# Run benchmarks  
go test -bench=. ./...
```

## Architecture

```
pkg/minikanren/     # Core miniKanren implementation
├── core.go         # Basic types (Term, Var, Atom, Pair, Substitution, Stream, Goal)
├── primitives.go   # Core operations (Fresh, Eq, Conj, Disj, Run, Appendo)
├── parallel.go     # Parallel execution framework
├── constraints.go  # Extended constraint operators
└── *_test.go      # Comprehensive test suites

internal/parallel/  # Parallel execution internals
├── pool.go        # Worker pools and backpressure control

cmd/example/       # Example applications and demos
docs/             # Architecture and API documentation
```

## Complete Operator Reference

### Core Operations
- `Fresh(name)` - Create fresh logic variables
- `Eq(t1, t2)` - Unification constraint (t1 == t2)
- `Conj(goals...)` - Logical AND (all goals must succeed)
- `Disj(goals...)` - Logical OR (any goal can succeed)
- `Run(n, goalFunc)` - Execute goal and return up to n solutions
- `RunStar(goalFunc)` - Execute goal and return all solutions

### Constraint Operators
- `Neq(t1, t2)` - Disequality constraint (t1 != t2)
- `Absento(absent, term)` - Absence constraint (absent not in term)
- `Symbolo(term)` - Type constraint (term must be symbol/string)
- `Numbero(term)` - Type constraint (term must be number)
- `Membero(elem, list)` - List membership relation
- `Onceo(goal)` - Cut operator (succeed at most once)

### Committed Choice
- `Conda(clauses...)` - If-then-else with cut
- `Condu(clauses...)` - If-then-else with uniqueness requirement

### Advanced Operations
- `Project(vars, goalFunc)` - Variable projection and computation
- `Car(list, head)` - Extract list head
- `Cdr(list, tail)` - Extract list tail  
- `Cons(head, tail, list)` - Construct list
- `Nullo(term)` - Empty list check
- `Pairo(term)` - Non-empty list check

### Parallel Execution
- `ParallelRun(n, goalFunc)` - Parallel goal execution
- `ParallelDisj(goals...)` - Concurrent disjunction
- `ParallelStream` - Concurrent stream processing

### List Relations
- `Appendo(l1, l2, l3)` - List append relation (l1 + l2 = l3)
- `List(elements...)` - Construct proper lists
- `Nil` - Empty list constant

## Usage Examples

### Basic Unification
```go
import "gokando/pkg/minikanren"

// Simple value binding
results := minikanren.Run(1, func(q *minikanren.Var) minikanren.Goal {
    return minikanren.Eq(q, minikanren.NewAtom("hello"))
})
fmt.Println(results) // [hello]
```

### List Operations
```go
// List append relation
results := minikanren.Run(5, func(q *minikanren.Var) minikanren.Goal {
    return minikanren.Appendo(
        minikanren.List(minikanren.NewAtom(1), minikanren.NewAtom(2)),
        minikanren.List(minikanren.NewAtom(3)),
        q,
    )
})
fmt.Println(results) // [(1 2 3)]
```

### Constraints
```go
// Find symbols that aren't "forbidden"
results := minikanren.Run(1, func(q *minikanren.Var) minikanren.Goal {
    return minikanren.Conj(
        minikanren.Eq(q, minikanren.NewAtom("allowed")),      // Bind first
        minikanren.Symbolo(q),                                 // Then check type
        minikanren.Neq(q, minikanren.NewAtom("forbidden")),   // Then check inequality
    )
})
```

### Parallel Execution
```go
// Concurrent goal evaluation
results := minikanren.ParallelRun(10, func(q *minikanren.Var) minikanren.Goal {
    return minikanren.ParallelDisj(
        minikanren.Eq(q, minikanren.NewAtom(1)),
        minikanren.Eq(q, minikanren.NewAtom(2)),
        minikanren.Eq(q, minikanren.NewAtom(3)),
    )
})
```

## Important Usage Notes

### Order-Independent Constraints

✨ **Feature**: Constraints in this implementation are **order-independent**. The system provides maximum flexibility:

1. **Constraints work before or after unification**:
   ```go
   // ✅ Both orders work identically
   
   // Constraint before unification
   minikanren.Conj(
       minikanren.Numbero(q),                       // Constraint added to store
       minikanren.Eq(q, minikanren.NewAtom(42)),    // Unification checks constraints
   )
   
   // Unification before constraint  
   minikanren.Conj(
       minikanren.Eq(q, minikanren.NewAtom(42)),    // Bind first
       minikanren.Numbero(q),                       // Check constraint after
   )
   ```

2. **How this works**: Our hybrid constraint system uses LocalConstraintStore + GlobalConstraintBus to automatically coordinate constraint checking regardless of goal ordering. This provides both flexibility and performance.

### Parallel Execution Considerations

- **Overhead**: Parallel execution has coordination overhead. For simple goals, sequential execution may be faster
- **Resource Usage**: Parallel execution uses more memory and CPU cores
- **Cancellation**: All parallel operations support context cancellation for clean shutdown

### Performance Characteristics

- **Sequential**: Optimal for simple queries and small solution spaces
- **Parallel**: Best for complex goals with independent choice points
- **Memory**: Stream-based processing supports large solution spaces efficiently
- **Concurrency**: Thread-safe for use across multiple goroutines

## Testing

```bash
# Run all tests
go test ./...

# Run with verbose output
go test ./... -v

# Run specific test suites
go test ./pkg/minikanren/ -run TestConstraints -v

# Run benchmarks
go test -bench=. ./...

# Run with race detection
go test -race ./...
```

## Performance Tuning

### Parallel Configuration
```go
config := &minikanren.ParallelConfig{
    MaxWorkers:      runtime.NumCPU(),
    BufferSize:      1000,
    BackpressureEnabled: true,
}

results := minikanren.ParallelRunWithConfig(10, goalFunc, config)
```

### Memory Management
- Use `RunStar` carefully - it returns ALL solutions
- Consider `Run(n, ...)` with reasonable limits for large solution spaces  
- Parallel execution automatically manages worker pools and backpressure

## API Stability

This implementation provides:
- **Stable Core API**: Core miniKanren operations follow standard semantics
- **Production Quality**: Comprehensive testing and thread-safety guarantees
- **Performance Focus**: Optimized for concurrent Go applications
- **Complete Feature Set**: Full miniKanren operator coverage

## Comparison with Other miniKanren Implementations

| Feature | GoKando | core.logic (Clojure) | miniKanren (Scheme) |
|---------|----------|----------------------|---------------------|
| Thread Safety | ✅ Built-in | ❌ Requires care | ❌ Single-threaded |
| Parallel Execution | ✅ Native | ❌ Manual | ❌ No |
| Constraint Ordering | ✅ Order-independent | ✅ Order-independent | ✅ Order-independent |
| Performance | ✅ High (compiled) | ✅ Good (JVM) | ✅ Good (compiled) |
| Type Safety | ✅ Static types | ❌ Dynamic | ❌ Dynamic |

## Contributing

1. Ensure all tests pass: `go test ./...`
2. Run benchmarks to check performance: `go test -bench=. ./...`
3. Follow Go conventions and document new APIs
4. Add tests for new functionality

## License

MIT License - see LICENSE file for details.