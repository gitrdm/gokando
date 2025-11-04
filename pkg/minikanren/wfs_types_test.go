package minikanren

import (
	"testing"
)

func TestDelaySet_Basic(t *testing.T) {
	ds := NewDelaySet()
	if !ds.Empty() {
		t.Fatalf("new DelaySet should be empty")
	}

	var dep1 uint64 = 42
	var dep2 uint64 = 99
	ds.Add(dep1)
	if ds.Empty() {
		t.Fatalf("DelaySet should not be empty after Add")
	}
	if !ds.Has(dep1) {
		t.Fatalf("DelaySet missing dep1")
	}
	if ds.Has(dep2) {
		t.Fatalf("DelaySet should not contain dep2")
	}

	ds2 := NewDelaySet()
	ds2.Add(dep2)
	ds.Merge(ds2)
	if !ds.Has(dep2) {
		t.Fatalf("DelaySet should contain dep2 after merge")
	}
}

func TestAnswerRecordIterator_WrapsAnswers(t *testing.T) {
	trie := NewAnswerTrie()
	ans1 := map[int64]Term{1: NewAtom(1)}
	ans2 := map[int64]Term{2: NewAtom(2)}
	if !trie.Insert(ans1) || !trie.Insert(ans2) {
		t.Fatalf("failed to insert answers")
	}

	// Provide delay sets by index: [0]->{7}, [1]->{}
	delayProvider := func(index int) DelaySet {
		if index == 0 {
			ds := NewDelaySet()
			ds.Add(7)
			return ds
		}
		return nil
	}

	it := NewAnswerRecordIterator(trie, delayProvider)
	rec, ok := it.Next()
	if !ok || len(rec.Bindings) != 1 || !rec.Delay.Has(7) {
		t.Fatalf("unexpected first record: %#v, ok=%v", rec, ok)
	}
	rec2, ok := it.Next()
	if !ok || len(rec2.Bindings) != 1 || (rec2.Delay != nil && !rec2.Delay.Empty()) {
		t.Fatalf("unexpected second record: %#v, ok=%v", rec2, ok)
	}
	_, ok = it.Next()
	if ok {
		t.Fatalf("expected iterator exhausted")
	}
}
