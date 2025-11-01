# Critical Performance Issues - Phase 2

## 1. Inequality Constraint - MAJOR INEFFICIENCY ❌

**Current:** 4.86ms for 10 variables (38× slower than Arithmetic)

**Problem:** Naive loop-based domain pruning
```go
// Current code in propLT, propLE, propGT, propGE:
for v := maxY; v <= xDom.MaxValue(); v++ {
    if xDom.Has(v) {
        newXDom = newXDom.Remove(v)
    }
}
```

**Issue:** 
- Creates O(domain_size) individual `Remove()` calls
- Each `Remove()` creates a new BitSetDomain copy
- For domain size 100, this is 100 allocations per propagation

**Fix:** Add bulk operations to Domain interface
```go
// Add to Domain interface:
RemoveAbove(threshold int) Domain  // Remove all values > threshold
RemoveBelow(threshold int) Domain  // Remove all values < threshold
KeepRange(min, max int) Domain     // Keep only values in [min, max]
```

**Implementation:** Direct bitset manipulation
```go
// In BitSetDomain:
func (d *BitSetDomain) RemoveAbove(threshold int) Domain {
    // Set all bits above threshold to 0 in one operation
    // O(1) instead of O(domain_size)
}
```

**Expected improvement:** 10-20× faster (from 4.86ms to ~200-400µs)

---

## 2. AllDifferent - ALGORITHMIC DISASTER ❌

**Current:** 18.25 seconds for 50 variables

**Problem 1:** Redundant matching computations
```go
// In Propagate():
for i, v := range c.variables {
    domains[i].IterateValues(func(val int) {
        if !c.hasSupport(i, val, domains, n, maxVal) {
            toRemove = append(toRemove, val)
        }
    })
}

// hasSupport calls maxMatching EVERY TIME:
func (c *AllDifferent) hasSupport(...) bool {
    _, matchSize := c.maxMatching(tempDomains, maxVal)
    return matchSize == n
}
```

**Issue:**
- For 50 variables with 50 values each: potentially 2,500 matching computations
- Each matching is O(n²·d) = O(50²·50) = 125,000 operations
- Total: ~312 million operations per propagation

**Fix Option 1:** Implement Régin's AC algorithm properly
- Use **value graph** and **SCC decomposition** (already have code for this)
- Compute matching ONCE, not per-value
- Only re-match on domain changes

**Fix Option 2:** Switch to GAC-Schema for large n
- More efficient for n > 20
- Uses residual supports instead of repeated matching

**Expected improvement:** 100-1000× faster (from 18.25s to ~20-200ms)

---

## 3. Memory Allocation Overhead ⚠️

**Current:** 4,368 allocations for AllDiff-8vars

**Problem:** Excessive intermediate allocations
```go
// Each Remove() creates a new BitSetDomain:
for _, val := range toRemove {
    newDomain = newDomain.Remove(val)  // ALLOCATION
}

// In hasSupport, creates temp domain array:
tempDomains := make([]Domain, n)  // ALLOCATION
```

**Fix:** Object pooling
```go
var domainPool = sync.Pool{
    New: func() interface{} {
        return &BitSetDomain{...}
    },
}

// Reuse instead of allocate
func (d *BitSetDomain) RemoveMany(values []int) Domain {
    result := domainPool.Get().(*BitSetDomain)
    // ... modify result
    return result
}
```

**Expected improvement:** 50% reduction in allocations

---

## 4. Propagation Triggering - UNNECESSARY WORK ⚠️

**Current:** Propagates on every `SetDomain` call

**Problem:** No change detection
```go
// In solver.go:
func (s *Solver) SetDomain(state *SolverState, varID int, domain Domain) *SolverState {
    // Always creates new state, even if domain unchanged
    newState := &SolverState{...}
    // Caller then calls propagate() on newState
}
```

**Fix:** Skip propagation when no actual change
```go
func (s *Solver) SetDomain(state *SolverState, varID int, domain Domain) (*SolverState, bool) {
    oldDomain := s.GetDomain(state, varID)
    if oldDomain.Equal(domain) {
        return state, false  // No change, no propagation needed
    }
    newState := &SolverState{...}
    return newState, true  // Changed, propagation needed
}
```

**Expected improvement:** 20-30% reduction in propagation calls

---

## Priority Ranking

### P0 - Critical (Block Phase 3)
1. ✅ **Fix Inequality range operations** (1-2 hours work, 10-20× speedup)
2. ✅ **Fix AllDifferent redundant matching** (4-6 hours work, 100-1000× speedup)

### P1 - High (Should do before Phase 3)
3. **Add object pooling** (2-3 hours work, 50% allocation reduction)
4. **Add change detection** (1-2 hours work, 20-30% fewer propagations)

### P2 - Medium (Can defer to Phase 3)
5. Lazy propagation (delay until variable selection)
6. Constraint priority scheduling (cheap constraints first)

---

## Implementation Plan

### Step 1: Domain Range Operations (Fix Inequality)
```go
// Add to domain.go:
type Domain interface {
    // ... existing methods ...
    
    // Efficient range operations
    RemoveAbove(threshold int) Domain
    RemoveBelow(threshold int) Domain
    RemoveAtOrAbove(threshold int) Domain
    RemoveAtOrBelow(threshold int) Domain
}

// Implement in BitSetDomain using bit masking
```

### Step 2: Fix AllDifferent Algorithm
```go
// In propagation.go AllDifferent.Propagate():

// BEFORE:
for i, v := range c.variables {
    domains[i].IterateValues(func(val int) {
        if !c.hasSupport(i, val, domains, n, maxVal) {
            toRemove = append(toRemove, val)
        }
    })
}

// AFTER:
matching, _ := c.maxMatching(domains, maxVal)
valueGraph := c.buildValueGraph(domains, matching, maxVal)
sccs := c.computeSCCs(valueGraph)

// Only values in same SCC as matched values are supported
for i, v := range c.variables {
    matchedVal := matching[i]
    sccID := sccs[matchedVal]
    
    domains[i].IterateValues(func(val int) {
        if sccs[val] != sccID {
            toRemove = append(toRemove, val)
        }
    })
}
```

### Step 3: Add Object Pooling
```go
// Add pools for frequently allocated objects
var (
    bitSetPool = sync.Pool{New: func() interface{} { return &BitSetDomain{} }}
    intSlicePool = sync.Pool{New: func() interface{} { return make([]int, 0, 64) }}
)
```

### Step 4: Change Detection
```go
// Modify SetDomain signature and callers
func (s *Solver) SetDomain(state, varID, domain) (*SolverState, bool)

// In propagate():
newState, changed := solver.SetDomain(state, varID, newDomain)
if !changed {
    continue  // Skip re-propagation
}
```

---

## Expected Results After Fixes

| Benchmark | Before | After (Estimated) | Improvement |
|-----------|--------|-------------------|-------------|
| Inequality-10vars | 4.86ms | ~300µs | 16× faster |
| AllDiff-8vars | 3.09ms | ~2ms | 1.5× faster |
| AllDiff-12vars | 19.3ms | ~8ms | 2.4× faster |
| AllDiff-50vars | 18.25s | ~100ms | 180× faster |
| 8-Queens | 12.4ms | ~3ms | 4× faster |

**Overall Grade After Optimization:** A- (from current B+)
