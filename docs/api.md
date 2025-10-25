# GoKanren API Reference

## Core Types

### Term
```go
type Term interface {
    String() string
    Equal(other Term) bool
    IsVar() bool
    Clone() Term
}
```

Base interface for all values in miniKanren. Implementations must be thread-safe.

### Var
```go
type Var struct {
    // Unexported fields for thread safety
}

func Fresh(name string) *Var
```

Logic variables that can be bound to values through unification.

**Example:**
```go
x := Fresh("x")        // Named variable
y := Fresh("")         // Anonymous variable
```

### Atom
```go
type Atom struct {
    // Unexported fields
}

func NewAtom(value interface{}) *Atom
func (a *Atom) Value() interface{}
```

Atomic values representing themselves (numbers, strings, booleans, etc.).

**Example:**
```go
hello := NewAtom("hello")
num := NewAtom(42)
flag := NewAtom(true)
```

### Pair
```go
type Pair struct {
    // Unexported fields
}

func NewPair(car, cdr Term) *Pair
func (p *Pair) Car() Term
func (p *Pair) Cdr() Term
```

Cons cells for building compound structures and lists.

**Example:**
```go
// Build (1 . 2)
pair := NewPair(NewAtom(1), NewAtom(2))

// Build (1 . (2 . nil)) - a list
list := NewPair(NewAtom(1), NewPair(NewAtom(2), NewAtom(nil)))
```

### Substitution
```go
type Substitution struct {
    // Unexported fields
}

func NewSubstitution() *Substitution
func (s *Substitution) Clone() *Substitution
func (s *Substitution) Lookup(v *Var) Term
func (s *Substitution) Bind(v *Var, term Term) *Substitution
func (s *Substitution) Walk(term Term) Term
func (s *Substitution) Size() int
```

Thread-safe mapping from variables to terms. Operations are immutable.

**Example:**
```go
sub := NewSubstitution()
x := Fresh("x")
newSub := sub.Bind(x, NewAtom("hello"))
result := newSub.Lookup(x) // Returns "hello" atom
```

### Stream
```go
type Stream struct {
    // Unexported fields
}

func NewStream() *Stream
func (s *Stream) Take(n int) ([]*Substitution, bool)
func (s *Stream) Put(sub *Substitution)
func (s *Stream) Close()
```

Channel-based sequence of substitutions representing multiple solutions.

### Goal
```go
type Goal func(ctx context.Context, sub *Substitution) *Stream
```

Function that transforms a substitution into a stream of substitutions.

## Core Operations

### Eq (Unification)
```go
func Eq(term1, term2 Term) Goal
```

Creates a goal that unifies two terms.

**Example:**
```go
x := Fresh("x")
goal := Eq(x, NewAtom("hello")) // Binds x to "hello"
```

### Conj (Conjunction/AND)
```go
func Conj(goals ...Goal) Goal
```

Creates a goal that succeeds only if all sub-goals succeed.

**Example:**
```go
x := Fresh("x")
y := Fresh("y")
goal := Conj(
    Eq(x, NewAtom(1)),
    Eq(y, NewAtom(2)),
) // Both x=1 AND y=2 must hold
```

### Disj (Disjunction/OR)
```go
func Disj(goals ...Goal) Goal
func Conde(goals ...Goal) Goal // Alias for Disj
```

Creates a goal that succeeds if any sub-goal succeeds.

**Example:**
```go
x := Fresh("x")
goal := Disj(
    Eq(x, NewAtom(1)),
    Eq(x, NewAtom(2)),
    Eq(x, NewAtom(3)),
) // x can be 1 OR 2 OR 3
```

### Run
```go
func Run(n int, goalFunc func(*Var) Goal) []Term
func RunStar(goalFunc func(*Var) Goal) []Term
func RunWithContext(ctx context.Context, n int, goalFunc func(*Var) Goal) []Term
func RunStarWithContext(ctx context.Context, goalFunc func(*Var) Goal) []Term
func RunWithIsolation(n int, goalFunc func(*Var) Goal) []Term
func RunWithIsolationContext(ctx context.Context, n int, goalFunc func(*Var) Goal) []Term
```

