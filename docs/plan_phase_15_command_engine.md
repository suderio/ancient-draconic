# Phase 15: Clean-Slate Command Engine in `internal/manifest/`

## Goal

Build a new, self-contained command engine in `internal/manifest/` that is fully driven by a YAML manifest. No modifications to existing code in `internal/{engine,command,parser,session,rules,data}` — those packages are legacy and will be replaced over time.

## Design Decisions (Resolved)

| # | Question | Decision |
|---|----------|----------|
| 1 | Where to put new code? | `internal/manifest/` — clean slate |
| 2 | Variable interpolation | Pure CEL variables. `actor.proficiencies[command.skill]` for dynamic key access |
| 3 | `ask` semantics | Formula evaluates to a CEL list: `[target, option1, option2, ...]`. An `AskIssuedEvent` is emitted, freezing the game until `allow`/`deny` resolves it |
| 4 | Execution order | `prereq → game → targets(per target) → actor` |
| 5 | Multi-word commands | Parser sugar: `encounter_start` in manifest ↔ `encounter start` in input. Handled by the parser, not the engine |
| 6 | Entity model | Reuse the existing `Entity` field structure (ID, Name, Types, Classes, Stats, Resources, Spent, Conditions, Proficiencies, Statuses, Inventory) as the model for characters/monsters YAML files |
| 7 | Hardcoded commands | `roll`, `help`, `hint`, `ask`, `adjudicate`, `allow`, `deny` — exist in every game regardless of manifest |

---

## Manifest Structure

```yaml
restrictions:
  adjudication:
    commands: "grapple"           # comma-separated or list
  gm_commands: ["encounter_start", "encounter_end"]

commands:
  encounter_start:
    name: "encounter start"       # Display name
    params:                       # Parameter definitions
      - name: "with"
        type: "list<target>"
        required: false
    prereq:                       # Must all pass, or return error
      - name: "check_conflict"
        formula: "!is_encounter_active"
        error: "an encounter is already active"
    hint: "Roll initiative for all actors."
    help: "Starts an encounter."
    error: "encounter [by: GM] start [with: Target1 [and: Target2]*]"
    game: [...]                   # Steps run once
    targets: [...]                # Steps run per-target
    actor: [...]                  # Steps affecting the acting entity
```

---

## Proposed Files

All new files go in `internal/manifest/`.

### [NEW] `internal/manifest/types.go` — Data Model

Core types for the manifest, game state, entities, and events.

```go
// --- Manifest model ---
type ParamDef struct { Name, Type string; Required bool }
type PrereqStep struct { Name, Formula, Error string }
type GameStep struct { Name, Formula, Event string }
type CommandDef struct {
    Name    string
    Params  []ParamDef
    Prereq  []PrereqStep
    Hint, Help, Error string
    Game, Targets, Actor []GameStep
}
type Restrictions struct {
    Adjudication struct { Commands []string }
    GMCommands   []string `yaml:"gm_commands"`
}
type Manifest struct {
    Restrictions Restrictions
    Commands     map[string]CommandDef
}

// --- Entity model (actors, targets, characters, monsters) ---
type Entity struct {
    ID, Name      string
    Types         []string
    Classes       map[string]string
    Stats         map[string]int
    Resources     map[string]int
    Spent         map[string]int
    Conditions    []string
    Proficiencies map[string]int
    Statuses      map[string]string
    Inventory     map[string]int
}

// --- Game state ---
type Loop struct {
    Active    bool
    Actors    []string
    Order     map[string]int  // actor → sort key
    Ascending bool
    Current   int
}
type GameState struct {
    Entities map[string]*Entity
    Loops    map[string]*Loop  // e.g. "encounter" → Loop
    Metadata map[string]any    // arbitrary state
}

// --- Events ---
type Event interface {
    Type() string
    Apply(state *GameState) error
    Message() string
}
```

**Event types** (minimal set):

| Event | Fields | Purpose |
|-------|--------|---------|
| `LoopEvent` | `LoopName string`, `Active bool` | Start/stop a loop |
| `LoopOrderAscendingEvent` | `LoopName string`, `Ascending bool` | Set sort direction |
| `LoopOrderEvent` | `LoopName string`, `ActorID string`, `Value int` | Set actor's order key |
| `ActorAddedEvent` | `LoopName string`, `ActorID string` | Add actor to a loop |
| `AttributeChangedEvent` | `ActorID string`, `Section string`, `Key string`, `Value any` | Change any entity field (spent, stats, resources, etc.) |
| `ConditionEvent` | `ActorID string`, `Condition string`, `Add bool` | Add/remove a condition |
| `AskIssuedEvent` | `TargetID string`, `Options []string` | Request player input (freezes game) |
| `HintEvent` | `Message string` | Display-only message |
| `DiceRolledEvent` | `ActorID string`, `Dice string`, `Result int` | Record dice roll |
| `MetadataChangedEvent` | `Key string`, `Value any` | Change global metadata |

### [NEW] `internal/manifest/loader.go` — YAML Loading

- `LoadManifest(path string) (*Manifest, error)` — parse `manifest.yaml`
- `LoadEntity(path string) (*Entity, error)` — parse character/monster YAML

