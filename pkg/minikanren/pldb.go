// Package minikanren provides pldb, an in-memory relational database for logic programming.
//
// pldb enables efficient storage and querying of ground facts with indexed access.
// Relations are defined with a name, arity, and optional column indexes.
// The Database is a persistent data structure using copy-on-write semantics,
// enabling cheap snapshots for backtracking search.
//
// Example usage:
//
//	parent := DbRel("parent", 2, 0, 1)  // Index both columns
//	db := NewDatabase()
//	db = db.AddFact(parent, NewAtom("alice"), NewAtom("bob"))
//	db = db.AddFact(parent, NewAtom("bob"), NewAtom("charlie"))
//
//	// Query: who are alice's children?
//	goal := db.Query(parent, NewAtom("alice"), Fresh("child"))
package minikanren

import (
	"context"
	"fmt"
	"hash/fnv"
	"sync"
)

// Relation represents a named relation with a fixed arity and indexed columns.
// Relations are immutable after creation.
type Relation struct {
	name    string
	arity   int
	indexes map[int]bool // set of indexed column positions (0-based)
}

// DbRel creates a new relation with the given name, arity, and optional indexed columns.
// Column indexes are 0-based. Indexing a column enables O(1) lookups for queries
// with ground terms in that position.
//
// Example:
//
//	parent := DbRel("parent", 2, 0, 1)  // Both columns indexed
//	edge := DbRel("edge", 2, 0)         // Only source column indexed
//
// Returns an error if arity is <= 0 or if any index is out of range.
func DbRel(name string, arity int, indexedCols ...int) (*Relation, error) {
	if arity <= 0 {
		return nil, fmt.Errorf("pldb: relation arity must be positive, got %d", arity)
	}
	if name == "" {
		return nil, fmt.Errorf("pldb: relation name cannot be empty")
	}

	indexes := make(map[int]bool)
	for _, col := range indexedCols {
		if col < 0 || col >= arity {
			return nil, fmt.Errorf("pldb: index column %d out of range for arity %d", col, arity)
		}
		indexes[col] = true
	}

	return &Relation{
		name:    name,
		arity:   arity,
		indexes: indexes,
	}, nil
}

// Name returns the relation's name.
func (r *Relation) Name() string {
	return r.name
}

// Arity returns the relation's arity (number of columns).
func (r *Relation) Arity() int {
	return r.arity
}

// IsIndexed returns true if the given column is indexed.
func (r *Relation) IsIndexed(col int) bool {
	return r.indexes[col]
}

// Fact represents a single row in a relation.
// Facts must be ground (contain only atoms, no variables).
// Facts are immutable after creation.
type Fact struct {
	terms []Term
	hash  uint64
}

// newFact creates a fact from ground terms, computing its hash for deduplication.
func newFact(terms []Term) (*Fact, error) {
	for i, t := range terms {
		if !isGround(t) {
			return nil, fmt.Errorf("pldb: fact term at position %d is not ground: %v", i, t)
		}
	}

	h := fnv.New64a()
	for _, t := range terms {
		// Hash the string representation for simplicity.
		// In production, a more efficient term hashing would be preferable.
		fmt.Fprintf(h, "%v|", t)
	}

	return &Fact{
		terms: terms,
		hash:  h.Sum64(),
	}, nil
}

// isGround returns true if the term contains no variables.
func isGround(t Term) bool {
	switch v := t.(type) {
	case *Var:
		return false
	case *Pair:
		return isGround(v.car) && isGround(v.cdr)
	default:
		return true
	}
}

// Equal returns true if two facts have identical terms.
func (f *Fact) Equal(other *Fact) bool {
	if len(f.terms) != len(other.terms) {
		return false
	}
	for i := range f.terms {
		if !termEqual(f.terms[i], other.terms[i]) {
			return false
		}
	}
	return true
}

