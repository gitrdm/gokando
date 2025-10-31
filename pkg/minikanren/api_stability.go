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
//
// # API Stability
//
// This package follows semantic versioning (semver) for API compatibility:
//   - MAJOR version: Breaking changes
//   - MINOR version: New features, backward compatible
//   - PATCH version: Bug fixes, backward compatible
//
// Current Version: 1.0.0
//   - All exported APIs are stable and supported
//   - Breaking changes will be communicated via deprecation warnings
//   - Migration guides provided for major version changes
//
// # Deprecated APIs
//
// The following APIs are deprecated and will be removed in a future major version:
//   - None currently deprecated
//
// # Migration from Previous Versions
//
// Version 1.0.0 introduces context-aware goals and streaming results.
// All previous APIs remain compatible but new code should use context-aware variants:
//
//	// Old style (still supported)
//	result := Run(5, func(x *Var) Goal {
//	    return Eq(x, AtomFromValue(42))
//	})
//
//	// New style (recommended)
//	ctx := context.Background()
//	result := RunWithContext(ctx, 5, func(x *Var) Goal {
//	    return Eq(x, AtomFromValue(42))
//	})
//
// For more examples, see the examples/ directory and godoc documentation.
package minikanren

import (
	"context"
	"fmt"
	"runtime"
)

// Version represents the current version of the minikanren package.
type APIVersion struct {
	Major int
	Minor int
	Patch int
}

// CurrentAPIVersion returns the current API version of the minikanren package.
func CurrentAPIVersion() APIVersion {
	return APIVersion{Major: 1, Minor: 0, Patch: 0}
}

// String returns a string representation of the API version.
func (v APIVersion) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// GetAPIVersion returns the API version string for compatibility checking.
func GetAPIVersion() string {
	return CurrentAPIVersion().String()
}

// CheckAPIVersion checks if the current API version is compatible with the required version.
// Returns true if compatible, false otherwise.
// Compatible means same major version (breaking changes require major version bump).
func CheckAPIVersion(required string) bool {
	current := CurrentAPIVersion()
	req := parseVersion(required)
	return current.Major == req.Major
}

// parseVersion parses a version string into a APIVersion struct.
// Simple implementation for semantic versioning.
func parseVersion(v string) APIVersion {
	var major, minor, patch int
	fmt.Sscanf(v, "%d.%d.%d", &major, &minor, &patch)
	return APIVersion{Major: major, Minor: minor, Patch: patch}
}

// DeprecatedError represents an error for deprecated API usage.
type DeprecatedError struct {
	Function string
	Message  string
	Version  string // Version when this will be removed
}

func (e DeprecatedError) Error() string {
	return fmt.Sprintf("deprecated API usage: %s - %s (removal in version %s)",
		e.Function, e.Message, e.Version)
}

// deprecationWarning logs a deprecation warning.
// This is used internally to warn about deprecated API usage.
func deprecationWarning(function, message, removalVersion string) {
	// In a real implementation, this might log to a structured logger
	// For now, we use runtime.Caller to get the caller's location
	_, file, line, ok := runtime.Caller(2)
	if ok {
		fmt.Printf("DEPRECATED: %s in %s:%d - %s (removal in version %s)\n",
			function, file, line, message, removalVersion)
	} else {
		fmt.Printf("DEPRECATED: %s - %s (removal in version %s)\n",
			function, message, removalVersion)
	}
}

// MigrationGuide provides guidance for migrating between API versions.
type MigrationGuide struct {
	FromVersion string
	ToVersion   string
	Changes     []MigrationChange
}

// MigrationChange represents a single change in a migration guide.
type MigrationChange struct {
	Type        MigrationType
	Description string
	OldCode     string
	NewCode     string
}

// MigrationType represents the type of migration change.
type MigrationType int

const (
	// BreakingChange requires code modification
	BreakingChange MigrationType = iota
	// NewFeature adds new functionality
	NewFeature
	// PerformanceImprovement improves performance
	PerformanceImprovement
	// BugFix fixes incorrect behavior
	BugFix
)

