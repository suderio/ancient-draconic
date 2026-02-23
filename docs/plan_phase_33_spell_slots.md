# Phase 33: Spell Slot Management

This phase introduces tracking and consumption of spell slots for spellcasting entities.

## Proposed Changes

### [Component] Engine State

#### [MODIFY] [state.go](file:///home/paulo/org/projetos/draconic/internal/engine/state.go)

- Add `SpellSlots` field to `Entity` struct.
- `SpellSlots` will be a map from level (int) to a struct containing `Current` and `Max`.

```go
type SpellSlotState struct {
 Current int `json:"current"`
 Max     int `json:"max"`
}

type Entity struct {
    // ...
    SpellSlots map[int]*SpellSlotState `json:"spell_slots"`
}
```

### [Component] Parser

#### [MODIFY] [ast.go](file:///home/paulo/org/projetos/draconic/internal/parser/ast.go)

- Add `CastCmd` to the `Command` union.
- Define `CastCmd` struct with `Actor`, `SpellName`, and `Level`.

```go
type CastCmd struct {
 Actor *ActorExpr `parser:"'cast' ('by' @@)?"`
 Spell string     `parser:"@String"`
 Level int        `parser:"('at' 'level' @Int)?"`
}
```

### [Component] Engine Events

#### [MODIFY] [event.go](file:///home/paulo/org/projetos/draconic/internal/engine/event.go)

- Add `SpellCastEvent`.
- `Apply` logic will decrement the appropriate spell slot and consume an action if in combat.

### [Component] Commands

#### [NEW] [cast.go](file:///home/paulo/org/projetos/draconic/internal/command/cast.go)

- Implement `ExecuteCast`.
- Logic should verify slot availability and action economy.

### [Component] Data Loading

#### [MODIFY] [models.go](file:///home/paulo/org/projetos/draconic/internal/data/models.go)

- Add `SpellSlots` to `Character` and `Monster` schemas.

## Verification Plan

### Automated Tests

- Integration tests in `internal/command/spell_test.go`:
  - Cast a spell at base level (consumes slot).
  - Cast a spell at higher level (consumes slot).
  - Attempt to cast with no slots remaining (error).
  - Verify action economy consumption (uses an action).

### Manual Verification

- Testing via the REPL:
  - `cast by: Elara "Fireball" at level 3`
  - Observe `SpellCastEvent` and state update in the TUI.
