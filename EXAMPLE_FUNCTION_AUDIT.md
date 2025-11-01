# GoKanDo Example Function Coverage Audit
**Date:** October 31, 2025  
**Branch:** go-to-core  
**Purpose:** Comprehensive review of end-user API functions and their Example function coverage

## Scope: True End-User Functions Only

This audit focuses on **functions that typical users write in their programs**, excluding:
- Internal constructors (`NewDisequalityConstraint`, `NewLocalConstraintStore`, etc.)
- Low-level store operations (power users only)
- Debugging/diagnostic functions
- API versioning utilities

**Total True End-User Functions:** ~45 (refined from initial 87)

## Methodology
1. Reviewed implementation-roadmap.md for completed features
2. Identified exported functions users write in typical programs
3. Categorized by functional area and priority
4. Matched against existing Example functions
5. Identified critical documentation gaps

---

## Core miniKanren API (primitives.go, core.go)

### Execution Functions
| Function | Location | Has Example | Example Location | Priority |
|----------|----------|-------------|------------------|----------|
| `Run(n int, goalFunc)` | primitives.go:361 | ✅ YES | core_test.go:1023, examples_test.go:91 (duplicate) | HIGH |
| `RunStar(goalFunc)` | primitives.go:422 | ✅ YES | core_test.go:1046, examples_test.go:105 (duplicate) | HIGH |
| `RunWithContext(ctx, n, goalFunc)` | primitives.go:375 | ❌ NO | - | MEDIUM |
| `RunStarWithContext(ctx, goalFunc)` | primitives.go:427 | ❌ NO | - | MEDIUM |
| `RunWithIsolation(n, goalFunc)` | primitives.go:483 | ❌ NO | - | LOW |
| `RunWithIsolationContext(ctx, n, goalFunc)` | primitives.go:488 | ❌ NO | - | LOW |
| `RunDB(n, goalFunc)` | primitives.go:594 | ❌ NO | - | LOW |
| `RunDBWithContext(ctx, n, goalFunc)` | primitives.go:599 | ❌ NO | - | LOW |

### Variable & Term Creation
| Function | Location | Has Example | Example Location | Priority |
|----------|----------|-------------|------------------|----------|
| `Fresh(name string)` | primitives.go:25 | ✅ YES | core_test.go:927, examples_test.go:125 (duplicate) | HIGH |
| `NewAtom(value)` | core.go:102 | ✅ YES | examples_test.go:553 | HIGH |
| `NewPair(car, cdr)` | core.go:143 | ✅ YES | examples_test.go:566 | HIGH |
| `List(terms...)` | primitives.go:540 | ✅ YES | core_test.go:1062, examples_test.go:231 (duplicate) | HIGH |
| `AtomFromValue(value)` | primitives.go:529 | ❌ NO | - | LOW |

### Core Goals & Combinators
| Function | Location | Has Example | Example Location | Priority |
|----------|----------|-------------|------------------|----------|
| `Eq(term1, term2)` | primitives.go:48 | ✅ YES | core_test.go:947, examples_test.go:145 (duplicate) | HIGH |
| `Conj(goals...)` | primitives.go:185 | ✅ YES | core_test.go:970, examples_test.go:159 (duplicate) | HIGH |
| `Disj(goals...)` | primitives.go:268 | ✅ YES | core_test.go:988, examples_test.go:179 (duplicate) | HIGH |
| `Conde(goals...)` | primitives.go:323 | ✅ YES | core_test.go:1003, examples_test.go:264 (duplicate) | HIGH |
| `And(goals...)` | primitives.go:335 | ✅ YES | core_test.go:1128, examples_test.go:200 (duplicate) | MEDIUM |
| `Or(goals...)` | primitives.go:346 | ✅ YES | examples_test.go:215 | MEDIUM |
| `Success()` | core.go | ✅ YES | core_test.go:1098 | MEDIUM |
| `Failure()` | core.go | ✅ YES | core_test.go:1113 | MEDIUM |

