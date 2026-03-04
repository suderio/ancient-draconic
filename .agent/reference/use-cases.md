# Use Cases and Corner Cases

## Purpose

Provide implementation-ready scenarios that make expected behavior and failure handling unambiguous.

## Normative Rules

1. New capabilities MUST include at least one normal scenario and one corner-case scenario.
2. Each scenario MUST define inputs, preconditions, execution steps, and expected outputs.
3. Failure paths MUST name expected error category.

---

## Use Cases

### Bot Telegram

Register a global Telegram bot token so the application can connect to the Telegram API.

**Normal flow:**

1. User runs `draconic bot telegram --token <TOKEN>`.
2. Token is saved to Viper config (`~/.draconic.yaml`).
3. Output: "Telegram bot token saved successfully."

**Interactive flow:**

1. User runs `draconic bot telegram` without `--token`.
2. CLI displays BotFather instructions.
3. User pastes token interactively.
4. Token is saved.

**Corner case — empty token:**

1. User presses Enter without typing a token.
2. No config is written, no error message. Silent no-op.

---

### Campaign Create

Bootstrap a new campaign directory with an empty event log under a world.

**Normal flow:**

1. User runs `draconic campaign create dnd5e my_campaign`.
2. Creates `worlds/dnd5e/my_campaign/` with `log.jsonl` and `data/` subdirs.
3. Output: "Successfully created campaign!" with log path.

**Corner case — missing arguments:**

1. User runs `draconic campaign create` with no world name.
2. Output: error message, exit code 1.

**Corner case — duplicate campaign:**

1. User runs `draconic campaign create dnd5e my_campaign` when it already exists.
2. `CampaignManager.Create()` returns an error.
3. Output: "Error creating campaign: ..."

---

### Campaign Telegram

Configure Telegram settings (chat ID, user-to-entity mappings) for a specific campaign.

**Normal flow:**

1. User runs `draconic campaign telegram dnd5e my_campaign --chat_id -12345 --user fighter:123456`.
2. Creates/updates `worlds/dnd5e/my_campaign/telegram.yaml` with chat ID and user map.
3. Output: "Telegram campaign configuration saved to ..."

**Interactive flow:**

1. User runs `draconic campaign telegram dnd5e my_campaign` without `--chat_id`.
2. CLI displays instructions for finding chat ID.
3. User pastes chat ID interactively.

**Corner case — invalid user pair format:**

1. User passes `--user "badformat"` (no colon separator).
2. Warning printed, mapping skipped, other mappings still saved.

**Corner case — campaign doesn't exist:**

1. User runs for a non-existent campaign directory.
2. Output: "Error: campaign directory ... does not exist. Run 'campaign create' first."

---

### Repl

Start an interactive game session with the TUI (BubbleTea) for sending DSL commands.

**Normal flow:**

1. User runs `draconic repl dnd5e my_campaign`.
2. Session loads manifest (prefers `manifest.lua`, falls back to `manifest.yaml`).
3. Replays existing event log to rebuild game state.
4. TUI renders with command input, autocomplete, and state display.
5. User types commands (e.g., `encounter start by: GM with: Fighter and: Goblin`).
6. Commands are parsed, executed, events persisted, state updated.

**Corner case — no manifest found:**

1. The world directory has neither `manifest.lua` nor `manifest.yaml`.
2. Session creation fails with a clear error.

**Corner case — Lua syntax error in manifest:**

1. `manifest.lua` has invalid Lua syntax.
2. `L.DoFile()` fails with line number and error message.
3. Session creation fails, TUI never starts.

**Corner case — formula error during execution:**

1. A command's formula references `actor.stats.nonexistent`.
2. Lua returns nil; the evaluator wraps the error with step name and command name.
3. Error displayed in TUI, game state unchanged.

---

### Lua Manifest Loading

Load a Lua manifest and capture game rules into the engine.

**Normal flow:**

1. Engine creates sandboxed `LState` with `base`, `table`, `string`, `math`.
2. Registers Go functions: `roll()`, `mod()`.
3. Executes `manifest.lua` via `L.DoFile()`.
4. Reads `commands` global table → builds `map[string]CommandDef`.
5. Reads `restrictions` global table → builds `Restrictions` struct.
6. Helper functions (`mod()`, `sizes()`, `skill_to_ability()`) remain in Lua scope for formula use.

**Corner case — sandbox escape attempt:**

1. `manifest.lua` contains `os.execute("rm -rf /")`.
2. `os` library is not loaded; Lua raises "attempt to index a nil value."
3. Manifest load fails safely.

**Corner case — missing commands table:**

1. `manifest.lua` doesn't define a `commands` global.
2. Loader returns error: "manifest.lua must define a 'commands' table."

---

### Hybrid Formula Evaluation

Execute commands with both string and closure formulas.

**Normal flow (string formula):**

1. Command `encounter_start` has `formula = "true"` (string).
2. Evaluator runs `L.DoString("return true")`.
3. Result: Lua `LTrue` → Go `true` → `LoopEvent{Active: true}`.

**Normal flow (closure formula):**

1. Command `grapple` has `formula = function() return roll("1d20") + mod(actor.stats.str) end`.
2. Evaluator calls `L.CallByParam()` on the closure.
3. Globals `actor`, `command` were set before the call.
4. Result: Lua `LNumber(17)` → Go `int(17)` → `ContestStarted{Value: 17}`.

**Corner case — closure references stale globals:**

1. Between two formula calls, the engine fails to reset `target`.
2. Second formula reads stale `target` from the previous call.
3. This MUST NOT happen: globals MUST be reset before each evaluation.
