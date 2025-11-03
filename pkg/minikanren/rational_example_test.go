package minikanren

import "fmt"

// ExampleRational_arithmetic demonstrates basic rational number arithmetic.
func ExampleRational_arithmetic() {
	a := NewRational(1, 2) // 1/2
	b := NewRational(1, 3) // 1/3

	sum := a.Add(b)
	diff := a.Sub(b)
	product := a.Mul(b)
	quotient := a.Div(b)

	fmt.Printf("1/2 + 1/3 = %s\n", sum)
	fmt.Printf("1/2 - 1/3 = %s\n", diff)
	fmt.Printf("1/2 * 1/3 = %s\n", product)
	fmt.Printf("1/2 / 1/3 = %s\n", quotient)

	// Output:
	// 1/2 + 1/3 = 5/6
	// 1/2 - 1/3 = 1/6
	// 1/2 * 1/3 = 1/6
	// 1/2 / 1/3 = 3/2
}

// ExampleCommonIrrationals demonstrates using predefined irrational approximations.
func ExampleCommonIrrationals() {
	// Pi approximations
	fmt.Printf("π ≈ %s (Archimedes)\n", CommonIrrationals.PiArchimedes)
	fmt.Printf("π ≈ %s (Zu Chongzhi)\n", CommonIrrationals.PiZu)

	// Square root of 2
	fmt.Printf("√2 ≈ %s (simple)\n", CommonIrrationals.Sqrt2Simple)

	// Euler's number
	fmt.Printf("e ≈ %s\n", CommonIrrationals.ESimple)

	// Golden ratio
	fmt.Printf("φ ≈ %s\n", CommonIrrationals.PhiSimple)

	// Output:
	// π ≈ 22/7 (Archimedes)
	// π ≈ 355/113 (Zu Chongzhi)
	// √2 ≈ 99/70 (simple)
	// e ≈ 2721/1000
	// φ ≈ 809/500
}

// ExampleApproximateIrrational demonstrates getting irrational approximations by name.
func ExampleApproximateIrrational() {
	// Get pi with different precision levels
	piLow, _ := ApproximateIrrational("pi", 2)
	piHigh, _ := ApproximateIrrational("pi", 6)

	fmt.Printf("π (low precision): %s ≈ %.4f\n", piLow, piLow.ToFloat())
	fmt.Printf("π (high precision): %s ≈ %.6f\n", piHigh, piHigh.ToFloat())

	// Get sqrt(2)
	sqrt2, _ := ApproximateIrrational("sqrt2", 4)
	fmt.Printf("√2: %s ≈ %.4f\n", sqrt2, sqrt2.ToFloat())

	// Output:
	// π (low precision): 22/7 ≈ 3.1429
	// π (high precision): 355/113 ≈ 3.141593
	// √2: 99/70 ≈ 1.4143
}

// ExampleRational_circumference demonstrates using rational π for circle calculations.
func ExampleRational_circumference() {
	// Calculate circumference = π * diameter using rational approximation
	pi := CommonIrrationals.PiArchimedes // 22/7
	diameter := NewRational(14, 1)       // diameter = 14

	circumference := pi.Mul(diameter)

	fmt.Printf("For diameter = %s\n", diameter)
	fmt.Printf("Circumference = π × d = %s × %s = %s\n", pi, diameter, circumference)
	fmt.Printf("Circumference ≈ %.2f\n", circumference.ToFloat())

	// Output:
	// For diameter = 14
	// Circumference = π × d = 22/7 × 14 = 44
	// Circumference ≈ 44.00
}
