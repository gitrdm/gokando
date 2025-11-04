package minikanren

import (
	"context"
)

// Rembero creates a goal that relates an element to input and output lists,
// where the output list is the input list with the first occurrence of the element removed.
// This operation works bidirectionally:
//   - Given element and inputList, computes outputList
//   - Given element and outputList, can generate possible inputLists
//   - Given inputList and outputList, can determine what element was removed
//
// Example:
//
//	// Remove 2 from [1,2,3]: output is [1,3]
//	Rembero(NewAtom(2), List(NewAtom(1), NewAtom(2), NewAtom(3)), output)
//
//	// Generate lists that when 2 is removed give [1,3]
//	Rembero(NewAtom(2), input, List(NewAtom(1), NewAtom(3)))
func Rembero(element, inputList, outputList Term) Goal {
	return Disj(
		// Base case: input is (element . rest), output is rest
		func(ctx context.Context, store ConstraintStore) *Stream {
			rest := Fresh("rest")
			return Conj(
				Eq(inputList, NewPair(element, rest)),
				Eq(outputList, rest),
			)(ctx, store)
		},

		// Recursive case: input is (head . tail), output is (head . recursiveOutput)
		// where element is removed from tail giving recursiveOutput
		// Note: head != element is implicit - if they were equal, first branch would succeed
		func(ctx context.Context, store ConstraintStore) *Stream {
			head := Fresh("head")
			tail := Fresh("tail")
			recursiveOutput := Fresh("recursiveOutput")

			return Conj(
				Eq(inputList, NewPair(head, tail)),
				Eq(outputList, NewPair(head, recursiveOutput)),
				Rembero(element, tail, recursiveOutput),
			)(ctx, store)
		},
	)
}

// SameLengtho creates a goal that succeeds if two lists have the same length.
// This is used to constrain search and prevent divergence in relations like Reverso
// where Appendo could otherwise generate arbitrarily long lists.
//
// This relation is bidirectional: it can verify equality of lengths or constrain
// one list's length based on another's.
func SameLengtho(xs, ys Term) Goal {
	return Conde(
		// Base case: both empty
		Conj(
			Eq(xs, Nil),
			Eq(ys, Nil),
		),

		// Recursive case: both non-empty with same-length tails
		func(ctx context.Context, store ConstraintStore) *Stream {
			x := Fresh("x")
			xsTail := Fresh("xs'")
			y := Fresh("y")
			ysTail := Fresh("ys'")

			return Conj(
				Eq(xs, NewPair(x, xsTail)),
				Eq(ys, NewPair(y, ysTail)),
				SameLengtho(xsTail, ysTail),
			)(ctx, store)
		},
	)
}

// reversoCore implements the core reversal logic without length constraints.
// This is separated to allow Reverso to impose length equality first.
func reversoCore(list, reversed Term) Goal {
	return Conde(
		// Base case: empty reverses to empty
		Conj(
			Eq(list, Nil),
			Eq(reversed, Nil),
		),

		// Recursive case: reverse tail, then append head as singleton at the end
		func(ctx context.Context, store ConstraintStore) *Stream {
			head := Fresh("head")
			tail := Fresh("tail")
			revTail := Fresh("revTail")

			return Conj(
				Eq(list, NewPair(head, tail)),
				reversoCore(tail, revTail),
				Appendo(revTail, NewPair(head, Nil), reversed),
			)(ctx, store)
		},
	)
}

// Reverso creates a goal that relates a list to its reverse.
// This operation works bidirectionally and terminates in all modes by first
// constraining both lists to have the same length (preventing Appendo from diverging).
//
// Implementation follows the StackOverflow solution for core.logic's reverso:
// https://stackoverflow.com/questions/70159176/non-termination-when-query-variable-is-on-a-specific-position
//
// Example:
//
//	// Reverse [1,2,3] to get [3,2,1]
//	Reverso(List(NewAtom(1), NewAtom(2), NewAtom(3)), reversed)
//
//	// Verify [1,2,3] and [3,2,1] are reverses
//	Reverso(List(NewAtom(1), NewAtom(2), NewAtom(3)), List(NewAtom(3), NewAtom(2), NewAtom(1)))
//
//	// Works in backward mode: find list that reverses to [3,2,1]
//	Reverso(list, List(NewAtom(3), NewAtom(2), NewAtom(1)))
func Reverso(list, reversed Term) Goal {
	return Conj(
		SameLengtho(list, reversed),
		reversoCore(list, reversed),
	)
}