### List Operations
| Function | Location | Has Example | Example Location | Priority |
|----------|----------|-------------|------------------|----------|
| `Appendo(l1, l2, l3)` | primitives.go:564 | ✅ YES | core_test.go:1082, examples_test.go:248 (duplicate) | HIGH |
| `Car(pair, car)` | constraints.go:316 | ✅ YES | examples_test.go:398 | MEDIUM |
| `Cdr(pair, cdr)` | constraints.go:328 | ✅ YES | examples_test.go:412 | MEDIUM |
| `Cons(car, cdr, pair)` | constraints.go:340 | ✅ YES | examples_test.go:426 | MEDIUM |
| `Pairo(term)` | constraints.go:358 | ✅ YES | examples_test.go:364 | MEDIUM |
| `Nullo(term)` | constraints.go:349 | ✅ YES | examples_test.go:381 | MEDIUM |

---

## Constraint API (constraints.go)

### Type Constraints
| Function | Location | Has Example | Example Location | Priority |
|----------|----------|-------------|------------------|----------|
| `Numbero(term)` | constraints.go:65 | ✅ YES | examples_test.go:284 | HIGH |
| `Symbolo(term)` | constraints.go:55 | ✅ YES | examples_test.go:301 | HIGH |

### Relational Constraints
| Function | Location | Has Example | Example Location | Priority |
|----------|----------|-------------|------------------|----------|
| `Neq(t1, t2)` | constraints.go:21 | ✅ YES | examples_test.go:348 | HIGH |
| `Absento(absent, term)` | constraints.go:32 | ✅ YES | examples_test.go:317 | MEDIUM |
| `Membero(element, list)` | constraints.go:77 | ✅ YES | examples_test.go:333 | MEDIUM |

### Control Flow
| Function | Location | Has Example | Example Location | Priority |
|----------|----------|-------------|------------------|----------|
| `Onceo(goal)` | constraints.go:109 | ✅ YES | examples_test.go:439 | MEDIUM |
| `Conda(clauses...)` | constraints.go:139 | ❌ NO | - | LOW |
| `Condu(clauses...)` | constraints.go:236 | ❌ NO | - | LOW |
| `Project(vars, goalFunc)` | constraints.go:296 | ✅ YES | examples_test.go:461 | MEDIUM |

### Safety & Validation
| Function | Location | Has Example | Example Location | Priority |
|----------|----------|-------------|------------------|----------|
| `SafeRun(timeout, goal)` | constraints.go:476 | ✅ YES | examples_test.go:76 | MEDIUM |
| `WithTimeout(timeout, goal)` | constraints.go:545 | ❌ NO | - | LOW |
| `WithConstraintValidation(goal)` | constraints.go:582 | ❌ NO | - | LOW |
| `ValidateConstraintStore(store)` | constraints.go:372 | ❌ NO | - | LOW |
| `SafeConstraintGoal(constraint)` | (various) | ✅ YES | examples_test.go:15 | MEDIUM |
| `DeferredConstraintGoal(constraint)` | (various) | ✅ YES | examples_test.go:42 | MEDIUM |

---

## Finite Domain (FD) API (fd_goals.go, fd.go)

### FD Goal Constructors
| Function | Location | Has Example | Example Location | Priority |
|----------|----------|-------------|------------------|----------|
| `FDAllDifferent(vars...)` | fd_goals.go:567 | ✅ YES | examples_test.go:488 | HIGH |
| `FDAllDifferentGoal(vars, domainSize)` | fd_goals.go:12 | ✅ YES | fd_domains_test.go:468 | HIGH |
| `FDIn(variable, values, options...)` | fd_goals.go:667 | ✅ YES | examples_test.go:510 | HIGH |
| `FDInGoal(variable, values)` | fd_goals.go:430 | ✅ YES | fd_domains_test.go:432 | HIGH |
| `FDInterval(variable, min, max, options...)` | fd_goals.go:719 | ✅ YES | examples_test.go:533 | HIGH |
| `FDIntervalGoal(variable, min, max)` | fd_goals.go:483 | ✅ YES | fd_domains_test.go:451 | HIGH |
| `FDDomainGoal(variable, domain)` | fd_goals.go:379 | ✅ YES | fd_domains_test.go:413 | MEDIUM |
| `FDInequalityGoal(x, y, typ)` | fd_goals.go:206 | ✅ YES | fd_domains_test.go:487 | MEDIUM |
| `FDCustomGoal(vars, constraint)` | fd_goals.go:294 | ❌ NO | - | LOW |
| `FDQueensGoal(vars, n)` | fd_goals.go:95 | ❌ NO | - | LOW |

