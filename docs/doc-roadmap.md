# Documentation Roadmap

This roadmap describes a pragmatic plan to create, verify, and publish documentation for the gokanlogic repository. The goal is to produce accurate, runnable, and maintainable docs that are generated from the codebase (using Go Example tests) and published to GitHub Pages.

## Principles

- Use existing `Example` functions as canonical runnable snippets.
- Keep examples deterministic and self-contained (so `// Output:` blocks can be used for verification).
- Automate verification in CI: run example tests and fail the docs build if examples are broken.
- Incremental delivery: prioritize core packages (`pkg/minikanren`, essential `examples/*`) first.

## Phases

1. Inventory & Prioritization (done)
   - Produce an inventory of Example functions and existing `// Output:` blocks. The inventory is in `docs/examples-inventory.md`.
   - Prioritize packages: start with `pkg/minikanren` and the top-level example programs used in guides.

2. Stabilize Examples (next)
   - For files missing `// Output:` blocks or that produce non-deterministic output, make examples deterministic and add exact `// Output:` comments.
   - Add small helper fixtures inside examples if needed to avoid reliance on internal/unexported helpers.
   - Verify by running `go test -run Example ./...`.

3. Extraction Tooling
   - Create `scripts/extract-examples.sh` (or a small Go tool) that extracts Example functions and their `// Output:` blocks into Markdown snippets.
   - The extractor should produce ready-to-embed Markdown code blocks and a brief caption describing the example's intent.

4. API Reference and Tutorial Pages
   - Generate API reference text via `go doc -all` for `pkg/...` and include it (or an HTML-rendered version) in the docs site.
   - Draft tutorials under `docs/tutorials/` using extracted example snippets: Getting started, HLAPI guide, Finite-Domain solver guide, PLDB guide, Tabling guide.

5. Docs Build & CI
   - Add `Makefile` targets and `scripts/build-docs.sh` to run:
     - `go test -run Example ./...` to verify examples
     - `go doc -all ./pkg/...` (or a static-site generator step to render pages)
   - Add a GitHub Actions workflow (`.github/workflows/docs.yml`) that runs on pushes to `master` and `proj-document`, runs the build script, and if successful, deploys to `gh-pages` branch.

6. Publish & Iterate
   - Publish docs to GitHub Pages from `gh-pages` branch.
   - Iterate on tutorials, add additional example-driven pages, and expand coverage.

## Quick local checks

Run examples-only tests locally to verify the Example functions and `// Output:` blocks:

```bash
# Run all example tests across the module
go test -run Example ./... -v

# Generate API text for inspection
go doc -all ./pkg/minikanren > docs/api-minikanren.txt
```

## CI suggested steps

- Step 1: Checkout code
- Step 2: Install Go (matching project version in go.mod)
- Step 3: Run `go test -run Example ./...`
- Step 4: Run `go test ./...` (full test-suite) or optional subset
- Step 5: If tests pass, run extraction tool and generate static docs
- Step 6: Commit generated docs to `gh-pages` branch (or use a deploy action)

## Timelines (suggested)

- Week 1: Inventory, stabilize missing examples (3 files), create extraction script prototype.
- Week 2: Draft Getting Started and HLAPI tutorial pages using extracted snippets.
- Week 3: Add CI workflow and automated publish to GitHub Pages; iterate on tutorial content.

## Next actions I can take for you

- I can open and propose deterministic `// Output:` comments for the three files identified as missing outputs and push them to the `proj-document` branch.
- I can implement `scripts/extract-examples.sh` that produces Markdown snippets from Example functions.
- I can scaffold a GitHub Actions workflow to run examples and publish the docs to `gh-pages`.

If you'd like, tell me which of the three next actions to start with and I'll proceed.
