package minikanren

import "fmt"

// ExamplePluso demonstrates basic addition with Pluso.
func ExamplePluso() {
	result := Run(1, func(q *Var) Goal {
		return Pluso(NewAtom(2), NewAtom(3), q)
	})
	fmt.Println(result[0])
	// Output: 5
}

// ExamplePluso_backward demonstrates solving for x in x + 3 = 8.
func ExamplePluso_backward() {
	result := Run(1, func(q *Var) Goal {
		return Pluso(q, NewAtom(3), NewAtom(8))
	})
	fmt.Println(result[0])
	// Output: 5
}

// ExamplePluso_generate demonstrates generating pairs that sum to 5.
func ExamplePluso_generate() {
	result := Run(6, func(q *Var) Goal {
		x := Fresh("x")
		y := Fresh("y")
		return Conj(
			Pluso(x, y, NewAtom(5)),
			Eq(q, NewPair(x, y)),
		)
	})

	fmt.Printf("Generated %d pairs\n", len(result))
	// Verify all pairs sum to 5
	for _, r := range result {
		pair := r.(*Pair)
		x, _ := extractNumber(pair.Car())
		y, _ := extractNumber(pair.Cdr())
		if x+y == 5 {
			fmt.Println("Valid pair")
		}
	}
	// Output:
	// Generated 6 pairs
	// Valid pair
	// Valid pair
	// Valid pair
	// Valid pair
	// Valid pair
	// Valid pair
} // ExampleMinuso demonstrates basic subtraction with Minuso.
func ExampleMinuso() {
	result := Run(1, func(q *Var) Goal {
		return Minuso(NewAtom(10), NewAtom(3), q)
	})
	fmt.Println(result[0])
	// Output: 7
}

// ExampleMinuso_backward demonstrates solving for x in 10 - x = 6.
func ExampleMinuso_backward() {
	result := Run(1, func(q *Var) Goal {
		return Minuso(NewAtom(10), q, NewAtom(6))
	})
	fmt.Println(result[0])
	// Output: 4
}

// ExampleMinuso_negative demonstrates subtraction producing negative results.
func ExampleMinuso_negative() {
	result := Run(1, func(q *Var) Goal {
		return Minuso(NewAtom(3), NewAtom(7), q)
	})
	fmt.Println(result[0])
	// Output: -4
}

// ExampleTimeso demonstrates basic multiplication with Timeso.
func ExampleTimeso() {
	result := Run(1, func(q *Var) Goal {
		return Timeso(NewAtom(4), NewAtom(5), q)
	})
	fmt.Println(result[0])
	// Output: 20
}

// ExampleTimeso_backward demonstrates solving for x in x * 6 = 24.
func ExampleTimeso_backward() {
	result := Run(1, func(q *Var) Goal {
		return Timeso(q, NewAtom(6), NewAtom(24))
	})
	fmt.Println(result[0])
	// Output: 4
}

// ExampleTimeso_notDivisible demonstrates that non-divisible cases fail.
func ExampleTimeso_notDivisible() {
	// ? * 3 = 10 has no integer solution
	result := Run(1, func(q *Var) Goal {
		return Timeso(q, NewAtom(3), NewAtom(10))
	})
	fmt.Println(len(result))
	// Output: 0
}

// ExampleDivo demonstrates integer division with Divo.
func ExampleDivo() {
	result := Run(1, func(q *Var) Goal {
		return Divo(NewAtom(15), NewAtom(3), q)
	})
	fmt.Println(result[0])
	// Output: 5
}

// ExampleDivo_integerDivision demonstrates truncation in integer division.
func ExampleDivo_integerDivision() {
	result := Run(1, func(q *Var) Goal {
		return Divo(NewAtom(7), NewAtom(2), q)
	})
	fmt.Println(result[0])
	// Output: 3
}

// ExampleDivo_backward demonstrates solving for x in x / 5 = 3.
func ExampleDivo_backward() {
	result := Run(1, func(q *Var) Goal {
		return Divo(q, NewAtom(5), NewAtom(3))
	})
	fmt.Println(result[0])
	// Output: 15
}

// ExampleExpo demonstrates exponentiation with Expo.
func ExampleExpo() {
	result := Run(1, func(q *Var) Goal {
		return Expo(NewAtom(2), NewAtom(10), q)
	})
	fmt.Println(result[0])
	// Output: 1024
}

// ExampleExpo_zeroExponent demonstrates that any number to the power of 0 is 1.
func ExampleExpo_zeroExponent() {
	result := Run(1, func(q *Var) Goal {
		return Expo(NewAtom(5), NewAtom(0), q)
	})
	fmt.Println(result[0])
	// Output: 1
}

