# FD Solver Gaps and Generalization Plan

Based on the analysis of the current FD solver implementation in `pkg/minikanren/fd.go`, here is a summary of the identified gaps and a plan for generalization and improvement.

### 1. Missing Core FD Functionality

The current solver is a good starting point but lacks several fundamental features required for a general-purpose FD solver.

*   **Global Constraints:** The solver has a strong implementation of `AllDifferent` and basic `Offset` and `Inequality` constraints. The `fd_custom.go` file provides a `SumConstraint`, but its propagation is minimal. A robust FD solver needs a richer library of pre-built global constraints, such as:
    *   A more powerful `Sum` constraint with bounds propagation.
    *   `Count`: `Count(value, vars) = n`
    *   `Element`: `vars[index] = value`
    *   `Lexicographic Ordering`: `(vars1...) < (vars2...)`
*   **Reification and Meta-Constraints:** The ability to reflect the truth value of a constraint into a boolean variable (0/1) is missing. This is crucial for expressing complex logical relationships. The `twelve-statements` and `send-more-money` examples work around this by using `Project` to implement a "generate-and-test" strategy, which is inefficient as it cannot prune the search space. The lack of reification and meta-constraints (like `Count`) forces a fallback to manual validation in Go instead of allowing the solver to use the constraints for propagation.
*   **Optimization:** The current solver only finds solutions but doesn't support optimization (finding a solution that minimizes or maximizes an objective function). This requires adding a cost variable and a search strategy that prunes branches that cannot lead to a better solution.

*   **Specialized Global Constraints:** The `knights-tour` example highlights the need for more specialized global constraints. The example implements a custom constraint for knight moves but with an empty propagator, relying on post-search validation. This demonstrates the need for constraints that can reason about paths or sequences, such as a `Circuit` constraint. Without effective propagators for such complex relationships, the solver is limited to simple generate-and-test approaches for entire classes of problems.

The propagation engine is based on a simple AC-3 style loop, which is a good baseline but can be improved.

*   **Advanced Propagation for Global Constraints:** The solver includes an implementation of Regin's filtering algorithm for `AllDifferent` (`fd_regin.go`), which provides strong pruning. This is a major strength. However, to become a more general-purpose solver, other global constraints would need their own high-performance propagation algorithms. For example, the `magic-square` example demonstrates that the current `SumConstraint` is too weak to solve the problem efficiently from scratch, as it only propagates when all but one variable is fixed. A more advanced implementation would prune domains based on the bounds of all variables involved in the sum.
*   **Higher-Order Constraints:** The solver doesn't have a clear mechanism for handling constraints that involve other constraints (e.g., reified constraints).

### 3. Search and Heuristics

The search implementation is a good starting point but can be made more flexible and powerful.

*   **More Search Strategies:** The current implementation uses a standard backtracking search. Other search strategies could be beneficial, such as:
    *   **Large Neighborhood Search (LNS):** Useful for optimization problems.
    *   **Restart-based search:** Can help escape from "thrashing" behavior in certain parts of the search space.
*   **Dynamic Heuristics:** The current variable and value ordering heuristics are static. Dynamic heuristics that adapt during the search (e.g., based on constraint failures) can be much more effective. The `HeuristicActivity` is a placeholder for such a heuristic.
*   **Parallel Search:** The current `Solve` method is single-threaded. For a modern solver, parallel search is a core requirement, not an afterthought. The search strategy should be designed from the ground up to support parallel exploration of the search tree, allowing it to take advantage of multi-core architectures.
    *   **Symmetry Breaking:** Many problems have symmetries that lead to redundant search. The solver should include mechanisms for symmetry breaking.

### 4. Generalization and API Design

The current API is somewhat rigid and could be made more extensible.

*   **Constraint Abstraction:** A general `Constraint` interface exists in `fd_custom.go`, which is a great feature for extensibility. The opportunity here is to build out a standard library of powerful, pre-built constraints that use this interface, reducing the need for users to implement common constraints from scratch.
*   **Solver as a Library:** The solver should be designed to be easily embeddable in other Go applications. This includes a clear separation of the model definition, search, and solution handling.
*   **Extensible Domain Infrastructure:** The current implementation is tightly coupled to integer domains represented by `BitSet`. To minimize future refactoring, the core infrastructure should be designed with extensibility in mind. This means abstracting the concepts of `Variable` and `Domain` so that other domain types (e.g., **Set Variables** for subset problems) can be added without requiring a rewrite of the core propagation and search loops.
*   **State Management:** The solver's state management (undoing changes on backtrack) is tied to the `FDStore`. A more modular design might separate the state management mechanism, allowing for different strategies (e.g., copying vs. trailing).

### 5. Higher-Level API

The current API requires users to interact directly with the `FDStore`, manually creating variables and adding constraints. While this provides essential low-level control, best practice for solver libraries is to also offer a higher-level, declarative API for ease of use.

*   **Lack of a Declarative Model:** There is no clear separation between the model definition and the solver itself. A higher-level API would allow users to define their constraint model (variables and their domains, constraints) declaratively, and then pass that model to a solver.
*   **Boilerplate:** The current approach involves significant boilerplate for setting up the store and variables, as seen in the `sudoku` and `magic-square` examples.
*   **Integration with MiniKanren:** While `fd_goals.go` provides a bridge, a more comprehensive high-level API could make the integration between the FD solver and relational goals feel seamless, allowing users to mix and match them more naturally.

### 6. Architectural Refactoring (Prerequisites for Gap Closure)

Before the functional gaps identified above can be effectively addressed, several foundational aspects of the current implementation should be refactored. The current abstractions are a good starting point but will hinder the development of features like parallel search and a high-level API.

