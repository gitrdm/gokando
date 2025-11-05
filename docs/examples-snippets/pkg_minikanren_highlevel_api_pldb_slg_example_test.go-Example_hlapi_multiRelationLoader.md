```go
func Example_hlapi_multiRelationLoader() {
	emp, mgr := MustRel("employee", 2, 0, 1), MustRel("manager", 2, 0, 1)
	rels := map[string]*Relation{"employee": emp, "manager": mgr}
	data := map[string][][]interface{}{
		"employee": {{"alice", "eng"}, {"bob", "eng"}},
		"manager":  {{"bob", "alice"}},
	}
	// Load both relations in one pass
	db, _ := NewDBFromMap(rels, data)

	mgrVar := Fresh("mgr")
	goal := TQ(db, mgr, mgrVar, "alice")

	ctx := context.Background()
	stores := goal(ctx, NewLocalConstraintStore(NewGlobalConstraintBus()))
	rows, _ := stores.Take(10)
	fmt.Println(len(rows))
	// Output:
	// 1
}

```


