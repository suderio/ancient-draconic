# Domain Vocabulary and Invariants

## Purpose

Define shared business language and non-negotiable invariants so behavior remains consistent across modules.

## Key Terms

| Term | Definition |
| :--- | :--- |
| **Entity** | Any tracked game participant (character, monster, NPC). Has stats, resources, conditions, proficiencies. |
| **Manifest** | The Lua file (`manifest.lua`) defining all game rules: commands, restrictions, and helper functions. |
| **Command** | A named action defined in the manifest (e.g., `encounter_start`, `grapple`, `check`). |
| **Formula** | A Lua expression (string or closure) evaluated at command execution time to compute a game value. |
| **Game Step** | A named formula + event pair within a command's execution pipeline. |
| **Phase** | One of three execution stages: `game` (global effects), `targets` (per-target effects), `actor` (actor-affecting effects). |
| **Event** | An immutable record of a state change produced by a game step (e.g., `LoopEvent`, `AttributeChangedEvent`). |
| **Loop** | An ordered sequence of entity turns (e.g., combat encounter). Has actors, turn order, and active/inactive state. |
| **Game State** | The full projection of all entities, loops, and metadata, built from applied events. |
| **Sandbox** | The restricted Lua environment where formulas execute. Only safe libraries are available. |
| **Session** | The runtime context tying together manifest, state, input parsing, and event persistence. |
| **Campaign** | A named directory under a world containing an event log and campaign-specific entity data. |
| **World** | A top-level directory containing the manifest, shared data, and one or more campaigns. |

## Business Invariants

1. Game state MUST be fully reproducible by replaying the event log from scratch.
2. Events are immutable once persisted; corrections are new events, not mutations.
3. Each command execution MUST produce zero or more events; no silent state changes.
4. Formulas MUST be evaluated at execution time, never at manifest load time.
5. Loop actor ordering MUST be deterministic for identical order values (stable sort).
6. Entity identity is by `ID`; names are display-only and not unique.

## Phase Execution Order

Commands execute phases in strict order: `prereq` → `game` → `targets` → `actor`. Each phase accumulates results into its own namespace (`game`, `targets`, `actor_results`). A failed prereq aborts the entire command.

## Failure Modes

1. Formula evaluated at load time instead of execution time (eager evaluation bug).
2. Event log diverges from in-memory state (replay inconsistency).
3. Entity referenced by ID that doesn't exist in game state.