// ExampleExpo_verification demonstrates verifying an exponential equation.
func ExampleExpo_verification() {
	result := Run(1, func(q *Var) Goal {
		return Conj(
			Expo(NewAtom(3), NewAtom(4), NewAtom(81)),
			Eq(q, NewAtom("correct")),
		)
	})
	fmt.Println(result[0])
	// Output: correct
}

// ExampleLogo demonstrates logarithm computation with Logo.
func ExampleLogo() {
	result := Run(1, func(q *Var) Goal {
		return Logo(NewAtom(2), NewAtom(1024), q)
	})
	fmt.Println(result[0])
	// Output: 10
}

// ExampleLogo_base10 demonstrates base-10 logarithm.
func ExampleLogo_base10() {
	result := Run(1, func(q *Var) Goal {
		return Logo(NewAtom(10), NewAtom(1000), q)
	})
	fmt.Println(result[0])
	// Output: 3
}

// ExampleLessThano demonstrates less-than comparison.
func ExampleLessThano() {
	result := Run(1, func(q *Var) Goal {
		return Conj(
			LessThano(NewAtom(3), NewAtom(5)),
			Eq(q, NewAtom("yes")),
		)
	})
	fmt.Println(result[0])
	// Output: yes
}

// ExampleLessThano_filter demonstrates filtering with LessThano.
// Goal order doesn't matter - constraints are declarative.
func ExampleLessThano_filter() {
	result := Run(10, func(q *Var) Goal {
		return Conj(
			LessThano(q, NewAtom(5)),
			Membero(q, List(NewAtom(1), NewAtom(3), NewAtom(7), NewAtom(2))),
		)
	})
	fmt.Printf("Found %d values\n", len(result))
	// Output: Found 3 values
}

// ExampleGreaterThano demonstrates greater-than comparison.
func ExampleGreaterThano() {
	result := Run(1, func(q *Var) Goal {
		return Conj(
			GreaterThano(NewAtom(10), NewAtom(5)),
			Eq(q, NewAtom("yes")),
		)
	})
	fmt.Println(result[0])
	// Output: yes
}

// ExampleLessEqualo demonstrates less-than-or-equal comparison.
func ExampleLessEqualo() {
	result := Run(1, func(q *Var) Goal {
		return Conj(
			LessEqualo(NewAtom(5), NewAtom(5)),
			Eq(q, NewAtom("yes")),
		)
	})
	fmt.Println(result[0])
	// Output: yes
}

// ExampleGreaterEqualo demonstrates greater-than-or-equal comparison.
func ExampleGreaterEqualo() {
	result := Run(1, func(q *Var) Goal {
		return Conj(
			GreaterEqualo(NewAtom(10), NewAtom(5)),
			Eq(q, NewAtom("yes")),
		)
	})
	fmt.Println(result[0])
	// Output: yes
}

// ExamplePluso_composition demonstrates composing arithmetic operations.
func ExamplePluso_composition() {
	// Solve (x + 3) * 2 = 10 for x
	result := Run(1, func(q *Var) Goal {
		temp := Fresh("temp")
		return Conj(
			Timeso(temp, NewAtom(2), NewAtom(10)), // temp = 5
			Pluso(q, NewAtom(3), temp),            // q + 3 = 5
		)
	})
	fmt.Println(result[0])
	// Output: 2
}

// ExamplePluso_chained demonstrates chaining multiple arithmetic operations.
func ExamplePluso_chained() {
	// x + y = 5, y + z = 7, with x = 2, solve for y and z
	result := Run(1, func(q *Var) Goal {
		x := Fresh("x")
		y := Fresh("y")
		z := Fresh("z")
		return Conj(
			Eq(x, NewAtom(2)),
			Pluso(x, y, NewAtom(5)),
			Pluso(y, z, NewAtom(7)),
			Eq(q, List(x, y, z)),
		)
	})

	// Extract list values
	list := result[0]
	var vals []Term
	for {
		if pair, ok := list.(*Pair); ok {
			vals = append(vals, pair.Car())
			list = pair.Cdr()
		} else {
			break
		}
	}
	fmt.Printf("x=%v, y=%v, z=%v\n", vals[0], vals[1], vals[2])
	// Output: x=2, y=3, z=4
}

// ExampleLessThano_withArithmetic demonstrates combining comparison with arithmetic.
func ExampleLessThano_withArithmetic() {
	// Find x where x + 2 < 10 and x = 3
	result := Run(1, func(q *Var) Goal {
		temp := Fresh("temp")
		return Conj(
			Eq(q, NewAtom(3)),
			Pluso(q, NewAtom(2), temp),
			LessThano(temp, NewAtom(10)),
		)
	})
	fmt.Println(result[0])
	// Output: 3
}
