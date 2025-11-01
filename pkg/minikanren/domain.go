// Package minikanren provides constraint programming abstractions.
// This file defines the Domain interface for representing finite domains
// over discrete values, enabling solver-agnostic constraint propagation.
package minikanren

import (
	"fmt"
	"math/bits"
	"strings"
	"sync"
)

// Domain pools for reducing allocations during constraint propagation.
// Separate pools for common domain sizes to minimize allocation overhead.
var (
	// Pool for small domains (1-64 values) - most common in Sudoku, N-Queens
	smallDomainPool = sync.Pool{
		New: func() interface{} {
			return &BitSetDomain{words: make([]uint64, 1)}
		},
	}

	// Pool for medium domains (65-128 values)
	mediumDomainPool = sync.Pool{
		New: func() interface{} {
			return &BitSetDomain{words: make([]uint64, 2)}
		},
	}

	// Pool for large domains (129-256 values)
	largeDomainPool = sync.Pool{
		New: func() interface{} {
			return &BitSetDomain{words: make([]uint64, 4)}
		},
	}
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

	// RemoveAbove returns a new domain with all values > threshold removed.
	// Efficient bulk operation for inequality constraints.
	// Example: domain {1,2,3,4,5}.RemoveAbove(3) = {1,2,3}
	RemoveAbove(threshold int) Domain

	// RemoveBelow returns a new domain with all values < threshold removed.
	// Efficient bulk operation for inequality constraints.
	// Example: domain {1,2,3,4,5}.RemoveBelow(3) = {3,4,5}
	RemoveBelow(threshold int) Domain

	// RemoveAtOrAbove returns a new domain with all values >= threshold removed.
	// Efficient bulk operation for inequality constraints.
	// Example: domain {1,2,3,4,5}.RemoveAtOrAbove(3) = {1,2}
	RemoveAtOrAbove(threshold int) Domain

	// RemoveAtOrBelow returns a new domain with all values <= threshold removed.
	// Efficient bulk operation for inequality constraints.
	// Example: domain {1,2,3,4,5}.RemoveAtOrBelow(3) = {4,5}
	RemoveAtOrBelow(threshold int) Domain

	// Min returns the minimum value in the domain.
	// Returns 0 if domain is empty.
	Min() int

	// Max returns the maximum value in the domain.
	// Returns 0 if domain is empty.
	Max() int

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

// getDomainFromPool retrieves a BitSetDomain from the appropriate pool.
// Returns nil if domain would be too large for pooling.
func getDomainFromPool(maxValue int) *BitSetDomain {
	numWords := (maxValue + 63) / 64

	var d *BitSetDomain
	switch {
	case numWords == 1:
		d = smallDomainPool.Get().(*BitSetDomain)
	case numWords == 2:
		d = mediumDomainPool.Get().(*BitSetDomain)
	case numWords <= 4:
		d = largeDomainPool.Get().(*BitSetDomain)
	default:
		// Domain too large for pooling, allocate directly
		return nil
	}

	// Ensure words slice has correct capacity
	if cap(d.words) < numWords {
		d.words = make([]uint64, numWords)
	} else {
		d.words = d.words[:numWords]
	}

	// Clear all words
	for i := range d.words {
		d.words[i] = 0
	}

	d.maxValue = maxValue
	return d
}

// releaseDomainToPool returns a BitSetDomain to the appropriate pool for reuse.
func releaseDomainToPool(d *BitSetDomain) {
	if d == nil || d.words == nil {
		return
	}

	numWords := len(d.words)
	switch {
	case numWords == 1:
		smallDomainPool.Put(d)
	case numWords == 2:
		mediumDomainPool.Put(d)
	case numWords <= 4:
		largeDomainPool.Put(d)
		// Domains > 4 words are not pooled, will be garbage collected
	}
}

// NewBitSetDomain creates a new domain containing all values from 1 to maxValue (inclusive).
// maxValue must be positive. Uses object pooling for common domain sizes.
func NewBitSetDomain(maxValue int) *BitSetDomain {
	if maxValue <= 0 {
		return &BitSetDomain{maxValue: 0, words: nil}
	}

	// Try to get from pool
	d := getDomainFromPool(maxValue)
	if d == nil {
		// Too large for pool, allocate directly
		numWords := (maxValue + 63) / 64
		d = &BitSetDomain{
			maxValue: maxValue,
			words:    make([]uint64, numWords),
		}
	}

	// Set bits 0 to maxValue-1 (representing values 1 to maxValue)
	for i := 0; i < maxValue; i++ {
		wordIdx := i / 64
		bitOffset := uint(i % 64)
		d.words[wordIdx] |= 1 << bitOffset
	}

	return d
}

// NewBitSetDomainFromValues creates a domain containing only the specified values.
// Values outside [1, maxValue] are ignored. Uses object pooling for common domain sizes.
func NewBitSetDomainFromValues(maxValue int, values []int) *BitSetDomain {
	if maxValue <= 0 {
		return &BitSetDomain{maxValue: 0, words: nil}
	}

	// Try to get from pool
	d := getDomainFromPool(maxValue)
	if d == nil {
		// Too large for pool, allocate directly
		numWords := (maxValue + 63) / 64
		d = &BitSetDomain{
			maxValue: maxValue,
			words:    make([]uint64, numWords),
		}
	}

	// Set bits for specified values
	for _, v := range values {
		if v >= 1 && v <= maxValue {
			wordIdx := (v - 1) / 64
			bitOffset := uint((v - 1) % 64)
			d.words[wordIdx] |= 1 << bitOffset
		}
	}

	return d
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
// O(number of words) operation. Uses object pooling for common domain sizes.
func (d *BitSetDomain) Clone() Domain {
	// Try to get from pool
	newDomain := getDomainFromPool(d.maxValue)
	if newDomain == nil {
		// Too large for pool, allocate directly
		newWords := make([]uint64, len(d.words))
		copy(newWords, d.words)
		return &BitSetDomain{
			maxValue: d.maxValue,
			words:    newWords,
		}
	}

	// Copy words into pooled domain
	copy(newDomain.words, d.words)
	return newDomain
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

// RemoveAbove returns a new domain with all values > threshold removed.
// Uses efficient bit masking - O(words) not O(domain_size).
// Example: {1,2,3,4,5}.RemoveAbove(3) = {1,2,3}
func (d *BitSetDomain) RemoveAbove(threshold int) Domain {
	if threshold <= 0 {
		// Remove all values
		return &BitSetDomain{maxValue: d.maxValue, words: make([]uint64, len(d.words))}
	}
	if threshold >= d.maxValue {
		// Nothing to remove
		return d
	}

	newWords := make([]uint64, len(d.words))
	copy(newWords, d.words)

	// Clear all bits representing values > threshold
	// Bit i represents value i+1, so value > threshold means bit >= threshold
	bitIdx := threshold // bit index for value threshold+1
	wordIdx := bitIdx / 64
	bitOffset := uint(bitIdx % 64)

	// Clear remaining bits in the partial word
	if wordIdx < len(newWords) {
		mask := (uint64(1) << bitOffset) - 1 // Keep bits 0..bitOffset-1
		newWords[wordIdx] &= mask
	}

	// Clear all subsequent words
	for i := wordIdx + 1; i < len(newWords); i++ {
		newWords[i] = 0
	}

	return &BitSetDomain{maxValue: d.maxValue, words: newWords}
}

// RemoveBelow returns a new domain with all values < threshold removed.
// Uses efficient bit masking - O(words) not O(domain_size).
// Example: {1,2,3,4,5}.RemoveBelow(3) = {3,4,5}
func (d *BitSetDomain) RemoveBelow(threshold int) Domain {
	if threshold <= 1 {
		// Nothing to remove (minimum value is 1)
		return d
	}
	if threshold > d.maxValue {
		// Remove all values
		return &BitSetDomain{maxValue: d.maxValue, words: make([]uint64, len(d.words))}
	}

	newWords := make([]uint64, len(d.words))
	copy(newWords, d.words)

	// Clear all bits representing values < threshold
	// Bit i represents value i+1, so to keep values >= threshold, clear bits 0..(threshold-2)
	// For threshold=6, clear bits 0-4 (values 1-5), keep bits 5+ (values 6+)
	bitIdx := threshold - 2 // Last bit index to clear (value threshold-1)
	wordIdx := bitIdx / 64
	bitOffset := uint(bitIdx % 64)

	// Clear all words before the partial word
	for i := 0; i < wordIdx && i < len(newWords); i++ {
		newWords[i] = 0
	}

	// Clear lower bits in the partial word (bits 0..bitOffset inclusive)
	if wordIdx < len(newWords) {
		mask := ^((uint64(1) << (bitOffset + 1)) - 1) // Clear bits 0..bitOffset
		newWords[wordIdx] &= mask
	}

	return &BitSetDomain{maxValue: d.maxValue, words: newWords}
}

// RemoveAtOrAbove returns a new domain with all values >= threshold removed.
// Uses efficient bit masking - O(words) not O(domain_size).
// Example: {1,2,3,4,5}.RemoveAtOrAbove(3) = {1,2}
func (d *BitSetDomain) RemoveAtOrAbove(threshold int) Domain {
	if threshold <= 1 {
		// Remove all values
		return &BitSetDomain{maxValue: d.maxValue, words: make([]uint64, len(d.words))}
	}
	return d.RemoveAbove(threshold - 1)
}

// RemoveAtOrBelow returns a new domain with all values <= threshold removed.
// Uses efficient bit masking - O(words) not O(domain_size).
// Example: {1,2,3,4,5}.RemoveAtOrBelow(3) = {4,5}
func (d *BitSetDomain) RemoveAtOrBelow(threshold int) Domain {
	if threshold >= d.maxValue {
		// Remove all values
		return &BitSetDomain{maxValue: d.maxValue, words: make([]uint64, len(d.words))}
	}
	return d.RemoveBelow(threshold + 1)
}

// Min returns the minimum value in the domain.
// Returns 0 if domain is empty.
// O(words) in worst case, but typically O(1) as minimum is in first word.
func (d *BitSetDomain) Min() int {
	for wordIdx, word := range d.words {
		if word != 0 {
			// Find first set bit in this word
			bitOffset := 0
			for bitOffset < 64 && (word&(1<<uint(bitOffset))) == 0 {
				bitOffset++
			}
			// Bit i represents value i+1
			return wordIdx*64 + bitOffset + 1
		}
	}
	return 0 // Empty domain
}

// Max returns the maximum value in the domain.
// Returns 0 if domain is empty.
// O(words) in worst case, but typically O(1) as maximum is in last word.
func (d *BitSetDomain) Max() int {
	// Search from last word backwards
	for wordIdx := len(d.words) - 1; wordIdx >= 0; wordIdx-- {
		word := d.words[wordIdx]
		if word != 0 {
			// Find last set bit in this word
			bitOffset := 63
			for bitOffset >= 0 && (word&(1<<uint(bitOffset))) == 0 {
				bitOffset--
			}
			// Bit i represents value i+1
			value := wordIdx*64 + bitOffset + 1
			// Don't return values beyond maxValue
			if value > d.maxValue {
				continue
			}
			return value
		}
	}
	return 0 // Empty domain
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
