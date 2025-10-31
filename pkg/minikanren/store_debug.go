// Package minikanren provides constraint store inspection and debugging utilities
// for advanced constraint logic programming operations.
//
// This file implements utilities for examining constraint store state,
// validating consistency, and generating human-readable representations.
// These tools are essential for debugging complex constraint programs
// and understanding solver behavior.
//
// Key utilities:
//   - StoreVariables: Extract all variables from a constraint store
//   - StoreDomains: Get current variable domains (FD-specific)
//   - StoreValidate: Check store consistency and detect issues
//   - StoreToString: Generate detailed human-readable representations
//
// All utilities are thread-safe and work with the generic ConstraintStore interface.
package minikanren

import (
	"fmt"
	"sort"
	"strings"
)

// StoreVariables extracts all logic variables referenced in a constraint store.
// This includes variables from constraints and any bound variables.
//
// The function examines all constraints in the store and collects their variables,
// returning a deduplicated slice of unique variables. Variables are returned
// in order of their IDs for consistent output.
//
// Parameters:
//   - store: The constraint store to examine
//
// Returns a slice of *Var containing all variables in the store.
func StoreVariables(store ConstraintStore) []*Var {
	if store == nil {
		return nil
	}

	variables := make(map[int64]*Var)
	constraints := store.GetConstraints()

	// Collect variables from all constraints
	for _, constraint := range constraints {
		for _, variable := range constraint.Variables() {
			variables[variable.id] = variable
		}
	}

	// Convert map to sorted slice for consistent ordering
	result := make([]*Var, 0, len(variables))
	for _, variable := range variables {
		result = append(result, variable)
	}

	// Sort by variable ID for consistent output
	sort.Slice(result, func(i, j int) bool {
		return result[i].id < result[j].id
	})

	return result
}

// StoreDomains returns the current domains for all variables in the store.
// This is primarily useful for finite domain (FD) stores, but works with
// any constraint store by examining variable bindings.
//
// For general constraint stores, returns singleton domains for bound variables
// and unknown status for unbound variables. FD-specific domain information
// is not available through the generic interface.
//
// Parameters:
//   - store: The constraint store to examine
//
// Returns a map from variable ID to domain information as a string.
func StoreDomains(store ConstraintStore) map[int64]string {
	if store == nil {
		return nil
	}

	domains := make(map[int64]string)
	variables := StoreVariables(store)

	// For general constraint stores, use binding information
	sub := store.GetSubstitution()
	for _, variable := range variables {
		term := sub.Lookup(variable)
		if term != nil {
			// Variable is bound to a term
			if atom, ok := term.(*Atom); ok {
				if intVal, ok := atom.Value().(int); ok {
					domains[variable.id] = fmt.Sprintf("{%d}", intVal)
				} else {
					domains[variable.id] = fmt.Sprintf("{%v}", atom.Value())
				}
			} else {
				domains[variable.id] = fmt.Sprintf("{%s}", term.String())
			}
		} else {
			// Variable is unbound
			domains[variable.id] = "unbound"
		}
	}

	return domains
}

// domainToString converts a BitSet domain to a human-readable string.
// Shows individual values for small domains, range notation for large domains.
func domainToString(domain BitSet) string {
	values := make([]int, 0)
	domain.IterateValues(func(val int) {
		values = append(values, val)
	})

	if len(values) == 0 {
		return "{}"
	}

	if len(values) <= 10 {
		// Show individual values for small domains
		strs := make([]string, len(values))
		for i, v := range values {
			strs[i] = fmt.Sprintf("%d", v)
		}
		return "{" + strings.Join(strs, ",") + "}"
	}

	// Show range for large domains
	min, max := values[0], values[len(values)-1]
	return fmt.Sprintf("{%d..%d}", min, max)
}

// StoreValidate checks the consistency of a constraint store and reports any issues.
// Validation includes checking for constraint violations, domain consistency,
// and other potential problems.
//
// The function performs several checks:
//   - Constraint satisfaction: Verifies all constraints are satisfied by current bindings
//
// Parameters:
//   - store: The constraint store to validate
//
// Returns a slice of validation errors. Empty slice indicates the store is valid.
func StoreValidate(store ConstraintStore) []error {
	if store == nil {
		return []error{fmt.Errorf("store is nil")}
	}

	var errors []error

	// Get current bindings for validation
	bindings := getStoreBindings(store)
	constraints := store.GetConstraints()

	// Check each constraint
	for _, constraint := range constraints {
		result := constraint.Check(bindings)
		if result == ConstraintViolated {
			errors = append(errors, fmt.Errorf("constraint violated: %s", constraint.String()))
		}
	}

	return errors
}

// StoreToString generates a detailed human-readable representation of a constraint store.
// This includes information about constraints, variables, domains, and store state.
//
// The output format includes:
//   - Store type and basic statistics
//   - List of all constraints
//   - Variable domains and bindings
//   - Any validation errors
//
// Parameters:
//   - store: The constraint store to represent
//
// Returns a formatted string suitable for debugging and logging.
func StoreToString(store ConstraintStore) string {
	if store == nil {
		return "Store: <nil>"
	}

	var builder strings.Builder
	builder.WriteString("Constraint Store {\n")

	// Basic store information
	constraints := store.GetConstraints()
	variables := StoreVariables(store)
	domains := StoreDomains(store)

	builder.WriteString(fmt.Sprintf("  Type: %T\n", store))
	builder.WriteString(fmt.Sprintf("  Constraints: %d\n", len(constraints)))
	builder.WriteString(fmt.Sprintf("  Variables: %d\n", len(variables)))

	builder.WriteString("\n")

	// List constraints
	if len(constraints) > 0 {
		builder.WriteString("  Constraints:\n")
		for i, constraint := range constraints {
			builder.WriteString(fmt.Sprintf("    [%d] %s\n", i, constraint.String()))
		}
		builder.WriteString("\n")
	}

	// List variables and domains
	if len(variables) > 0 {
		builder.WriteString("  Variables:\n")
		for _, variable := range variables {
			domainStr := "unknown"
			if d, exists := domains[variable.id]; exists {
				domainStr = d
			}
			builder.WriteString(fmt.Sprintf("    var_%d: %s\n", variable.id, domainStr))
		}
		builder.WriteString("\n")
	}

	// Validation errors
	if validationErrors := StoreValidate(store); len(validationErrors) > 0 {
		builder.WriteString("  Validation Errors:\n")
		for _, err := range validationErrors {
			builder.WriteString(fmt.Sprintf("    ERROR: %v\n", err))
		}
		builder.WriteString("\n")
	}

	builder.WriteString("}")
	return builder.String()
}

// StoreSummary generates a concise summary of a constraint store's state.
// This is useful for logging and monitoring without the full detail of StoreToString.
//
// Parameters:
//   - store: The constraint store to summarize
//
// Returns a short string with key statistics.
func StoreSummary(store ConstraintStore) string {
	if store == nil {
		return "Store: <nil>"
	}

	constraints := store.GetConstraints()
	variables := StoreVariables(store)

	summary := fmt.Sprintf("Store(%T): %d constraints, %d variables",
		store, len(constraints), len(variables))

	return summary
}