### FD Options
| Function | Location | Has Example | Example Location | Priority |
|----------|----------|-------------|------------------|----------|
| `WithDomainSize(size)` | fd_goals.go:540 | ❌ NO | - | LOW |
| `WithSearchStrategy(strategy)` | fd_goals.go:547 | ❌ NO | - | LOW |
| `WithLabelingStrategy(labeling)` | fd_goals.go:554 | ❌ NO | - | LOW |

### FD Arithmetic (fd_constraints.go)
| Function | Location | Has Example | Example Location | Priority |
|----------|----------|-------------|------------------|----------|
| `FDPlus(a, b, c)` | fd_constraints.go:274 | ❌ NO | - | **HIGH** |
| `FDMinus(a, b, c)` | fd_constraints.go:298 | ❌ NO | - | **HIGH** |
| `FDMultiply(a, b, c)` | fd_constraints.go:322 | ❌ NO | - | **HIGH** |
| `FDQuotient(a, b, c)` | fd_constraints.go:346 | ❌ NO | - | **HIGH** |
| `FDModulo(a, b, c)` | fd_constraints.go:370 | ❌ NO | - | **HIGH** |
| `FDEqual(a, b, c)` | fd_constraints.go:394 | ❌ NO | - | **HIGH** |

**NOTE:** These are the Phase 6/7 "true relational arithmetic" functions - they work WITHOUT projection and are key end-user APIs!

---

## Advanced Features

### Tabling (tabling.go)
| Function | Location | Has Example | Example Location | Priority |
|----------|----------|-------------|------------------|----------|
| `TableGoal(name, goal)` | tabling.go:98 | ❌ NO | - | **HIGH** |
| `SetGlobalTableManager(manager)` | tabling.go | ✅ YES | tabling_test.go:206 | LOW |
| `GetGlobalTableManager()` | tabling.go:145 | ❌ NO | - | LOW |

**NOTE:** `TableGoal` is the main user-facing tabling function - critical for recursive relations!

### Advanced Search Strategies (primitives.go)
| Function | Location | Has Example | Example Location | Priority |
|----------|----------|-------------|------------------|----------|
| `RunDB(n, goalFunc)` | primitives.go:594 | ❌ NO | - | **MEDIUM** |
| `RunDBWithContext(ctx, n, goalFunc)` | primitives.go:599 | ❌ NO | - | MEDIUM |
| `RunNC(n, goalFunc)` | primitives.go:639 | ❌ NO | - | MEDIUM |
| `RunNCWithContext(ctx, n, goalFunc)` | primitives.go:653 | ❌ NO | - | MEDIUM |

**NOTE:** These are Phase 9 enhanced search strategies - database-style and non-chronological search!

### Fact Store (fact_store.go)
| Function | Location | Has Example | Example Location | Priority |
|----------|----------|-------------|------------------|----------|
| `NewFactStore()` | fact_store.go | ❌ NO | - | MEDIUM |
| `Assert(relation, terms...)` | fact_store.go | ❌ NO | - | MEDIUM |
| `Retract(relation, terms...)` | fact_store.go | ❌ NO | - | MEDIUM |
| `Query(relation, terms...)` | fact_store.go | ❌ NO | - | MEDIUM |

### Store Operations (store_ops.go, store_debug.go)
| Function | Location | Has Example | Example Location | Priority |
|----------|----------|-------------|------------------|----------|
| `EmptyStore()` | store_ops.go | ✅ YES | store_test.go:452 | MEDIUM |
| `StoreWithConstraint(store, c)` | store_ops.go | ✅ YES | store_test.go:470 | MEDIUM |
| `StoreWithoutConstraint(store, id)` | store_ops.go | ❌ NO | - | LOW |
| `StoreUnion(s1, s2)` | store_ops.go | ❌ NO | - | LOW |
| `StoreIntersection(s1, s2)` | store_ops.go | ❌ NO | - | LOW |
| `StoreDifference(s1, s2)` | store_ops.go | ❌ NO | - | LOW |
| `StoreVariables(store)` | store_debug.go | ✅ YES | store_test.go:498 | LOW |
| `StoreDomains(store)` | store_debug.go | ✅ YES | store_test.go:552 | LOW |
| `StoreValidate(store)` | store_debug.go | ❌ NO | - | LOW |
| `StoreToString(store)` | store_debug.go | ✅ YES | store_test.go:527 | LOW |
| `StoreSummary(store)` | store_debug.go | ✅ YES | store_test.go:582 | LOW |

