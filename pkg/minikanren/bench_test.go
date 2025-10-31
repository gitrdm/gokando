package minikanren

import (
	"context"
	"fmt"
	"os"
	"runtime/pprof"
	"testing"
	"time"
)

// BenchmarkComprehensiveSuite provides comprehensive performance benchmarks
// for all major GoKanDo components. This suite enables systematic performance
// monitoring, bottleneck identification, and regression detection.
//
// Run benchmarks with:
//   go test -bench=. -benchmem ./pkg/minikanren
//   go test -bench=. -benchmem -cpuprofile=cpu.prof -memprofile=mem.prof ./pkg/minikanren
//
// Analyze profiles with:
//   go tool pprof cpu.prof
//   go tool pprof mem.prof

// =============================================================================
// CORE MINI KANREN BENCHMARKS
// =============================================================================

// BenchmarkFreshCreation benchmarks variable creation performance
func BenchmarkFreshCreation(b *testing.B) {
	b.Run("SingleVariable", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = Fresh("x")
		}
	})

	b.Run("MultipleVariables", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x := Fresh("x")
			y := Fresh("y")
			z := Fresh("z")
			_, _, _ = x, y, z
		}
	})

	b.Run("NamedVariables", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = Fresh(fmt.Sprintf("var_%d", i))
		}
	})
}

// BenchmarkUnificationCore benchmarks term unification performance
func BenchmarkUnificationCore(b *testing.B) {
	// Enable CPU profiling for unification benchmarks
	defer profileCPU(b, "unification")()

	b.Run("AtomUnification", func(b *testing.B) {
		b.ReportAllocs()
		store := NewLocalConstraintStore(nil)
		x := Fresh("x")

		for i := 0; i < b.N; i++ {
			// Reset store for each iteration
			store.Reset()
			_ = Eq(x, NewAtom(i))(context.Background(), store)
		}
	})

	b.Run("VariableUnification", func(b *testing.B) {
		b.ReportAllocs()
		store := NewLocalConstraintStore(nil)
		x := Fresh("x")
		y := Fresh("y")

		for i := 0; i < b.N; i++ {
			store.Reset()
			_ = Eq(x, y)(context.Background(), store)
		}
	})

	b.Run("ListUnification", func(b *testing.B) {
		b.ReportAllocs()
		store := NewLocalConstraintStore(nil)
		x := Fresh("x")

		for i := 0; i < b.N; i++ {
			store.Reset()
			list := List(NewAtom(i), NewAtom(i+1), NewAtom(i+2))
			_ = Eq(x, list)(context.Background(), store)
		}
	})

	b.Run("ComplexTermUnification", func(b *testing.B) {
		b.ReportAllocs()
		store := NewLocalConstraintStore(nil)
		x := Fresh("x")

		for i := 0; i < b.N; i++ {
			store.Reset()
			complexTerm := List(
				NewAtom("a"),
				List(NewAtom("b"), NewAtom(i)),
				NewAtom("c"),
			)
			_ = Eq(x, complexTerm)(context.Background(), store)
		}
	})
}

// BenchmarkGoalExecution benchmarks basic goal execution patterns
func BenchmarkGoalExecution(b *testing.B) {
	b.Run("SimpleEquality", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			Run(1, func(q *Var) Goal {
				return Eq(q, NewAtom(i))
			})
		}
	})

	b.Run("Conjunction", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			Run(1, func(q *Var) Goal {
				x := Fresh("x")
				y := Fresh("y")
				return Conj(
					Eq(x, NewAtom(i)),
					Eq(y, NewAtom(i*2)),
					Eq(q, List(x, y)),
				)
			})
		}
	})

	b.Run("Disjunction", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			Run(1, func(q *Var) Goal {
				return Disj(
					Eq(q, NewAtom(i)),
					Eq(q, NewAtom(i+1)),
					Eq(q, NewAtom(i+2)),
				)
			})
		}
	})

	b.Run("NestedGoals", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			Run(1, func(q *Var) Goal {
				x := Fresh("x")
				y := Fresh("y")
				return Conj(
					Disj(
						Eq(x, NewAtom(i)),
						Eq(x, NewAtom(i+1)),
					),
					Eq(y, x),
					Eq(q, List(x, y)),
				)
			})
		}
	})
}

// =============================================================================
// CONSTRAINT SYSTEM BENCHMARKS
// =============================================================================

