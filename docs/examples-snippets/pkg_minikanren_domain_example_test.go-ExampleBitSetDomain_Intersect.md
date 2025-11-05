```go
func ExampleBitSetDomain_Intersect() {
	// Variable must be in {1,2,3,4,5} from one constraint
	domain1 := minikanren.NewBitSetDomainFromValues(10, []int{1, 2, 3, 4, 5})

	// Variable must be in {3,4,5,6,7} from another constraint
	domain2 := minikanren.NewBitSetDomainFromValues(10, []int{3, 4, 5, 6, 7})

	// Intersection gives values satisfying both constraints
	intersection := domain1.Intersect(domain2)

	fmt.Printf("Domain 1: %s\n", domain1.String())
	fmt.Printf("Domain 2: %s\n", domain2.String())
	fmt.Printf("Intersection: %s\n", intersection.String())

	// Output:
	// Domain 1: {1..5}
	// Domain 2: {3..7}
	// Intersection: {3..5}
}

```


