# Phase 20: Syntax Refactor (Post-Colon Prepositions)

This phase involves refactoring the Domain Specific Language (DSL) to use post-colon prepositions (e.g., `with:`, `by:`, `to:`) instead of the current pre-colon syntax (e.g., `with:`, `by:`, `to:`). This change aims to make the language more natural and readable for users.

## User Review Required

> [!IMPORTANT]
> This is a breaking change for the DSL. All existing automation scripts or saved command logs using the old syntax will fail until migrated.

## Proposed Changes

### 1. internal/parser/ast.go

- **[MODIFY]**: Update all `parser` tags in structs to swap the order of `":"` and the keyword.
- **Example**:
 - Before: `parser:": " "by"`
 - After: `parser:" "by" ":"`

### 2. internal/telegram/worker.go

- **[MODIFY]**: Update the command translation logic that injects the `by:` preposition.
- **Example**:
 - Before: `parts[0] + " by: " + actorID`
 - After: `parts[0] + " by: " + actorID`

### 3. internal/parser/ast_test.go

- **[MODIFY]**: Update all test strings to use the new syntax.

### 4. README.md

- **[MODIFY]**: Update all DSL examples to reflect the new syntax.

### 5. internal/parser/errors.go

- **[VERIFY]**: Ensure error mapping still provides useful feedback for the new syntax.

## Verification Plan

### Automated Tests

- Run `go test ./internal/parser/...` to verify all AST parsing logic.
- Run `go test ./internal/telegram/...` (if any tests exist that check command translation).
- Run the full suite with `go test ./...`.

### Manual Verification

- Start the REPL and manually type several commands (attack, damage, initiative) using the new `with:`, `by:`, `to:` syntax.
- Verify that the help command still works and (if it shows examples) displays the new syntax.
