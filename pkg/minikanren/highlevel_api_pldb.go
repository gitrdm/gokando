package minikanren

// High-level API sugar for pldb (in-memory relational database) and tabling.
// These helpers are thin wrappers to reduce boilerplate when working with
// relations, databases, and queries. They only adapt arguments and delegate
// to the existing production implementations.

import "fmt"

// MustRel creates a Relation and panics on error. Useful in examples/tests
// where arity and indexes are known constants.
func MustRel(name string, arity int, indexedCols ...int) *Relation {
	r, err := DbRel(name, arity, indexedCols...)
	if err != nil {
		panic(err)
	}
	return r
}

// DB returns a new, empty Database.
func DB() *Database { return NewDatabase() }

// Add inserts a single fact converting non-Term arguments to Atoms.
// It returns the new immutable Database instance.
func (db *Database) Add(rel *Relation, values ...interface{}) (*Database, error) {
	if rel == nil {
		return nil, fmt.Errorf("hlapi: relation cannot be nil")
	}
	if len(values) != rel.Arity() {
		return nil, fmt.Errorf("hlapi: relation %s expects %d values, got %d", rel.Name(), rel.Arity(), len(values))
	}
	terms := make([]Term, len(values))
	for i, v := range values {
		if t, ok := v.(Term); ok {
			terms[i] = t
		} else {
			terms[i] = NewAtom(v)
		}
	}
	return db.AddFact(rel, terms...)
}

// MustAdd is Add but panics on error. Convenient for compact examples.
func (db *Database) MustAdd(rel *Relation, values ...interface{}) *Database {
	ndb, err := db.Add(rel, values...)
	if err != nil {
		panic(err)
	}
	return ndb
}

// AddFacts inserts many facts at once. Each row must have length = arity.
// Elements are converted to Terms (Term as-is; otherwise wrapped as Atom).
func (db *Database) AddFacts(rel *Relation, rows ...[]interface{}) (*Database, error) {
	ndb := db
	var err error
	for _, row := range rows {
		ndb, err = ndb.Add(rel, row...)
		if err != nil {
			return nil, err
		}
	}
	return ndb, nil
}

// MustAddFacts is AddFacts but panics on error.
func (db *Database) MustAddFacts(rel *Relation, rows ...[]interface{}) *Database {
	ndb, err := db.AddFacts(rel, rows...)
	if err != nil {
		panic(err)
	}
	return ndb
}

// Q queries a relation, accepting native values or Terms.
// It converts non-Terms to Atoms before delegating to Database.Query.
func (db *Database) Q(rel *Relation, args ...interface{}) Goal {
	terms := make([]Term, len(args))
	for i, a := range args {
		if t, ok := a.(Term); ok {
			terms[i] = t
		} else {
			terms[i] = NewAtom(a)
		}
	}
	return db.Query(rel, terms...)
}

// TQ performs a tabled query using rel.Name() as the predicate identifier.
// Accepts native values or Terms and converts as needed.
func TQ(db *Database, rel *Relation, args ...interface{}) Goal {
	if db == nil || rel == nil {
		return Failure
	}
	terms := make([]Term, len(args))
	for i, a := range args {
		if t, ok := a.(Term); ok {
			terms[i] = t
		} else {
			terms[i] = NewAtom(a)
		}
	}
	return TabledQuery(db, rel, rel.Name(), terms...)
}

// TablePred returns a function that builds tabled goals for the given predicateID
// while accepting native values or Terms.
func TablePred(db *Database, rel *Relation, predicateID string) func(args ...interface{}) Goal {
	return func(args ...interface{}) Goal {
		if db == nil || rel == nil {
			return Failure
		}
		if len(args) != rel.Arity() {
			return Failure
		}
		terms := make([]Term, len(args))
		for i, a := range args {
			if t, ok := a.(Term); ok {
				terms[i] = t
			} else {
				terms[i] = NewAtom(a)
			}
		}
		return TabledQuery(db, rel, predicateID, terms...)
	}
}

// TabledDB is a convenience wrapper for WithTabledDatabase.
func TabledDB(db *Database, idPrefix string) *TabledDatabase { return WithTabledDatabase(db, idPrefix) }

// Q on a TabledDatabase: same as Database.Q but tabled automatically.
func (tdb *TabledDatabase) Q(rel *Relation, args ...interface{}) Goal {
	if tdb == nil {
		return Failure
	}
	terms := make([]Term, len(args))
	for i, a := range args {
		if t, ok := a.(Term); ok {
			terms[i] = t
		} else {
			terms[i] = NewAtom(a)
		}
	}
	return tdb.Query(rel, terms...)
}

