```go
func ExampleHybridSolver_realWorldScheduling() {
	// FD model: task start times with temporal constraints
	model := NewModel()
	task1 := model.NewVariableWithName(NewBitSetDomain(10), "task1_time")
	task2 := model.NewVariableWithName(NewBitSetDomain(10), "task2_time")
	task3 := model.NewVariableWithName(NewBitSetDomain(10), "task3_time")

	// FD constraint: task2 must start after task1 (task1 + 2 = task2)
	precedence, _ := NewArithmetic(task1, task2, 2)
	model.AddConstraint(precedence)

	// FD constraint: all tasks at different times
	allDiff, _ := NewAllDifferent([]*FDVariable{task1, task2, task3})
	model.AddConstraint(allDiff)

	// Create solver and store from model helper; then set initial domains
	solver, store, err := NewHybridSolverFromModel(model)
	if err != nil {
		panic(err)
	}

	timeSlots := NewBitSetDomainFromValues(10, []int{1, 2, 3, 4, 5})
	store, _ = store.SetDomain(task1.ID(), timeSlots)
	store, _ = store.SetDomain(task2.ID(), timeSlots)
	store, _ = store.SetDomain(task3.ID(), timeSlots)

	// Relational constraint: task1 must be a number (type safety)
	task1Var := Fresh("task1")
	typeConstraint := NewTypeConstraint(task1Var, NumberType)
	store = store.AddConstraint(typeConstraint)

	// External decision: task1 scheduled at time 1 (from relational reasoning)
	store, _ = store.AddBinding(int64(task1.ID()), NewAtom(1))

	// Hybrid propagation
	result, _ := solver.Propagate(store)

	// Results show hybrid cooperation:
	// - Relational binding (task1=1) → FD domain {1}
	// - FD arithmetic (1+2=3) → task2 domain {3}
	// - FD AllDifferent → task3 domain excludes {1,3}

	task1Time := result.GetDomain(task1.ID()).SingletonValue()
	task2Time := result.GetDomain(task2.ID()).SingletonValue()
	task3Dom := result.GetDomain(task3.ID())

	fmt.Printf("Task 1 starts at: %d\n", task1Time)
	fmt.Printf("Task 2 starts at: %d (precedence constraint)\n", task2Time)
	fmt.Printf("Task 3 possible times: %d slots\n", task3Dom.Count())
	fmt.Printf("Task 3 cannot use time 1: %v\n", !task3Dom.Has(1))
	fmt.Printf("Task 3 cannot use time 3: %v\n", !task3Dom.Has(3))

	// Output:
	// Task 1 starts at: 1
	// Task 2 starts at: 3 (precedence constraint)
	// Task 3 possible times: 3 slots
	// Task 3 cannot use time 1: true
	// Task 3 cannot use time 3: true
}

```


