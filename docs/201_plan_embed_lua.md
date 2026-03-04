# Development Plan: Lua Integration, YAML Migration, and CEL-go Removal

## 1. Objective

Replace the static YAML + CEL-go evaluation engine with a unified **GopherLua** environment. This transition will support "Environment Capture" (no-return files), dynamic logic, and the removal of the CEL-go dependency to streamline the `antigravity` engine.

---

## 2. Phase I: GopherLua Sandbox & Environment Capture

### 2.1 The "No-Return" Loader

* **Logic:** Use `L.LoadFile` combined with `L.SetFEnv` to capture top-level assignments.
* **Implementation:** Execute scripts within a private `LTable` to capture variables (e.g., `hp = 10`) as object properties.

### 2.2 Sandboxing & Global Bridge

* **Restricted Libs:** Load only `base`, `table`, `string`, and `math`.
* **Go-Lua Bridge:** Expose `roll(dice)` and `help(topic)` to the Lua global scope.
* **CEL-go Replacement:** Any logic previously handled by CEL (e.g., `hp: player.level * 10`) must now be written as standard Lua code within the `.lua` files.

---

## 3. Phase II: Recursive YAML-to-Lua Transpiler

### 3.1 Transpiler Logic (`--luafy`)

* **CLI Flag:** Add `--luafy <directory>` to the `init` command.
* **Recursive Conversion:** * Walk the directory for `.yaml` files.
* Convert YAML keys to flat Lua assignments: `key = value`.
* **Formula Handling:** If a YAML value was previously a CEL expression string, the transpiler should write it as a raw Lua expression (ensure strings are quoted, but numbers/logic remain bare).

---

## 4. Phase III: Dependency Cleanup (The "Anti-Gravity" Polish)

### 4.1 CEL-go Removal

* **Code Audit:** Identify all imports of `github.com/google/cel-go`.
* **Refactoring:** * Remove the `eval` or `expression` packages that utilized CEL.
* Replace internal calls to `cel.Program.Eval()` with `L.PCall()` or direct Lua table lookups.

* **Go Mod Cleanup:** Run `go mod tidy` to fully remove CEL-go and its heavy transitive dependencies from `go.mod` and `go.sum`.

---

## 5. Phase IV: Manifest & Data Migration

### 5.1 System Configuration

* Convert `manifest.yaml` to `manifest.lua`.
* Ensure the Go `Config` struct is populated by the captured Lua environment during the engine startup.

---

## 6. Technical Requirements & Constraints

* **No CGO:** Stick to pure Go `GopherLua`.
* **Error Reporting:** Provide clear Lua syntax error messages (with line numbers) to replace CEL's evaluation errors.
* **Performance:** Lua execution is generally faster than CEL interpretation for complex logic, but ensure the `LState` is pooled or reused where appropriate.

---

### Comparison of the Logic Shift

| Feature | Old System (YAML + CEL) | New System (Captured Lua) |
| --- | --- | --- |
| **Storage** | `monster.yaml` | `monster.lua` |
| **Logic** | `damage: "roll('1d6') + 2"` (as string) | `damage = roll("1d6") + 2` (as code) |
| **Engine** | `cel-go` + `yaml.v3` | `GopherLua` only |
| **Boilerplate** | High (String parsing) | Minimal (Native execution) |

## 7. Example of manifest.md to manifest.lua transformation