### Nominal Logic (nominal.go, nominal_constraints.go)
| Function | Location | Has Example | Example Location | Priority |
|----------|----------|-------------|------------------|----------|
| `NewName(symbol)` | nominal.go | ❌ NO | - | LOW |
| `FreshName(symbol)` | nominal.go | ❌ NO | - | LOW |
| `NominalEq(t1, t2)` | nominal.go | ❌ NO | - | LOW |

### Constraint Builder (constraints.go)
| Function | Location | Has Example | Example Location | Priority |
|----------|----------|-------------|------------------|----------|
| `NewConstraintBuilder()` | constraints.go | ✅ YES | constraints_test.go:798 | LOW |

### Pool Management (constraint_bus_pool.go)
| Function | Location | Has Example | Example Location | Priority |
|----------|----------|-------------|------------------|----------|
| `GetPooledGlobalBus()` | constraint_bus_pool.go | ❌ NO | - | LOW |
| `ReturnPooledGlobalBus(bus)` | constraint_bus_pool.go | ✅ YES | examples_test.go:63 | LOW |

### API Versioning (api_stability.go)
| Function | Location | Has Example | Example Location | Priority |
|----------|----------|-------------|------------------|----------|
| `CurrentAPIVersion()` | api_stability.go | ✅ YES | api_test.go:120 | LOW |
| `GetAPIVersion(category)` | api_stability.go | ✅ YES | api_test.go:129 | LOW |
| `CheckAPIVersion(category, min)` | api_stability.go | ✅ YES | api_test.go:138 | LOW |
| `GetMigrationGuide(from, to)` | api_stability.go | ✅ YES | api_test.go:152 | LOW |

---

## Summary Statistics

### Overall Coverage (Revised)
- **Total End-User Functions Identified:** ~45 (refined from initial 87)
- **Functions with Examples:** ~30
- **Functions without Examples:** ~15
- **Current Coverage:** ~67% (estimated)
- **Roadmap Target:** 100% for HIGH priority functions

### Critical Gaps by Category
| Category | Missing Examples | Priority |
|----------|-----------------|----------|
| FD Arithmetic (6 functions) | FDPlus, FDMinus, FDMultiply, FDQuotient, FDModulo, FDEqual | **HIGH** |
| Context Execution (2 functions) | RunWithContext, RunStarWithContext | **HIGH** |
| Tabling (1 function) | TableGoal | **HIGH** |
| Advanced Search (4 functions) | RunDB, RunDBWithContext, RunNC, RunNCWithContext | MEDIUM |
| Control Flow (2 functions) | Conda, Condu | LOW |

### Priority Analysis
| Priority | Total | With Examples | Missing |
|----------|-------|---------------|---------|
| **HIGH** | ~20 | ~11 | 9 (including 6 FD arithmetic + context + tabling) |
| **MEDIUM** | ~15 | ~12 | 3 |
| **LOW** | ~10 | ~7 | 3 |

### Duplicate Examples
The following functions have duplicate examples across files (need consolidation):
- `Run`: core_test.go + examples_test.go
- `RunStar`: core_test.go + examples_test.go
- `Fresh`: core_test.go + examples_test.go
- `Eq`: core_test.go + examples_test.go
- `Conj`: core_test.go + examples_test.go
- `Disj`: core_test.go + examples_test.go
- `Conde`: core_test.go + examples_test.go
- `And`: core_test.go + examples_test.go
- `List`: core_test.go + examples_test.go
- `Appendo`: core_test.go + examples_test.go

**Total Duplicates:** 10 functions

---

## High Priority Gaps (Need Examples)

### FD Arithmetic Goals (Phase 6/7 Completed - fd_constraints.go lines 274-394)
1. `FDPlus(a, b, c)` - Relational addition: a + b = c
2. `FDMinus(a, b, c)` - Relational subtraction: a - b = c
3. `FDMultiply(a, b, c)` - Relational multiplication: a × b = c
4. `FDQuotient(a, b, c)` - Relational quotient: a ÷ b = c
5. `FDModulo(a, b, c)` - Relational modulo: a mod b = c
6. `FDEqual(a, b)` - FD arithmetic equality

### Context-Aware Execution (primitives.go)
7. `RunWithContext(ctx, n, goalFunc)` - Context-aware bounded execution
8. `RunStarWithContext(ctx, goalFunc)` - Context-aware unlimited execution

