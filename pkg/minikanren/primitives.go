package minikanren

import (
	"context"
	"sync"
	"sync/atomic"
)

// Variable counter for generating unique variable IDs
var varCounter int64

// Fresh creates a new logic variable with an optional name for debugging.
// Each call to Fresh generates a variable with a globally unique ID,
// ensuring no variable conflicts even in concurrent environments.
//
// Example:
//
//	x := Fresh("x")  // Creates a variable named x
//	y := Fresh("")   // Creates an anonymous variable
func Fresh(name string) *Var {
	id := atomic.AddInt64(&varCounter, 1)
	return &Var{id: id, name: name}
}

// Eq creates a unification goal that constrains two terms to be equal.
// This is the fundamental operation in miniKanren - it attempts to make
// two terms identical by binding variables as needed.
//
// The new implementation works with constraint stores to provide
// order-independent constraint semantics. Variable bindings are
// checked against all active constraints before being accepted.
//
// Unification Rules:
//   - Atom == Atom: succeeds if atoms have the same value
//   - Var == Term: binds the variable to the term (subject to constraints)
//   - Pair == Pair: recursively unifies car and cdr
//   - Otherwise: fails
//
// Example:
//
//	x := Fresh("x")
//	goal := Eq(x, NewAtom("hello"))  // Binds x to "hello"
func Eq(term1, term2 Term) Goal {
	return func(ctx context.Context, store ConstraintStore) *Stream {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			stream := NewStream()
			stream.Close()
			return stream
		default:
		}

		// Attempt unification through the constraint store
		newStore, success := unifyWithConstraints(term1, term2, store)

		stream := NewStream()
		go func() {
			defer stream.Close()
			if success {
				stream.Put(newStore)
			}
		}()

		return stream
	}
}

// unify performs the unification algorithm.
// Returns a new substitution if unification succeeds, nil if it fails.
func unify(term1, term2 Term, sub *Substitution) *Substitution {
	// Walk both terms to their final values
	t1 := sub.Walk(term1)
	t2 := sub.Walk(term2)

	// If they're the same object, unification succeeds
	if t1.Equal(t2) {
		return sub
	}

	// If t1 is a variable, bind it to t2
	if t1.IsVar() {
		return sub.Bind(t1.(*Var), t2)
	}

	// If t2 is a variable, bind it to t1
	if t2.IsVar() {
		return sub.Bind(t2.(*Var), t1)
	}

	// If both are pairs, unify recursively
	if p1, ok1 := t1.(*Pair); ok1 {
		if p2, ok2 := t2.(*Pair); ok2 {
			// First unify the cars
			subAfterCar := unify(p1.Car(), p2.Car(), sub)
			if subAfterCar == nil {
				return nil // Car unification failed
			}

			// Then unify the cdrs with the updated substitution
			return unify(p1.Cdr(), p2.Cdr(), subAfterCar)
		}
	}

	// If we get here, unification fails
	return nil
}

// unifyWithConstraints performs unification using the constraint store system.
// Returns a new constraint store if unification succeeds, and a boolean
// indicating success. This replaces the old unify function to work with
// the order-independent constraint system.
func unifyWithConstraints(term1, term2 Term, store ConstraintStore) (ConstraintStore, bool) {
	// Clone the store to avoid modifying the original
	newStore := store.Clone()

	// Get current substitution for walking terms
	currentSub := newStore.GetSubstitution()

	// Walk both terms to their final values
	t1 := currentSub.Walk(term1)
	t2 := currentSub.Walk(term2)

	// If they're the same object, unification succeeds
	if t1.Equal(t2) {
		return newStore, true
	}

	// If t1 is a variable, bind it to t2
	if t1.IsVar() {
		err := newStore.AddBinding(t1.(*Var).id, t2)
		return newStore, err == nil
	}

	// If t2 is a variable, bind it to t1
	if t2.IsVar() {
		err := newStore.AddBinding(t2.(*Var).id, t1)
		return newStore, err == nil
	}

	// If both are pairs, unify recursively
	if p1, ok := t1.(*Pair); ok {
		if p2, ok := t2.(*Pair); ok {
			// Unify the cars
			store1, success1 := unifyWithConstraints(p1.Car(), p2.Car(), newStore)
			if !success1 {
				return store, false
			}

			// Unify the cdrs
			store2, success2 := unifyWithConstraints(p1.Cdr(), p2.Cdr(), store1)
			return store2, success2
		}
	}

	// If both are atoms, check equality
	if a1, ok := t1.(*Atom); ok {
		if a2, ok := t2.(*Atom); ok {
			return newStore, a1.Equal(a2)
		}
	}

	// Unification failed
	return store, false
}

