# gokanlogic Server Implementation Roadmap

This roadmap specifies how to expose gokanlogic’s hybrid engine (miniKanren + FD + Nominal + SLG/WFS + pldb + DCG) as a production-grade server. It mirrors the structure and rigor of `implementation_roadmap_v3.md`: clear phases, design tenets, acceptance criteria, and coding standards.

> Scope: HTTP/REST MVP first, then streaming (SSE/WebSocket), gRPC, hybrid endpoints, tabling/WFS truth probes, pldb queries, nominal ops, DCG parsing, async jobs, observability, security, and deployability. All features use declarative specs (data), not user-supplied code.

## Capability Matrix (Quick Reference)

| Phase | Capability Domain | Exposure Type | Primary Endpoints | Notes |
|-------|-------------------|---------------|-------------------|-------|
| A | Finite Domain (FD) core constraints & optimization | Model/Objective JSON | `/v1/solve`, `/v1/optimize` | Only FD to keep MVP small; not a limitation of overall design. |
| B | FD incremental enumeration | Server-Sent Events | `/v1/solve/stream` | Adds streaming; still FD only. |
| C | FD (same as A/B) via gRPC | Proto RPC + streaming | `Solve`, `SolveStream`, `Optimize` | Transport expansion, not new logic. |
| D | Hybrid (Relational + FD) | Relational AST + FD model | `/v1/hybrid/solve` | Introduces unified store & cross-pruning. |
| E | SLG/WFS Tabling & Negation Truth | Tabled predicate spec, truth probe | `/v1/table/query`, `/v1/wfs/truth` | Enumerated answers + three-valued truth. |
| F | pldb (Indexed relational DB) | Relation + facts spec | `/v1/pldb/query` | Ephemeral or preloaded DB queries, optional FD filtering. |
| G | Nominal Logic (Tie, Freshness, AlphaEq) | Nominal term JSON | `/v1/nominal/alphaeq`, `/v1/nominal/fresh` | Binder semantics; freshness validation. |
| H | DCG Parsing (Pattern-based SLG) | Grammar spec (rules) | `/v1/dcg/parse` | Left recursion safe via SLG engine. |
| I | Any long-running op (FD, Hybrid, Tabling, Optimize) | Async job control | `/v1/jobs/*` | Unified job lifecycle, cancellation. |
| J | Observability | Metrics/Tracing | `/metrics`, traces | Cross-cutting. |
| K | Security/Multi-tenancy | Auth headers & quotas | All endpoints | Quotas & rate limiting. |
| L | Deployment | Binary/Container/Helm | Make targets | Infra only. |
| M | Contracts & SDKs | OpenAPI/Proto/Clients | Docs assets | Developer experience. |

Clarification: The roadmap is layered. Early phases (A–C) intentionally start with FD only to minimize surface and stabilize infrastructure. Subsequent phases (D onward) progressively expose hybrid, tabling/WFS, pldb, nominal, and DCG capabilities. The design goal from the outset is comprehensive hybrid logic exposure; FD-first is a delivery strategy, not a functional cap.

---

## Architecture Philosophy (Server)

1. Thread-safe and parallel-friendly: No global hot locks; use per-request contexts and bounded worker pools.
2. Data-driven, not code-injection: All operations are specified via JSON/Proto schemas; no user functions/closures cross the API boundary.
3. Resource governance by default: Timeouts, node/solution limits, and concurrency caps are mandatory.
4. Compositional: Cleanly layer REST/gRPC on the existing engines (FD, relational, hybrid, nominal, SLG/WFS, pldb, DCG).
5. Production-ready: Race-free, leak-free, deterministic error model, robust observability, and well-documented contracts.
6. Backward compatibility: Versioned APIs; additive evolution preferred; feature flags for preview endpoints.

---

## Coding Standards (Server)

- Context usage
  - Every handler derives `ctx := r.Context()`; long work uses `context.WithTimeout` from request-specified or default limits.
  - Cancellation respected at all blocking points (stream read, solver step, DB iteration) to avoid goroutine leaks.
- Concurrency & safety
  - No execution in constructors. Handlers construct plans/specs; the engine executes when the request runs.
  - Use channels with bounded buffers and select on `ctx.Done()` to avoid unbounded growth.
  - Avoid global state; when unavoidable (e.g., table cache), guard with explicit policies and expose admin-only operations.
- Error handling
  - Structured errors with machine-readable code, human message, and retryability hint.
  - Map to HTTP: 400 (spec validation), 408 (timeout), 409 (conflict), 422 (infeasible/unsatisfiable), 429 (rate/quotas), 500 (internal), 503 (overload).