### Tabling (Phase 5.2 Completed - tabling.go:98)
9. `TableGoal(name, goal)` - Memoization for recursive relations

**Total High Priority Gaps:** 9 functions (6 FD arithmetic + 2 context + 1 tabling)

---

## Recommendations

### 1. Example Function Style Guide
**Keep them smoke-test simple with meaningful output.**

All Example functions should:
- **Focused demonstrations** - Show 1-2 key features per example (10-20 lines max)
- **Meaningful output** - Print verifiable results that demonstrate the feature works
- **Self-contained** - Stand alone without external dependencies
- **Clear comments** - Brief explanation of what's being demonstrated
- **Realistic usage** - Show typical API patterns, not edge cases
- **NOT comprehensive tests** - Save exhaustive testing for *_test.go files

Examples should demonstrate API usage, not test all possible scenarios.

### 2. Remove Duplicate Examples
Delete duplicate Example functions from `examples_test.go`:
- Lines 91-103: ExampleRun (keep core_test.go version)
- Lines 105-123: ExampleRunStar (keep core_test.go version)
- Lines 125-143: ExampleFresh (keep core_test.go version)
- Lines 145-157: ExampleEq (keep core_test.go version)
- Lines 159-177: ExampleConj (keep core_test.go version)
- Lines 179-198: ExampleDisj (keep core_test.go version)
- Lines 200-213: ExampleAnd (keep core_test.go version)
- Lines 231-246: ExampleList (keep core_test.go version)
- Lines 248-262: ExampleAppendo (keep core_test.go version)
- Lines 264-282: ExampleConde (keep core_test.go version)

### 3. Add High Priority Examples (CRITICAL)
Create Example functions for the 9 HIGH priority gaps:

**FD Arithmetic Examples** (fd_constraints_test.go or fd_constraints.go):
- `ExampleFDPlus` - Show relational addition like `2 + 3 = 5` (~15 lines)
- `ExampleFDMinus` - Show relational subtraction like `5 - 3 = 2` (~15 lines)
- `ExampleFDMultiply` - Show relational multiplication like `3 × 4 = 12` (~15 lines)
- `ExampleFDQuotient` - Show relational division like `12 ÷ 3 = 4` (~15 lines)
- `ExampleFDModulo` - Show relational modulo like `7 mod 3 = 1` (~15 lines)
- `ExampleFDEqual` - Show FD equality constraint (~15 lines)

**Context-Aware Examples** (core_test.go):
- `ExampleRunWithContext` - Show context cancellation (~15 lines)
- `ExampleRunStarWithContext` - Show context timeout (~15 lines)

**Tabling Example** (tabling_test.go):
- `ExampleTableGoal` - Show memoization benefit for recursive relations (~20 lines)

### 4. File Organization
- Keep general miniKanren examples in `core_test.go`
- Keep FD-specific examples in `fd_domains_test.go`
- Keep constraint-specific examples in `constraints_test.go`
- Keep store operation examples in `store_test.go`
- Use `examples_test.go` only for cross-cutting or safety features

---

## Next Steps

1. ✅ Create comprehensive audit document
2. ✅ Cross-reference implementation-roadmap.md for completed features
3. ✅ Identify missing HIGH priority functions (FD arithmetic, tabling, context execution)
4. ⏳ **Remove 10 duplicate examples from examples_test.go (lines 91-282)**
5. ⏳ **Add 9 HIGH priority examples (6 FD arithmetic + 2 context + 1 tabling)**
6. ⏳ Add 4 MEDIUM priority examples (advanced search strategies: RunDB, RunNC variants)
7. ⏳ Verify all examples compile: `go test -v -run '^Example' ./pkg/minikanren`
8. ⏳ Run full test suite: `go test -race ./pkg/minikanren` (confirm 406 tests pass)
9. ⏳ Update implementation-roadmap.md Task 11.3 with final coverage statistics

**Priority Order:**
1. FD Arithmetic examples (6) - **CRITICAL** - Phase 6/7 key features
2. Context execution examples (2) - **CRITICAL** - Production readiness
3. Tabling example (1) - **HIGH** - Phase 5.2 feature
4. Remove duplicates (10) - **HIGH** - Code hygiene
5. Advanced search examples (4) - MEDIUM - Specialized features

---

**End of Audit - Ready for Review**
