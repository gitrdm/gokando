```go
func ExampleFDVariable_IsBound() {
	domain := minikanren.NewBitSetDomain(10)
	variable := minikanren.NewFDVariable(0, domain)

	fmt.Printf("Unbound variable: IsBound=%v\n", variable.IsBound())

	// Create a singleton domain (bound variable)
	singletonDomain := minikanren.NewBitSetDomainFromValues(10, []int{5})
	boundVariable := minikanren.NewFDVariable(1, singletonDomain)

	fmt.Printf("Bound variable: IsBound=%v, Value=%d\n", boundVariable.IsBound(), boundVariable.Value())

	// Output:
	// Unbound variable: IsBound=false
	// Bound variable: IsBound=true, Value=5
}

```


