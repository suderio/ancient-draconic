# Phase 32: Monster Recharge Logic

Implement the "Recharge X-Y" mechanic for monster abilities, ensuring that powerful actions like Dragon's Breath (Recharge 5-6) are correctly tracked and recharged during combat.

## Proposed Changes

### [internal/data](file:///home/paulo/org/projetos/draconic/internal/data)

#### [MODIFY] [models.go](file:///home/paulo/org/projetos/draconic/internal/data/models.go)

- Update `Action` struct to include `Recharge string`json:"recharge" module:"yaml"`.

### [internal/engine](file:///home/paulo/org/projetos/draconic/internal/engine)

#### [MODIFY] [state.go](file:///home/paulo/org/projetos/draconic/internal/engine/state.go)

- Add `SpentRecharges map[string][]string` to `GameState` to track which actions are currently cooling down for each entity.

#### [MODIFY] [event.go](file:///home/paulo/org/projetos/draconic/internal/engine/event.go)

- [NEW] `RechargeRollEvent`: Records a d6 roll for a specific ability.
- [NEW] `AbilityRechargedEvent`: Clears the spent status for an ability.
- [NEW] `AbilitySpentEvent`: Adds an ability to the spent list.
- [MODIFY] `TurnChangedEvent.Apply`: Logic to iterate over `SpentRecharges` for the new actor and generate recharge rolls.

### [internal/command](file:///home/paulo/org/projetos/draconic/internal/command)

#### [MODIFY] [attack.go](file:///home/paulo/org/projetos/draconic/internal/command/attack.go)

- Check if the requested weapon/action index is in `SpentRecharges`.
- Emit `AbilitySpentEvent` if the action has a `Recharge` property.

## Verification Plan

### Automated Tests

- `internal/command/recharge_test.go`:
  - Verify an ability becomes spent after use.
  - Verify recharge roll succeeds on 5-6 (mocked dice).
  - Verify ability becomes available again.

### Manual Verification

- Test with a monster like "Displace Beast" or custom Dragon in the TUI.
