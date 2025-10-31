// Package minikanren provides concrete implementations of constraints
// for the hybrid constraint system. These constraints implement the
// Constraint interface and provide the core constraint logic for
// disequality, absence, type checking, and other relational operations.
//
// Each constraint implementation follows the same pattern:
//   - Efficient local checking when all variables are bound
//   - Graceful handling of unbound variables (returns ConstraintPending)
//   - Thread-safe operations for concurrent constraint checking
//   - Proper variable dependency tracking for optimization
//
// The constraint implementations are designed to be:
//   - Fast: Optimized for the common case of local constraint checking
//   - Safe: Thread-safe and defensive against malformed input
//   - Debuggable: Comprehensive error messages and string representations
package minikanren

import (
	"fmt"
	"reflect"
	"strings"
	"sync/atomic"
)

// constraintIDCounter provides unique IDs for constraint instances
var constraintIDCounter int64

// generateConstraintID creates a unique identifier for a constraint instance.
func generateConstraintID(constraintType string) string {
	id := atomic.AddInt64(&constraintIDCounter, 1)
	return fmt.Sprintf("%s-%d", constraintType, id)
}

// DisequalityConstraint implements the disequality constraint (≠).
// It ensures that two terms are not equal, providing order-independent
// constraint semantics for the Neq operation.
//
// The constraint tracks two terms and checks that they never become
// equal through unification. If both terms are variables, the constraint
// remains pending until at least one is bound to a concrete value.
type DisequalityConstraint struct {
	// id uniquely identifies this constraint instance
	id string

	// term1 and term2 are the terms that must not be equal
	term1, term2 Term

	// isLocal indicates whether this constraint can be checked locally
	isLocal bool
}

// NewDisequalityConstraint creates a new disequality constraint.
// The constraint is considered local if both terms are in the same
// constraint store context, enabling fast local checking.
func NewDisequalityConstraint(term1, term2 Term) *DisequalityConstraint {
	return &DisequalityConstraint{
		id:      generateConstraintID("neq"),
		term1:   term1,
		term2:   term2,
		isLocal: true, // Most constraints are local by default
	}
}

// ID returns the unique identifier for this constraint instance.
// Implements the Constraint interface.
func (dc *DisequalityConstraint) ID() string {
	return dc.id
}

// IsLocal returns true if this constraint can be evaluated purely
// within a local constraint store.
// Implements the Constraint interface.
func (dc *DisequalityConstraint) IsLocal() bool {
	return dc.isLocal
}

// Variables returns the logic variables that this constraint depends on.
// Used to determine when the constraint needs to be re-evaluated.
// Implements the Constraint interface.
func (dc *DisequalityConstraint) Variables() []*Var {
	var vars []*Var

	// Extract variables from term1
	vars = append(vars, extractVariables(dc.term1)...)

	// Extract variables from term2
	vars = append(vars, extractVariables(dc.term2)...)

	return vars
}

// Check evaluates the disequality constraint against current variable bindings.
// Returns ConstraintViolated if the terms are equal, ConstraintPending if
// variables are unbound, or ConstraintSatisfied if terms are provably unequal.
// Implements the Constraint interface.
func (dc *DisequalityConstraint) Check(bindings map[int64]Term) ConstraintResult {
	// Walk both terms to their final values
	val1 := walkTerm(dc.term1, bindings)
	val2 := walkTerm(dc.term2, bindings)

	// If either term is still a variable, constraint is pending
	if val1.IsVar() || val2.IsVar() {
		return ConstraintPending
	}

	// Both terms are concrete - check equality
	if val1.Equal(val2) {
		return ConstraintViolated // Terms are equal, constraint violated
	}

	return ConstraintSatisfied // Terms are different, constraint satisfied
}

// String returns a human-readable representation of the constraint.
// Implements the Constraint interface.
func (dc *DisequalityConstraint) String() string {
	return fmt.Sprintf("(%s ≠ %s)", dc.term1.String(), dc.term2.String())
}

// Clone creates a deep copy of the constraint for parallel execution.
// Implements the Constraint interface.
func (dc *DisequalityConstraint) Clone() Constraint {
	return &DisequalityConstraint{
		id:      dc.id, // Keep same ID for tracking
		term1:   dc.term1.Clone(),
		term2:   dc.term2.Clone(),
		isLocal: dc.isLocal,
	}
}