### [NEW] `internal/manifest/eval.go` — CEL Evaluation

- Thin wrapper around `cel-go` with game-specific functions (`roll`, `mod`, `has`, etc.)
- Builds the CEL context from `GameState`, `Entity` (actor/target), command params, and step results
- Dynamic key access works natively: `actor.proficiencies[command.skill]` is valid CEL when `proficiencies` is a `map[string]int`

### [NEW] `internal/manifest/executor.go` — Command Execution

The core execution pipeline:

```
ExecuteCommand(cmdName, actorID, targets, params, state, manifest) → []Event, error

1. Lookup: Find CommandDef in manifest (or handle hardcoded commands)
2. Restrictions: Check gm_commands, adjudication
3. Params: Validate against ParamDef (type, required)
4. Prereq: Evaluate each prereq formula → must all be true, else return error
5. Game: Evaluate game[] steps sequentially, collect events
6. Targets: For each target, evaluate targets[] steps, collect events
7. Actor: Evaluate actor[] steps, collect events for the acting entity
8. Return all collected events
```

Each step's CEL formula receives a context containing:

- `actor` — the acting Entity as a map
- `target` — current target Entity (in `targets` section) or nil
- `command` — the parsed params (e.g., `command.skill`, `command.with`)
- `steps` — results of previous steps in the same section (e.g., `steps.contest`)
- `is_<loop>_active` — boolean from each Loop
- `metadata` — global metadata map

### [NEW] `internal/manifest/hardcoded.go` — Built-in Commands

Handlers for the 7 hardcoded commands that exist regardless of manifest:

| Command | Behavior |
|---------|----------|
| `roll` | Evaluate dice expression, return `DiceRolledEvent` |
| `help` | Read `help` field from manifest command definitions |
| `hint` | Read `hint` field from the last executed command |
| `ask` | Emit `AskIssuedEvent` with target + options list, freeze game |
| `adjudicate` | GM reviews a pending ask/contest |
| `allow` | GM approves a pending action |
| `deny` | GM rejects a pending action |

### [NEW] `internal/manifest/executor_test.go` — Tests

Table-driven tests covering:

1. **`TestPrereqValidation`** — prereq failure returns correct error message
2. **`TestParamValidation`** — missing required param, wrong type
3. **`TestGMRestriction`** — non-GM actor on GM-only command
4. **`TestLoopLifecycle`** — start encounter, add actors, set order, end encounter
5. **`TestGameSteps`** — sequential step evaluation with `steps.previous_result`
6. **`TestTargetIteration`** — `targets` section runs per-target with `target` variable
7. **`TestActorSteps`** — actor section modifies the acting entity
8. **`TestAskIssuedEvent`** — formula returns list → `AskIssuedEvent` emitted
9. **`TestHelpCommand`** — returns help text from manifest
10. **`TestHintCommand`** — returns hint text from last command

---

## Manifest Improvement Suggestions

> [!TIP]
> These are suggestions for improving the `world/dnd5e/manifest.yaml` draft.

1. **`restrictions.adjudication.commands`** should be a list `["grapple"]` not a string — consistent with `gm_commands`
2. **`check` command** (line 120-122): the formula uses `$command.skill` and `$command.dc` — these should be `command.skill` and `command.dc` (CEL variables, no `$` prefix)
3. **`grapple.game.ask_grapple`**: the formula should be the CEL list form: `[target.id, 'check skill: athletics dc: ' + string(steps.contest), 'check skill: acrobatics dc: ' + string(steps.contest)]`
4. **`grapple.game.grappled`**: `formula: "Grappled"` is a bare string for `AddConditionEvent`. Consider making this more explicit: `formula: "'Grappled'"` (CEL string literal) or using a dedicated `condition` field in the step
5. **`grapple.actor.consume_action`**: `formula: "actions"` with `event: "AddSpentEvent"` is terse. Consider using `AttributeChangedEvent` with `formula: "{'section': 'spent', 'key': 'actions', 'value': actor.spent.actions + 1}"` for consistency, or keep `AddSpentEvent` as a shorthand that increments `spent[formula_result]` by 1
6. **Missing `encounter_start.targets` context**: The `targets` section references `$target` but the `encounter_start` command's `with` param is a `list<target>`. Clarify: `targets` section iterates over the `with` parameter? Or over all entities? **Recommendation**: iterate over the resolved target list from the command

---

## Verification Plan

### Automated Tests

All tests live in `internal/manifest/executor_test.go`. Run with:

```bash
go test -v ./internal/manifest/...
```

Each test uses a small inline manifest and mock entities — no dependency on external YAML files or the legacy codebase.

### Integration Test

After the core engine is working, create a test that loads the actual `world/dnd5e/manifest.yaml` and runs a scripted sequence:

```bash
go test -v -run TestDnD5eIntegration ./internal/manifest/...
```

### Manual Verification

Once integrated with the CLI (future phase), verify via:

```
draconic repl --world_dir ~/world --campaign_dir teste
> help
> help encounter
> encounter by: GM start with: Player1
> hint
> initiative
```

> [!NOTE]
> Manual verification requires CLI integration, which is a follow-up phase after the core engine is proven via automated tests.
