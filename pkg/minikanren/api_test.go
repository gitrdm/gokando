package minikanren

import (
	"context"
	"fmt"
	"testing"
)

// TestAPIVersion tests version-related API functions.
func TestAPIVersion(t *testing.T) {
	version := CurrentAPIVersion()
	if version.Major != 1 || version.Minor != 0 || version.Patch != 0 {
		t.Errorf("expected version 1.0.0, got %s", version.String())
	}

	apiVersion := GetAPIVersion()
	if apiVersion != "1.0.0" {
		t.Errorf("expected API version '1.0.0', got '%s'", apiVersion)
	}

	if !CheckAPIVersion("1.0.0") {
		t.Error("expected API version 1.0.0 to be compatible with itself")
	}

	if CheckAPIVersion("2.0.0") {
		t.Error("expected API version 2.0.0 to be incompatible with 1.0.0")
	}
}

// TestMigrationGuide tests the migration guide functionality.
func TestMigrationGuide(t *testing.T) {
	guide := GetMigrationGuide("0.9.0")
	if guide.FromVersion != "0.9.0" {
		t.Errorf("expected FromVersion '0.9.0', got '%s'", guide.FromVersion)
	}

	if guide.ToVersion != "1.0.0" {
		t.Errorf("expected ToVersion '1.0.0', got '%s'", guide.ToVersion)
	}

	if len(guide.Changes) == 0 {
		t.Error("expected migration guide to have changes")
	}
}

// TestAPICompatibilityTest tests the API compatibility test function.
func TestAPICompatibilityTest(t *testing.T) {
	ctx := context.Background()
	err := APICompatibilityTest(ctx)
	if err != nil {
		t.Errorf("API compatibility test failed: %v", err)
	}
}

// TestValidateAPIConsistency tests the API consistency validation.
func TestValidateAPIConsistency(t *testing.T) {
	issues := ValidateAPIConsistency()
	// Currently we expect no issues, but this test ensures the function runs
	if issues == nil {
		t.Error("ValidateAPIConsistency returned nil instead of empty slice")
	}
}

// TestDeprecatedError tests the deprecated error type.
func TestDeprecatedError(t *testing.T) {
	err := DeprecatedError{
		Function: "OldFunction",
		Message:  "use NewFunction instead",
		Version:  "2.0.0",
	}

	expected := "deprecated API usage: OldFunction - use NewFunction instead (removal in version 2.0.0)"
	if err.Error() != expected {
		t.Errorf("expected error message '%s', got '%s'", expected, err.Error())
	}
}

// TestIssueTypeString tests the IssueType string representation.
func TestIssueTypeString(t *testing.T) {
	tests := []struct {
		issueType IssueType
		expected  string
	}{
		{NamingConvention, "NamingConvention"},
		{DocumentationIssue, "DocumentationIssue"},
		{ParameterInconsistency, "ParameterInconsistency"},
		{ReturnValueInconsistency, "ReturnValueInconsistency"},
		{IssueType(999), "Unknown"},
	}

	for _, test := range tests {
		if test.issueType.String() != test.expected {
			t.Errorf("expected %s.String() = '%s', got '%s'",
				test.issueType, test.expected, test.issueType.String())
		}
	}
}

// TestSeverityString tests the Severity string representation.
func TestSeverityString(t *testing.T) {
	tests := []struct {
		severity Severity
		expected string
	}{
		{Low, "Low"},
		{Medium, "Medium"},
		{High, "High"},
		{Severity(999), "Unknown"},
	}

	for _, test := range tests {
		if test.severity.String() != test.expected {
			t.Errorf("expected %s.String() = '%s', got '%s'",
				test.severity, test.expected, test.severity.String())
		}
	}
}

// ExampleCurrentAPIVersion demonstrates getting the current API version.
func ExampleCurrentAPIVersion() {
	version := CurrentAPIVersion()
	fmt.Printf("Current API version: %s\n", version.String())

	// Output:
	// Current API version: 1.0.0
}

// ExampleGetAPIVersion demonstrates getting the API version as a string.
func ExampleGetAPIVersion() {
	version := GetAPIVersion()
	fmt.Printf("API version: %s\n", version)

	// Output:
	// API version: 1.0.0
}

// ExampleCheckAPIVersion demonstrates checking API version compatibility.
func ExampleCheckAPIVersion() {
	// Check if current version is compatible with required version
	compatible := CheckAPIVersion("1.0.0")
	fmt.Printf("Compatible with 1.0.0: %v\n", compatible)

	compatible = CheckAPIVersion("2.0.0")
	fmt.Printf("Compatible with 2.0.0: %v\n", compatible)

	// Output:
	// Compatible with 1.0.0: true
	// Compatible with 2.0.0: false
}

// ExampleGetMigrationGuide demonstrates getting migration guidance.
func ExampleGetMigrationGuide() {
	guide := GetMigrationGuide("0.9.0")
	if guide != nil {
		fmt.Printf("Migration from %s to %s\n", guide.FromVersion, guide.ToVersion)
		fmt.Printf("Changes: %d\n", len(guide.Changes))
		if len(guide.Changes) > 0 {
			fmt.Printf("First change: %s\n", guide.Changes[0].Description)
		}
	} else {
		fmt.Println("No migration guide available")
	}

	// Output:
	// Migration from 0.9.0 to 1.0.0
	// Changes: 1
	// First change: Context-aware goals and streaming results
}

// ExampleAPICompatibilityTest demonstrates running API compatibility tests.
func ExampleAPICompatibilityTest() {
	ctx := context.Background()
	err := APICompatibilityTest(ctx)
	if err != nil {
		fmt.Printf("API compatibility test failed: %v\n", err)
	} else {
		fmt.Println("API compatibility test passed")
	}

	// Output:
	// API compatibility test passed
}

// ExampleValidateAPIConsistency demonstrates validating API consistency.
func ExampleValidateAPIConsistency() {
	issues := ValidateAPIConsistency()
	if len(issues) == 0 {
		fmt.Println("API is consistent")
	} else {
		fmt.Printf("Found %d API consistency issues\n", len(issues))
		for _, issue := range issues {
			fmt.Printf("- %s: %s (%s)\n", issue.Type, issue.Description, issue.Severity)
		}
	}

	// Output:
	// API is consistent
}
