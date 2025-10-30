package minikanren

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// Fact represents a logical fact stored in the fact database.
// Facts are immutable tuples of terms that can be queried through unification.
type Fact struct {
	// ID is a unique identifier for this fact instance
	ID string

	// Terms contains the fact data as a slice of terms
	Terms []Term

	// Metadata contains optional fact metadata
	Metadata map[string]interface{}
}

// NewFact creates a new fact with the given terms.
func NewFact(terms ...Term) *Fact {
	return &Fact{
		ID:       generateFactID(),
		Terms:    terms,
		Metadata: make(map[string]interface{}),
	}
}

// NewFactWithID creates a new fact with a specific ID and terms.
func NewFactWithID(id string, terms ...Term) *Fact {
	return &Fact{
		ID:       id,
		Terms:    terms,
		Metadata: make(map[string]interface{}),
	}
}

// generateFactID generates a unique ID for facts using timestamp and counter.
var factIDCounter int64

func generateFactID() string {
	counter := atomic.AddInt64(&factIDCounter, 1)
	timestamp := time.Now().UnixNano()
	return fmt.Sprintf("fact_%d_%d", timestamp, counter)
}

// Clone creates a deep copy of the fact.
func (f *Fact) Clone() *Fact {
	terms := make([]Term, len(f.Terms))
	copy(terms, f.Terms)

	metadata := make(map[string]interface{})
	for k, v := range f.Metadata {
		metadata[k] = v
	}

	return &Fact{
		ID:       f.ID,
		Terms:    terms,
		Metadata: metadata,
	}
}

// String returns a string representation of the fact.
func (f *Fact) String() string {
	if len(f.Terms) == 0 {
		return "fact()"
	}

	result := "fact("
	for i, term := range f.Terms {
		if i > 0 {
			result += ", "
		}
		result += fmt.Sprintf("%v", term)
	}
	result += ")"
	return result
}

// FactIndex provides efficient lookup of facts by indexed terms.
// The index maps term values to sets of fact IDs for fast retrieval.
type FactIndex struct {
	mu sync.RWMutex

	// index maps position -> term value -> set of fact IDs
	index map[int]map[string]map[string]bool

	// reverseIndex maps fact ID -> position -> term value for cleanup
	reverseIndex map[string]map[int]string
}

// NewFactIndex creates a new fact index.
func NewFactIndex() *FactIndex {
	return &FactIndex{
		index:        make(map[int]map[string]map[string]bool),
		reverseIndex: make(map[string]map[int]string),
	}
}

// Add indexes a fact for the given positions.
func (fi *FactIndex) Add(factID string, positions []int, fact *Fact) {
	fi.mu.Lock()
	defer fi.mu.Unlock()

	// Initialize reverse index for this fact
	if fi.reverseIndex[factID] == nil {
		fi.reverseIndex[factID] = make(map[int]string)
	}

	for _, pos := range positions {
		if pos >= len(fact.Terms) {
			continue
		}

		term := fact.Terms[pos]
		termStr := term.String()

		// Initialize position index if needed
		if fi.index[pos] == nil {
			fi.index[pos] = make(map[string]map[string]bool)
		}

		// Initialize term index if needed
		if fi.index[pos][termStr] == nil {
			fi.index[pos][termStr] = make(map[string]bool)
		}

		// Add fact ID to index
		fi.index[pos][termStr][factID] = true

		// Update reverse index
		fi.reverseIndex[factID][pos] = termStr
	}
}

// Remove removes a fact from the index.
func (fi *FactIndex) Remove(factID string) {
	fi.mu.Lock()
	defer fi.mu.Unlock()

	reverse, exists := fi.reverseIndex[factID]
	if !exists {
		return
	}

	// Remove from all position indexes
	for pos, termStr := range reverse {
		if posIndex, exists := fi.index[pos]; exists {
			if factSet, exists := posIndex[termStr]; exists {
				delete(factSet, factID)
				// Clean up empty term sets
				if len(factSet) == 0 {
					delete(posIndex, termStr)
				}
			}
			// Clean up empty position indexes
			if len(posIndex) == 0 {
				delete(fi.index, pos)
			}
		}
	}

	// Remove from reverse index
	delete(fi.reverseIndex, factID)
}

