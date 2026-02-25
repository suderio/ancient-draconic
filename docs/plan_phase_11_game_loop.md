# Phase 11: Game Loop & Resource Agnosticism Implementation Plan

This phase aims to remove all hardcoded D&D-specific assumptions regarding combat rounds, turn duration, action economy, and specific conditions (like `Dodging` or `Helped`) from the core Go engine. These will be shifted completely to the YAML manifest and generic events.

## Current State Analysis

- **Turn Reset Logic**: `TurnChangedEvent.Apply` currently has hardcoded logic to remove `Dodging` and `Disengaged` conditions and expire `Help` conditions. It also blindly resets all `Spent` resources, which assumes all resources (e.g., actions) regenerate fully on every turn. In reality, some resources do not regenerate every round or turn.
- **Specific Events**: The engine defines strongly typed D&D events like `DodgeTakenEvent`, `HelpTakenEvent`, `ActionConsumedEvent`, and `GrappleTakenEvent`.
- **Game Loop Constraints**: The concepts of a "Round" (where everyone has acted) and a "Turn" (a single actor's segment) are implicit. The loop lacks a generic way to define the order of actors (Initiative) when starting, relying on hardcoding.

## Proposed Changes

### 1. Generic Event Refactoring (`internal/engine/event.go`)

Refactor specific D&D events into versatile, genre-agnostic events:

#### [DELETE] `DodgeTakenEvent`, `ActionConsumedEvent`, `HelpTakenEvent`, `GrappleTakenEvent`

These are no longer necessary. They will be replaced by `AttributeChangedEvent` (for resources) and `ConditionAppliedEvent` (for conditions) emitted directly by the manifest.

#### [MODIFY] `TurnChangedEvent.Apply`

Remove the hardcoded Go logic that deletes `"Dodging"` and `"Disengaged"`.
Remove the hardcoded logic that expires `"HelpedCheck:"` and `"HelpedAttack:"`.
Instead, we will rely on a new generic "End of Turn" and "Start of Turn" trigger mechanism orchestrated by the manifest.

### 2. Manifest Triggers & Scripts (`data/manifest.yaml`)

To transfer game loops and conditions to the manifest, we need a way to run logic when a turn starts or a round ends.

#### Define Built-in Loop Commands

We will update `manifest.yaml` to include commands that can be invoked during turn transitions or explicitly by the encounter system, allowing the loop to be explicitly defined:

- **`initiative`**: The method by which the encounter determines the order of turns for participants.
- **`start_turn`**: Invoked automatically when a turn changes to a new actor. This will explicitly declare which generic resources are recovered (e.g., `spent.actions`, `spent.bonus_actions`, `spent.movement`), as opposed to blindly resetting all `spent` trackers.
- **`end_turn`**: Invoked automatically before the turn changes. This will handle expiring conditions that last "until end of next turn" or "until start of next turn" and transition the state towards the next actor.
- **`end_round`**: Invoked when the `TurnOrder` cycles back to the first actor, allowing mechanics that trigger "once per round" or "at the end of the round" (the 6s period where everyone has acted).

#### Updating Existing Manifest Commands

- **`dodge`**: Update to use `ConditionApplied` with a value of `Dodging`.
- **`help_action`**: Update to use `ConditionApplied` with a value like `HelpedCheck:HelperID`.
- **`attack` / `check` etc.**: Update the `actionConsumed` step from the legacy `ActionConsumedEvent` string to an `AttributeChanged` event that increments `actor.spent.actions`.

### 3. Tracking Condition Durations (`internal/engine/state.go`)

Right now, `Entity.Conditions` is just a `[]string`.
To support durations accurately without hardcoding names, we will use the `Metadata` map to track the specifics of conditions, e.g.:

```go
// Metadata tracking
"conditions_expiry": map[string]any{
  "TargetID:Dodging": map[string]string{ "expires_on": "start_turn", "reference_actor": "TargetID" },
  "HelperID:HelpedCheck:TargetID": map[string]string{ "expires_on": "end_turn", "reference_actor": "TargetID" }
}
```

During the `start_turn` and `end_turn` manifest evaluations, the generic engine will check if any conditions tied to the *current* actor should expire, emitting `ConditionRemovedEvent`s accordingly. This properly models effects that end on the *target's* next turn versus the *caster's* next turn.

### 4. Engine Hook for Turn Loop (`internal/command/executor.go` & `session.go`)

Modify `session.go`'s `Execute` and `executor.go`'s `ExecuteGenericCommand`.
When the `turn` command is parsed, it triggers the `turn` command in the manifest. The manifest's `turn` command will now explicitly fire steps that handle resource resets and condition expirations.

## Verification Plan

1. **Automated Tests**:
   - `mechanics_test.go`: Ensure that when an actor takes the Dodge action, the `Dodging` condition is applied, attacks against them have Disadvantage, and critically, the condition is removed when their *next* turn starts (or ends, per 5e rules).
   - `recharge_test.go`: Verify that cooldowns and recharges still function correctly after removing the D&D-specific turn logic.
   - Assert that `actions` are consumed properly and prevent double-attacks without using `ActionConsumedEvent`.
2. **Integration Verification**:
   - Run existing `RunCombatScript` scripts to verify encounter flow and turn rotation remain intact, and characters regain actions appropriately via the manifest scripts.
