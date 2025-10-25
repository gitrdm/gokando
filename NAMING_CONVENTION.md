# GoKando - Naming Conventions

This document clarifies the naming conventions used throughout GoKando documentation and code.

## Project Name: GoKando

**GoKando** is the project name and should be used:
- In introductions and titles
- When referring to the overall project
- In documentation headings
- In user-facing materials

Examples:
- ✅ "GoKando is a thread-safe, parallel implementation of miniKanren in Go"
- ✅ "GoKando provides advanced parallel execution capabilities"
- ✅ "Best practices for GoKando"

## Algorithm/DSL: miniKanren

**miniKanren** should be used when referring to:
- The constraint logic programming language/DSL
- The algorithm or approach
- The domain-specific language itself

Examples:
- ✅ "miniKanren is a domain-specific language"
- ✅ "This is a miniKanren implementation"
- ✅ "The core miniKanren operators"

## Package Import Path: gokando

**gokando** (lowercase) should be used in:
- Package import statements
- Repository references
- GitHub URLs
- Go module paths

Examples:
- ✅ `import "github.com/gitrdm/gokando/pkg/minikanren"`
- ✅ `github.com/gitrdm/gokando` (repository)
- ✅ `go get github.com/gitrdm/gokando@latest`

## Correct Usage Examples

### Documentation Headers
```markdown
# GoKando - miniKanren Best Practices
# GoKando - Parallel Execution Guide
# Contributing to GoKando
```

### Introduction Sentences
```
GoKando is a production-quality implementation of miniKanren in Go...
GoKando provides support for miniKanren constraints...
This GoKando guide covers...
```

### Package References
```go
import "github.com/gitrdm/gokando/pkg/minikanren"
import "github.com/gitrdm/gokando/internal/parallel"
```

### GitHub/Repository References
```
Repository: https://github.com/gitrdm/gokando
Issues: https://github.com/gitrdm/gokando/issues
```

## Summary

| Context | Term | Example |
|---------|------|---------|
| Project name | GoKando | "GoKando documentation" |
| Algorithm/DSL | miniKanren | "miniKanren operators" |
| Import path | gokando | `github.com/gitrdm/gokando` |
| Casual mentions | gokando | "see gokando docs" |

---

When in doubt, use:
- **GoKando** for project references in formal documentation
- **miniKanren** when discussing the algorithm/language
- **gokando** in code and URLs
