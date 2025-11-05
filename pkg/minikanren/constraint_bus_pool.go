package minikanren

import (
	"sync"
)

// GlobalConstraintBusPool manages a pool of reusable constraint buses
type GlobalConstraintBusPool struct {
	pool sync.Pool
}

// NewGlobalConstraintBusPool creates a new pool of constraint buses
func NewGlobalConstraintBusPool() *GlobalConstraintBusPool {
	return &GlobalConstraintBusPool{
		pool: sync.Pool{
			New: func() interface{} {
				return NewGlobalConstraintBus()
			},
		},
	}
}

// Get retrieves a constraint bus from the pool
func (p *GlobalConstraintBusPool) Get() *GlobalConstraintBus {
	return p.pool.Get().(*GlobalConstraintBus)
}

// Put returns a constraint bus to the pool after cleaning it
func (p *GlobalConstraintBusPool) Put(bus *GlobalConstraintBus) {
	// Don't return shutdown buses to the pool
	if bus.shutdown {
		return // Let it be garbage collected
	}

	// Clean the bus before returning to pool
	bus.Reset()
	p.pool.Put(bus)
}

// Global pool instance for reuse
var defaultBusPool = NewGlobalConstraintBusPool()

// Singleton pattern for simple cases
var (
	defaultGlobalBus     *GlobalConstraintBus
	defaultGlobalBusOnce sync.Once
)

// GetDefaultGlobalBus returns a shared global constraint bus instance
// Use this for operations that don't require constraint isolation between goals
func GetDefaultGlobalBus() *GlobalConstraintBus {
	defaultGlobalBusOnce.Do(func() {
		defaultGlobalBus = NewGlobalConstraintBus()
	})
	return defaultGlobalBus
}

// GetPooledGlobalBus gets a constraint bus from the pool for operations
// that need isolation but can reuse cleaned instances
func GetPooledGlobalBus() *GlobalConstraintBus {
	return defaultBusPool.Get()
}

// ReturnPooledGlobalBus returns a bus to the pool
func ReturnPooledGlobalBus(bus *GlobalConstraintBus) {
	defaultBusPool.Put(bus)
}
