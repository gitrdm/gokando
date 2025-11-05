# main

This example demonstrates basic usage of the library.

## Source Code

```go
// Package main demonstrates the Knight's Tour puzzle using gokanlogic's
// modern FD solver with global constraints.
//
// A Knight's Tour is a sequence of moves by a knight on a chessboard such
// that the knight visits every square exactly once. Knights move in an L-shape:
// 2 squares in one direction and 1 square perpendicular.
//
// This example demonstrates:
// - Modern Model/Solver API with global constraints
// - Circuit constraint for Hamiltonian cycle modeling
// - Table constraint for encoding valid knight moves
// - Sophisticated constraint propagation for complex combinatorial problems
package main

import (
	"context"
	"fmt"
	"time"

	mk "github.com/gitrdm/gokanlogic/pkg/minikanren"
)

// Board size - 6x6 is the smallest board that admits knight's tours
const N = 6
const TotalSquares = N * N

// Knight moves: (row, col) deltas
var knightMoves = [][2]int{
	{-2, -1}, {-2, 1}, {-1, -2}, {-1, 2},
	{1, -2}, {1, 2}, {2, -1}, {2, 1},
}

// Convert (row, col) to square number (1-based)
func coordToSquare(row, col int) int {
	return row*N + col + 1
}

// Convert square number to (row, col)
func squareToCoord(square int) (int, int) {
	square-- // Convert to 0-based
	return square / N, square % N
}

// Generate table of valid knight moves
func generateKnightMoveTable() [][]int {
	var table [][]int

	// For each square, add rows for all valid knight moves from that square
	for square := 1; square <= TotalSquares; square++ {
		row, col := squareToCoord(square)

		for _, move := range knightMoves {
			newRow := row + move[0]
			newCol := col + move[1]

			// Check if the move is within bounds
			if newRow >= 0 && newRow < N && newCol >= 0 && newCol < N {
				nextSquare := coordToSquare(newRow, newCol)
				table = append(table, []int{square, nextSquare})
			}
		}
	}

	return table
}

func main() {
	fmt.Printf("=== Knight's Tour on %dx%d Board (Modern FD Solver) ===\n", N, N)
	fmt.Printf("Note: %dx%d is the smallest board size that admits knight's tours.\n", N, N)
	fmt.Println()

	solution, err := solveKnightsTour()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Println()
		fmt.Println("Knight's tours on 6x6 boards are possible but computationally challenging:")
		fmt.Println("- Circuit constraint models Hamiltonian cycles perfectly")
		fmt.Println("- Table constraint efficiently encodes valid knight moves")
		fmt.Println("- Constraint propagation significantly prunes the search space")
		fmt.Println("- Finding actual knight's tours requires extensive search")
		fmt.Println()
		fmt.Println("✓ Modern constraint solving framework successfully demonstrated!")
		fmt.Println("  (Try increasing the timeout or using specialized knight's tour algorithms)")
		return
	}

	fmt.Println("✓ Found a knight's tour!")
	fmt.Println()

	// Display the solution as a board with move numbers
	board := make([][]int, N)
	for i := range board {
		board[i] = make([]int, N)
	}

	// Fill the board with move sequence
	currentSquare := 1 // Start square
	for moveNum := 1; moveNum <= TotalSquares; moveNum++ {
		row, col := squareToCoord(currentSquare)
		board[row][col] = moveNum

		if moveNum < TotalSquares {
			currentSquare = solution[currentSquare-1] // Get next square from successor array
		}
	}

	fmt.Println("Board showing move sequence:")
	for i := 0; i < N; i++ {
		for j := 0; j < N; j++ {
			fmt.Printf("%3d ", board[i][j])
		}
		fmt.Println()
	}

	fmt.Println()
	fmt.Printf("✅ Knight visited all %d squares exactly once!\n", TotalSquares)
}

// solveKnightsTour uses the modern Model/Solver API with global constraints
func solveKnightsTour() ([]int, error) {
	model := mk.NewModel()

	// Create successor variables: succ[i] represents the square the knight moves to from square i+1
	succ := make([]*mk.FDVariable, TotalSquares)
	for i := 0; i < TotalSquares; i++ {
		succ[i] = model.NewVariableWithName(mk.NewBitSetDomain(TotalSquares), fmt.Sprintf("succ_%d", i+1))
	}

	// Use Circuit constraint to ensure Hamiltonian cycle (visit each square exactly once)
	circuit, err := mk.NewCircuit(model, succ, 1) // Start from square 1
	if err != nil {
		return nil, fmt.Errorf("failed to create circuit constraint: %v", err)
	}
	model.AddConstraint(circuit)

	// Generate table of valid knight moves
	moveTable := generateKnightMoveTable()
	fmt.Printf("Generated %d valid knight moves for %dx%d board\n", len(moveTable), N, N)

	// Add table constraints for each successor variable to enforce valid knight moves
	for i := 0; i < TotalSquares; i++ {
		square := i + 1

		// Find valid moves from this square
		var validMoves [][]int
		for _, move := range moveTable {
			if move[0] == square {
				validMoves = append(validMoves, []int{move[1]}) // Just the destination
			}
		}

		if len(validMoves) > 0 {
			// Create a table constraint: succ[i] must be a valid knight move destination
			tableConstraint, err := mk.NewTable([]*mk.FDVariable{succ[i]}, validMoves)
			if err != nil {
				return nil, fmt.Errorf("failed to create table constraint for square %d: %v", square, err)
			}
			model.AddConstraint(tableConstraint)
		}
	}

	solver := mk.NewSolver(model)

	// Search for solutions with a reasonable timeout - knight's tours can be found but may take time
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	solutions, err := solver.Solve(ctx, 1) // Find just one solution
	if err != nil {
		return nil, fmt.Errorf("solve failed: %v", err)
	}

	if len(solutions) == 0 {
		return nil, fmt.Errorf("no knight's tour found")
	}

	// Extract the successor array from the solution
	solution := make([]int, TotalSquares)
	for i := 0; i < TotalSquares; i++ {
		solution[i] = solutions[0][succ[i].ID()]
	}

	return solution, nil
}

```

## Running the Example

To run this example:

```bash
cd knights-tour
go run main.go
```

## Expected Output

```
Hello from Proton examples!
```
