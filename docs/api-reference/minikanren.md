# minikanren API

Complete API documentation for the minikanren package.

**Import Path:** `github.com/gitrdm/gokando/pkg/minikanren`

## Package Documentation

Package minikanren provides constraint system infrastructure for order-independent
constraint logic programming. This file defines the core interfaces and types
for managing constraints in a hybrid local/global architecture.

The constraint system uses a two-tier approach:
  - Local constraints: managed within individual goal contexts for fast checking
  - Global constraints: coordinated across contexts when constraints span multiple stores

This design provides order-independent constraint semantics while maintaining
high performance for the common case of locally-scoped constraints.

Package minikanren provides concrete implementations of constraints
for the hybrid constraint system. These constraints implement the
Constraint interface and provide the core constraint logic for
disequality, absence, type checking, and other relational operations.

Each constraint implementation follows the same pattern:
  - Efficient local checking when all variables are bound
  - Graceful handling of unbound variables (returns ConstraintPending)
  - Thread-safe operations for concurrent constraint checking
  - Proper variable dependency tracking for optimization

The constraint implementations are designed to be:
  - Fast: Optimized for the common case of local constraint checking
  - Safe: Thread-safe and defensive against malformed input
  - Debuggable: Comprehensive error messages and string representations

Package minikanren provides a thread-safe, parallel implementation of miniKanren
in Go. This implementation follows the core principles of relational programming
while leveraging Go's concurrency primitives for parallel execution.

miniKanren is a domain-specific language for constraint logic programming.
It provides a minimal set of operators for building relational programs:
  - Unification (==): Constrains two terms to be equal
  - Fresh variables: Introduces new logic variables
  - Disjunction (conde): Represents choice points
  - Conjunction: Combines goals that must all succeed
  - Run: Executes a goal and returns solutions

This implementation is designed for production use with:
  - Thread-safe operations using sync package primitives
  - Parallel goal evaluation using goroutines and channels
  - Type-safe interfaces leveraging Go's type system
  - Comprehensive error handling and resource management

Package minikanren provides the LocalConstraintStore implementation for
managing constraints and variable bindings within individual goal contexts.

The LocalConstraintStore is the core component of the hybrid constraint system,
providing fast local constraint checking while coordinating with the global
constraint bus when necessary for cross-store constraints.

Key design principles:
  - Fast path: Local constraint checking without coordination overhead
  - Slow path: Global coordination only when cross-store constraints are involved
  - Thread-safe: Safe for concurrent access and parallel goal evaluation
  - Efficient cloning: Optimized for parallel execution where stores are frequently copied

Package minikanren provides a thread-safe parallel implementation of miniKanren in Go.

Version: 0.9.1

This package offers a complete set of miniKanren operators with high-performance
concurrent execution capabilities, designed for production use.


## Constants

### Version

Version represents the current version of the GoKando miniKanren implementation.


```go
&{<nil> [Version] <nil> [0xc00040d700] <nil>}
```

## Variables

### Nil

Nil represents the empty list


```go
&{<nil> [Nil] <nil> [0xc000381240] <nil>}
```

## Types

### AbsenceConstraint
AbsenceConstraint implements the absence constraint (absento). It ensures that a specific term does not occur anywhere within another term's structure, providing structural constraint checking. This constraint performs recursive structural inspection to detect the presence of the forbidden term at any level of nesting.

#### Example Usage

```go
// Create a new AbsenceConstraint
absenceconstraint := AbsenceConstraint{
    id: "example",
    absent: Term{},
    container: Term{},
    isLocal: true,
}
```

#### Type Definition

