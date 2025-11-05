# DCG Architecture Analysis: Fundamental Issues and Production Solution

## Current State (November 2025)

### What Works
1. **Run(0) semantics fixed**: `RunWithContext` now correctly enumerates all results when `n <= 0`
2. **Two-pass hack removed**: Eliminated `dcgInhibitKey` context type and manual base-case seeding
3. **Streaming evaluator**: Uses `batch=1` to emit answers incrementally
4. **DCG left-recursive test passes**: But **only** with base-case-first clause ordering

### What's Fragile

#### 1. **Clause Ordering Dependency**
```go
// WORKS (base case first):
TabledDCGRule("expr", Alternation(
    NonTerminal("term"),                                              // base
    Seq(NonTerminal("expr"), Terminal(NewAtom("+")), NonTerminal("term")), // recursive
))

// TIMES OUT (recursive case first):
TabledDCGRule("expr", Alternation(
    Seq(NonTerminal("expr"), Terminal(NewAtom("+")), NonTerminal("term")), // recursive
    NonTerminal("term"),                                              // base
))
```

This is an **operational dependency** in a supposedly **declarative** system. Production code should not require manual clause reordering.

#### 2. **Conde vs Disj Trade-off**
- `Conde` (lazy interleaving): Requires base-case-first ordering, deterministic but fragile
- `Disj` (eager parallel): Eliminates timeout but makes answer order non-deterministic, breaking examples

Neither is a real solution—both are band-aids.

---

## Root Cause: Circular Execution Architecture

### The Fundamental Problem

**Evaluators execute recursive calls directly, creating circular producer-consumer dependencies.**

Current architecture (simplified):
```go
func (slg *SLGEngine) Evaluate(predicate string, args []Term, ...) GoalEvaluator {
    return func(...) <-chan ConstraintStore {
        rule := dcgRegistry.m[predicate]
        
        // THIS IS THE PROBLEM: rule.body contains recursive NonTerminal calls
        goalStream := rule.body(inst0, inst1)(ctx, fresh)
        
        for {
            stores, more := goalStream.Take(1) // Drain answers
            for _, st := range stores {
                answerCh <- st  // Emit to consumers
            }
            if !more { break }
        }
    }
}
```

When `rule.body` is:
```go
Alternation(
    Seq(NonTerminal("expr"), Terminal("+"), NonTerminal("term")), // recursive
    NonTerminal("term"),                                           // base
)
```

Execution flow:
1. Evaluator starts, calls `rule.body(...)`
2. `Alternation` uses `Conde` (lazy interleaving)
3. **Recursive branch** evaluates first: calls `NonTerminal("expr")`
4. `NonTerminal` calls `slg.Evaluate("expr", ...)` → **circular dependency**
5. SLG returns cached answers **if available**, else **blocks waiting**
6. But cache is empty because **original evaluator hasn't emitted yet** (still draining)
7. **Deadlock or timeout**

With base-case-first ordering:
1. Base branch evaluates first: `NonTerminal("term")` succeeds immediately
2. Evaluator emits base answers to cache
3. Recursive branch evaluates: finds cached answers, uses them, succeeds
4. **Works, but only because we manually ordered clauses**

---

## Why Removing the Two-Pass Hack Helped (But Didn't Fix It)

The old two-pass system:
```go
// Pass 1: Inhibit self-calls, manually seed base case
ctx = context.WithValue(ctx, dcgInhibitKey{}, predicate)
// ... manual fixpoint triggering ...

// Pass 2: Allow self-calls, use seeded answers
```

This was fragile because:
- Manual orchestration of what SLG should do automatically
- Tight coupling between DCG layer and SLG internals
- Required explicit fixpoint triggers

Removing it exposed the real issue: **evaluators shouldn't contain execution, they should return descriptions.**

---

## Production-Quality Solution

### Architecture Principles

1. **Evaluators construct goal descriptions (data), not execute them**
2. **SLG orchestrates evaluation** of those descriptions
3. **No circular execution chains** within evaluator bodies
4. **Clause order irrelevant** (declarative semantics)

### Proposed Redesign

#### Current (Fragile):
```go
type GoalEvaluator func(context.Context, ConstraintStore, FreshVarSupply) <-chan ConstraintStore

// Evaluator EXECUTES rule.body directly
func (slg *SLGEngine) Evaluate(predicate string, args []Term, ...) GoalEvaluator {
    return func(...) <-chan ConstraintStore {
        rule := dcgRegistry.m[predicate]
        goalStream := rule.body(inst0, inst1)(ctx, fresh) // EXECUTES recursion
        // ... drain and emit ...
    }
}
```

