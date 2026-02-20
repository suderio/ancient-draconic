# Phase 12: Ask and Check Commands Plan

## **1. Objective**

Implement the GM-controlled `ask` command, which requests an environmental or conditional check from players, and the `check` command, which players use to execute those requests. This pairing introduces Encounter Freezing and automated Condition/Damage state changes based on standard DC logic.

## **2. Command Syntax (Participle AST)**

* **`ask`**: `ask :by GM :check <skill|ability|save> :of <character|monster> [:and <character|monster>]* :dc <N> [:fails <condition|damage xdy>] [:succeeds <condition|damage xdy|half>]`
* **`check`**: `check :by <character|monster> <ability|skill|save>`

## **3. GameState Additions**

To orchestrate the blocking mechanism:

* **`PendingChecks map[string]*PendingCheckState`**: A map tracking which actors are actively being "asked" to roll. While this map is not empty, `GameState.IsFrozen()` returns true (barring specific check actions).
* **`PendingCheckState`**:
  * `CheckType`: string (e.g. `save`, `skill`, `ability`)
  * `CheckTarget`: string (e.g. `dexterity`, `athletics`)
  * `DC`: int
  * `Fails`: string (raw AST string for condition or damage)
  * `Succeeds`: string (raw AST string for condition, damage, or 'half')

## **4. Engine Execution Flow**

### **A. Ask Command**

1. **Validation**: Enforce `:by GM`.
2. **Setup**: Populate `GameState.PendingChecks` for every target in the `:of` list.
3. **Emission**: Emit an `EventAskIssued` which writes to the log and sets the GameState locks.

### **B. Check Command**

1. **Validation**: Check if `GameState.PendingChecks` contains the acting character.
2. **Logic Check**: Verify the player's check attempt matches what the GM requested (e.g., if asked for `athletics`, they can't type `check :by Paulo stealth`).
3. **Data Loading**:
    * Retrieve the entity (Character or Monster).
    * If `skill` (e.g. `athletics`), parse `data/skills/athletics.yaml` to figure out it uses `STR`, then check if the Character is "proficient" (Need to discuss this).
    * If `ability` (e.g. `str`), just read `STR` modifier.
    * If `save` (e.g. `str save`), resolve like ability and check "saving throw proficiencies".
4. **Resolution**: Roll `1d20 + TotalModifier`.
5. **Evaluation**: Matches or beats the DC from `PendingChecks`.
    * **Failure**: Parse the `Fails` condition. If it's pure damage (`damage 2d6`), roll it and emit `HPChangedEvent`. If it's a condition (`grappled`), emit a `ConditionAppliedEvent`.
    * **Success**: Parse the `Succeeds` condition. If it's `half` (and there is a `Fails` damage), roll damage and halve it before emitting `HPChangedEvent`.
6. **Cleanup**: Remove the actor from `PendingChecks`. If `PendingChecks` is empty, the encounter unfreezes.

## **5. Engine/Hint Updates**

* **`hint`**: Expanded logic to check `PendingChecks`. If populated, `hint` returns: `> Waiting for check of [Names...]`.

## **6. Resolution of Clarifications**

Following the GM's guidance, these rules formally encode into the implementation:

1. **Character Proficiencies**: `data/characters/*.yaml` natively carries a `proficiencies` slice. The `check` evaluator will loop through this struct and extract the matching `value` directly (which already includes the character's base stat + proficiency bonus). If the proficiency is *not* found in the array, the system falls back seamlessly to the pure base ability modifier calculation.
2. **Condition Mechanics**: Conditions (like `blinded`, `restrained`, `poisoned`) are not just floating strings; they hold mechanical weight. Because the YAML definitions (`data/conditions/*.yaml`) use human-readable text strings for their descriptors, the engine will natively map their mechanical impacts (advantage/disadvantage matrices) directly into the `Attack` and `Check` evaluation cycles via string-matching switches.
   * *Scope Expansion*: Implementing this requires augmenting `ExecuteAttack` and `ExecuteCheck` to cross-reference targets' `[]Conditions` arrays prior to parsing the final 1d20 rolls.
3. **Loose Checking Typology**: Both definitions (`dex` or `dexterity`, `str` or `strength`) correlate to identical structs. The parser and loader check loose matching prefixes automatically.
4. **Universal Availability**: `ask` and `check` freeze active encounters locally, but they are fully supported *outside* the encounter loop for traps, social encounters, or environment hazards without panicking the system.
