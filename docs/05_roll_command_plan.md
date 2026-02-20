# Roll Command Implementation Plan

This document outlines the approach for implementing the `roll` command in the DSL:
`Roll :by Someone1 3d6 + 1`

## Chosen Approach

We are proceeding with **Option C: Custom Implementation using `participle/v2`**.
This approach leverages the existing Lexer/AST builder we already have installed, ensuring 100% architectural compatibility and avoiding dirty string-passing to third-party libraries.

### Supported Syntax

The standard format is based on the industry standard RPG syntax:
`xdy[Modifiers][+/-c]`

**1. Standard Keep/Drop Modifiers:**
Syntax: `[k|d][h|l]z`

- `k`: Keep
- `d`: Drop
- `h`: Highest
- `l`: Lowest
- `z`: Count

*Examples:*

- `4d6kh3` (Roll 4d6, keep the highest 3) - Standard stat rolling
- `2d20dl1` (Roll 2d20, drop the lowest 1) - Equivalent to advantage
- `3d8kl2` (Roll 3d8, keep the lowest 2)

**2. Advantage/Disadvantage Shorthand:**
Syntax: `[a|d]`

- `a`: Advantage (Implicitly expands to `2dykh1`)
- `d`: Disadvantage (Implicitly expands to `2dykl1`)

*Examples:*

- `1d20a` (Roll 2d20, keep the highest 1)
- `1d20d` (Roll 2d20, keep the lowest 1)

**3. Flat Modifiers:**
Syntax: `[+/-c]`

- `+c`: Add static integer
- `-c`: Subtract static integer

## AST Defintion Strategy

We will update the grammar using Participle tags:

```go
type DiceExpr struct {
    Number   int           `@Int?` // Optional number of dice, defaults to 1
    Sides    int           `"d" | "D" @Int`
    AdvDis   string        `@( "a" | "A" | "d" | "D" )?` // Shorthand advantage/disadvantage
    KeepDrop *KeepDropExpr `@@?`
    Modifier *ModifierExpr `@@?`
}

type KeepDropExpr struct {
    Op    string `@("k"|"K"|"d"|"D")`
    Order string `@("h"|"H"|"l"|"L")`
    Count int    `@Int`
}

type ModifierExpr struct {
    Op    string `@("+" | "-")`
    Value int    `@Int`
}
```

## Action Plan

1. **Lexer Updates**: In `internal/parser`, define the lexical grammar. Since dice expressions don't use spaces strictly inside the token (like `3d6kh3+1`), the lexer might need to properly tokenize numbers and identifiers, or we might need to rely on the `Elide("Whitespace")` defaults.
2. **AST Definition**: Create `internal/parser/ast.go` to define `RollCommand` that embeds an `Actor` struct and the `DiceExpr` struct.
3. **Engine Logic**: Create an internal module `internal/engine/dice.go`.
    - Function `Eval(expr *DiceExpr) (int, []int, error)`.
    - Use `crypto/rand` mapped to standard ranges to ensure fair and secure rolls.
    - Evaluate keep/drop logic and sorting safely.
    - Expand shorthand `a`/`d` transparently into keep/drop logic before evaluation.
4. **REPL Wiring**: Integrate the `participle.Build()` call within `cmd/repl.go` to intercept inputs that match the `roll` command and simply print the result trace `(e.g., "Rolled [14, 2] = 14 + 1 = 15")` to standard output. Wait to emit any permanent JSON events until deeper architectural needs arise.
