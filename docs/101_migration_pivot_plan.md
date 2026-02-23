# Plan: Architecture Pivot - Logic as Data

## Goal Description

Transition the DnDSL engine from hard-coded D&D 5e logic to a data-driven RPG simulator. By using **CEL-Go (Common Expression Language)**, we will store rules, requirements, and effects as strings in YAML data files. This makes the engine easily expandable and system-agnostic.

## User Review Required
>
> [!IMPORTANT]
>
> - **Schema Change**: The `Entity` struct will move from fixed fields (e.g., `Strength`) to a `Stats` map.
> - **Logic Shift**: Code like `if actor.Class == "Rogue"` will be replaced by evaluating CEL strings from data files.
> - **Manifest Introduction**: A `campaign.yaml` will now define system-wide formulas (e.g., how to calculate a "hit").

## Proposed Changes

### [Infrastructure] Rule Engine

#### [NEW] [rules](file:///home/paulo/org/projetos/dndsl/internal/rules)

- Create `internal/rules/env.go` to initialize the CEL environment.
- Register custom functions like `roll()` to handle dice strings within CEL.

### [Core] Data Model

#### [MODIFY] [state.go](file:///home/paulo/org/projetos/dndsl/internal/engine/state.go)

- Refactor `Entity` to use generic maps for stats and attributes.
- Add `Abilities` slice containing CEL conditions and effects.

#### [MODIFY] [models.go](file:///home/paulo/org/projetos/dndsl/internal/data/models.go)

- Update `Character` and `Monster` to match the generic `Entity` blueprint.

### [Engine] Command Resolution

#### [MODIFY] [attack.go](file:///home/paulo/org/projetos/dndsl/internal/command/attack.go)

- Replace D&D-specific hit/damage logic with calls to `eval()` using formulas from the `CampaignManifest`.

#### [NEW] [manifest.go](file:///home/paulo/org/projetos/dndsl/internal/engine/manifest.go)

- Define `CampaignManifest` to hold system-wide resolution logic.

## Verification Plan

### Automated Tests

- `go test ./internal/rules/...`: Verify CEL environment and custom functions.
- `go test ./internal/engine/generic_test.go`: Load a Sci-Fi character and verify a "hack" action works without Go code changes.
- Regression check: Ensure existing D&D tests still pass by providing a D&D `campaign.yaml`.

### Manual Verification

- Execute a custom ability defined solely in YAML via the REPL.