```yaml
restrictions:
  adjudication:
    commands: ["grapple"]
  gm_commands: ["encounter_start", "encounter_end"] # Sends the message 'unauthorized: this command can only be executed by the GM'

# variables are a way of creating vectors and maps for common transformations
# they can be used as a variable or a function:
# sizes[1] returns small
# sizes('small') returns 1
variables:
  - name: sizes
    value: ['tiny', 'small', 'medium', 'large', 'huge', 'gargantuan']
  - name: modifiers
    value: [-6, -5, -4, -4, -3, -3, -2, -2, -1, -1, 0, 0, 1, 1, 2, 2, 3, 3, 4, 4, 5, 5, 6, 6, 7, 7, 8, 8, 9, 9, 10]
  - name: skill_to_ability
    value: {'athletics': 'str', 'acrobatics': 'dex'}

commands:
  encounter_start: # Commands separated by '_' can be used as a_b by: xxx or a by: xxx b.
    name: "encounter start"
    params:
    # Until now we are processing commands with no definition of how they should be written in the manifest.
    # That means the parameters are hardcoded in the executor.
    # The by parameter is implicit. It is always the actor that issues the command.
      - name: "with"
        type: "list<target>" # The list<...> type means we can do with: xxx and: yyy and: zzz
        required: false
    prereq: # Every prereq formula must return true, or the error message is returned.
      - name: "check_conflict"
        formula: "!is_encounter_active" # Every LoopEvent creates a GameState variable 'is_<command_name>_active'.
        error: "an encounter is already active. End it first'" 
    hint: "Encounter has started. Roll initiative for all actors." # Message to show if someone asks for hint after the command encounter is executed.
    help: "Encounter start command starts an encounter." # Message to show with the help encounter command.
    error: "encounter start [with: Target1 [and: Target2]*]" # The message to show when the command is invalid. "The command must be: ..."
    game:
      # Loop is a new concept. Loops are made of actors that perform some commands in some sequence.
      # The loop has a list of actors that are filled with the ActorAddedEvent.
      # Each actor has a number (initially, the order they were added) that shows when they will act.
      - name: "create_loop"
        formula: "true"
        event: "LoopEvent" # LoopEvent begins or ends a 'loop' based on the value of formula (true/false). Changes the value of the variable is_<command_name>_active.
      - name: "order_loop"
        formula: "false"
        event: "LoopOrderAscendingEvent" # LoopOrderAscendingEvent sets the way to order the actors.
      - name: "add_actor"
        formula: "command.with" # Processing of commands must return a list everytime a command is defined with argument 'something: xxx and: yyy and: zzz'
        event: "ActorAddedEvent" # This event adds the list 'command.with' to actors of the loop. Actor is an engine term like target.
    targets:
      - name: "Ask Initiative"
        formula: "[target, 'initiative']" # formulas in the 'targets' can use the target variable to indicate each target of the command (not the entire list)
        event: "AskIssuedEvent"

  encounter_end:
    name: "encounter end"
    prereq:
      - name: "check_conflict"
        formula: "is_encounter_active" # Every LoopEvent creates a GameState variable 'is_<command_name>_active'.
        error: "no active encounter to end"
    hint: "Encounter has ended." # Message to show if someone asks for hint after the command encounter is executed.
    help: "Encounter end command ends an encounter." # Message to show with the help encounter command.
    error: "encounter end" # The message to show when the command is invalid. "The command must be: ..."
    game:
      - name: "state_change"
        formula: "false" # State changed means some variable changes value.
        loop: "encounter_start"
        event: "LoopEvent" # LoopEvent begins or ends a 'loop' based on the value of formula (true/false). Changes the value of the variable is_<command_name>_active.

  initiative:
    name: "initiative"
    prereq:
      - name: "check_active"
        formula: "is_encounter_active" # Every LoopEvent creates a GameState variable 'is_<command_name>_active'.
        error: "an encounter is not active. Start it first'" 
    hint: "Is it your turn? Wait for your turn" # Message to show if someone asks for hint after the command encounter is executed.
    help: "Initiative command rolls initiative for the actors." # Message to show with the help encounter command.
    error: "initiative" # The message to show when the command is invalid. "The command must be: ..."
    game:
      - name: "roll_score"
        formula: "roll('1d20') + mod(actor.stats.dex)"
        event: LoopOrderEvent # LoopOrderEvent changes the number associated with the actor in the Loop. Since the LoopOrderAscendintEvent is false, the one whit the biggest initiative will go first.

  grapple:
    name: "grapple"
    params:
      - name: "to"
        type: "target"
        required: true
    prereq:
      - name: "check_action"
        formula: "actor.spent.actions < actor.resources.actions"
        error: "no actions remaining"
    hint: "Grapple command grapples the target."
    help: "Grapple command grapples the target."
    error: "grapple [to: <target>]"
    game:
      - name: "contest"
        formula: "roll('1d20') + mod(actor.stats.str) + ('athletics' in actor.proficiencies ? actor.proficiencies.athletics * actor.stats.prof_bonus : 0)"
        event: "ContestStarted"
      - name: "ask_grapple"
        formula: "[target, 'check skill: athletics dc: ' + string(steps.contest), 'check skill: acrobatics dc: ' + string(steps.contest)]" # todo: steps -> game
        event: "AskIssuedEvent"
      - name: "resolve_grapple"
        formula: "targets.ask_grapple" # ask returns the result of the check, which is true if the check was successful.
        event: "ContestResolvedEvent" # No other step is processed if this one returns true.
      - name: "grappled"
        formula: "'grappled'"
        event: "AddConditionEvent"
    actor:
      - name: "consume_action"
        formula: "actions"
        event: "AddSpentEvent"
  
  check:
    name: "check"
    params:
      - name: "type"
        type: "string"
        values: ['skill', 'ability', 'save']
        required: true
      - name: "name"
        type: "string"
        required: true
      - name: "dc"
        type: "int"
        required: true
    hint: "Check command checks the target."
    help: "Check command checks the target."
    error: "check [type: <skill|ability|save>] [name: <name>] [dc: <dc>]"
    game:
      - name: "contest"
        formula: "roll('1d20') + mod[actor.stats[skill_to_ability[[command.skill]]]] + ('command.skill' in actor.proficiencies ? actor.proficiencies.command.skill * actor.stats.prof_bonus : 0) >= command.dc"
        event: "CheckEvent"

```