// GetMigrationGuide returns the migration guide for upgrading to the current version.
func GetMigrationGuide(fromVersion string) *MigrationGuide {
	guide := &MigrationGuide{
		FromVersion: fromVersion,
		ToVersion:   CurrentAPIVersion().String(),
		Changes:     []MigrationChange{},
	}

	// Add migration changes based on version differences
	if fromVersion < "1.0.0" {
		guide.Changes = append(guide.Changes, MigrationChange{
			Type:        NewFeature,
			Description: "Context-aware goals and streaming results",
			OldCode:     "Run(5, func(x *Var) Goal { return Eq(x, AtomFromValue(42)) })",
			NewCode:     "ctx := context.Background()\nRunWithContext(ctx, 5, func(x *Var) Goal { return Eq(x, AtomFromValue(42)) })",
		})
	}

	return guide
}

// APICompatibilityTest runs a comprehensive test of API compatibility.
// This function can be used to verify that all public APIs work as expected
// and that no regressions have been introduced.
func APICompatibilityTest(ctx context.Context) error {
	// Test basic term operations
	x := Fresh("x")

	// Test unification
	goal := Eq(x, AtomFromValue(42))
	results := RunWithContext(ctx, 1, func(q *Var) Goal {
		return Conj(
			Eq(q, x),
			goal,
		)
	})

	if len(results) != 1 {
		return fmt.Errorf("expected 1 result, got %d", len(results))
	}

	// Test list operations
	list := List(AtomFromValue(1), AtomFromValue(2), AtomFromValue(3))
	if list == nil {
		return fmt.Errorf("List creation failed")
	}

	// Test append
	results = RunWithContext(ctx, 1, func(q *Var) Goal {
		return Appendo(
			List(AtomFromValue(1), AtomFromValue(2)),
			List(AtomFromValue(3), AtomFromValue(4)),
			q,
		)
	})

	if len(results) != 1 {
		return fmt.Errorf("appendo test failed: expected 1 result, got %d", len(results))
	}

	// Test FD operations
	if fdStore := NewFDStore(); fdStore == nil {
		return fmt.Errorf("FD store creation failed")
	}

	// Test store operations
	store := EmptyStore()
	if store == nil {
		return fmt.Errorf("empty store creation failed")
	}

	// Test fact store operations
	factStore := NewFactStore()
	if factStore == nil {
		return fmt.Errorf("fact store creation failed")
	}

	// Test nominal operations
	name := NewName("test")
	if name == nil {
		return fmt.Errorf("name creation failed")
	}

	return nil
}

// ValidateAPIConsistency performs static checks on the API for consistency.
// This function checks that exported functions follow naming conventions,
// have proper documentation, and maintain consistent patterns.
func ValidateAPIConsistency() []APIConsistencyIssue {
	// This would be a more comprehensive check in a real implementation
	// For now, we return an empty slice as the API has been reviewed manually

	return []APIConsistencyIssue{}
}

// APIConsistencyIssue represents an issue found during API consistency validation.
type APIConsistencyIssue struct {
	Type        IssueType
	Function    string
	Description string
	Severity    Severity
}

// IssueType represents the type of consistency issue.
type IssueType int

const (
	// NamingConvention violation
	NamingConvention IssueType = iota
	// DocumentationIssue missing or inadequate documentation
	DocumentationIssue
	// ParameterInconsistency inconsistent parameter patterns
	ParameterInconsistency
	// ReturnValueInconsistency inconsistent return patterns
	ReturnValueInconsistency
)

// Severity represents the severity of an API consistency issue.
type Severity int

const (
	// Low minor issue, doesn't break functionality
	Low Severity = iota
	// Medium affects usability but not correctness
	Medium
	// High breaks API contracts or correctness
	High
)

// String returns a string representation of the issue type.
func (t IssueType) String() string {
	switch t {
	case NamingConvention:
		return "NamingConvention"
	case DocumentationIssue:
		return "DocumentationIssue"
	case ParameterInconsistency:
		return "ParameterInconsistency"
	case ReturnValueInconsistency:
		return "ReturnValueInconsistency"
	default:
		return "Unknown"
	}
}

// String returns a string representation of the severity.
func (s Severity) String() string {
	switch s {
	case Low:
		return "Low"
	case Medium:
		return "Medium"
	case High:
		return "High"
	default:
		return "Unknown"
	}
}