// Conj creates a conjunction goal that requires all goals to succeed.
// The goals are evaluated sequentially, with each goal operating on
// the constraint stores produced by the previous goal.
//
// Example:
//
//	x := Fresh("x")
//	y := Fresh("y")
//	goal := Conj(Eq(x, NewAtom(1)), Eq(y, NewAtom(2)))
func Conj(goals ...Goal) Goal {
	if len(goals) == 0 {
		return Success
	}

	if len(goals) == 1 {
		return goals[0]
	}

	return func(ctx context.Context, store ConstraintStore) *Stream {
		return conjHelper(ctx, goals, store)
	}
}

// conjHelper recursively evaluates conjunction goals
func conjHelper(ctx context.Context, goals []Goal, store ConstraintStore) *Stream {
	if len(goals) == 0 {
		stream := NewStream()
		go func() {
			defer stream.Close()
			stream.Put(store)
		}()
		return stream
	}

	firstGoal := goals[0]
	restGoals := goals[1:]

	stream := NewStream()

	go func() {
		defer stream.Close()

		// Evaluate the first goal
		firstStream := firstGoal(ctx, store)

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			// Get next result from first goal
			subs, hasMore := firstStream.Take(1)
			if len(subs) == 0 {
				return // No more results
			}

			// Recursively evaluate remaining goals with each result
			restStream := conjHelper(ctx, restGoals, subs[0])

			// Forward all results from rest stream
			for {
				results, moreAvailable := restStream.Take(10)
				if len(results) == 0 {
					break
				}
				for _, result := range results {
					stream.Put(result)
				}
				if !moreAvailable {
					break
				}
			}

			if !hasMore {
				return
			}
		}
	}()

	return stream
}

// Disj creates a disjunction goal that succeeds if any of the goals succeed.
// This represents choice points in the search space. All solutions from
// all goals are included in the result stream.
//
// Example:
//
//	x := Fresh("x")
//	goal := Disj(Eq(x, NewAtom(1)), Eq(x, NewAtom(2)))  // x can be 1 or 2
func Disj(goals ...Goal) Goal {
	if len(goals) == 0 {
		return Failure
	}

	if len(goals) == 1 {
		return goals[0]
	}

	return func(ctx context.Context, store ConstraintStore) *Stream {
		stream := NewStream()

		go func() {
			defer stream.Close()

			var wg sync.WaitGroup

			// Evaluate all goals concurrently
			for _, goal := range goals {
				wg.Add(1)
				go func(g Goal) {
					defer wg.Done()

					goalStream := g(ctx, store)

					// Forward all substitutions from this goal
					for {
						select {
						case <-ctx.Done():
							return
						default:
						}

						subs, hasMore := goalStream.Take(1)
						if len(subs) == 0 {
							if !hasMore {
								break
							}
							continue
						}

						stream.Put(subs[0])
					}
				}(goal)
			}

			wg.Wait()
		}()

		return stream
	}
}

// Conde is an alias for Disj, following miniKanren naming conventions.
// "conde" represents "count" in Spanish, indicating enumeration of choices.
func Conde(goals ...Goal) Goal {
	return Disj(goals...)
}

// Run executes a goal and returns up to n solutions.
// This is the main entry point for executing miniKanren programs.
// It takes a goal that introduces one or more fresh variables and
// returns the values those variables can take.
//
// Example:
//
//	solutions := Run(5, func(q *Var) Goal {
//	    return Eq(q, NewAtom("hello"))
//	})
//	// Returns: [hello]
func Run(n int, goalFunc func(*Var) Goal) []Term {
	return RunWithContext(context.Background(), n, goalFunc)
}

// RunWithContext executes a goal with a context for cancellation and timeouts.
// This allows for better control over long-running or infinite searches.
//
// Example:
//
//	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
//	defer cancel()
//	solutions := RunWithContext(ctx, 100, func(q *Var) Goal {
//	    return someLongRunningGoal(q)
//	})
func RunWithContext(ctx context.Context, n int, goalFunc func(*Var) Goal) []Term {
	q := Fresh("q")
	goal := goalFunc(q)

	// Use shared global bus for better performance
	initialStore := NewLocalConstraintStore(GetDefaultGlobalBus())
	stream := goal(ctx, initialStore)

	solutions, _ := stream.Take(n)

	var results []Term
	for _, store := range solutions {
		value := store.GetSubstitution().DeepWalk(q)
		results = append(results, value)
	}

	return results
}

