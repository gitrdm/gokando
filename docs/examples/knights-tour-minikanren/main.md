# main

This example demonstrates basic usage of the library.

## Source Code

```go
// Package main demonstrates the Knight's Tour puzzle using gokanlogic's miniKanren.
//
// A Knight's Tour is a sequence of moves by a knight on a chessboard such
// that the knight visits every square exactly once. Knights move in an L-shape:
// 2 squares in one direction and 1 square perpendicular.
//
// This example demonstrates:
// - Using miniKanren for relational constraint solving
// - Defining knight moves as relations
// - Comparing miniKanren vs FD approaches for combinatorial problems
// - The challenges of finding complete knight's tours
package main

import (
	"context"
	"fmt"
	"time"

	. "github.com/gitrdm/gokanlogic/pkg/minikanren"
)

const N = 6
const TotalSquares = N * N

// Position represents a (row, col) coordinate on the board
type Position struct {
	Row, Col int
}

// knightMoves defines all possible knight moves from a position
func knightMoves(pos Position) []Position {
	moves := [][2]int{
		{-2, -1}, {-2, 1}, {-1, -2}, {-1, 2},
		{1, -2}, {1, 2}, {2, -1}, {2, 1},
	}

	var validMoves []Position
	for _, move := range moves {
		newRow := pos.Row + move[0]
		newCol := pos.Col + move[1]
		if newRow >= 0 && newRow < N && newCol >= 0 && newCol < N {
			validMoves = append(validMoves, Position{newRow, newCol})
		}
	}
	return validMoves
}

// positionToIndex converts a position to a linear index (0-24 for 5x5 board)
func positionToIndex(pos Position) int {
	return pos.Row*N + pos.Col
}

// indexToPosition converts a linear index to a position
func indexToPosition(index int) Position {
	return Position{index / N, index % N}
}

// knightMoveo defines the relational knight move constraint
// knightMoveo(fromIdx, toIdx) succeeds if a knight can move from fromIdx to toIdx
func knightMoveo(fromIdx, toIdx Term) Goal {
	return func(ctx context.Context, store ConstraintStore) *Stream {
		stream := NewStream()
		go func() {
			defer stream.Close()

			// Generate all possible knight moves
			for row := 0; row < N; row++ {
				for col := 0; col < N; col++ {
					from := Position{row, col}
					moves := knightMoves(from)

					for _, to := range moves {
						fromAtom := NewAtom(positionToIndex(from))
						toAtom := NewAtom(positionToIndex(to))

						// Try to unify with the given indices
						newStore := store.Clone()
						unifyStream := Conj(
							Eq(fromIdx, fromAtom),
							Eq(toIdx, toAtom),
						)(ctx, newStore)

						results, _ := unifyStream.Take(1)
						for _, result := range results {
							stream.Put(result)
						}
					}
				}
			}
		}()
		return stream
	}
}

// tourStep defines a single step in the knight's tour
// tourStep(currentPos, visited, nextPos) finds next unvisited position
func tourStep(currentPos, visited, nextPos Term) Goal {
	return Conj(
		// nextPos is a valid knight move from currentPos
		knightMoveo(currentPos, nextPos),
		// nextPos is not in the visited list
		func(ctx context.Context, store ConstraintStore) *Stream {
			return Project([]Term{nextPos, visited}, func(vals []Term) Goal {
				if nextAtom, ok := vals[0].(*Atom); ok {
					if nextIdx, ok := nextAtom.Value().(int); ok {
						// Check if nextIdx is in the visited list
						visitedList := vals[1]
						found := false

						// Walk through visited list
						pair := visitedList
						for pair != nil {
							if p, ok := pair.(*Pair); ok {
								if car, ok := p.Car().(*Atom); ok {
									if idx, ok := car.Value().(int); ok && idx == nextIdx {
										found = true
										break
									}
								}
								pair = p.Cdr()
							} else {
								break
							}
						}

						if !found {
							return Success
						}
					}
				}
				return Failure
			})(ctx, store)
		},
	)
}

// knightsTour attempts to find a knight's tour using miniKanren
func knightsTour(startIdx Term, tour Term) Goal {
	// For demonstration, we'll try a simpler approach:
	// Find a sequence where each step is a valid knight move
	// and all positions are unique

	pos1 := Fresh("p1")
	pos2 := Fresh("p2")
	pos3 := Fresh("p3")
	pos4 := Fresh("p4")

	return Conj(
		// Start from given position
		Eq(pos1, startIdx),

		// Each consecutive pair must be a valid knight move
		knightMoveo(pos1, pos2),
		knightMoveo(pos2, pos3),
		knightMoveo(pos3, pos4),

		// All positions must be different
		Neq(pos1, pos2),
		Neq(pos1, pos3),
		Neq(pos1, pos4),
		Neq(pos2, pos3),
		Neq(pos2, pos4),
		Neq(pos3, pos4),

		// Return the tour as a list
		Eq(tour, List(pos1, pos2, pos3, pos4)),
	)
}

func main() {
	fmt.Printf("=== Knight's Tour with miniKanren (%dx%d) ===\n", N, N)
	fmt.Println()

	startTime := time.Now()

	// Try to find a partial knight's tour starting from (0,0)
	startIdx := NewAtom(0) // position (0,0)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	results := RunWithContext(ctx, 5, func(q *Var) Goal {
		return knightsTour(startIdx, q)
	})

	elapsed := time.Since(startTime)

	fmt.Printf("Search completed in %.2fs\n", elapsed.Seconds())
	fmt.Printf("Found %d partial tours (4 moves each)\n\n", len(results))

	if len(results) > 0 {
		fmt.Println("Sample partial tour:")
		displayTour(results[0])
		fmt.Println()
	}

	fmt.Println("Key insights about miniKanren vs FD solver:")
	fmt.Println()
	fmt.Println("miniKanren advantages:")
	fmt.Println("- More expressive for relational definitions")
	fmt.Println("- Knight moves defined naturally as relations")
	fmt.Println("- Composable constraint building")
	fmt.Println()
	fmt.Println("miniKanren challenges:")
	fmt.Println("- No automatic domain propagation")
	fmt.Println("- Search space grows exponentially")
	fmt.Println("- Requires careful constraint ordering")
	fmt.Println()
	fmt.Println("FD solver advantages:")
	fmt.Println("- Efficient domain propagation")
	fmt.Println("- Automatic constraint propagation")
	fmt.Println("- Better for combinatorial optimization")
	fmt.Println()
	fmt.Println("Conclusion: For knight's tour, FD solver with custom validation")
	fmt.Println("is more practical than pure miniKanren for complete solutions.")
}

// displayTour pretty-prints a partial knight's tour
func displayTour(tour Term) {
	// Extract positions from the tour
	var positions []int
	pair := tour.(*Pair)

	for pair != nil {
		if atom, ok := pair.Car().(*Atom); ok {
			if idx, ok := atom.Value().(int); ok {
				positions = append(positions, idx)
			}
		}

		if nextPair, ok := pair.Cdr().(*Pair); ok {
			pair = nextPair
		} else {
			break
		}
	}

	// Create board representation
	board := make([][]int, N)
	for i := range board {
		board[i] = make([]int, N)
	}

	// Fill board with move numbers (1-based)
	for moveNum, posIdx := range positions {
		pos := indexToPosition(posIdx)
		board[pos.Row][pos.Col] = moveNum + 1
	}

	// Display board
	fmt.Println("Partial Knight's Tour (move numbers):")
	fmt.Println()
	for i := 0; i < N; i++ {
		for j := 0; j < N; j++ {
			if board[i][j] == 0 {
				fmt.Print(" . ")
			} else {
				fmt.Printf("%2d ", board[i][j])
			}
		}
		fmt.Println()
	}

	fmt.Println()
	fmt.Print("Move sequence: ")
	for i, posIdx := range positions {
		pos := indexToPosition(posIdx)
		if i > 0 {
			fmt.Print(" → ")
		}
		fmt.Printf("(%d,%d)", pos.Row, pos.Col)
	}
	fmt.Println()
}

```

## Running the Example

To run this example:

```bash
cd knights-tour-minikanren
go run main.go
```

## Expected Output

```
Hello from Proton examples!
```
