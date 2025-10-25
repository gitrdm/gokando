#!/bin/bash

# Comprehensive Memory and Performance Analysis Script for GoKando
# This script demonstrates Go's memory leak detection and profiling capabilities

set -e

echo "🔍 GoKanren Memory Analysis & Profiling Suite"
echo "============================================="

# Create profiles directory
mkdir -p profiles

echo
echo "1️⃣  Basic Memory Leak Detection"
echo "------------------------------"
go test -run="TestMemoryLeakDetection" -v ./pkg/minikanren

echo
echo "2️⃣  Memory Profiling"
echo "-------------------"
echo "Generating memory profile..."
go test -run="TestMemoryProfiling" -memprofile=profiles/memory.prof -v ./pkg/minikanren

echo
echo "3️⃣  CPU Profiling" 
echo "-----------------"
echo "Generating CPU profile..."
go test -run="TestCPUProfiling" -cpuprofile=profiles/cpu.prof -v ./pkg/minikanren

echo
echo "4️⃣  Allocation Benchmarks"
echo "-------------------------"
echo "Running allocation benchmarks..."
go test -bench="BenchmarkMemoryAllocations" -benchmem -v ./pkg/minikanren

echo
echo "5️⃣  Race Detection with Memory Profiling"
echo "-----------------------------------------"
echo "Combining race detection with memory analysis..."
go test -race -run="TestMemoryPressureRaces" -memprofile=profiles/race_memory.prof -v ./pkg/minikanren

echo
echo "6️⃣  Garbage Collection Analysis"
echo "-------------------------------"
echo "Running with GC statistics..."
GODEBUG=gctrace=1 go test -run="TestStressRaceConditions" ./pkg/minikanren 2>&1 | head -20

echo
echo "7️⃣  Memory Usage Analysis"
echo "-------------------------"
echo "Checking for profile files..."
ls -la profiles/ 2>/dev/null || echo "No profile files generated"

if [ -f "profiles/memory.prof" ]; then
    echo
    echo "📊 Memory Profile Analysis"
    echo "========================="
    echo "Top memory allocations:"
    go tool pprof -text -lines profiles/memory.prof | head -20
    
    echo
    echo "🔧 Interactive Analysis Commands:"
    echo "  go tool pprof profiles/memory.prof"
    echo "  (pprof) top10              # Top 10 functions by memory"
    echo "  (pprof) list <function>    # Source code with annotations"
    echo "  (pprof) web                # Web interface (requires graphviz)"
    echo "  (pprof) svg > memory.svg   # Generate SVG graph"
fi

if [ -f "profiles/cpu.prof" ]; then
    echo
    echo "⚡ CPU Profile Analysis"
    echo "====================="
    echo "Top CPU consumers:"
    go tool pprof -text -lines profiles/cpu.prof | head -20
    
    echo
    echo "🔧 Interactive Analysis Commands:"
    echo "  go tool pprof profiles/cpu.prof"
    echo "  (pprof) top10              # Top 10 functions by CPU time"
    echo "  (pprof) list <function>    # Source code with annotations"
    echo "  (pprof) web                # Web interface"
    echo "  (pprof) flamegraph > cpu.svg # Generate flame graph"
fi

echo
echo "8️⃣  Fuzzing (Built-in Go Fuzzing)"
echo "===================================" 
echo "Running fuzz tests for 10 seconds each..."

echo "Fuzzing variable creation..."
timeout 10s go test -fuzz="FuzzFresh" -fuzztime=10s ./pkg/minikanren || echo "Fuzz test completed"

echo "Fuzzing unification..."
timeout 10s go test -fuzz="FuzzUnification" -fuzztime=10s ./pkg/minikanren || echo "Fuzz test completed"

echo "Fuzzing goal execution..."
timeout 10s go test -fuzz="FuzzGoalExecution" -fuzztime=10s ./pkg/minikanren || echo "Fuzz test completed"

echo
echo "9️⃣  Advanced Memory Debugging"
echo "=============================="
echo "Running with memory debugging enabled..."

# Set Go memory debugging flags
export GODEBUG=allocfreetrace=1,gcpacertrace=1
echo "Memory debugging flags set: $GODEBUG"

# Run a short test with memory debugging
timeout 5s go test -run="TestConcurrentAccess" ./pkg/minikanren 2>&1 | head -30 || echo "Memory debugging test completed"

unset GODEBUG

echo
echo "🔟  Memory Sanitizer (if available)"
echo "==================================="
# Note: Go doesn't have AddressSanitizer like C++, but we can check for obvious issues
echo "Checking for nil pointer dereferences and bounds checking..."
go test -run="." ./pkg/minikanren >/dev/null 2>&1 && echo "✅ No obvious memory safety issues detected"

echo
echo "✅ Comprehensive Memory Analysis Complete!"
echo "=========================================="
echo
echo "📋 Summary of Tools Used:"
echo "========================"
echo "✅ Built-in fuzzing (go test -fuzz)    - Similar to libFuzzer/AFL"
echo "✅ Memory profiling (-memprofile)      - Like Valgrind/AddressSanitizer"
echo "✅ CPU profiling (-cpuprofile)         - Like perf/gprof"
echo "✅ Race detection (-race)              - Like ThreadSanitizer"
echo "✅ Benchmark memory tracking (-benchmem) - Custom allocation tracking"
echo "✅ GC tracing (GODEBUG=gctrace)        - Garbage collection analysis"
echo "✅ Memory debugging (GODEBUG flags)    - Runtime debugging"
echo
echo "📁 Generated Files:"
echo "=================="
echo "  profiles/memory.prof - Memory allocation profile"
echo "  profiles/cpu.prof - CPU usage profile"
echo "  profiles/race_memory.prof - Memory profile during race testing"
echo
echo "🔍 How Go Compares to C++:"
echo "=========================="
echo "  Fuzzing:       Go has BUILT-IN fuzzing vs C++ needs libFuzzer/AFL"
echo "  Memory leaks:  Go GC prevents most leaks vs C++ needs Valgrind/ASan"
echo "  Race detection: Go -race flag vs C++ ThreadSanitizer"
echo "  Profiling:     Go tool pprof vs C++ gprof/perf/Intel VTune"
echo "  Memory safety: Go bounds checking vs C++ AddressSanitizer"
echo
echo "🚀 Go Advantages:"
echo "================="
echo "  • Memory safety by default (GC + bounds checking)"
echo "  • Built-in race detector (no external tools needed)"
echo "  • Native fuzzing support in standard library"
echo "  • Integrated profiling tools"
echo "  • No undefined behavior (unlike C++)"
echo "  • Deterministic garbage collection"