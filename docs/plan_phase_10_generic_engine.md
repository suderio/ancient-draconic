# Phase 10: Truly Generic Engine Architecture

This phase aims to decouple the game engine from D&D 5e specifically, making it a truly generic engine where all system-specific rules, stats, and flows are defined in the manifest.

## User Review Required

> [!IMPORTANT]
> This phase involves breaking changes to the `engine.Entity` and `engine.GameState` structures, as well as the event log format. Legacy data will not be compatible.

> [!WARNING]
> Hardcoded references to "str", "dex", "prof_bonus", etc., will be removed from Go code and moved to the manifest.

## Proposed Changes

### [engine] core data model refactor

#### [MODIFY] [internal/engine/state.go](file:///home/paulo/org/projetos/dndsl/internal/engine/state.go)

- Redefine `Entity` struct according to the proposed schema:
  - `Types []string`
  - `Classes map[string]string`
  - `Stats map[string]int`
  - `Resources map[string]int` (max values)
  - `Spent map[string]int` (current usage)
  - `Conditions []string`
  - `Proficiencies map[string]int`
  - `Statuses map[string]string`
  - `Inventory map[string]int`
- Refactor `GameState` to be more flexible:
  - Move `PendingDamage`, `PendingChecks`, `PendingAdjudication` into a more generic `Metadata map[string]any` or similar if needed, or keep them if they can be generalized.
  - Redefine `IsFrozen` to be manifest-driven.
- **Generic Freeze Mechanism**: Implement a `freeze` command capability in the manifest. A command can trigger a "Freeze" state with a specified `unfreeze_condition` (CEL expression). The engine remains frozen until the condition evaluates to true.

#### [MODIFY] [internal/engine/event.go](file:///home/paulo/org/projetos/dndsl/internal/engine/event.go)

- Implement a generic `Event` system.
- Instead of `HPChangedEvent`, `DodgeTakenEvent`, etc., use generic events like:
  - `AttributeChangedEvent` (stat, resource, status)
  - `ConditionToggledEvent`
  - `ResourceTransactionEvent`
- The `Apply` method will become more generic, possibly using manifest logic to determine state transitions.

### [command] manifest-driven meta-commands

#### [MODIFY] [internal/command/executor.go](file:///home/paulo/org/projetos/dndsl/internal/command/executor.go)

- Update `ExecuteGenericCommand` to handle the new `Entity` structure.
- Remove weapon-specific hardcoding (e.g., `ResolveEntityAction` calls that expect "weapon" param).
- **Maneuver Separation**: Refactor `opportunity` and `off-hand` attacks as separate manifest commands rather than flags within the main `attack` command. This simplifies the core `attack` logic.

#### [MODIFY] [internal/session/session.go](file:///home/paulo/org/projetos/dndsl/internal/session/session.go)

- Refactor `Execute` to remove the huge `if/else` block.
- Use the `Registry` to find the command definition and then call `ExecuteGenericCommand`.
- Support meta-commands (`adjudicate`, `help`, `hint`) by reading their definitions from the manifest.

### [data] manifest schema expansion

#### [MODIFY] [data/manifest.yaml](file:///home/paulo/org/projetos/dndsl/data/manifest.yaml)

- Add sections for meta-commands.
- Example:

  ```yaml
  adjudicated:
    commands: ["grapple", "shove"] # 'opportunity' moved to separate command
  help:
    # help descriptions for commands
  hint:
    # hint logic
  ```

## Verification Plan

### Dead Code Cleanup

- Perform a comprehensive search for and removal of dead code, legacy structs, and unused constants across the entire repository once the migration is complete.

### Automated Tests

- `go test ./...`
- Create new integration tests that use a completely different RPG system (e.g., a simple 2D6 system) to verify "genericity".

### Manual Verification

- Test `help <command>` in the REPL.
- Test `adjudicate` flow for various commands.
