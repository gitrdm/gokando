// Package minikanren provides constraint solving infrastructure.
// This file defines additional Solver API methods.
package minikanren

// Model returns the model being solved.
// The model is read-only during solving and safe for concurrent access
// by multiple solver instances.
func (s *Solver) Model() *Model {
	return s.model
}
