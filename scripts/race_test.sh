#!/bin/bash

# Comprehensive Race Detection Script for GoKando
# This script runs various levels of race detection testing

set -e

echo "🧪 GoKando Comprehensive Race Detection Testing"
echo "=============================================="

echo
echo "1️⃣  Basic Race Detection"
echo "----------------------"
go test -race ./...

echo
echo "2️⃣  Intensive Race Detection (Multiple Iterations)"
echo "------------------------------------------------"
go test -race -count=20 -parallel=32 ./...

echo
echo "3️⃣  Stress Testing (Long Duration)"
echo "----------------------------------"
go test -race -run="TestStressRaceConditions" -v ./pkg/minikanren

echo
echo "4️⃣  Memory Pressure Race Detection"
echo "-----------------------------------"
go test -race -run="TestMemoryPressureRaces" -v ./pkg/minikanren

echo
echo "5️⃣  Benchmark with Race Detection (Short)"
echo "------------------------------------------"
go test -race -bench=. -benchtime=2s ./pkg/minikanren

echo
echo "6️⃣  Extended Concurrent Testing"
echo "--------------------------------"
# Run the parallel tests multiple times with high concurrency
go test -race -run="TestConcurrentParallelExecution" -count=10 -parallel=8 ./pkg/minikanren

echo
echo "7️⃣  Chaos Testing (Random Timing)"
echo "----------------------------------"
# Run with CPU limiting to force more scheduling pressure
GOMAXPROCS=1 go test -race -count=20 ./pkg/minikanren

echo
echo "✅ Comprehensive Race Detection Complete!"
echo "========================================"
echo
echo "If all tests passed, the race detection is ROBUST and production-ready."
echo "If any test failed, there are race conditions that need to be fixed."