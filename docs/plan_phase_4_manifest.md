# Phase 4 Implementation Plan: Generic Command Engine

This phase focuses on replacing hardcoded, system-specific command handlers (like `ExecuteAttack` and `ExecuteCheck`) with a **Generic Command Engine**. This engine resolves command logic dynamically by reading definitions from a central `CampaignManifest`.

## Key Goals

- **Unified Dispatch**: Centralize all combat and check logic into a single generic executor.
- **Data-Driven Rules**: Move complex 5e mechanics (advantage, proficiency, help actions) into YAML-based CEL formulas.
- **Refined Event Mapping**: Ensure the engine correctly captures both successful outcomes and numeric results (scores) for downstream feedback.

## Proposed Changes

### [Component] Data Models (`internal/data`)

#### [MODIFY] [models.go](file:///home/paulo/org/projetos/dndsl/internal/data/models.go)

- Implement `CommandDefinition` and `CampaignManifest` structs.
- Support step-based execution with optional event production.

### [Component] Rule Engine (`internal/rules`)

#### [MODIFY] [env.go](file:///home/paulo/org/projetos/dndsl/internal/rules/env.go)

- Expose resource fields and turn state in the CEL context.
- Enable `ext.Strings()` and `ext.Lists()` extensions.
- Implement RPG-specific helpers like `get_condition`.

### [Component] Generic Executor (`internal/command`)

#### [NEW] [executor.go](file:///home/paulo/org/projetos/dndsl/internal/command/executor.go)

- Implement `ExecuteGenericCommand` with result chaining via the `steps` map.
- Refine `mapManifestEvent` to correctly handle `CheckResolved` and `AttackResolved` nuances.

### [Component] Cleanup & Migration

#### [DELETE] [attack.go](file:///home/paulo/org/projetos/dndsl/internal/command/attack.go)

#### [DELETE] [check.go](file:///home/paulo/org/projetos/dndsl/internal/command/check.go)

- These files have been fully replaced by the manifest-driven generic executor.

## Verification Plan

### Automated Tests

- Full regression suite in `mechanics_test.go` and `recharge_test.go`.
- Verified automated `Help` condition removal and proficiency-based saving throw calculation.
