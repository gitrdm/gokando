# GoKanren - Thread-Safe Parallel miniKanren in Go

GoKanren is a production-quality implementation of miniKanren in Go, designed with thread-safety and parallel execution as first-class concerns.

## Features

- **Thread-Safe**: All operations are safe for concurrent use
- **Parallel Execution**: Goals can be evaluated in parallel using goroutines
- **Type-Safe**: Leverages Go's type system for safe relational programming
- **Well-Documented**: Comprehensive documentation with examples
- **Production Ready**: Extensive testing and benchmarking

## Architecture

```
pkg/minikanren/     # Core miniKanren implementation
internal/parallel/  # Parallel execution internals
cmd/example/       # Example applications
docs/             # Additional documentation
```

## Quick Start

```go
import "gokando/pkg/minikanren"

// Basic unification example
run := minikanren.Run(1, func(q minikanren.Var) minikanren.Goal {
    return minikanren.Eq(q, minikanren.Atom("hello"))
})

fmt.Println(run) // [hello]
```

## Documentation

Run `go doc` to view inline documentation, or see the `docs/` directory for detailed guides.

## Testing

```bash
go test ./...
go test -bench=. ./...
```

## License

MIT License - see LICENSE file for details.