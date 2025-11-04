```go
func ExampleTabledQuery_multiRelation() {
	employee, _ := DbRel("employee", 2, 0, 1) // (name, dept)
	manager, _ := DbRel("manager", 2, 0, 1)   // (mgr, employee)

	db := NewDatabase()
	db, _ = db.AddFact(employee, NewAtom("alice"), NewAtom("engineering"))
	db, _ = db.AddFact(employee, NewAtom("bob"), NewAtom("engineering"))
	db, _ = db.AddFact(manager, NewAtom("bob"), NewAtom("alice"))

	// Who manages Alice?
	mgr := Fresh("mgr")
	goal := Conj(
		TabledQuery(db, manager, "mgr", mgr, NewAtom("alice")),
		TabledQuery(db, employee, "emp", mgr, Fresh("_")), // ensure mgr is an employee
	)

	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)
	results, _ := stream.Take(10)

	if len(results) > 0 {
		binding := results[0].GetBinding(mgr.ID())
		if atom, ok := binding.(*Atom); ok {
			fmt.Printf("%s manages alice\n", atom.String())
		}
	}

	// Output:
	// bob manages alice
}

```


