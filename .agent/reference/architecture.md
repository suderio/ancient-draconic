# Architecture and Boundaries

## Purpose

Define component boundaries, dependency direction, and orchestration ownership.

## Layer Model

1. `main.go`: executable entrypoint only.
2. `cmd/`: command parsing, TUI, bot startup, and output formatting.
3. `internal/engine/`: game engine — types, Lua evaluator, executor, manifest loader. **Owns the Lua sandbox** (`*lua.LState`) and exposes formula evaluation to session.
4. `internal/session/`: session orchestration — input parsing, event store, campaign management. Holds a `sync.Mutex` to serialize Lua access.
5. `internal/data/`: legacy data models and loaders (to be migrated).
6. `internal/dnd5eapi/`: SRD data retrieval from the D&D 5e API.
7. `internal/telegram/`: Telegram bot integration.

### Lua Boundary

`manifest.lua` is **user-authored game logic**, not Go code. It sits between the engine (Go) and the game rules (Lua). The engine loads it, captures the `commands` and `restrictions` tables, and evaluates formulas through the sandbox. This boundary means:

- New game rules MUST NOT require Go code changes.
- The sandbox MUST restrict Lua to `base`, `table`, `string`, `math` (no `os`, `io`, `debug`).
- Go functions exposed to Lua (`roll()`, `mod()`) are registered at `LState` creation.

## Allowed Dependency Directions

1. `cmd` → `internal/session`, `internal/telegram`.
2. `internal/session` → `internal/engine`.
3. `internal/telegram` → nothing (defines its own `Executor` interface; `cmd/` provides the adapter).

## Forbidden Dependencies

1. `internal/engine` MUST NOT import `internal/session` or `cmd`.
2. `internal/*` MUST NOT import `cmd` packages.
3. `internal/telegram` MUST NOT import `internal/engine` directly (uses interface).

## Component Interaction

### TUI

1. GM starts TUI via `draconic repl`.
2. Players connect via Telegram.
3. GM and Players send commands.
4. Game State is updated by every command.

### Loop

1. GM starts a game loop (e.g., `encounter start`).
2. GM and Players send commands in the order allowed by the loop definition.
3. At every command the Game State is updated.
4. At every loop iteration the Game State is updated with the actions described in the loop definition.

## Failure Modes

1. Circular dependencies between session and engine.
2. Telegram worker importing engine types directly instead of using the interface.