```lua
-- Restrictions: defines command access and GM-only permissions
restrictions = {
  adjudication = {
    commands = {"grapple"}
  },
  gm_commands = {"encounter_start", "encounter_end"}
}

-- Tabelas locais (privadas) para servir de base de dados
local _sizes_list = {'tiny', 'small', 'medium', 'large', 'huge', 'gargantuan'}
local _mod_table = {-6, -5, -4, -4, -3, -3, -2, -2, -1, -1, 0, 0, 1, 1, 2, 2, 3, 3, 4, 4, 5, 5, 6, 6, 7, 7, 8, 8, 9, 9, 10}
local _skill_map = { athletics = 'str', acrobatics = 'dex', stealth = 'dex', sleight_of_hand = 'dex', perception = 'wis', insight = 'wis', survival = 'wis', investigation = 'int', arcana = 'int', history = 'int', religion = 'int', nature = 'int', medicine = 'wis', persuasion = 'cha', intimidation = 'cha', performance = 'cha', deception = 'cha' }

--- Calcula o modificador a partir de um valor de habilidade
--- Não precisamos mais da tabela _mod_table
-- @param val número (ex: 10)
-- @return número (ex: 0)
function mod(val)
    return math.floor(val / 2) - 5
end

--- Converte nome de tamanho em número ou vice-versa
-- @param input string ("small") ou número (1)
function sizes(input)
    for i, name in ipairs(_sizes_list) do
        if name == input then return i end
    end
    return 0
end

--- Mapeia uma perícia para seu atributo base
-- @param skill string (ex: "athletics")
function skill_to_ability(skill)
    return _skill_map[skill] or 'str' -- 'str' como fallback
end

-- Commands: the core logic of the engine
commands = {
  encounter_start = {
    name = "encounter start",
    params = {
      { name = "with", type = "list<target>", required = false }
    },
    prereq = {
      { 
        name = "check_conflict", 
        formula = not is_encounter_active, 
        error = "an encounter is already active. End it first" 
      }
    },
    hint = "Encounter has started. Roll initiative for all actors.",
    help = "Encounter start command starts an encounter.",
    error = "encounter start [with: Target1 [and: Target2]*]",
    game = {
      { name = "create_loop", formula = true, event = "LoopEvent" },
      { name = "order_loop", formula = false, event = "LoopOrderAscendingEvent" },
      { name = "add_actor", formula = command.with, event = "ActorAddedEvent" }
    },
    targets = {
      { name = "Ask Initiative", formula = {target, "initiative"}, event = "AskIssuedEvent" }
    }
  },

  encounter_end = {
    name = "encounter end",
    prereq = {
      { 
        name = "check_conflict", 
        formula = is_encounter_active, 
        error = "no active encounter to end" 
      }
    },
    hint = "Encounter has ended.",
    help = "Encounter end command ends an encounter.",
    error = "encounter end",
    game = {
      { 
        name = "state_change", 
        formula = false, 
        loop = "encounter_start", 
        event = "LoopEvent" 
      }
    }
  },

  initiative = {
    name = "initiative",
    prereq = {
      { 
        name = "check_active", 
        formula = is_encounter_active, 
        error = "an encounter is not active. Start it first" 
      }
    },
    hint = "Is it your turn? Wait for your turn",
    help = "Initiative command rolls initiative for the actors.",
    error = "initiative",
    game = {
      { 
        name = "roll_score", 
        formula = roll("1d20") + mod(actor.stats.dex), 
        event = "LoopOrderEvent" 
      }
    }
  },

  grapple = {
    name = "grapple",
    params = {
      { name = "to", type = "target", required = true }
    },
    prereq = {
      { 
        name = "check_action", 
        formula = actor.spent.actions < actor.resources.actions, 
        error = "no actions remaining" 
      }
    },
    hint = "Grapple command grapples the target.",
    help = "Grapple command grapples the target.",
    error = "grapple [to: <target>]",
    game = {
      { 
        name = "contest", 
        formula = roll("1d20") + mod(actor.stats.str) + (actor.proficiencies.athletics and (actor.proficiencies.athletics * actor.stats.prof_bonus) or 0), 
        event = "ContestStarted" 
      },
      { 
        name = "ask_grapple", 
        formula = {target, "check skill: athletics dc: " .. tostring(game.contest), "check skill: acrobatics dc: " .. tostring(game.contest)}, 
        event = "AskIssuedEvent" 
      },
      { 
        name = "resolve_grapple", 
        formula = targets.ask_grapple, 
        event = "ContestResolvedEvent" 
      },
      { 
        name = "grappled", 
        formula = "grappled", 
        event = "AddConditionEvent" 
      }
    },
    actor = {
      { name = "consume_action", formula = "actions", event = "AddSpentEvent" }
    }
  },

  check = {
    name = "check",
    params = {
      { name = "skill", type = "string", required = true },
      { name = "dc", type = "int", required = true }
    },
    hint = "Check command checks the target.",
    help = "Check command checks the target.",
    error = "check [skill: <skill>] [dc: <dc>]",
    game = {
      { 
        name = "contest", 
        formula = roll("1d20") + mod(actor.stats[skill_to_ability(command.skill)]) + ((actor.proficiencies[command.skill] * actor.stats.prof_bonus) or 0) >= command.dc, 
        event = "CheckEvent" 
      }
    }
  }
}
```

