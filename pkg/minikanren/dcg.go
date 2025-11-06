// Package minikanren provides DCG (Definite Clause Grammar) support with
// SLG resolution and pattern-based evaluation.
//
// # Pattern-Based DCG Architecture
//
// DCGs in this package implement a pattern-based architecture where grammar
// rules return DESCRIPTIONS of goals rather than executing them directly.
// This design enables:
//   - Clause-order independence (declarative semantics)
//   - Left recursion via SLG fixpoint iteration
//   - Clean separation between grammar construction and evaluation
//
// # Difference Lists
//
// DCGs use difference lists to represent sequences:
//   - Input list s0, output list s1
//   - Sequence [a,b,c] represented as: s0 = [a,b,c|s1]
//   - Empty sequence: s0 = s1
//
// # Pattern Types
//
// DCG patterns construct goal descriptions without executing them:
//   - Terminal(t): Matches single token (s0=[t|s1])
//   - Seq(p1, p2): Sequential composition
//   - Alternation(p1, p2, ...): Choice (declarative, order-independent)
//   - NonTerminal(engine, name): Reference to defined rule
//
// # SLG Integration
//
// When evaluating rules, the SLG engine orchestrates pattern expansion:
//  1. Rule bodies return GoalPattern descriptions
//  2. SLG expands patterns to concrete Goals
//  3. Recursive NonTerminal calls route through SLG (cycle detection, caching)
//  4. No circular execution chains within pattern constructors
//
// # Example: Left-Recursive Grammar
//
//	engine := NewSLGEngine(nil)
//	DefineRule("expr", Alternation(
//	    NonTerminal(engine, "term"),
//	    Seq(NonTerminal(engine, "expr"), Terminal(NewAtom("+")), NonTerminal(engine, "term")),
//	))
//	DefineRule("term", Terminal(NewAtom("1")))
//
//	// Parse with SLG tabling
//	results := Run(5, func(q *Var) Goal {
//	    input := MakeList(NewAtom("1"), NewAtom("+"), NewAtom("1"))
//	    rest := Fresh("rest")
//	    return Conj(
//	        ParseWithSLG(engine, "expr", input, rest),
//	        Eq(q, rest),
//	    )
//	})
package minikanren

import (
	"context"
	"fmt"
	"sync"
)

// GoalPattern represents a pattern that can be expanded into concrete goals.
// Patterns are descriptions of computation, not executions. SLG orchestrates
// their evaluation.
type GoalPattern interface {
	// Expand converts the pattern into a concrete Goal given input/output terms.
	// The returned Goal performs the actual unification and constraint propagation.
	Expand(s0, s1 Term) Goal
}

// DCGGoal represents a DCG goal as a pattern-based constructor.
// It is an alias for GoalPattern for clarity in DCG contexts.
type DCGGoal = GoalPattern

type dcgRule struct {
	name string      // predicate name for debugging
	body GoalPattern // pattern description (not executed)
}

type dcgRegistryType struct {
	mu sync.RWMutex
	m  map[string]*dcgRule // predicate name → rule
}

var globalDCGRegistry = &dcgRegistryType{
	m: make(map[string]*dcgRule),
}

// DefineRule registers a DCG rule in the global registry.
//
// Rules are pattern descriptions that SLG will orchestrate. Clause order
// in Alternation patterns does not affect semantics.
//
// Example:
//
//	DefineRule("noun", Alternation(
//	    Terminal(NewAtom("cat")),
//	    Terminal(NewAtom("dog")),
//	))
func DefineRule(name string, body GoalPattern) {
	globalDCGRegistry.mu.Lock()
	defer globalDCGRegistry.mu.Unlock()
	globalDCGRegistry.m[name] = &dcgRule{name: name, body: body}
}