```go
type AbsenceConstraint struct {
    id string
    absent Term
    container Term
    isLocal bool
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| id | `string` | id uniquely identifies this constraint instance |
| absent | `Term` | absent is the term that must not occur |
| container | `Term` | container is the term that must not contain the absent term |
| isLocal | `bool` | isLocal indicates whether this constraint can be checked locally |

### Constructor Functions

### NewAbsenceConstraint

NewAbsenceConstraint creates a new absence constraint.

```go
func NewAbsenceConstraint(absent, container Term) *AbsenceConstraint
```

**Parameters:**
- `absent` (Term)
- `container` (Term)

**Returns:**
- *AbsenceConstraint

## Methods

### Check

Check evaluates the absence constraint against current bindings. Returns ConstraintViolated if the absent term is found in the container, ConstraintPending if variables are unbound, or ConstraintSatisfied otherwise. Implements the Constraint interface.

```go
func (*MembershipConstraint) Check(bindings map[int64]Term) ConstraintResult
```

**Parameters:**
- `bindings` (map[int64]Term)

**Returns:**
- ConstraintResult

### Clone

Clone creates a deep copy of the constraint for parallel execution. Implements the Constraint interface.

```go
func (*MembershipConstraint) Clone() Constraint
```

**Parameters:**
  None

**Returns:**
- Constraint

### ID

ID returns the unique identifier for this constraint instance. Implements the Constraint interface.

```go
func (*MembershipConstraint) ID() string
```

**Parameters:**
  None

**Returns:**
- string

### IsLocal

IsLocal returns true if this constraint can be evaluated locally. Implements the Constraint interface.

```go
func (*MembershipConstraint) IsLocal() bool
```

**Parameters:**
  None

**Returns:**
- bool

### String

String returns a human-readable representation of the constraint. Implements the Constraint interface.

```go
func (*MembershipConstraint) String() string
```

**Parameters:**
  None

**Returns:**
- string

### Variables

Variables returns the logic variables this constraint depends on. Implements the Constraint interface.

```go
func (*MembershipConstraint) Variables() []*Var
```

**Parameters:**
  None

**Returns:**
- []*Var

### Atom
Atom represents an atomic value (symbol, number, string, etc.). Atoms are immutable and represent themselves.

#### Example Usage

```go
// Create a new Atom
atom := Atom{
    value: /* value */,
}
```

#### Type Definition

```go
type Atom struct {
    value interface{}
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| value | `interface{}` | The underlying Go value |

### Constructor Functions

### AtomFromValue

AtomFromValue creates a new atomic term from any Go value. This is a convenience function that's equivalent to NewAtom.

```go
func AtomFromValue(value interface{}) *Atom
```

**Parameters:**
- `value` (interface{})

**Returns:**
- *Atom

### NewAtom

NewAtom creates a new atom from any Go value.

```go
func NewAtom(value interface{}) *Atom
```

**Parameters:**
- `value` (interface{})

**Returns:**
- *Atom

## Methods

### Clone

Clone creates a copy of the atom.

```go
func (*Substitution) Clone() *Substitution
```

**Parameters:**
  None

**Returns:**
- *Substitution

### Equal

Equal checks if two atoms have the same value.

```go
func (*Pair) Equal(other Term) bool
```

**Parameters:**
- `other` (Term)

**Returns:**
- bool

### IsVar

IsVar always returns false for atoms.

```go
func (*Pair) IsVar() bool
```

**Parameters:**
  None

**Returns:**
- bool

### String

String returns a string representation of the atom.

```go
func (ConstraintEventType) String() string
```

**Parameters:**
  None

**Returns:**
- string

### Value

Value returns the underlying Go value.

```go
func (*Atom) Value() interface{}
```

**Parameters:**
  None

**Returns:**
- interface{}

### Constraint
Constraint represents a logical constraint that can be checked against variable bindings. Constraints are the core abstraction that enables order-independent constraint logic programming. Constraints must be thread-safe as they may be checked concurrently during parallel goal evaluation.

#### Example Usage

```go
// Example implementation of Constraint
type MyConstraint struct {
    // Add your fields here
}

func (m MyConstraint) ID() string {
    // Implement your logic here
    return
}

func (m MyConstraint) IsLocal() bool {
    // Implement your logic here
    return
}

func (m MyConstraint) Variables() []*Var {
    // Implement your logic here
    return
}

func (m MyConstraint) Check(param1 map[int64]Term) ConstraintResult {
    // Implement your logic here
    return
}

func (m MyConstraint) String() string {
    // Implement your logic here
    return
}

func (m MyConstraint) Clone() Constraint {
    // Implement your logic here
    return
}


```

#### Type Definition

```go
type Constraint interface {
    ID() string
    IsLocal() bool
    Variables() []*Var
    Check(bindings map[int64]Term) ConstraintResult
    String() string
    Clone() Constraint
}
```

## Methods

| Method | Description |
| ------ | ----------- |

### ConstraintEvent
ConstraintEvent represents a notification about constraint-related activities. Used for coordinating between local stores and the global constraint bus.

#### Example Usage

```go
// Create a new ConstraintEvent
constraintevent := ConstraintEvent{
    Type: ConstraintEventType{},
    StoreID: "example",
    VarID: 42,
    Term: Term{},
    Constraint: Constraint{},
    Timestamp: 42,
}
```

#### Type Definition

```go
type ConstraintEvent struct {
    Type ConstraintEventType
    StoreID string
    VarID int64
    Term Term
    Constraint Constraint
    Timestamp int64
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| Type | `ConstraintEventType` | Type indicates the kind of event (constraint added, variable bound, etc.) |
| StoreID | `string` | StoreID identifies which local constraint store generated this event |
| VarID | `int64` | VarID is the variable ID involved in the event (for binding events) |
| Term | `Term` | Term is the term being bound to the variable (for binding events) |
| Constraint | `Constraint` | Constraint is the constraint involved in the event (for constraint events) |
| Timestamp | `int64` | Timestamp helps with debugging and event ordering |

### ConstraintEventType
ConstraintEventType categorizes different kinds of constraint events for efficient processing by the global constraint bus.

#### Example Usage

```go
// Example usage of ConstraintEventType
var value ConstraintEventType
// Initialize with appropriate value
```

#### Type Definition

```go
type ConstraintEventType int
```

## Methods

### String

String returns a human-readable representation of the constraint event type.

```go
func (ConstraintEventType) String() string
```

**Parameters:**
  None

**Returns:**
- string

### ConstraintResult
ConstraintResult represents the outcome of evaluating a constraint. Constraints can be satisfied (no violation), violated (goal should fail), or pending (waiting for more variable bindings).

#### Example Usage

```go
// Example usage of ConstraintResult
var value ConstraintResult
// Initialize with appropriate value
```

#### Type Definition

```go
type ConstraintResult int
```

## Methods

### String

String returns a human-readable representation of the constraint result.

```go
func (*MembershipConstraint) String() string
```

**Parameters:**
  None

**Returns:**
- string

### ConstraintStore
ConstraintStore represents a collection of constraints and variable bindings. This interface abstracts over both local and global constraint storage.

#### Example Usage

```go
// Example implementation of ConstraintStore
type MyConstraintStore struct {
    // Add your fields here
}

func (m MyConstraintStore) AddConstraint(param1 Constraint) error {
    // Implement your logic here
    return
}

func (m MyConstraintStore) AddBinding(param1 int64, param2 Term) error {
    // Implement your logic here
    return
}

func (m MyConstraintStore) GetBinding(param1 int64) Term {
    // Implement your logic here
    return
}

func (m MyConstraintStore) GetSubstitution() *Substitution {
    // Implement your logic here
    return
}

func (m MyConstraintStore) GetConstraints() []Constraint {
    // Implement your logic here
    return
}

func (m MyConstraintStore) Clone() ConstraintStore {
    // Implement your logic here
    return
}

func (m MyConstraintStore) String() string {
    // Implement your logic here
    return
}


```

#### Type Definition

```go
type ConstraintStore interface {
    AddConstraint(constraint Constraint) error
    AddBinding(varID int64, term Term) error
    GetBinding(varID int64) Term
    GetSubstitution() *Substitution
    GetConstraints() []Constraint
    Clone() ConstraintStore
    String() string
}
```

## Methods

| Method | Description |
| ------ | ----------- |

### Constructor Functions

### unifyWithConstraints

unifyWithConstraints performs unification using the constraint store system. Returns a new constraint store if unification succeeds, and a boolean indicating success. This replaces the old unify function to work with the order-independent constraint system.

```go
func unifyWithConstraints(term1, term2 Term, store ConstraintStore) (ConstraintStore, bool)
```

**Parameters:**
- `term1` (Term)
- `term2` (Term)
- `store` (ConstraintStore)

**Returns:**
- ConstraintStore
- bool

### ConstraintViolationError
ConstraintViolationError represents an error caused by constraint violations. It provides detailed information about which constraint was violated and why.

#### Example Usage

```go
// Create a new ConstraintViolationError
constraintviolationerror := ConstraintViolationError{
    Constraint: Constraint{},
    Bindings: map[],
    Message: "example",
}
```

#### Type Definition

```go
type ConstraintViolationError struct {
    Constraint Constraint
    Bindings map[int64]Term
    Message string
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| Constraint | `Constraint` |  |
| Bindings | `map[int64]Term` |  |
| Message | `string` |  |

### Constructor Functions

### NewConstraintViolationError

NewConstraintViolationError creates a new constraint violation error.

```go
func NewConstraintViolationError(constraint Constraint, bindings map[int64]Term, message string) *ConstraintViolationError
```

**Parameters:**
- `constraint` (Constraint)
- `bindings` (map[int64]Term)
- `message` (string)

**Returns:**
- *ConstraintViolationError

## Methods

### Error

Error returns a detailed error message about the constraint violation.

```go
func (*ConstraintViolationError) Error() string
```

**Parameters:**
  None

**Returns:**
- string

### DisequalityConstraint
DisequalityConstraint implements the disequality constraint (≠). It ensures that two terms are not equal, providing order-independent constraint semantics for the Neq operation. The constraint tracks two terms and checks that they never become equal through unification. If both terms are variables, the constraint remains pending until at least one is bound to a concrete value.

#### Example Usage

```go
// Create a new DisequalityConstraint
disequalityconstraint := DisequalityConstraint{
    id: "example",
    term1: Term{},
    isLocal: true,
}
```

#### Type Definition

```go
type DisequalityConstraint struct {
    id string
    term1 Term
    term2 Term
    isLocal bool
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| id | `string` | id uniquely identifies this constraint instance |
| term1 | `Term` | term1 and term2 are the terms that must not be equal |
| term2 | `Term` | term1 and term2 are the terms that must not be equal |
| isLocal | `bool` | isLocal indicates whether this constraint can be checked locally |

### Constructor Functions

### NewDisequalityConstraint

NewDisequalityConstraint creates a new disequality constraint. The constraint is considered local if both terms are in the same constraint store context, enabling fast local checking.

```go
func NewDisequalityConstraint(term1, term2 Term) *DisequalityConstraint
```

**Parameters:**
- `term1` (Term)
- `term2` (Term)

**Returns:**
- *DisequalityConstraint

## Methods

### Check

Check evaluates the disequality constraint against current variable bindings. Returns ConstraintViolated if the terms are equal, ConstraintPending if variables are unbound, or ConstraintSatisfied if terms are provably unequal. Implements the Constraint interface.

```go
func (*MembershipConstraint) Check(bindings map[int64]Term) ConstraintResult
```

**Parameters:**
- `bindings` (map[int64]Term)

**Returns:**
- ConstraintResult

### Clone

Clone creates a deep copy of the constraint for parallel execution. Implements the Constraint interface.

```go
func (*MembershipConstraint) Clone() Constraint
```

**Parameters:**
  None

**Returns:**
- Constraint

### ID

ID returns the unique identifier for this constraint instance. Implements the Constraint interface.

```go
func (*MembershipConstraint) ID() string
```

**Parameters:**
  None

**Returns:**
- string

### IsLocal

IsLocal returns true if this constraint can be evaluated purely within a local constraint store. Implements the Constraint interface.

```go
func (*MembershipConstraint) IsLocal() bool
```

**Parameters:**
  None

**Returns:**
- bool

### String

String returns a human-readable representation of the constraint. Implements the Constraint interface.

```go
func (*MembershipConstraint) String() string
```

**Parameters:**
  None

**Returns:**
- string

### Variables

Variables returns the logic variables that this constraint depends on. Used to determine when the constraint needs to be re-evaluated. Implements the Constraint interface.

```go
func (*MembershipConstraint) Variables() []*Var
```

**Parameters:**
  None

**Returns:**
- []*Var

### GlobalConstraintBus
GlobalConstraintBus coordinates constraint checking across multiple local constraint stores. It handles cross-store constraints and provides a coordination point for complex constraint interactions. The bus is designed to minimize coordination overhead - most constraints should be local and not require global coordination.

#### Example Usage

```go
// Create a new GlobalConstraintBus
globalconstraintbus := GlobalConstraintBus{
    crossStoreConstraints: map[],
    storeRegistry: map[],
    events: /* value */,
    eventCounter: 42,
    mu: /* value */,
    shutdown: true,
    shutdownCh: /* value */,
    refCount: 42,
}
```

#### Type Definition

```go
type GlobalConstraintBus struct {
    crossStoreConstraints map[string]Constraint
    storeRegistry map[string]LocalConstraintStore
    events chan ConstraintEvent
    eventCounter int64
    mu sync.RWMutex
    shutdown bool
    shutdownCh chan *ast.StructType
    refCount int64
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| crossStoreConstraints | `map[string]Constraint` | crossStoreConstraints holds constraints that span multiple stores |
| storeRegistry | `map[string]LocalConstraintStore` | storeRegistry tracks all active local constraint stores |
| events | `chan ConstraintEvent` | events is the channel for constraint events requiring global coordination |
| eventCounter | `int64` | eventCounter provides unique timestamps for events |
| mu | `sync.RWMutex` | mu protects concurrent access to bus state |
| shutdown | `bool` | shutdown indicates if the bus is shutting down |
| shutdownCh | `chan *ast.StructType` | shutdownCh is closed when the bus shuts down |
| refCount | `int64` | refCount tracks active references to this bus for automatic cleanup |

### Constructor Functions

### GetDefaultGlobalBus

GetDefaultGlobalBus returns a shared global constraint bus instance Use this for operations that don't require constraint isolation between goals

```go
func GetDefaultGlobalBus() *GlobalConstraintBus
```

**Parameters:**
  None

**Returns:**
- *GlobalConstraintBus

### GetPooledGlobalBus

GetPooledGlobalBus gets a constraint bus from the pool for operations that need isolation but can reuse cleaned instances

```go
func GetPooledGlobalBus() *GlobalConstraintBus
```

**Parameters:**
  None

**Returns:**
- *GlobalConstraintBus

### NewGlobalConstraintBus

NewGlobalConstraintBus creates a new global constraint bus for coordinating constraint checking across multiple local stores.

```go
func NewGlobalConstraintBus() *GlobalConstraintBus
```

**Parameters:**
  None

**Returns:**
- *GlobalConstraintBus

## Methods

### AddCrossStoreConstraint

AddCrossStoreConstraint registers a constraint that requires global coordination. Such constraints are checked whenever any relevant variable is bound in any store.

```go
func (*GlobalConstraintBus) AddCrossStoreConstraint(constraint Constraint) error
```

**Parameters:**
- `constraint` (Constraint)

**Returns:**
- error

### CoordinateBinding

CoordinateBinding attempts to bind a variable across all relevant stores. This is used when a binding might affect cross-store constraints.

```go
func (*GlobalConstraintBus) CoordinateBinding(varID int64, term Term, originStoreID string) error
```

**Parameters:**
- `varID` (int64)
- `term` (Term)
- `originStoreID` (string)

**Returns:**
- error

### RegisterStore

RegisterStore adds a local constraint store to the global registry. This enables the bus to coordinate constraints across the store.

```go
func (*GlobalConstraintBus) RegisterStore(store LocalConstraintStore) error
```

**Parameters:**
- `store` (LocalConstraintStore)

**Returns:**
- error

### Reset

Reset clears the constraint bus state for reuse in a pool. This method prepares the bus for safe reuse by clearing all state while keeping the goroutine and channels alive.

```go
func (*GlobalConstraintBus) Reset()
```

**Parameters:**
  None

**Returns:**
  None

### Shutdown

Shutdown gracefully shuts down the global constraint bus. Should be called when constraint processing is complete.

```go
func (*GlobalConstraintBus) Shutdown()
```

**Parameters:**
  None

**Returns:**
  None

### UnregisterStore

UnregisterStore removes a local constraint store from the global registry. Automatically shuts down the bus when no stores remain (reference counting).

```go
func (*GlobalConstraintBus) UnregisterStore(storeID string)
```

**Parameters:**
- `storeID` (string)

**Returns:**
  None

### handleConstraintAdded

handleConstraintAdded processes events when new constraints are added.

```go
func (*GlobalConstraintBus) handleConstraintAdded(event ConstraintEvent)
```

**Parameters:**
- `event` (ConstraintEvent)

**Returns:**
  None

### handleConstraintViolated

handleConstraintViolated processes constraint violation events.

```go
func (*GlobalConstraintBus) handleConstraintViolated(event ConstraintEvent)
```

**Parameters:**
- `event` (ConstraintEvent)

**Returns:**
  None

### handleStoreCloned

handleStoreCloned processes store cloning events for parallel execution.

```go
func (*GlobalConstraintBus) handleStoreCloned(event ConstraintEvent)
```

**Parameters:**
- `event` (ConstraintEvent)

**Returns:**
  None

### handleVariableBound

handleVariableBound processes events when variables are bound.

```go
func (*GlobalConstraintBus) handleVariableBound(event ConstraintEvent)
```

**Parameters:**
- `event` (ConstraintEvent)

**Returns:**
  None

### processEvents

processEvents handles constraint events in a dedicated goroutine. This provides asynchronous processing of cross-store constraint coordination.

```go
func (*GlobalConstraintBus) processEvents()
```

**Parameters:**
  None

**Returns:**
  None

### wouldBindingViolateConstraint

wouldBindingViolateConstraint checks if a proposed variable binding would violate a cross-store constraint by examining the combined state of all registered stores.

```go
func (*GlobalConstraintBus) wouldBindingViolateConstraint(constraint Constraint, varID int64, term Term) bool
```

**Parameters:**
- `constraint` (Constraint)
- `varID` (int64)
- `term` (Term)

**Returns:**
- bool

### GlobalConstraintBusPool
GlobalConstraintBusPool manages a pool of reusable constraint buses

#### Example Usage

```go
// Create a new GlobalConstraintBusPool
globalconstraintbuspool := GlobalConstraintBusPool{
    pool: /* value */,
}
```

#### Type Definition

```go
type GlobalConstraintBusPool struct {
    pool sync.Pool
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| pool | `sync.Pool` |  |

### Constructor Functions

### NewGlobalConstraintBusPool

NewGlobalConstraintBusPool creates a new pool of constraint buses

```go
func NewGlobalConstraintBusPool() *GlobalConstraintBusPool
```

**Parameters:**
  None

**Returns:**
- *GlobalConstraintBusPool

## Methods

### Get

Get retrieves a constraint bus from the pool

```go
func (*GlobalConstraintBusPool) Get() *GlobalConstraintBus
```

**Parameters:**
  None

**Returns:**
- *GlobalConstraintBus

### Put

Put returns a constraint bus to the pool after cleaning it

```go
func (*GlobalConstraintBusPool) Put(bus *GlobalConstraintBus)
```

**Parameters:**
- `bus` (*GlobalConstraintBus)

**Returns:**
  None

### Goal
Goal represents a constraint or a combination of constraints. Goals are functions that take a constraint store and return a stream of constraint stores representing all possible ways to satisfy the goal. Goals can be composed to build complex relational programs. The constraint store contains both variable bindings and active constraints, enabling order-independent constraint logic programming.

#### Example Usage

```go
// Example usage of Goal
var value Goal
// Initialize with appropriate value
```

#### Type Definition

```go
type Goal func(ctx context.Context, store ConstraintStore) *Stream
```

### Constructor Functions

### Absento

Absento creates a constraint ensuring that a term does not appear anywhere within another term (at any level of structure). Example: x := Fresh("x") goal := Conj(Absento(NewAtom("bad"), x), Eq(x, List(NewAtom("good"))))

```go
func Absento(absent, term Term) Goal
```

**Parameters:**
- `absent` (Term)
- `term` (Term)

**Returns:**
- Goal

### Appendo

Appendo creates a goal that relates three lists where the third list is the result of appending the first two lists. This is a classic example of a relational operation in miniKanren. Example: x := Fresh("x") goal := Appendo(List(Atom(1), Atom(2)), List(Atom(3)), x) // x will be bound to (1 2 3)

```go
func Appendo(l1, l2, l3 Term) Goal
```

**Parameters:**
- `l1` (Term)
- `l2` (Term)
- `l3` (Term)

**Returns:**
- Goal

### Car

Car extracts the first element of a pair/list. Example: goal := Car(List(NewAtom(1), NewAtom(2)), x) // x = 1

```go
func (*Pair) Car() Term
```

**Parameters:**
  None

**Returns:**
- Term

### Cdr

Cdr extracts the rest of a pair/list. Example: goal := Cdr(List(NewAtom(1), NewAtom(2)), x) // x = List(NewAtom(2))

```go
func (*Pair) Cdr() Term
```

**Parameters:**
  None

**Returns:**
- Term

### Conda

Conda implements committed choice (if-then-else with cut). Takes pairs of condition-goal clauses and commits to the first condition that succeeds. Example: goal := Conda( []Goal{condition1, thenGoal1}, []Goal{condition2, thenGoal2}, []Goal{Success, elseGoal}, // default case )

```go
func Conda(clauses ...[]Goal) Goal
```

**Parameters:**
- `clauses` (...[]Goal)

**Returns:**
- Goal

### Conde

Conde is an alias for Disj, following miniKanren naming conventions. "conde" represents "count" in Spanish, indicating enumeration of choices.

```go
func Conde(goals ...Goal) Goal
```

**Parameters:**
- `goals` (...Goal)

**Returns:**
- Goal

### Condu

Condu implements committed choice with a unique solution requirement. Like Conda but only commits if the condition has exactly one solution. Example: goal := Condu( []Goal{uniqueCondition, thenGoal}, []Goal{Success, elseGoal}, )

```go
func Condu(clauses ...[]Goal) Goal
```

**Parameters:**
- `clauses` (...[]Goal)

**Returns:**
- Goal

### Conj

Conj creates a conjunction goal that requires all goals to succeed. The goals are evaluated sequentially, with each goal operating on the constraint stores produced by the previous goal. Example: x := Fresh("x") y := Fresh("y") goal := Conj(Eq(x, NewAtom(1)), Eq(y, NewAtom(2)))

```go
func Conj(goals ...Goal) Goal
```

**Parameters:**
- `goals` (...Goal)

**Returns:**
- Goal

### Cons

Cons creates a pair/list construction goal. Example: goal := Cons(NewAtom(1), Nil, x) // x = List(NewAtom(1))

```go
func Cons(car, cdr, pair Term) Goal
```

**Parameters:**
- `car` (Term)
- `cdr` (Term)
- `pair` (Term)

**Returns:**
- Goal

### Disj

Disj creates a disjunction goal that succeeds if any of the goals succeed. This represents choice points in the search space. All solutions from all goals are included in the result stream. Example: x := Fresh("x") goal := Disj(Eq(x, NewAtom(1)), Eq(x, NewAtom(2)))  // x can be 1 or 2

```go
func Disj(goals ...Goal) Goal
```

**Parameters:**
- `goals` (...Goal)

**Returns:**
- Goal

### Eq

Eq creates a unification goal that constrains two terms to be equal. This is the fundamental operation in miniKanren - it attempts to make two terms identical by binding variables as needed. The new implementation works with constraint stores to provide order-independent constraint semantics. Variable bindings are checked against all active constraints before being accepted. Unification Rules: - Atom == Atom: succeeds if atoms have the same value - Var == Term: binds the variable to the term (subject to constraints) - Pair == Pair: recursively unifies car and cdr - Otherwise: fails Example: x := Fresh("x") goal := Eq(x, NewAtom("hello"))  // Binds x to "hello"

```go
func Eq(term1, term2 Term) Goal
```

**Parameters:**
- `term1` (Term)
- `term2` (Term)

**Returns:**
- Goal

### Membero

Membero creates a goal that relates an element to a list it's a member of. This is the relational membership predicate. Example: x := Fresh("x") list := List(NewAtom(1), NewAtom(2), NewAtom(3)) goal := Membero(x, list) // x can be 1, 2, or 3

```go
func Membero(element, list Term) Goal
```

**Parameters:**
- `element` (Term)
- `list` (Term)

**Returns:**
- Goal

### Neq

Neq creates a disequality constraint that ensures two terms are NOT equal. This is a constraint that's checked during unification and can cause goals to fail if the constraint would be violated. Example: x := Fresh("x") goal := Conj(Neq(x, NewAtom("forbidden")), Eq(x, NewAtom("allowed"))) Neq implements the disequality constraint. It ensures that two terms are not equal.

```go
func Neq(t1, t2 Term) Goal
```

**Parameters:**
- `t1` (Term)
- `t2` (Term)

**Returns:**
- Goal

### Nullo

Nullo checks if a term is the empty list (nil). Example: goal := Nullo(x) // x must be nil

```go
func Nullo(term Term) Goal
```

**Parameters:**
- `term` (Term)

**Returns:**
- Goal

### Numbero

Numbero constrains a term to be a number. Example: x := Fresh("x") goal := Conj(Numbero(x), Eq(x, NewAtom(42)))

```go
func Numbero(term Term) Goal
```

**Parameters:**
- `term` (Term)

**Returns:**
- Goal

### Onceo

Onceo ensures a goal succeeds at most once (cuts choice points). Example: goal := Onceo(Disj(Eq(x, NewAtom(1)), Eq(x, NewAtom(2)))) // Will only return the first solution

```go
func Onceo(goal Goal) Goal
```

**Parameters:**
- `goal` (Goal)

**Returns:**
- Goal

### Pairo

Pairo checks if a term is a pair (non-empty list). Example: goal := Pairo(x) // x must be a pair

```go
func Pairo(term Term) Goal
```

**Parameters:**
- `term` (Term)

**Returns:**
- Goal

### Project

Project extracts the values of variables from the current substitution and passes them to a function that creates a new goal. Example: goal := Project([]Term{x, y}, func(values []Term) Goal { // values[0] is the value of x, values[1] is the value of y return someGoalUsing(values) })

```go
func Project(vars []Term, goalFunc func([]Term) Goal) Goal
```

**Parameters:**
- `vars` ([]Term)
- `goalFunc` (func([]Term) Goal)

**Returns:**
- Goal

### Symbolo

Symbolo constrains a term to be a symbol (string atom). Example: x := Fresh("x") goal := Conj(Symbolo(x), Eq(x, NewAtom("symbol")))

```go
func Symbolo(term Term) Goal
```

**Parameters:**
- `term` (Term)

**Returns:**
- Goal

### LocalConstraintStore
LocalConstraintStore interface defines the operations needed by the GlobalConstraintBus to coordinate with local stores.

#### Example Usage

```go
// Example implementation of LocalConstraintStore
type MyLocalConstraintStore struct {
    // Add your fields here
}

func (m MyLocalConstraintStore) ID() string {
    // Implement your logic here
    return
}

func (m MyLocalConstraintStore) getAllBindings() map[int64]Term {
    // Implement your logic here
    return
}


```

#### Type Definition

```go
type LocalConstraintStore interface {
    ID() string
    getAllBindings() map[int64]Term
}
```

## Methods

| Method | Description |
| ------ | ----------- |

### LocalConstraintStoreImpl
LocalConstraintStoreImpl provides a concrete implementation of LocalConstraintStore for managing constraints and variable bindings within a single goal context. The store maintains two separate collections: - Local constraints: Checked quickly without global coordination - Local bindings: Variable-to-term mappings for this context When constraints or bindings are added, the store first checks all local constraints for immediate violations, then coordinates with the global bus if necessary for cross-store constraints.

#### Example Usage

```go
// Create a new LocalConstraintStoreImpl
localconstraintstoreimpl := LocalConstraintStoreImpl{
    id: "example",
    constraints: [],
    bindings: map[],
    globalBus: &GlobalConstraintBus{}{},
    generation: 42,
    mu: /* value */,
}
```

#### Type Definition

```go
type LocalConstraintStoreImpl struct {
    id string
    constraints []Constraint
    bindings map[int64]Term
    globalBus *GlobalConstraintBus
    generation int64
    mu sync.RWMutex
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| id | `string` | id uniquely identifies this store instance |
| constraints | `[]Constraint` | constraints holds all local constraints for this store |
| bindings | `map[int64]Term` | bindings maps variable IDs to their bound terms |
| globalBus | `*GlobalConstraintBus` | globalBus coordinates cross-store constraints (optional) |
| generation | `int64` | generation tracks the number of modifications for efficient cloning |
| mu | `sync.RWMutex` | mu protects concurrent access to store state |

### Constructor Functions

### NewLocalConstraintStore

NewLocalConstraintStore creates a new local constraint store with optional global constraint bus integration. If globalBus is nil, the store operates in local-only mode with no cross-store constraint coordination. This is suitable for simple use cases where all constraints are local.

```go
func NewLocalConstraintStore(globalBus *GlobalConstraintBus) *LocalConstraintStoreImpl
```

**Parameters:**
- `globalBus` (*GlobalConstraintBus)

**Returns:**
- *LocalConstraintStoreImpl

## Methods

### AddBinding

AddBinding attempts to bind a variable to a term, checking all relevant constraints for violations. The binding process follows these steps: 1. Check all local constraints against the proposed binding 2. If any local constraint is violated, reject the binding 3. If the binding affects cross-store constraints, coordinate with global bus 4. If all checks pass, add the binding to the store Returns an error if the binding would violate any constraint.

```go
func (*LocalConstraintStoreImpl) AddBinding(varID int64, term Term) error
```

**Parameters:**
- `varID` (int64)
- `term` (Term)

**Returns:**
- error

### AddConstraint

AddConstraint adds a new constraint to the store and checks it against current bindings for immediate violations. The constraint is first checked locally for immediate violations. If the constraint is not local (requires global coordination), it is also registered with the global constraint bus. Returns an error if the constraint is immediately violated.

```go
func (*LocalConstraintStoreImpl) AddConstraint(constraint Constraint) error
```

**Parameters:**
- `constraint` (Constraint)

**Returns:**
- error

### Clone

Clone creates a deep copy of the constraint store for parallel execution. The clone shares no mutable state with the original store, making it safe for concurrent use in parallel goal evaluation. Cloning is optimized for performance as it's used frequently in parallel execution contexts. The clone initially shares constraint references with the original but will copy-on-write if modified. Implements the ConstraintStore interface.

```go
func (*Substitution) Clone() *Substitution
```

**Parameters:**
  None

**Returns:**
- *Substitution

### Generation

Generation returns the current generation number of the store. The generation increments with each modification, enabling efficient change detection and caching strategies.

```go
func (*LocalConstraintStoreImpl) Generation() int64
```

**Parameters:**
  None

**Returns:**
- int64

### GetBinding

GetBinding retrieves the current binding for a variable. Returns nil if the variable is unbound. Implements the ConstraintStore interface.

```go
func (*LocalConstraintStoreImpl) GetBinding(varID int64) Term
```

**Parameters:**
- `varID` (int64)

**Returns:**
- Term

### GetConstraints

GetConstraints returns a copy of all constraints in the store. Used for debugging and testing purposes.

```go
func (*LocalConstraintStoreImpl) GetConstraints() []Constraint
```

**Parameters:**
  None

**Returns:**
- []Constraint

### GetSubstitution

GetSubstitution returns a substitution representing all current bindings. This bridges between the constraint store system and the existing miniKanren substitution-based APIs. Implements the ConstraintStore interface.

```go
func (*LocalConstraintStoreImpl) GetSubstitution() *Substitution
```

**Parameters:**
  None

**Returns:**
- *Substitution

### ID

ID returns the unique identifier for this constraint store. Implements the LocalConstraintStore interface.

```go
func (*Var) ID() int64
```

**Parameters:**
  None

**Returns:**
- int64

### IsEmpty

IsEmpty returns true if the store has no constraints or bindings. Useful for optimization and testing.

```go
func (*LocalConstraintStoreImpl) IsEmpty() bool
```

**Parameters:**
  None

**Returns:**
- bool

### Shutdown

Shutdown cleanly shuts down the store and unregisters it from the global constraint bus. Should be called when the store is no longer needed to prevent memory leaks.

```go
func (*ParallelExecutor) Shutdown()
```

**Parameters:**
  None

**Returns:**
  None

### String

String returns a human-readable representation of the constraint store for debugging and error reporting. Implements the ConstraintStore interface.

```go
func (ConstraintEventType) String() string
```

**Parameters:**
  None

**Returns:**
- string

### getAllBindings

getAllBindings returns a copy of all current bindings. Used by the global constraint bus for cross-store constraint checking. Implements the LocalConstraintStore interface.

```go
func (*LocalConstraintStoreImpl) getAllBindings() map[int64]Term
```

**Parameters:**
  None

**Returns:**
- map[int64]Term

### MembershipConstraint
MembershipConstraint implements the membership constraint (membero). It ensures that an element is a member of a list, providing relational list membership checking that can work in both directions.

#### Example Usage

```go
// Create a new MembershipConstraint
membershipconstraint := MembershipConstraint{
    id: "example",
    element: Term{},
    list: Term{},
    isLocal: true,
}
```

#### Type Definition

```go
type MembershipConstraint struct {
    id string
    element Term
    list Term
    isLocal bool
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| id | `string` | id uniquely identifies this constraint instance |
| element | `Term` | element is the term that should be a member of the list |
| list | `Term` | list is the list that should contain the element |
| isLocal | `bool` | isLocal indicates whether this constraint can be checked locally |

### Constructor Functions

### NewMembershipConstraint

NewMembershipConstraint creates a new membership constraint.

```go
func NewMembershipConstraint(element, list Term) *MembershipConstraint
```

**Parameters:**
- `element` (Term)
- `list` (Term)

**Returns:**
- *MembershipConstraint

## Methods

### Check

Check evaluates the membership constraint against current bindings. Note: This is a simplified implementation. The full membero relation is typically implemented as a recursive goal rather than a simple constraint. Implements the Constraint interface.

```go
func (*MembershipConstraint) Check(bindings map[int64]Term) ConstraintResult
```

**Parameters:**
- `bindings` (map[int64]Term)

**Returns:**
- ConstraintResult

### Clone

Clone creates a deep copy of the constraint for parallel execution. Implements the Constraint interface.

```go
func (*Substitution) Clone() *Substitution
```

**Parameters:**
  None

**Returns:**
- *Substitution

### ID

ID returns the unique identifier for this constraint instance. Implements the Constraint interface.

```go
func (*MembershipConstraint) ID() string
```

**Parameters:**
  None

**Returns:**
- string

### IsLocal

IsLocal returns true if this constraint can be evaluated locally. Implements the Constraint interface.

```go
func (*MembershipConstraint) IsLocal() bool
```

**Parameters:**
  None

**Returns:**
- bool

### String

String returns a human-readable representation of the constraint. Implements the Constraint interface.

```go
func (*LocalConstraintStoreImpl) String() string
```

**Parameters:**
  None

**Returns:**
- string

### Variables

Variables returns the logic variables this constraint depends on. Implements the Constraint interface.

```go
func (*MembershipConstraint) Variables() []*Var
```

**Parameters:**
  None

**Returns:**
- []*Var

### Pair
Pair represents a cons cell (pair) in miniKanren. Pairs are used to build lists and other compound structures.

#### Example Usage

```go
// Create a new Pair
pair := Pair{
    car: Term{},
    cdr: Term{},
    mu: /* value */,
}
```

#### Type Definition

```go
type Pair struct {
    car Term
    cdr Term
    mu sync.RWMutex
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| car | `Term` | First element |
| cdr | `Term` | Rest of the structure |
| mu | `sync.RWMutex` | Protects concurrent access |

### Constructor Functions

### NewPair

NewPair creates a new pair with the given car and cdr.

```go
func NewPair(car, cdr Term) *Pair
```

**Parameters:**
- `car` (Term)
- `cdr` (Term)

**Returns:**
- *Pair

## Methods

### Car

Car returns the first element of the pair.

```go
func (*Pair) Car() Term
```

**Parameters:**
  None

**Returns:**
- Term

### Cdr

Cdr returns the rest of the pair.

```go
func Cdr(pair, cdr Term) Goal
```

**Parameters:**
- `pair` (Term)
- `cdr` (Term)

**Returns:**
- Goal

### Clone

Clone creates a deep copy of the pair.

```go
func (*Substitution) Clone() *Substitution
```

**Parameters:**
  None

**Returns:**
- *Substitution

### Equal

Equal checks if two pairs are structurally equal.

```go
func (*Pair) Equal(other Term) bool
```

**Parameters:**
- `other` (Term)

**Returns:**
- bool

### IsVar

IsVar always returns false for pairs.

```go
func (*Pair) IsVar() bool
```

**Parameters:**
  None

**Returns:**
- bool

### String

String returns a string representation of the pair.

```go
func (*Substitution) String() string
```

**Parameters:**
  None

**Returns:**
- string

### ParallelConfig
ParallelConfig holds configuration for parallel goal execution.

#### Example Usage

```go
// Create a new ParallelConfig
parallelconfig := ParallelConfig{
    MaxWorkers: 42,
    MaxQueueSize: 42,
    EnableBackpressure: true,
    RateLimit: 42,
}
```

#### Type Definition

```go
type ParallelConfig struct {
    MaxWorkers int
    MaxQueueSize int
    EnableBackpressure bool
    RateLimit int
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| MaxWorkers | `int` | MaxWorkers is the maximum number of concurrent workers. If 0, defaults to runtime.NumCPU(). |
| MaxQueueSize | `int` | MaxQueueSize is the maximum number of pending tasks. If 0, defaults to MaxWorkers * 10. |
| EnableBackpressure | `bool` | EnableBackpressure enables backpressure control to prevent memory exhaustion during large search spaces. |
| RateLimit | `int` | RateLimit sets the maximum operations per second. If 0, no rate limiting is applied. |

### Constructor Functions

### DefaultParallelConfig

DefaultParallelConfig returns a default configuration for parallel execution.

```go
func DefaultParallelConfig() *ParallelConfig
```

**Parameters:**
  None

**Returns:**
- *ParallelConfig

### ParallelExecutor
ParallelExecutor manages parallel execution of miniKanren goals.

#### Example Usage

```go
// Create a new ParallelExecutor
parallelexecutor := ParallelExecutor{
    config: &ParallelConfig{}{},
    workerPool: &/* value */{},
    backpressureCtrl: &/* value */{},
    rateLimiter: &/* value */{},
    mu: /* value */,
    shutdown: true,
}
```

#### Type Definition

```go
type ParallelExecutor struct {
    config *ParallelConfig
    workerPool *parallel.WorkerPool
    backpressureCtrl *parallel.BackpressureController
    rateLimiter *parallel.RateLimiter
    mu sync.RWMutex
    shutdown bool
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| config | `*ParallelConfig` |  |
| workerPool | `*parallel.WorkerPool` |  |
| backpressureCtrl | `*parallel.BackpressureController` |  |
| rateLimiter | `*parallel.RateLimiter` |  |
| mu | `sync.RWMutex` |  |
| shutdown | `bool` |  |

### Constructor Functions

### NewParallelExecutor

NewParallelExecutor creates a new parallel executor with the given configuration.

```go
func NewParallelExecutor(config *ParallelConfig) *ParallelExecutor
```

**Parameters:**
- `config` (*ParallelConfig)

**Returns:**
- *ParallelExecutor

## Methods

### ParallelDisj

ParallelDisj creates a disjunction goal that evaluates all sub-goals in parallel using the parallel executor. This can significantly improve performance when dealing with computationally intensive goals or large search spaces.

```go
func (*ParallelExecutor) ParallelDisj(goals ...Goal) Goal
```

**Parameters:**
- `goals` (...Goal)

**Returns:**
- Goal

### Shutdown

Shutdown gracefully shuts down the parallel executor.

```go
func (*LocalConstraintStoreImpl) Shutdown()
```

**Parameters:**
  None

**Returns:**
  None

### ParallelStream
ParallelStream represents a stream that can be evaluated in parallel. It wraps the standard Stream with additional parallel capabilities.

#### Example Usage

```go
// Create a new ParallelStream
parallelstream := ParallelStream{
    executor: &ParallelExecutor{}{},
    ctx: /* value */,
}
```

#### Type Definition

```go
type ParallelStream struct {
    *Stream
    executor *ParallelExecutor
    ctx context.Context
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| **Stream | `*Stream` |  |
| executor | `*ParallelExecutor` |  |
| ctx | `context.Context` |  |

### Constructor Functions

### NewParallelStream

NewParallelStream creates a new parallel stream with the given executor.

```go
func NewParallelStream(ctx context.Context, executor *ParallelExecutor) *ParallelStream
```

**Parameters:**
- `ctx` (context.Context)
- `executor` (*ParallelExecutor)

**Returns:**
- *ParallelStream

## Methods

### Collect

Collect gathers all constraint stores from the parallel stream.

```go
func (*ParallelStream) Collect() []ConstraintStore
```

**Parameters:**
  None

**Returns:**
- []ConstraintStore

### ParallelFilter

ParallelFilter filters constraint stores in the stream in parallel.

```go
func (*ParallelStream) ParallelFilter(predicate func(ConstraintStore) bool) *ParallelStream
```

**Parameters:**
- `predicate` (func(ConstraintStore) bool)

**Returns:**
- *ParallelStream

### ParallelMap

ParallelMap applies a function to each constraint store in the stream in parallel.

```go
func (*ParallelStream) ParallelMap(fn func(ConstraintStore) ConstraintStore) *ParallelStream
```

**Parameters:**
- `fn` (func(ConstraintStore) ConstraintStore)

**Returns:**
- *ParallelStream

### Stream
Stream represents a (potentially infinite) sequence of constraint stores. Streams are the core data structure for representing multiple solutions in miniKanren. Each constraint store contains variable bindings and active constraints representing a consistent logical state. This implementation uses channels for thread-safe concurrent access and supports parallel evaluation with proper constraint coordination.

#### Example Usage

```go
// Create a new Stream
stream := Stream{
    ch: /* value */,
    done: /* value */,
    mu: /* value */,
}
```

#### Type Definition

```go
type Stream struct {
    ch chan ConstraintStore
    done chan *ast.StructType
    mu sync.Mutex
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| ch | `chan ConstraintStore` | Channel for streaming constraint stores |
| done | `chan *ast.StructType` | Channel to signal completion |
| mu | `sync.Mutex` | Protects stream state |

### Constructor Functions

### NewStream

NewStream creates a new empty stream.

```go
func NewStream() *Stream
```

**Parameters:**
  None

**Returns:**
- *Stream

### conjHelper

conjHelper recursively evaluates conjunction goals

```go
func conjHelper(ctx context.Context, goals []Goal, store ConstraintStore) *Stream
```

**Parameters:**
- `ctx` (context.Context)
- `goals` ([]Goal)
- `store` (ConstraintStore)

**Returns:**
- *Stream

## Methods

### Close

Close closes the stream, indicating no more substitutions will be added.

```go
func (*Stream) Close()
```

**Parameters:**
  None

**Returns:**
  None

### Put



```go
func (*GlobalConstraintBusPool) Put(bus *GlobalConstraintBus)
```

**Parameters:**
- `bus` (*GlobalConstraintBus)

**Returns:**
  None

### Take

Take retrieves up to n constraint stores from the stream. Returns a slice of constraint stores and a boolean indicating if more stores might be available.

```go
func (*Stream) Take(n int) ([]ConstraintStore, bool)
```

**Parameters:**
- `n` (int)

**Returns:**
- []ConstraintStore
- bool

### Substitution
Substitution represents a mapping from variables to terms. It's used to track bindings during unification and goal evaluation. The implementation is thread-safe and supports concurrent access.

#### Example Usage

```go
// Create a new Substitution
substitution := Substitution{
    bindings: map[],
    mu: /* value */,
}
```

#### Type Definition

```go
type Substitution struct {
    bindings map[int64]Term
    mu sync.RWMutex
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| bindings | `map[int64]Term` | Maps variable IDs to terms |
| mu | `sync.RWMutex` | Protects concurrent access |

### Constructor Functions

### NewSubstitution

NewSubstitution creates an empty substitution.

```go
func NewSubstitution() *Substitution
```

**Parameters:**
  None

**Returns:**
- *Substitution

### unify

unify performs the unification algorithm. Returns a new substitution if unification succeeds, nil if it fails.

```go
func unify(term1, term2 Term, sub *Substitution) *Substitution
```

**Parameters:**
- `term1` (Term)
- `term2` (Term)
- `sub` (*Substitution)

**Returns:**
- *Substitution

## Methods

### Bind

Bind creates a new substitution with an additional binding. Returns nil if the binding would create an inconsistency.

```go
func (*Substitution) Bind(v *Var, term Term) *Substitution
```

**Parameters:**
- `v` (*Var)
- `term` (Term)

**Returns:**
- *Substitution

### Clone

Clone creates a deep copy of the substitution.

```go
func (*Substitution) Clone() *Substitution
```

**Parameters:**
  None

**Returns:**
- *Substitution

### Lookup

Lookup returns the term bound to a variable, or nil if unbound.

```go
func (*Substitution) Lookup(v *Var) Term
```

**Parameters:**
- `v` (*Var)

**Returns:**
- Term

### Size

Size returns the number of bindings in the substitution.

```go
func (*Substitution) Size() int
```

**Parameters:**
  None

**Returns:**
- int

### String

String returns a string representation of the substitution.

```go
func (ConstraintEventType) String() string
```

**Parameters:**
  None

**Returns:**
- string

### Walk

Walk traverses a term following variable bindings in the substitution.

```go
func (*Substitution) Walk(term Term) Term
```

**Parameters:**
- `term` (Term)

**Returns:**
- Term

### Term
Term represents any value in the miniKanren universe. Terms can be atoms, variables, compound structures, or any Go value. All Term implementations must be comparable and thread-safe.

#### Example Usage

```go
// Example implementation of Term
type MyTerm struct {
    // Add your fields here
}

func (m MyTerm) String() string {
    // Implement your logic here
    return
}

func (m MyTerm) Equal(param1 Term) bool {
    // Implement your logic here
    return
}

func (m MyTerm) IsVar() bool {
    // Implement your logic here
    return
}

func (m MyTerm) Clone() Term {
    // Implement your logic here
    return
}


```

#### Type Definition

```go
type Term interface {
    String() string
    Equal(other Term) bool
    IsVar() bool
    Clone() Term
}
```

## Methods

| Method | Description |
| ------ | ----------- |

### Constructor Functions

### List

List creates a list (chain of pairs) from a slice of terms. The list is terminated with nil (empty list). Example: lst := List(NewAtom(1), NewAtom(2), NewAtom(3)) // Creates: (1 . (2 . (3 . nil)))

```go
func List(terms ...Term) Term
```

**Parameters:**
- `terms` (...Term)

**Returns:**
- Term

### ParallelRun

ParallelRun executes a goal in parallel and returns up to n solutions. This function creates a parallel executor, runs the goal, and cleans up.

```go
func ParallelRun(n int, goalFunc func(*Var) Goal) []Term
```

**Parameters:**
- `n` (int)
- `goalFunc` (func(*Var) Goal)

**Returns:**
- []Term

### ParallelRunWithConfig

ParallelRunWithConfig executes a goal in parallel with custom configuration.

```go
func ParallelRunWithConfig(n int, goalFunc func(*Var) Goal, config *ParallelConfig) []Term
```

**Parameters:**
- `n` (int)
- `goalFunc` (func(*Var) Goal)
- `config` (*ParallelConfig)

**Returns:**
- []Term

### ParallelRunWithContext

ParallelRunWithContext executes a goal in parallel with context and configuration.

```go
func ParallelRunWithContext(ctx context.Context, n int, goalFunc func(*Var) Goal, config *ParallelConfig) []Term
```

**Parameters:**
- `ctx` (context.Context)
- `n` (int)
- `goalFunc` (func(*Var) Goal)
- `config` (*ParallelConfig)

**Returns:**
- []Term

### Run

Run executes a goal and returns up to n solutions. This is the main entry point for executing miniKanren programs. It takes a goal that introduces one or more fresh variables and returns the values those variables can take. Example: solutions := Run(5, func(q *Var) Goal { return Eq(q, NewAtom("hello")) }) // Returns: [hello]

```go
func Run(n int, goalFunc func(*Var) Goal) []Term
```

**Parameters:**
- `n` (int)
- `goalFunc` (func(*Var) Goal)

**Returns:**
- []Term

### RunStar

RunStar executes a goal and returns all solutions. WARNING: This can run forever if the goal has infinite solutions. Use RunWithContext with a timeout for safer execution. Example: solutions := RunStar(func(q *Var) Goal { return Disj(Eq(q, NewAtom(1)), Eq(q, NewAtom(2))) }) // Returns: [1, 2]

```go
func RunStar(goalFunc func(*Var) Goal) []Term
```

**Parameters:**
- `goalFunc` (func(*Var) Goal)

**Returns:**
- []Term

### RunStarWithContext

RunStarWithContext executes a goal and returns all solutions with context support.

```go
func RunStarWithContext(ctx context.Context, goalFunc func(*Var) Goal) []Term
```

**Parameters:**
- `ctx` (context.Context)
- `goalFunc` (func(*Var) Goal)

**Returns:**
- []Term

### RunWithContext

RunWithContext executes a goal with a context for cancellation and timeouts. This allows for better control over long-running or infinite searches. Example: ctx, cancel := context.WithTimeout(context.Background(), time.Second) defer cancel() solutions := RunWithContext(ctx, 100, func(q *Var) Goal { return someLongRunningGoal(q) })

```go
func RunWithContext(ctx context.Context, n int, goalFunc func(*Var) Goal) []Term
```

**Parameters:**
- `ctx` (context.Context)
- `n` (int)
- `goalFunc` (func(*Var) Goal)

**Returns:**
- []Term

### RunWithIsolation

RunWithIsolation is like Run but uses an isolated constraint bus. Use this when you need complete constraint isolation between goals. Slightly slower than Run() but provides stronger isolation guarantees.

```go
func RunWithIsolation(n int, goalFunc func(*Var) Goal) []Term
```

**Parameters:**
- `n` (int)
- `goalFunc` (func(*Var) Goal)

**Returns:**
- []Term

### RunWithIsolationContext

RunWithIsolationContext is like RunWithContext but uses an isolated constraint bus.

```go
func RunWithIsolationContext(ctx context.Context, n int, goalFunc func(*Var) Goal) []Term
```

**Parameters:**
- `ctx` (context.Context)
- `n` (int)
- `goalFunc` (func(*Var) Goal)

**Returns:**
- []Term

### walkTerm

walkTerm follows variable bindings to find the final value of a term.

```go
func walkTerm(term Term, bindings map[int64]Term) Term
```

**Parameters:**
- `term` (Term)
- `bindings` (map[int64]Term)

**Returns:**
- Term

### TypeConstraint
TypeConstraint implements type-based constraints (symbolo, numbero, etc.). It ensures that a term has a specific type, enabling type-safe relational programming patterns.

#### Example Usage

```go
// Create a new TypeConstraint
typeconstraint := TypeConstraint{
    id: "example",
    term: Term{},
    expectedType: TypeConstraintKind{},
    isLocal: true,
}
```

#### Type Definition

```go
type TypeConstraint struct {
    id string
    term Term
    expectedType TypeConstraintKind
    isLocal bool
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| id | `string` | id uniquely identifies this constraint instance |
| term | `Term` | term is the term that must have the specified type |
| expectedType | `TypeConstraintKind` | expectedType specifies what type the term must have |
| isLocal | `bool` | isLocal indicates whether this constraint can be checked locally |

### Constructor Functions

### NewTypeConstraint

NewTypeConstraint creates a new type constraint.

```go
func NewTypeConstraint(term Term, expectedType TypeConstraintKind) *TypeConstraint
```

**Parameters:**
- `term` (Term)
- `expectedType` (TypeConstraintKind)

**Returns:**
- *TypeConstraint

## Methods

### Check

Check evaluates the type constraint against current bindings. Returns ConstraintViolated if the term has the wrong type, ConstraintPending if the term is unbound, or ConstraintSatisfied if the term has the correct type. Implements the Constraint interface.

```go
func (*MembershipConstraint) Check(bindings map[int64]Term) ConstraintResult
```

**Parameters:**
- `bindings` (map[int64]Term)

**Returns:**
- ConstraintResult

### Clone

Clone creates a deep copy of the constraint for parallel execution. Implements the Constraint interface.

```go
func (*Substitution) Clone() *Substitution
```

**Parameters:**
  None

**Returns:**
- *Substitution

### ID

ID returns the unique identifier for this constraint instance. Implements the Constraint interface.

```go
func (*LocalConstraintStoreImpl) ID() string
```

**Parameters:**
  None

**Returns:**
- string

### IsLocal

IsLocal returns true if this constraint can be evaluated locally. Implements the Constraint interface.

```go
func (*MembershipConstraint) IsLocal() bool
```

**Parameters:**
  None

**Returns:**
- bool

### String

String returns a human-readable representation of the constraint. Implements the Constraint interface.

```go
func (*LocalConstraintStoreImpl) String() string
```

**Parameters:**
  None

**Returns:**
- string

### Variables

Variables returns the logic variables this constraint depends on. Implements the Constraint interface.

```go
func (*MembershipConstraint) Variables() []*Var
```

**Parameters:**
  None

**Returns:**
- []*Var

### hasExpectedType

hasExpectedType checks if a term has the type expected by this constraint.

```go
func (*TypeConstraint) hasExpectedType(term Term) bool
```

**Parameters:**
- `term` (Term)

**Returns:**
- bool

### TypeConstraintKind
TypeConstraintKind represents the different types that can be constrained.

#### Example Usage

```go
// Example usage of TypeConstraintKind
var value TypeConstraintKind
// Initialize with appropriate value
```

#### Type Definition

```go
type TypeConstraintKind int
```

## Methods

### String

String returns a human-readable representation of the type constraint kind.

```go
func (ConstraintEventType) String() string
```

**Parameters:**
  None

**Returns:**
- string

### Var
Var represents a logic variable in miniKanren. Variables can be bound to values through unification. Each variable has a unique identifier to distinguish it from others.

#### Example Usage

```go
// Create a new Var
var := Var{
    id: 42,
    name: "example",
    mu: /* value */,
}
```

#### Type Definition

```go
type Var struct {
    id int64
    name string
    mu sync.RWMutex
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| id | `int64` | Unique identifier |
| name | `string` | Optional name for debugging |
| mu | `sync.RWMutex` | Protects concurrent access |

### Constructor Functions

### Fresh

Fresh creates a new logic variable with an optional name for debugging. Each call to Fresh generates a variable with a globally unique ID, ensuring no variable conflicts even in concurrent environments. Example: x := Fresh("x")  // Creates a variable named x y := Fresh("")   // Creates an anonymous variable

```go
func Fresh(name string) *Var
```

**Parameters:**
- `name` (string)

**Returns:**
- *Var

### extractVariables

extractVariables recursively extracts all variables from a term.

```go
func extractVariables(term Term) []*Var
```

**Parameters:**
- `term` (Term)

**Returns:**
- []*Var

## Methods

### Clone

Clone creates a copy of the variable with the same identity.

```go
func (*MembershipConstraint) Clone() Constraint
```

**Parameters:**
  None

**Returns:**
- Constraint

### Equal

Equal checks if two variables are the same variable.

```go
func (*Pair) Equal(other Term) bool
```

**Parameters:**
- `other` (Term)

**Returns:**
- bool

### ID

ID returns the unique identifier of the variable.

```go
func (*MembershipConstraint) ID() string
```

**Parameters:**
  None

**Returns:**
- string

### IsVar

IsVar always returns true for variables.

```go
func (*Pair) IsVar() bool
```

**Parameters:**
  None

**Returns:**
- bool

### String

String returns a string representation of the variable.

```go
func (ConstraintEventType) String() string
```

**Parameters:**
  None

**Returns:**
- string

### VersionInfo
VersionInfo provides detailed version information.

#### Example Usage

```go
// Create a new VersionInfo
versioninfo := VersionInfo{
    Version: "example",
    GoVersion: "example",
    GitCommit: "example",
    BuildDate: "example",
}
```

#### Type Definition

```go
type VersionInfo struct {
    Version string `json:"version"`
    GoVersion string `json:"go_version"`
    GitCommit string `json:"git_commit,omitempty"`
    BuildDate string `json:"build_date,omitempty"`
}
```

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| Version | `string` |  |
| GoVersion | `string` |  |
| GitCommit | `string` |  |
| BuildDate | `string` |  |

### Constructor Functions

### GetVersionInfo

GetVersionInfo returns detailed version information.

```go
func GetVersionInfo() VersionInfo
```

**Parameters:**
  None

**Returns:**
- VersionInfo

## Functions

### GetVersion
GetVersion returns the current version string.

```go
func GetVersion() string
```

**Parameters:**
None

**Returns:**
| Type | Description |
|------|-------------|
| `string` | |

**Example:**

```go
// Example usage of GetVersion
result := GetVersion(/* parameters */)
```

### ReturnPooledGlobalBus
ReturnPooledGlobalBus returns a bus to the pool

```go
func ReturnPooledGlobalBus(bus *GlobalConstraintBus)
```

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `bus` | `*GlobalConstraintBus` | |

**Returns:**
None

**Example:**

```go
// Example usage of ReturnPooledGlobalBus
result := ReturnPooledGlobalBus(/* parameters */)
```

## External Links

- [Package Overview](../packages/minikanren.md)
- [pkg.go.dev Documentation](https://pkg.go.dev/github.com/gitrdm/gokando/pkg/minikanren)
- [Source Code](https://github.com/gitrdm/gokando/tree/master/pkg/minikanren)