---

## 8. Implementation Audit — Problems, Gaps, and Decisions

### 8.1 Critical: Eager vs Deferred Evaluation

The Lua manifest example has a **fatal semantic error**: all `formula` values are written as bare Lua expressions, which means they are evaluated **at file-load time** rather than **at command-execution time**.

```lua
-- THIS IS WRONG: evaluates `not is_encounter_active` when the file is loaded,
-- before any game state exists. `is_encounter_active` is nil → `not nil` is `true`
-- → the prereq becomes the constant `true` forever.
prereq = {
  { name = "check_conflict", formula = not is_encounter_active, error = "..." }
}

-- SAME PROBLEM: `roll("1d20") + mod(actor.stats.dex)` executes immediately.
-- `actor` is nil at load time → runtime crash.
game = {
  { name = "roll_score", formula = roll("1d20") + mod(actor.stats.dex), event = "LoopOrderEvent" }
}
```

#### Decision Required: Deferred Evaluation Strategy

There are two viable approaches:

##### **Option A — Formulas as strings (minimal Lua change)**

Keep `formula` values as strings that the Go engine compiles and evaluates on demand, exactly like the current CEL approach but using `L.DoString()`:

```lua
prereq = {
  { name = "check_conflict", formula = "not is_encounter_active", error = "..." }
}
game = {
  { name = "roll_score", formula = "roll('1d20') + mod(actor.stats.dex)", event = "LoopOrderEvent" }
}
```

* Pros: Simplest migration from CEL. The executor logic stays almost identical.
* Cons: Still evaluating strings; loses some of the "native execution" benefit pitched in the plan.

##### **Option B — Formulas as closures (idiomatic Lua)**

Wrap formulas in anonymous functions that capture the environment at call time:

```lua
prereq = {
  { name = "check_conflict", formula = function() return not is_encounter_active end, error = "..." }
}
game = {
  { name = "roll_score", formula = function() return roll("1d20") + mod(actor.stats.dex) end, event = "LoopOrderEvent" }
}
```

* Pros: True native Lua; type-safe; IDE/linter support; no string compilation overhead.
* Cons: More verbose; requires the Go engine to call `L.CallByParam()` instead of `L.DoString()`; the Go executor must inject `actor`, `target`, `command`, `steps`, and `is_*_active` into the Lua global scope _before_ each closure call.

> [!IMPORTANT]
> The plan MUST choose one of these two options before implementation begins. The entire engine bridge and executor pipeline depend on this choice.

### 8.2 Syntax Errors in the Lua Example

The `check` command at line 372 has an unbalanced parenthesis:

```lua
-- BROKEN: extra `)` after `or 0`
formula = roll("1d20") + mod(actor.stats[skill_to_ability(proficiencies[command.skill])]) + (actor.proficiencies[command.skill] * actor.stats.prof_bonus) or 0) >= command.dc
--                                                                                                                                                           ^
-- Also: `proficiencies[command.skill]` should be `command.skill` (the skill name string)
```

Additionally, the `check` command YAML has `type`, `name`, and `dc` params, but the Lua version changed `type` and `name` to just `skill` and `dc` without noting this is a deliberate simplification.

### 8.3 Gap: The `game.contest` / `steps.contest` Reference

The grapple command uses `game.contest` and `targets.ask_grapple`:

```lua
formula = {target, "check skill: athletics dc: " .. tostring(game.contest), ...}
-- and later:
formula = targets.ask_grapple
```

In the current Go engine, `steps` (not `game` or `targets`) is the map that accumulates results from previous steps in the same phase. The Lua example uses `game.contest` which implies:

1. A new naming convention (`game.*` for game-phase results, `targets.*` for target-phase results).
2. Or a misunderstanding — only `steps` exists in the current engine, and it's a flat namespace across all phases.

> [!WARNING]
> The plan must clarify the result-accumulation semantics. Currently `BuildContext` in `eval.go` puts all prior step results into a single `steps` map. If Lua introduces phase-scoped namespaces (`game`, `targets`, `actor`), the Go executor needs corresponding changes.

### 8.4 Gap: Go ↔ Lua Bridge for Entity Data

