package minikanren

import (
	"math/rand"
	"sort"
)

// FirstFailLabeling implements the first-fail principle using domain size / degree ratio.
// This is one of the most effective general-purpose variable ordering heuristics.
// Variables with smaller domain/degree ratios are selected first, combining domain size
// and constraint tightness for optimal branching decisions.
type FirstFailLabeling struct{}

// NewFirstFailLabeling creates a new first-fail labeling strategy.
func NewFirstFailLabeling() *FirstFailLabeling {
	return &FirstFailLabeling{}
}

// SelectVariable implements the first-fail heuristic (domain size / degree).
// Selects the variable with the smallest domain/degree ratio.
func (s *FirstFailLabeling) SelectVariable(store *FDStore) (int, []int) {
	bestID := -1
	bestScore := -1.0
	var bestChoices []int

	for _, v := range store.vars {
		size := v.domain.Count()
		if size <= 1 {
			continue
		}
		degree := store.variableDegree(v)
		score := float64(size) / float64(1+degree) // dom/deg ratio
		if bestID == -1 || score < bestScore {
			bestScore = score
			bestID = v.ID
		}
	}

	if bestID == -1 {
		return -1, nil
	}

	dom := store.idToVar[bestID].domain
	dom.IterateValues(func(val int) { bestChoices = append(bestChoices, val) })
	sort.Ints(bestChoices) // ascending order
	return bestID, bestChoices
}

// Name returns the strategy name.
func (s *FirstFailLabeling) Name() string {
	return "first-fail"
}

// Description returns detailed information about the strategy.
func (s *FirstFailLabeling) Description() string {
	return "First-fail heuristic: selects variables with smallest domain/degree ratio, combining domain size and constraint tightness for optimal branching"
}

// DomainSizeLabeling implements pure domain size ordering.
// Variables with the smallest domains are selected first.
// This is effective for problems where domain reduction is the primary constraint.
type DomainSizeLabeling struct{}

// NewDomainSizeLabeling creates a new domain size labeling strategy.
func NewDomainSizeLabeling() *DomainSizeLabeling {
	return &DomainSizeLabeling{}
}

// SelectVariable selects the variable with the smallest domain.
func (s *DomainSizeLabeling) SelectVariable(store *FDStore) (int, []int) {
	bestID := -1
	bestSize := -1
	var bestChoices []int

	for _, v := range store.vars {
		size := v.domain.Count()
		if size <= 1 {
			continue
		}
		if bestID == -1 || size < bestSize {
			bestSize = size
			bestID = v.ID
		}
	}

	if bestID == -1 {
		return -1, nil
	}

	dom := store.idToVar[bestID].domain
	dom.IterateValues(func(val int) { bestChoices = append(bestChoices, val) })
	sort.Ints(bestChoices)
	return bestID, bestChoices
}

// Name returns the strategy name.
func (s *DomainSizeLabeling) Name() string {
	return "domain-size"
}

// Description returns detailed information about the strategy.
func (s *DomainSizeLabeling) Description() string {
	return "Domain size heuristic: selects variables with smallest domains first, effective when domain reduction is the primary constraint"
}

// DegreeLabeling implements degree-based variable ordering.
// Variables with the highest number of constraints (degree) are selected first.
// This is effective for tightly constrained problems where constraint propagation matters most.
type DegreeLabeling struct{}

// NewDegreeLabeling creates a new degree labeling strategy.
func NewDegreeLabeling() *DegreeLabeling {
	return &DegreeLabeling{}
}

// SelectVariable selects the variable with the highest degree (most constraints).
func (s *DegreeLabeling) SelectVariable(store *FDStore) (int, []int) {
	bestID := -1
	bestDegree := -1
	var bestChoices []int

	for _, v := range store.vars {
		size := v.domain.Count()
		if size <= 1 {
			continue
		}
		degree := store.variableDegree(v)
		if bestID == -1 || degree > bestDegree {
			bestDegree = degree
			bestID = v.ID
		}
	}

	if bestID == -1 {
		return -1, nil
	}

	dom := store.idToVar[bestID].domain
	dom.IterateValues(func(val int) { bestChoices = append(bestChoices, val) })
	sort.Ints(bestChoices)
	return bestID, bestChoices
}

// Name returns the strategy name.
func (s *DegreeLabeling) Name() string {
	return "degree"
}

// Description returns detailed information about the strategy.
func (s *DegreeLabeling) Description() string {
	return "Degree heuristic: selects variables with highest number of constraints first, effective for tightly constrained problems"
}

// LexicographicLabeling implements lexicographic variable ordering.
// Variables are selected in order of their IDs (creation order).
// This provides deterministic, predictable behavior and is often fastest for small problems.
type LexicographicLabeling struct{}

// NewLexicographicLabeling creates a new lexicographic labeling strategy.
func NewLexicographicLabeling() *LexicographicLabeling {
	return &LexicographicLabeling{}
}