// BenchmarkConstraintOperations benchmarks constraint system performance
func BenchmarkConstraintOperations(b *testing.B) {
	// Enable memory profiling for constraint benchmarks
	defer profileMemory(b, "constraints")

	b.Run("TypeConstraints", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			Run(1, func(q *Var) Goal {
				return Conj(
					Symbolo(q),
					Eq(q, NewAtom("test")),
				)
			})
		}
	})

	b.Run("DisequalityConstraints", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			Run(1, func(q *Var) Goal {
				x := Fresh("x")
				return Conj(
					Neq(x, NewAtom(i)),
					Eq(q, x),
				)
			})
		}
	})

	b.Run("AbsenceConstraints", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			Run(1, func(q *Var) Goal {
				return Conj(
					Absento(NewAtom("absent"), q),
					Eq(q, List(NewAtom(i), NewAtom(i+1))),
				)
			})
		}
	})

	b.Run("ListOperations", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			Run(1, func(q *Var) Goal {
				x := Fresh("x")
				y := Fresh("y")
				return Conj(
					Cons(x, y, q),
					Eq(x, NewAtom(i)),
					Eq(y, List(NewAtom(i+1))),
				)
			})
		}
	})
}

// =============================================================================
// FINITE DOMAIN BENCHMARKS
// =============================================================================

// BenchmarkFDBasic benchmarks basic finite domain operations
func BenchmarkFDBasic(b *testing.B) {
	b.Run("DomainCreation", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = NewBitSet(100)
		}
	})

	b.Run("FDStoreCreation", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = NewFDStoreWithDomain(10)
		}
	})

	b.Run("FDVariableCreation", func(b *testing.B) {
		b.ReportAllocs()
		store := NewFDStoreWithDomain(10)
		for i := 0; i < b.N; i++ {
			_ = store.NewVar()
		}
	})
}

// BenchmarkFDComplex benchmarks complex finite domain problems
func BenchmarkFDComplex(b *testing.B) {
	b.Run("NQueens_4x4", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			store := NewFDStoreWithDomain(8)
			cols := store.MakeFDVars(4)
			d1 := store.MakeFDVars(4)
			d2 := store.MakeFDVars(4)
			for i := 0; i < 4; i++ {
				store.AddOffsetLink(cols[i], i, d1[i])
				store.AddOffsetLink(cols[i], -i+4, d2[i])
			}
			store.ApplyAllDifferentRegin(cols)
			store.ApplyAllDifferentRegin(d1)
			store.ApplyAllDifferentRegin(d2)
			// constrain columns to 1..4
			for _, v := range cols {
				for j := 5; j <= 8; j++ {
					store.Remove(v, j)
				}
			}
			_, _ = store.Solve(context.Background(), 1)
		}
	})
}

// =============================================================================
// STREAMING AND PARALLEL BENCHMARKS
// =============================================================================

// BenchmarkStreaming benchmarks streaming performance
func BenchmarkStreaming(b *testing.B) {
	b.Run("ChannelStream_Put", func(b *testing.B) {
		b.ReportAllocs()
		stream := NewChannelResultStream(10)

		for i := 0; i < b.N; i++ {
			store := NewLocalConstraintStore(nil)
			stream.Put(context.Background(), store)
		}
		stream.Close()
	})

	b.Run("ChannelStream_Take", func(b *testing.B) {
		stream := NewChannelResultStream(10)
		ctx := context.Background()

		// Pre-populate stream
		for i := 0; i < 100; i++ {
			store := NewLocalConstraintStore(nil)
			stream.Put(ctx, store)
		}
		stream.Close()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			results, _, _ := stream.Take(ctx, 1)
			if len(results) == 0 {
				break
			}
		}
	})

	b.Run("PooledStream", func(b *testing.B) {
		pool := NewConstraintStorePool(100)
		stream := NewPooledResultStream(pool, 10, false)
		ctx := context.Background()

		for i := 0; i < b.N; i++ {
			store := pool.GetLocal()
			stream.Put(ctx, store)
		}
		stream.Close()
	})
}

// BenchmarkParallelExecution benchmarks parallel execution performance
func BenchmarkParallelExecution(b *testing.B) {
	b.Run("ParallelRun_Small", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ParallelRun(10, func(q *Var) Goal {
				return Eq(q, NewAtom(i%5))
			})
		}
	})

	b.Run("ParallelRun_Medium", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ParallelRun(50, func(q *Var) Goal {
				x := Fresh("x")
				y := Fresh("y")
				return Conj(
					Eq(x, NewAtom(i%10)),
					Eq(y, NewAtom((i+1)%10)),
					Eq(q, List(x, y)),
				)
			})
		}
	})
}

