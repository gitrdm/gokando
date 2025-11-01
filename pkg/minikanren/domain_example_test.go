package minikanren_test

import (
	"fmt"

	"github.com/gitrdm/gokando/pkg/minikanren"
)

// ExampleNewBitSetDomain demonstrates creating a finite domain with all values from 1 to n.
// This is the most common way to create domains for CSP problems.
func ExampleNewBitSetDomain() {
	// Create a domain for Sudoku: values 1 through 9
	domain := minikanren.NewBitSetDomain(9)

	fmt.Printf("Domain size: %d\n", domain.Count())
	fmt.Printf("Contains 5: %v\n", domain.Has(5))
	fmt.Printf("Contains 0: %v\n", domain.Has(0))
	fmt.Printf("Contains 10: %v\n", domain.Has(10))

	// Output:
	// Domain size: 9
	// Contains 5: true
	// Contains 0: false
	// Contains 10: false
}

// ExampleNewBitSetDomainFromValues demonstrates creating a domain with specific values.
// Useful for modeling problems with irregular value sets.
func ExampleNewBitSetDomainFromValues() {
	// Create a domain with only even digits
	evenDigits := minikanren.NewBitSetDomainFromValues(9, []int{2, 4, 6, 8})

	fmt.Printf("Domain: %s\n", evenDigits.String())
	fmt.Printf("Size: %d\n", evenDigits.Count())
	fmt.Printf("Has 3: %v\n", evenDigits.Has(3))
	fmt.Printf("Has 4: %v\n", evenDigits.Has(4))

	// Output:
	// Domain: {2,4,6,8}
	// Size: 4
	// Has 3: false
	// Has 4: true
}

// ExampleBitSetDomain_Remove demonstrates domain pruning, the fundamental
// operation in constraint propagation.
func ExampleBitSetDomain_Remove() {
	domain := minikanren.NewBitSetDomain(5)
	fmt.Printf("Initial: %s\n", domain.String())

	// Remove value 3
	domain = domain.Remove(3).(*minikanren.BitSetDomain)
	fmt.Printf("After removing 3: %s\n", domain.String())

	// Remove value 5
	domain = domain.Remove(5).(*minikanren.BitSetDomain)
	fmt.Printf("After removing 5: %s\n", domain.String())

	// Output:
	// Initial: {1..5}
	// After removing 3: {1,2,4,5}
	// After removing 5: {1,2,4}
}

// ExampleBitSetDomain_Intersect demonstrates domain intersection, used when
// multiple constraints restrict the same variable.
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

// ExampleBitSetDomain_IsSingleton demonstrates checking if a variable is bound.
// When a domain becomes singleton, the variable effectively has a single value.
func ExampleBitSetDomain_IsSingleton() {
	domain := minikanren.NewBitSetDomain(5)
	fmt.Printf("Initial domain %s is singleton: %v\n", domain.String(), domain.IsSingleton())

	// Prune until singleton
	domain = domain.Remove(1).(*minikanren.BitSetDomain)
	domain = domain.Remove(2).(*minikanren.BitSetDomain)
	domain = domain.Remove(4).(*minikanren.BitSetDomain)
	domain = domain.Remove(5).(*minikanren.BitSetDomain)

	fmt.Printf("Domain %s is singleton: %v\n", domain.String(), domain.IsSingleton())
	if domain.IsSingleton() {
		fmt.Printf("Value: %d\n", domain.SingletonValue())
	}

	// Output:
	// Initial domain {1..5} is singleton: false
	// Domain {3} is singleton: true
	// Value: 3
}

// ExampleBitSetDomain_IterateValues demonstrates iterating over domain values.
// Values are always provided in ascending order for deterministic behavior.
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

// ExampleBitSetDomain_Union demonstrates combining domains from multiple sources.
// Useful for disjunctive constraints or relaxation.
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

// ExampleBitSetDomain_Complement demonstrates domain complement.
// Useful for expressing negative constraints (X cannot be in set S).
func ExampleBitSetDomain_Complement() {
	// Domain {2,4,6,8} within range 1-10
	evenDigits := minikanren.NewBitSetDomainFromValues(10, []int{2, 4, 6, 8})

	// Complement gives odd digits plus 10
	oddDigits := evenDigits.Complement()

	fmt.Printf("Even: %s\n", evenDigits.String())
	fmt.Printf("Odd: %s\n", oddDigits.String())

	// Output:
	// Even: {2,4,6,8}
	// Odd: {1,3,5,7,9,10}
}