// termEqual checks deep equality of two terms.
func termEqual(a, b Term) bool {
	switch va := a.(type) {
	case *Atom:
		vb, ok := b.(*Atom)
		return ok && va.value == vb.value
	case *Var:
		vb, ok := b.(*Var)
		return ok && va.id == vb.id
	case *Pair:
		vb, ok := b.(*Pair)
		return ok && termEqual(va.car, vb.car) && termEqual(va.cdr, vb.cdr)
	default:
		return false
	}
}

// factIndex maps a column value (hash of the atom) to a set of fact row IDs.
type factIndex struct {
	// map from term hash to list of fact IDs with that value in this column
	index map[uint64][]int
}

func newFactIndex() *factIndex {
	return &factIndex{
		index: make(map[uint64][]int),
	}
}

// add inserts a fact ID into the index for a given term.
func (fi *factIndex) add(term Term, factID int) {
	h := hashTerm(term)
	fi.index[h] = append(fi.index[h], factID)
}

// lookup returns all fact IDs matching the given term.
func (fi *factIndex) lookup(term Term) []int {
	h := hashTerm(term)
	return fi.index[h]
}

// clone creates a shallow copy of the index for copy-on-write.
func (fi *factIndex) clone() *factIndex {
	newIndex := make(map[uint64][]int, len(fi.index))
	for k, v := range fi.index {
		// Share the slice for now; real COW would copy on modification
		newIndex[k] = v
	}
	return &factIndex{index: newIndex}
}

// relationData holds the facts and indexes for a single relation.
type relationData struct {
	facts      []*Fact
	indexes    map[int]*factIndex // column -> index
	factSet    map[uint64]bool    // deduplication via fact hash
	tombstones map[int]bool       // deleted fact IDs (for COW removal)
}

func newRelationData(rel *Relation) *relationData {
	indexes := make(map[int]*factIndex)
	for col := range rel.indexes {
		indexes[col] = newFactIndex()
	}
	return &relationData{
		facts:      make([]*Fact, 0),
		indexes:    indexes,
		factSet:    make(map[uint64]bool),
		tombstones: make(map[int]bool),
	}
}

// clone creates a shallow copy for copy-on-write.
// Facts are immutable and can be shared. Only mutable metadata is copied.
func (rd *relationData) clone() *relationData {
	newIndexes := make(map[int]*factIndex, len(rd.indexes))
	for col, idx := range rd.indexes {
		newIndexes[col] = idx.clone()
	}
	newFactSet := make(map[uint64]bool, len(rd.factSet))
	for k, v := range rd.factSet {
		newFactSet[k] = v
	}
	newTombstones := make(map[int]bool, len(rd.tombstones))
	for k, v := range rd.tombstones {
		newTombstones[k] = v
	}
	return &relationData{
		facts:      append([]*Fact(nil), rd.facts...), // shallow copy - facts are immutable
		indexes:    newIndexes,
		factSet:    newFactSet,
		tombstones: newTombstones,
	}
}

// Database is an immutable collection of relations and their facts.
// Operations return new Database instances with copy-on-write semantics.
type Database struct {
	relations map[string]*relationData
	mu        sync.RWMutex // protects read/write for concurrent queries
}

// NewDatabase creates an empty database.
func NewDatabase() *Database {
	return &Database{
		relations: make(map[string]*relationData),
	}
}

