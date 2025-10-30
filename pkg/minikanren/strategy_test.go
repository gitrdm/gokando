package minikanren

import (
	"context"
	"testing"
	"time"
)

func TestStrategyConfig(t *testing.T) {
	t.Run("DefaultStrategyConfig", func(t *testing.T) {
		config := DefaultStrategyConfig()
		if config == nil {
			t.Fatal("DefaultStrategyConfig returned nil")
		}
		if config.Labeling == nil {
			t.Error("Default config missing labeling strategy")
		}
		if config.Search == nil {
			t.Error("Default config missing search strategy")
		}
		if config.RandomSeed != 42 {
			t.Errorf("Expected random seed 42, got %d", config.RandomSeed)
		}
	})

	t.Run("StrategyConfigValidation", func(t *testing.T) {
		config := &StrategyConfig{}
		err := config.Validate()
		if err == nil {
			t.Error("Expected validation error for nil strategies")
		}

		config.Labeling = NewFirstFailLabeling()
		err = config.Validate()
		if err == nil {
			t.Error("Expected validation error for nil search strategy")
		}

		config.Search = NewDFSSearch()
		err = config.Validate()
		if err != nil {
			t.Errorf("Expected no validation error, got: %v", err)
		}
	})

	t.Run("StrategyConfigClone", func(t *testing.T) {
		original := DefaultStrategyConfig()
		clone := original.Clone()

		if clone.Labeling.Name() != original.Labeling.Name() {
			t.Error("Clone has different labeling strategy")
		}
		if clone.Search.Name() != original.Search.Name() {
			t.Error("Clone has different search strategy")
		}
		if clone.RandomSeed != original.RandomSeed {
			t.Error("Clone has different random seed")
		}
	})
}

func TestStrategyRegistry(t *testing.T) {
	t.Run("NewStrategyRegistry", func(t *testing.T) {
		reg := NewStrategyRegistry()
		if reg == nil {
			t.Fatal("NewStrategyRegistry returned nil")
		}

		// Check built-in strategies are registered
		labeling := reg.ListLabeling()
		if len(labeling) == 0 {
			t.Error("No labeling strategies registered")
		}

		search := reg.ListSearch()
		if len(search) == 0 {
			t.Error("No search strategies registered")
		}
	})

	t.Run("StrategyRegistration", func(t *testing.T) {
		reg := NewStrategyRegistry()

		// Test labeling strategy registration
		labeling := NewFirstFailLabeling()
		reg.RegisterLabeling(labeling)

		retrieved, ok := reg.GetLabeling(labeling.Name())
		if !ok {
			t.Error("Failed to retrieve registered labeling strategy")
		}
		if retrieved.Name() != labeling.Name() {
			t.Error("Retrieved wrong labeling strategy")
		}

		// Test search strategy registration
		search := NewDFSSearch()
		reg.RegisterSearch(search)

		retrievedSearch, ok := reg.GetSearch(search.Name())
		if !ok {
			t.Error("Failed to retrieve registered search strategy")
		}
		if retrievedSearch.Name() != search.Name() {
			t.Error("Retrieved wrong search strategy")
		}
	})
}

func TestLabelingStrategies(t *testing.T) {
	store := NewFDStoreWithDomain(4)
	vars := store.MakeFDVars(3)

	// Set up a simple constraint problem
	store.AddAllDifferent(vars)

	strategies := []LabelingStrategy{
		NewFirstFailLabeling(),
		NewDomainSizeLabeling(),
		NewDegreeLabeling(),
		NewLexicographicLabeling(),
		NewRandomLabeling(42),
	}

	for _, strategy := range strategies {
		t.Run("Strategy_"+strategy.Name(), func(t *testing.T) {
			varID, values := strategy.SelectVariable(store)
			if varID == -1 {
				t.Errorf("Strategy %s returned no variable to select", strategy.Name())
			}
			if len(values) == 0 {
				t.Errorf("Strategy %s returned no values for variable %d", strategy.Name(), varID)
			}

			// Verify values are valid for the domain
			for _, val := range values {
				if val < 1 || val > 4 {
					t.Errorf("Strategy %s returned invalid value %d", strategy.Name(), val)
				}
			}
		})
	}
}