// Permuteo creates a goal that relates a list to one of its permutations.
// This operation generates all permutations when 'permutation' is a variable,
// or verifies if 'permutation' is a valid permutation of 'list'.
//
// Note: This generates n! permutations for a list of length n.
// Use with caution for lists longer than ~8-10 elements.
//
// Example:
//
//	// Generate all permutations of [1,2,3]
//	Permuteo(List(NewAtom(1), NewAtom(2), NewAtom(3)), perm)
//
//	// Verify [3,1,2] is a permutation of [1,2,3]
//	Permuteo(List(NewAtom(1), NewAtom(2), NewAtom(3)), List(NewAtom(3), NewAtom(1), NewAtom(2)))
func Permuteo(list, permutation Term) Goal {
	return Disj(
		// Base case: empty list has only the empty permutation
		Conj(
			Eq(list, Nil),
			Eq(permutation, Nil),
		),

		// Recursive case: remove an element from list, permute the rest,
		// then insert the element at the front of the permutation
		func(ctx context.Context, store ConstraintStore) *Stream {
			element := Fresh("element")
			restList := Fresh("restList")
			restPerm := Fresh("restPerm")

			return Conj(
				// Remove one element from the list
				Rembero(element, list, restList),
				// Permute the rest
				Permuteo(restList, restPerm),
				// The permutation starts with the removed element
				Eq(permutation, NewPair(element, restPerm)),
			)(ctx, store)
		},
	)
}

// Subseto creates a goal that relates two lists where the first is a subset of the second.
// For subset generation, each element from the superset appears at most once in any subset.
// For subset verification, checks if all elements in subset appear in superset.
//
// Note: When generating subsets, produces 2^n subsets for a list of length n.
//
// Example:
//
//	// Verify [1,3] is a subset of [1,2,3,4]
//	Subseto(List(NewAtom(1), NewAtom(3)), List(NewAtom(1), NewAtom(2), NewAtom(3), NewAtom(4)))
//
//	// Generate all subsets of [1,2,3] (produces 8 subsets: [], [1], [2], [3], [1,2], [1,3], [2,3], [1,2,3])
//	Subseto(subset, List(NewAtom(1), NewAtom(2), NewAtom(3)))
func Subseto(subset, superset Term) Goal {
	return Conde(
		// Base case: empty superset means subset must be empty
		Conj(
			Eq(superset, Nil),
			Eq(subset, Nil),
		),

		// Recursive case: superset = (head . tail)
		// Either include head in subset or don't
		func(ctx context.Context, store ConstraintStore) *Stream {
			head := Fresh("head")
			tail := Fresh("tail")
			subsetTail := Fresh("subsetTail")

			return Conj(
				Eq(superset, NewPair(head, tail)),
				Conde(
					// Option 1: Include head in subset
					Conj(
						Eq(subset, NewPair(head, subsetTail)),
						Subseto(subsetTail, tail),
					),
					// Option 2: Don't include head in subset
					Subseto(subset, tail),
				),
			)(ctx, store)
		},
	)
}

// Lengtho creates a goal that relates a list to its length.
// The length is represented as a Peano number (nested pairs): 0 = nil, S(n) = (s . n)
// This operation works bidirectionally:
//   - Given list, computes length
//   - Given length, can verify if a list has that length
//   - Can generate lists of a specific length (with unbound elements)
//
// For working with integer lengths, use LengthoInt instead.
//
// Example:
//
//	// Get length of [1,2,3] as Peano number
//	Lengtho(List(NewAtom(1), NewAtom(2), NewAtom(3)), length)
//	// length will be (s . (s . (s . nil)))
//
//	// Verify a list has length 3
//	three := NewPair(NewAtom("s"), NewPair(NewAtom("s"), NewPair(NewAtom("s"), Nil)))
//	Lengtho(someList, three)
func Lengtho(list, length Term) Goal {
	return Disj(
		// Base case: empty list has length 0 (represented as Nil)
		Conj(
			Eq(list, Nil),
			Eq(length, Nil),
		),

		// Recursive case: list is (head . tail)
		// length is S(restLength) where restLength is the length of tail
		func(ctx context.Context, store ConstraintStore) *Stream {
			head := Fresh("head")
			tail := Fresh("tail")
			restLength := Fresh("restLength")

			return Conj(
				Eq(list, NewPair(head, tail)),
				Eq(length, NewPair(NewAtom("s"), restLength)),
				Lengtho(tail, restLength),
			)(ctx, store)
		},
	)
}

// LengthoInt creates a goal that relates a list to its length as an integer.
// This is a convenience wrapper around Lengtho that works with Go integers
// instead of Peano numbers.
//
// Example:
//
//	// Get length of [1,2,3] as integer
//	LengthoInt(List(NewAtom(1), NewAtom(2), NewAtom(3)), length)
//	// length will be NewAtom(3)
//
//	// Verify a list has length 3
//	LengthoInt(someList, NewAtom(3))
func LengthoInt(list, length Term) Goal {
	return func(ctx context.Context, store ConstraintStore) *Stream {
		// If length is ground and an integer, convert to Peano
		if atom, ok := length.(*Atom); ok {
			if n, ok := atom.value.(int); ok {
				peano := intToPeano(n)
				return Lengtho(list, peano)(ctx, store)
			}
		}

		// Otherwise, compute Peano length and convert back
		peanoLength := Fresh("peanoLength")
		return Conj(
			Lengtho(list, peanoLength),
			func(ctx context.Context, store ConstraintStore) *Stream {
				// DeepWalk to fully resolve the Peano structure
				walked := store.GetSubstitution().DeepWalk(peanoLength)
				n := peanoToInt(walked)
				return Eq(length, NewAtom(n))(ctx, store)
			},
		)(ctx, store)
	}
}

