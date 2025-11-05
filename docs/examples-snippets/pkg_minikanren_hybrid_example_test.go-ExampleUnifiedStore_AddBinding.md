```go
func ExampleUnifiedStore_AddBinding() {
	store := NewUnifiedStore()

	// Add bindings (using variable IDs 1 and 2)
	store, _ = store.AddBinding(1, NewAtom("hello"))
	store, _ = store.AddBinding(2, NewAtom(42))

	// Retrieve bindings
	xBinding := store.GetBinding(1)
	yBinding := store.GetBinding(2)

	fmt.Printf("var 1 = %v\n", xBinding.(*Atom).Value())
	fmt.Printf("var 2 = %v\n", yBinding.(*Atom).Value())

	// Output:
	// var 1 = hello
	// var 2 = 42
}

```


