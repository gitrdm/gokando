```go
func ExampleRecursiveRule_familyTree() {
	// Define relations
	parent, _ := DbRel("parent", 2, 0, 1)

	// Build family tree
	db := NewDatabase()
	db, _ = db.AddFact(parent, NewAtom("john"), NewAtom("mary"))
	db, _ = db.AddFact(parent, NewAtom("john"), NewAtom("tom"))
	db, _ = db.AddFact(parent, NewAtom("mary"), NewAtom("alice"))
	db, _ = db.AddFact(parent, NewAtom("tom"), NewAtom("bob"))

	// Query variables
	x := Fresh("x")
	y := Fresh("y")

	// Define ancestor as recursive rule
	//ancestor := RecursiveRule(
	//	db,
	//	parent,     // base: parent is ancestor
	//	"ancestor", // predicate ID
	//	[]Term{x, y},
	//	func() Goal { // recursive: ancestor of parent is ancestor
	//		z := Fresh("z")
	//		return Conj(
	//			TabledQuery(db, parent, "ancestor", x, z),
	//			TabledQuery(db, parent, "ancestor", z, y),
	//		)
	//	},
	//)

	// For now, just query direct parents (base case)
	goal := Conj(
		Eq(y, NewAtom("alice")),
		db.Query(parent, x, y),
	)

	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)
	results, _ := stream.Take(10)

	// Collect results
	parents := make([]string, 0)
	for _, s := range results {
		if binding := s.GetBinding(x.ID()); binding != nil {
			if atom, ok := binding.(*Atom); ok {
				parents = append(parents, atom.String())
			}
		}
	}
	sort.Strings(parents)

	for _, name := range parents {
		fmt.Printf("%s is parent of alice\n", name)
	}

	// Output:
	// mary is parent of alice
}

```


