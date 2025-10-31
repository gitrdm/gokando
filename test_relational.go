package main

import (
	"fmt"

	"github.com/gitrdm/gokando/pkg/minikanren"
)

func main() {
	// Test 1: Valid arithmetic 2 + 3 = 5
	fmt.Println("Test 1: 2 + 3 = 5 (should succeed)")
	results1 := minikanren.Run(1, func(q *minikanren.Var) minikanren.Goal {
		x, y, z := minikanren.Fresh("x"), minikanren.Fresh("y"), minikanren.Fresh("z")

		return minikanren.Conj(
			minikanren.FDPlus(x, y, z),
			minikanren.Eq(x, minikanren.NewAtom(2)),
			minikanren.Eq(y, minikanren.NewAtom(3)),
			minikanren.Eq(z, minikanren.NewAtom(5)),
			minikanren.Eq(q, minikanren.NewAtom("success")),
		)
	})

	if len(results1) > 0 {
		fmt.Println("✅ Valid arithmetic test passed")
	} else {
		fmt.Println("❌ Valid arithmetic test failed - no results")
	}

	// Test 2: Invalid arithmetic 2 + 3 = 6 (should fail)
	fmt.Println("Test 2: 2 + 3 = 6 (should fail)")
	results2 := minikanren.Run(1, func(q *minikanren.Var) minikanren.Goal {
		x, y, z := minikanren.Fresh("x"), minikanren.Fresh("y"), minikanren.Fresh("z")

		return minikanren.Conj(
			minikanren.FDPlus(x, y, z),
			minikanren.Eq(x, minikanren.NewAtom(2)),
			minikanren.Eq(y, minikanren.NewAtom(3)),
			minikanren.Eq(z, minikanren.NewAtom(6)), // Wrong result
			minikanren.Eq(q, minikanren.NewAtom("success")),
		)
	})

	if len(results2) > 0 {
		fmt.Println("❌ Invalid arithmetic test failed - should have no results but got:", results2)
	} else {
		fmt.Println("✅ Invalid arithmetic test passed - correctly rejected")
	}

	// Test 3: Using Project as fallback (should work)
	fmt.Println("Test 3: Using Project fallback")
	results3 := minikanren.Run(1, func(q *minikanren.Var) minikanren.Goal {
		x, y, z := minikanren.Fresh("x"), minikanren.Fresh("y"), minikanren.Fresh("z")

		return minikanren.Conj(
			minikanren.FDPlus(x, y, z),
			minikanren.Eq(x, minikanren.NewAtom(2)),
			minikanren.Eq(y, minikanren.NewAtom(3)),
			minikanren.Project([]minikanren.Term{x, y, z}, func(vals []minikanren.Term) minikanren.Goal {
				xVal, yVal, zVal := vals[0], vals[1], vals[2]

				xAtom, xOk := xVal.(*minikanren.Atom)
				yAtom, yOk := yVal.(*minikanren.Atom)
				zAtom, zOk := zVal.(*minikanren.Atom)

				if !xOk || !yOk || !zOk {
					return minikanren.Failure
				}

				xInt, xOk2 := xAtom.Value().(int)
				yInt, yOk2 := yAtom.Value().(int)
				zInt, zOk2 := zAtom.Value().(int)

				if !xOk2 || !yOk2 || !zOk2 {
					return minikanren.Failure
				}

				if xInt+yInt == zInt {
					return minikanren.Eq(q, minikanren.NewAtom("success"))
				}

				return minikanren.Failure
			}),
		)
	})

	if len(results3) > 0 {
		fmt.Println("✅ Project fallback test passed")
	} else {
		fmt.Println("❌ Project fallback test failed - no results")
	}

	// Test 4: Test FDEqual
	fmt.Println("Test 4: FDEqual 5 = 5 = 5 (should succeed)")
	results4 := minikanren.Run(1, func(q *minikanren.Var) minikanren.Goal {
		x, y, z := minikanren.Fresh("x"), minikanren.Fresh("y"), minikanren.Fresh("z")

		return minikanren.Conj(
			minikanren.FDEqual(x, y, z),
			minikanren.Eq(x, minikanren.NewAtom(5)),
			minikanren.Eq(y, minikanren.NewAtom(5)),
			minikanren.Eq(z, minikanren.NewAtom(5)),
			minikanren.Eq(q, minikanren.NewAtom("success")),
		)
	})

	if len(results4) > 0 {
		fmt.Println("✅ FDEqual test passed")
	} else {
		fmt.Println("❌ FDEqual test failed - no results")
	}

	// Test 5: Test FDEqual with wrong values (should fail)
	fmt.Println("Test 5: FDEqual 5 = 5 = 6 (should fail)")
	results5 := minikanren.Run(1, func(q *minikanren.Var) minikanren.Goal {
		x, y, z := minikanren.Fresh("x"), minikanren.Fresh("y"), minikanren.Fresh("z")

		return minikanren.Conj(
			minikanren.FDEqual(x, y, z),
			minikanren.Eq(x, minikanren.NewAtom(5)),
			minikanren.Eq(y, minikanren.NewAtom(5)),
			minikanren.Eq(z, minikanren.NewAtom(6)), // Different value
			minikanren.Eq(q, minikanren.NewAtom("success")),
		)
	})

	if len(results5) > 0 {
		fmt.Println("❌ FDEqual invalid test failed - should have no results but got:", results5)
	} else {
		fmt.Println("✅ FDEqual invalid test passed - correctly rejected")
	}

	// Test 6: Test FDMinus
	fmt.Println("Test 6: FDMinus 8 - 3 = 5 (should succeed)")
	results6 := minikanren.Run(1, func(q *minikanren.Var) minikanren.Goal {
		x, y, z := minikanren.Fresh("x"), minikanren.Fresh("y"), minikanren.Fresh("z")

		return minikanren.Conj(
			minikanren.FDMinus(x, y, z),
			minikanren.Eq(x, minikanren.NewAtom(8)),
			minikanren.Eq(y, minikanren.NewAtom(3)),
			minikanren.Eq(z, minikanren.NewAtom(5)),
			minikanren.Eq(q, minikanren.NewAtom("success")),
		)
	})

	if len(results6) > 0 {
		fmt.Println("✅ FDMinus test passed")
	} else {
		fmt.Println("❌ FDMinus test failed - no results")
	}

	// Test 7: Test FDMultiply
	fmt.Println("Test 7: FDMultiply 4 * 3 = 12 (should succeed)")
	results7 := minikanren.Run(1, func(q *minikanren.Var) minikanren.Goal {
		x, y, z := minikanren.Fresh("x"), minikanren.Fresh("y"), minikanren.Fresh("z")

		return minikanren.Conj(
			minikanren.FDMultiply(x, y, z),
			minikanren.Eq(x, minikanren.NewAtom(4)),
			minikanren.Eq(y, minikanren.NewAtom(3)),
			minikanren.Eq(z, minikanren.NewAtom(12)),
			minikanren.Eq(q, minikanren.NewAtom("success")),
		)
	})

	if len(results7) > 0 {
		fmt.Println("✅ FDMultiply test passed")
	} else {
		fmt.Println("❌ FDMultiply test failed - no results")
	}

	// Test 8: Test FDQuotient
	fmt.Println("Test 8: FDQuotient 15 / 3 = 5 (should succeed)")
	results8 := minikanren.Run(1, func(q *minikanren.Var) minikanren.Goal {
		x, y, z := minikanren.Fresh("x"), minikanren.Fresh("y"), minikanren.Fresh("z")

		return minikanren.Conj(
			minikanren.FDQuotient(x, y, z),
			minikanren.Eq(x, minikanren.NewAtom(15)),
			minikanren.Eq(y, minikanren.NewAtom(3)),
			minikanren.Eq(z, minikanren.NewAtom(5)),
			minikanren.Eq(q, minikanren.NewAtom("success")),
		)
	})

	if len(results8) > 0 {
		fmt.Println("✅ FDQuotient test passed")
	} else {
		fmt.Println("❌ FDQuotient test failed - no results")
	}

	// Test 9: Test FDModulo
	fmt.Println("Test 9: FDModulo 17 % 5 = 2 (should succeed)")
	results9 := minikanren.Run(1, func(q *minikanren.Var) minikanren.Goal {
		x, y, z := minikanren.Fresh("x"), minikanren.Fresh("y"), minikanren.Fresh("z")

		return minikanren.Conj(
			minikanren.FDModulo(x, y, z),
			minikanren.Eq(x, minikanren.NewAtom(17)),
			minikanren.Eq(y, minikanren.NewAtom(5)),
			minikanren.Eq(z, minikanren.NewAtom(2)),
			minikanren.Eq(q, minikanren.NewAtom("success")),
		)
	})

	if len(results9) > 0 {
		fmt.Println("✅ FDModulo test passed")
	} else {
		fmt.Println("❌ FDModulo test failed - no results")
	}
}