// ParseWithSLG parses input using a defined DCG rule with SLG tabling.
//
// This function creates a Goal that can be used with Run/RunWithContext.
// The SLG engine handles cycle detection and fixpoint iteration for
// left-recursive grammars.
//
// Parameters:
//   - engine: SLGEngine instance for tabling
//   - ruleName: Name of the DCG rule to parse with
//   - input: Input list to parse
//   - output: Variable representing remaining unparsed input (use Fresh())
//
// Returns a Goal that succeeds for each valid parse of the input.
//
// Example:
//
//	engine := NewSLGEngine(nil)
//	DefineRule("digit", Alternation(Terminal(NewAtom("0")), Terminal(NewAtom("1"))))
//	results := Run(5, func(q *Var) Goal {
//	    input := MakeList(NewAtom("1"), NewAtom("0"))
//	    rest := Fresh("rest")
//	    return Conj(
//	        ParseWithSLG(engine, "digit", input, rest),
//	        Eq(q, rest),
//	    )
//	})
func ParseWithSLG(engine *SLGEngine, ruleName string, input, output Term) Goal {
	return func(ctx context.Context, store ConstraintStore) *Stream {

		globalDCGRegistry.mu.RLock()
		rule, exists := globalDCGRegistry.m[ruleName]
		globalDCGRegistry.mu.RUnlock()

		if !exists {
			stream := NewStream()
			stream.Close()
			return stream
		}

		// Create CallPattern for this DCG predicate
		pattern := NewCallPattern(fmt.Sprintf("dcg:%s", ruleName), []Term{input, output})

		// Create evaluator that expands the rule's pattern
		evaluator := func(evalCtx context.Context, answers chan<- map[int64]Term) error {
			// DON'T close answers channel - SLG engine handles it

			// Expand rule body pattern to Goal
			goal := rule.body.Expand(input, output)

			// Evaluate goal with a FRESH store (not the original store)
			freshStore := NewLocalConstraintStore(NewGlobalConstraintBus())
			goalStream := goal(evalCtx, freshStore)

			// Consume stream - use Take(1) to avoid blocking when fewer results available
			storeCount := 0

			for {
				select {
				case <-evalCtx.Done():
					return evalCtx.Err()
				default:
				}

				stores, more := goalStream.Take(1)

				// Process all stores returned by Take
				for _, resultStore := range stores {
					// Temporary targeted trace to diagnose left-recursive hang in TestLeftRecursion_BaseFirst
					// Remove once stabilized
					// tracing removed
					storeCount++
					answerMap := make(map[int64]Term)

					// Extract bindings for variables in input/output
					if v, ok := input.(*Var); ok {
						if binding := resultStore.GetBinding(v.id); binding != nil {
							answerMap[v.id] = binding
						}
					}
					if v, ok := output.(*Var); ok {
						if binding := resultStore.GetBinding(v.id); binding != nil {
							answerMap[v.id] = binding
						}
					}

					// Always send answer, even if empty (for ground queries)
					select {
					case answers <- answerMap:
					case <-evalCtx.Done():
						return evalCtx.Err()
					}
				}

				// Check if stream has more results
				if !more {
					// tracing removed
					return nil
				}
			}
		}
		// Check if we have a parent context (for debugging)
		if parent := ctx.Value(interface{}(struct{ name string }{"slgParent"})); parent != nil {
		}
		// Evaluate through SLG
		answerCh, err := engine.Evaluate(ctx, pattern, evaluator)
		if err != nil {
			stream := NewStream()
			stream.Close()
			return stream
		}

		// Convert answer channel back to Stream
		// Use same pattern as streamFromAnswers in pldb_slg.go
		stream := NewStream()
		go func() {
			defer stream.Close()
			answerCount := 0
			for {
				select {
				case answer, ok := <-answerCh:
					if !ok {
						return
					}
					answerCount++
					// tracing removed

					// Build conjunction of Eq goals to unify variables with answers
					// Use the pattern approach from pldb_slg.go
					goals := make([]Goal, 0, len(answer))

					// Check input/output to see which were variables and need binding
					if v, ok := input.(*Var); ok {
						if binding, exists := answer[v.id]; exists {
							goals = append(goals, Eq(v, binding))
						}
					}
					if v, ok := output.(*Var); ok {
						if binding, exists := answer[v.id]; exists {
							goals = append(goals, Eq(v, binding))
						}
					}

					// Execute conjunction on original store
					var unified *Stream
					if len(goals) > 0 {
						unified = Conj(goals...)(ctx, store)
					} else {
						// No variables or no bindings - answer matches as-is
						unified = Success(ctx, store)
					}

					if unified != nil {
						stores, _ := unified.Take(1)
						for _, s := range stores {
							stream.Put(s)
						}
					}

				case <-ctx.Done():
					return
				}
			}
		}()

		return stream
	}
}

// terminalPattern matches a single token from the input list.
type terminalPattern struct {
	token Term
}

