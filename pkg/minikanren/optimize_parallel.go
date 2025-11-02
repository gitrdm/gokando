package minikanren

import (
	"context"
	"sync"
	"sync/atomic"
)

// solveOptimalParallel runs branch-and-bound optimization using the shared work-queue
// parallel search infrastructure. It shares the incumbent objective across workers
// via atomics and applies dynamic objective cutoffs at each node to prune subtrees.
func (s *Solver) solveOptimalParallel(ctx context.Context, obj *FDVariable, minimize bool, cfg *optConfig) ([]int, int, error) {
	// Validate model
	if err := s.model.Validate(); err != nil {
		return nil, 0, err
	}

	// Initial propagation
	initialState := (*SolverState)(nil)
	propagatedState, err := s.propagate(initialState)
	if err != nil {
		return nil, 0, nil // root inconsistency: no solutions
	}

	// Early completion
	if s.isComplete(propagatedState) {
		sol := s.extractSolution(propagatedState)
		d := s.GetDomain(propagatedState, obj.ID())
		if d == nil || d.Count() == 0 {
			return nil, 0, nil
		}
		val := d.Min()
		if !minimize {
			val = d.Max()
		}
		return sol, val, nil
	}

	// Select the first variable to branch on
	varID, values := s.selectVariable(propagatedState)
	if varID == -1 {
		return nil, 0, nil
	}

	type optWork struct {
		state      *SolverState
		varID      int
		values     []int
		valueIndex int
		depth      int
	}

	workChan := make(chan *optWork, 1000)

	// Incumbent shared across workers
	var haveBest atomic.Bool
	var bestVal int64 // stores the integer objective
	var bestMu sync.Mutex
	var bestSol []int

	// Node limit across workers
	var nodes atomic.Int64

	// Enqueue initial work and set up task accounting
	var tasksWG sync.WaitGroup
	tasksWG.Add(1)
	workChan <- &optWork{state: propagatedState, varID: varID, values: values, valueIndex: 0, depth: 0}

	// Coordinator closes workChan when all tasks are done
	go func() {
		tasksWG.Wait()
		close(workChan)
	}()

	// Worker function
	worker := func(workerID int, cancel context.CancelFunc, hitLimit *atomic.Bool, targetReached *atomic.Bool) {
		for {
			select {
			case <-ctx.Done():
				// drain and release, decrement outstanding tasks for drained items
				for w := range workChan {
					s.ReleaseState(w.state)
					tasksWG.Done()
				}
				return
			case w, ok := <-workChan:
				if !ok {
					return
				}
				// Try each value for this variable
				for w.valueIndex < len(w.values) {
					select {
					case <-ctx.Done():
						// Cancelled while processing this work item: release and account once
						s.ReleaseState(w.state)
						tasksWG.Done()
						return
					default:
					}

					value := w.values[w.valueIndex]
					w.valueIndex++

					d := s.GetDomain(w.state, w.varID)
					nd := NewBitSetDomainFromValues(d.MaxValue(), []int{value})
					child, _ := s.SetDomain(w.state, w.varID, nd)

					// Apply incumbent cutoff to child objective domain
					if haveBest.Load() {
						cd := s.GetDomain(child, obj.ID())
						if cd != nil && cd.Count() > 0 {
							if minimize {
								cd = cd.RemoveAtOrAbove(int(atomic.LoadInt64(&bestVal)))
							} else {
								cd = cd.RemoveAtOrBelow(int(atomic.LoadInt64(&bestVal)))
							}
							child, _ = s.SetDomain(child, obj.ID(), cd)
						}
					}

					// Propagate
					ps, err := s.propagate(child)
					if err != nil {
						s.ReleaseState(child)
						continue
					}

					// Node limit is checked after leaf processing so we always
					// get at least one incumbent if a leaf is reachable.

					// Bound check for pruning with structural LB
					if haveBest.Load() {
						if b, ok := s.computeObjectiveBound(ps, obj, minimize); ok {
							cur := int(atomic.LoadInt64(&bestVal))
							if minimize && b >= cur {
								s.ReleaseState(ps)
								continue
							}
							if !minimize && b <= cur {
								s.ReleaseState(ps)
								continue
							}
						}
					}

					if s.isComplete(ps) {
						od := s.GetDomain(ps, obj.ID())
						if od != nil && od.IsSingleton() {
							val := od.SingletonValue()
							// CAS update bestVal
							updated := false
							if !haveBest.Load() {
								bestMu.Lock()
								if !haveBest.Load() {
									bestSol = s.extractSolution(ps)
									atomic.StoreInt64(&bestVal, int64(val))
									haveBest.Store(true)
									updated = true
								}
								bestMu.Unlock()
							} else {
								cur := int(atomic.LoadInt64(&bestVal))
								if (minimize && val < cur) || (!minimize && val > cur) {
									bestMu.Lock()
									// verify again under lock
									cur2 := int(atomic.LoadInt64(&bestVal))
									if (minimize && val < cur2) || (!minimize && val > cur2) {
										bestSol = s.extractSolution(ps)
										atomic.StoreInt64(&bestVal, int64(val))
										updated = true
									}
									bestMu.Unlock()
								}
							}
							// Early-accept if target reached
							if updated && cfg.targetObjective != nil && val == *cfg.targetObjective {
								// Early-accept
								targetReached.Store(true)
								s.ReleaseState(ps)
								// release the current work item and account once
								s.ReleaseState(w.state)
								tasksWG.Done()
								cancel()
								return
							}
						}
						s.ReleaseState(ps)
						// After processing a leaf, check node limit (count leaves)
						if cfg.nodeLimit > 0 && nodes.Add(1) >= int64(cfg.nodeLimit) {
							hitLimit.Store(true)
							cancel()
							// terminate this work item early on limit
							s.ReleaseState(w.state)
							tasksWG.Done()
							return
						}
						// leaf processed, try next value in this work item
						continue
					}

					// Select next var and enqueue work
					nid, nvals := s.selectVariable(ps)
					if nid == -1 {
						s.ReleaseState(ps)
						// nothing more to do for this branch; try next value
						continue
					}
					next := &optWork{state: ps, varID: nid, values: nvals, valueIndex: 0, depth: w.depth + 1}
					// Register new task before enqueue
					tasksWG.Add(1)
					select {
					case workChan <- next:
						// queued
					case <-ctx.Done():
						// roll back accounting and release
						tasksWG.Done()
						s.ReleaseState(ps)
						// cancel current work item as well
						s.ReleaseState(w.state)
						tasksWG.Done()
						return
					}
				}
				// Done with this work item
				s.ReleaseState(w.state)
				tasksWG.Done()
			}
		}
	}

	// Launch workers
	n := cfg.parallelWorkers
	if n <= 0 {
		n = 1
	}
	var wg sync.WaitGroup
	wg.Add(n)
	_, cancel := context.WithCancel(ctx)
	hitLimit := &atomic.Bool{}
	targetReached := &atomic.Bool{}
	for i := 0; i < n; i++ {
		go func(id int) { defer wg.Done(); worker(id, cancel, hitLimit, targetReached) }(i)
	}

	// Wait for all workers to finish (coordinator closes workChan via tasksWG)
	wg.Wait()

	if !haveBest.Load() {
		// No solution found
		if hitLimit.Load() {
			return nil, 0, ErrSearchLimitReached
		}
		return nil, 0, ctx.Err()
	}
	if hitLimit.Load() {
		return bestSol, int(atomic.LoadInt64(&bestVal)), ErrSearchLimitReached
	}
	return bestSol, int(atomic.LoadInt64(&bestVal)), nil
}
