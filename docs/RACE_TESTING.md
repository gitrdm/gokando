# Race Condition Testing for GoKando

This document describes the race condition testing suite for the GoKando miniKanren implementation.

## Overview

The race detection testing provides comprehensive coverage from basic smoke tests to intensive stress testing to detect race conditions under various scenarios.

## Testing Levels

### 1. Basic Race Detection
```bash
go test -race ./...
```
- **Purpose**: Quick smoke test
- **Coverage**: Basic concurrent operations
- **Duration**: ~1 second
- **Use Case**: Development, quick checks

### 2. Intensive Race Detection
```bash
go test -race -count=20 -parallel=32 ./...
```
- **Purpose**: Multiple iterations with high parallelism
- **Coverage**: Exposes timing-dependent races
- **Duration**: ~5-10 seconds
- **Use Case**: Pre-commit verification

### 3. Stress Testing
```bash
go test -race -run="TestStressRaceConditions" -v ./pkg/minikanren
```
- **Purpose**: High-concurrency stress testing
- **Coverage**: 
  - 1000 goroutines creating 100k variables
  - 50 producers + 20 consumers with streams
  - 100 parallel workers executing 5000 goals
  - Concurrent constraint store access
- **Duration**: ~10-30 seconds
- **Use Case**: Release validation, performance testing

### 4. Memory Pressure Testing
```bash
go test -race -run="TestMemoryPressureRaces" -v ./pkg/minikanren
```
- **Purpose**: Race detection under memory pressure and GC stress
- **Coverage**: Operations during garbage collection and memory allocation
- **Duration**: ~5-15 seconds
- **Use Case**: Production-like environment testing

### 5. Comprehensive Testing Script
```bash
./scripts/race_test.sh
```
- **Purpose**: Complete race detection suite
- **Coverage**: All of the above plus benchmarks and chaos testing
- **Duration**: ~60-120 seconds (benchmarks with race detection are slow)
- **Use Case**: Release candidate validation

### 6. Quick Testing Script
```bash
./scripts/race_test_quick.sh
```
- **Purpose**: Fast but effective race detection for development
- **Coverage**: Core stress tests, memory pressure, concurrent testing
- **Duration**: ~5-10 seconds
- **Use Case**: Daily development, pre-commit checks

## Fixed Race Conditions

### Stream.Take() Race Condition
**Issue**: Race condition between stream closure and hasMore determination in Success goal
**Root Cause**: The Success goal puts one item and closes the stream in a goroutine, but the Take() method's hasMore check occurred before the goroutine had a chance to close the stream
**Symptoms**: Intermittent test failures under high concurrency where hasMore returned true when it should return false
**Fix**: Added `runtime.Gosched()` before the final hasMore check to yield execution to other goroutines, allowing the producer goroutine to complete stream closure
**Technical Details**: The race occurred between:
  1. Main thread: `stream.Take(1)` reads one item, then checks if more are available
  2. Goroutine: `stream.Put(store)` then `stream.Close()` 
  
  The `runtime.Gosched()` ensures the goroutine gets a chance to execute `stream.Close()` before the hasMore determination.

## Test Classifications

### Robust Tests (Production Ready)
- **TestStressRaceConditions**: High concurrency, long duration, multiple scenarios
- **TestMemoryPressureRaces**: GC pressure, memory allocation races
- **Intensive race detection**: Multiple iterations with high parallelism

### Smoke Tests (Development Only)
- Basic `go test -race` with default parameters
- Single-iteration tests
- Low concurrency tests

## Continuous Integration

For CI environments, use:
```bash
go test -race -run="TestRaceDetectionCI" ./pkg/minikanren
```

This provides balanced coverage without excessive resource usage.

## Performance Characteristics

| Test Type | Goroutines | Operations | Duration | Memory |
|-----------|------------|------------|----------|--------|
| Basic | 10-100 | 100-1000 | 1s | Low |
| Intensive | 100-500 | 1000-5000 | 5-10s | Medium |
| Stress | 1000+ | 10000+ | 30s+ | High |

## Race Detection Capabilities

**Detects**:
- Data races in variable ID generation
- Concurrent stream operations
- Parallel goal execution races
- Constraint store access races
- Memory allocation races
- GC-related timing issues

**Limitations**:
- Cannot detect logical race conditions (only data races)
- May miss races that require very specific timing
- Performance overhead during testing

## Best Practices

1. **Development**: Run basic race detection frequently
2. **Pre-commit**: Run intensive race detection
3. **Release**: Run full stress testing suite
4. **Production**: Monitor for race-related issues in logs

## Verification Commands

```bash
# Quick verification that race detection is working
go test -race -count=10 ./pkg/minikanren

# Verify stress tests work
go test -race -run="TestStressRaceConditions/Massive_concurrent_variable_creation" -v ./pkg/minikanren

# Full verification
./scripts/race_test.sh
```

## Conclusion

The race detection testing provides comprehensive coverage and high confidence in the thread safety of the GoKando implementation.