func TestCompositeLabeling(t *testing.T) {
	store := NewFDStoreWithDomain(4)
	vars := store.MakeFDVars(3)
	store.AddAllDifferent(vars)

	strategies := []LabelingStrategy{
		NewFirstFailLabeling(),
		NewDomainSizeLabeling(),
	}

	composite := NewCompositeLabeling("test-composite", strategies...)

	t.Run("CompositeSelection", func(t *testing.T) {
		varID, values := composite.SelectVariable(store)
		if varID == -1 {
			t.Error("Composite strategy returned no variable")
		}
		if len(values) == 0 {
			t.Error("Composite strategy returned no values")
		}
	})

	t.Run("CompositeName", func(t *testing.T) {
		if composite.Name() != "test-composite" {
			t.Errorf("Expected name 'test-composite', got '%s'", composite.Name())
		}
	})
}

func TestAdaptiveLabeling(t *testing.T) {
	store := NewFDStoreWithDomain(4)
	vars := store.MakeFDVars(3)
	store.AddAllDifferent(vars)

	strategies := []LabelingStrategy{
		NewFirstFailLabeling(),
		NewDomainSizeLabeling(),
	}

	adaptive := NewAdaptiveLabeling("test-adaptive", 2, strategies...)

	t.Run("AdaptiveSelection", func(t *testing.T) {
		// Test multiple selections to trigger strategy switching
		for i := 0; i < 5; i++ {
			varID, values := adaptive.SelectVariable(store)
			if varID == -1 {
				continue // No more variables
			}
			if len(values) == 0 {
				t.Errorf("Adaptive strategy returned no values on iteration %d", i)
			}
		}
	})

	t.Run("AdaptiveName", func(t *testing.T) {
		if adaptive.Name() != "test-adaptive" {
			t.Errorf("Expected name 'test-adaptive', got '%s'", adaptive.Name())
		}
	})
}

func TestSearchStrategies(t *testing.T) {
	store := NewFDStoreWithDomain(4)
	vars := store.MakeFDVars(3)
	store.AddAllDifferent(vars)

	labeling := NewFirstFailLabeling()
	ctx := context.Background()

	strategies := []SearchStrategy{
		NewDFSSearch(),
		NewBFSSearch(),
		NewLimitedDepthSearch(10),
	}

	for _, strategy := range strategies {
		t.Run("Strategy_"+strategy.Name(), func(t *testing.T) {
			solutions, err := strategy.Search(ctx, store, labeling, 10)
			if err != nil {
				t.Errorf("Strategy %s failed: %v", strategy.Name(), err)
			}
			// For 3 variables with AllDifferent, we expect solutions
			if len(solutions) == 0 {
				t.Errorf("Strategy %s found no solutions", strategy.Name())
			}
			// Verify solution validity
			for _, sol := range solutions {
				if len(sol) != 3 {
					t.Errorf("Invalid solution length: %d", len(sol))
				}
				// Check AllDifferent constraint
				seen := make(map[int]bool)
				for _, val := range sol {
					if seen[val] {
						t.Errorf("Solution violates AllDifferent: %v", sol)
						break
					}
					seen[val] = true
				}
			}
		})
	}
}

func TestIterativeDeepeningSearch(t *testing.T) {
	store := NewFDStoreWithDomain(4)
	vars := store.MakeFDVars(3)
	store.AddAllDifferent(vars)

	labeling := NewFirstFailLabeling()
	ctx := context.Background()

	strategy := NewIterativeDeepeningSearch(10, 2)

	solutions, err := strategy.Search(ctx, store, labeling, 5)
	if err != nil {
		t.Errorf("Iterative deepening search failed: %v", err)
	}
	if len(solutions) == 0 {
		t.Error("Iterative deepening found no solutions")
	}
}

