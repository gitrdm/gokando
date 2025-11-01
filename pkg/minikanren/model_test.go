package minikanren

import (
	"testing"
)

func TestNewModel(t *testing.T) {
	model := NewModel()

	if model == nil {
		t.Fatal("NewModel() returned nil")
	}
	if model.VariableCount() != 0 {
		t.Errorf("VariableCount() = %d, want 0", model.VariableCount())
	}
	if model.ConstraintCount() != 0 {
		t.Errorf("ConstraintCount() = %d, want 0", model.ConstraintCount())
	}
	if model.Config() == nil {
		t.Error("Config() should not be nil")
	}
}

func TestNewModelWithConfig(t *testing.T) {
	config := &SolverConfig{
		VariableHeuristic: HeuristicDom,
		ValueHeuristic:    ValueOrderDesc,
		RandomSeed:        123,
	}

	model := NewModelWithConfig(config)

	if model == nil {
		t.Fatal("NewModelWithConfig() returned nil")
	}
	if model.Config() != config {
		t.Error("Config() should return the provided config")
	}

	// Test with nil config
	model2 := NewModelWithConfig(nil)
	if model2.Config() == nil {
		t.Error("NewModelWithConfig(nil) should use default config")
	}
}

func TestModel_NewVariable(t *testing.T) {
	model := NewModel()
	domain := NewBitSetDomain(10)

	v1 := model.NewVariable(domain)
	if v1 == nil {
		t.Fatal("NewVariable() returned nil")
	}
	if v1.ID() != 0 {
		t.Errorf("first variable ID = %d, want 0", v1.ID())
	}
	if model.VariableCount() != 1 {
		t.Errorf("VariableCount() = %d, want 1", model.VariableCount())
	}

	v2 := model.NewVariable(domain)
	if v2.ID() != 1 {
		t.Errorf("second variable ID = %d, want 1", v2.ID())
	}
	if model.VariableCount() != 2 {
		t.Errorf("VariableCount() = %d, want 2", model.VariableCount())
	}

	// Test MaxDomainSize tracking
	if model.MaxDomainSize() != 10 {
		t.Errorf("MaxDomainSize() = %d, want 10", model.MaxDomainSize())
	}

	largerDomain := NewBitSetDomain(20)
	model.NewVariable(largerDomain)
	if model.MaxDomainSize() != 20 {
		t.Errorf("MaxDomainSize() = %d, want 20", model.MaxDomainSize())
	}
}

func TestModel_NewVariableWithName(t *testing.T) {
	model := NewModel()
	domain := NewBitSetDomain(5)

	v := model.NewVariableWithName(domain, "testVar")
	if v == nil {
		t.Fatal("NewVariableWithName() returned nil")
	}
	if v.Name() != "testVar" {
		t.Errorf("Name() = %q, want %q", v.Name(), "testVar")
	}
	if v.ID() != 0 {
		t.Errorf("ID() = %d, want 0", v.ID())
	}
}

func TestModel_NewVariables(t *testing.T) {
	model := NewModel()
	domain := NewBitSetDomain(9)

	vars := model.NewVariables(5, domain)
	if len(vars) != 5 {
		t.Fatalf("NewVariables() returned %d variables, want 5", len(vars))
	}

	for i, v := range vars {
		if v.ID() != i {
			t.Errorf("vars[%d].ID() = %d, want %d", i, v.ID(), i)
		}
		if v.Domain().Count() != 9 {
			t.Errorf("vars[%d] domain count = %d, want 9", i, v.Domain().Count())
		}
	}

	if model.VariableCount() != 5 {
		t.Errorf("VariableCount() = %d, want 5", model.VariableCount())
	}
}

func TestModel_NewVariablesWithNames(t *testing.T) {
	model := NewModel()
	domain := NewBitSetDomain(3)
	names := []string{"red", "green", "blue"}

	vars := model.NewVariablesWithNames(names, domain)
	if len(vars) != len(names) {
		t.Fatalf("NewVariablesWithNames() returned %d variables, want %d", len(vars), len(names))
	}

	for i, v := range vars {
		if v.Name() != names[i] {
			t.Errorf("vars[%d].Name() = %q, want %q", i, v.Name(), names[i])
		}
	}
}

