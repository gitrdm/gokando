// Package minikanren provides constraint programming abstractions.
// This file defines the Domain interface for representing finite domains
// over discrete values, enabling solver-agnostic constraint propagation.
package minikanren

import (
	"fmt"
	"math/bits"
	"strings"
)

// Domain represents a finite set of values that a variable can take.
// Domains are the fundamental building block of finite-domain constraint programming.
// All domain implementations must be immutable - operations return new domains
// rather than modifying in place, enabling efficient copy-on-write semantics.
//
// Domains support efficient operations for:
//   - Membership testing
//   - Value removal (pruning)
//   - Cardinality queries
//   - Set operations (intersection, union, complement)
//   - Iteration over values
//
// Thread safety: Domain implementations must be safe for concurrent read access.
// Write operations (which return new domains) are inherently safe as they don't
// modify existing domains.
type Domain interface {
	// Count returns the number of values in the domain.
	// An empty domain (Count() == 0) represents an inconsistent constraint state.
	Count() int

	// Has returns true if the domain contains the given value.
	// Values are 1-indexed integers in the range [1, MaxValue].
	Has(value int) bool

	// Remove returns a new domain with the specified value removed.
	// If the value is not present, returns a domain equal to the original.
	// Returns an empty domain if removing the value would leave no values.
	Remove(value int) Domain

	// IsSingleton returns true if the domain contains exactly one value.
	// Singleton domains represent variables that are effectively bound.
	IsSingleton() bool

	// SingletonValue returns the single value if IsSingleton() is true.
	// Behavior is undefined if the domain is not a singleton.
	SingletonValue() int

	// IterateValues calls the provided function for each value in the domain.
	// Values are passed in ascending order.
	// The function must not modify the domain during iteration.
	IterateValues(f func(value int))

	// Intersect returns a new domain containing only values present in both domains.
	// This is the primary operation for constraint propagation.
	Intersect(other Domain) Domain

	// Union returns a new domain containing all values from both domains.
	// Used for constructive constraints and domain expansion.
	Union(other Domain) Domain

	// Complement returns a new domain containing all values NOT in this domain,
	// within the valid range [1, MaxValue].
	Complement() Domain

	// Clone returns a copy of the domain.
	// For immutable implementations, this may return the same instance.
	Clone() Domain

	// Equal returns true if this domain contains exactly the same values as other.
	Equal(other Domain) bool

	// MaxValue returns the maximum possible value in this domain type.
	// All domains have values in the range [1, MaxValue].
	MaxValue() int

	// String returns a human-readable representation of the domain.
	String() string
}

// BitSetDomain is a compact, efficient implementation of Domain using bitsets.
// Values are 1-indexed in the range [1, maxValue]. Each value is represented
// by a single bit in a uint64 word array, providing O(1) membership testing
// and very fast set operations.
//
// Memory usage: (maxValue + 63) / 64 * 8 bytes
// Example: maxValue=100 uses 16 bytes (2 uint64 words)
//
// BitSetDomain is immutable - all operations return new instances rather than
// modifying in place. This enables efficient structural sharing and
// copy-on-write semantics for parallel search.
type BitSetDomain struct {
	maxValue int      // Maximum value (inclusive), typically 9 for Sudoku, higher for other problems
	words    []uint64 // Bit array: bit i represents value i+1
}

// NewBitSetDomain creates a new domain containing all values from 1 to maxValue (inclusive).
// maxValue must be positive.
func NewBitSetDomain(maxValue int) *BitSetDomain {
	if maxValue <= 0 {
		return &BitSetDomain{maxValue: 0, words: nil}
	}

	numWords := (maxValue + 63) / 64
	words := make([]uint64, numWords)

	// Set bits 0 to maxValue-1 (representing values 1 to maxValue)
	for i := 0; i < maxValue; i++ {
		wordIdx := i / 64
		bitOffset := uint(i % 64)
		words[wordIdx] |= 1 << bitOffset
	}

	return &BitSetDomain{
		maxValue: maxValue,
		words:    words,
	}
}

