#!/bin/bash

# Quick Race Detection Script for gokanlogic
# This script provides fast but effective race detection testing

set -e

echo "⚡ gokanlogic Quick Race Detection Testing"
echo "======================================"

echo
echo "1️⃣  Basic Race Detection"
echo "----------------------"
go test -race ./...

echo
echo "2️⃣  Stress Testing (Quick)"
echo "--------------------------"
go test -race -run="TestStressRaceConditions" ./pkg/minikanren

echo
echo "3️⃣  Memory Pressure Testing"
echo "----------------------------"
go test -race -run="TestMemoryPressureRaces" ./pkg/minikanren

echo
echo "4️⃣  Concurrent Testing"
echo "----------------------"
go test -race -run="TestConcurrentParallelExecution" -count=5 ./pkg/minikanren

echo
echo "✅ Quick Race Detection Complete!"
echo "================================="
echo
echo "For comprehensive testing, use ./scripts/race_test.sh"
echo "This quick test provides good coverage in minimal time."