The current engine uses `BuildContext()` → `entityToMap()` to convert Entity structs to `map[string]any` for CEL. With GopherLua, we need to:

1. Convert `*Entity` → `*lua.LTable` (nested: `actor.stats.str`, `actor.proficiencies.athletics`).
2. Inject `actor`, `target`, `command`, `steps`, `metadata`, and all `is_*_active` flags into the Lua environment before each formula evaluation.
3. Extract results from Lua (`lua.LValue`) back to Go `any` for event construction.

This is approximately equivalent to `entityToMap` + `convertRefVal` but for Lua types (`LNumber`, `LString`, `LBool`, `LTable`, `LNil`).

### 8.5 Gap: Event Construction from Lua Results

The current `mapStepToEvent()` in `executor.go` (160 lines) handles 12 event types, each with specific field expectations. Example:

* `LoopEvent` expects a `bool` result
* `LoopOrderEvent` expects an `int` result
* `AskIssuedEvent` expects a `[]any{targetID, options...}` result
* `ActorAddedEvent` expects a `[]string` result

With Lua, the `LValue` returned by formula execution must be coerced to the correct Go type for each event. This coercion layer is not mentioned in the plan.

### 8.6 Gap: `LState` Lifecycle and Pooling

The plan mentions "ensure the `LState` is pooled or reused." Specifics:

* **One `LState` per Session**: Load `manifest.lua` once at session start. Formulas run in the same `LState`, with `actor`/`target`/`command`/`steps` globals reset before each formula call.
* **Thread safety**: If the Telegram bot calls `Session.Execute()` concurrently with the TUI, the `LState` is not thread-safe. Options: (a) a `sync.Mutex` around all Lua calls, or (b) an `LState` pool with `sync.Pool`.

### 8.7 Gap: The `--luafy` Transpiler Scope

Phase II proposes a `--luafy` CLI flag, but:

1. The plan says it goes on the `init` command, but the CLI has no `init` command — only `campaign create`. Should this be a standalone `draconic convert --luafy <dir>` command?
2. Entity files (`*.yaml`) have nested structures (stats, resources, proficiencies). The transpiler must handle nested maps → Lua tables, not just "flat assignments."
3. Should the transpiler output Option A (string formulas) or Option B (closures)?

### 8.8 Missing: `variables` Section Semantics

The YAML manifest has a `variables` section that defines shared lookup tables (`sizes`, `modifiers`, `skill_to_ability`). The Lua version correctly converts these to functions/tables at the top of the file, but:

1. The `Manifest` Go struct currently has no `Variables` field. The variables section was never implemented in the Go model.
2. With Lua, this becomes moot — the variables are just Lua code at the top of the file. But the Go `Manifest` struct loaded by `LoadManifest()` will need to change (or be replaced entirely by Lua environment capture).

### 8.9 Missing: Entity Loading (.lua vs .yaml)

The plan says entities move from `monster.yaml` to `monster.lua`. But the current `LoadEntity()` in `loader.go` uses `yaml.Unmarshal` into a typed `Entity` struct. With Lua:

1. Entity `.lua` files would set top-level variables (`id = "goblin"`, `hp = 10`, `stats = {str=10, dex=14}`).
2. The Go code must execute the Lua file and extract each field into the Go `Entity` struct.
3. Or: keep entities as YAML and only convert the manifest to Lua. This is simpler and avoids rewriting entity loading.

> [!IMPORTANT]
> Decision needed: Convert only `manifest.yaml → manifest.lua`, or also convert entity files to `.lua`? The plan implies both, but converting entities adds significant complexity for little benefit (entities are pure data, not logic).

### 8.10 Missing: Error Handling Parity

The current CEL evaluator provides structured errors with formula context:

```go
return nil, fmt.Errorf("CEL compile error: %w", issues.Err())
return nil, fmt.Errorf("game step '%s' failed: %w", step.Name, err)
```

The plan mentions "clear Lua syntax error messages with line numbers" but doesn't specify how GopherLua errors will be wrapped. `L.DoString()` returns `*lua.ApiError` which includes line/column info — this must be propagated through the executor pipeline.

---

## 9. Concrete Implementation Tasks

Based on the audit, here is the refined task breakdown:

### Phase 0 — Decisions (before any code)

* [ ] Choose formula evaluation model: **strings (Option A)** or **closures (Option B)**
* [ ] Choose entity format: **keep YAML** or **convert to Lua**
* [ ] Choose `variables` handling: **top-level Lua functions** (current example) or **Go-loaded lookup tables**
* [ ] Choose `steps` namespace: **flat `steps` map** (current) or **phase-scoped `game`/`targets`/`actor`**

### Phase 1 — GopherLua Sandbox (`internal/engine/lua.go`)

