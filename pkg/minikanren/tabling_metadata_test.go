package minikanren

import (
	"testing"
)

func TestSubgoalEntry_MetadataStorage(t *testing.T) {
	pattern := NewCallPattern("test", []Term{NewAtom(1)})
	entry := NewSubgoalEntry(pattern)

	// Initially no metadata
	if ds := entry.DelaySetFor(0); ds != nil {
		t.Fatalf("expected nil delay set for unmapped index, got %v", ds)
	}

	// Attach a delay set
	ds := NewDelaySet()
	ds.Add(42)
	ds.Add(99)
	entry.AttachDelaySet(0, ds)

	// Retrieve it
	retrieved := entry.DelaySetFor(0)
	if retrieved == nil || !retrieved.Has(42) || !retrieved.Has(99) {
		t.Fatalf("retrieved delay set missing expected values: %v", retrieved)
	}

	// Mutating returned copy shouldn't affect stored
	retrieved.Add(777)
	retrieved2 := entry.DelaySetFor(0)
	if retrieved2.Has(777) {
		t.Fatalf("external mutation leaked into stored delay set")
	}

	// Nil/empty delay sets are not stored
	entry.AttachDelaySet(1, nil)
	entry.AttachDelaySet(2, NewDelaySet())
	if entry.DelaySetFor(1) != nil || entry.DelaySetFor(2) != nil {
		t.Fatalf("nil/empty delay sets should not be stored")
	}
}

func TestSubgoalEntry_AnswerRecords(t *testing.T) {
	pattern := NewCallPattern("pred", []Term{NewAtom("X")})
	entry := NewSubgoalEntry(pattern)

	// Insert answers
	ans1 := map[int64]Term{1: NewAtom(10)}
	ans2 := map[int64]Term{1: NewAtom(20)}
	ans3 := map[int64]Term{1: NewAtom(30)}
	entry.Answers().Insert(ans1)
	entry.Answers().Insert(ans2)
	entry.Answers().Insert(ans3)

	// Attach delay set to second answer only
	ds := NewDelaySet()
	ds.Add(555)
	entry.AttachDelaySet(1, ds)

	// Iterate and verify
	it := entry.AnswerRecords()
	rec, ok := it.Next()
	if !ok || rec.Delay != nil {
		t.Fatalf("first record should be unconditional, got delay=%v", rec.Delay)
	}
	rec, ok = it.Next()
	if !ok || rec.Delay == nil || !rec.Delay.Has(555) {
		t.Fatalf("second record should have delay set with 555, got %v", rec.Delay)
	}
	rec, ok = it.Next()
	if !ok || rec.Delay != nil {
		t.Fatalf("third record should be unconditional, got delay=%v", rec.Delay)
	}
	_, ok = it.Next()
	if ok {
		t.Fatalf("iterator should be exhausted")
	}
}

func TestSubgoalEntry_AnswerRecordsFrom(t *testing.T) {
	pattern := NewCallPattern("pred", []Term{})
	entry := NewSubgoalEntry(pattern)

	// Insert 5 answers
	for i := 0; i < 5; i++ {
		entry.Answers().Insert(map[int64]Term{int64(i): NewAtom(i * 10)})
	}

	// Attach delay to indices 1 and 3
	ds1 := NewDelaySet()
	ds1.Add(111)
	entry.AttachDelaySet(1, ds1)
	ds3 := NewDelaySet()
	ds3.Add(333)
	entry.AttachDelaySet(3, ds3)

	// Start from index 2
	it := entry.AnswerRecordsFrom(2)
	rec, ok := it.Next()
	if !ok || rec.Delay != nil {
		t.Fatalf("index 2 should be unconditional")
	}
	rec, ok = it.Next()
	if !ok || !rec.Delay.Has(333) {
		t.Fatalf("index 3 should have delay 333")
	}
	rec, ok = it.Next()
	if !ok || rec.Delay != nil {
		t.Fatalf("index 4 should be unconditional")
	}
	_, ok = it.Next()
	if ok {
		t.Fatalf("should be exhausted after index 4")
	}
}

func TestSubgoalEntry_MetadataConcurrency(t *testing.T) {
	pattern := NewCallPattern("concurrent", []Term{})
	entry := NewSubgoalEntry(pattern)

	// Insert answers
	for i := 0; i < 100; i++ {
		entry.Answers().Insert(map[int64]Term{int64(i): NewAtom(i)})
	}

	// Concurrently attach and read metadata
	done := make(chan bool, 2)
	go func() {
		for i := 0; i < 100; i++ {
			ds := NewDelaySet()
			ds.Add(uint64(i))
			entry.AttachDelaySet(i, ds)
		}
		done <- true
	}()
	go func() {
		for i := 0; i < 100; i++ {
			_ = entry.DelaySetFor(i)
		}
		done <- true
	}()
	<-done
	<-done

	// Verify no corruption
	for i := 0; i < 100; i++ {
		ds := entry.DelaySetFor(i)
		if ds != nil && !ds.Has(uint64(i)) {
			t.Fatalf("index %d has corrupted delay set: %v", i, ds)
		}
	}
}
