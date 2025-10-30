package minikanren

import (
	"context"
)

// LabelingStrategy defines the interface for variable and value ordering strategies.
// Strategies control how variables are selected for assignment and how their values are ordered.
// This enables pluggable heuristics for constraint satisfaction problems.
type LabelingStrategy interface {
	// SelectVariable chooses the next unassigned variable to branch on.
	// Returns the variable ID and available values in preferred order.
	// Returns -1, nil if no variables remain to assign.
	SelectVariable(store *FDStore) (varID int, values []int)

	// Name returns a descriptive name for this strategy.
	Name() string

	// Description returns detailed information about the strategy's behavior.
	Description() string
}

// SearchStrategy defines the interface for search algorithms in constraint solving.
// Search strategies control the exploration of the solution space through backtracking,
// pruning, and other optimization techniques.
type SearchStrategy interface {
	// Search performs constraint satisfaction using the given labeling strategy.
	// Returns all solutions found up to the limit, or an error if the search fails.
	// The context enables cancellation and timeout handling.
	Search(ctx context.Context, store *FDStore, labeling LabelingStrategy, limit int) ([][]int, error)

	// Name returns a descriptive name for this search strategy.
	Name() string

	// Description returns detailed information about the search algorithm.
	Description() string

	// SupportsPruning returns true if this strategy supports search tree pruning.
	SupportsPruning() bool
}

// StrategyConfig holds configuration for strategy selection and composition.
// It allows combining different labeling and search strategies with performance tuning.
type StrategyConfig struct {
	// Labeling strategy for variable and value ordering
	Labeling LabelingStrategy

	// Search strategy for solution exploration
	Search SearchStrategy

	// Random seed for reproducible random strategies
	RandomSeed int64

	// Performance monitoring enabled
	MonitorEnabled bool
}

// DefaultStrategyConfig returns a default strategy configuration with standard heuristics.
func DefaultStrategyConfig() *StrategyConfig {
	return &StrategyConfig{
		Labeling:       NewFirstFailLabeling(),
		Search:         NewDFSSearch(),
		RandomSeed:     42,
		MonitorEnabled: false,
	}
}

// NewRandomizedStrategyConfig returns a configuration with random strategies for testing.
func NewRandomizedStrategyConfig(seed int64) *StrategyConfig {
	return &StrategyConfig{
		Labeling:       NewRandomLabeling(seed),
		Search:         NewDFSSearch(),
		RandomSeed:     seed,
		MonitorEnabled: false,
	}
}

// Validate checks that the strategy configuration is complete and valid.
func (c *StrategyConfig) Validate() error {
	if c.Labeling == nil {
		return NewValidationError("labeling strategy cannot be nil")
	}
	if c.Search == nil {
		return NewValidationError("search strategy cannot be nil")
	}
	return nil
}

// Clone creates a deep copy of the strategy configuration.
func (c *StrategyConfig) Clone() *StrategyConfig {
	return &StrategyConfig{
		Labeling:       c.Labeling, // Strategies are typically stateless/immutable
		Search:         c.Search,
		RandomSeed:     c.RandomSeed,
		MonitorEnabled: c.MonitorEnabled,
	}
}

// StrategyRegistry provides a centralized registry for strategy discovery and management.
// It enables dynamic strategy loading and selection based on problem characteristics.
type StrategyRegistry struct {
	labeling map[string]LabelingStrategy
	search   map[string]SearchStrategy
}

// NewStrategyRegistry creates a new strategy registry with built-in strategies.
func NewStrategyRegistry() *StrategyRegistry {
	reg := &StrategyRegistry{
		labeling: make(map[string]LabelingStrategy),
		search:   make(map[string]SearchStrategy),
	}

	// Register built-in labeling strategies
	reg.RegisterLabeling(NewFirstFailLabeling())
	reg.RegisterLabeling(NewDomainSizeLabeling())
	reg.RegisterLabeling(NewDegreeLabeling())
	reg.RegisterLabeling(NewLexicographicLabeling())
	reg.RegisterLabeling(NewRandomLabeling(42))

	// Register built-in search strategies
	reg.RegisterSearch(NewDFSSearch())
	reg.RegisterSearch(NewBFSSearch())
	reg.RegisterSearch(NewLimitedDepthSearch(1000))

	return reg
}

// RegisterLabeling adds a labeling strategy to the registry.
func (r *StrategyRegistry) RegisterLabeling(strategy LabelingStrategy) {
	r.labeling[strategy.Name()] = strategy
}