* [ ] Replace `eval.go` (245 lines) with `lua.go`
* [ ] `LuaEvaluator` struct: holds `*lua.LState`, `RollFunc`
* [ ] `NewLuaEvaluator(rollFunc)`: creates sandboxed `LState` with `base`, `table`, `string`, `math` only
* [ ] Register Go functions: `roll(dice)`, `mod(val)`, `help(topic)`
* [ ] `LoadManifestLua(path)`: executes `manifest.lua`, captures `commands` and `restrictions` tables
* [ ] `Eval(formula, ctx)`: sets globals (`actor`, `target`, `command`, `steps`, `is_*_active`), evaluates formula, extracts result
* [ ] `luaValueToGo(LValue) any`: recursive converter (`LTable` → `map`/`slice`, `LNumber` → `int`/`float64`, `LString` → `string`, `LBool` → `bool`)
* [ ] `goValueToLua(any) LValue`: reverse converter for context injection

### Phase 2 — Executor Adaptation (`internal/engine/executor.go`)

* [ ] Replace `Evaluator` references with `LuaEvaluator`
* [ ] Replace `BuildContext()` calls with Lua global injection
* [ ] Keep `mapStepToEvent()` — it's event-type mapping, independent of evaluation engine
* [ ] Update `ExecuteCommand()` signature if the evaluator interface changes

### Phase 3 — Manifest Format Migration

* [ ] Write `manifest.lua` for `world/dnd5e/`
* [ ] Update `session.go`'s `findAndLoadManifest()` to look for `manifest.lua` first, fall back to `manifest.yaml`
* [ ] Add `--luafy` command (if decided) or manual conversion

### Phase 4 — Dependency Cleanup

* [ ] Remove all `cel-go` imports from `eval.go` / delete `eval.go`
* [ ] `go mod tidy` to drop `cel-go` and transitive deps (`google.golang.org/genproto`, `google.golang.org/protobuf`, etc.)
* [ ] Verify binary size reduction

### Phase 5 — Testing

* [ ] Port all 12 executor tests to use `LuaEvaluator`
* [ ] Add Lua-specific tests: sandbox escapes, error messages, `LState` reuse
* [ ] `go test -race ./...` to verify thread safety
* [ ] Regression: `go build ./...` && `go test ./...`

---

## 10. Resolved Decisions

| Decision | Resolution |
| :--- | :--- |
| Formula evaluation model | **Hybrid**: strings for simple formulas, closures for complex ones. The evaluator MUST handle both `LString` (compile+eval via `L.DoString("return " .. formula)`) and `LFunction` (call via `L.CallByParam()`). |
| Entity format | **Keep YAML for now**. `LoadEntity()` continues to use `yaml.Unmarshal`. Entity `.lua` conversion is deferred to a future `--luafy` pass. |
| `variables` handling | **Top-level Lua code** in `manifest.lua`. Functions like `mod()`, `sizes()`, `skill_to_ability()` are plain Lua — no Go struct needed. The old YAML `variables` section was never implemented in Go and served only as a reference. |
| `steps` namespace | **Phase-scoped**: replace the flat `steps` map with `game`, `targets`, and `actor` namespaces. Each phase accumulates results into its own Lua global table. This is a **scope change** to the executor. |
| `--luafy` CLI flag | **Skip for now**. The `init` command (`Hidden: true`) is the future home for `--luafy`. Manual conversion of `manifest.yaml` → `manifest.lua` is sufficient for this change. |
| `check` command params | The Lua version uses `skill` and `dc` (dropping `type` and `name`). This is a **deliberate simplification**, not a bug. |

---

## 11. Thread Safety Analysis

Currently, `Session.Execute()` is called from two potential sources concurrently:

1. **TUI** (main goroutine via BubbleTea `Update`)
2. **Telegram bot** (background goroutine via `bot.Start()`)

Since `LState` is **not thread-safe**, we need serialization.

### Recommended: `sync.Mutex` on `Session.Execute()`

The simplest viable solution — ~5 lines of code:

```go
type Session struct {
    mu       sync.Mutex
    // ... existing fields
}

func (s *Session) Execute(input string) ([]engine.Event, error) {
    s.mu.Lock()
    defer s.mu.Unlock()
    // ... existing logic
}
```

* No command is dropped or queued; the second caller simply blocks until the first finishes.
* Command execution is fast (sub-millisecond for most formulas), so contention is negligible.
* No capacity limits, no timeouts, no extra complexity.

### Alternative: Buffered channel (if needed later)

A `chan` with capacity 1–3 and a `select` with timeout would provide drop-on-overflow semantics. This is ~15 lines but adds complexity without current need. **Defer to a future change if contention becomes measurable.**

> [!NOTE]
> The mutex approach is recommended because command execution is fast and contention between TUI and Telegram is rare (human typing speed is the bottleneck). Over-engineering the queue would violate the project's "no overcomplication" principle.

---

## 12. Final Scope Summary

