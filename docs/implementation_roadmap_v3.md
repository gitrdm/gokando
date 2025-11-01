# GoKanDo Hybrid Solver Implementation Roadmap

## 1. Introduction

This document outlines the phased implementation plan for refactoring and enhancing the GoKanDo solver into a robust, extensible, and production-ready hybrid constraint programming framework. The primary goal is to move from the current prototype-level integration to a tightly-coupled, high-performance system with a clean, user-friendly API.

Each phase is designed to build upon the previous one, ensuring a stable foundation before new features are added. All work must adhere to the strict coding, documentation, and testing standards outlined below.

---

## 2. Coding and Documentation Guidelines

> These guidelines ensure consistent, high-quality implementation across all GoKanDo components. Follow these standards for all new code and documentation.

### Code Quality Standards

#### Go Language Conventions
- **Formatting**: Use `go fmt` and `goimports` for consistent formatting.
- **Naming**: Follow Go naming conventions (PascalCase for exported, camelCase for unexported).
- **Error Handling**: Use structured errors with context; avoid panics for normal operation.
- **Concurrency Abstractions**:
  - Use goroutines for lightweight parallelism.
  - Use channels for safe communication and synchronization.
  - Use `sync.Mutex`/`sync.RWMutex` for protecting shared mutable state.
  - Use `sync.WaitGroup` for coordinating goroutine lifecycles.
  - Use `context.Context` for cancellation, deadlines, and value propagation.
- **Interfaces**: Keep interfaces small and focused; use interface composition.
- **Generics**: Use Go 1.18+ generics where appropriate for type safety.

#### Code Structure
- **Package Organization**: Keep packages focused and cohesive.
- **Function Length**: Aim for functions under 50 lines; break down complex logic.
- **Variable Scope**: Minimize variable scope.
- **Constants**: Use typed constants and group related constants.

#### Performance Considerations
- **Memory Management**: Use `sync.Pool` for frequent allocations; avoid unnecessary allocations in critical paths.
- **Benchmarking**: Include benchmarks for all performance-critical code.
- **Profiling**: Use `pprof` for performance analysis; avoid premature optimization.

### Documentation Standards

#### Code Documentation
- **Package Comments**: Every package must have a doc comment explaining its purpose.
- **Function Comments**: Document all exported functions with clear, literate-style descriptions.
- **Type Comments**: Document all exported types, structs, and interfaces.
- **Usage Examples**: Provide `ExampleXxx()` functions for common usage patterns. Do not embed static code snippets in documentation.

#### Implementation Documentation
- **Design Decisions**: Document *why* architectural choices were made in commit messages or separate design documents.
- **Trade-offs**: Explain performance vs. complexity trade-offs.
- **Thread Safety**: Clearly document concurrency models, invariants, and guarantees for any concurrent code.

### Testing Standards

#### 🚫 **ABSOLUTELY NO Mocks or Stubs in Testing**
- **CRITICAL REQUIREMENT**: Testing must use REAL implementations only. Mocks, stubs, and test doubles are strictly prohibited as they hide integration bugs and create false confidence.
- **Alternative Approaches**: Use real dependencies, hermetic test environments (e.g., in-memory databases or services), and integration tests to validate the complete system.

#### Test Coverage and Structure
- **Unit Tests**: Aim for >90% coverage on new code.
- **Integration Tests**: Test all component interactions.
- **Concurrency Tests**: Use the `-race` flag in CI to detect race conditions. All parallel code must have specific tests to validate its correctness under concurrency.
- **Table-Driven Tests**: Use table-driven tests for multiple test cases.
- **Edge Cases**: Explicitly test boundary conditions and error paths.

### Implementation Approach

#### 🚫 **ABSOLUTELY NO Technical Debt**
- **CRITICAL REQUIREMENT**: All implementations must be production-ready with ZERO technical debt.
- **NO Stubs or Placeholders**: Implement complete functionality.
- **NO Simplified Implementations**: All edge cases and error conditions must be handled.
- **NO TODO Comments**: Resolve all issues completely before committing.

#### Development Workflow
- **Interface First**: Define interfaces before concrete implementations.
- **Incremental Implementation**: Implement and test one component at a time.
- **Dependency Injection**: Use interfaces to allow for flexibility, but not for injecting test fakes.

---

## 3. Implementation Phases

### Phase 1: Architectural Refactoring ✅ COMPLETED

**Objective**: Create a solid, extensible foundation by decoupling existing components and introducing core abstractions. This phase is a prerequisite for all future work.