// intToPeano converts a non-negative integer to a Peano number.
// 0 -> Nil
// n -> (s . intToPeano(n-1))
func intToPeano(n int) Term {
	if n <= 0 {
		return Nil
	}
	return NewPair(NewAtom("s"), intToPeano(n-1))
}

// peanoToInt converts a Peano number to an integer.
// Nil -> 0
// (s . rest) -> 1 + peanoToInt(rest)
// Anything else -> 0
func peanoToInt(t Term) int {
	if t == Nil {
		return 0
	}
	if pair, ok := t.(*Pair); ok {
		if atom, ok := pair.car.(*Atom); ok {
			if s, ok := atom.value.(string); ok && s == "s" {
				return 1 + peanoToInt(pair.cdr)
			}
		}
	}
	return 0
}

// Flatteno creates a goal that relates a nested list structure to its flattened form.
// This operation converts a tree-like structure of nested lists into a flat list.
// Atoms are preserved as singleton elements in the result.
//
// Example:
//
//	// Flatten [[1,2],[3,[4,5]]] to [1,2,3,4,5]
//	nested := List(List(NewAtom(1), NewAtom(2)), List(NewAtom(3), List(NewAtom(4), NewAtom(5))))
//	Flatteno(nested, flat)
func Flatteno(nested, flat Term) Goal {
	return func(ctx context.Context, store ConstraintStore) *Stream {
		walked := store.GetSubstitution().Walk(nested)

		// Case 1: Nil flattens to Nil
		if walked.Equal(Nil) {
			return Eq(flat, Nil)(ctx, store)
		}

		// Case 2: Atom becomes singleton list
		if _, ok := walked.(*Pair); !ok {
			return Eq(flat, NewPair(nested, Nil))(ctx, store)
		}

		// Case 3: Pair - we need to flatten the head element and recurse on the tail list
		// For a list [a, b, c], this is pair(a, pair(b, pair(c, nil)))
		// We flatten 'a' to get its flat form, then flatten the rest [b,c] and append
		head := Fresh("head")
		tail := Fresh("tail")
		flatHead := Fresh("flatHead")
		flatTail := Fresh("flatTail")

		return Conj(
			Eq(nested, NewPair(head, tail)),
			Flatteno(head, flatHead), // Flatten the element
			Flatteno(tail, flatTail), // Flatten the rest of the list
			Appendo(flatHead, flatTail, flat),
		)(ctx, store)
	}
}

// Distincto creates a goal that succeeds if all elements in a list are distinct.
// This is useful for constraint problems where uniqueness is required.
//
// Example:
//
//	// Verify [1,2,3] has all distinct elements
//	Distincto(List(NewAtom(1), NewAtom(2), NewAtom(3)))
//
//	// Verify [1,2,1] fails (duplicate 1)
//	Distincto(List(NewAtom(1), NewAtom(2), NewAtom(1))) // fails
func Distincto(list Term) Goal {
	return Disj(
		// Base case: empty list has all distinct elements
		Eq(list, Nil),

		// Recursive case: list is (head . tail)
		// head must not be in tail, and tail must have all distinct elements
		func(ctx context.Context, store ConstraintStore) *Stream {
			head := Fresh("head")
			tail := Fresh("tail")

			return Conj(
				Eq(list, NewPair(head, tail)),
				Noto(Membero(head, tail)), // head NOT in tail
				Distincto(tail),
			)(ctx, store)
		},
	)
}

// Noto creates a goal that succeeds if the given goal fails.
// This is the negation operator for goals.
//
// Note: This uses negation-as-failure, which is not purely relational.
// The goal must be ground (fully instantiated) for negation to be sound.
func Noto(goal Goal) Goal {
	return func(ctx context.Context, store ConstraintStore) *Stream {
		stream := NewStream()
		go func() {
			defer stream.Close()

			// It's crucial to check for context cancellation *before* starting the
			// sub-goal, and especially after any blocking operations.
			select {
			case <-ctx.Done():
				return // Parent context was cancelled.
			default:
			}

			testStream := goal(ctx, store)
			// Take(1) blocks until either a result is available or the underlying
			// stream is closed.
			results, hasMore := testStream.Take(1)

			// After the blocking call, we must check the context again. If the
			// test runner timed out while we were waiting, we need to exit
			// immediately to prevent the goroutine from leaking.
			select {
			case <-ctx.Done():
				return // Parent context was cancelled.
			default:
			}

			// If the goal produces any solutions (len > 0), the negation fails.
			// We simply return, and the empty stream indicates failure.
			if len(results) > 0 {
				return
			}

			// If the goal produced no results (len == 0) AND the stream is now
			// exhausted (!hasMore), it means the goal has failed completely.
			// The negation succeeds, so we yield the original constraint store.
			if !hasMore {
				stream.Put(store)
			}
			// The final case is len(results) == 0 and hasMore == true. This means
			// the sub-goal's stream is open but hasn't produced a result yet.
			// A result *could* still appear, so the negation cannot succeed.
			// We do nothing and let the stream close, correctly indicating failure.
		}()
		return stream
	}
}