- JSON contracts
  - Schemas are explicit and versioned (`"apiVersion": "v1"`). Reserved fields prefixed with `_`.
  - Strict decoding with unknown-field rejection in production; relaxed in dev via feature flag.
- Logging & metrics
  - Structured logs (slog or zap) with request-id correlation and outcome fields.
  - Prometheus metrics: counters for requests/solutions/emitted, histograms for durations, gauges for active jobs.
  - OpenTelemetry traces around solver phases and streaming loops.
- Testing policy
  - Unit + integration tests; streaming tests validate chunking and cancellation; race detector enabled in CI.
  - Golden tests for schema I/O; property tests for spec validators.
  - End-to-end client tests with JimTcl (optional but recommended): REST happy paths, error mapping, and SSE streaming using Jim’s event loop.
- Performance
  - Hot-path allocations profiled; JSON encoding via stdlib; optional `jsoniter` behind a build tag if needed.
- Documentation
  - Godoc for public server packages; API reference (OpenAPI/Proto) checked into repo and published in `book/`.

---

## Data Contracts (v1 Overview)

- Common envelope
  - Request: `{ "apiVersion": "v1", "op": "solve|optimize|...", "spec": { ... } }`
  - Response: `{ "data": { ... }, "meta": { "elapsedMs": 123, "partial": false, "limits": { ... } } }`
- Limits
  - `maxSolutions` default: 100; cap enforced server-side.
  - `timeoutMs` default: 5000; configurable per endpoint; hard server ceiling applies.
- Errors
  - `{ "error": { "code": "VALIDATION", "message": "...", "details": { ... } } }`

Detailed per-endpoint schemas will live in `docs/api-reference/server/*.md` and OpenAPI/Proto sources.

---

## Phases

### Phase A: REST MVP (FD Model Solving & Optimization) ✅ PLANNED

- Objectives
  - Provide `/solve` and `/optimize` over HTTP/JSON to execute FD models using the existing model builder and solver APIs.
  - Enforce time/node limits and return structured results with metadata.
- Design
  - Router: `chi` (lightweight), middlewares for request-id, logging, recovery, rate limit.
  - Spec → Model: `internal/server/buildfd` translates JSON spec to `*minikanren.Model`; supports AllDifferent, LinearSum, GCC, Among, NoOverlap, Cumulative, Regular, Table, BinPacking, Scale, ScaledDivision, Modulo, Absolute, IntervalArithmetic, RationalLinearSum.
  - Execution: `SolveN(ctx, m, maxSolutions)`; `OptimizeWithOptions(ctx, m, obj, minimize, opts...)`.
- Endpoints
  - POST `/v1/solve` → `{solutions: [][]int, order: [varNames], meta: {...}}`
  - POST `/v1/optimize` → `{best: []int, objective: int, meta: {...}}`
- Acceptance criteria
  - Full model roundtrips for a representative constraint set.
  - Deterministic results; correct error mapping for infeasible/timeouts.
  - 95th percentile p50 latency within budget on sample problems.
- Tests
  - Unit: spec validation and model builder edge cases.
  - Integration: known examples parity with tests under `pkg/minikanren`.
  - Race: handlers + execution path.
  - JimTcl e2e: `health`, `solve` minimal AllDifferent, `optimize` linear sum; assert JSON fields and latency.

### Phase B: Streaming Solutions (SSE) ✅ PLANNED

- Objectives
  - Emit solutions incrementally for large search spaces without waiting for completion.
- Design
  - SSE endpoint with `Content-Type: text/event-stream`; events: `solution`, `progress`, `done`, `error`.
  - Backpressure: server-paced; client can cancel by closing connection.
- Endpoints
  - GET `/v1/solve/stream?timeoutMs=&maxSolutions=`
- Acceptance criteria
  - Clients receive first solution promptly; graceful cancellation; no goroutine leaks.
- Tests
  - Integration with simulated slow producer; validate event ordering and termination.
  - JimTcl SSE: receive first `solution` event < 1s; collect N solutions; handle client-side cancel.

### Phase C: gRPC API ✅ PLANNED

- Objectives
  - Strongly-typed interfaces and natural streaming for high-performance clients.
- Design
  - Protos in `proto/gokanlogic/v1/*.proto`; codegen with `buf` or `protoc`.
  - Services: `Solve`, `SolveStream`, `Optimize`, `ListExamples`, future `HybridSolve`, `NegationTruth`.
- Acceptance criteria
  - Interop tests in Go and one additional language (e.g., Python).
  - JimTcl not primary here (focus REST); optional smoke via `grpcurl` shell from Tcl wrapper.

