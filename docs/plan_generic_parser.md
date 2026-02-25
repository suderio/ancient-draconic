# Goal Description

The engine currently relies on data-driven manifestations (`manifest.yaml`) to execute mechanical rules, but the Lexer/Parser (`ast.go`), the Session dispatcher (`session.go`), and the TUI auto-completion (`tui.go`) are still heavily hardcoded with D&D-specific terminology (e.g., `attack`, `dodge`, `ask`, `grapple`). This tight coupling prevents the system from seamlessly supporting completely different manifestations, like PDQ (which has commands like `conflict` and `simple_task`).

The objective is to refactor these entry layers to use a dynamic, unified `GenericCmd` structure that parses arbitrary key-value arguments, thereby completely removing D&D-specific AST structures and allowing the TUI and Session to be fully driven by the loaded `manifest.yaml`.

## Proposed Changes

### AST & Parser Layer

- **Delete D&D-specific AST structs:** Remove `AttackCmd`, `DodgeCmd`, `GrappleCmd`, `AskCmd`, `CheckCmd`, `HelpActionCmd`, `DamageCmd`, `ActionCmd`, `TurnCmd` and any specific D&D argument structs from `/internal/parser/ast.go`.
- **Introduce `GenericCmd`:**

  ```go
  type GenericCmd struct {
      Name    string     `parser:"@Ident"`
      Actor   *ActorExpr `parser:"( \"by\" \":\" @@ )?"`
      Args    []*ArgExpr `parser:"@@*"` // Parses generic "key: value value..." tuples
  }
  ```

- **Update Lexer (`lexer.go`):** Remove hardcoded D&D keywords like `attack`, `dodge`, `grapple` from the `Keyword` token rule so they seamlessly parse as generic `@Ident` command names. Keep core engine keywords (`roll`, `adjudicate`, `allow`, `deny`, `hint`).

### Session Layer

- **Simplify `session.go`:** Strip out the massive `if/else` block inspecting individual D&D AST nodes in `Execute()`. Instead, intercept `astCmd.Generic != nil` and elegantly map the parsed `ArgExpr` tuples into the generic `params := map[string]any{}` before calling `ExecuteGenericCommand()`. Engine-level operations (Rolls, Hints, Adjudications) will retain their specific dispatch logic.

### UI Layer

- **Dynamic Autocomplete in `tui.go`:** Replace the hardcoded `cmds := []string{"attack by: ", "dodge by: "...}` slice inside `updateSuggestions()`. Read from `m.app.Loader().LoadManifest().Commands` on initialization to dynamically generate initial command scaffolding suggestions based on the active campaign manifest.

## Verification Plan

### Automated Tests

- Run `go test ./internal/parser/...` after refactoring `ast.go` and updating test cases to verify generic argument tuple parsing works properly.
- Run `go test ./internal/command/...` to ensure that integration tests (which utilize the string Parser via Session or generic calls) still parse and correctly execute commands natively.
- Run `go test ./internal/session/...` to guarantee the dispatcher routes GenericCmds directly to `ExecuteGenericCommand`.

### Manual Verification

- Launch the TUI using `go run cmd/draconic/main.go repl`. Verify that pressing `tab` provides command suggestions natively gleaned from the loaded manifest (e.g. suggesting `conflict` when playing PDQ, or `shove` / `grapple` when playing D&D). Ensure command executions log properly to the TUI display.