*   **1. Separate the Model from the Solver:**
    *   **Problem:** The `FDStore` currently acts as a "god object," managing state, constraints, propagation, and search logic. This tight coupling makes the system hard to extend and test.
    *   **Action:** Decompose `FDStore` into at least two distinct components:
        *   A `Model` that declaratively holds the variables and constraints.
        *   A `Solver` that takes a `Model` and executes the search.
    *   **Benefit:** This separation is the cornerstone of a high-level API and makes the solver's components more modular and reusable.

*   **2. Introduce Core Abstractions for Extensibility:**
    *   **Problem:** The implementation is concretely tied to `FDVar` and `BitSet` for integer domains. This makes it difficult to add other domain types (e.g., set variables) in the future.
    *   **Action:** Introduce `Variable` and `Domain` interfaces. Refactor the core search and propagation logic to operate on these interfaces rather than the concrete types.
    *   **Benefit:** This makes the solver's architecture flexible and forward-looking, ensuring that future extensions like set variables don't require a complete rewrite.

*   **3. Re-architect the Concurrency Model for Parallel Search:**
    *   **Problem:** The solver is protected by a single, coarse-grained `sync.Mutex`, which will become a major bottleneck and prevent true parallel search. The current use of goroutines in `fd_goals.go` only isolates solver instances; it does not parallelize the search itself.
    *   **Action:** The `Solve` method must be redesigned to be internally concurrent. This involves replacing the global lock with a more sophisticated strategy. For example, making the `Model` read-only during a solve and giving each worker thread its own local state (like a copy of the domains or a local undo trail).
    *   **Benefit:** This is a critical prerequisite for meeting the core requirement of a parallel solver and achieving performance on multi-core systems.

Addressing these architectural issues first will create a solid foundation, making the subsequent work of adding new constraints, search strategies, and optimization features much smoother and more robust.

### 7. Hybrid Solver Integration

The goal of a hybrid system is to make the two solving paradigms (relational and finite domain) feel like a natural extension of each other. The current integration in `fd_goals.go` is a valuable proof-of-concept but falls short of the seamless interoperability required for a robust, production-ready system.

*   **1. "Black Box" Integration:**
    *   **Problem:** Each `FD...Goal` is a self-contained "black box." It creates a new `FDStore`, translates bindings from miniKanren, runs the *entire* FD `Solve` process, and then translates solutions back. There is no dynamic, incremental communication between the two solvers.
    *   **Action:** Move from a coarse-grained, "batch" integration to a fine-grained, tightly coupled one. The solvers should be able to communicate and share information throughout the search process.

*   **2. Lack of a Unified State:**
    *   **Problem:** The system maintains two separate worlds: the miniKanren `ConstraintStore` and the `FDStore`. This leads to significant boilerplate code for translating between them and prevents true interoperability.
    *   **Action:** Architect a **unified constraint store** that can manage both relational bindings and domain constraints for variables. This is the most critical step toward a seamless system.

*   **3. Implement Attributed Variables:**
    *   **Problem:** A developer must manually manage two sets of variables (`minikanren.Var` and `minikanren.FDVar`).
    *   **Action:** Introduce the concept of **attributed variables**. A single logical variable should be able to have "attributes" attached to it, such as a finite domain. When a constraint is applied to this variable, the unified store would automatically know which propagation engine to trigger.

*   **4. Create a Shared Propagation Mechanism:**
    *   **Problem:** The two solvers have separate propagation loops and cannot inform each other of new deductions.
    *   **Action:** Design a **shared propagation queue**. When an FD constraint reduces a domain to a single value, it should post a unification event to the queue. When a relational goal unifies a variable that has a domain attribute, it should post a domain-pruning event. This enables dynamic, bidirectional communication.

*   **5. Preserve Standalone Usability via Layering:**
    *   **Problem:** A tightly integrated system risks becoming a monolith, making it difficult to use one solver without the overhead of the other.
    *   **Action:** The architecture must be layered. The core relational and FD engines should remain as standalone, lightweight components. The `HybridSolver` should be a higher-level layer that orchestrates these engines.
    *   **Benefit:** This ensures that users who only need a pure relational solver or a pure FD solver can use those core engines directly without any performance or complexity penalty from the hybrid integration layer.

*   **6. Design the Hybrid Solver as a Pluggable Framework:**
    *   **Problem:** A hardcoded integration between just the relational and FD solvers limits future growth.
    *   **Action:** Evolve the `HybridSolver` to be a central dispatcher that manages a set of "pluggable" domain solvers. Define a `SolverPlugin` interface that any specialized solver (relational, FD, set-based, etc.) can implement to register itself with the main dispatcher.
    *   **Benefit:** This creates a truly extensible and future-proof architecture. New solving paradigms can be added to the system without modifying the core framework or other existing solvers, fostering a rich, composable ecosystem.

### 8. Proposed Plan for Improvement

1.  **Phase 1: Core Constraint System**
    *   [ ] Implement a general `Constraint` interface.
    *   [ ] Add core global constraints: `Sum`, `Count`, `Element`.
    *   [ ] Implement reification for basic arithmetic constraints.

2.  **Phase 2: Advanced Propagation**
    *   [ ] Implement Regin's algorithm for `AllDifferent`.
    *   [ ] Add dedicated propagators for the new global constraints.

3.  **Phase 3: Search and Optimization**
    *   [ ] Add support for an objective variable and a `minimize`/`maximize` search.
    *   [ ] Implement a restart-based search strategy.
    *   [ ] Implement a basic activity-based dynamic heuristic.

4.  **Phase 4: API and Usability**
    *   [ ] Refactor the API for better separation of concerns (model vs. search).
    *   [ ] Improve documentation and add examples for the new features.
    *   [ ] Add support for symmetry breaking.
