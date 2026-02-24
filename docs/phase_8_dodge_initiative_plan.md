# Plan - Phase 8: Migrating Dodge and Initiative

I will migrate the `dodge` and `initiative` commands to the manifest-driven engine to complete the core combat action set.

## Proposed Changes

### [Manifest] [manifest.yaml](file:///home/paulo/org/projetos/dndsl/data/manifest.yaml)

- [NEW] Define `dodge` command.
- [NEW] Define `initiative` command.
- [MODIFY] Update `attack` formula to apply disadvantage if the target has the `Dodging` condition.

### [Command Engine] [executor.go](file:///home/paulo/org/projetos/dndsl/internal/command/executor.go)

- [MODIFY] Map `DodgeTaken` and `InitiativeRolled` manifest events to engine events.
- [MODIFY] Add `RollInitiative` helper to the `command` package.
- [MODIFY] Fix `ResolveEntityAction` to ignore empty strings (preventing recharge check bugs).

### [Rules Bridge] [bridge.go](file:///home/paulo/org/projetos/dndsl/internal/rules/bridge.go)

- [MODIFY] Add `prof_bonus` to default stats to avoid CEL evaluation errors for uninitialized entities.

### [Session] [session.go](file:///home/paulo/org/projetos/dndsl/internal/session/session.go)

- [MODIFY] Route `dodge` and `initiative` AST commands to `ExecuteGenericCommand`.

## Verification Plan

### Automated Tests

- `go test -v ./internal/command/...`
- Specific tests in `mechanics_test.go`:
  - `TestDodgeMechanic`: Verify that dodging grants disadvantage to attackers.
  - `TestTwoWeaponFighting`: Verify that initiative and action economy work correctly with the new engine.
- Verify `recharge_test.go` passes (regression check for the recharge bug).