// NewBitSetDomainFromValues creates a domain containing only the specified values.
// Values outside [1, maxValue] are ignored.
func NewBitSetDomainFromValues(maxValue int, values []int) *BitSetDomain {
	if maxValue <= 0 {
		return &BitSetDomain{maxValue: 0, words: nil}
	}

	numWords := (maxValue + 63) / 64
	words := make([]uint64, numWords)

	for _, v := range values {
		if v >= 1 && v <= maxValue {
			wordIdx := (v - 1) / 64
			bitOffset := uint((v - 1) % 64)
			words[wordIdx] |= 1 << bitOffset
		}
	}

	return &BitSetDomain{
		maxValue: maxValue,
		words:    words,
	}
}

// Count returns the number of values in the domain.
// Uses hardware popcount instructions for efficiency (O(number of words)).
func (d *BitSetDomain) Count() int {
	count := 0
	for _, word := range d.words {
		count += bits.OnesCount64(word)
	}
	return count
}

// Has returns true if the domain contains the value.
// Values are 1-indexed. O(1) operation.
func (d *BitSetDomain) Has(value int) bool {
	if value < 1 || value > d.maxValue {
		return false
	}

	wordIdx := (value - 1) / 64
	bitOffset := uint((value - 1) % 64)
	return (d.words[wordIdx]>>bitOffset)&1 == 1
}

// Remove returns a new domain without the specified value.
// If the value is not present, returns an equivalent domain.
// O(number of words) due to array copy.
func (d *BitSetDomain) Remove(value int) Domain {
	if value < 1 || value > d.maxValue || !d.Has(value) {
		return d.Clone()
	}

	newWords := make([]uint64, len(d.words))
	copy(newWords, d.words)

	wordIdx := (value - 1) / 64
	bitOffset := uint((value - 1) % 64)
	newWords[wordIdx] &^= (1 << bitOffset)

	return &BitSetDomain{
		maxValue: d.maxValue,
		words:    newWords,
	}
}

// IsSingleton returns true if the domain contains exactly one value.
// O(number of words) operation.
func (d *BitSetDomain) IsSingleton() bool {
	return d.Count() == 1
}

// SingletonValue returns the single value in the domain.
// Panics if the domain is not a singleton.
// O(number of words) operation.
func (d *BitSetDomain) SingletonValue() int {
	for i, word := range d.words {
		if word != 0 {
			bitOffset := bits.TrailingZeros64(word)
			return i*64 + bitOffset + 1
		}
	}
	panic("SingletonValue called on non-singleton domain")
}

// IterateValues calls f for each value in the domain in ascending order.
// The function must not retain references to mutable state during iteration.
func (d *BitSetDomain) IterateValues(f func(value int)) {
	for wordIdx, word := range d.words {
		for word != 0 {
			// Extract lowest set bit
			lowestBit := word & -word
			bitOffset := bits.TrailingZeros64(word)
			value := wordIdx*64 + bitOffset + 1

			f(value)

			// Clear the lowest set bit
			word &^= lowestBit
		}
	}
}

// Intersect returns a new domain containing values in both this and other.
// This is the core operation for constraint propagation.
// O(number of words) operation.
func (d *BitSetDomain) Intersect(other Domain) Domain {
	otherBitSet, ok := other.(*BitSetDomain)
	if !ok || d.maxValue != otherBitSet.maxValue {
		// Different domain types or sizes - return empty domain
		return &BitSetDomain{
			maxValue: d.maxValue,
			words:    make([]uint64, len(d.words)),
		}
	}

	newWords := make([]uint64, len(d.words))
	for i := range d.words {
		newWords[i] = d.words[i] & otherBitSet.words[i]
	}

	return &BitSetDomain{
		maxValue: d.maxValue,
		words:    newWords,
	}
}

