```go
func ExampleNoto() {
	// Noto succeeds because Membero(4, [1,2,3]) fails
	goalSuccess := Noto(Membero(NewAtom(4), List(NewAtom(1), NewAtom(2), NewAtom(3))))
	resultsSuccess := runGoal(goalSuccess)
	fmt.Printf("Noto(fail) succeeds: %v\n", len(resultsSuccess) > 0)

	// Noto fails because Membero(2, [1,2,3]) succeeds
	goalFail := Noto(Membero(NewAtom(2), List(NewAtom(1), NewAtom(2), NewAtom(3))))
	resultsFail := runGoal(goalFail)
	fmt.Printf("Noto(success) fails: %v\n", len(resultsFail) == 0)
	// Output:
	// Noto(fail) succeeds: true
	// Noto(success) fails: true
}

```


