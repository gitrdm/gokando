package minikanren

import "fmt"

// NominalPlugin handles nominal logic constraints (freshness, alpha-equality) within the HybridSolver.
// Currently, it validates FreshnessConstraint instances against the UnifiedStore's relational bindings.
type NominalPlugin struct{}

// NewNominalPlugin creates a new nominal plugin instance.
func NewNominalPlugin() *NominalPlugin { return &NominalPlugin{} }

// Name implements SolverPlugin.
func (np *NominalPlugin) Name() string { return "Nominal" }

// CanHandle implements SolverPlugin. Returns true for nominal constraints we recognize.
func (np *NominalPlugin) CanHandle(constraint interface{}) bool {
	switch constraint.(type) {
	case *FreshnessConstraint:
		return true
	case *AlphaEqConstraint:
		return true
	default:
		return false
	}
}

// Propagate implements SolverPlugin. Validates nominal constraints; returns error on violation.
// Note: This plugin currently does not modify the UnifiedStore. Future enhancements may include
// alpha-equivalence-aware normalization and derived constraints.
func (np *NominalPlugin) Propagate(store *UnifiedStore) (*UnifiedStore, error) {
	bindings := store.getAllBindings()

	for _, c := range store.GetConstraints() {
		switch nc := c.(type) {
		case *FreshnessConstraint:
			switch nc.Check(bindings) {
			case ConstraintViolated:
				return nil, fmt.Errorf("nominal freshness violated: %s", nc.String())
			default:
				// satisfied or pending - ok
			}
		case *AlphaEqConstraint:
			switch nc.Check(bindings) {
			case ConstraintViolated:
				return nil, fmt.Errorf("alpha-equivalence violated: %s", nc.String())
			default:
				// satisfied or pending - ok
			}
		}
	}

	return store, nil
}