Execute goals and return solutions.

- `Run` / `RunStar`: Standard execution with shared constraint bus
- `RunWithIsolation`: Execution with isolated constraint bus for complete separation
- Context variants: Support timeout and cancellation

**Example:**
```go
// Get up to 5 solutions
results := Run(5, func(q *Var) Goal {
    return Disj(
        Eq(q, NewAtom(1)),
        Eq(q, NewAtom(2)),
        Eq(q, NewAtom(3)),
    )
})
// results: [1, 2, 3]

// Get all solutions (be careful with infinite streams!)
allResults := RunStar(func(q *Var) Goal {
    return Eq(q, NewAtom("hello"))
})
// allResults: ["hello"]

// Isolated execution for constraint separation
isolatedResults := RunWithIsolation(10, func(q *Var) Goal {
    return someGoalWithConstraints(q)
})
```

## Data Construction Functions

### AtomFromValue
```go
func AtomFromValue(value interface{}) *Atom
```

Creates an atom from any Go value. The value is stored directly in the atom.

**Example:**
```go
atom := AtomFromValue(42)
stringAtom := AtomFromValue("hello")
sliceAtom := AtomFromValue([]int{1, 2, 3})
```

### List
```go
func List(terms ...Term) Term
```

Creates a list from terms.

**Example:**
```go
lst := List(NewAtom(1), NewAtom(2), NewAtom(3))
// Creates: (1 . (2 . (3 . nil)))
```

### Appendo
```go
func Appendo(l1, l2, l3 Term) Goal
```

Relational list append. l3 is the result of appending l1 and l2.

**Example:**
```go
// Forward: append([1,2], [3,4]) = ?
results := Run(1, func(q *Var) Goal {
    list12 := List(NewAtom(1), NewAtom(2))
    list34 := List(NewAtom(3), NewAtom(4))
    return Appendo(list12, list34, q)
})
// results: [(1 . (2 . (3 . (4 . nil))))]

// Backward: append(?, [3,4]) = [1,2,3,4]
results = Run(3, func(q *Var) Goal {
    list34 := List(NewAtom(3), NewAtom(4))
    list1234 := List(NewAtom(1), NewAtom(2), NewAtom(3), NewAtom(4))
    return Appendo(q, list34, list1234)
})
// results: [(1 . (2 . nil))]
```

## Constraint Operations

### Neq (Disequality)
```go
func Neq(t1, t2 Term) Goal
```

Creates a constraint that t1 and t2 must not unify.

**Example:**
```go
x := Fresh("x")
goal := Conj(
    Neq(x, NewAtom("forbidden")),
    Eq(x, NewAtom("allowed")),
)
```

### Absento (Absence)
```go
func Absento(absent, term Term) Goal
```

Creates a constraint that absent does not occur anywhere in term.

**Example:**
```go
x := Fresh("x")
list := List(NewAtom(1), x, NewAtom(3))
goal := Conj(
    Absento(NewAtom(2), list), // 2 must not appear in list
    Eq(x, NewAtom(5)),         // So x can be 5 but not 2
)
```

### Type Constraints
```go
func Symbolo(term Term) Goal
func Numbero(term Term) Goal
```

Type constraints for symbols (strings) and numbers.

**Example:**
```go
x := Fresh("x")
goal := Conj(
    Symbolo(x),                    // x must be a string
    Eq(x, NewAtom("hello")),       // Valid: "hello" is a string
)

y := Fresh("y")
goal2 := Conj(
    Numbero(y),                    // y must be a number
    Eq(y, NewAtom(42)),           // Valid: 42 is a number
)
```

### List Constraints
```go
func Membero(element, list Term) Goal
func Car(pair, car Term) Goal
func Cdr(pair, cdr Term) Goal
func Cons(car, cdr, pair Term) Goal
func Nullo(term Term) Goal
func Pairo(term Term) Goal
```

