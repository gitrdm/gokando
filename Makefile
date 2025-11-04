SHELL := /bin/bash
.PHONY: docs extract-examples assemble-examples force

docs: force extract-examples assemble-examples
	@echo "Docs assembled: docs/generated-examples.md"

force:
	@true

extract-examples:
	@chmod +x scripts/extract-examples.sh
	@./scripts/extract-examples.sh

assemble-examples:
	@mkdir -p docs
	@echo "# Generated Examples" > docs/generated-examples.md
	@for f in docs/examples-snippets/*.md; do \
		echo "## $$(basename $$f)" >> docs/generated-examples.md; \
		sed -n '1,2000p' "$$f" >> docs/generated-examples.md; \
		echo "\n" >> docs/generated-examples.md; \
	done
	@echo "Assembled $$(ls -1 docs/examples-snippets | wc -l) snippets into docs/generated-examples.md"

# Test targets
.PHONY: test test-examples test-fast test-race test-bench test-pkg

# Run full test-suite (verbose)
test:
	@go test ./... -v

# Run only Example tests (useful for docs verification)
test-examples:
	@go test -run Example ./... -v

# Fast tests (short mode); useful for CI quick checks
test-fast:
	@go test -short ./... -v

# Run tests with the race detector
test-race:
	@go test -race ./... -v

# Run benchmarks (all)
test-bench:
	@go test -bench . -run ^$$ ./... -v

# Test a specific package; set PKG (example: PKG=./pkg/minikanren)
test-pkg:
	@if [ -z "$(PKG)" ]; then echo "Usage: make test-pkg PKG=./pkg/minikanren"; exit 1; fi
	@go test $(PKG) -v