// SelectVariable selects the first unassigned variable by ID order.
func (s *LexicographicLabeling) SelectVariable(store *FDStore) (int, []int) {
	for _, v := range store.vars {
		size := v.domain.Count()
		if size <= 1 {
			continue
		}
		dom := v.domain
		var choices []int
		dom.IterateValues(func(val int) { choices = append(choices, val) })
		sort.Ints(choices)
		return v.ID, choices
	}
	return -1, nil
}

// Name returns the strategy name.
func (s *LexicographicLabeling) Name() string {
	return "lexicographic"
}

// Description returns detailed information about the strategy.
func (s *LexicographicLabeling) Description() string {
	return "Lexicographic heuristic: selects variables in creation order, providing deterministic behavior and speed for small problems"
}

// RandomLabeling implements random variable ordering.
// Variables are selected randomly, with reproducible results using a seed.
// Useful for testing strategy robustness and finding different solution paths.
type RandomLabeling struct {
	rng *rand.Rand
}

// NewRandomLabeling creates a new random labeling strategy with the given seed.
func NewRandomLabeling(seed int64) *RandomLabeling {
	return &RandomLabeling{
		rng: rand.New(rand.NewSource(seed)),
	}
}

// SelectVariable selects a random unassigned variable.
func (s *RandomLabeling) SelectVariable(store *FDStore) (int, []int) {
	// Collect candidates
	var candidates []*FDVar
	for _, v := range store.vars {
		if v.domain.Count() > 1 {
			candidates = append(candidates, v)
		}
	}

	if len(candidates) == 0 {
		return -1, nil
	}

	// Select random variable
	selected := candidates[s.rng.Intn(len(candidates))]
	dom := selected.domain
	var choices []int
	dom.IterateValues(func(val int) { choices = append(choices, val) })

	// Shuffle choices randomly
	for i := len(choices) - 1; i > 0; i-- {
		j := s.rng.Intn(i + 1)
		choices[i], choices[j] = choices[j], choices[i]
	}

	return selected.ID, choices
}

// Name returns the strategy name.
func (s *RandomLabeling) Name() string {
	return "random"
}

// Description returns detailed information about the strategy.
func (s *RandomLabeling) Description() string {
	return "Random heuristic: selects variables randomly, useful for testing strategy robustness and finding alternative solutions"
}

// CompositeLabeling allows combining multiple labeling strategies.
// Strategies are tried in order until one successfully selects a variable.
// This enables fallback strategies and hybrid approaches.
type CompositeLabeling struct {
	strategies []LabelingStrategy
	name       string
}

// NewCompositeLabeling creates a new composite labeling strategy.
func NewCompositeLabeling(name string, strategies ...LabelingStrategy) *CompositeLabeling {
	return &CompositeLabeling{
		strategies: strategies,
		name:       name,
	}
}

// SelectVariable tries each strategy in order until one succeeds.
func (s *CompositeLabeling) SelectVariable(store *FDStore) (int, []int) {
	for _, strategy := range s.strategies {
		if varID, values := strategy.SelectVariable(store); varID != -1 {
			return varID, values
		}
	}
	return -1, nil
}

// Name returns the composite strategy name.
func (s *CompositeLabeling) Name() string {
	return s.name
}

// Description returns detailed information about the composite strategy.
func (s *CompositeLabeling) Description() string {
	desc := "Composite strategy combining: "
	for i, strategy := range s.strategies {
		if i > 0 {
			desc += ", "
		}
		desc += strategy.Name()
	}
	return desc
}

// AdaptiveLabeling provides dynamic strategy selection based on search progress.
// It starts with one strategy and switches to others based on performance metrics.
// This enables adaptive behavior during constraint solving.
type AdaptiveLabeling struct {
	strategies  []LabelingStrategy
	current     int
	switchAfter int
	counter     int
	name        string
}

// NewAdaptiveLabeling creates a new adaptive labeling strategy.
func NewAdaptiveLabeling(name string, switchAfter int, strategies ...LabelingStrategy) *AdaptiveLabeling {
	if len(strategies) == 0 {
		strategies = []LabelingStrategy{NewFirstFailLabeling()}
	}
	return &AdaptiveLabeling{
		strategies:  strategies,
		current:     0,
		switchAfter: switchAfter,
		counter:     0,
		name:        name,
	}
}

// SelectVariable uses the current strategy and switches periodically.
func (s *AdaptiveLabeling) SelectVariable(store *FDStore) (int, []int) {
	varID, values := s.strategies[s.current].SelectVariable(store)
	if varID != -1 {
		s.counter++
		if s.counter >= s.switchAfter {
			s.current = (s.current + 1) % len(s.strategies)
			s.counter = 0
		}
	}
	return varID, values
}

// Name returns the adaptive strategy name.
func (s *AdaptiveLabeling) Name() string {
	return s.name
}

// Description returns detailed information about the adaptive strategy.
func (s *AdaptiveLabeling) Description() string {
	desc := "Adaptive strategy switching every " + string(rune(s.switchAfter)) + " selections between: "
	for i, strategy := range s.strategies {
		if i > 0 {
			desc += ", "
		}
		desc += strategy.Name()
	}
	return desc
}
