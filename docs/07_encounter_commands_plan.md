# Encounter Commands Implementation Plan

## Overview

Introduce three new commands: `encounter`, `add`, and `initiative`. These commands will manage the lifecycle of a combat encounter, track participants, validate GM constraints, and handle initiative rolls automatically based on the SRD.

## Command Syntax and AST Definitions

We will augment `internal/parser/ast.go` and `lexer.go` to support these patterns:

### 1. Encounter Command

- **Start**: `encounter by: GM start with: <Name> [and: <Name>]*`
- **End**: `encounter by: GM end`

### 2. Add Command

- **Syntax**: `add by: GM <Name> [and: <Name>]*`

### 3. Initiative Command

- **Syntax**: `initiative by: <Name>`

*(Note: We will extract the `and: <Name>` repeating blocks into a reusable AST list structure).*

## Validations Required

The commands must perform strict state-based validations before emitting events:

1. **Authorization**: `encounter` and `add` MUST be executed by the `GM`.
2. **Encounter State**:
  - `encounter start`: Valid only if there is **no** active encounter.
  - `encounter end`: Valid only if there **is** an active encounter.
  - `add`: Valid only if there **is** an active encounter.
  - `initiative`: Valid only if there **is** an active encounter.

If any of these fail, the `command` package will return an error, which the REPL will print out as an informative message without modifying the store.

## Auto-rolling Initiative & Data Integration

- When `encounter start` or `add` is executed, the engine automatically rolls initiative for all **monsters** listed.
- Characters roll their own initiative using the `initiative` command.

**Data Access**: To automatically roll a monster's initiative (1d20 + DEX modifier), the execution layer needs to fetch the monster's stats from the `internal/data` models. This means our `command` package will need access to a `data.Loader` or similar repository.

## Performance & State Management

You mentioned considering a distinct data structure for the duration of the encounter to avoid reading from the log for performance reasons.

**Current Architecture Advantage**: We already have this! The `GameState` struct inside the `session.Session` serves exactly this purpose.
When an event occurs, `Session.ApplyAndAppend(evt)` applies the event incrementally to the in-memory `GameState` without reading the entire log again.

**Proposed Enhancements to State**:
We will augment `GameState` (in `internal/engine/state.go`) to include:

- `IsEncounterActive bool`
- `Initiatives map[string]int`
This allows instant \(O(1)\) validation checks against the live state.

## Resolved Rules & Ambiguities

Based on the rules clarifications:

1. **Who can roll initiative?**
  Characters roll their own initiative. The command `initiative by: <Character>` is valid for characters, but `encounter` and `add` are locked strictly `by: GM`.

2. **Character vs. Monster Detection & Validation**:
  When joining an encounter (`encounter start with: <Name>` or `add by: GM <Name>`), the engine must perform strict file-based entity resolution:
  - First, check if `<Name>.yaml` (or a file with `name: <Name>`) exists in the `data/characters` directory. If yes, it's a **Character** (waits for manual initiative).
  - If not found, check the `data/monsters` directory by sanitizing the name: `replace(decapitalize(<Name>), " ", "-").yaml` (e.g., "Green Dragon" -> `green-dragon.yaml`). If yes, it's a **Monster** (engine auto-rolls initiative).
  - If neither are found, the command **fails immediately** with an error message indicating the entity could not be found.

3. **Initiative Value Override**:
  The engine will rely on the core validation rules for initiative. Specific bonuses/advantages will be parsed and evaluated seamlessly from the character sheets or rules logic as they are added in the future. The UI will just submit `initiative by: <Character>`.

## Proposed Execution Plan

1. Retrieve answers to the ambiguities above.
2. Extend `internal/parser` with mapping for `encounter`, `start/end`, `add`, `initiative`, `with`, `and`.
3. Enhance `GameState` to track `IsEncounterActive` and Initiative scores.
4. Implement `command.ExecuteEncounter`, `command.ExecuteAdd`, `command.ExecuteInitiative` injecting `GameState` for validation and `data.Loader` for Monster SRD fetching.
5. Route the parsing from `session.go` to these new commands.
