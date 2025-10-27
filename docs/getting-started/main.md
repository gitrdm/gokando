# Getting Started with main

Package main solves the famous Zebra puzzle (Einstein's Riddle) using GoKando.

The Zebra puzzle is a logic puzzle with the following constraints:
  - There are five houses.
  - The English man lives in the red house.
  - The Swede has a dog.
  - The Dane drinks tea.
  - The green house is immediately to the left of the white house.
  - They drink coffee in the green house.
  - The man who smokes Pall Mall has a bird.
  - In the yellow house they smoke Dunhill.
  - In the middle house they drink milk.
  - The Norwegian lives in the first house.
  - The Blend-smoker lives in the house next to the house with a cat.
  - In a house next to the house with a horse, they smoke Dunhill.
  - The man who smokes Blue Master drinks beer.
  - The German smokes Prince.
  - The Norwegian lives next to the blue house.
  - They drink water in a house next to the house where they smoke Blend.

Question: Who owns the zebra?


## Overview

**Import Path:** `github.com/gitrdm/gokando/examples/zebra`

Package main solves the famous Zebra puzzle (Einstein's Riddle) using GoKando.

The Zebra puzzle is a logic puzzle with the following constraints:
  - There are five houses.
  - The English man lives in the red house.
  - The Swede has a dog.
  - The Dane drinks tea.
  - The green house is immediately to the left of the white house.
  - They drink coffee in the green house.
  - The man who smokes Pall Mall has a bird.
  - In the yellow house they smoke Dunhill.
  - In the middle house they drink milk.
  - The Norwegian lives in the first house.
  - The Blend-smoker lives in the house next to the house with a cat.
  - In a house next to the house with a horse, they smoke Dunhill.
  - The man who smokes Blue Master drinks beer.
  - The German smokes Prince.
  - The Norwegian lives next to the blue house.
  - They drink water in a house next to the house where they smoke Blend.

Question: Who owns the zebra?


## Installation

### Install the package

```bash
go get github.com/gitrdm/gokando/examples/zebra
```

### Verify installation

Create a simple test file to verify the package works:

```go
package main

import (
    "fmt"
    "github.com/gitrdm/gokando/examples/zebra"
)

func main() {
    fmt.Println("main package imported successfully!")
}
```

Run it:

```bash
go run main.go
```

## Quick Start

Here's a basic example to get you started with main:

```go
package main

import (
    "fmt"
    "log"

    "github.com/gitrdm/gokando/examples/zebra"
)

func main() {
    // TODO: Add basic usage example
    fmt.Println("Hello from main!")
}
```

## Key Features

## Usage Examples

For more detailed examples, see the [Examples](../examples/README.md) section.

## Next Steps

- [Full API Reference](../api-reference/main.md) - Complete API documentation
- [Examples](../examples/README.md) - Working examples and tutorials
- [Best Practices](../guides/main/best-practices.md) - Recommended patterns and usage

## Documentation Links

- [pkg.go.dev Documentation](https://pkg.go.dev/github.com/gitrdm/gokando/examples/zebra)
- [Source Code](https://github.com/gitrdm/gokando/tree/master/examples/zebra)
- [GitHub Issues](https://github.com/gitrdm/gokando/issues)
