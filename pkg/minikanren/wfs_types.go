package minikanren

// WFS scaffolding: types and iterators to support conditional answers with delay sets.
//
// This file introduces minimal, backwards-compatible structures to carry
// well-founded semantics (WFS) metadata alongside existing answer bindings.
// It does not change the storage layout of AnswerTrie; instead, it provides
// an optional metadata-aware iterator that can be wired to a delay provider.

// DelaySet represents the set of negatively depended-on subgoals (by key/hash)
// that must be resolved before an answer can be considered unconditional.
// Keys are the CallPattern hash values of the depended subgoals.
type DelaySet map[uint64]struct{}

// NewDelaySet creates an empty delay set.
func NewDelaySet() DelaySet { return make(DelaySet) }

// Add inserts a dependency into the set.
func (ds DelaySet) Add(dep uint64) { ds[dep] = struct{}{} }

// Has checks membership.
func (ds DelaySet) Has(dep uint64) bool { _, ok := ds[dep]; return ok }

// Empty reports whether the set is empty.
func (ds DelaySet) Empty() bool { return len(ds) == 0 }

// Merge unions other into ds in-place.
func (ds DelaySet) Merge(other DelaySet) {
	for k := range other {
		ds[k] = struct{}{}
	}
}

// AnswerRecord bundles an answer's bindings with its WFS delay set.
// If Delay is empty, the answer is unconditional.
type AnswerRecord struct {
	Bindings map[int64]Term
	Delay    DelaySet
}

// AnswerRecordIterator is a metadata-aware iterator that wraps the existing
// AnswerIterator and pairs each binding with a DelaySet provided by a callback.
// The callback allows us to wire per-answer metadata later without changing
// the current AnswerTrie layout.
type AnswerRecordIterator struct {
	inner         *AnswerIterator
	startIndex    int
	delayProvider func(index int) DelaySet // nil provider yields empty delay sets
}

// Next returns the next AnswerRecord or ok=false when exhausted.
func (it *AnswerRecordIterator) Next() (rec AnswerRecord, ok bool) {
	bindings, ok := it.inner.Next()
	if !ok {
		return AnswerRecord{}, false
	}
	idx := it.startIndex
	it.startIndex++
	var ds DelaySet
	if it.delayProvider != nil {
		ds = it.delayProvider(idx)
	} else {
		ds = nil
	}
	return AnswerRecord{Bindings: bindings, Delay: ds}, true
}

// NewAnswerRecordIterator constructs a metadata-aware iterator over the given
// trie starting at index 0. Delay metadata is supplied by delayProvider; pass nil
// to provide empty delay sets (unconditional semantics).
func NewAnswerRecordIterator(trie *AnswerTrie, delayProvider func(index int) DelaySet) *AnswerRecordIterator {
	return &AnswerRecordIterator{
		inner:         trie.Iterator(),
		startIndex:    0,
		delayProvider: delayProvider,
	}
}

// NewAnswerRecordIteratorFrom constructs a metadata-aware iterator starting at start.
func NewAnswerRecordIteratorFrom(trie *AnswerTrie, start int, delayProvider func(index int) DelaySet) *AnswerRecordIterator {
	return &AnswerRecordIterator{
		inner:         trie.IteratorFrom(start),
		startIndex:    start,
		delayProvider: delayProvider,
	}
}
