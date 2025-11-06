# Definite Clause Grammars (DCG) with SLG Tabling

DCG in gokanlogic is built on a pattern-based architecture that integrates directly with the SLG tabling engine. This enables declarative clause-order independence and robust support for left recursion without operational hacks or timeouts.

## Core ideas

- Patterns are data, not executions. A grammar rule returns a GoalPattern that describes the computation; the SLG engine orchestrates its evaluation.
- Difference lists represent the input/output pair for parsing. A rule relates an input list s0 to an output list s1.
- Left recursion works via tabling. Recursive nonterminals route through SLG, which computes a fixpoint over strongly connected components.
- Clause-order independence. Alternation builds a set of branches; under SLG, equivalent derivations are deduplicated and evaluation order does not change the result set.

## API

- Terminal(token Term) GoalPattern
  - Match a single token at the head of the input list.
- Seq(patterns ...GoalPattern) GoalPattern
  - Sequence patterns by threading an intermediate difference list.
- Alternation(patterns ...GoalPattern) GoalPattern
  - Declarative choice over patterns. Implemented with Disj for non-blocking parallel branch evaluation.
- NonTerminal(engine *SLGEngine, name string) GoalPattern
  - Reference another grammar rule. All recursive calls route through the provided SLG engine.
- DefineRule(name string, body GoalPattern)
  - Register a rule by name. The body must be a pattern (no execution inside constructors).
- ParseWithSLG(engine *SLGEngine, ruleName string, input, output Term) Goal
  - Entry point for parsing. Builds a tabled call and streams answers through SLG.

Contract (tiny):
- Inputs: input, output are terms representing lists (Nil or Pair); either may be variables.
- Success: Goal succeeds for each parse that transforms input → output according to the rule.
- Error modes: Undefined rule yields zero answers (graceful failure). Context cancellation is honored.

## Usage snippets

- Recognition mode (ground → ground): see Example_dcgRecognition.
- Parsing with variables (ground input, var output): see Example_dcgSeq and Example_dcgTerminal.
- Left recursion: see Example_dcgParse_leftRecursion.
- Ambiguous grammar deduplication: see Example_dcgAmbiguousDedup.
- Undefined rule behavior: see Example_dcgUndefinedRule.

## Clause-order independence

Because Alternation constructs a set of branches and NonTerminal calls are routed through SLG, the engine computes a fixpoint over cyclic dependencies. This removes operational sensitivity to whether a base clause appears before or after a recursive clause. The tests cover base-first, recursive-first, and mixed orders.

## Ambiguity and deduplication

SLG caches answers in an AnswerTrie and deduplicates identical bindings. Ambiguous grammars that derive the same bindings via multiple branches will produce a single answer. If different derivations produce different bindings (e.g., different remainders), you will see multiple answers accordingly.

## Performance notes

- Alternation evaluates branches with Disj, allowing progress on one branch even if another is blocked under recursion. This is critical to reach SLG fixpoints.
- ParseWithSLG streams answers incrementally using Take(1), minimizing head-of-line blocking for recursive grammars.

## Troubleshooting

- Left recursion hangs: ensure your NonTerminal uses the same SLG engine instance passed to ParseWithSLG and that rule bodies return patterns (no goal execution inside constructors).
- No answers for an existing rule: confirm the input is a proper list (Nil or Pair via NewPair), and tokens match the exact atoms you constructed.
- Too many answers: check for genuinely distinct bindings; SLG only deduplicates structurally identical answers.