// RegisterSearch adds a search strategy to the registry.
func (r *StrategyRegistry) RegisterSearch(strategy SearchStrategy) {
	r.search[strategy.Name()] = strategy
}

// GetLabeling retrieves a labeling strategy by name.
func (r *StrategyRegistry) GetLabeling(name string) (LabelingStrategy, bool) {
	strategy, ok := r.labeling[name]
	return strategy, ok
}

// GetSearch retrieves a search strategy by name.
func (r *StrategyRegistry) GetSearch(name string) (SearchStrategy, bool) {
	strategy, ok := r.search[name]
	return strategy, ok
}

// ListLabeling returns all registered labeling strategy names.
func (r *StrategyRegistry) ListLabeling() []string {
	names := make([]string, 0, len(r.labeling))
	for name := range r.labeling {
		names = append(names, name)
	}
	return names
}

// ListSearch returns all registered search strategy names.
func (r *StrategyRegistry) ListSearch() []string {
	names := make([]string, 0, len(r.search))
	for name := range r.search {
		names = append(names, name)
	}
	return names
}

// Global registry instance for default strategy access
var globalRegistry = NewStrategyRegistry()

// GetGlobalRegistry returns the global strategy registry.
func GetGlobalRegistry() *StrategyRegistry {
	return globalRegistry
}

// StrategySelector provides intelligent strategy selection based on problem characteristics.
// It analyzes the constraint store and recommends optimal strategies for different problem types.
type StrategySelector struct {
	registry *StrategyRegistry
}

// NewStrategySelector creates a new strategy selector with the given registry.
func NewStrategySelector(registry *StrategyRegistry) *StrategySelector {
	if registry == nil {
		registry = globalRegistry
	}
	return &StrategySelector{registry: registry}
}

// SelectForProblem analyzes the FD store and returns recommended strategies.
// The selection is based on problem size, constraint types, and domain characteristics.
func (s *StrategySelector) SelectForProblem(store *FDStore) *StrategyConfig {
	config := DefaultStrategyConfig().Clone()

	// Analyze problem characteristics
	varCount := len(store.vars)
	avgDomainSize := 0
	maxDomainSize := 0
	totalConstraints := 0

	for _, v := range store.vars {
		size := v.domain.Count()
		avgDomainSize += size
		if size > maxDomainSize {
			maxDomainSize = size
		}
		totalConstraints += len(v.peers)
	}

	if varCount > 0 {
		avgDomainSize /= varCount
		totalConstraints /= 2 // peers are bidirectional
	}

	// Select strategies based on problem analysis
	if varCount < 10 && avgDomainSize < 5 {
		// Small problems: lexicographic ordering is often fastest
		if lex, ok := s.registry.GetLabeling("lexicographic"); ok {
			config.Labeling = lex
		}
	} else if totalConstraints > varCount*2 {
		// Highly constrained: degree-based ordering helps
		if deg, ok := s.registry.GetLabeling("degree"); ok {
			config.Labeling = deg
		}
	} else {
		// Default to first-fail (dom/deg) for general problems
		if ff, ok := s.registry.GetLabeling("first-fail"); ok {
			config.Labeling = ff
		}
	}

	// Select search strategy based on problem characteristics
	if varCount > 50 || maxDomainSize > 100 {
		// Large problems: use limited depth to prevent stack overflow
		if limited, ok := s.registry.GetSearch("limited-depth"); ok {
			config.Search = limited
		} else if dfs, ok := s.registry.GetSearch("dfs"); ok {
			config.Search = dfs // fallback
		}
	} else if varCount < 5 && avgDomainSize < 3 {
		// Very small problems: BFS can be efficient for finding all solutions
		if bfs, ok := s.registry.GetSearch("bfs"); ok {
			config.Search = bfs
		} else if dfs, ok := s.registry.GetSearch("dfs"); ok {
			config.Search = dfs // fallback
		}
	} else {
		// Default to DFS for general constraint solving
		if dfs, ok := s.registry.GetSearch("dfs"); ok {
			config.Search = dfs
		}
	}

	return config
}

// ValidationError represents strategy configuration validation errors.
type ValidationError struct {
	Message string
}

func (e ValidationError) Error() string {
	return "strategy validation error: " + e.Message
}

// NewValidationError creates a new validation error.
func NewValidationError(message string) ValidationError {
	return ValidationError{Message: message}
}
