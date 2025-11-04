```go
func Example_hlapi_rowsAll_timeout() {
	x, y := Fresh("x"), Fresh("y")
	goal := Disj(
		Conj(Eq(x, A(1)), Eq(y, A("a"))),
		Conj(Eq(x, A(2)), Eq(y, A("b"))),
	)
	rows := RowsAllTimeout(50*time.Millisecond, goal, x, y)
	fmt.Println(len(rows))
	// Output:
	// 2
}

```


