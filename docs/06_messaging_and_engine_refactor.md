# Event Messaging & Engine Refactoring Plan

## Motivation

As the DnD DSL evolves and supports more commands, placing the responsibility of formatting output messages (e.g., parsing the dice roll trace, displaying HP changes) within the `repl` command becomes unmaintainable.

Furthermore, the REPL is just one presentation layer. In the future, other interfaces (like a Discord, Telegram, or Signal bot) will need to execute these same commands and display the resulting messages.

To address this, we need to completely decouple:

1. **Command Execution**: Transforming an AST into an `Event`.
2. **Message Generation**: Describing the `Event` in a human-readable format.
3. **Presentation**: Printing the text to the terminal or sending it as a chat message.
4. **State Management**: Handling the appending to the store and the loading of the game state gracefully.

## Proposed Changes

### 1. Update the `engine.Event` Interface

We will add a method to the `Event` interface so each event intrinsically knows how to describe itself. This ensures that any presentation layer can simply request the string representation of an event.

```go
type Event interface {
 Type() EventType
 Apply(state *GameState) error
 Message() string // <-- NEW: Formats the event for human reading
}
```

*Implementation*:

- `DiceRolledEvent.Message()` will return the tree-formatted string of the dice trace.
- `HPChangedEvent.Message()` will return something like `"Took 3 damage"` or `"Healed for 5 HP"`.
- `ActorAddedEvent.Message()` will return `"Added Goblin (10/10 HP)"`.

*(Alternatively, this formatting logic could be placed in the `command` package or a dedicated rendering layer, but attaching it to the `Event` is an elegant way to ensure that historical logs loaded from disk can also easily be displayed when inspected).*

### 2. Refactor `cmd/repl.go` Presentation Logic

We will strip out the entire `switch` block parsing the pseudo-commands (`add`, `damage`, `heal`) and the hardcoded type assertions `if r, ok := evt.(*engine.DiceRolledEvent)`.

The REPL loop will become purely a presentation and entry point layer:

```go
evt, err := dispatcher.Execute(line) // Example conceptual flow
if err != nil {
    fmt.Printf("Error: %v\n", err)
} else if evt != nil {
    fmt.Println(evt.Message())
}
```

### 3. Introduce a `GameSession` (Engine Level Controller)

Currently, `repl.go` manages appending the event to the JSONL log, reloading entirely from disk, and rebuilding the projection (`proj.Build(events)`). This is inefficient and exposes persistence details to the UI.

We should centralize this to an `engine.Session` or `engine.Game` manager.

- **`Session.Execute(cmdString string) (engine.Event, error)`**: Takes raw terminal/chat input via the parser, delegates to the `command` package, appends the resulting event to the store, mutates the in-memory `GameState`, and returns the event so the caller can print `.Message()`.

## Rollout Steps

1. **[Refactor Event Interface]**: Add `Message() string` to `engine.Event` and implement it across all existing event structs in `internal/engine/event.go`.
2. **[Refactor REPL Display]**: Strip formatting logic out of `repl.go` and replace it with `fmt.Println(evt.Message())`.
3. **[Migrate Pseudo-Commands]**: Move `add`, `heal`, `damage` from hardcoded REPL checks into actual AST commands or fallback patterns managed inside `internal/command`.
4. **[Centralize Game State]**: Create `internal/engine/session.go` to wrap the `Store` and `GameState` projection, abstracting file persistence out of the REPL.
