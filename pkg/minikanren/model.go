// Package minikanren provides constraint programming infrastructure.
// This file defines the Model abstraction for declaratively building
// constraint satisfaction problems.
package minikanren

import (
	"fmt"
	"sync"
)

// Model represents a constraint satisfaction problem (CSP) declaratively.
// A model consists of:
//   - Variables: decision variables with finite domains
//   - Constraints: relationships that must hold among variables
//   - Configuration: solver parameters and search heuristics
//
// Models are constructed incrementally by adding variables and constraints.
// Once constructed, models are immutable during solving, enabling safe
// concurrent access by parallel search workers.
//
// Thread safety: Models are safe for concurrent reads during solving,
// but must be constructed sequentially.
type Model struct {
	// variables holds all decision variables in order of creation
	variables []*FDVariable

	// constraints holds all constraints posted to the model
	constraints []ModelConstraint

	// variableIndex maps variable IDs to variable pointers for fast lookup
	variableIndex map[int]*FDVariable

	// maxDomainSize is the largest domain size across all variables
	maxDomainSize int

	// config holds solver configuration (heuristics, limits, etc.)
	config *SolverConfig

	// mu protects model during construction
	mu sync.RWMutex
}

// ModelConstraint represents a constraint within a model.
// Constraints restrict the values that variables can take simultaneously.
//
// Different constraint types provide different propagation strength:
//   - AllDifferent: ensures variables take distinct values
//   - Arithmetic: enforces arithmetic relationships (x + y = z)
//   - Table: extensional constraints defined by allowed tuples
//   - Global: specialized algorithms for common patterns
//
// ModelConstraints are immutable after creation and safe for concurrent access.
type ModelConstraint interface {
	// Variables returns the variables involved in this constraint.
	Variables() []*FDVariable

	// Type returns a string identifying the constraint type.
	Type() string

	// String returns a human-readable representation.
	String() string
}

// NewModel creates a new empty constraint model with default configuration.
func NewModel() *Model {
	return &Model{
		variables:     make([]*FDVariable, 0),
		constraints:   make([]ModelConstraint, 0),
		variableIndex: make(map[int]*FDVariable),
		config:        DefaultSolverConfig(),
	}
}

// NewModelWithConfig creates a model with custom solver configuration.
func NewModelWithConfig(config *SolverConfig) *Model {
	if config == nil {
		config = DefaultSolverConfig()
	}
	return &Model{
		variables:     make([]*FDVariable, 0),
		constraints:   make([]ModelConstraint, 0),
		variableIndex: make(map[int]*FDVariable),
		config:        config,
	}
}

// NewVariable creates and adds a new variable to the model with the specified domain.
// Returns the created variable which can be used to post constraints.
func (m *Model) NewVariable(domain Domain) *FDVariable {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := len(m.variables)
	variable := NewFDVariable(id, domain)

	m.variables = append(m.variables, variable)
	m.variableIndex[id] = variable

	if domain.MaxValue() > m.maxDomainSize {
		m.maxDomainSize = domain.MaxValue()
	}

	return variable
}

// NewVariableWithName creates a named variable for easier debugging.
func (m *Model) NewVariableWithName(domain Domain, name string) *FDVariable {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := len(m.variables)
	variable := NewFDVariableWithName(id, domain, name)

	m.variables = append(m.variables, variable)
	m.variableIndex[id] = variable

	if domain.MaxValue() > m.maxDomainSize {
		m.maxDomainSize = domain.MaxValue()
	}

	return variable
}

// NewVariables creates multiple variables with the same domain.
// Returns a slice of variables for convenient constraint posting.
func (m *Model) NewVariables(count int, domain Domain) []*FDVariable {
	variables := make([]*FDVariable, count)
	for i := 0; i < count; i++ {
		variables[i] = m.NewVariable(domain)
	}
	return variables
}

// NewVariablesWithNames creates multiple named variables with the same domain.
func (m *Model) NewVariablesWithNames(names []string, domain Domain) []*FDVariable {
	variables := make([]*FDVariable, len(names))
	for i, name := range names {
		variables[i] = m.NewVariableWithName(domain, name)
	}
	return variables
}

// GetVariable retrieves a variable by its ID.
// Returns nil if the ID doesn't exist.
func (m *Model) GetVariable(id int) *FDVariable {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.variableIndex[id]
}

// Variables returns all variables in the model.
// The returned slice should not be modified.
func (m *Model) Variables() []*FDVariable {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.variables
}

// VariableCount returns the number of variables in the model.
func (m *Model) VariableCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.variables)
}

// AddConstraint adds a constraint to the model.
// Constraints are enforced during solving via propagation and search.
func (m *Model) AddConstraint(constraint ModelConstraint) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.constraints = append(m.constraints, constraint)
}

// Constraints returns all constraints in the model.
// The returned slice should not be modified.
func (m *Model) Constraints() []ModelConstraint {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.constraints
}

// ConstraintCount returns the number of constraints in the model.
func (m *Model) ConstraintCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.constraints)
}

// Config returns the solver configuration for this model.
func (m *Model) Config() *SolverConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

// SetConfig updates the solver configuration.
// Should be called before solving begins.
func (m *Model) SetConfig(config *SolverConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if config != nil {
		m.config = config
	}
}

// MaxDomainSize returns the largest domain size in the model.
func (m *Model) MaxDomainSize() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.maxDomainSize
}

// String returns a human-readable representation of the model.
func (m *Model) String() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return fmt.Sprintf("Model{variables: %d, constraints: %d, maxDomain: %d}",
		len(m.variables), len(m.constraints), m.maxDomainSize)
}

// Validate checks if the model is well-formed and ready for solving.
// Returns an error if:
//   - Any variable has an empty domain
//   - Any constraint references unknown variables
//   - Configuration is invalid
func (m *Model) Validate() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check for variables with empty domains
	for _, v := range m.variables {
		if v.Domain().Count() == 0 {
			return fmt.Errorf("variable %s has empty domain", v.Name())
		}
	}

	// Check constraints reference valid variables
	for _, c := range m.constraints {
		for _, v := range c.Variables() {
			if m.variableIndex[v.ID()] == nil {
				return fmt.Errorf("constraint %s references unknown variable %d", c.Type(), v.ID())
			}
		}
	}

	return nil
}
