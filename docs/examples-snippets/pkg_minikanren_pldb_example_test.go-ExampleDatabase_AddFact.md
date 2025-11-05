```go
func ExampleDatabase_AddFact() {
	parent, _ := DbRel("parent", 2, 0, 1)

	// Start with an empty database
	db := NewDatabase()

	// Add facts using copy-on-write semantics
	db1, _ := db.AddFact(parent, NewAtom("alice"), NewAtom("bob"))
	db2, _ := db1.AddFact(parent, NewAtom("bob"), NewAtom("charlie"))
	db3, _ := db2.AddFact(parent, NewAtom("alice"), NewAtom("diana"))

	// Each version maintains its own state
	fmt.Printf("Original: %d facts\n", db.FactCount(parent))
	fmt.Printf("After 1:  %d facts\n", db1.FactCount(parent))
	fmt.Printf("After 2:  %d facts\n", db2.FactCount(parent))
	fmt.Printf("After 3:  %d facts\n", db3.FactCount(parent))

	// Output:
	// Original: 0 facts
	// After 1:  1 facts
	// After 2:  2 facts
	// After 3:  3 facts
}

```


