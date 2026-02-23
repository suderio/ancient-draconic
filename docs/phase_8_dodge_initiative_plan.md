# Implementation Plan - Phase 8: Dodge and Initiative

Implement the `dodge` and `initiative` mechanics using the manifest-driven generic command engine.

## Proposed Changes

### [manifest.yaml](file:///home/paulo/org/projetos/dndsl/data/manifest.yaml)

- **Dodge**:
  - Adds a "Dodging" condition to the actor.
  - Consumes one Action.
- **Attack**:
  - Update the `hit` step formula to check if the target has the "Dodging" condition.
  - If dodging, the attack roll should have disadvantage (simulated via `roll('2d20l1')` or similar).
- **Initiative**:
  - New command to roll initiative.
  - Formula: `roll('1d20') + mod(actor.stats.dex)`.
  - Event: Map the result to set the actor's initiative in the game state.

### [internal/command/executor.go](file:///home/paulo/org/projetos/dndsl/internal/command/executor.go)

- Ensure the `HPChanged` and other event mappings can handle the results from the new commands.
- (TBD) Add mapping for initiative updates if the generic `AttributeSet` is insufficient.

## Verification Plan

### Automated Tests

- `TestDodgeMechanic`: Verify that taking the dodge action adds the condition and subsequent attacks have correctly adjusted formulas.
- `TestInitiativeCommand`: Verify that rolling initiative updates the game state's `Initiatives` map and `TurnOrder`.