// =============================================================================
// MEMORY MANAGEMENT BENCHMARKS
// =============================================================================

// BenchmarkMemoryManagement benchmarks memory pool and allocation performance
func BenchmarkMemoryManagement(b *testing.B) {
	b.Run("ConstraintStorePool_GetPut", func(b *testing.B) {
		b.ReportAllocs()
		pool := NewConstraintStorePool(0) // Unlimited pool

		for i := 0; i < b.N; i++ {
			store := pool.GetLocal()
			pool.PutLocal(store)
		}
	})

	b.Run("GlobalBusPool_GetPut", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			bus := GetPooledGlobalBus()
			ReturnPooledGlobalBus(bus)
		}
	})

	b.Run("StoreCreation_New", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = NewLocalConstraintStore(nil)
		}
	})

	b.Run("StoreCreation_WithBus", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = NewLocalConstraintStore(NewGlobalConstraintBus())
		}
	})
}

// =============================================================================
// PERFORMANCE PROFILING UTILITIES
// =============================================================================

// profileCPU runs CPU profiling during benchmark execution
func profileCPU(b *testing.B, profileName string) func() {
	f, err := os.Create(profileName + ".cpu.prof")
	if err != nil {
		b.Fatal(err)
	}
	if err := pprof.StartCPUProfile(f); err != nil {
		b.Fatal(err)
	}
	return func() {
		pprof.StopCPUProfile()
		f.Close()
	}
}

// profileMemory captures memory profile after benchmark execution
func profileMemory(b *testing.B, profileName string) {
	f, err := os.Create(profileName + ".mem.prof")
	if err != nil {
		b.Fatal(err)
	}
	defer f.Close()
	if err := pprof.WriteHeapProfile(f); err != nil {
		b.Fatal(err)
	}
}

// BenchmarkPerformanceRegression detects performance regressions
func BenchmarkPerformanceRegression(b *testing.B) {
	// Define performance thresholds (adjust based on your system)
	const (
		maxAllocsPerOp = 100    // Maximum allocations per operation
		maxNsPerOp     = 150000 // Maximum nanoseconds per operation (150μs)
		minOpsPerSec   = 25     // Minimum operations per second
	)

	b.Run("RegressionCheck_Unification", func(b *testing.B) {
		b.ReportAllocs()

		start := time.Now()

		for i := 0; i < b.N; i++ {
			Run(1, func(q *Var) Goal {
				x := Fresh("x")
				y := Fresh("y")
				return Conj(
					Eq(x, NewAtom(i)),
					Eq(y, x),
					Eq(q, List(x, y)),
				)
			})
		}

		duration := time.Since(start)

		// Check performance thresholds
		nsPerOp := float64(duration.Nanoseconds()) / float64(b.N)
		if nsPerOp > maxNsPerOp {
			b.Errorf("Too slow: %.2f ns/op (threshold: %d ns/op)", nsPerOp, maxNsPerOp)
		}

		opsPerSec := float64(b.N) / duration.Seconds()
		if opsPerSec < minOpsPerSec {
			b.Errorf("Too slow: %.2f ops/sec (threshold: %d ops/sec)", opsPerSec, minOpsPerSec)
		}

		b.Logf("Performance: %.2f ns/op, %.2f ops/sec", nsPerOp, opsPerSec)
	})
}

// BenchmarkUnificationProfile runs unification with profiling enabled
func BenchmarkUnificationProfile(b *testing.B) {
	// Enable CPU profiling for the entire benchmark
	defer profileCPU(b, "unification_profile")()

	b.ReportAllocs()
	store := NewLocalConstraintStore(nil)

	for i := 0; i < b.N; i++ {
		// Reset store for each iteration
		store.Reset()
		x := Fresh("x")
		y := Fresh("y")
		// Mix of different unification operations
		_ = Eq(x, NewAtom(i))(context.Background(), store)
		_ = Eq(y, x)(context.Background(), store)
		list := List(NewAtom(i), NewAtom(i+1), NewAtom(i+2))
		_ = Eq(x, list)(context.Background(), store)
	}
}
