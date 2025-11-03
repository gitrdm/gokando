package minikanren

import (
	"testing"
)

func TestSubsumption_SpecificThenGeneral(t *testing.T) {
	pattern := NewCallPattern("p", []Term{NewAtom("x"), NewAtom("y")})
	entry := NewSubgoalEntry(pattern)

	// Insert specific answer
	ansSpecific := map[int64]Term{1: NewAtom("a"), 2: NewAtom("b")}
	if ok, idx := entry.InsertAnswerWithSubsumption(ansSpecific); !ok || idx != 0 {
		t.Fatalf("expected first insert to succeed at index 0, got ok=%v idx=%d", ok, idx)
	}

	// Insert general answer that subsumes the first one
	ansGeneral := map[int64]Term{1: NewAtom("a")}
	if ok, idx := entry.InsertAnswerWithSubsumption(ansGeneral); !ok || idx != 1 {
		t.Fatalf("expected second insert to succeed at index 1, got ok=%v idx=%d", ok, idx)
	}

	// The first (specific) should now be retracted
	if !entry.IsRetracted(0) {
		t.Fatalf("expected index 0 to be retracted after subsuming insert")
	}
	if entry.IsRetracted(1) {
		t.Fatalf("did not expect index 1 to be retracted")
	}

	// Iterator should only yield the general answer
	it := entry.AnswerRecords()
	seen := 0
	for {
		rec, ok := it.Next()
		if !ok {
			break
		}
		seen++
		if v, ok := rec.Bindings[1].(*Atom); !ok || v.Value() != "a" {
			t.Fatalf("unexpected binding for var 1: %v", rec.Bindings[1])
		}
		if _, exists := rec.Bindings[2]; exists {
			t.Fatalf("expected general answer to not bind var 2")
		}
	}
	if seen != 1 {
		t.Fatalf("expected 1 visible answer, got %d", seen)
	}
}

func TestSubsumption_GeneralThenSpecific(t *testing.T) {
	pattern := NewCallPattern("p", []Term{NewAtom("x"), NewAtom("y")})
	entry := NewSubgoalEntry(pattern)

	ansGeneral := map[int64]Term{1: NewAtom("a")}
	if ok, idx := entry.InsertAnswerWithSubsumption(ansGeneral); !ok || idx != 0 {
		t.Fatalf("expected first insert to succeed at index 0, got ok=%v idx=%d", ok, idx)
	}

	ansSpecific := map[int64]Term{1: NewAtom("a"), 2: NewAtom("b")}
	if ok, idx := entry.InsertAnswerWithSubsumption(ansSpecific); ok || idx != -1 {
		t.Fatalf("expected specific answer to be subsumed and skipped, got ok=%v idx=%d", ok, idx)
	}

	// Only the general answer should be visible
	it := entry.AnswerRecords()
	seen := 0
	for {
		_, ok := it.Next()
		if !ok {
			break
		}
		seen++
	}
	if seen != 1 {
		t.Fatalf("expected 1 visible answer, got %d", seen)
	}
}

func TestInvalidateByDomain(t *testing.T) {
	pattern := NewCallPattern("q", []Term{NewAtom("x")})
	entry := NewSubgoalEntry(pattern)

	varID := int64(1)
	// Insert three concrete answers for the same varID
	entry.InsertAnswerWithSubsumption(map[int64]Term{varID: NewAtom(1)})
	entry.InsertAnswerWithSubsumption(map[int64]Term{varID: NewAtom(2)})
	entry.InsertAnswerWithSubsumption(map[int64]Term{varID: NewAtom(3)})

	// Invalidate answers not in {1,3}
	dom := NewBitSetDomainFromValues(10, []int{1, 3})
	retracted := entry.InvalidateByDomain(varID, dom)
	if retracted != 1 {
		t.Fatalf("expected to retract 1 answer, got %d", retracted)
	}

	// Visible answers should be the ones with values 1 and 3
	it := entry.AnswerRecords()
	vals := make(map[int]bool)
	for {
		rec, ok := it.Next()
		if !ok {
			break
		}
		if a, ok := rec.Bindings[varID].(*Atom); ok {
			if iv, ok := a.Value().(int); ok {
				vals[iv] = true
			}
		}
	}
	if len(vals) != 2 || !vals[1] || !vals[3] {
		t.Fatalf("expected remaining values {1,3}, got %v", vals)
	}
}
