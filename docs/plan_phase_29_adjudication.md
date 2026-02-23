# Implementation Plan: Adjudication & Action Economy (v2)

Implement a GM-authorization system (Adjudication) and enforce 5e action economy (1 Action, 1 Bonus, 1 Reaction) along with new combat actions.

## User Review Required

> [!IMPORTANT]
> **Multiple Actions**: We will use `ActionsRemaining` instead of booleans to support features like Action Surge or Haste.
> **Dodge Mechanics**: `dodge` will be a standard action (non-adjudicated). It will grant the "Dodging" condition, which will be checked during attack and save resolutions.

## Proposed Changes

### [Component] Engine & State (`internal/engine/`)

#### [MODIFY] [state.go](file:///home/paulo/org/projetos/draconic/internal/engine/state.go)

- Update `GameState`:
  - Add `PendingAdjudication *PendingAdjudicationState`.
- Update `Entity`:
  - Add `ActionsRemaining int`, `BonusActionsRemaining int`, `ReactionsRemaining int`.
  - Add `AttacksRemaining int`.
- Update `IsFrozen()` to include `PendingAdjudication != nil`.

#### [MODIFY] [events.go](file:///home/paulo/org/projetos/draconic/internal/engine/events.go)

- Add `AdjudicationStartedEvent`: Stores the original command that requires GM approval.
- Add `AdjudicationResolvedEvent`: Records if a command was `Allowed` or `Denied`.
- Add `DodgeEvent`: Simple event to mark that an actor took the Dodge action.
- Add `GrappleEvent`: Triggers adjudication-style flow or records result after GM approval.

#### [MODIFY] [projector.go](file:///home/paulo/org/projetos/draconic/internal/engine/projector.go)

- `TurnStartedEvent`: Resets counts (`ActionsRemaining=1`, `BonusActionsRemaining=1`, `ReactionsRemaining=1`, `AttacksRemaining=1` or based on stat block).
- `DodgeEvent`: Adds the "Dodging" condition to the actor.
- `TurnStartedEvent`: Removes the "Dodging" condition from the actor whose turn is starting.

---

### [Component] Parser (`internal/parser/`)

#### [MODIFY] [ast.go](file:///home/paulo/org/projetos/draconic/internal/parser/ast.go)

- Add `AdjudicateCmd`, `AllowCmd`, `DenyCmd`, `DodgeCmd`, `GrappleCmd` to parsing rules.

---

### [Component] Session & Commands (`internal/session/` & `internal/command/`)

#### [MODIFY] [session.go](file:///home/paulo/org/projetos/draconic/internal/session/session.go)

- Update `Execute` to block commands if `IsFrozen`, unless it's a "control" command from the GM.

#### [NEW] [dodge.go](file:///home/paulo/org/projetos/draconic/internal/command/dodge.go)

- **Implementation**: Records a `DodgeEvent`.
- **Consequences**:
  - Attack Logic: In `ExecuteAttack`, if the target has "Dodging", the roll is forced to Disadvantage.
  - Save Logic: In `ExecuteCheck` (or similar for saves), if the actor is making a DEX save and has "Dodging", the roll is forced to Advantage.

#### [MODIFY] [attack.go](file:///home/paulo/org/projetos/draconic/internal/command/attack.go)

- Inject logic to check for "Dodging" condition on targets.

---

## Verification Plan

### Automated Tests

- `integration_test.go`:
  - Verify `dodge` grants "Dodging" and it disappears at start of next turn.
  - Verify `attack` against a "Dodging" target uses disadvantage.
  - Verify `grapple` freezes the system until `allow/deny`.
  - Verify Action Surge style support (multiple actions).

### Manual Verification

- TUI: Verify `hint` shows "Waiting for GM adjudication" when frozen.
