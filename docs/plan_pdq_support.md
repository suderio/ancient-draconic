# Plan: Enhancing the Generic Engine for PDQ and D&D

This document outlines proposed engine upgrades to fully support the Prose Descriptive Qualities (PDQ) ruleset while simultaneously improving the experience for the existing D&D campaign.

## Current Limitations for PDQ

While the newly created `world/pdq-campaign/manifest.yaml` successfully encodes the 2d6 + Quality opposed roll mechanic, it currently has to hack around two critical PDQ rules:

1. **Damage Allocation**: In PDQ, the defender decides *which* of their Qualities takes the downshift (Failure/Damage Ranks). The current engine forces the manifest to blindly apply damage (presently mapped to a generic `HP` reduction).
2. **True Opposed Rolls**: While CEL can roll 2d6 twice in a single formula, this hides the active participation of the defender. In true tabletop, the attacker rolls and waits for the defender's response.

## Proposed Engine Improvements

### 1. `ChoiceIssuedEvent`

**Concept**: A new event type, similar to `AskIssuedEvent`, but instead of a simple Yes/No/Check, it presents the user with a list of valid choices to select from.

**PDQ Application**:
Instead of automatically dealing HP damage, the manifest emits:

```json
{
  "type": "ChoiceIssued",
  "actor_id": "defender",
  "prompt": "You took 2 Damage Ranks. Choose a Quality to Downshift:",
  "options": "actor.qualities",
  "resolves_with": "ApplyDamageRanks"
}
```

**D&D Applications**:

- Selecting which Spell Slot to expend when casting a spell. *(Note: Primarily handled as an optional argument to the `cast` command).*
- Choosing a Maneuver for a Battle Master fighter. *(Note: Handled as an argument to the attack command or its own separate command).*
- Selecting a specific weapon to draw from inventory, or a tool to use. *(Note: Can serve as a fallback if the `with:` argument is not provided).*
- Choosing whether to apply smite damage after seeing a successful hit. *(Note: Issued only when the actor has Smite, still has spell slots, and hits. This will be an elegant but complex integration).*

### 2. Contested / Opposed Actions

**Concept**: Expand `AskIssuedEvent` or create `ContestStartedEvent` that pauses execution, asks a target to make a specific `check`, and injects their roll result back into the original attacker's context.

**PDQ Application**:
The attacker declares an attack using `Biker_Dude`. The engine pauses and asks the defender: "Defend against Biker_Dude attack." The defender responds: `defend using Dirty_Fighter`. Both rolls are captured and compared in real-time.

**D&D Applications**:

- Grapple and Shove contests (Athletics vs Athletics/Acrobatics). Currently, the D&D manifest assumes a static DC (8 + STR mod) for grapple defense. A contested roll feature would allow the defender to actively roll Athletics or Acrobatics to resist. *(Note: This is a must do!)*
- Stealth vs active Perception. *(Note: This wouldn't be automatic, it should be asked by the GM).*

## Verification Plan

Should these changes be implemented:

1. **Unit Tests**: Modify `mechanics_test.go` to inject mock choices for `ChoiceIssuedEvent` to verify that state mutates based on user selection.
2. **Integration Tests**: Write an integration script that runs a PDQ conflict where an actor takes a Damage Rank and downshifts a specific chosen Quality.
