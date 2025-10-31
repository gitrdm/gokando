package minikanren

import (
	"fmt"
	"reflect"
)

// fd_relations.go: arithmetic relation constraints for true relational arithmetic

// ArithmeticRelationConstraint represents a constraint between three variables: x op y = z
// This implements true relational arithmetic without projection to FD solving.
type ArithmeticRelationConstraint struct {
	id      string
	x, y, z Term
	op      ArithmeticConstraintType
	isLocal bool
}

// NewArithmeticRelationConstraint creates a new arithmetic relation constraint x op y = z
func NewArithmeticRelationConstraint(x, y, z Term, op ArithmeticConstraintType) *ArithmeticRelationConstraint {
	return &ArithmeticRelationConstraint{
		id:      generateConstraintID(fmt.Sprintf("arith-%s", op.String())),
		x:       x,
		y:       y,
		z:       z,
		op:      op,
		isLocal: true, // Arithmetic constraints can be checked locally when all vars are bound
	}
}

// ID returns the unique identifier for this constraint
func (arc *ArithmeticRelationConstraint) ID() string {
	return arc.id
}

// IsLocal returns true if this constraint can be evaluated locally
func (arc *ArithmeticRelationConstraint) IsLocal() bool {
	return arc.isLocal
}

// Variables returns the logic variables this constraint depends on
func (arc *ArithmeticRelationConstraint) Variables() []*Var {
	var vars []*Var
	vars = append(vars, extractVariables(arc.x)...)
	vars = append(vars, extractVariables(arc.y)...)
	vars = append(vars, extractVariables(arc.z)...)
	return vars
}

// Check evaluates the arithmetic relation constraint against current bindings
func (arc *ArithmeticRelationConstraint) Check(bindings map[int64]Term) ConstraintResult {
	xVal := walkTerm(arc.x, bindings)
	yVal := walkTerm(arc.y, bindings)
	zVal := walkTerm(arc.z, bindings)

	// If any term contains unbound variables, constraint is pending
	if hasUnboundVariables(xVal) || hasUnboundVariables(yVal) || hasUnboundVariables(zVal) {
		return ConstraintPending
	}

	// All terms are bound - check if they satisfy the arithmetic relation
	if arc.satisfiesArithmetic(xVal, yVal, zVal) {
		return ConstraintSatisfied
	}

	return ConstraintViolated
}

// satisfiesArithmetic checks if x, y, z satisfy the arithmetic relation x op y = z
func (arc *ArithmeticRelationConstraint) satisfiesArithmetic(x, y, z Term) bool {
	// Extract numeric values
	xNum, xOk := extractNumber(x)
	yNum, yOk := extractNumber(y)
	zNum, zOk := extractNumber(z)

	if !xOk || !yOk || !zOk {
		return false // Non-numeric terms cannot satisfy arithmetic
	}

	// Check the arithmetic relation
	switch arc.op {
	case ArithmeticPlus:
		return xNum+yNum == zNum
	case ArithmeticMinus:
		return xNum-yNum == zNum
	case ArithmeticMultiply:
		return xNum*yNum == zNum
	case ArithmeticQuotient:
		if yNum == 0 {
			return false // Division by zero
		}
		return xNum/yNum == zNum
	case ArithmeticModulo:
		if yNum == 0 {
			return false // Modulo by zero
		}
		return xNum%yNum == zNum
	case ArithmeticEquality:
		return xNum == yNum && yNum == zNum
	default:
		return false
	}
}

// String returns a human-readable representation of the constraint
func (arc *ArithmeticRelationConstraint) String() string {
	return fmt.Sprintf("(%s %s %s = %s)", arc.x.String(), arc.op.String(), arc.y.String(), arc.z.String())
}

// Clone creates a deep copy of the constraint
func (arc *ArithmeticRelationConstraint) Clone() Constraint {
	return &ArithmeticRelationConstraint{
		id:      arc.id,
		x:       arc.x.Clone(),
		y:       arc.y.Clone(),
		z:       arc.z.Clone(),
		op:      arc.op,
		isLocal: arc.isLocal,
	}
}

// extractNumber extracts a numeric value from a term
func extractNumber(term Term) (int, bool) {
	if atom, ok := term.(*Atom); ok {
		val := atom.Value()
		rv := reflect.ValueOf(val)
		switch rv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return int(rv.Int()), true
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return int(rv.Uint()), true
		default:
			return 0, false
		}
	}
	return 0, false
}
