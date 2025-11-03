package minikanren_test

import (
	"context"
	"fmt"

	. "github.com/gitrdm/gokando/pkg/minikanren"
)

// ExampleNewScaledDivision demonstrates using ScaledDivision for scaled
// integer arithmetic, following the PicoLisp pattern.
//
// This example calculates employee bonuses as 10% of salary using scaled
// integer division (all values in cents, scaled by 100).
func ExampleNewScaledDivision() {
	model := NewModel()

	// All monetary values scaled by 100 (cents)
	// Salary range: $500-$700 → 50000-70000 cents
	salaryValues := make([]int, 0)
	for s := 50000; s <= 70000; s += 10000 {
		salaryValues = append(salaryValues, s)
	}
	salaryVar := model.NewVariableWithName(
		NewBitSetDomainFromValues(70001, salaryValues),
		"salary",
	)

	// Bonus initially unconstrained
	bonusVar := model.NewVariableWithName(
		NewBitSetDomainFromValues(10000, makeRangeEx(1, 10000)),
		"bonus",
	)

	// Constraint: bonus = salary / 10 (10% bonus)
	// Since values are in cents, this gives us exact integer division
	constraint, err := NewScaledDivision(salaryVar, 10, bonusVar)
	if err != nil {
		panic(err)
	}
	model.AddConstraint(constraint)

	// Solve to propagate constraints
	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	// Check propagated bonus domain
	finalBonus := solver.GetDomain(nil, bonusVar.ID())

	fmt.Println("Salary-to-Bonus constraint (10% bonus):")
	fmt.Printf("Salary range: $500.00 - $700.00\n")
	fmt.Printf("Bonus range: $%d.%02d - $%d.%02d\n",
		finalBonus.Min()/100, finalBonus.Min()%100,
		finalBonus.Max()/100, finalBonus.Max()%100)
	fmt.Printf("Possible bonuses: ")

	bonuses := finalBonus.ToSlice()
	for i, b := range bonuses {
		if i > 0 {
			fmt.Printf(", ")
		}
		fmt.Printf("$%d.%02d", b/100, b%100)
	}
	fmt.Println()

	// Output:
	// Salary-to-Bonus constraint (10% bonus):
	// Salary range: $500.00 - $700.00
	// Bonus range: $50.00 - $70.00
	// Possible bonuses: $50.00, $60.00, $70.00
}

// ExampleNewScaledDivision_bidirectional demonstrates bidirectional propagation
// where both dividend and quotient constrain each other.
func ExampleNewScaledDivision_bidirectional() {
	model := NewModel()

	// Price scaled by 100 (cents)
	priceVar := model.NewVariableWithName(
		NewBitSetDomainFromValues(2000, makeRangeEx(500, 1500)),
		"price_cents",
	)

	// Discount rate (percentage): 10-20%
	discountVar := model.NewVariableWithName(
		NewBitSetDomainFromValues(21, makeRangeEx(10, 20)),
		"discount_pct",
	)

	// Constraint: discount_pct = price_cents / 100
	// This means price must be divisible by 100 for exact percentage
	constraint, _ := NewScaledDivision(priceVar, 100, discountVar)
	model.AddConstraint(constraint)

	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	finalPrice := solver.GetDomain(nil, priceVar.ID())
	finalDiscount := solver.GetDomain(nil, discountVar.ID())

	fmt.Println("Bidirectional propagation:")
	fmt.Printf("Price: $%d.%02d - $%d.%02d\n",
		finalPrice.Min()/100, finalPrice.Min()%100,
		finalPrice.Max()/100, finalPrice.Max()%100)
	fmt.Printf("Discount: %d%% - %d%%\n",
		finalDiscount.Min(), finalDiscount.Max())

	// Output:
	// Bidirectional propagation:
	// Price: $10.00 - $15.00
	// Discount: 10% - 15%
}

// Helper to create range [start..end]
func makeRangeEx(start, end int) []int {
	result := make([]int, end-start+1)
	for i := range result {
		result[i] = start + i
	}
	return result
}
