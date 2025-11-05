# Documentation Generation Guide

Complete guide for generating and maintaining gokanlogic documentation.

## Quick Reference

```bash
# Most common workflow
make docs-serve    # Preview at http://localhost:3000 (auto-reload)

# Build documentation
make docs          # Build from examples (no Proton)
make docs-all      # Full build with Proton API generation

# Clean up
make docs-clean    # Remove generated files
```

## Prerequisites

### mdBook (Required)

```bash
# Install mdBook binary
curl -L https://github.com/rust-lang/mdBook/releases/download/v0.4.40/mdbook-v0.4.40-x86_64-unknown-linux-gnu.tar.gz | tar xz
sudo mv mdbook /usr/local/bin/mdbook
mdbook --version
```

### Proton (Optional - only for API regeneration)

```bash
go install github.com/kolosys/proton/cmd/proton@latest
```

## Documentation Workflow

### 1. Preview Changes Locally

```bash
make docs-serve
# Opens http://localhost:3000 with auto-reload
```

### 2. Build for Deployment

```bash
# If you only changed examples or manual docs:
make docs

# If you changed Go code/comments:
make docs-all
```

### 3. Deploy

Push to `master` - GitHub Actions automatically builds and deploys.

## What Gets Generated

### You Edit These

- `docs/SUMMARY.md` - Navigation structure
- `docs/README.md` - Introduction page  
- `docs/getting-started/*.md` - Guides
- `docs/examples/*/README.md` - Example documentation
- `pkg/minikanren/*_test.go` - Example code

### Auto-Generated (Don't Edit)

- `docs/api-reference/minikanren.md` - From Proton
- `docs/api-reference/nominal.md` - From Proton
- `docs/api-reference/parallel.md` - From Proton
- `docs/generated-examples.md` - From test examples
- `book/` - mdBook output (not committed)

## Makefile Targets

| Command | What It Does |
|---------|-------------|
| `make docs` | Extract examples + build mdBook (fast) |
| `make docs-all` | Proton + examples + mdBook (complete) |
| `make docs-proton` | Run Proton API generator only |
| `make docs-serve` | Preview at localhost:3000 |
| `make docs-clean` | Remove all generated files |
| `make help` | Show all available targets |

## Adding New Content

### Add a New Example

1. Create `docs/examples/my-example/README.md`
2. Add entry to `docs/SUMMARY.md`:
   ```markdown
   - [My Example](examples/my-example/README.md)
   ```
3. Run `make docs-serve` to preview

### Add a Code Example

1. Add to `pkg/minikanren/*_test.go`:
   ```go
   func ExampleMyFeature() {
       // Your example code
       // Output:
       // Expected output
   }
   ```
2. Run `make docs` - it's auto-extracted

## Configuration

### `.proton/config.yml`

Configured to exclude `examples/` and `cmd/` from API generation.
Only generates docs for `pkg/minikanren`.

### `book.toml`

mdBook configuration - theme, output directory, etc.

## Troubleshooting

**Q: Links showing 404?**
- Check paths in `docs/SUMMARY.md` - must be relative to `docs/`
- Run `make docs-serve` to test locally

**Q: Examples not updating?**
```bash
make docs-clean
make docs
```

**Q: Proton generating wrong files?**
- Check `.proton/config.yml` exclude patterns
- Verify only `pkg/` is included

## Deployment

GitHub Actions automatically:
1. Installs mdBook and Proton
2. Generates API docs
3. Extracts examples
4. Builds mdBook site
5. Deploys to https://gitrdm.github.io/gokanlogic/

## Tips

- Use `make docs-serve` for instant feedback while writing
- Run `make test-examples` to verify example code works
- Check `.gitignore` - `book/` directory is excluded
- Don't edit generated API files - they'll be overwritten