### Phase D: Hybrid Relational + FD Endpoints ✅ PLANNED

- Objectives
  - Expose combined goals that reference FD variables and relational terms, evaluated via `NewHybridSolverFromModel`.
- Design
  - Declarative relational AST: `eq`, `conj`, `disj`, `fresh`, lists, pattern clauses (Matche/Matcha/Matchu).
  - Safety: depth and node caps; deny unbounded generators unless `maxSolutions` present.
- Endpoints
  - POST `/v1/hybrid/solve`
- Acceptance criteria
  - Demonstrate FD variable pruning from relational bindings and vice versa (end-to-end correctness on example suite).
  - JimTcl hybrid test: small relational disjunction + FD AllDifferent; assert combined solutions and absence of infeasible assignments.

### Phase E: SLG/WFS Tabling & Negation Truth ✅ PLANNED

- Objectives
  - Provide safe tabled queries and truth probes without exposing internal engine state.
- Design
  - Tabled predicates registered through spec (name + arity + definition via relational AST or pldb rules).
  - Truth probe wraps `SLGEngine.NegationTruth` and returns `true|false|undefined`.
- Endpoints
  - POST `/v1/table/query`
  - POST `/v1/wfs/truth`
- Acceptance criteria
  - Stratified cases return expected truth; undefined behavior represented explicitly.
  - JimTcl truth probe: query unreachable vs reachable pair; assert `truth: true|false|undefined` values.

### Phase F: pldb (In-Memory Relational DB) ✅ PLANNED

- Objectives
  - Create ephemeral databases in-request or reference preloaded datasets.
- Design
  - Spec defines relations with arity and fact batches; queries specify patterns; optional FD filtering internally (akin to `FDFilteredQuery`).
- Endpoints
  - POST `/v1/pldb/query`
- Acceptance criteria
  - Indexed queries demonstrate sub-linear performance vs. linear scan baseline; joins verified.
  - JimTcl pldb test: ephemeral relation `edge`; query successors of `a`; assert response list membership.

### Phase G: Nominal Logic Endpoints ✅ PLANNED

- Objectives
  - Surface nominal constructs (Tie, Freshness, AlphaEq) safely as data.
- Design
  - Terms serialized as nested structures; server enforces size limits and add-time validation semantics.
- Endpoints
  - POST `/v1/nominal/alphaeq`
  - POST `/v1/nominal/fresh`
- Acceptance criteria
  - Alpha-equivalence parity with library tests; freshness immediate rejection surfaced as 422.
  - JimTcl nominal test: alpha-eq of `λx.x` vs `λy.y` returns success; freshness violation returns 422 with code `INFEASIBLE` or `VALIDATION` as defined.

### Phase H: DCG Parsing (Pattern-Based SLG) ✅ PLANNED

- Objectives
  - Declarative grammars with left recursion supported via SLG.
- Endpoints
  - POST `/v1/dcg/parse`
- Acceptance criteria
  - Clause-order independence; no timeouts required; deterministic answers.
  - JimTcl DCG test: arithmetic expr grammar parse of `number + number * number`; assert tree shape markers.

### Phase I: Jobs & Cancellation ✅ PLANNED

- Objectives
  - Async execution for long optimizations; explicit cancellation.
- Design
  - POST to create job → returns `jobId`; background worker with bounded queue; GET to poll; SSE/WebSocket for push updates; DELETE to cancel.
- Endpoints
  - POST `/v1/jobs/solve|optimize|hybrid|...`
  - GET `/v1/jobs/{id}`
  - DELETE `/v1/jobs/{id}`
- Acceptance criteria
  - No orphan goroutines; job TTL and cleanup policies; persistence optional in v1 (in-memory okay).
  - JimTcl job test: create optimize job; poll status until `done`; issue cancel before completion for second job and assert terminal canceled state.

### Phase J: Observability ✅ PLANNED

- Objectives
  - Production-grade logs, metrics, and traces.
- Deliverables
  - Prometheus `/metrics`; OTel traces; log sampling knobs; per-endpoint SLIs (success rate, latency, error budget tracking).

### Phase K: Security & Multi-Tenancy ✅ PLANNED

- Objectives
  - Basic authN/Z, rate limits, and resource isolation.
- Deliverables
  - API keys (header); per-key quotas; per-request limits (variables count, domain size, max facts); input size caps; request/response redaction rules in logs.

### Phase L: Deployment Paths ✅ PLANNED

- Objectives
  - Single binary; container; systemd; Helm chart.
- Deliverables
  - Makefile targets: `server`, `docker-build`, `docker-run`.
  - Dockerfile (multi-stage); basic Helm chart (values for limits and features).

