// Package minikanren provides a thread-safe, parallel implementation of miniKanren
// in Go. This implementation follows the core principles of relational programming
// while leveraging Go's concurrency primitives for parallel execution.
//
// miniKanren is a domain-specific language for constraint logic programming.
// It provides a minimal set of operators for building relational programs:
//   - Unification (==): Constrains two terms to be equal
//   - Fresh variables: Introduces new logic variables
//   - Disjunction (conde): Represents choice points
//   - Conjunction: Combines goals that must all succeed
//   - Run: Executes a goal and returns solutions
//
// This implementation is designed for production use with:
//   - Thread-safe operations using sync package primitives
//   - Parallel goal evaluation using goroutines and channels
//   - Type-safe interfaces leveraging Go's type system
//   - Comprehensive error handling and resource management
package minikanren

import (
	"context"
	"fmt"
	"runtime"
	"sync"
)

// Term represents any value in the miniKanren universe.
// Terms can be atoms, variables, compound structures, or any Go value.
// All Term implementations must be comparable and thread-safe.
type Term interface {
	// String returns a human-readable representation of the term.
	String() string

	// Equal checks if this term is structurally equal to another term.
	// This is different from unification - it's a strict equality check.
	Equal(other Term) bool

	// IsVar returns true if this term is a logic variable.
	IsVar() bool

	// Clone creates a deep copy of the term for thread-safety.
	Clone() Term
}

// Var represents a logic variable in miniKanren.
// Variables can be bound to values through unification.
// Each variable has a unique identifier to distinguish it from others.
type Var struct {
	id   int64        // Unique identifier
	name string       // Optional name for debugging
	mu   sync.RWMutex // Protects concurrent access
}

// String returns a string representation of the variable.
func (v *Var) String() string {
	v.mu.RLock()
	defer v.mu.RUnlock()
	if v.name != "" {
		return fmt.Sprintf("_%s_%d", v.name, v.id)
	}
	return fmt.Sprintf("_%d", v.id)
}

// ID returns the unique identifier of the variable.
func (v *Var) ID() int64 {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.id
}

// Equal checks if two variables are the same variable.
func (v *Var) Equal(other Term) bool {
	if otherVar, ok := other.(*Var); ok {
		v.mu.RLock()
		otherVar.mu.RLock()
		defer v.mu.RUnlock()
		defer otherVar.mu.RUnlock()
		return v.id == otherVar.id
	}
	return false
}

// IsVar always returns true for variables.
func (v *Var) IsVar() bool {
	return true
}

// Clone creates a copy of the variable with the same identity.
func (v *Var) Clone() Term {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return &Var{id: v.id, name: v.name}
}

// Atom represents an atomic value (symbol, number, string, etc.).
// Atoms are immutable and represent themselves.
type Atom struct {
	value interface{} // The underlying Go value
}

// NewAtom creates a new atom from any Go value.
func NewAtom(value interface{}) *Atom {
	return &Atom{value: value}
}

// String returns a string representation of the atom.
func (a *Atom) String() string {
	return fmt.Sprintf("%v", a.value)
}

// Equal checks if two atoms have the same value.
func (a *Atom) Equal(other Term) bool {
	if otherAtom, ok := other.(*Atom); ok {
		return a.value == otherAtom.value
	}
	return false
}

// IsVar always returns false for atoms.
func (a *Atom) IsVar() bool {
	return false
}

// Clone creates a copy of the atom.
func (a *Atom) Clone() Term {
	return &Atom{value: a.value}
}

// Value returns the underlying Go value.
func (a *Atom) Value() interface{} {
	return a.value
}

// Pair represents a cons cell (pair) in miniKanren.
// Pairs are used to build lists and other compound structures.
type Pair struct {
	car Term         // First element
	cdr Term         // Rest of the structure
	mu  sync.RWMutex // Protects concurrent access
}

// NewPair creates a new pair with the given car and cdr.
func NewPair(car, cdr Term) *Pair {
	return &Pair{car: car, cdr: cdr}
}

// String returns a string representation of the pair.
func (p *Pair) String() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return fmt.Sprintf("(%s . %s)", p.car.String(), p.cdr.String())
}

// Equal checks if two pairs are structurally equal.
func (p *Pair) Equal(other Term) bool {
	if otherPair, ok := other.(*Pair); ok {
		p.mu.RLock()
		otherPair.mu.RLock()
		defer p.mu.RUnlock()
		defer otherPair.mu.RUnlock()
		return p.car.Equal(otherPair.car) && p.cdr.Equal(otherPair.cdr)
	}
	return false
}

// IsVar always returns false for pairs.
func (p *Pair) IsVar() bool {
	return false
}

// Clone creates a deep copy of the pair.
func (p *Pair) Clone() Term {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return &Pair{car: p.car.Clone(), cdr: p.cdr.Clone()}
}

// Car returns the first element of the pair.
func (p *Pair) Car() Term {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.car
}

