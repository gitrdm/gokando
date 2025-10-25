package minikanren

import (
	"context"
	"testing"
)

// BenchmarkConstraintBusStrategies compares different constraint bus allocation strategies
func BenchmarkConstraintBusStrategies(b *testing.B) {

	b.Run("Original_NewBusPerRun", func(b *testing.B) {
		// Simulate the old approach
		for i := 0; i < b.N; i++ {
			q := Fresh("q")
			goal := func(ctx context.Context, store ConstraintStore) *Stream {
				return Eq(q, NewAtom(i))(ctx, store)
			}

			// Old approach: new bus every time
			initialStore := NewLocalConstraintStore(NewGlobalConstraintBus())
			stream := goal(context.Background(), initialStore)
			stream.Take(1)
		}
	})

	b.Run("Optimized_SharedBus", func(b *testing.B) {
		// New approach with shared bus
		for i := 0; i < b.N; i++ {
			q := Fresh("q")
			goal := func(ctx context.Context, store ConstraintStore) *Stream {
				return Eq(q, NewAtom(i))(ctx, store)
			}

			// New approach: shared bus
			initialStore := NewLocalConstraintStore(GetDefaultGlobalBus())
			stream := goal(context.Background(), initialStore)
			stream.Take(1)
		}
	})

	b.Run("Optimized_PooledBus", func(b *testing.B) {
		// Pooled approach for isolation
		for i := 0; i < b.N; i++ {
			q := Fresh("q")
			goal := func(ctx context.Context, store ConstraintStore) *Stream {
				return Eq(q, NewAtom(i))(ctx, store)
			}

			// Pooled approach
			bus := GetPooledGlobalBus()
			initialStore := NewLocalConstraintStore(bus)
			stream := goal(context.Background(), initialStore)
			stream.Take(1)
			ReturnPooledGlobalBus(bus)
		}
	})

	b.Run("StandardRun_After_Optimization", func(b *testing.B) {
		// Test the optimized Run function
		for i := 0; i < b.N; i++ {
			Run(1, func(q *Var) Goal {
				return Eq(q, NewAtom(i))
			})
		}
	})

	b.Run("IsolatedRun", func(b *testing.B) {
		// Test the isolated Run function
		for i := 0; i < b.N; i++ {
			RunWithIsolation(1, func(q *Var) Goal {
				return Eq(q, NewAtom(i))
			})
		}
	})
}

// BenchmarkMemoryAllocation tests memory allocation patterns
func BenchmarkMemoryAllocation(b *testing.B) {
	b.ReportAllocs()

	b.Run("SharedBus", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			Run(1, func(q *Var) Goal {
				return Eq(q, NewAtom(i))
			})
		}
	})

	b.Run("PooledBus", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			RunWithIsolation(1, func(q *Var) Goal {
				return Eq(q, NewAtom(i))
			})
		}
	})
}