### In scope for this change

1. **`internal/engine/lua.go`** — GopherLua evaluator replacing `eval.go` (CEL). Handles both string and closure formulas.
2. **`internal/engine/executor.go`** — Adapt pipeline to use Lua evaluator; replace flat `steps` with phase-scoped `game`/`targets`/`actor` namespaces.
3. **`internal/engine/types.go`** — Remove CEL-specific doc references; adapt `GameStep.Formula` field type if needed (currently `string`, may need `any` for closures).
4. **`world/dnd5e/manifest.lua`** — New Lua manifest replacing `manifest.yaml`. Includes `restrictions`, `variables` (as functions), and `commands`.
5. **`internal/session/session.go`** — Add `sync.Mutex` for thread safety; update `findAndLoadManifest()` to prefer `.lua`.
6. **Dependency cleanup** — Remove `cel-go`, `go mod tidy`.
7. **Tests** — Port executor tests to Lua evaluator.

### Deferred

1. `--luafy` transpiler on the `init` command.
2. Entity `.yaml` → `.lua` conversion.
3. Command queue with drop/timeout semantics.

### Remaining clarification

> [!WARNING]
> The **`GameStep.Formula` field type** needs a design decision. Currently it is `string` (for YAML unmarshaling via `yaml:"formula"`). With the hybrid approach (strings + closures), the `Manifest` loaded from Lua will populate `Formula` as either a Go `string` or a `*lua.LFunction`. Two options:
>
> **Option 1**: Change `Formula` from `string` to `any`. The executor type-switches to determine how to evaluate. Simple but loses static typing.
>
> **Option 2**: Keep `Formula` as `string`. When loading from Lua, convert closures to a sentinel string (e.g., `"__closure:stepname"`) and store the actual `*lua.LFunction` in a side map. The executor checks the sentinel and looks up the closure. More complex but keeps the struct clean.
>
> **Recommendation**: Option 1 (`any`) is simpler and aligns with the fact that the `Manifest` struct will no longer be populated by YAML unmarshaling at all — it will be built from Lua table traversal, where `any` is the natural type.

---

## 13. Future Vision: Lua as a Game Scripting Language

The real power of embedded Lua isn't just replacing CEL — it's enabling **users to script entire game systems** without touching Go code. Below are ideas for how `manifest.lua` evolves from a configuration file into a lightweight game authoring platform.

### 13.1 Game-Agnostic Primitives

The engine already provides system-neutral building blocks. With Lua, these become a small standard library that any game can compose:

| Primitive | Engine Concept | Lua API |
| :--- | :--- | :--- |
| **Loop** | Turn order, phases, rounds | `loop("combat")`, `end_loop("combat")` |
| **Entity** | Any tracked participant | `entity(id, {stats={...}})` |
| **Resource** | Depletable pools (HP, mana, ammo) | `spend(actor, "hp", 5)`, `restore(actor, "mp", 3)` |
| **Condition** | Named status effects | `add_condition(target, "stunned")`, `remove_condition(...)` |
| **Check** | Any pass/fail test | `check(actor, "skill", dc)` |
| **Roll** | Dice evaluation | `roll("2d6+3")` |
| **Ask** | Deferred player input | `ask(target, "choose: fight or flee")` |

A game author doesn't need to know Go, events, or JSONL — they just compose these primitives.

### 13.2 Minimal Manifest for a Non-D&D Game

To show how easy scripting becomes, here's a hypothetical **Fate Accelerated** manifest:

```lua
-- fate_accelerated/manifest.lua

function mod(val) return val end -- Fate doesn't use modifiers

-- Approaches instead of ability scores
approaches = {"careful", "clever", "flashy", "forceful", "quick", "sneaky"}

-- Fate uses only 4dF (Fudge dice: each die is -1, 0, or +1)
function fate_roll()
    return roll("4dF") -- engine recognizes "F" as fudge dice
end

restrictions = {
    gm_commands = {"scene_start", "scene_end", "compel"}
}

commands = {
    scene_start = {
        name = "scene start",
        game = {
            { name = "begin", formula = "true", event = "LoopEvent" }
        }
    },

    overcome = {
        name = "overcome",
        params = {
            { name = "approach", type = "string", required = true },
            { name = "dc", type = "int", required = true }
        },
        game = {
            {
                name = "roll_result",
                formula = function()
                    return fate_roll() + actor.stats[command.approach]
                end,
                event = "CheckEvent"
            }
        }
    },

    compel = {
        name = "compel",
        params = {
            { name = "to", type = "target", required = true },
            { name = "aspect", type = "string", required = true }
        },
        game = {
            { name = "offer", formula = function() return {target, "accept compel on: " .. command.aspect .. "?"} end, event = "AskIssuedEvent" },
            { name = "award_fate", formula = "1", event = "AttributeChangedEvent" }
        }
    }
}
```