// Cdr returns the rest of the pair.
func (p *Pair) Cdr() Term {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.cdr
}

// Substitution represents a mapping from variables to terms.
// It's used to track bindings during unification and goal evaluation.
// The implementation is thread-safe and supports concurrent access.
type Substitution struct {
	bindings map[int64]Term // Maps variable IDs to terms
	mu       sync.RWMutex   // Protects concurrent access
}

// NewSubstitution creates an empty substitution.
func NewSubstitution() *Substitution {
	return &Substitution{
		bindings: make(map[int64]Term),
	}
}

// Clone creates a deep copy of the substitution.
func (s *Substitution) Clone() *Substitution {
	s.mu.RLock()
	defer s.mu.RUnlock()

	newBindings := make(map[int64]Term, len(s.bindings))
	for k, v := range s.bindings {
		newBindings[k] = v.Clone()
	}

	return &Substitution{bindings: newBindings}
}

// Lookup returns the term bound to a variable, or nil if unbound.
func (s *Substitution) Lookup(v *Var) Term {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.bindings[v.id]
}

// Bind creates a new substitution with an additional binding.
// Returns nil if the binding would create an inconsistency.
func (s *Substitution) Bind(v *Var, term Term) *Substitution {
	// Prevent binding a variable to itself
	if term.IsVar() && term.(*Var).id == v.id {
		return s
	}

	newSub := s.Clone()
	newSub.mu.Lock()
	defer newSub.mu.Unlock()

	newSub.bindings[v.id] = term
	return newSub
}

// Walk traverses a term following variable bindings in the substitution.
func (s *Substitution) Walk(term Term) Term {
	if !term.IsVar() {
		return term
	}

	v := term.(*Var)
	if bound := s.Lookup(v); bound != nil {
		return s.Walk(bound) // Follow the binding chain
	}

	return term // Unbound variable
}

// Size returns the number of bindings in the substitution.
func (s *Substitution) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.bindings)
}

// String returns a string representation of the substitution.
func (s *Substitution) String() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.bindings) == 0 {
		return "{}"
	}

	result := "{"
	first := true
	for id, term := range s.bindings {
		if !first {
			result += ", "
		}
		result += fmt.Sprintf("_%d=%s", id, term.String())
		first = false
	}
	result += "}"
	return result
}

// Stream represents a (potentially infinite) sequence of constraint stores.
// Streams are the core data structure for representing multiple solutions
// in miniKanren. Each constraint store contains variable bindings and
// active constraints representing a consistent logical state.
//
// This implementation uses channels for thread-safe concurrent access
// and supports parallel evaluation with proper constraint coordination.
type Stream struct {
	ch   chan ConstraintStore // Channel for streaming constraint stores
	done chan struct{}        // Channel to signal completion
	mu   sync.Mutex           // Protects stream state
}

// NewStream creates a new empty stream.
func NewStream() *Stream {
	return &Stream{
		ch:   make(chan ConstraintStore),
		done: make(chan struct{}),
	}
}

// Take retrieves up to n constraint stores from the stream.
// Returns a slice of constraint stores and a boolean indicating
// if more stores might be available.
func (s *Stream) Take(n int) ([]ConstraintStore, bool) {
	var results []ConstraintStore

	for i := 0; i < n; i++ {
		select {
		case store := <-s.ch:
			if store != nil {
				results = append(results, store)
			}
		case <-s.done:
			// Stream is closed, no more items will come
			return results, false
		}
	}

	// After successfully taking n items, check if stream is done
	// Use runtime.Gosched() to yield to other goroutines that might be closing the stream
	runtime.Gosched()

	select {
	case <-s.done:
		return results, false // Stream is closed
	default:
		return results, true // Stream is still open, might have more
	}
} // Put adds a constraint store to the stream.
func (s *Stream) Put(store ConstraintStore) {
	select {
	case s.ch <- store:
	case <-s.done:
		// Stream is closed
	}
}

// Close closes the stream, indicating no more substitutions will be added.
func (s *Stream) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-s.done:
		// Already closed
	default:
		close(s.done)
	}
}

// Goal represents a constraint or a combination of constraints.
// Goals are functions that take a constraint store and return a stream
// of constraint stores representing all possible ways to satisfy the goal.
// Goals can be composed to build complex relational programs.
//
// The constraint store contains both variable bindings and active constraints,
// enabling order-independent constraint logic programming.
type Goal func(ctx context.Context, store ConstraintStore) *Stream

// Success is a goal that always succeeds with the given constraint store.
var Success Goal = func(ctx context.Context, store ConstraintStore) *Stream {
	stream := NewStream()
	go func() {
		defer stream.Close()
		stream.Put(store)
	}()
	return stream
}

// Failure is a goal that always fails (returns no constraint stores).
var Failure Goal = func(ctx context.Context, store ConstraintStore) *Stream {
	stream := NewStream()
	stream.Close() // Immediately close to indicate no solutions
	return stream
}