func TestFDStoreStrategyIntegration(t *testing.T) {
	t.Run("DefaultStrategy", func(t *testing.T) {
		store := NewFDStoreWithDomain(4)
		strategy := store.GetStrategy()
		if strategy == nil {
			t.Error("Store returned nil strategy")
		}
	})

	t.Run("SetStrategy", func(t *testing.T) {
		store := NewFDStoreWithDomain(4)
		newStrategy := &StrategyConfig{
			Labeling:   NewDomainSizeLabeling(),
			Search:     NewBFSSearch(),
			RandomSeed: 123,
		}

		store.SetStrategy(newStrategy)
		current := store.GetStrategy()

		if current.Labeling.Name() != "domain-size" {
			t.Error("Strategy not updated correctly")
		}
		if current.Search.Name() != "bfs" {
			t.Error("Search strategy not updated correctly")
		}
		if current.RandomSeed != 123 {
			t.Error("Random seed not updated correctly")
		}
	})

	t.Run("StrategyComponents", func(t *testing.T) {
		store := NewFDStoreWithDomain(4)

		// Test individual component updates
		store.SetLabelingStrategy(NewDegreeLabeling())
		if store.GetLabelingStrategy().Name() != "degree" {
			t.Error("Labeling strategy not set correctly")
		}

		store.SetSearchStrategy(NewLimitedDepthSearch(5))
		if store.GetSearchStrategy().Name() != "limited-depth" {
			t.Error("Search strategy not set correctly")
		}
	})
}

func TestStrategyCancellation(t *testing.T) {
	store := NewFDStoreWithDomain(10) // Larger domain to make search slower
	vars := store.MakeFDVars(5)
	store.AddAllDifferent(vars)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	strategy := DefaultStrategyConfig()
	solutions, err := store.SolveWithStrategy(ctx, strategy, 100)

	// Should either succeed with some solutions or be cancelled
	if err != nil && err != context.DeadlineExceeded && err != context.Canceled {
		t.Errorf("Unexpected error: %v", err)
	}
	// Solutions might be empty if cancelled quickly
	_ = solutions
}

func TestBackwardCompatibility(t *testing.T) {
	// Test that old SolverConfig still works
	oldConfig := DefaultSolverConfig()
	store := NewFDStoreWithConfig(4, oldConfig)

	// Should be able to solve
	ctx := context.Background()
	solutions, err := store.Solve(ctx, 1)
	if err != nil {
		t.Errorf("Backward compatibility failed: %v", err)
	}
	if len(solutions) == 0 {
		t.Error("No solutions found with old config")
	}
}

func BenchmarkLabelingStrategies(b *testing.B) {
	store := NewFDStoreWithDomain(8)
	vars := store.MakeFDVars(6)
	store.AddAllDifferent(vars)

	strategies := []struct {
		name     string
		strategy LabelingStrategy
	}{
		{"first-fail", NewFirstFailLabeling()},
		{"domain-size", NewDomainSizeLabeling()},
		{"degree", NewDegreeLabeling()},
		{"lexicographic", NewLexicographicLabeling()},
	}

	for _, s := range strategies {
		b.Run(s.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				s.strategy.SelectVariable(store)
			}
		})
	}
}

func BenchmarkSearchStrategies(b *testing.B) {
	store := NewFDStoreWithDomain(6)
	vars := store.MakeFDVars(4)
	store.AddAllDifferent(vars)

	labeling := NewFirstFailLabeling()
	ctx := context.Background()

	strategies := []struct {
		name     string
		strategy SearchStrategy
	}{
		{"dfs", NewDFSSearch()},
		{"bfs", NewBFSSearch()},
		{"limited-depth", NewLimitedDepthSearch(10)},
	}

	for _, s := range strategies {
		b.Run(s.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _ = s.strategy.Search(ctx, store, labeling, 1)
			}
		})
	}
}

