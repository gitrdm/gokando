```go
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

```


