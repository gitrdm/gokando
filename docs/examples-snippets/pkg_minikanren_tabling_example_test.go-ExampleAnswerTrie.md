```go
func ExampleAnswerTrie() {
	trie := NewAnswerTrie()

	// Insert first answer: {1: a, 2: b}
	answer1 := map[int64]Term{
		1: NewAtom("a"),
		2: NewAtom("b"),
	}
	inserted := trie.Insert(answer1)
	fmt.Printf("First answer inserted: %v\n", inserted)
	fmt.Printf("Count: %d\n", trie.Count())

	// Insert duplicate
	duplicate := map[int64]Term{
		1: NewAtom("a"),
		2: NewAtom("b"),
	}
	inserted = trie.Insert(duplicate)
	fmt.Printf("Duplicate inserted: %v\n", inserted)
	fmt.Printf("Count: %d\n", trie.Count())

	// Insert different answer: {1: a, 2: c}
	answer2 := map[int64]Term{
		1: NewAtom("a"),
		2: NewAtom("c"),
	}
	inserted = trie.Insert(answer2)
	fmt.Printf("Different answer inserted: %v\n", inserted)
	fmt.Printf("Final count: %d\n", trie.Count())

	// Output:
	// First answer inserted: true
	// Count: 1
	// Duplicate inserted: false
	// Count: 1
	// Different answer inserted: true
	// Final count: 2
}

```


