package minikanren

import "fmt"

// ExampleTypeChecko_lambda demonstrates checking a lambda against an arrow type with variables.
func ExampleTypeChecko_lambda() {
	a := NewAtom("a")
	T := Fresh("T")
	term := Lambda(a, a)
	ty := ArrType(T, T)
	results := Run(1, func(q *Var) Goal { return TypeChecko(term, Nil, ty) })
	fmt.Println(len(results) > 0)
	// Output: true
}

// ExampleTypeChecko_app demonstrates checking an application against an environment.
func ExampleTypeChecko_app() {
	b := NewAtom("b")
	intT := NewAtom("Int")
	id := NewAtom("id")
	env := EnvExtend(EnvExtend(Nil, b, intT), id, ArrType(intT, intT))
	term := App(id, b)
	results := Run(1, func(q *Var) Goal { return TypeChecko(term, env, intT) })
	fmt.Println(len(results) > 0)
	// Output: true
}
