```go
func ExampleModel_helpers_intVarValues() {
	m := NewModel()
	x := m.IntVarValues([]int{1, 3, 5}, "x")

	s := NewSolver(m)
	// Initial domain reflects the provided set exactly
	fmt.Println(s.GetDomain(nil, x.ID()))
	// Output:
	// {1,3,5}
}

```


