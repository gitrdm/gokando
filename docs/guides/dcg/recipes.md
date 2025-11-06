# DCG recipes: common patterns

A grab bag of concise patterns you can copy into your own grammars. All examples assume:

- An SLG engine `engine := minikanren.NewSLGEngine(nil)`
- Rules defined with `minikanren.DefineRule(name, pattern)`
- Term/Pair/Nil list constructors from `minikanren`

See also: ../dcg/README.md (tutorial) and ../../api-reference/dcg.md (API).

## Tokenization pipeline

Turn a Go slice of strings into a miniKanren list of atoms for DCG input:

```go
func atoms(xs ...string) minikanren.Term {
    t := minikanren.Nil
    // build reversed then reverse again (or prepend in reverse order)
    for i := len(xs) - 1; i >= 0; i-- {
        t = minikanren.NewPair(minikanren.NewAtom(xs[i]), t)
    }
    return t
}

// Example input: (1 + 1)
input := atoms("1", "+", "1")
```

If you need character-level tokens, split the string first and apply the same builder.

## Zero or more / one or more

Define repetition using recursion with clause-order independence:

```go
// star(X) ::= ε | X star(X)
func defineStar(rule string, x minikanren.GoalPattern) {
    minikanren.DefineRule(rule, minikanren.Alternation(
        // ε (empty): input == output
        minikanren.Seq(),
        // X followed by star(X)
        minikanren.Seq(x, minikanren.NonTerminal(engine, rule)),
    ))
}

// plus(X) ::= X star(X)
func definePlus(rule string, x minikanren.GoalPattern, starRule string) {
    minikanren.DefineRule(rule, minikanren.Seq(x, minikanren.NonTerminal(engine, starRule)))
}
```

Usage example for digits:

```go
digit := minikanren.Terminal(minikanren.NewAtom("1")) // or Alternation of many terminals
defineStar("digits0", digit)
definePlus("digits1", digit, "digits0")
```

## Optional

```go
// opt(X) ::= ε | X
func defineOpt(rule string, x minikanren.GoalPattern) {
    minikanren.DefineRule(rule, minikanren.Alternation(
        minikanren.Seq(),
        x,
    ))
}
```

## Separated lists (sepBy / sepBy1)

```go
// list1 ::= item (sep item)*
func defineSepBy1(rule string, item, sep minikanren.GoalPattern, manyRule string) {
    // manyTail ::= sep item manyTail | ε
    minikanren.DefineRule(manyRule, minikanren.Alternation(
        minikanren.Seq(sep, item, minikanren.NonTerminal(engine, manyRule)),
        minikanren.Seq(),
    ))
    minikanren.DefineRule(rule, minikanren.Seq(item, minikanren.NonTerminal(engine, manyRule)))
}

// list ::= list1 | ε
func defineSepBy(rule, list1 string) {
    minikanren.DefineRule(rule, minikanren.Alternation(
        minikanren.NonTerminal(engine, list1),
        minikanren.Seq(),
    ))
}
```

## Expression precedence (classic E→T→F)

Left-associative operators are natural with left recursion under SLG:

```go
// E ::= E "+" T | T
// T ::= T "*" F | F
// F ::= "(" E ")" | num
num := minikanren.Terminal(minikanren.NewAtom("1")) // stand-in for number tokens

minikanren.DefineRule("F", minikanren.Alternation(
    minikanren.Seq(minikanren.Terminal(minikanren.NewAtom("(")),
                   minikanren.NonTerminal(engine, "E"),
                   minikanren.Terminal(minikanren.NewAtom(")"))),
    num,
))

minikanren.DefineRule("T", minikanren.Alternation(
    minikanren.Seq(minikanren.NonTerminal(engine, "T"),
                   minikanren.Terminal(minikanren.NewAtom("*")),
                   minikanren.NonTerminal(engine, "F")),
    minikanren.NonTerminal(engine, "F"),
))

minikanren.DefineRule("E", minikanren.Alternation(
    minikanren.Seq(minikanren.NonTerminal(engine, "E"),
                   minikanren.Terminal(minikanren.NewAtom("+")),
                   minikanren.NonTerminal(engine, "T")),
    minikanren.NonTerminal(engine, "T"),
))
```

Notes:
- Clause order doesn’t affect correctness; SLG finds the fixpoint and deduplicates identical parses.
- For recognition mode, set output to Nil; for parsing with remainder, unify the remainder with a variable.
- Use Alternation for choice; it runs branches concurrently to avoid stalls under recursion.