// DisjQ builds a disjunction (logical OR) of multiple relation queries.
// Each variant is a row of arguments (native values or Terms) with arity
// matching the relation. It returns Failure if no variants are provided.
//
// Example:
//
//	// parent(gp, p) OR parent(gp, gc)
//	goal := DisjQ(db, parent, []interface{}{gp, p}, []interface{}{gp, gc})
func DisjQ(db *Database, rel *Relation, variants ...[]interface{}) Goal {
	if db == nil || rel == nil {
		return Failure
	}
	if len(variants) == 0 {
		return Failure
	}
	goals := make([]Goal, 0, len(variants))
	for _, row := range variants {
		if len(row) != rel.Arity() {
			// Skip malformed rows to keep behavior predictable in examples
			continue
		}
		goals = append(goals, db.Q(rel, row...))
	}
	if len(goals) == 0 {
		return Failure
	}
	return Disj(goals...)
}

// RecursiveTablePred provides a thin HLAPI wrapper around TabledRecursivePredicate.
// It returns a predicate constructor that accepts native values or Terms when
// called, converting non-Terms to Atoms automatically.
//
// The recursive definition uses the same signature as TabledRecursivePredicate:
// a callback that receives a self predicate (for recursive calls) and the
// instantiated call arguments as Terms, and must return the recursive case Goal.
// The base case over baseRel is handled automatically by the underlying helper.
//
// Example:
//
//	ancestor := RecursiveTablePred(db, parent, "ancestor2",
//	  func(self func(...Term) Goal, args ...Term) Goal {
//	    x, y := args[0], args[1]
//	    z := Fresh("z")
//	    return Conj(
//	      db.Query(parent, x, z), // base facts used in recursive step
//	      self(z, y),              // recursive call to tabled predicate
//	    )
//	  })
//	// Use native values or Terms at call sites
//	goal := ancestor(Fresh("x"), "carol")
func RecursiveTablePred(
	db *Database,
	baseRel *Relation,
	predicateID string,
	recursive func(self func(...Term) Goal, args ...Term) Goal,
) func(...interface{}) Goal {
	inner := TabledRecursivePredicate(db, baseRel, predicateID, recursive)
	return func(args ...interface{}) Goal {
		if baseRel == nil || len(args) != baseRel.Arity() {
			return Failure
		}
		terms := make([]Term, len(args))
		for i, a := range args {
			if t, ok := a.(Term); ok {
				terms[i] = t
			} else {
				terms[i] = NewAtom(a)
			}
		}
		return inner(terms...)
	}
}

// FactsSpec describes facts for a relation for bulk loading.
type FactsSpec struct {
	Rel  *Relation
	Rows [][]interface{}
}

// Load inserts facts for multiple relations in sequence and returns the new DB.
func Load(db *Database, specs ...FactsSpec) (*Database, error) {
	ndb := db
	var err error
	for _, s := range specs {
		if s.Rel == nil {
			return nil, fmt.Errorf("hlapi: nil relation in FactsSpec")
		}
		ndb, err = ndb.AddFacts(s.Rel, s.Rows...)
		if err != nil {
			return nil, err
		}
	}
	return ndb, nil
}

// MustLoad is Load but panics on error.
func MustLoad(db *Database, specs ...FactsSpec) *Database {
	ndb, err := Load(db, specs...)
	if err != nil {
		panic(err)
	}
	return ndb
}

// NewDBFromMap loads facts from a map keyed by relation name using the provided
// relation registry. This is convenient for multi-relation setups where data is
// produced as JSON-like maps.
//
// Example:
//
//	rels := map[string]*Relation{"employee": emp, "manager": mgr}
//	data := map[string][][]interface{}{
//	    "employee": {{"alice","eng"}, {"bob","eng"}},
//	    "manager":  {{"bob","alice"}},
//	}
//	db, _ := NewDBFromMap(rels, data)
func NewDBFromMap(relations map[string]*Relation, data map[string][][]interface{}) (*Database, error) {
	if relations == nil {
		return nil, fmt.Errorf("hlapi: relations registry is nil")
	}
	db := NewDatabase()
	var err error
	for name, rows := range data {
		rel := relations[name]
		if rel == nil {
			return nil, fmt.Errorf("hlapi: unknown relation %q in data", name)
		}
		db, err = db.AddFacts(rel, rows...)
		if err != nil {
			return nil, err
		}
	}
	return db, nil
}