// Lookup finds fact IDs that match the given position and term.
func (fi *FactIndex) Lookup(position int, term Term) map[string]bool {
	fi.mu.RLock()
	defer fi.mu.RUnlock()

	posIndex, exists := fi.index[position]
	if !exists {
		return nil
	}

	termStr := term.String()
	factSet, exists := posIndex[termStr]
	if !exists {
		return nil
	}

	// Return a copy to avoid external modification
	result := make(map[string]bool)
	for id := range factSet {
		result[id] = true
	}
	return result
}

// GetAllFactIDs returns all fact IDs in the index.
func (fi *FactIndex) GetAllFactIDs() map[string]bool {
	fi.mu.RLock()
	defer fi.mu.RUnlock()

	result := make(map[string]bool)
	for _, posIndex := range fi.index {
		for _, factSet := range posIndex {
			for factID := range factSet {
				result[factID] = true
			}
		}
	}
	return result
}

// FactStore provides PLDB-style fact storage with indexing and querying capabilities.
// Facts are stored as immutable tuples and can be queried through unification.
type FactStore struct {
	mu sync.RWMutex

	// facts maps fact ID to fact instance
	facts map[string]*Fact

	// indexes provides fast lookup by indexed positions
	indexes map[string]*FactIndex // index name -> index

	// defaultIndexPositions defines which positions are indexed by default
	defaultIndexPositions []int

	// customIndexes maps index names to their position lists
	customIndexes map[string][]int
}

// NewFactStore creates a new fact store with default indexing on positions 0 and 1.
func NewFactStore() *FactStore {
	return NewFactStoreWithIndexes([]int{0, 1})
}

// NewFactStoreWithIndexes creates a new fact store with custom default indexing positions.
func NewFactStoreWithIndexes(defaultPositions []int) *FactStore {
	return &FactStore{
		facts:                 make(map[string]*Fact),
		indexes:               make(map[string]*FactIndex),
		defaultIndexPositions: defaultPositions,
		customIndexes:         make(map[string][]int),
	}
}

// Assert adds a fact to the store. If a fact with the same ID already exists, it is replaced.
func (fs *FactStore) Assert(fact *Fact) error {
	if fact == nil {
		return fmt.Errorf("cannot assert nil fact")
	}

	if len(fact.Terms) == 0 {
		return fmt.Errorf("cannot assert fact with no terms")
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Remove existing fact if it exists
	if existing, exists := fs.facts[fact.ID]; exists {
		fs.removeFromIndexes(existing)
	}

	// Add to storage
	fs.facts[fact.ID] = fact.Clone()

	// Add to indexes
	fs.addToIndexes(fact)

	return nil
}

// Retract removes a fact from the store by ID.
func (fs *FactStore) Retract(factID string) bool {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	fact, exists := fs.facts[factID]
	if !exists {
		return false
	}

	// Remove from indexes
	fs.removeFromIndexes(fact)

	// Remove from storage
	delete(fs.facts, factID)

	return true
}

// Get retrieves a fact by ID.
func (fs *FactStore) Get(factID string) (*Fact, bool) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	fact, exists := fs.facts[factID]
	if !exists {
		return nil, false
	}

	return fact.Clone(), true
}

// Query finds facts that unify with the given query terms.
// Returns a stream of matching facts with their unifications.
func (fs *FactStore) Query(ctx context.Context, queryTerms ...Term) ResultStream {
	stream := NewStream()

	go func() {
		defer stream.Close()

		fs.mu.RLock()
		facts := make([]*Fact, 0, len(fs.facts))
		for _, fact := range fs.facts {
			facts = append(facts, fact)
		}
		fs.mu.RUnlock()

		// Try to optimize query using indexes
		candidateFacts := fs.optimizeQuery(queryTerms)

		for _, fact := range candidateFacts {
			select {
			case <-ctx.Done():
				return
			default:
			}

			// Create a fresh store for unification
			store := NewLocalConstraintStore(NewGlobalConstraintBus())

			// Try to unify fact terms with query terms
			if fs.unifyTerms(store, fact.Terms, queryTerms) {
				stream.Put(ctx, store)
			}
		}
	}()

	return stream
}