// AbsenceConstraint implements the absence constraint (absento).
// It ensures that a specific term does not occur anywhere within
// another term's structure, providing structural constraint checking.
//
// This constraint performs recursive structural inspection to detect
// the presence of the forbidden term at any level of nesting.
type AbsenceConstraint struct {
	// id uniquely identifies this constraint instance
	id string

	// absent is the term that must not occur
	absent Term

	// container is the term that must not contain the absent term
	container Term

	// isLocal indicates whether this constraint can be checked locally
	isLocal bool
}

// NewAbsenceConstraint creates a new absence constraint.
func NewAbsenceConstraint(absent, container Term) *AbsenceConstraint {
	return &AbsenceConstraint{
		id:        generateConstraintID("absento"),
		absent:    absent,
		container: container,
		isLocal:   true,
	}
}

// ID returns the unique identifier for this constraint instance.
// Implements the Constraint interface.
func (ac *AbsenceConstraint) ID() string {
	return ac.id
}

// IsLocal returns true if this constraint can be evaluated locally.
// Implements the Constraint interface.
func (ac *AbsenceConstraint) IsLocal() bool {
	return ac.isLocal
}

// Variables returns the logic variables this constraint depends on.
// Implements the Constraint interface.
func (ac *AbsenceConstraint) Variables() []*Var {
	var vars []*Var
	vars = append(vars, extractVariables(ac.absent)...)
	vars = append(vars, extractVariables(ac.container)...)
	return vars
}

// Check evaluates the absence constraint against current bindings.
// Returns ConstraintViolated if the absent term is found in the container,
// ConstraintPending if variables are unbound, or ConstraintSatisfied otherwise.
// Implements the Constraint interface.
func (ac *AbsenceConstraint) Check(bindings map[int64]Term) ConstraintResult {
	absentVal := walkTerm(ac.absent, bindings)
	containerVal := walkTerm(ac.container, bindings)

	// If either term contains unbound variables, constraint is pending
	if hasUnboundVariables(absentVal) || hasUnboundVariables(containerVal) {
		return ConstraintPending
	}

	// Check if absent term occurs anywhere in container structure
	if occurs(absentVal, containerVal) {
		return ConstraintViolated
	}

	return ConstraintSatisfied
}

// String returns a human-readable representation of the constraint.
// Implements the Constraint interface.
func (ac *AbsenceConstraint) String() string {
	return fmt.Sprintf("(absento %s %s)", ac.absent.String(), ac.container.String())
}

// Clone creates a deep copy of the constraint for parallel execution.
// Implements the Constraint interface.
func (ac *AbsenceConstraint) Clone() Constraint {
	return &AbsenceConstraint{
		id:        ac.id,
		absent:    ac.absent.Clone(),
		container: ac.container.Clone(),
		isLocal:   ac.isLocal,
	}
}

// TypeConstraint implements type-based constraints (symbolo, numbero, etc.).
// It ensures that a term has a specific type, enabling type-safe
// relational programming patterns.
type TypeConstraint struct {
	// id uniquely identifies this constraint instance
	id string

	// term is the term that must have the specified type
	term Term

	// expectedType specifies what type the term must have
	expectedType TypeConstraintKind

	// isLocal indicates whether this constraint can be checked locally
	isLocal bool
}

// TypeConstraintKind represents the different types that can be constrained.
type TypeConstraintKind int

const (
	// SymbolType requires the term to be a string atom
	SymbolType TypeConstraintKind = iota

	// NumberType requires the term to be a numeric atom
	NumberType

	// PairType requires the term to be a pair (non-empty list)
	PairType

	// NullType requires the term to be the empty list (nil)
	NullType
)

// String returns a human-readable representation of the type constraint kind.
func (tck TypeConstraintKind) String() string {
	switch tck {
	case SymbolType:
		return "symbol"
	case NumberType:
		return "number"
	case PairType:
		return "pair"
	case NullType:
		return "null"
	default:
		return "unknown"
	}
}

// NewTypeConstraint creates a new type constraint.
func NewTypeConstraint(term Term, expectedType TypeConstraintKind) *TypeConstraint {
	constraintName := fmt.Sprintf("%so", expectedType.String())
	return &TypeConstraint{
		id:           generateConstraintID(constraintName),
		term:         term,
		expectedType: expectedType,
		isLocal:      true,
	}
}

