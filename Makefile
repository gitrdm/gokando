SHELL := /bin/bash

# Default tools (override on command line if needed)
PROTON ?= proton

.PHONY: help docs docs-all docs-proton extract-examples assemble-examples force \
	test test-examples test-fast test-race test-bench test-pkg

help: ## Show this help message
	@echo "Available targets:" && echo && \
	awk 'BEGIN {FS=":.*##"} /^[a-zA-Z0-9_.-]+:.*##/ {printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST) | sort

docs: force extract-examples assemble-examples ## Build docs from examples and generate static site with mdBook
	@command -v mdbook >/dev/null 2>&1 || { echo "Error: mdbook not found. Install: https://rust-lang.github.io/mdBook/guide/installation.html"; exit 1; }
	@mdbook build
	@echo "Static docs built in ./book/"

docs-all: docs-proton docs ## Full docs build: Proton + examples + mdBook (hybrid)
	@echo "Docs build complete (proton + examples + static site)."

docs-proton: ## Generate API and guides via Proton (uses project Proton config)
	@command -v $(PROTON) >/dev/null 2>&1 || { echo "Error: '$(PROTON)' not found in PATH"; exit 1; }
	@echo "Running $(PROTON) generate ..."
	@$(PROTON) generate
	@echo "Proton generation finished."

force: ## The 'force' target is a no-op used for phony dependencies
	@true

extract-examples: ## Extract code examples from _test.go into docs/examples-snippets
	@chmod +x scripts/extract-examples.sh
	@./scripts/extract-examples.sh

assemble-examples: ## Assemble generated examples into a single docs/generated-examples.md
	@mkdir -p docs
	@echo "# Generated Examples" > docs/generated-examples.md
	@for f in docs/examples-snippets/*.md; do \
		echo "## $$(basename $$f)" >> docs/generated-examples.md; \
		sed -n '1,2000p' "$$f" >> docs/generated-examples.md; \
		echo "\n" >> docs/generated-examples.md; \
	 done
	@echo "Assembled $$(ls -1 docs/examples-snippets | wc -l) snippets into docs/generated-examples.md"

test: ## Run full test-suite (verbose)
	@go test ./... -v

test-examples: ## Run only Example tests (useful for docs verification)
	@go test -run Example ./... -v

test-fast: ## Fast tests (short mode); useful for CI quick checks
	@go test -short ./... -v

test-race: ## Run tests with the race detector
	@go test -race ./... -v

test-bench: ## Run benchmarks (all)
	@go test -bench . -run ^$$ ./... -v

test-pkg: ## Test a specific package; set PKG (example: PKG=./pkg/minikanren)
	@if [ -z "$(PKG)" ]; then echo "Usage: make test-pkg PKG=./pkg/minikanren"; exit 1; fi
	@go test $(PKG) -v
