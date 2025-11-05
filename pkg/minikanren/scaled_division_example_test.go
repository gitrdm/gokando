package minikanren_test

import (
	"context"
	"fmt"

	. "github.com/gitrdm/gokanlogic/pkg/minikanren"
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

// ExampleNewScaledDivision_piCircumference demonstrates the fixed-point arithmetic
// pattern for handling irrational constants like π. This is an alternative to using
// RationalLinearSum when you want explicit control over scaling.
func ExampleNewScaledDivision_piCircumference() {
	model := NewModel()

	// Circle diameter: 1-10 units
	diameter := model.NewVariableWithName(
		NewBitSetDomainFromValues(11, makeRangeEx(1, 10)),
		"diameter",
	)

	// Circumference (scaled by 10000): π * diameter * 10000
	// Using π ≈ 31416/10000 (more precision than 22/7)
	circumferenceScaled := model.NewVariableWithName(
		NewBitSetDomain(350000),
		"circumference_scaled",
	)

	// Actual circumference (in original units)
	circumference := model.NewVariableWithName(
		NewBitSetDomain(35),
		"circumference",
	)

	// Pattern: Fixed-point arithmetic for irrationals
	// 1. Scale the constant: π ≈ 31416/10000
	// 2. Use LinearSum with scaled constant: circumference_scaled = 31416 * diameter
	// 3. Use ScaledDivision to get final result: circumference = circumference_scaled / 10000

	// Step 1: circumference_scaled = 31416 * diameter
	coeffs := []int{31416}
	linearConstraint, _ := NewLinearSum(
		[]*FDVariable{diameter},
		coeffs,
		circumferenceScaled,
	)
	model.AddConstraint(linearConstraint)

	// Step 2: circumference = circumference_scaled / 10000
	divConstraint, _ := NewScaledDivision(circumferenceScaled, 10000, circumference)
	model.AddConstraint(divConstraint)

	// Fix diameter = 7 for demonstration
	diameter7Domain := NewBitSetDomainFromValues(11, []int{7})
	diameter = model.NewVariableWithName(diameter7Domain, "diameter")

	// Rebuild constraints with fixed diameter
	model = NewModel()
	diameter = model.NewVariableWithName(diameter7Domain, "diameter")
	circumferenceScaled = model.NewVariableWithName(NewBitSetDomain(350000), "circumference_scaled")
	circumference = model.NewVariableWithName(NewBitSetDomain(35), "circumference")

	linearConstraint, _ = NewLinearSum([]*FDVariable{diameter}, coeffs, circumferenceScaled)
	model.AddConstraint(linearConstraint)

	divConstraint, _ = NewScaledDivision(circumferenceScaled, 10000, circumference)
	model.AddConstraint(divConstraint)

	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	finalCircum := solver.GetDomain(nil, circumference.ID())
	finalScaled := solver.GetDomain(nil, circumferenceScaled.ID())

	fmt.Println("Fixed-point π calculation:")
	fmt.Printf("Diameter: 7 units\n")
	fmt.Printf("π * 7 * 10000 = %d (scaled)\n", finalScaled.Min())
	fmt.Printf("Circumference: %d units (actual)\n", finalCircum.Min())
	fmt.Printf("Precision: Using π ≈ 3.1416\n")

	// Output:
	// Fixed-point π calculation:
	// Diameter: 7 units
	// π * 7 * 10000 = 219912 (scaled)
	// Circumference: 21 units (actual)
	// Precision: Using π ≈ 3.1416
}

// ExampleNewScaledDivision_percentageWithScaling demonstrates combining multiple
// fixed-point operations for compound calculations.
func ExampleNewScaledDivision_percentageWithScaling() {
	model := NewModel()

	// Investment amount: $1000 (in dollars, not cents for this example)
	principal := model.NewVariableWithName(
		NewBitSetDomainFromValues(10001, []int{1000}), // $1000
		"principal",
	)

	// Annual interest rate: 5.25% → stored as 525 basis points (5.25 * 100)
	// Calculate: $1000 * 5.25 / 100 = $52.50
	// For integer result, scale by 100: 1000 * 525 / 100 = 5250 (in cents)
	interestScaled := model.NewVariableWithName(
		NewBitSetDomain(1000000),
		"interest_scaled",
	)

	interestCents := model.NewVariableWithName(
		NewBitSetDomain(10000),
		"interest_cents",
	)

	// Pattern: principal * 525 / 100 = interest_cents
	// Step 1: interest_scaled = principal * 525
	coeffs := []int{525}
	linearConstraint, _ := NewLinearSum(
		[]*FDVariable{principal},
		coeffs,
		interestScaled,
	)
	model.AddConstraint(linearConstraint)

	// Step 2: interest_cents = interest_scaled / 100
	divConstraint, _ := NewScaledDivision(interestScaled, 100, interestCents)
	model.AddConstraint(divConstraint)

	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	finalInterest := solver.GetDomain(nil, interestCents.ID())

	fmt.Println("Fixed-point percentage calculation:")
	fmt.Printf("Principal: $1,000.00\n")
	fmt.Printf("Rate: 5.25%% (525 basis points)\n")
	fmt.Printf("Interest: $%d.%02d\n",
		finalInterest.Min()/100, finalInterest.Min()%100)

	// Output:
	// Fixed-point percentage calculation:
	// Principal: $1,000.00
	// Rate: 5.25% (525 basis points)
	// Interest: $52.50
}