#### Production (Correct):
```go
// Step 1: Evaluators return PATTERNS/DESCRIPTIONS
type GoalPattern interface {
    // Returns goals to evaluate, NOT executed streams
    Expand(ctx context.Context, store ConstraintStore) []Goal
}

// Step 2: SLG orchestrates pattern expansion
func (slg *SLGEngine) Evaluate(predicate string, args []Term, ...) GoalEvaluator {
    return func(...) <-chan ConstraintStore {
        rule := dcgRegistry.m[predicate]
        
        // Get goal DESCRIPTIONS from rule.body
        pattern := rule.body.AsPattern(inst0, inst1)
        
        // SLG expands pattern, detecting cycles
        goals := pattern.Expand(ctx, fresh)
        
        // For each goal, if it's a NonTerminal call, route through SLG
        for _, goal := range goals {
            if isNonTerminal(goal) {
                // Recursive calls go back through SLG (cycle detection, caching)
                slg.Evaluate(extractPredicate(goal), extractArgs(goal), ...)
            } else {
                // Regular goals execute normally
                goalStream := goal(ctx, fresh)
                // ... drain and emit ...
            }
        }
    }
}
```

### Key Changes Required

1. **Separate pattern construction from execution**
   - `DCGGoal` returns a description, not an executed `Goal`
   - `Alternation`, `Seq`, etc. build pattern trees

2. **SLG becomes the orchestrator**
   - Detects recursive calls within patterns
   - Routes them back through SLG (cycle detection, answer caching)
   - No evaluator body contains direct recursive execution

3. **Clause order independence**
   - All branches of `Alternation` are evaluated **by SLG's scheduler**
   - Not by Conde/Disj interleaving within evaluator
   - Base and recursive cases process in parallel via SLG's fixpoint iteration

---

## Implementation Roadmap

### Phase 1: Pattern Abstraction Layer
- [ ] Define `GoalPattern` interface
- [ ] Implement pattern constructors for `Terminal`, `Seq`, `Alternation`
- [ ] Add `AsPattern()` method to `DCGGoal`

### Phase 2: SLG Integration
- [ ] Modify `SLGEngine.Evaluate` to expand patterns
- [ ] Route recursive `NonTerminal` calls through SLG
- [ ] Implement pattern-based fixpoint iteration

### Phase 3: Remove Operational Dependencies
- [ ] Replace `Conde` in `Alternation` with SLG-orchestrated parallel evaluation
- [ ] Remove clause ordering requirements from tests
- [ ] Validate all DCG tests pass with any clause ordering

### Phase 4: Production Hardening
- [ ] Add comprehensive tests for all clause orderings
- [ ] Performance benchmarks (ensure no regression vs manual ordering)
- [ ] Documentation: explain pattern-based evaluation model

---

## Acceptance Criteria for "Production Ready"

1. ✅ **No clause ordering dependencies**: All tests pass with any ordering
2. ✅ **No timeouts needed**: Left recursion terminates via SLG fixpoint
3. ✅ **No operational workarounds**: No inhibit contexts, manual seeds, etc.
4. ✅ **Deterministic answers** (within SLG scheduling): Same query → same answers
5. ✅ **Clean separation of concerns**: DCG layer constructs patterns, SLG orchestrates

---

## Current Status

**Not production ready.** The system works for specific clause orderings but requires:
- Manual base-case-first ordering (operational knowledge)
- Conde for determinism (lazy interleaving fragility)
- Streaming evaluator (band-aid to reduce deadlock risk)

The Run(0) fix is solid and can be kept. The DCG architecture needs the fundamental redesign described above.

---

## Alternatives Considered

### Alt 1: Keep Current Architecture, Use Disj Everywhere
**Rejected**: Makes answer order non-deterministic, breaks examples and any code expecting deterministic results.

### Alt 2: Keep Current Architecture, Require Base-Case-First
**Rejected**: Not declarative, requires operational knowledge, fragile to refactoring.

### Alt 3: Use XSB/Prolog-style tabling (incremental completion)
**Possible**: Similar to proposed solution but more complex. Would still require separating pattern construction from execution.

### Alt 4: Proposed Pattern-Based Architecture
**Recommended**: Clean separation, no operational dependencies, standard SLG semantics.

---

## Conclusion

The current implementation is **not production quality**. It works for specific cases but is fragile and requires operational expertise (clause ordering). The fundamental issue is architectural: evaluators execute recursion instead of describing it.

**Production solution**: Redesign DCG/SLG integration so evaluators return lazy patterns that SLG orchestrates. This enables clause-order-independent, deterministic evaluation via standard SLG fixpoint iteration.

**Immediate recommendation**: Document current limitations clearly, add warnings about clause ordering requirements, and schedule the architectural redesign for production deployment.
