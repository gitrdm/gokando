# DCG tutorial: pattern-based parsing with SLG tabling

This short tutorial walks through building a tiny grammar using the pattern-based DCG layer integrated with the SLG tabling engine. You’ll see recognition, parsing with a remainder variable, and proper left recursion—all declarative and clause-order independent.

If you’re new to the DCG API, skim the API page first, then come back here for a hands-on path.

- API reference: ../../api-reference/dcg.md
- Examples (Go): ../../..//pkg/minikanren/dcg_examples_test.go

## Prerequisites

- Import the package and create an SLG engine.
- DCG rules are registered once via `DefineRule(name, body)` where `body` is a GoalPattern built from `Terminal`, `Seq`, `Alternation`, and `NonTerminal`.
- Parsing uses `ParseWithSLG(engine, ruleName, input, output)` where input/output are difference-list endpoints (lists built from `Pair`/`Nil`).

Minimal setup snippet (for reference):

```go
engine := minikanren.NewSLGEngine(nil)
minikanren.DefineRule("digit1", minikanren.Terminal(minikanren.NewAtom("1")))
```

## Step 1 — Recognition mode (ground → ground)

We’ll recognize the whole input by making the output ground (empty list). For a simple rule `ab ::= "a" "b"`:

```go
engine := minikanren.NewSLGEngine(nil)
minikanren.DefineRule("ab", minikanren.Seq(
    minikanren.Terminal(minikanren.NewAtom("a")),
    minikanren.Terminal(minikanren.NewAtom("b")),
))

input := // (a b)
    minikanren.NewPair(minikanren.NewAtom("a"),
        minikanren.NewPair(minikanren.NewAtom("b"), minikanren.Nil))
rest := minikanren.NewAtom(nil) // expect fully consumed

// Succeeds iff input matches exactly "a b"
goal := minikanren.ParseWithSLG(engine, "ab", input, rest)
```

Tip: This mirrors Example_dcgRecognition.

## Step 2 — Parsing with a remainder (ground → variable)

Let `q` unify with the remainder after parsing `ab`. This is useful when chaining grammars or inspecting what’s left.

```go
rest := minikanren.Fresh("rest")
goal := minikanren.Conj(
    minikanren.ParseWithSLG(engine, "ab", input, rest),
    minikanren.Eq(q, rest),
)
```

This follows Example_dcgSeq/Example_dcgTerminal patterns.

## Step 3 — Left recursion via SLG (clause-order independent)

Define a classic left-recursive grammar:

```
expr  ::= term | expr "+" term
term  ::= "1"
```

Implementation (note the shared SLG engine and NonTerminal calls):

```go
engine := minikanren.NewSLGEngine(nil)
minikanren.DefineRule("expr", minikanren.Alternation(
    minikanren.NonTerminal(engine, "term"),
    minikanren.Seq(
        minikanren.NonTerminal(engine, "expr"),
        minikanren.Terminal(minikanren.NewAtom("+")),
        minikanren.NonTerminal(engine, "term"),
    ),
))
minikanren.DefineRule("term", minikanren.Terminal(minikanren.NewAtom("1")))

// Parse (1 + 1)
input := minikanren.NewPair(minikanren.NewAtom("1"),
    minikanren.NewPair(minikanren.NewAtom("+"),
        minikanren.NewPair(minikanren.NewAtom("1"), minikanren.Nil)))
rest := minikanren.Fresh("rest")

// Works regardless of clause order thanks to SLG fixpoint and answer dedup
goal := minikanren.Conj(
    minikanren.ParseWithSLG(engine, "expr", input, rest),
    minikanren.Eq(q, rest),
)
```

This mirrors Example_dcgParse_leftRecursion and the left recursion tests. The SLG engine guarantees termination and deduplicates identical answers.

## Where to go next

- API deep-dive: ../../api-reference/dcg.md
- Explore ready-to-run examples: ../../..//pkg/minikanren/dcg_examples_test.go
- Browse recipes for common patterns: ./recipes.md
- Try adding tokens and non-terminals to build up a small expression grammar.
- For performance tips and semantics (clause-order independence, ambiguity dedup), see the API doc’s notes.

## Performance notes

- Streaming vs batch: ParseWithSLG streams answers incrementally. Conj in the evaluator consumes answers with small batches (effectively one at a time) so left/mixed/right recursion terminate and interleave. Prefer this over collecting all answers eagerly.
- Alternation concurrency: Alternation runs branches concurrently to avoid stalls under recursion. This is important for left-associative grammars. For very wide alternations, consider using context deadlines to keep runaway branches in check.
- Deduplication: Ambiguous grammars that yield structurally identical bindings are deduplicated by the AnswerTrie, so you’ll typically see one canonical answer.