func (p *terminalPattern) Expand(s0, s1 Term) Goal {
	return func(ctx context.Context, store ConstraintStore) *Stream {
		// s0 = [token | s1]
		// Unify s0 with a pair whose car is the token and cdr is s1
		head := Fresh("head")

		// Build the list structure: s0 = (head . s1)
		pairGoal := Eq(s0, NewPair(head, s1))
		// Then unify head with the expected token
		tokenGoal := Eq(head, p.token)

		// Combine both goals
		return Conj(pairGoal, tokenGoal)(ctx, store)
	}
}

// Terminal creates a pattern that matches a single token.
//
// The pattern succeeds when the input list starts with the specified token.
// Consumes one element from s0, leaving s1 as the tail.
//
// Example:
//
//	DefineRule("digit1", Terminal(NewAtom("1"))) // matches "1"
func Terminal(token Term) GoalPattern {
	return &terminalPattern{token: token}
}

// seqPattern sequences two patterns, threading difference lists.
type seqPattern struct {
	first, second GoalPattern
}

func (p *seqPattern) Expand(s0, s1 Term) Goal {
	// Thread through intermediate variable
	sMid := Fresh("seq")
	return Conj(
		p.first.Expand(s0, sMid),
		p.second.Expand(sMid, s1),
	)
}

// Seq creates a pattern that sequences two or more patterns.
//
// The first pattern consumes from s0 to an intermediate state,
// then the second pattern consumes from that state to s1, and so on.
//
// Example:
//
//	DefineRule("twoDigits", Seq(
//	    Terminal(NewAtom("1")),
//	    Terminal(NewAtom("2")),
//	)) // matches "1" followed by "2"
func Seq(patterns ...GoalPattern) GoalPattern {
	if len(patterns) == 0 {
		panic("dcg: Seq requires at least one pattern")
	}
	if len(patterns) == 1 {
		return patterns[0]
	}

	result := patterns[0]
	for _, p := range patterns[1:] {
		result = &seqPattern{first: result, second: p}
	}
	return result
}

// alternationPattern offers a choice between patterns.
type alternationPattern struct {
	branches []GoalPattern
}

func (p *alternationPattern) Expand(s0, s1 Term) Goal {
	goals := make([]Goal, len(p.branches))
	for i, branch := range p.branches {
		goals[i] = branch.Expand(s0, s1)
	}
	// Use Disj for concurrent evaluation - critical for handling recursive branches
	// Disj evaluates each branch in its own goroutine, preventing one blocked branch
	// from blocking others. This enables proper SLG fixpoint iteration for left recursion.
	return Disj(goals...)
}

// Alternation creates a pattern offering a choice between alternatives.
//
// Each branch is tried, and all successful parses are returned.
// Clause order does not affect declarative semantics when used with SLG.
//
// Example:
//
//	DefineRule("digit", Alternation(
//	    Terminal(NewAtom("0")),
//	    Terminal(NewAtom("1")),
//	)) // matches either "0" or "1"
func Alternation(patterns ...GoalPattern) GoalPattern {
	if len(patterns) == 0 {
		panic("dcg: Alternation requires at least one pattern")
	}
	return &alternationPattern{branches: patterns}
}

// nonTerminalPattern references another DCG rule.
type nonTerminalPattern struct {
	ruleName string
	engine   *SLGEngine // must be provided at construction time
}

func (p *nonTerminalPattern) Expand(s0, s1 Term) Goal {
	// Delegate to ParseWithSLG which handles recursive calls through SLG
	return ParseWithSLG(p.engine, p.ruleName, s0, s1)
}

// NonTerminal creates a pattern that references another DCG rule by name.
//
// IMPORTANT: Requires an SLGEngine to handle recursive calls and cycle detection.
// The engine parameter must be the same SLGEngine used for parsing.
//
// This is the key to left recursion: recursive calls go through SLG,
// which maintains an answer trie and performs fixpoint iteration.
//
// Example:
//
//	engine := NewSLGEngine(nil)
//	DefineRule("expr", Alternation(
//	    NonTerminal(engine, "term"),
//	    Seq(NonTerminal(engine, "expr"), Terminal(NewAtom("+")), NonTerminal(engine, "term")),
//	)) // Left-recursive expression grammar
func NonTerminal(engine *SLGEngine, ruleName string) GoalPattern {
	if engine == nil {
		panic("dcg: NonTerminal requires non-nil SLGEngine")
	}
	return &nonTerminalPattern{ruleName: ruleName, engine: engine}
}
