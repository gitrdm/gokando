package minikanren_test

import (
	"fmt"

	. "github.com/gitrdm/gokanlogic/pkg/minikanren"
)

// ExampleModel_helpers_intVarValues shows creating a variable over a
// non-contiguous set of values using IntVarValues.
func ExampleModel_helpers_intVarValues() {
	m := NewModel()
	x := m.IntVarValues([]int{1, 3, 5}, "x")

	s := NewSolver(m)
	// Initial domain reflects the provided set exactly
	fmt.Println(s.GetDomain(nil, x.ID()))
	// Output:
	// {1,3,5}
}
