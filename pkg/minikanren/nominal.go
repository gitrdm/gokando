// Package minikanren provides nominal logic support for reasoning about names and binding.
// Nominal logic extends miniKanren with the ability to handle fresh names, binding,
// and scope, enabling reasoning about programming language constructs and alpha-equivalence.
//
// Key concepts in nominal logic:
//   - Names: Unique identifiers that can be bound to values
//   - Freshness: Ensuring names don't conflict with existing bindings
//   - Binding: Associating names with values in lexical scopes
//   - Alpha-equivalence: Terms that differ only in bound variable names are equivalent
//
// This implementation provides:
//   - Name terms for representing unique identifiers
//   - Nominal unification with freshness constraints
//   - Name binding and scoping operations
//   - Integration with the constraint system for order-independent reasoning
package minikanren

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// Name represents a nominal name (unique identifier) in nominal logic.
// Names are atomic and can be bound to values, but have special unification rules
// that respect freshness and binding constraints. Unlike regular atoms, names
// participate in nominal unification where alpha-equivalent terms are considered equal.
type Name struct {
	id  int64        // Unique identifier for the name
	sym string       // Optional symbolic name for debugging
	mu  sync.RWMutex // Protects concurrent access
}

// nameCounter provides thread-safe generation of unique name IDs
var nameCounter int64

// NewName creates a new unique name with an optional symbolic identifier.
// Each call generates a globally unique name that cannot conflict with existing names.
//
// Example:
//
//	name1 := NewName("x")  // Creates a name with symbol "x"
//	name2 := NewName("")   // Creates an anonymous name
func NewName(sym string) *Name {
	id := atomic.AddInt64(&nameCounter, 1)
	return &Name{id: id, sym: sym}
}

// String returns a string representation of the name.
func (n *Name) String() string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	if n.sym != "" {
		return fmt.Sprintf("'%s_%d", n.sym, n.id)
	}
	return fmt.Sprintf("'%d", n.id)
}

// Equal checks if two names are the same name (identical IDs).
// In nominal logic, names are equal only if they have the same unique identifier.
func (n *Name) Equal(other Term) bool {
	if otherName, ok := other.(*Name); ok {
		n.mu.RLock()
		otherName.mu.RLock()
		defer n.mu.RUnlock()
		defer otherName.mu.RUnlock()
		return n.id == otherName.id
	}
	return false
}

// IsVar returns false for names (names are not logic variables).
func (n *Name) IsVar() bool {
	return false
}

// Clone creates a copy of the name with the same identity.
func (n *Name) Clone() Term {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return &Name{id: n.id, sym: n.sym}
}

// ID returns the unique identifier of the name.
func (n *Name) ID() int64 {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.id
}

// Symbol returns the symbolic name of the name (may be empty).
func (n *Name) Symbol() string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.sym
}

// NominalBinding represents a binding of a name to a term within a scope.
// This is used to track name bindings in nominal unification and constraint solving.
// Bindings form the foundation of lexical scoping in nominal logic.
type NominalBinding struct {
	name *Name        // The name being bound
	term Term         // The term the name is bound to
	mu   sync.RWMutex // Protects concurrent access
}

// NewNominalBinding creates a new name binding.
func NewNominalBinding(name *Name, term Term) *NominalBinding {
	return &NominalBinding{name: name, term: term}
}

// String returns a string representation of the binding.
func (nb *NominalBinding) String() string {
	nb.mu.RLock()
	defer nb.mu.RUnlock()
	return fmt.Sprintf("%s ↦ %s", nb.name.String(), nb.term.String())
}

// Clone creates a deep copy of the binding.
func (nb *NominalBinding) Clone() *NominalBinding {
	nb.mu.RLock()
	defer nb.mu.RUnlock()
	return &NominalBinding{
		name: nb.name.Clone().(*Name),
		term: nb.term.Clone(),
	}
}

// Name returns the bound name.
func (nb *NominalBinding) Name() *Name {
	nb.mu.RLock()
	defer nb.mu.RUnlock()
	return nb.name
}

// Term returns the term the name is bound to.
func (nb *NominalBinding) Term() Term {
	nb.mu.RLock()
	defer nb.mu.RUnlock()
	return nb.term
}

// NominalScope represents a lexical scope containing name bindings.
// Scopes can be nested, and name lookups respect lexical scoping rules.
// This enables proper handling of nested bindings and shadowing.
type NominalScope struct {
	bindings map[int64]Term // Maps name IDs to bound terms
	parent   *NominalScope  // Parent scope for lexical lookup
	mu       sync.RWMutex   // Protects concurrent access
}

// NewNominalScope creates a new empty scope with no parent.
func NewNominalScope() *NominalScope {
	return &NominalScope{
		bindings: make(map[int64]Term),
		parent:   nil,
	}
}

// NewNominalScopeWithParent creates a new scope with a parent scope.
func NewNominalScopeWithParent(parent *NominalScope) *NominalScope {
	return &NominalScope{
		bindings: make(map[int64]Term),
		parent:   parent,
	}
}

// Bind adds a binding to this scope. If the name is already bound in this scope,
// the binding is updated. This represents lexical shadowing.
func (ns *NominalScope) Bind(name *Name, term Term) {
	ns.mu.Lock()
	defer ns.mu.Unlock()
	ns.bindings[name.ID()] = term
}

// Lookup finds the term bound to a name, searching from this scope outward.
// Returns nil if the name is not bound in this scope chain.
func (ns *NominalScope) Lookup(name *Name) Term {
	ns.mu.RLock()
	defer ns.mu.RUnlock()

	// Check this scope first
	if term, exists := ns.bindings[name.ID()]; exists {
		return term
	}

	// Check parent scopes
	if ns.parent != nil {
		return ns.parent.Lookup(name)
	}

	return nil // Not found
}

// IsBound checks if a name is bound in this scope chain.
func (ns *NominalScope) IsBound(name *Name) bool {
	return ns.Lookup(name) != nil
}

// Clone creates a deep copy of the scope and its bindings.
func (ns *NominalScope) Clone() *NominalScope {
	ns.mu.RLock()
	defer ns.mu.RUnlock()

	newBindings := make(map[int64]Term, len(ns.bindings))
	for id, term := range ns.bindings {
		newBindings[id] = term.Clone()
	}

	var newParent *NominalScope
	if ns.parent != nil {
		newParent = ns.parent.Clone()
	}

	return &NominalScope{
		bindings: newBindings,
		parent:   newParent,
	}
}

// String returns a string representation of the scope.
func (ns *NominalScope) String() string {
	ns.mu.RLock()
	defer ns.mu.RUnlock()

	if len(ns.bindings) == 0 {
		if ns.parent == nil {
			return "{}"
		}
		return fmt.Sprintf("{parent: %s}", ns.parent.String())
	}

	result := "{"
	first := true
	for id, term := range ns.bindings {
		if !first {
			result += ", "
		}
		result += fmt.Sprintf("'%d ↦ %s", id, term.String())
		first = false
	}
	if ns.parent != nil {
		result += fmt.Sprintf(", parent: %s", ns.parent.String())
	}
	result += "}"
	return result
}

// Size returns the number of bindings in this scope (not including parent scopes).
func (ns *NominalScope) Size() int {
	ns.mu.RLock()
	defer ns.mu.RUnlock()
	return len(ns.bindings)
}