// ID returns the unique identifier for this constraint instance.
// Implements the Constraint interface.
func (tc *TypeConstraint) ID() string {
	return tc.id
}

// IsLocal returns true if this constraint can be evaluated locally.
// Implements the Constraint interface.
func (tc *TypeConstraint) IsLocal() bool {
	return tc.isLocal
}

// Variables returns the logic variables this constraint depends on.
// Implements the Constraint interface.
func (tc *TypeConstraint) Variables() []*Var {
	return extractVariables(tc.term)
}

// Check evaluates the type constraint against current bindings.
// Returns ConstraintViolated if the term has the wrong type,
// ConstraintPending if the term is unbound, or ConstraintSatisfied
// if the term has the correct type.
// Implements the Constraint interface.
func (tc *TypeConstraint) Check(bindings map[int64]Term) ConstraintResult {
	termVal := walkTerm(tc.term, bindings)

	// If term is still a variable, constraint is pending
	if termVal.IsVar() {
		return ConstraintPending
	}

	// Check if term has the expected type
	if tc.hasExpectedType(termVal) {
		return ConstraintSatisfied
	}

	return ConstraintViolated
}

// hasExpectedType checks if a term has the type expected by this constraint.
func (tc *TypeConstraint) hasExpectedType(term Term) bool {
	switch tc.expectedType {
	case SymbolType:
		if atom, ok := term.(*Atom); ok {
			_, isString := atom.Value().(string)
			return isString
		}
		return false

	case NumberType:
		if atom, ok := term.(*Atom); ok {
			val := atom.Value()
			rv := reflect.ValueOf(val)
			return rv.Kind() >= reflect.Int && rv.Kind() <= reflect.Complex128
		}
		return false

	case PairType:
		_, isPair := term.(*Pair)
		return isPair

	case NullType:
		// Check if term is the empty list (nil)
		if atom, ok := term.(*Atom); ok {
			return atom.Value() == nil
		}
		return term == Nil

	default:
		return false
	}
}

// String returns a human-readable representation of the constraint.
// Implements the Constraint interface.
func (tc *TypeConstraint) String() string {
	return fmt.Sprintf("(%so %s)", tc.expectedType.String(), tc.term.String())
}

// Clone creates a deep copy of the constraint for parallel execution.
// Implements the Constraint interface.
func (tc *TypeConstraint) Clone() Constraint {
	return &TypeConstraint{
		id:           tc.id,
		term:         tc.term.Clone(),
		expectedType: tc.expectedType,
		isLocal:      tc.isLocal,
	}
}

// Helper functions for constraint implementation

// extractVariables recursively extracts all variables from a term.
func extractVariables(term Term) []*Var {
	var vars []*Var

	switch t := term.(type) {
	case *Var:
		vars = append(vars, t)
	case *Pair:
		vars = append(vars, extractVariables(t.Car())...)
		vars = append(vars, extractVariables(t.Cdr())...)
	case *Atom:
		// Atoms contain no variables
	}

	return vars
}

// walkTerm follows variable bindings to find the final value of a term.
func walkTerm(term Term, bindings map[int64]Term) Term {
	if variable, ok := term.(*Var); ok {
		if binding, exists := bindings[variable.id]; exists {
			// Recursively walk the binding in case it's also a variable
			return walkTerm(binding, bindings)
		}
		// Variable is unbound
		return variable
	}
	return term
}

// hasUnboundVariables checks if a term contains any unbound variables.
func hasUnboundVariables(term Term) bool {
	switch t := term.(type) {
	case *Var:
		return true // Unbound variable
	case *Pair:
		return hasUnboundVariables(t.Car()) || hasUnboundVariables(t.Cdr())
	case *Atom:
		return false // Atoms have no variables
	default:
		return false
	}
}

// MembershipConstraint implements the membership constraint (membero).
// It ensures that an element is a member of a list, providing relational
// list membership checking that can work in both directions.
type MembershipConstraint struct {
	// id uniquely identifies this constraint instance
	id string

	// element is the term that should be a member of the list
	element Term

	// list is the list that should contain the element
	list Term

	// isLocal indicates whether this constraint can be checked locally
	isLocal bool
}

// NewMembershipConstraint creates a new membership constraint.
func NewMembershipConstraint(element, list Term) *MembershipConstraint {
	return &MembershipConstraint{
		id:      generateConstraintID("membero"),
		element: element,
		list:    list,
		isLocal: true,
	}
}

