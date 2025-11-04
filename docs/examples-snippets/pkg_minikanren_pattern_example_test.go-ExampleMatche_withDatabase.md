```go
func ExampleMatche_withDatabase() {
	// Create a relation for shapes
	shape, _ := DbRel("shape", 2, 0)
	db := NewDatabase()
	db, _ = db.AddFact(shape, NewAtom("circle"), NewAtom(10))
	db, _ = db.AddFact(shape, NewAtom("square"), NewAtom(5))
	db, _ = db.AddFact(shape, NewAtom("triangle"), NewAtom(3))

	// Query and pattern match on shape type
	name := Fresh("name")
	size := Fresh("size")

	result := Run(10, func(q *Var) Goal {
		return Conj(
			db.Query(shape, name, size),
			Matche(name,
				NewClause(NewAtom("circle"), Eq(q, NewAtom("round"))),
				NewClause(NewAtom("square"), Eq(q, NewAtom("angular"))),
				NewClause(NewAtom("triangle"), Eq(q, NewAtom("angular"))),
			),
		)
	})

	// Count results
	fmt.Printf("Matched %d shapes\n", len(result))

	// Output:
	// Matched 3 shapes
}

```