// RunStar executes a goal and returns all solutions.
// WARNING: This can run forever if the goal has infinite solutions.
// Use RunWithContext with a timeout for safer execution.
//
// Example:
//
//	solutions := RunStar(func(q *Var) Goal {
//	    return Disj(Eq(q, NewAtom(1)), Eq(q, NewAtom(2)))
//	})
//	// Returns: [1, 2]
func RunStar(goalFunc func(*Var) Goal) []Term {
	return RunStarWithContext(context.Background(), goalFunc)
}

// RunStarWithContext executes a goal and returns all solutions with context support.
func RunStarWithContext(ctx context.Context, goalFunc func(*Var) Goal) []Term {
	q := Fresh("q")
	goal := goalFunc(q)

	// Use shared global bus for better performance
	initialStore := NewLocalConstraintStore(GetDefaultGlobalBus())
	stream := goal(ctx, initialStore)

	var results []Term

	for {
		select {
		case <-ctx.Done():
			return results
		default:
		}

		solutions, hasMore := stream.Take(10) // Take in batches

		for _, store := range solutions {
			value := store.GetSubstitution().DeepWalk(q)
			results = append(results, value)
		}

		if !hasMore {
			break
		}
	}

	return results
}

// RunWithIsolation is like Run but uses an isolated constraint bus.
// Use this when you need complete constraint isolation between goals.
// Slightly slower than Run() but provides stronger isolation guarantees.
func RunWithIsolation(n int, goalFunc func(*Var) Goal) []Term {
	return RunWithIsolationContext(context.Background(), n, goalFunc)
}

// RunWithIsolationContext is like RunWithContext but uses an isolated constraint bus.
func RunWithIsolationContext(ctx context.Context, n int, goalFunc func(*Var) Goal) []Term {
	q := Fresh("q")
	goal := goalFunc(q)

	// Use pooled bus for isolation while still optimizing allocations
	bus := GetPooledGlobalBus()
	defer ReturnPooledGlobalBus(bus)

	initialStore := NewLocalConstraintStore(bus)
	defer initialStore.Shutdown() // Ensure proper cleanup

	stream := goal(ctx, initialStore)
	solutions, _ := stream.Take(n)

	var results []Term
	for _, store := range solutions {
		value := store.GetSubstitution().DeepWalk(q)
		results = append(results, value)
	}

	return results
}

// AtomFromValue creates a new atomic term from any Go value.
// This is a convenience function that's equivalent to NewAtom.
func AtomFromValue(value interface{}) *Atom {
	return NewAtom(value)
}

// List creates a list (chain of pairs) from a slice of terms.
// The list is terminated with nil (empty list).
//
// Example:
//
//	lst := List(NewAtom(1), NewAtom(2), NewAtom(3))
//	// Creates: (1 . (2 . (3 . nil)))
func List(terms ...Term) Term {
	if len(terms) == 0 {
		return NewAtom(nil) // Empty list
	}

	var result Term = NewAtom(nil)

	// Build the list from right to left
	for i := len(terms) - 1; i >= 0; i-- {
		result = NewPair(terms[i], result)
	}

	return result
}

// Appendo creates a goal that relates three lists where the third list
// is the result of appending the first two lists.
// This is a classic example of a relational operation in miniKanren.
//
// Example:
//
//	x := Fresh("x")
//	goal := Appendo(List(Atom(1), Atom(2)), List(Atom(3)), x)
//	// x will be bound to (1 2 3)
func Appendo(l1, l2, l3 Term) Goal {
	return Disj(
		// Base case: appending empty list to l2 gives l2
		Conj(Eq(l1, Nil), Eq(l2, l3)),

		// Recursive case: l1 = (a . d), l3 = (a . res), append(d, l2, res)
		func(ctx context.Context, store ConstraintStore) *Stream {
			a := Fresh("a")
			d := Fresh("d")
			res := Fresh("res")

			return Conj(
				Eq(l1, NewPair(a, d)),
				Eq(l3, NewPair(a, res)),
				Appendo(d, l2, res),
			)(ctx, store)
		},
	)
}