func TestStrategySelector(t *testing.T) {
	selector := NewStrategySelector(nil) // Uses global registry

	t.Run("SmallProblemSelection", func(t *testing.T) {
		// Create a small problem: 4 variables, small domains, few constraints
		store := NewFDStoreWithDomain(3)
		vars := store.MakeFDVars(4)
		// Add minimal constraints
		store.AddAllDifferent(vars[:2]) // Only 2 variables constrained

		config := selector.SelectForProblem(store)

		// Should select lexicographic for small problems
		if config.Labeling.Name() != "lexicographic" {
			t.Errorf("Expected lexicographic labeling for small problem, got %s", config.Labeling.Name())
		}
		// Should select DFS for general problems
		if config.Search.Name() != "dfs" {
			t.Errorf("Expected DFS search for small problem, got %s", config.Search.Name())
		}
	})

	t.Run("DefaultProblemSelection", func(t *testing.T) {
		// Create medium problem that should default to first-fail
		store := NewFDStoreWithDomain(6) // avgDomainSize = 6 >= 5
		vars := store.MakeFDVars(12)     // varCount = 12 >= 10
		store.AddAllDifferent(vars[:3])  // Some constraints but not highly constrained

		config := selector.SelectForProblem(store)

		// Should select first-fail for general problems
		if config.Labeling.Name() != "first-fail" {
			t.Errorf("Expected first-fail labeling for general problem, got %s", config.Labeling.Name())
		}
		if config.Search.Name() != "dfs" {
			t.Errorf("Expected DFS search for general problem, got %s", config.Search.Name())
		}
	})

	t.Run("LargeProblemSelection", func(t *testing.T) {
		// Create large problem: many variables or large domains
		store := NewFDStoreWithDomain(150) // Large domain
		vars := store.MakeFDVars(10)
		_ = vars // Mark as intentionally unused - we're testing domain size analysis

		config := selector.SelectForProblem(store)

		// Should select limited-depth search for large domains
		if config.Search.Name() != "limited-depth" {
			t.Errorf("Expected limited-depth search for large problem, got %s", config.Search.Name())
		}
	})

	t.Run("VerySmallProblemSelection", func(t *testing.T) {
		// Create very small problem
		store := NewFDStoreWithDomain(2)
		vars := store.MakeFDVars(3)
		_ = vars // Mark as intentionally unused - we're testing domain size analysis

		config := selector.SelectForProblem(store)

		// Should select BFS for very small problems
		if config.Search.Name() != "bfs" {
			t.Errorf("Expected BFS search for very small problem, got %s", config.Search.Name())
		}
	})

	t.Run("FallbackBehavior", func(t *testing.T) {
		// Test fallback when preferred strategies aren't available
		emptyRegistry := &StrategyRegistry{
			labeling: make(map[string]LabelingStrategy),
			search:   make(map[string]SearchStrategy),
		}
		// Only register DFS
		emptyRegistry.RegisterSearch(NewDFSSearch())

		selector := NewStrategySelector(emptyRegistry)
		store := NewFDStoreWithDomain(100) // Would normally select limited-depth

		config := selector.SelectForProblem(store)

		// Should fall back to DFS when limited-depth isn't available
		if config.Search.Name() != "dfs" {
			t.Errorf("Expected DFS fallback, got %s", config.Search.Name())
		}
	})
}

func TestStrategySelectorIntegration(t *testing.T) {
	selector := NewStrategySelector(nil)

	t.Run("SelectedStrategiesWork", func(t *testing.T) {
		// Create a test problem
		store := NewFDStoreWithDomain(4)
		vars := store.MakeFDVars(3)
		store.AddAllDifferent(vars)

		// Get recommended strategy
		config := selector.SelectForProblem(store)

		// Verify the selected strategy actually works
		ctx := context.Background()
		solutions, err := store.SolveWithStrategy(ctx, config, 5)

		if err != nil {
			t.Errorf("Selected strategy failed: %v", err)
		}
		if len(solutions) == 0 {
			t.Error("Selected strategy found no solutions")
		}
	})

	t.Run("ProblemAnalysisAccuracy", func(t *testing.T) {
		testCases := []struct {
			name           string
			domainSize     int
			varCount       int
			constraints    int
			expectLabeling string
			expectSearch   string
		}{
			{"small-simple", 3, 4, 1, "lexicographic", "dfs"},
			{"general", 6, 12, 3, "first-fail", "dfs"},
			{"large-domain", 150, 10, 5, "first-fail", "limited-depth"},
			{"very-small", 2, 3, 2, "lexicographic", "bfs"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				store := NewFDStoreWithDomain(tc.domainSize)
				vars := store.MakeFDVars(tc.varCount)

				// Add constraints
				for i := 0; i < tc.constraints && i < tc.varCount-1; i++ {
					store.AddAllDifferent(vars[i : i+2])
				}

				config := selector.SelectForProblem(store)

				if config.Labeling.Name() != tc.expectLabeling {
					t.Errorf("Expected labeling %s, got %s", tc.expectLabeling, config.Labeling.Name())
				}
				if config.Search.Name() != tc.expectSearch {
					t.Errorf("Expected search %s, got %s", tc.expectSearch, config.Search.Name())
				}
			})
		}
	})
}
