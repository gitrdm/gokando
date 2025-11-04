```go
func ExampleDatabase_RemoveFact() {
	person, _ := DbRel("person", 1, 0)

	// Create database with some people using low-level API versus the HLAPI for demonstration
	db := NewDatabase()
	db, _ = db.AddFact(person, NewAtom("alice"))
	db, _ = db.AddFact(person, NewAtom("bob"))
	db, _ = db.AddFact(person, NewAtom("charlie"))

	fmt.Printf("Before removal: %d people\n", db.FactCount(person))

	// Remove bob
	db2, _ := db.RemoveFact(person, NewAtom("bob"))

	fmt.Printf("After removal:  %d people\n", db2.FactCount(person))

	// Original database unchanged
	fmt.Printf("Original still: %d people\n", db.FactCount(person))

	// Facts can be re-added
	db3, _ := db2.AddFact(person, NewAtom("bob"))
	fmt.Printf("After re-add:   %d people\n", db3.FactCount(person))

	// Output:
	// Before removal: 3 people
	// After removal:  2 people
	// Original still: 3 people
	// After re-add:   3 people
}

```


