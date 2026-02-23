# Implementation Plan: Saving Throw Fixes & New Actions

Address the misunderstanding between ability checks and saving throws, and implement the remaining combat actions: Two-Weapon Fighting and Opportunity Attacks.

## User Review Required

> [!IMPORTANT]
> **Saving Throw Logic**: We will distinguish between `check [ability]` (Ability Check) and `check [ability] save` (Saving Throw). An Ability Check only adds the ability modifier. A Saving Throw adds the proficiency bonus if the character or monster is proficient in that specific save. This requires the user to explicitly include the word "save" or "st" in their check command.
> **Two-Weapon Fighting**: This will be implemented as a bonus action attack. Per 5e rules, the positive ability modifier will be removed from the damage roll.

## Proposed Changes

### [Component] Mechanics Fixes (`internal/command/`)

#### [MODIFY] [check.go](file:///home/paulo/org/projetos/draconic/internal/command/check.go)

- Update `evalModifier` function:
  - Add `isSave bool` parameter.
  - Implement proficiency lookup for **Monsters**.
  - If `isSave` is true, prioritize `Saving Throw: [Ability]` in the actor's proficiencies.
- Update `ExecuteCheck`:
  - Check `state.PendingChecks` for the actor; if the requested check contains the word "save", pass `isSave: true` to `evalModifier`.

### [Component] Two-Weapon Fighting (`internal/parser/` & `internal/command/`)

#### [MODIFY] [lexer.go](file:///home/paulo/org/projetos/draconic/internal/parser/lexer.go) - New Keywords

- Add `bonus` and `offhand` to keywords.

#### [MODIFY] [ast.go](file:///home/paulo/org/projetos/draconic/internal/parser/ast.go) - Two-Weapon Fighting

- Add `Bonus bool` field to `AttackCmd`.
- Update grammar to: `Keyword: "attack" [Bonus: "bonus"] ...`.

#### [MODIFY] [attack.go](file:///home/paulo/org/projetos/draconic/internal/command/attack.go) - Two-Weapon Fighting

- In `ExecuteAttack`, check `ent.BonusActionsRemaining` if `cmd.Bonus` is true.
- Record `IsBonus` in the `AttackResolvedEvent`.

#### [MODIFY] [damage.go](file:///home/paulo/org/projetos/draconic/internal/command/damage.go) - Off-hand Damage

- In `ExecuteDamage`, if `state.PendingDamage.IsBonus` is true, modify the damage dice string to remove positive modifiers (e.g., `1d6+3` becomes `1d6`).

### Opportunity Attack Combat Action (`internal/parser/` & `internal/command/`)

#### [MODIFY] [ast.go](file:///home/paulo/org/projetos/draconic/internal/parser/ast.go) - Opportunity Attack

- Add `Reaction bool` field to `AttackCmd`.
- Update grammar to support `attack reaction ...`.

#### [MODIFY] [attack.go](file:///home/paulo/org/projetos/draconic/internal/command/attack.go) - Opportunity Attack

- In `ExecuteAttack`, if `cmd.Reaction` is true:
  - Check `ent.ReactionsRemaining`.
  - If `PendingAdjudication` is nil, trigger adjudication: `adjudicate "opportunity attack by: [Actor] against: [Target]"`.
  - If `PendingAdjudication` is approved, consume 1 reaction and proceed with the attack.

---

## Verification Plan

### Automated Tests

- Run `go test ./internal/command/...` to ensure no regressions.
- Create a new integration test in `internal/command/mechanics_test.go`:
    1. Test that `grapple` (which asks for a save) allows Elara and Thorne to use their proficiencies even when they just type `check dex` or `check str`.
    2. Test that `attack bonus with: Dagger` consumes a bonus action.
    3. Test that damage for a bonus attack correctly strips the modifier.
    4. Test that `attack reaction with: Longsword` consumes a reaction.

### Manual Verification

- In the TUI, verify that `attack bonus ...` correctly updates the state (`hint` should show 0 bonus actions remaining).
- Verify that a monster's saving throw (e.g. Aboleth) correctly includes its proficiency bonus when answering a check.
