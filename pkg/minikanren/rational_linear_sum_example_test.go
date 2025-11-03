package minikanren

import (
	"context"
	"fmt"
)

// ExampleNewRationalLinearSum demonstrates creating a linear sum constraint with rational coefficients.
// When all coefficients have the same denominator, the constraint scales efficiently.
func ExampleNewRationalLinearSum() {
	model := NewModel()

	// Variables: hours worked
	hours := model.NewVariable(NewBitSetDomainFromValues(50, []int{8})) // 8 hours worked
	payment := model.NewVariable(NewBitSetDomain(1000))                 // payment in dollars

	// Constraint: payment = 25 * hours (hourly rate of $25)
	coeffs := []Rational{NewRational(25, 1)} // $25/hour as coefficient
	rls, err := NewRationalLinearSum([]*FDVariable{hours}, coeffs, payment)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	model.AddConstraint(rls)

	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	paymentDomain := solver.GetDomain(nil, payment.ID()).(*BitSetDomain)
	fmt.Printf("Payment: $%d\n", paymentDomain.Min())

	// Output:
	// Payment: $200
}

// ExampleRationalLinearSum_piCircumference demonstrates using rational approximations
// of irrational numbers (π) in constraint programming.
func ExampleRationalLinearSum_piCircumference() {
	model := NewModel()

	// Circle with diameter = 7 units
	diameter := model.NewVariable(NewBitSetDomainFromValues(10, []int{7}))
	circumference := model.NewVariable(NewBitSetDomain(100))

	// Constraint: circumference = π * diameter
	// Using Archimedes' approximation: π ≈ 22/7
	pi := CommonIrrationals.PiArchimedes
	coeffs := []Rational{pi}

	rls, div, err := NewRationalLinearSumWithScaling(
		[]*FDVariable{diameter},
		coeffs,
		circumference,
		model,
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	model.AddConstraint(rls)
	if div != nil {
		model.AddConstraint(div)
	}

	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	circumDomain := solver.GetDomain(nil, circumference.ID()).(*BitSetDomain)
	fmt.Printf("Diameter: %d units\n", 7)
	fmt.Printf("Circumference: %d units (using π ≈ 22/7)\n", circumDomain.Min())

	// Output:
	// Diameter: 7 units
	// Circumference: 22 units (using π ≈ 22/7)
}

// ExampleRationalLinearSum_percentageCalculation demonstrates using rational coefficients
// for percentage calculations in constraint programming.
func ExampleRationalLinearSum_percentageCalculation() {
	model := NewModel()

	// Base salary: $50,000
	baseSalary := model.NewVariable(NewBitSetDomainFromValues(100000, []int{50000}))
	// Total with 10% bonus
	totalPay := model.NewVariable(NewBitSetDomain(100000))

	// Constraint: totalPay = 1.1 * baseSalary = (11/10) * baseSalary
	coeffs := []Rational{NewRational(11, 10)} // 110% = 11/10

	rls, div, err := NewRationalLinearSumWithScaling(
		[]*FDVariable{baseSalary},
		coeffs,
		totalPay,
		model,
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	model.AddConstraint(rls)
	if div != nil {
		model.AddConstraint(div)
	}

	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	totalDomain := solver.GetDomain(nil, totalPay.ID()).(*BitSetDomain)
	fmt.Printf("Base salary: $50,000\n")
	fmt.Printf("With 10%% bonus: $%d\n", totalDomain.Min())

	// Output:
	// Base salary: $50,000
	// With 10% bonus: $55000
}

// ExampleNewRationalLinearSumWithScaling demonstrates the scaling helper that handles
// the case where rational coefficients need to be scaled to integers.
func ExampleNewRationalLinearSumWithScaling() {
	model := NewModel()

	// Three investors with different ownership percentages
	investorA := model.NewVariable(NewBitSetDomainFromValues(10000, []int{3000})) // $3000 invested
	investorB := model.NewVariable(NewBitSetDomainFromValues(10000, []int{2000})) // $2000 invested
	// Total investment
	total := model.NewVariable(NewBitSetDomain(10000))

	// Constraint: total = (1/3)*A + (1/2)*B (fractional ownership)
	// Note: This is a simplified example; in reality you'd sum all investments
	coeffs := []Rational{
		NewRational(1, 3), // investor A owns 1/3 of their contribution
		NewRational(1, 2), // investor B owns 1/2 of their contribution
	}

	rls, div, err := NewRationalLinearSumWithScaling(
		[]*FDVariable{investorA, investorB},
		coeffs,
		total,
		model,
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	model.AddConstraint(rls)
	if div != nil {
		// The scaling helper created an intermediate variable and ScaledDivision constraint
		model.AddConstraint(div)
		fmt.Println("Scaling was needed (LCM > 1)")
	}

	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	totalDomain := solver.GetDomain(nil, total.ID()).(*BitSetDomain)
	fmt.Printf("Total: $%d\n", totalDomain.Min())

	// Output:
	// Scaling was needed (LCM > 1)
	// Total: $2000
}