List structure and membership constraints.

**Example:**
```go
// Membership
x := Fresh("x")
list := List(NewAtom(1), NewAtom(2), NewAtom(3))
memberGoal := Membero(x, list) // x can be 1, 2, or 3

// List deconstruction
head := Fresh("head")
tail := Fresh("tail")
carGoal := Car(list, head)     // head = 1
cdrGoal := Cdr(list, tail)     // tail = (2 . (3 . nil))

// List construction
newPair := Fresh("pair")
consGoal := Cons(NewAtom(0), list, newPair) // pair = (0 . (1 . (2 . (3 . nil))))

// Type checks
emptyList := NewAtom(nil)
nullGoal := Nullo(emptyList)   // succeeds: nil is empty list
pairGoal := Pairo(list)        // succeeds: list is non-empty
```

### Control Flow
```go
func Onceo(goal Goal) Goal
func Conda(clauses ...[]Goal) Goal
func Condu(clauses ...[]Goal) Goal
```

Control flow and cut operations.

**Example:**
```go
// Cut - succeed at most once
x := Fresh("x")
onceGoal := Onceo(Disj(
    Eq(x, NewAtom(1)),
    Eq(x, NewAtom(2)),
    Eq(x, NewAtom(3)),
)) // Only returns first solution: x = 1

// If-then-else with cut (conda)
condaGoal := Conda(
    []Goal{Eq(x, NewAtom(1)), Success},              // if x=1 then succeed
    []Goal{Eq(x, NewAtom(2)), Eq(x, NewAtom(20))},   // else if x=2 then x=20
    []Goal{Success, Failure},                         // else fail
)

// If-then-else with uniqueness (condu)
conduGoal := Condu(
    []Goal{Eq(x, NewAtom(1)), Success},              // if x=1 uniquely then succeed
    []Goal{Success, Failure},                         // else fail
)
```

### Advanced Operations
```go
func Project(vars []Term, goalFunc func([]Term) Goal) Goal
```

Project variables out of the logic context for computation.

**Example:**
```go
x := Fresh("x")
y := Fresh("y")
goal := Conj(
    Eq(x, NewAtom(5)),
    Eq(y, NewAtom(3)),
    Project([]Term{x, y}, func(vals []Term) Goal {
        // vals[0] = 5, vals[1] = 3
        if atom1, ok := vals[0].(*Atom); ok {
            if atom2, ok := vals[1].(*Atom); ok {
                if val1, ok := atom1.Value().(int); ok {
                    if val2, ok := atom2.Value().(int); ok {
                        sum := val1 + val2
                        result := Fresh("result")
                        return Eq(result, NewAtom(sum)) // result = 8
                    }
                }
            }
        }
        return Failure
    }),
)
```

## Parallel Execution

### ParallelConfig
```go
type ParallelConfig struct {
    MaxWorkers         int  // Number of worker goroutines
    MaxQueueSize       int  // Maximum pending tasks
    EnableBackpressure bool // Enable backpressure control
    RateLimit          int  // Operations per second (0 = no limit)
}

func DefaultParallelConfig() *ParallelConfig
```

Configuration for parallel execution.

### ParallelExecutor
```go
type ParallelExecutor struct {
    // Unexported fields
}

func NewParallelExecutor(config *ParallelConfig) *ParallelExecutor
func (pe *ParallelExecutor) Shutdown()
func (pe *ParallelExecutor) ParallelDisj(goals ...Goal) Goal
```

Manages parallel goal execution with worker pools and backpressure.

**Example:**
```go
config := &ParallelConfig{
    MaxWorkers:   4,
    MaxQueueSize: 20,
    EnableBackpressure: true,
}
executor := NewParallelExecutor(config)
defer executor.Shutdown()

// Parallel disjunction
goal := executor.ParallelDisj(
    heavyGoal1,
    heavyGoal2,
    heavyGoal3,
)
```

