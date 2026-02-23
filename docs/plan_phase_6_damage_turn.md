# Phase 6: Damage and Turn Migration

This phase focuses on migrating the remaining hardcoded command logic (`damage` and `turn`) to the manifest-driven Generic Command Engine.

## Proposed Changes

### [internal/command]

#### [MODIFY] [executor.go](file:///home/paulo/org/projetos/dndsl/internal/command/executor.go)

- Refine `ExecuteGenericCommand` to handle `damage` specific context (multiplier, pending damage state).
- Update `mapManifestEvent` to support `HPChanged`, `RechargeRolled`, and `AbilityRecharged` events from manifest steps.

#### [DELETE] [damage.go](file:///home/paulo/org/projetos/dndsl/internal/command/damage.go)

#### [DELETE] [turn.go](file:///home/paulo/org/projetos/dndsl/internal/command/turn.go)

### [internal/rules]

#### [MODIFY] [bridge.go](file:///home/paulo/org/projetos/dndsl/internal/rules/bridge.go)

- Expose `Defenses` (Resistances, Immunities, Vulnerabilities) in the CEL context for entities.
- Expose `SpentRecharges` in the context to support turn-start recharge rolls.

### [data]

#### [MODIFY] [manifest.yaml](file:///home/paulo/org/projetos/dndsl/data/manifest.yaml)

- Add `damage` command:
  - Step 1: Calculate multiplier based on `action.type` and `target.defenses`.
  - Step 2: Roll damage and apply multiplier.
- Add `turn` command:
  - Step 1: End current turn (`TurnEnded`).
  - Step 2: Start next turn (`TurnChanged`).
  - Step 3 (Optional): Handle recharges for the next actor.

## Verification Plan

### Automated Tests

- Run `mechanics_test.go` and `recharge_test.go` to ensure these commands still function identically.
- Verify that `TestMonsterRecharge` passed using the manifest-driven logic.