// ID returns the unique identifier for this constraint instance.
// Implements the Constraint interface.
func (mc *MembershipConstraint) ID() string {
	return mc.id
}

// IsLocal returns true if this constraint can be evaluated locally.
// Implements the Constraint interface.
func (mc *MembershipConstraint) IsLocal() bool {
	return mc.isLocal
}

// Variables returns the logic variables this constraint depends on.
// Implements the Constraint interface.
func (mc *MembershipConstraint) Variables() []*Var {
	var vars []*Var
	vars = append(vars, extractVariables(mc.element)...)
	vars = append(vars, extractVariables(mc.list)...)
	return vars
}

// Check evaluates the membership constraint against current bindings.
// Note: This is a simplified implementation. The full membero relation
// is typically implemented as a recursive goal rather than a simple constraint.
// Implements the Constraint interface.
func (mc *MembershipConstraint) Check(bindings map[int64]Term) ConstraintResult {
	elementVal := walkTerm(mc.element, bindings)
	listVal := walkTerm(mc.list, bindings)

	// If either term contains unbound variables, constraint is pending
	if hasUnboundVariables(elementVal) || hasUnboundVariables(listVal) {
		return ConstraintPending
	}

	// Check if element is a member of the list
	if isMember(elementVal, listVal) {
		return ConstraintSatisfied
	}

	return ConstraintViolated
}

// isMember checks if an element is a member of a list structure.
func isMember(element, list Term) bool {
	switch l := list.(type) {
	case *Pair:
		// Check if element equals the car, or is a member of the cdr
		if element.Equal(l.Car()) {
			return true
		}
		return isMember(element, l.Cdr())
	case *Atom:
		// Check if this is the empty list
		return l.Value() == nil && element.Equal(Nil)
	default:
		return false
	}
}

// String returns a human-readable representation of the constraint.
// Implements the Constraint interface.
func (mc *MembershipConstraint) String() string {
	return fmt.Sprintf("(membero %s %s)", mc.element.String(), mc.list.String())
}

// Clone creates a deep copy of the constraint for parallel execution.
// Implements the Constraint interface.
func (mc *MembershipConstraint) Clone() Constraint {
	return &MembershipConstraint{
		id:      mc.id,
		element: mc.element.Clone(),
		list:    mc.list.Clone(),
		isLocal: mc.isLocal,
	}
}

// ConstraintViolationError represents an error caused by constraint violations.
// It provides detailed information about which constraint was violated and why.
type ConstraintViolationError struct {
	Constraint Constraint
	Bindings   map[int64]Term
	Message    string
}

// Error returns a detailed error message about the constraint violation.
func (cve *ConstraintViolationError) Error() string {
	var bindingStrs []string
	for varID, term := range cve.Bindings {
		bindingStrs = append(bindingStrs, fmt.Sprintf("var_%d = %s", varID, term.String()))
	}

	return fmt.Sprintf("constraint violation: %s failed with bindings [%s]: %s",
		cve.Constraint.String(), strings.Join(bindingStrs, ", "), cve.Message)
}

// NewConstraintViolationError creates a new constraint violation error.
func NewConstraintViolationError(constraint Constraint, bindings map[int64]Term, message string) *ConstraintViolationError {
	// Copy bindings to prevent modification
	bindingsCopy := make(map[int64]Term, len(bindings))
	for id, term := range bindings {
		bindingsCopy[id] = term
	}

	return &ConstraintViolationError{
		Constraint: constraint,
		Bindings:   bindingsCopy,
		Message:    message,
	}
}

// ConstraintBuilder provides a fluent interface for building constraints.
// It allows chaining constraint creation with a more declarative syntax.
//
// Example:
//
//	builder := NewConstraintBuilder().
//		WithDisequality(x, y).
//		WithType(z, SymbolType).
//		WithAbsence(NewAtom("bad"), x)
//	store := builder.Build()
type ConstraintBuilder struct {
	constraints []Constraint
}

// NewConstraintBuilder creates a new constraint builder.
func NewConstraintBuilder() *ConstraintBuilder {
	return &ConstraintBuilder{
		constraints: make([]Constraint, 0),
	}
}

// WithDisequality adds a disequality constraint between two terms.
func (cb *ConstraintBuilder) WithDisequality(term1, term2 Term) *ConstraintBuilder {
	cb.constraints = append(cb.constraints, NewDisequalityConstraint(term1, term2))
	return cb
}

