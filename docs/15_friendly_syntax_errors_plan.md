# Phase 15: Friendly Syntax Errors

Improve the user experience by replacing technical parser errors with friendly guidance.

## Proposed Changes

### 1. internal/parser

- **[NEW] `errors.go`**: Implement a `MapError(input string, err error) error` function.
 - Inspect the raw input to detect the intended command (e.g., checking the first word).
 - Return a friendly error message including the correct usage format for that command.
 - Return a generic "I wasn't able to understand your command" message for unrecognized input.

### 2. internal/session

- **[MODIFY] `session.go`**: Update `Execute` to use `parser.MapError` when `langParser.ParseString` fails.

## Verification Plan

### Automated Tests

- `internal/parser/errors_test.go`: Verify that incomplete commands (e.g., `attack by:`) return the expected friendly usage message.
- Verify that completely unrecognized input returns the fallback message.
