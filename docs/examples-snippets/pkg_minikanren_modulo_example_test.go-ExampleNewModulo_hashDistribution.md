```go
func ExampleNewModulo_hashDistribution() {
	model := NewModel()

	// Variables: hash value and bucket assignment
	hashValue := model.NewVariable(NewBitSetDomainFromValues(101, []int{23, 47, 89, 156, 234})) // hash values
	bucket := model.NewVariable(NewBitSetDomainFromValues(9, rangeValues(1, 8)))                // 8 buckets

	// Constraint: bucket = hash_value mod 8
	constraint, err := NewModulo(hashValue, 8, bucket)
	if err != nil {
		panic(err)
	}

	model.AddConstraint(constraint)

	// Solve
	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	// Get final domains after propagation
	hashDomain := solver.GetDomain(nil, hashValue.ID())
	bucketDomain := solver.GetDomain(nil, bucket.ID())

	fmt.Printf("Hash values:")
	hashDomain.IterateValues(func(v int) {
		fmt.Printf(" %d", v)
	})
	fmt.Printf("\nBucket assignments:")
	bucketDomain.IterateValues(func(v int) {
		fmt.Printf(" %d", v)
	})
	fmt.Printf("\n")

	// Output:
	// Hash values: 23 47 89
	// Bucket assignments: 1 7
}

```