// WithAbsence adds an absence constraint ensuring absent term doesn't occur in container.
func (cb *ConstraintBuilder) WithAbsence(absent, container Term) *ConstraintBuilder {
	cb.constraints = append(cb.constraints, NewAbsenceConstraint(absent, container))
	return cb
}

// WithType adds a type constraint ensuring a term has the specified type.
func (cb *ConstraintBuilder) WithType(term Term, expectedType TypeConstraintKind) *ConstraintBuilder {
	cb.constraints = append(cb.constraints, NewTypeConstraint(term, expectedType))
	return cb
}

// WithMembership adds a membership constraint ensuring an element is in a list.
func (cb *ConstraintBuilder) WithMembership(element, list Term) *ConstraintBuilder {
	cb.constraints = append(cb.constraints, NewMembershipConstraint(element, list))
	return cb
}

// Build creates a constraint store containing all the built constraints.
// This method now uses deferred constraint checking to prevent immediate
// failures that can cause infinite loops in goal compositions.
//
// Instead of failing immediately when constraints are incompatible,
// the store is created successfully and constraint violations are
// detected later during goal execution when more context is available.
func (cb *ConstraintBuilder) Build() ConstraintStore {
	store := EmptyStore()
	for _, constraint := range cb.constraints {
		var err error
		// Use deferred checking to prevent immediate failures
		store, err = StoreWithConstraintDeferred(store, constraint)
		if err != nil {
			// In the deferred approach, we don't panic on constraint addition failures.
			// Instead, we continue building the store. Constraint violations will be
			// caught later during goal execution when bindings are applied.
			//
			// This prevents the "immediate failure trap" where constraint builders
			// could cause infinite loops if users don't handle errors properly.
			continue
		}
	}
	return store
}

// BuildConstraints returns the list of constraints without building a store.
func (cb *ConstraintBuilder) BuildConstraints() []Constraint {
	constraints := make([]Constraint, len(cb.constraints))
	copy(constraints, cb.constraints)
	return constraints
}

// Declarative constraint constructors with builder patterns

// Disequality creates a disequality constraint using a builder pattern.
// This provides a more declarative way to create constraints.
//
// Example:
//
//	constraint := Disequality(x).NotEqualTo(y)
func Disequality(term Term) *DisequalityBuilder {
	return &DisequalityBuilder{term1: term}
}

// DisequalityBuilder provides a fluent interface for building disequality constraints.
type DisequalityBuilder struct {
	term1 Term
}

// NotEqualTo completes the disequality constraint.
func (db *DisequalityBuilder) NotEqualTo(term2 Term) *DisequalityConstraint {
	return NewDisequalityConstraint(db.term1, term2)
}

// Absence creates an absence constraint using a builder pattern.
//
// Example:
//
//	constraint := Absence(forbiddenTerm).NotIn(containerTerm)
func Absence(absent Term) *AbsenceBuilder {
	return &AbsenceBuilder{absent: absent}
}

// AbsenceBuilder provides a fluent interface for building absence constraints.
type AbsenceBuilder struct {
	absent Term
}

// NotIn completes the absence constraint.
func (ab *AbsenceBuilder) NotIn(container Term) *AbsenceConstraint {
	return NewAbsenceConstraint(ab.absent, container)
}

// Type creates a type constraint using a builder pattern.
//
// Example:
//
//	constraint := Type(variable).MustBe(SymbolType)
func Type(term Term) *TypeBuilder {
	return &TypeBuilder{term: term}
}

// TypeBuilder provides a fluent interface for building type constraints.
type TypeBuilder struct {
	term Term
}

// MustBe completes the type constraint.
func (tb *TypeBuilder) MustBe(expectedType TypeConstraintKind) *TypeConstraint {
	return NewTypeConstraint(tb.term, expectedType)
}

// Membership creates a membership constraint using a builder pattern.
//
// Example:
//
//	constraint := Membership(element).In(list)
func Membership(element Term) *MembershipBuilder {
	return &MembershipBuilder{element: element}
}

// MembershipBuilder provides a fluent interface for building membership constraints.
type MembershipBuilder struct {
	element Term
}

// In completes the membership constraint.
func (mb *MembershipBuilder) In(list Term) *MembershipConstraint {
	return NewMembershipConstraint(mb.element, list)
}
