```go
func ExampleBitSetDomain_IterateValues() {
	domain := minikanren.NewBitSetDomainFromValues(10, []int{2, 5, 7, 9})

	fmt.Print("Values: ")
	domain.IterateValues(func(v int) {
		fmt.Printf("%d ", v)
	})
	fmt.Println()

	// Output:
	// Values: 2 5 7 9
}

```