// Union returns a new domain containing values from both this and other.
// O(number of words) operation.
func (d *BitSetDomain) Union(other Domain) Domain {
	otherBitSet, ok := other.(*BitSetDomain)
	if !ok {
		return d.Clone()
	}

	maxLen := len(d.words)
	if len(otherBitSet.words) > maxLen {
		maxLen = len(otherBitSet.words)
	}

	newWords := make([]uint64, maxLen)

	for i := 0; i < len(d.words) && i < len(otherBitSet.words); i++ {
		newWords[i] = d.words[i] | otherBitSet.words[i]
	}

	// Copy remaining words from the longer domain
	if len(d.words) > len(otherBitSet.words) {
		copy(newWords[len(otherBitSet.words):], d.words[len(otherBitSet.words):])
	} else if len(otherBitSet.words) > len(d.words) {
		copy(newWords[len(d.words):], otherBitSet.words[len(d.words):])
	}

	return &BitSetDomain{
		maxValue: d.maxValue,
		words:    newWords,
	}
}

// Complement returns a new domain with all values NOT in this domain.
// Values are within the range [1, maxValue].
// O(number of words) operation.
func (d *BitSetDomain) Complement() Domain {
	newWords := make([]uint64, len(d.words))

	for i := range d.words {
		newWords[i] = ^d.words[i]
	}

	// Mask out bits beyond maxValue in the last word
	if d.maxValue%64 != 0 {
		lastWordBits := d.maxValue % 64
		mask := (uint64(1) << uint(lastWordBits)) - 1
		newWords[len(newWords)-1] &= mask
	}

	return &BitSetDomain{
		maxValue: d.maxValue,
		words:    newWords,
	}
}

// Clone returns a copy of the domain.
// O(number of words) operation.
func (d *BitSetDomain) Clone() Domain {
	newWords := make([]uint64, len(d.words))
	copy(newWords, d.words)

	return &BitSetDomain{
		maxValue: d.maxValue,
		words:    newWords,
	}
}

// Equal returns true if this domain contains exactly the same values as other.
// O(number of words) operation.
func (d *BitSetDomain) Equal(other Domain) bool {
	otherBitSet, ok := other.(*BitSetDomain)
	if !ok {
		return false
	}

	if d.maxValue != otherBitSet.maxValue || len(d.words) != len(otherBitSet.words) {
		return false
	}

	for i := range d.words {
		if d.words[i] != otherBitSet.words[i] {
			return false
		}
	}

	return true
}

// MaxValue returns the maximum value that can be in this domain.
func (d *BitSetDomain) MaxValue() int {
	return d.maxValue
}

// String returns a human-readable representation of the domain.
// Example: "{1,3,5,7,9}" or "{1..100}" for ranges.
func (d *BitSetDomain) String() string {
	count := d.Count()
	if count == 0 {
		return "{}"
	}

	var values []int
	d.IterateValues(func(v int) {
		values = append(values, v)
	})

	// Singleton is shown as {value}, not {value..value}
	if count == 1 {
		return fmt.Sprintf("{%d}", values[0])
	}

	// If many consecutive values, use range notation
	if d.isConsecutiveRange(values) {
		return fmt.Sprintf("{%d..%d}", values[0], values[len(values)-1])
	}

	// Otherwise list individual values
	var builder strings.Builder
	builder.WriteString("{")
	for i, v := range values {
		if i > 0 {
			builder.WriteString(",")
		}
		builder.WriteString(fmt.Sprintf("%d", v))

		// Truncate if too many values
		if i >= 19 && len(values) > 20 {
			builder.WriteString(fmt.Sprintf(",...+%d more", len(values)-20))
			break
		}
	}
	builder.WriteString("}")

	return builder.String()
}

// isConsecutiveRange checks if values form a consecutive range.
func (d *BitSetDomain) isConsecutiveRange(values []int) bool {
	if len(values) <= 1 {
		return true
	}

	for i := 1; i < len(values); i++ {
		if values[i] != values[i-1]+1 {
			return false
		}
	}

	return true
}

// ToSlice returns all values in the domain as a sorted slice.
// Useful for testing and debugging.
func (d *BitSetDomain) ToSlice() []int {
	var values []int
	d.IterateValues(func(v int) {
		values = append(values, v)
	})
	return values
}