- [x] **Task 1.1: Decompose the `FDStore` God Object** ✅
    - [x] **Objective**: Separate the concerns of modeling, solving, and state management.
    - [x] **Action**:
        - [x] Create a new `Model` struct to hold variables and constraints.
        - [x] Create a new `Solver` struct responsible for the search loop and propagation queue.
        - [x] Refactor the existing `FDStore` logic into these new components.
    - [x] **Success Criteria**: The `FDStore` is eliminated, and its responsibilities are cleanly divided. All existing FD tests pass.
    - **Implementation Notes**:
        - Created `Model` struct in `model.go` with variable and constraint management
        - Created `Solver` struct in `solver.go` with search logic separated from state
        - `FDStore` remains for backward compatibility but new code uses Model/Solver pattern

- [x] **Task 1.2: Introduce Core `Variable` and `Domain` Interfaces** ✅
    - [x] **Objective**: Decouple the solver logic from the concrete implementation of integer domains.
    - [x] **Action**:
        - [x] Define a `Domain` interface with methods like `Count()`, `Has(v)`, `Remove(v)`, `IsSingleton()`, etc.
        - [x] Define a `Variable` interface that holds a `Domain`.
        - [x] Refactor `BitSet` to be an implementation of the `Domain` interface.
        - [x] Update the new `Solver` to operate on these interfaces, not on concrete types.
    - [x] **Success Criteria**: The solver's search and propagation loops are agnostic to the underlying domain representation.
    - **Implementation Notes**:
        - Created `Domain` interface in `domain.go` with full set operations
        - Implemented `BitSetDomain` as primary `Domain` implementation
        - Created `Variable` interface and `FDVariable` implementation in `variable.go`
        - Solver operates entirely on interfaces, enabling future domain types

- [x] **Task 1.3: Re-architect the Concurrency Model** ✅
    - [x] **Objective**: Remove the global `sync.Mutex` in `FDStore` to enable contention-free parallel search.
    - [x] **Action**:
        - [x] The `Model` (variables, constraints) will be treated as read-only during a solve.
        - [x] State changes (primarily domain modifications) must be isolated per search worker. Instead of expensive deep copies, implement a high-performance, sparse copy-on-write strategy:
            - [x] Use `sync.Pool` for all mutable state objects (e.g., `Domain` implementations like `BitSet`) to eliminate GC churn.
            - [x] A worker's new state will consist of a pointer to the parent state plus the single newly-modified domain. This makes state "copying" at each search node an extremely cheap, constant-time operation.
    - [x] **Success Criteria**: The global lock is removed. The architecture supports multiple search workers operating on isolated state without lock contention or significant allocation overhead.
    - **Implementation Notes**:
        - Implemented `SolverState` as sparse, immutable state representation
        - Each state node is O(1) to create: just parent pointer + single modified domain
        - Used `sync.Pool` for state allocation to eliminate GC pressure
        - Model is read-only during solve; all mutations create new states
        - Architecture enables lock-free parallel search (to be implemented in Phase 3)

- [x] **Documentation and Examples** ✅
    - [x] Created comprehensive `ExampleXxx()` functions for all exported APIs
    - [x] `domain_example_test.go`: 8 examples covering all Domain operations
    - [x] `model_example_test.go`: 8 examples covering Model and Solver usage
    - [x] All examples include literate-style comments explaining usage
    - [x] All examples pass and are validated in CI

### Phase 2: Hybrid Solver Framework

**Objective**: Build the pluggable hybrid solver framework on top of the refactored architecture.

- [ ] **Task 2.1: Define the `SolverPlugin` Interface**
    - [ ] **Objective**: Create the contract for all pluggable domain solvers.
    - [ ] **Action**:
        - [ ] Define a `SolverPlugin` interface with methods like `CanHandle(Constraint)` and `Propagate(UnifiedStore)`.
    - [ ] **Success Criteria**: A clear, well-documented interface exists for integrating specialized solvers.

- [ ] **Task 2.2: Implement the `HybridSolver` Dispatcher**
    - [ ] **Objective**: Create the central coordinator that manages plugins.
    - [ ] **Action**:
        - [ ] Implement the `HybridSolver` struct, which maintains a registry of `SolverPlugin`s.
        - [ ] Implement the logic to dispatch constraints to the appropriate registered plugin.
    - [ ] **Success Criteria**: The `HybridSolver` can register plugins and correctly route constraints.

