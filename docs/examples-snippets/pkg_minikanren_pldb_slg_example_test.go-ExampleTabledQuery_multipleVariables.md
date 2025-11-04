```go
func ExampleTabledQuery_multipleVariables() {
	person, _ := DbRel("person", 3, 0, 1, 2) // name, age, city
	db := NewDatabase()
	db, _ = db.AddFact(person, NewAtom("alice"), NewAtom(30), NewAtom("nyc"))
	db, _ = db.AddFact(person, NewAtom("bob"), NewAtom(25), NewAtom("sf"))
	db, _ = db.AddFact(person, NewAtom("charlie"), NewAtom(35), NewAtom("nyc"))

	name := Fresh("name")
	age := Fresh("age")
	city := Fresh("city")

	// Query all fields
	goal := TabledQuery(db, person, "person", name, age, city)

	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)
	results, _ := stream.Take(10)

	// Collect names for consistent output
	names := make([]string, 0, len(results))
	for _, s := range results {
		if n := s.GetBinding(name.ID()); n != nil {
			if atom, ok := n.(*Atom); ok {
				names = append(names, atom.Value().(string))
			}
		}
	}
	sort.Strings(names)

	fmt.Printf("Found people: %v\n", names)

	// Output:
	// Found people: [alice bob charlie]
}

```