### Phase M: API Contracts & SDKs ✅ PLANNED

- Objectives
  - OpenAPI for REST; Proto for gRPC; thin SDKs.
- Deliverables
  - `openapi/server.v1.yaml` + doc site integration; generated clients (Go/TS minimal examples); versioning policy documented.

---

## Non-Goals (v1)

- Arbitrary user code execution (closures, plugins) via the API.
- Raw access to internal tables, delay sets, or SCC graphs.
- Unbounded enumeration by default (server enforces caps even if client omits).

---

## Acceptance & Quality Gates (Global)

- Build: PASS — `go build ./...` for server packages; container builds succeed.
- Lint/Typecheck: PASS — `golangci-lint` baseline (or `staticcheck`); no warnings in changed files.
- Tests: PASS — unit + integration; race detector green; streaming tests deterministic.
- Performance: PASS — target throughput/latency for representative workloads; no leaks in `pprof` smoke tests.
- Docs: PASS — OpenAPI/Proto published; user guide updated in `book/`.

---

## Initial Package Layout (Proposed)

```
cmd/
  server/
    main.go                 # bootstrap, routes, feature flags
internal/server/
  router.go                 # chi setup, middlewares
  handlers_fd.go            # /solve, /optimize
  handlers_stream.go        # /solve/stream (SSE)
  handlers_hybrid.go        # hybrid endpoints
  handlers_tab_wfs.go       # tabled queries, negation truth
  handlers_pldb.go          # pldb queries
  handlers_nominal.go       # alphaeq, fresh
  handlers_dcg.go           # dcg parsing
  jobs.go                   # job manager, in-memory store
  buildfd/
    spec.go                 # JSON schema structs
    validate.go             # spec validation (sizes, limits)
    builder.go              # spec → *minikanren.Model
  relast/
    ast.go, validate.go     # relational AST + safe interpreter
  respond/
    error.go, write.go      # error mapping and JSON writer
pkg/api/
  openapi/                  # optional: generated assets or schemas
proto/gokanlogic/v1/        # gRPC IDL
tests/jim/                  # JimTcl client harness (scripts, helpers, cases)
  http.tcl                  # request wrapper (curl), JSON parse helpers
  assert.tcl                # assertion procs (eq, json-path, http status)
  cases/                    # individual end-to-end test scripts
tests/data/                 # JSON spec fixtures for reuse
```

---

## Example Error Model

- VALIDATION: input schema/constraints invalid → 400
- INFEASIBLE: model has no solutions → 422
- TIMEOUT: context deadline exceeded → 408 (partial flag may be true)
- RATE_LIMITED: over quota → 429
- OVERLOAD: server load-shed → 503
- INTERNAL: unexpected failure → 500 (with error id)

---

## Example Limits (Server Defaults)

- maxSolutions: 100 (hard upper bound 10k with privileged key)
- timeoutMs: 5000 (hard cap 60000)
- maxVariables: 200; maxConstraints: 2000
- maxRelationalNodes: 20k; maxListDepth: 2k
- pldb: maxFacts per relation 100k (MVP), payload size 8 MB
- nominal: term size 10k nodes

---

## Open Questions / Future Work

- Job persistence: plug-in interface for Redis/Postgres-backed job store.
- Policy knobs: per-tenant config via env or dynamic control plane.
- GraphQL façade: optional on top of REST for flexible projection.
- WASM playground: embed core engine and reuse JSON specs client-side.

---

## Success Criteria (End-to-End)

- Users can:
  - Solve/optimize FD models via REST with guardrails and clear errors.
  - Stream large result sets without leaks; cancel at will.
  - Run hybrid and tabled/WFS queries safely with deterministic outcomes.
  - Query pldb-backed datasets and combine with FD filters.
  - Use nominal operators (alpha-eq, fresh) as data.
  - Parse with DCG declaratively.
- Operators can:
  - Monitor with Prometheus/OTel; enforce quotas and limits; deploy via Docker/Helm.
  - Upgrade API versions without breaking existing clients.

---

## Phase A Delivery Checklist (MVP Ready-to-Start)

- [ ] Package skeleton created (see layout) with `/solve` and `/optimize` wired
- [ ] JSON schemas with validation and examples committed
- [ ] Basic observability (logs, metrics) and health endpoint
- [ ] Unit + integration tests green; race detector passes
- [ ] Makefile targets: `server`, `test-server`, `docker-build`, `docker-run`
- [ ] Minimal OpenAPI for the two endpoints exposed and published to `book/`

---

Document owner: Server/Platform. Please keep this roadmap in lock-step with `implementation_roadmap_v3.md` and update statuses as phases land.