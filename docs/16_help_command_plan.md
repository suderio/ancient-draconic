# Phase 16: Context-Aware Help Command

Implement a context-aware help system that filters commands based on the actor and current game state.

## Proposed Changes

### 1. internal/parser

- **[MODIFY] `ast.go`**: Add `HelpCmd` struct and include it in `Command`.
- **[MODIFY] `lexer.go`**: Add `help` and `all` keywords to the lexer.
- **[MODIFY] `errors.go`**: Add usage instructions for the `help` command.

### 2. internal/command

- **[NEW] `help.go`**: Implement `ExecuteHelp`.
 - Filter commands based on `actor` (GM vs Player).
 - Filter commands based on `GameState` (e.g., combat actions only available during active turns).
 - Support `help by: <actor> <command|all>`.

### 3. internal/session

- **[MODIFY] `session.go`**: Hook `HelpCmd` into the `Execute` loop.

## Verification Plan

- Unit tests for contextual filtering (GM vs Player, Setup vs Combat).
- Manual verification in REPL.
