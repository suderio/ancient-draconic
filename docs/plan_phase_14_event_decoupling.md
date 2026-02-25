# Goal Description

The final structural layer of hardcoded D&D logic resides in three specific event structs located in `internal/engine/event.go`: `HPChangedEvent`, `AttackResolvedEvent`, and `InitiativeRolledEvent`.
These events hide D&D math inside their Go `Apply()` methods (e.g., capping healing at `Resources["hp"]`, artificially setting a `pending_damage` metadata structure for attacks, and numerically sorting an `initiatives` map descending).

This phase proposes completely eradicating these structs. The game logic will exclusively execute natively via `manifest.yaml` CEL rulesets, and state mutations will be persisted through purely generic alternative events (e.g., `AttributeChangedEvent` or `MetadataChangedEvent`).

## User Review Required

> [!WARNING]
> Eradicating these three events means we **must** delete them from the persistence layer's `store.go` unmarshaling cases. All these changes, until we reach a stable version, are backward incompatible. We must ensure tests delete old campaign logs to avoid errors.

## Proposed Changes

### **1. Event Layer Deprecation**

#### [MODIFY] [event.go](file:///home/paulo/org/projetos/dndsl/internal/engine/event.go)

- Delete `HPChangedEvent`. (Replaced by `AttributeChangedEvent`).
- Delete `AttackResolvedEvent`. (Replaced by `MetadataChangedEvent` and `HintEvent`).
- Delete `InitiativeRolledEvent`.
- Add a new generic `TurnOrderUpdatedEvent` that accepts `TurnOrder []string` and simply replaces the global `state.TurnOrder`.

---

### **2. Command Dispatcher Optimization**

#### [MODIFY] [executor.go](file:///home/paulo/org/projetos/dndsl/internal/command/executor.go)

- Remove the `switch e.Type` specific routing blocks handling `"HPChanged"`, `"AttackResolved"`, and `"InitiativeRolled"` strings when parsing manifest outputs.

---

### **3. Persistence Layer Updates**

#### [MODIFY] [store.go](file:///home/paulo/org/projetos/dndsl/internal/persistence/store.go)

- Remove `engine.EventHPChanged`, `engine.EventAttackResolved`, and `engine.EventInitiativeRolled` mappings from the event serialization loader.

---

### **4. Generic Manifest Implementations**

#### [MODIFY] [dnd manifest.yaml](file:///home/paulo/org/projetos/dndsl/world/dnd-campaign/manifest.yaml) & [pdq manifest.yaml](file:///home/paulo/org/projetos/dndsl/world/pdq-campaign/manifest.yaml)

- **Attack Rule Modifications**:
  Instead of declaring `Event: AttackResolved`, the attack mechanics must utilize CEL to evaluate the boolean hits, write the results to a structured `pending_damage` map, and emit `Event: MetadataChanged`.
- **Damage/Healing Rule Modifications**:
  Instead of declaring `Event: HPChanged` and relying on the Go backend to floor at 0 and cap at MaxHP, the CEL must do the capping natively: `math.min(actor.spent.hp + heal_amt, actor.resources.hp)` and then emit `Event: AttributeChanged`.
- **Initiative Rule Modifications**:
  Currently, `ExecuteGenericCommand` manually calculates strings arrays from `state.Metadata["initiatives"]`. The CEL math must sort the map, emit the `TurnOrderUpdatedEvent`, and inject the string array. (Due to CEL constraints for complex sorting, we might introduce a helper function `sort_initiatives(map[string]int) []string` into the `Registry`).

---

### **5. Test Fixes**

#### [MODIFY] [Command Suite](file:///home/paulo/org/projetos/dndsl/internal/command/)

- Update all integration tests (e.g., `mechanics_test.go`, `cel_attack_test.go`, and `store_test.go`) to expect `AttributeChangedEvent` arrays rather than `HPChangedEvent`.

## Verification Plan

### Automated Tests

- Run `go test ./internal/...` after every modification. We must specifically verify that `TestExecuteDamage` correctly caps hit points using `AttributeChangedEvent` via the manifest execution, and that the CEL environment natively handles `sort_initiatives()`.

### Manual Verification

- Launch the REPL with `go run main.go repl world/dnd-campaign/`.
- Verify a standard `attack` -> `damage` pipeline works successfully, showing the TUI updating properly based on `MetadataChangedEvent` outputs.
