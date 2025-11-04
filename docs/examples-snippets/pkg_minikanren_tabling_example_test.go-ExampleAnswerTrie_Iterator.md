```go
func ExampleAnswerTrie_Iterator() {
	trie := NewAnswerTrie()

	// Insert multiple answers
	for i := 1; i <= 3; i++ {
		bindings := map[int64]Term{
			1: NewAtom(fmt.Sprintf("value%d", i)),
		}
		trie.Insert(bindings)
	}

	// Iterate over all answers
	iter := trie.Iterator()
	count := 0
	for {
		answer, ok := iter.Next()
		if !ok {
			break
		}
		count++
		// Note: iteration order is not guaranteed
		fmt.Printf("Answer has %d bindings\n", len(answer))
	}

	fmt.Printf("Total answers iterated: %d\n", count)

	// Output:
	// Answer has 1 bindings
	// Answer has 1 bindings
	// Answer has 1 bindings
	// Total answers iterated: 3
}

```