func TestModel_GetVariable(t *testing.T) {
	model := NewModel()
	domain := NewBitSetDomain(5)

	v1 := model.NewVariable(domain)
	v2 := model.NewVariable(domain)

	retrieved := model.GetVariable(0)
	if retrieved != v1 {
		t.Error("GetVariable(0) should return first variable")
	}

	retrieved = model.GetVariable(1)
	if retrieved != v2 {
		t.Error("GetVariable(1) should return second variable")
	}

	retrieved = model.GetVariable(999)
	if retrieved != nil {
		t.Error("GetVariable(999) should return nil for non-existent ID")
	}
}

func TestModel_Variables(t *testing.T) {
	model := NewModel()
	domain := NewBitSetDomain(5)

	v1 := model.NewVariable(domain)
	v2 := model.NewVariable(domain)
	v3 := model.NewVariable(domain)

	vars := model.Variables()
	if len(vars) != 3 {
		t.Fatalf("Variables() returned %d variables, want 3", len(vars))
	}

	if vars[0] != v1 || vars[1] != v2 || vars[2] != v3 {
		t.Error("Variables() returned variables in wrong order")
	}
}

func TestModel_SetConfig(t *testing.T) {
	model := NewModel()
	originalConfig := model.Config()

	newConfig := &SolverConfig{
		VariableHeuristic: HeuristicDeg,
		ValueHeuristic:    ValueOrderRandom,
		RandomSeed:        999,
	}

	model.SetConfig(newConfig)
	if model.Config() != newConfig {
		t.Error("Config() should return the new config")
	}

	// Test setting nil config (should not change)
	model.SetConfig(nil)
	if model.Config() != newConfig {
		t.Error("SetConfig(nil) should not change config")
	}

	// Verify original config is different
	if originalConfig == newConfig {
		t.Error("original and new config should be different instances")
	}
}

func TestModel_Validate(t *testing.T) {
	tests := []struct {
		name    string
		setup   func() *Model
		wantErr bool
	}{
		{
			name: "valid model",
			setup: func() *Model {
				m := NewModel()
				m.NewVariables(3, NewBitSetDomain(5))
				return m
			},
			wantErr: false,
		},
		{
			name: "empty model",
			setup: func() *Model {
				return NewModel()
			},
			wantErr: false,
		},
		{
			name: "variable with empty domain",
			setup: func() *Model {
				m := NewModel()
				m.NewVariable(NewBitSetDomainFromValues(5, []int{}))
				return m
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := tt.setup()
			err := model.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestModel_String(t *testing.T) {
	model := NewModel()
	model.NewVariables(5, NewBitSetDomain(9))

	str := model.String()
	if str == "" {
		t.Error("String() should not be empty")
	}

	// Should contain information about variables
	t.Logf("Model.String() = %s", str)
}

func TestModel_ConcurrentReads(t *testing.T) {
	model := NewModel()
	model.NewVariables(10, NewBitSetDomain(10))

	// Test concurrent reads are safe
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			_ = model.VariableCount()
			_ = model.Variables()
			_ = model.Config()
			_ = model.MaxDomainSize()
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

// Edge case tests for >90% coverage
// Note: Constraint-related tests will be added in Phase 2 when constraints are implemented
func TestModel_EdgeCases(t *testing.T) {
	t.Run("ConstraintCount on empty model", func(t *testing.T) {
		model := NewModel()
		if model.ConstraintCount() != 0 {
			t.Errorf("ConstraintCount = %d, want 0", model.ConstraintCount())
		}
	})

	t.Run("Constraints on empty model", func(t *testing.T) {
		model := NewModel()
		constraints := model.Constraints()
		if len(constraints) != 0 {
			t.Errorf("Constraints length = %d, want 0", len(constraints))
		}
	})
}
