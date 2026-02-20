# Game State Management Implementation Plan

## 1. Goal Description

Establish the core DnD 5e DSL engine. The initial scope focuses purely on Game State Management (tracking active state such as HP, encounters, conditions, tracking turns) without Character Creation. The engine will adopt an **Event Sourcing** model to maintain a sequential log of facts.

* `data/` directory will be treated as the read-only source of truth for standard mechanics (monsters, spells, weapons).
* `worlds/` directory will serve as the read/write storage layer for long-term campaigns.

## 2. Proposed Changes

### Project Initialization

- Initialize a Go module (`go mod init dndsl`).
* Set up a root CLI command using Cobra (`cobra-cli init`) and configure `viper` to track the `worlds_dir` (default: `./worlds`) and `data_dir` (default: `./data`).

### Read-Only Data Layer (`internal/data`)

- Create a `Loader` service to read YAML files from `data/`.
* Provide typed Go structures matching the `data/` schema to support game logic (e.g., `Monster`, `Weapon`, `Spell`).

### Game Engine Core (`internal/engine`)

- **`GameState`**: The central struct tracking the active encounter state (entities, HP, initiative order, active conditions).
* **`Event` Interface**: Defines the contract for an event that mutates the state (e.g., `Apply(*GameState) error`).
* **Initial Events**: Implement `EncounterStarted`, `ActorAdded`, `DamageApplied`, `HPChanged`, `TurnChanged`.
* **`Projector`**: Rebuilds the current `GameState` by sequentially applying a slice of `Event` objects.
* **`Store` Interface**: Abstraction for appending events and retrieving the full event log.

### Persistence Layer (`internal/persistence`)

- Implement the `Store` interface with a file-backed JSONL serializer.
* Define a `CampaignManager` to handle `worlds/` organization.
  * Structure: `worlds/<WorldName>/<CampaignName>/log.jsonl`.
* Support `CampaignManager.Create(name)` and `CampaignManager.Load(name)`.

### CLI Interface (`cmd`)

- Scaffold the `dndsl campaign create <name>` and `dndsl campaign load <name>` commands.
* Implement the baseline structure for the `repl` command. Initially, parsing basic DSL fragments is optional, but it must be wired up to the `Store` appending logic so every action appends to `log.jsonl`.

## 3. Verification Plan

### Automated Tests

- Create `internal/engine/projector_test.go` to simulate a mock array of events (e.g., `ActorAdded` and `HPChanged`) and verify the `GameState` projector correctly reduces them.
* Create `internal/persistence/store_test.go` to test creating, appending to, and reloading a JSONL file in a temporary testing directory.

### Manual Testing

1. Run `go run main.go campaign create "Test World" "Starter Campaign"`.
2. Inspect `worlds/Test World/Starter Campaign/log.jsonl` to verify it was created.
3. Use a preliminary `repl` hook to spawn an internal test event and observe the JSONL file updates correctly.