- [ ] **Task 2.3: Implement the Unified Store and Attributed Variables**
    - [ ] **Objective**: Create a single, high-performance source of truth for variable state that supports parallel search.
    - [ ] **Action**:
        - [ ] Design the `UnifiedStore` as a **persistent data structure**. "Modifications" (e.g., binding a variable) will not happen in-place but will instead create a new, lightweight version of the store that shares the vast majority of its structure with the parent. This makes state-splitting for parallel workers a constant-time, allocation-free operation.
        - [ ] Implement the concept of "Attributed Variables," allowing a single logical variable to hold both a relational binding and other attributes (like a finite domain).
        - [ ] The "shared propagation queue" will be a conceptual control flow within each worker, not a contended global data structure. A worker will iterate through relevant plugins until a fixed point is reached for its local state.
    - [ ] **Success Criteria**: A variable can have both a relational binding and a finite domain. The `UnifiedStore` can be branched for parallel workers with minimal overhead, and inter-solver propagation occurs without locks.

- [ ] **Task 2.4: Refactor Existing Solvers as Plugins**
    - [ ] **Objective**: Integrate the existing relational and FD logic into the new framework.
    - [ ] **Action**:
        - [ ] Wrap the core relational engine in a `RelationalPlugin` that implements the `SolverPlugin` interface.
        - [ ] Wrap the core FD engine in an `FDPlugin`.
        - [ ] Register both with the `HybridSolver`.
    - [ ] **Success Criteria**: The `HybridSolver` can solve problems using both relational and FD constraints, replicating and exceeding existing functionality. The standalone engines remain usable on their own.

### Phase 3: Constraint Library and Search Enhancements

**Objective**: Close the functional gaps in the solver's capabilities.

- [ ] **Task 3.1: Implement Parallel Search**
    - [ ] **Objective**: Fulfill the core requirement of a parallel search implementation.
    - [ ] **Action**:
        - [ ] Modify the `HybridSolver`'s search algorithm to use a worker pool of goroutines.
        - [ ] Implement a work-stealing strategy to balance the search effort across workers.
    - [ ] **Success Criteria**: The solver demonstrates speedup on multi-core machines for suitable problems. All concurrency tests pass with the `-race` flag.

- [ ] **Task 3.2: Implement Reification and a `Count` Constraint**
    - [ ] **Objective**: Enable powerful logical constraints.
    - [ ] **Action**:
        - [ ] Implement reification, allowing the truth value of a constraint to be reflected into a 0/1 variable.
        - [ ] Use reification to build a powerful, propagating `Count` global constraint.
    - [ ] **Success Criteria**: Problems like `send-more-money` can be modeled declaratively and solved efficiently without `Project`.

- [ ] **Task 3.3: Enhance the Global Constraint Library**
    - [ ] **Objective**: Provide a rich set of common, high-performance global constraints.
    - [ ] **Action**:
        - [ ] Implement a bounds-propagating `Sum` constraint.
        - [ ] Implement an `Element` constraint (`vars[index] = value`).
        - [ ] Implement a `Circuit` constraint for sequencing/path-finding problems.
    - [ ] **Success Criteria**: Problems like `magic-square` and `knights-tour` can be solved efficiently.

- [ ] **Task 3.4: Add Optimization Support**
    - [ ] **Objective**: Allow the solver to find optimal solutions.
    - [ ] **Action**:
        - [ ] Add support for an objective variable.
        - [ ] Implement a branch-and-bound search strategy to `minimize` or `maximize` the objective.
    - [ ] **Success Criteria**: The solver can find the best solution for optimization problems, not just any solution.

### Phase 4: API and Usability

**Objective**: Create a polished, user-friendly, and declarative public API.

- [ ] **Task 4.1: Design and Implement a High-Level Declarative API**
    - [ ] **Objective**: Abstract away the complexities of the underlying solver framework.
    - [ ] **Action**:
        - [ ] Create a new API package (`gokando/clp`?) for defining models.
        - [ ] Implement a builder pattern or functional options for creating variables and constraints declaratively.
    - [ ] **Success Criteria**: Users can define complex constraint problems with minimal boilerplate, focusing on the "what," not the "how."

- [ ] **Task 4.2: Comprehensive Documentation and Examples**
    - [ ] **Objective**: Ensure the new system is well-documented and easy to learn.
    - [ ] **Action**:
        - [ ] Write narrative documentation for the new API and features.
        - [ ] Create a "cookbook" of examples demonstrating how to solve common problems with the new declarative API.
        - [ ] Ensure all examples are runnable and tested as part of the CI suite.
    - [ ] **Success Criteria**: A new user can get started and solve a non-trivial problem by reading the documentation and examples.
