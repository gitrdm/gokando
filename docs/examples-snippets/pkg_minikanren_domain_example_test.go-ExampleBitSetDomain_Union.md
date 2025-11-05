```go
func ExampleBitSetDomain_Union() {
	// One constraint allows {1,2,3}
	domain1 := minikanren.NewBitSetDomainFromValues(10, []int{1, 2, 3})

	// Another constraint allows {3,4,5}
	domain2 := minikanren.NewBitSetDomainFromValues(10, []int{3, 4, 5})

	// Union gives all allowed values from either constraint
	union := domain1.Union(domain2)

	fmt.Printf("Domain 1: %s\n", domain1.String())
	fmt.Printf("Domain 2: %s\n", domain2.String())
	fmt.Printf("Union: %s\n", union.String())

	// Output:
	// Domain 1: {1..3}
	// Domain 2: {3..5}
	// Union: {1..5}
}

```


