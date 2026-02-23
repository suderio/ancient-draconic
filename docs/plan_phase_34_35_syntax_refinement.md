# Plan: Combat Mechanics Syntax and Rule Refinement

This plan aims to refine the Two-Weapon Fighting (TWF) and Opportunity Attack mechanics to match common 5e rules and user-specified syntax.

## User Review Required

> [!IMPORTANT]
>
> - **Syntax Change**: `attack bonus` becomes `attack off-hand` and `attack reaction` becomes `attack opportunity`.
> - **Strict Order**: The `by: <actor>` expression must now immediately follow the `attack` command.
> - **TWF Restrictions**: Off-hand attacks now require the actor to have attacked previously in the same turn with a different weapon.

## Proposed Changes

### [Component] Parser/Lexer

#### [MODIFY] [lexer.go](file:///home/paulo/org/projetos/dndsl/internal/parser/lexer.go)

- Rename `bonus` keyword to `off-hand`.
- Rename `reaction` keyword to `opportunity`.

#### [MODIFY] [ast.go](file:///home/paulo/org/projetos/dndsl/internal/parser/ast.go)

- Refactor `AttackCmd` struct:
  - Rename `Bonus` field to `OffHand`.
  - Rename `Reaction` field to `Opportunity`.
  - Update parser tags to ensure `OffHand` and `Opportunity` follow the `Actor` field.

---

### [Component] Engine State

#### [MODIFY] [state.go](file:///home/paulo/org/projetos/dndsl/internal/engine/state.go)

- Add `HasAttackedThisTurn bool` and `LastAttackedWithWeapon string` to the `Entity` struct.

#### [MODIFY] [event.go](file:///home/paulo/org/projetos/dndsl/internal/engine/event.go)

- Update `AttackResolvedEvent.Apply`:
  - If it's a standard attack (not off-hand/opportunity), set `ent.HasAttackedThisTurn = true` and `ent.LastAttackedWithWeapon = e.Weapon`.
- Update `TurnChangedEvent.Apply`:
  - Reset `ent.HasAttackedThisTurn = false` and `ent.LastAttackedWithWeapon = ""` for the active actor.

---

### [Component] Commands

#### [MODIFY] [attack.go](file:///home/paulo/org/projetos/dndsl/internal/command/attack.go)

- Update `ExecuteAttack` to use the new fields:
  - If `cmd.OffHand` is set:
    - Verify `ent.HasAttackedThisTurn` is true.
    - Verify `cmd.Weapon` is different from `ent.LastAttackedWithWeapon`.
- Update `ExecuteAttack` to handle renamed fields (`OffHand`, `Opportunity`).

---

### [Component] Tests

#### [MODIFY] [mechanics_test.go](file:///home/paulo/org/projetos/dndsl/internal/command/mechanics_test.go)

- Update `TestTwoWeaponFighting` and `TestOpportunityAttack` to use new syntax.
- Add test case verifying TWF failure if no prior attack or same weapon used.

## Verification Plan

### Automated Tests

- Run updated unit tests:

  ```bash
  go test -v ./internal/command/...
  ```

- Specifically verify new behavior:
  - `attack off-hand` fails if it's the first attack of the turn.
  - `attack off-hand` fails if using the same weapon as the first attack.
  - `attack off-hand` succeeds if after a standard attack with a different weapon.
  - `attack by: Elara off-hand` works.
  - `attack off-hand by: Elara` fails (syntax error).
