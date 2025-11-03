# Phase 4.1 Performance Summary

This note captures representative benchmark and profiling results for the parallel search implementation. Profiles referenced below are produced from `pkg/minikanren` benchmarks and stored under `profiles/`.

## Benchmarks (representative)

- Sequential vs Parallel (4-Queens)
  - Sequential: ~130 µs/op, 214 KB/op, 5,329 allocs/op
  - Parallel 1 worker: ~140 µs/op, 223 KB/op, 5,343 allocs/op
  - Parallel 2 workers: ~128 µs/op, 224 KB/op, 5,347 allocs/op
  - Parallel 4 workers: ~121 µs/op, 224 KB/op, 5,353 allocs/op
  - Parallel NumCPU: ~191 µs/op (overhead dominates small problems)

- Sequential vs Parallel (8-Queens, find-all)
  - Sequential: ~2.37 ms/op, ~2.99 MB/op, 83,933 allocs/op
  - Parallel 2 workers: ~3.11 ms/op, ~6.63 MB/op, 183,316 allocs/op
  - Parallel 4 workers: ~3.63 ms/op, ~8.53 MB/op, 236,639 allocs/op
  - Parallel 8 workers: ~7.37 ms/op, ~10.23 MB/op, 283,982 allocs/op

- Limited solutions (find-first)
  - Sequential: ~55 µs/op (N-Queens find-first harness)
  - Parallel: multi-ms due to overheads and coordination; use sequential for small or find-first problems unless branches are very heavy.

## Profiles

- CPU (Sequential 8-Queens): `profiles/phase4_cpu_seq_8q.prof`
- CPU (Parallel 8-Queens, 4 workers): `profiles/phase4_cpu_par4_8q.prof`
- Memory (Sequential 8-Queens): `profiles/phase4_mem_seq_8q.prof`
- Memory (Parallel 8-Queens, 4 workers): `profiles/phase4_mem_par4_8q.prof`

To view:
- CPU top:
  go tool pprof -top -nodecount=40 ./minikanren.test profiles/phase4_cpu_seq_8q.prof

- Memory (alloc_space):
  go tool pprof -top -nodecount=30 -sample_index=alloc_space ./minikanren.test profiles/phase4_mem_seq_8q.prof

## Notes and guidance

- Build without the race detector for accurate CPU attribution; otherwise TSAN frames dominate.
- Parallel search helps when subproblems have enough work to amortize coordination. Small or highly prunable searches may be slower in parallel.
- Hotspots:
  - AllDifferent.Propagate, maxMatching, and buildValueGraph dominate allocations.
  - Arithmetic.imageForTarget and BitSetDomain operations are next largest contributors.
- Termination is tasksWG-based; collector drains after cancel/limit to avoid sender blocking; all channel closes are centralized in the coordinator.

## Next opportunities

- Reduce allocation in AllDifferent (reuse slices/graphs across propagations where safe).
- Explore tighter domain representations for diagonals in N-Queens to cut image construction.
- Consider branch ordering heuristics tuned for parallelism (e.g., higher fan-out near root).
