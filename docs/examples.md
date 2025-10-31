# Example Index — pkg/minikanren

This page collects the Go examples for the `pkg/minikanren` package and explains how both humans and LLMs can discover and run them.

Why this exists
- `go doc` and godoc are great for humans, but LLMs (and some tooling) benefit from a machine-readable manifest that lists example names, source files, and expected outputs.
- This document plus the JSON manifest (`pkg/minikanren/examples_index.json`) makes examples discoverable without hunting through many test files.

Where the examples live
- Examples live with the package tests in `pkg/minikanren/*.go` or `*_test.go`. Example functions follow the `func ExampleXxx()` convention so `godoc` picks them up.
- A small manifest was added at `pkg/minikanren/examples_index.json` listing Example function names, source files, and short descriptions.

Quick commands

Show package examples with godoc (local):

```bash
# Start a local godoc server and browse to http://localhost:6060/pkg/minikanren
godoc -http=:6060
```

Troubleshooting godoc "cannot find package" errors

- If you see a page that says something like:

	"cannot find package \".\" in: /src/minikanren"

	it means the local godoc server couldn't resolve the package by the filesystem path. The godoc web UI exposes packages by their import path (for example `github.com/gitrdm/gokando/pkg/minikanren`) rather than the raw filesystem directory. Use the package import path in the browser URL.

	Example: if your module path is `github.com/gitrdm/gokando`, open:

	```text
	http://localhost:6060/pkg/github.com/gitrdm/gokando/pkg/minikanren
	```

- How to find the correct import path for the package from the repo root:

	```bash
	# from repository root
	go list -f '{{.ImportPath}}' ./pkg/minikanren
	```

- If you don't want to fiddle with import paths, use one of these alternatives:
	- Run Example functions as tests: `go test ./pkg/minikanren -run Example -v`
	- Use the generator to get a machine-readable manifest: see `scripts/generate_examples_manifest.go` and run:

		```bash
		go run scripts/generate_examples_manifest.go -pkg pkg/minikanren -out pkg/minikanren/examples_index.json
		jq . pkg/minikanren/examples_index.json | less
		```

Install godoc (if `godoc` command not found)

- Install via `go install` (recommended):

```bash
go install golang.org/x/tools/cmd/godoc@latest
# ensure $(go env GOPATH)/bin or $GOBIN is on your PATH
export PATH="$(go env GOPATH)/bin:$PATH"
```

- After installation run the server again and use the import-path URL described above.

List Example functions by grepping the package (fast):

```bash
grep -R "^func Example" pkg/minikanren
```

Run all package tests and examples:

```bash
go test ./... -v
```

How LLMs / tools can consume the examples
- Read `pkg/minikanren/examples_index.json` to get a flat list of example names, files, and short descriptions.
- Use the `file` and `name` fields to locate the `ExampleXxx` source and the `output` field to validate results.
- If you want programmatic extraction, run `go list -json ./pkg/minikanren` and parse `*_test.go` ASTs to extract `Example` functions and their output comments.

Best practices for authoring examples

- Keep examples short and focused. One concept per example.
- Make sure examples print simple outputs (strings, numbers) so they are stable and easy to validate by automation.
- Add a short description to the manifest when adding a new example.
- Use canonical naming: `Example<Feature>` or `Example_<Type>_<Action>` for clarity.

Next steps (optional)

- Add a small generator (`scripts/generate_examples_index.go`) that parses the package for `Example` functions and regenerates `examples_index.json`. This can be executed as `go run scripts/generate_examples_index.go` and added to CI.
- Publish a top-level `docs/examples.md` (this file) into the website or README for faster human discovery.

Examples currently indexed
- `ExampleSafeConstraintGoal` — demonstrates `SafeConstraintGoal` (file: `pkg/minikanren/examples_test.go`)
- `ExampleDeferredConstraintGoal` — demonstrates `DeferredConstraintGoal` (file: `pkg/minikanren/examples_test.go`)