// AddFact adds a ground fact to the relation, returning a new Database.
// Facts are deduplicated; adding the same fact twice is idempotent.
//
// Example:
//
//	db = db.AddFact(parent, NewAtom("alice"), NewAtom("bob"))
//
// Returns an error if:
//   - The relation is nil
//   - The number of terms doesn't match the relation's arity
//   - Any term is not ground (contains variables)
func (db *Database) AddFact(rel *Relation, terms ...Term) (*Database, error) {
	if rel == nil {
		return nil, fmt.Errorf("pldb: relation cannot be nil")
	}
	if len(terms) != rel.arity {
		return nil, fmt.Errorf("pldb: relation %s expects %d terms, got %d", rel.name, rel.arity, len(terms))
	}

	fact, err := newFact(terms)
	if err != nil {
		return nil, err
	}

	db.mu.Lock()
	defer db.mu.Unlock()

	// Clone the database for copy-on-write
	newDB := &Database{
		relations: make(map[string]*relationData, len(db.relations)),
	}
	for name, rd := range db.relations {
		if name == rel.name {
			newDB.relations[name] = rd.clone()
		} else {
			newDB.relations[name] = rd // share unchanged relations
		}
	}

	// Ensure relation data exists
	rd, exists := newDB.relations[rel.name]
	if !exists {
		rd = newRelationData(rel)
		newDB.relations[rel.name] = rd
	}

	// Deduplicate
	if rd.factSet[fact.hash] {
		return newDB, nil // fact already exists
	}

	// Add to facts and indexes
	factID := len(rd.facts)
	rd.facts = append(rd.facts, fact)
	rd.factSet[fact.hash] = true

	for col, idx := range rd.indexes {
		idx.add(terms[col], factID)
	}

	return newDB, nil
}

// RemoveFact removes a fact from the relation, returning a new Database.
// If the fact doesn't exist, returns the database unchanged.
//
// Uses tombstone marking for O(1) removal with stable fact IDs.
// Indexes remain valid as fact positions don't change.
func (db *Database) RemoveFact(rel *Relation, terms ...Term) (*Database, error) {
	if rel == nil {
		return nil, fmt.Errorf("pldb: relation cannot be nil")
	}
	if len(terms) != rel.arity {
		return nil, fmt.Errorf("pldb: relation %s expects %d terms, got %d", rel.name, rel.arity, len(terms))
	}

	fact, err := newFact(terms)
	if err != nil {
		return nil, err
	}

	db.mu.Lock()
	defer db.mu.Unlock()

	rd, exists := db.relations[rel.name]
	if !exists || !rd.factSet[fact.hash] {
		return db, nil // fact doesn't exist, no change
	}

	// Clone for copy-on-write
	newDB := &Database{
		relations: make(map[string]*relationData, len(db.relations)),
	}
	for name, oldRd := range db.relations {
		if name == rel.name {
			newDB.relations[name] = oldRd.clone()
		} else {
			newDB.relations[name] = oldRd
		}
	}

	newRd := newDB.relations[rel.name]

	// Find and tombstone the fact (O(n) worst-case, but maintains stable IDs)
	for i, f := range newRd.facts {
		if !newRd.tombstones[i] && f.Equal(fact) {
			// Mark as deleted - fact ID remains stable
			newRd.tombstones[i] = true
			delete(newRd.factSet, fact.hash)
			break
		}
	}

	return newDB, nil
}

// FactCount returns the number of non-deleted facts in the given relation.
func (db *Database) FactCount(rel *Relation) int {
	if rel == nil {
		return 0
	}

	db.mu.RLock()
	defer db.mu.RUnlock()

	rd, exists := db.relations[rel.name]
	if !exists {
		return 0
	}

	count := 0
	for i := range rd.facts {
		if !rd.tombstones[i] {
			count++
		}
	}
	return count
}

