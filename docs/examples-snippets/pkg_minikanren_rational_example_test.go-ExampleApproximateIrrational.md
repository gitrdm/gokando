```go
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

```