The same engine runs D&D 5e combat, Fate scenes, or any other tabletop system. The user only writes Lua.

### 13.3 Hook System for Game-Specific Behaviors

Many games have "when X happens, also do Y" logic (reactions, triggered abilities, aura effects). This could be a `hooks` table in the manifest:

```lua
hooks = {
    -- Fires after any entity takes damage
    on_damage = function(target, amount)
        -- Undead Fortitude: CON save to stay at 1 HP
        if has_trait(target, "undead_fortitude") and target.resources.hp - amount <= 0 then
            local save = roll("1d20") + mod(target.stats.con)
            if save >= 5 + amount then
                set_hp(target, 1)
                hint(target.name .. " survives with Undead Fortitude!")
                return true -- cancel the kill
            end
        end
    end,

    -- Fires at the start of each turn in a loop
    on_turn_start = function(actor, loop_name)
        -- Regeneration
        if has_trait(actor, "regeneration") then
            restore(actor, "hp", actor.traits.regeneration)
            hint(actor.name .. " regenerates " .. actor.traits.regeneration .. " HP")
        end

        -- Condition duration tick-down
        tick_conditions(actor)
    end,

    -- Fires when a condition is added
    on_condition_added = function(target, condition_name)
        if condition_name == "prone" then
            set_speed(target, 0)
        end
    end
}
```

The engine calls these hooks at the appropriate points in the executor pipeline. Authors get reactive game logic without defining new event types.

### 13.4 Entity Templates

Instead of writing each monster from scratch, users define templates that the engine expands:

```lua
-- In manifest.lua or a shared library
templates = {
    undead = {
        conditions_immune = {"poisoned", "exhaustion"},
        damage_immune = {"poison"},
        traits = {"darkvision_60"}
    },
    humanoid = {
        size = "medium",
        languages = {"common"}
    }
}

-- In zombie.lua (entity file)
extends("undead", "humanoid")

id = "zombie"
name = "Zombie"
stats = { str = 13, dex = 6, con = 16, int = 3, wis = 6, cha = 5 }
resources = { hp = 22, actions = 1 }
traits = { undead_fortitude = true }
attacks = {
    slam = { dice = "1d6", modifier = "str", type = "bludgeoning" }
}
```

`extends()` merges template fields, and the entity file overrides or adds specifics. This is dramatically less repetitive than copying YAML blocks.

### 13.5 Custom Events Defined in Lua

Currently, event types are hardcoded in Go (`LoopEvent`, `AttributeChangedEvent`, etc.). With Lua, users could define domain-specific events that the engine applies generically:

```lua
-- A "spell slot" event for D&D
events = {
    SpellCastEvent = {
        apply = function(state, actor, data)
            local level = data.spell_level
            actor.spent["spell_slot_" .. level] = (actor.spent["spell_slot_" .. level] or 0) + 1
        end,
        message = function(actor, data)
            return actor.name .. " casts " .. data.spell_name .. " (level " .. data.spell_level .. ")"
        end
    }
}
```

The engine would have a generic `CustomEvent` Go type that delegates `Apply()` and `Message()` to the Lua functions. This means new game mechanics never require Go code changes.

### 13.6 Hot-Reload During Play

Since the `LState` can re-execute `manifest.lua` at any time, a `reload` command would let the GM tweak rules mid-session:

```shell
> reload
Manifest reloaded. 5 commands updated, 2 hooks changed.
```

This is invaluable during playtesting — the GM edits `manifest.lua` in a text editor, types `reload`, and the new rules take effect immediately without restarting the session. The event log is unaffected since it stores the outcomes, not the rules.

### 13.7 Community Game Packs

With all game logic in `.lua` files, sharing becomes trivial:

```shell
worlds/
  dnd5e/          ← official D&D 5e pack
    manifest.lua
    data/monsters/
    data/spells/
  fate/            ← Fate Accelerated pack
    manifest.lua
  pbta/            ← Powered by the Apocalypse pack
    manifest.lua
  homebrew/        ← user's custom game
    manifest.lua   ← extends dnd5e, overrides specific commands
```

A `manifest.lua` could even `require()` another manifest as a base:

```lua
-- homebrew/manifest.lua
local base = require("dnd5e.manifest") -- loads the parent system

-- Override just the initiative command with a house rule
base.commands.initiative.game[1].formula = function()
    return roll("2d20kh1") + mod(actor.stats.dex) -- advantage on initiative
end

-- Add a homebrew command
base.commands.rally = {
    name = "rally",
    game = {
        { name = "inspire", formula = function() return roll("1d4") end, event = "AttributeChangedEvent" }
    }
}

-- Export the modified manifest
commands = base.commands
restrictions = base.restrictions
hooks = base.hooks
```

This turns game system authoring into something like "modding" — override what you want, keep the rest.