// AllFacts returns all non-deleted facts for a relation as a slice of term slices.
// Returns nil if the relation has no facts.
func (db *Database) AllFacts(rel *Relation) [][]Term {
	if rel == nil {
		return nil
	}

	db.mu.RLock()
	defer db.mu.RUnlock()

	rd, exists := db.relations[rel.name]
	if !exists {
		return nil
	}

	result := make([][]Term, 0, len(rd.facts))
	for i, f := range rd.facts {
		if !rd.tombstones[i] {
			result = append(result, f.terms)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// Query returns a Goal that unifies the given pattern with all matching facts.
// The pattern may contain variables, which will be unified with fact values.
//
// Query uses index selection heuristics:
//   - If any term is ground and indexed, use that index for O(1) lookup
//   - Otherwise, scan all facts (O(n))
//   - Repeated variables are checked for consistency
//
// Example:
//
//	// Find all of alice's children
//	goal := db.Query(parent, NewAtom("alice"), Fresh("child"))
//
//	// Find all parent-child pairs
//	goal := db.Query(parent, Fresh("p"), Fresh("c"))
//
//	// Find self-loops (repeated variable)
//	goal := db.Query(edge, Fresh("x"), Fresh("x"))
func (db *Database) Query(rel *Relation, pattern ...Term) Goal {
	if rel == nil {
		return Failure
	}
	if len(pattern) != rel.arity {
		return Failure
	}

	return func(ctx context.Context, store ConstraintStore) *Stream {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			stream := NewStream()
			stream.Close()
			return stream
		default:
		}

		db.mu.RLock()
		defer db.mu.RUnlock()

		rd, exists := db.relations[rel.name]
		if !exists {
			return Failure(ctx, store) // no facts, goal fails
		}

		// Select search strategy based on ground terms and indexes
		facts := selectFacts(rd, rel, pattern)

		// Build a disjunction of unifications for all matching facts
		goals := make([]Goal, 0, len(facts))
		for _, fact := range facts {
			goals = append(goals, unifyFactGoal(fact, pattern))
		}

		if len(goals) == 0 {
			return Failure(ctx, store)
		}

		// Use Disj to try each fact
		return Disj(goals...)(ctx, store)
	}
}

// selectFacts chooses facts to scan based on index availability and pattern.
// Skips tombstoned (deleted) facts.
func selectFacts(rd *relationData, rel *Relation, pattern []Term) []*Fact {
	// Find the first indexed ground term for O(1) lookup
	for col, term := range pattern {
		if isGround(term) && rel.IsIndexed(col) {
			if idx, exists := rd.indexes[col]; exists {
				factIDs := idx.lookup(term)
				facts := make([]*Fact, 0, len(factIDs))
				for _, id := range factIDs {
					if id < len(rd.facts) && !rd.tombstones[id] {
						facts = append(facts, rd.facts[id])
					}
				}
				return facts
			}
		}
	}

	// No suitable index; scan all non-deleted facts
	facts := make([]*Fact, 0, len(rd.facts))
	for i, f := range rd.facts {
		if !rd.tombstones[i] {
			facts = append(facts, f)
		}
	}
	return facts
}

// unifyFactGoal returns a goal that unifies a fact's terms with a pattern.
// Handles repeated variables (e.g., edge(X, X) requires same value in both positions).
func unifyFactGoal(fact *Fact, pattern []Term) Goal {
	// Build a conjunction of unifications for each column
	goals := make([]Goal, 0, len(pattern))

	// Track variable occurrences to enforce repeated variable constraints
	varPositions := make(map[int64][]int) // var ID -> list of positions
	for i, term := range pattern {
		if v, ok := term.(*Var); ok {
			varPositions[v.id] = append(varPositions[v.id], i)
		}
	}

	// For each variable that appears multiple times, enforce equality
	seenVars := make(map[int64]bool)
	for i, patternTerm := range pattern {
		if v, ok := patternTerm.(*Var); ok {
			positions := varPositions[v.id]
			if len(positions) > 1 && !seenVars[v.id] {
				// First occurrence: unify normally
				goals = append(goals, Eq(patternTerm, fact.terms[i]))
				seenVars[v.id] = true

				// Subsequent occurrences: check that fact values are equal
				for j := 1; j < len(positions); j++ {
					pos := positions[j]
					if !termEqual(fact.terms[i], fact.terms[pos]) {
						// Fact doesn't satisfy repeated variable constraint
						return Failure
					}
				}
			} else if !seenVars[v.id] {
				// Single occurrence
				goals = append(goals, Eq(patternTerm, fact.terms[i]))
				seenVars[v.id] = true
			}
			// Skip already-processed repeated variables
		} else {
			// Ground term: unify directly
			goals = append(goals, Eq(patternTerm, fact.terms[i]))
		}
	}

	if len(goals) == 0 {
		return Success
	}
	return Conj(goals...)
}
