# Phase 10: Attack, Damage, and Turn Commands Plan

## **1. Objective**

Implement the core combat loop commands: `attack`, `damage`, and `turn`.
These commands are governed by strict state-based execution rules (Turn Order and Hit Success) to guarantee a consistent flow of combat.

## **2. Command Syntax (Participle AST)**

The syntax will be parsed gracefully, handling optional arguments without breaking grammar.

* **`attack`**: `attack by: <actor> with: <weapon> to: <target> [and: <target>]* [dice: <dice_expr>]`
* **`damage`**: `damage by: <actor> [with: <weapon>] [dice: <dice_expr>]`
* **`turn`**: `turn by: <actor>`

## **3. GameState Additions**

To seamlessly handle combat flow, the `GameState` must track temporal data.

* **`TurnOrder []string`**: An array containing the IDs of all entities, sorted by their `Initiative` score in descending order.
* **`IsEncounterFrozen bool`**: If any active entity in the encounter lacks an Initiative score, the encounter is "frozen" and all `attack`/`damage`/`turn` commands are silently ignored.
* **`CurrentTurnIndex int`**: Points to the actor whose turn it is currently in `TurnOrder`.
* **`PendingDamage *PendingDamageState`**: Records `Attacker`, `Targets` (an array of entity IDs), `Weapon`, and `HitStatus` (a map of target ID to boolean `IsHit`) after an `attack` command is resolved.
If an attack misses all targets, `PendingDamage` is cleared or flagged to ignore incoming damage commands.

## **4. Command Validations & Ignoring Logic**

The user requested that certain commands be completely ignored (no log, no acknowledgment) rather than throwing standard errors.

1. **Frozen Encounters**: If `IsEncounterFrozen` is true (because an actor hasn't rolled initiative yet), actions are silently ignored.
2. **Out-of-Turn Actions**: Any action like `attack` or `turn` sent by an actor who does **not** hold the `CurrentTurn` index will be rejected with `engine.ErrSilentIgnore`. Exceptions: The GM can force these commands (e.g., `turn by: GM`). If the `by:` clause is omitted in the REPL, it defaults to `by: GM`.
3. **Orphaned Damage**: The `damage` command will check `GameState.PendingDamage`. If it is nil, or no targets were hit, or the `Actor` does not match the `Attacker`, it returns `ErrSilentIgnore`.

## **5. Engine Execution Flow**

### **A. Turn Command**

* **Logic**: Verifies the actor owns the current turn (or is the GM). If so, increments `CurrentTurnIndex` to `(CurrentTurnIndex + 1) % len(TurnOrder)` and emits a `TurnEndedEvent`.

### **B. Attack Command**

* **Logic**:
 1. Validate encounter is not frozen and actor has the turn (or is GM).
 2. Parse the attacker's weapon. The loader retrieves the entity (Character or Monster). It deep parses the `actions` or equipment to find the attack bonus.
 3. For each target, parse their Armor Class. "Meets it, beats it": `Roll >= AC` is a hit.
 4. Perform the auto-calculated hit roll (1d20 + `attack_bonus`). If `dice:` is provided, **override** the default auto-roll entirely.
 5. Emit `AttackResolvedEvent` comprising `{HitStatus: map[string]bool}` modifying `PendingDamage`.

### **C. Damage Command**

* **Logic**:
 1. Validate `PendingDamage` is active.
 2. Map the active targets from the hit map.
 3. Parse the weapon from the `PendingDamage` state to extract the `damage_dice` and `damage_modifier` from the YAML. If `dice:` is provided, **override** the default damage roll entirely.
 4. Emit `HPChangedEvent` reducing the Targets' HP. Clear `PendingDamage`.