// Count returns the number of facts in the store.
func (fs *FactStore) Count() int {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	return len(fs.facts)
}

// Clear removes all facts from the store.
func (fs *FactStore) Clear() {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	fs.facts = make(map[string]*Fact)
	for name := range fs.indexes {
		fs.indexes[name] = NewFactIndex()
	}
}

// AddIndex adds a custom index on the specified positions.
func (fs *FactStore) AddIndex(name string, positions []int) error {
	if name == "" {
		return fmt.Errorf("index name cannot be empty")
	}

	if len(positions) == 0 {
		return fmt.Errorf("index must specify at least one position")
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	if _, exists := fs.customIndexes[name]; exists {
		return fmt.Errorf("index %s already exists", name)
	}

	fs.customIndexes[name] = make([]int, len(positions))
	copy(fs.customIndexes[name], positions)

	// Create the index
	fs.indexes[name] = NewFactIndex()

	// Index all existing facts
	for _, fact := range fs.facts {
		fs.indexes[name].Add(fact.ID, positions, fact)
	}

	return nil
}

// RemoveIndex removes a custom index.
func (fs *FactStore) RemoveIndex(name string) bool {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if _, exists := fs.customIndexes[name]; !exists {
		return false
	}

	delete(fs.customIndexes, name)
	delete(fs.indexes, name)
	return true
}

// ListIndexes returns the names of all indexes.
func (fs *FactStore) ListIndexes() []string {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	names := make([]string, 0, len(fs.customIndexes)+1)
	names = append(names, "default")
	for name := range fs.customIndexes {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// addToIndexes adds a fact to all relevant indexes.
func (fs *FactStore) addToIndexes(fact *Fact) {
	// Add to default index
	if fs.indexes["default"] == nil {
		fs.indexes["default"] = NewFactIndex()
	}
	fs.indexes["default"].Add(fact.ID, fs.defaultIndexPositions, fact)

	// Add to custom indexes
	for name, positions := range fs.customIndexes {
		if fs.indexes[name] == nil {
			fs.indexes[name] = NewFactIndex()
		}
		fs.indexes[name].Add(fact.ID, positions, fact)
	}
}

// removeFromIndexes removes a fact from all indexes.
func (fs *FactStore) removeFromIndexes(fact *Fact) {
	for _, index := range fs.indexes {
		index.Remove(fact.ID)
	}
}

// optimizeQuery attempts to use indexes to reduce the number of facts to check.
func (fs *FactStore) optimizeQuery(queryTerms []Term) []*Fact {
	// Try to find the most selective index for the first non-variable term
	bestCandidates := make(map[string]bool)

	for _, index := range fs.indexes {
		for i, term := range queryTerms {
			if _, ok := term.(*Var); !ok { // Not a variable
				if candidates := index.Lookup(i, term); candidates != nil {
					// If we have candidates from an index, use them
					if len(bestCandidates) == 0 || len(candidates) < len(bestCandidates) {
						bestCandidates = candidates
					}
				}
			}
		}
	}

	var facts []*Fact

	// If no index candidates found, check all facts
	if len(bestCandidates) == 0 {
		fs.mu.RLock()
		defer fs.mu.RUnlock()

		facts = make([]*Fact, 0, len(fs.facts))
		for _, fact := range fs.facts {
			facts = append(facts, fact)
		}
		return facts
	}

	// Convert candidate IDs to facts
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	facts = make([]*Fact, 0, len(bestCandidates))
	for factID := range bestCandidates {
		if fact, exists := fs.facts[factID]; exists {
			facts = append(facts, fact)
		}
	}

	return facts
}

// unifyTerms attempts to unify two slices of terms.
func (fs *FactStore) unifyTerms(store ConstraintStore, factTerms, queryTerms []Term) bool {
	if len(factTerms) != len(queryTerms) {
		return false
	}

	for i := range factTerms {
		if _, success := unifyWithConstraints(factTerms[i], queryTerms[i], store); !success {
			return false
		}
	}

	return true
}
