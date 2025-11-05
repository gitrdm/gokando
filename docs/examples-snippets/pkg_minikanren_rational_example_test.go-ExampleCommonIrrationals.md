```go
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

```


