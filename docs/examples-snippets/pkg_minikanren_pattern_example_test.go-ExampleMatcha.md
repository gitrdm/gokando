```go
func ExampleMatcha() {
	// Safe head extraction with default value
	extractHead := func(list Term) Term {
		return Run(1, func(q *Var) Goal {
			head := Fresh("head")
			return Matcha(list,
				NewClause(Nil, Eq(q, NewAtom("empty"))),
				NewClause(NewPair(head, Fresh("_")), Eq(q, head)),
			)
		})[0]
	}

	// Non-empty list
	list1 := List(NewAtom(42), NewAtom(99))
	fmt.Println(extractHead(list1))

	// Empty list
	list2 := Nil
	fmt.Println(extractHead(list2))

	// Output:
	// 42
	// empty
}

```


