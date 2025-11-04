package minikanren_test

import (
	"fmt"

	. "github.com/gitrdm/gokando/pkg/minikanren"
)

// ExampleFormatTerm_basic shows single-term formatting consistent with FormatSolutions.
func ExampleFormatTerm_basic() {
	fmt.Println(FormatTerm(L(1, 2, 3)))
	fmt.Println(FormatTerm(A("hello")))
	// Output:
	// (1 2 3)
	// "hello"
}