### Parallel Run Functions
```go
func ParallelRun(n int, goalFunc func(*Var) Goal) []Term
func ParallelRunWithConfig(n int, goalFunc func(*Var) Goal, config *ParallelConfig) []Term
func ParallelRunWithContext(ctx context.Context, n int, goalFunc func(*Var) Goal, config *ParallelConfig) []Term
```

Convenience functions for parallel execution.

**Example:**
```go
// Simple parallel run
results := ParallelRun(10, func(q *Var) Goal {
    return someComplexGoal(q)
})

// With custom config and timeout
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
results = ParallelRunWithContext(ctx, 100, func(q *Var) Goal {
    return someComplexGoal(q)
}, &ParallelConfig{MaxWorkers: 8})
```

### ParallelStream
```go
type ParallelStream struct {
    *Stream
    // Additional parallel capabilities
}

func NewParallelStream(ctx context.Context, executor *ParallelExecutor) *ParallelStream
func (ps *ParallelStream) ParallelMap(fn func(*Substitution) *Substitution) *ParallelStream
func (ps *ParallelStream) ParallelFilter(predicate func(*Substitution) bool) *ParallelStream
func (ps *ParallelStream) Collect() []*Substitution
```

Enhanced stream with parallel processing capabilities.

**Example:**
```go
executor := NewParallelExecutor(nil)
defer executor.Shutdown()

stream := NewParallelStream(ctx, executor)

// Parallel map operation
mapped := stream.ParallelMap(func(sub *Substitution) *Substitution {
    // Transform substitution
    return transformSub(sub)
})

// Parallel filter
filtered := stream.ParallelFilter(func(sub *Substitution) bool {
    return sub.Size() > 0
})

results := filtered.Collect()
```

## Built-in Goals

### Success and Failure
```go
var Success Goal // Always succeeds
var Failure Goal // Always fails
```

Basic goals for control flow.

**Example:**
```go
// Conditional logic
goal := func(condition bool) Goal {
    if condition {
        return Success
    }
    return Failure
}
```

## Error Handling

All functions that can fail return errors explicitly. Use context for timeouts and cancellation:

```go
// Timeout example
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

results := RunWithContext(ctx, 1000, func(q *Var) Goal {
    return someSlowGoal(q)
})

// Check if context was cancelled
if ctx.Err() != nil {
    log.Printf("Execution timed out: %v", ctx.Err())
}
```

## Thread Safety

All public APIs are thread-safe. You can safely:
- Create variables concurrently
- Access substitutions from multiple goroutines
- Run goals in parallel
- Share executors across goroutines

**Example:**
```go
var wg sync.WaitGroup
results := make([][]Term, 10)

// Concurrent execution
for i := 0; i < 10; i++ {
    wg.Add(1)
    go func(index int) {
        defer wg.Done()
        results[index] = Run(5, func(q *Var) Goal {
            return Eq(q, NewAtom(index))
        })
    }(i)
}

wg.Wait()
```

## Best Practices

### Variable Naming
Use descriptive names for debugging:
```go
person := Fresh("person")
age := Fresh("age")
```

### Resource Management
Always clean up parallel executors:
```go
executor := NewParallelExecutor(config)
defer executor.Shutdown() // Important!
```

### Context Usage
Use contexts for cancellation and timeouts:
```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

// Cancel on signal
go func() {
    <-signalChan
    cancel()
}()

results := RunWithContext(ctx, 1000, goalFunc)
```

### Performance
- Use parallel execution for CPU-intensive goals
- Configure worker pools based on your workload
- Enable backpressure for large search spaces
- Profile your specific use case

### Testing
Write tests that verify both correctness and thread safety:
```go
func TestConcurrentGoals(t *testing.T) {
    const numGoroutines = 100
    var wg sync.WaitGroup
    
    for i := 0; i < numGoroutines; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            // Test concurrent execution
            results := Run(1, func(q *Var) Goal {
                return Eq(q, NewAtom("test"))
            })
            // Verify results
            if len(results) != 1 {
                t.Error("Expected 1 result")
            }
        }()
    }
    
    wg.Wait()
}
```