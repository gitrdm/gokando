## Examples inventory

This file summarizes the repository scan of Go Example functions and their `// Output:` comments. It was generated from the workspace root and is intended to help prioritize which examples need stabilization before extracting example text into documentation.

### Summary

- Total example files discovered: 54
- Total Example functions: 223
- Total `// Output:` occurrences: 220

Files with missing `// Output:` blocks (need stabilization)

- `./pkg/minikanren/count_example_test.go` — examples: 1, outputs: 0 (missing: 1)
- `./pkg/minikanren/parallel_search_examples_test.go` — examples: 4, outputs: 3 (missing: 1)
- `./pkg/minikanren/reification_example_test.go` — examples: 1, outputs: 0 (missing: 1)

These three files are the only files where the number of `// Output:` blocks is less than the number of Example functions. The rest of the example files already include output blocks and are ready to be used as canonical runnable snippets for docs extraction.

### Per-file catalogue

The per-file counts (file \t examples \t outputs) are recorded in `/tmp/examples_list.txt` on the host where the scan was run. For convenience the key lines are reproduced below:

```
./pkg/minikanren/among_example_test.go	1	1
./pkg/minikanren/bin_packing_example_test.go	1	1
./pkg/minikanren/circuit_example_test.go	1	1
./pkg/minikanren/count_example_test.go	1	0
./pkg/minikanren/cumulative_example_test.go	1	1
./pkg/minikanren/diffn_example_test.go	1	1
./pkg/minikanren/domain_example_test.go	8	8
./pkg/minikanren/element_example_test.go	1	1
./pkg/minikanren/enhancements_example_test.go	4	4
./pkg/minikanren/gcc_example_test.go	1	1
./pkg/minikanren/highlevel_api_collectors_example_test.go	4	4
./pkg/minikanren/highlevel_api_example_test.go	2	2
./pkg/minikanren/highlevel_api_format_example_test.go	1	1
./pkg/minikanren/highlevel_api_globals_example_test.go	6	6
./pkg/minikanren/highlevel_api_intvarvalues_example_test.go	1	1
./pkg/minikanren/highlevel_api_optimize_example_test.go	2	2
./pkg/minikanren/highlevel_api_pldb_disjq_example_test.go	1	1
./pkg/minikanren/highlevel_api_pldb_example_test.go	2	2
./pkg/minikanren/highlevel_api_pldb_recursive_example_test.go	2	2
./pkg/minikanren/highlevel_api_pldb_slg_example_test.go	4	4
./pkg/minikanren/hybrid_example_test.go	10	10
./pkg/minikanren/hybrid_registry_example_test.go	3	3
./pkg/minikanren/lex_example_test.go	1	1
./pkg/minikanren/list_ops_example_test.go	8	8
./pkg/minikanren/minmax_example_test.go	2	2
./pkg/minikanren/model_example_test.go	8	8
./pkg/minikanren/nooverlap_example_test.go	1	1
./pkg/minikanren/nvalue_example_test.go	2	2
./pkg/minikanren/optimization_example_test.go	3	3
./pkg/minikanren/parallel_search_examples_test.go	4	3
./pkg/minikanren/pattern_example_test.go	11	11
./pkg/minikanren/pldb_example_test.go	8	8
./pkg/minikanren/pldb_hybrid_example_test.go	5	5
./pkg/minikanren/pldb_hybrid_helpers_example_test.go	5	5
./pkg/minikanren/pldb_slg_example_test.go	10	10
./pkg/minikanren/pldb_slg_recursive_example_test.go	7	7
./pkg/minikanren/propagation_example_test.go	8	8
./pkg/minikanren/rational_example_test.go	4	4
./pkg/minikanren/rational_linear_sum_example_test.go	4	4
./pkg/minikanren/regular_example_test.go	1	1
./pkg/minikanren/reification_example_test.go	1	0
./pkg/minikanren/relational_arithmetic_example_test.go	25	25
./pkg/minikanren/scaled_division_example_test.go	4	4
./pkg/minikanren/send_more_money_example_test.go	1	1
./pkg/minikanren/sequence_example_test.go	1	1
./pkg/minikanren/slg_engine_example_test.go	11	11
./pkg/minikanren/slg_wfs_example_test.go	1	1
./pkg/minikanren/slg_wrappers_example_test.go	2	2
./pkg/minikanren/stretch_example_test.go	1	1
./pkg/minikanren/sum_example_test.go	1	1
./pkg/minikanren/table_example_test.go	1	1
./pkg/minikanren/tabling_example_test.go	9	9
./pkg/minikanren/term_utils_example_test.go	14	14
./pkg/minikanren/wfs_api_example_test.go	1	1
```

### Recommended next steps

1. Open and stabilize the three files listed above. For each missing Example, ensure the example is deterministic and add a `// Output:` comment that exactly matches the printed output. Prefer self-contained inputs (no random ordering or time-dependent output).
2. Re-run `go test ./...` (or `go test -run Example ./...`) to verify examples remain passing. This should be included in docs CI.
3. Add a small script (e.g., `scripts/extract-examples.sh`) that extracts Example functions and their `// Output:` blocks into Markdown snippets. Use `go test -run Example -v` or parse the source files.

### Notes

- The per-file raw listing was saved to `/tmp/examples_list.txt` on the machine where the scan was executed; re-run the generation script if you need refreshed data locally.
- Most examples already include `// Output:` and are ready for extraction into documentation pages.

Small follow-up: if you want, I can open the three files and propose concrete `// Output:` text or make the deterministic adjustments and push the fixes to the `proj-document` branch.
