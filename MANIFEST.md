# Manifest Reference

This document explains how to write a `manifest.lua` file — the configuration file that defines the rules of your game system for the Ancient Draconic engine.

## Table of Contents

- [Overview](#overview)
- [Section 1: Free Code](#section-1-free-code)
- [Section 2: Restrictions](#section-2-restrictions)
- [Section 3: Commands](#section-3-commands)
  - [Command Structure](#command-structure)
  - [Prereqs](#prereqs)
  - [Command Phases (Steps & Hooks)](#command-phases-steps--hooks)
  - [The value Field](#the-value-field)
- [Context Variables](#context-variables)
- [Built-in Commands](#built-in-commands)
- [Built-in Functions](#built-in-functions)
  - [roll](#roll)
  - [loop](#loop)
  - [loop_order](#loop_order)
  - [loop_value](#loop_value)
  - [add_actor](#add_actor)
  - [ask](#ask)
  - [condition](#condition)
  - [remove_condition](#remove_condition)
  - [spend](#spend)
  - [set_attr](#set_attr)
  - [contest](#contest)
  - [check_result](#check_result)
  - [hint](#hint)
  - [metadata](#metadata)
  - [emit](#emit)
- [Complete Example](#complete-example)

---

## Overview

A manifest file is a Lua script that tells the engine what commands exist in your game, what rules govern them, and what happens when they are executed. When you start a game session, the engine loads your `manifest.lua`, reads the `restrictions` and `commands` tables, and uses them to process every command the players type.

The file has three logical sections:

1. **Free Code** — helper functions, lookup tables, and utilities you define for reuse.
2. **Restrictions** — rules about who can run which commands.
3. **Commands** — the actual game commands: what parameters they accept, what conditions must be met, and what happens when they execute.

Your manifest file **must** define a global table called `commands`. The `restrictions` table is optional.

---

## Section 1: Free Code

At the top of your manifest, you can write any valid Lua code. This is where you define helper functions and lookup tables that your commands will use later. This code runs once when the manifest is loaded.

```lua
-- A lookup table mapping skills to their governing ability score
local _skill_map = {
    athletics = "str",
    acrobatics = "dex",
    stealth = "dex",
    perception = "wis",
}

-- A helper function to calculate an ability modifier (D&D 5e formula)
function mod(val)
    if not val then return -5 end
    return math.floor(val / 2) - 5
end

-- A helper function to look up which ability a skill uses
function skill_to_ability(skill)
    return _skill_map[skill] or "str"
end
```

**Key points:**

- Use `local` for tables or values that are private to this file.
- Use `function name()` (without `local`) for functions you want available inside your command closures.
- You have access to four standard Lua libraries: `base`, `table`, `string`, and `math`. File I/O, operating system access, and debug tools are **not** available (the engine runs a secure sandbox).

---

## Section 2: Restrictions

The `restrictions` table controls access to commands. It has two sub-tables:

```lua
restrictions = {
    adjudication = {
        commands = { "grapple", "shove" }
    },
    gm_commands = { "encounter_start", "encounter_end", "add_condition", "remove_condition" }
}
```

### `gm_commands`

A list of command names that **only the GM can execute**. If a player tries to run one of these commands, the engine returns an "unauthorized" error.

In standard D&D 5e configuration, we usually reserve these explicit sandbox commands for the GM:

- `encounter_start`, `encounter_end` (for loop management)
- `add_condition`, `remove_condition` (for manual target condition management)

### `adjudication.commands`

A list of command names that require **GM approval** before they take effect. When a player issues one of these commands, the GM gets a prompt to allow or deny it.

If you don't need restrictions, you can omit the entire `restrictions` table.

---

## Section 3: Commands

The `commands` table is the heart of the manifest. Each key is the command's internal name (using underscores), and each value is a table describing how that command works.

```lua
commands = {
    encounter_start = { ... },
    initiative = { ... },
    grapple = { ... },
}
```

The internal name `encounter_start` maps to the typed command `encounter start` (the engine converts spaces to underscores automatically).

### Command Structure

Every command can have the following fields:

```lua
encounter_start = {
    -- Display name (shown in help and autocomplete)
    name = "encounter start",

    -- Parameters the user can provide
    params = {
        { name = "with", type = "list<target>", required = false }
    },

    -- Conditions that must be true before the command runs
    prereq = { ... },

    -- A short message shown after the command executes
    hint = "Roll initiative for all actors.",

    -- A longer explanation shown by the `help` command
    help = "Starts a new combat encounter.",

    -- Usage string shown when the user provides invalid parameters
    error = "encounter start [with: Actor1 [and: Actor2]*]",

    -- Steps that run once when the command is executed
    game = { ... },

    -- Steps that run once for each target
    targets = { ... },

    -- Steps that run once affecting the actor who issued the command
    actor = { ... },
}
```

**Parameter types:** `"string"`, `"int"`, `"target"`, `"list<target>"`.

### Prereqs

Prereqs are checks that must pass before the command does anything. If a prereq returns `false`, the engine stops and shows the error message.

```lua
prereq = {
    {
        name = "check_conflict",
        value = function() return not is_encounter_start_active end,
        error = "an encounter is already active"
    },
    {
        name = "check_action",
        value = function() return actor.spent.actions < actor.resources.actions end,
        error = "no actions remaining"
    }
}
```

Each prereq has:

- **`name`** — a label (used in error messages for debugging).
- **`value`** — a closure that returns `true` (pass) or `false` (fail).
- **`error`** — the message shown to the user when the check fails.

### Command Phases (Steps & Hooks)

The `game`, `targets`, and `actor` fields each define a Command Phase. A Command Phase can optionally contain `steps` and `hooks`.

The difference between `game`, `targets`, and `actor` is **when and to whom** their steps and hooks apply:

| Phase | Runs | Best used for |
| :--- | :--- | :--- |
| `game` | Once per command | Dice rolls, starting loops, global game hooks |
| `targets` | Once **per target** | Asking targets to make saves, applying conditions to targets, target-specific mechanics |
| `actor` | Once per command | Consuming the actor's action, applying self-buffs, adding individual hooks |

#### Steps

Steps execute immediately when the command is run. Each step is a table with two fields:

```lua
game = {
    steps = {
        { name = "create_loop", value = function() return loop("encounter_start", true) end }
    }
}
```

*For backward compatibility, if you define an array directly inside `game`, `targets`, or `actor` (without the `steps` wrapper), the engine will imply them as steps.*

- **`name`** — a label for this step. Other steps in the same command can reference its result by this name (e.g., `game.create_loop`).
- **`value`** — a closure that returns the step's result and an optional event.

#### Hooks

Hooks allow you to delay the execution of an effect until a specific phase of the game (such as the start of a turn).

```lua
actor = {
    steps = {
        { name = "disengage_apply", value = function() return condition("disengaged") end }
    },
    hooks = {
        { name = "end_disengage", type = "next_turn", value = function() return remove_condition("disengaged") end }
    }
}
```

A hook has:

- **`name`** — an identifier for the hook.
- **`type`** — defines **when** this hook is triggered (see supported types below).
- **`value`** — a closure (identical to a Step `value` closure) evaluated when the trigger occurs.

**Supported Hook Types**:

- `next_turn`: Runs at the beginning of *any* actor's turn.
- `next_turn_end`: Runs at the end of *any* actor's turn.
- `next_round`: Runs at the beginning of the next loop cycle/round.
- `next_round_end`: Runs at the end of the next loop cycle/round.
- `next_actor_turn` & `next_target_turn`: Runs at the beginning of the turn of the specific entity this hook was attached to (requires being defined in the `actor` or `targets` phase).
- `next_actor_turn_end` & `next_target_turn_end`: Runs at the end of the turn of the specific entity.

### The `value` Field

The `value` closure is where all game logic happens. It always looks like this:

```lua
value = function()
    -- your logic here
    return some_result
end
```

What you return determines what happens:

- **Return a helper function call** (like `loop(...)`, `condition(...)`, `spend(...)`) → the engine creates a game event and applies it to the state.
- **Return a plain value** (like a number, string, or boolean) → the value is stored as a step result that later steps can reference, but no event is created.

For example, this step rolls dice and stores the result, but doesn't emit any event:

```lua
{ name = "attack_roll", value = function() return roll("1d20") + mod(actor.stats.str) end }
```

And this step uses the previous result to emit a hint:

```lua
{ name = "show_result", value = function()
    return hint("Attack roll: " .. tostring(game.attack_roll))
end }
```

---

## Context Variables

Inside every `value` closure, the engine injects several global variables that give you access to the current game state. Which variables are available depends on which phase (`game`, `targets`, or `actor`) the step belongs to.

### Available in all phases

| Variable | Type | Description |
| :--- | :--- | :--- |
| `actor` | table | The entity performing the command. Has fields: `id`, `name`, `stats`, `resources`, `spent`, `conditions`, `proficiencies`, `statuses`, `inventory`, `types`, `classes`. |
| `command` | table | The parsed parameters from the user's input. For example, if the user typed `check skill: athletics dc: 15`, then `command.skill` is `"athletics"` and `command.dc` is `15`. |
| `game` | table | Results from steps in the `game` phase, keyed by step name. For example, after a step named `"contest"` runs, `game.contest` holds its return value. |
| `metadata` | table | The session's metadata store. Persistent key-value pairs set by `metadata()` calls. |
| `is_<name>_active` | boolean | Whether a loop named `<name>` is currently active. For example, `is_encounter_start_active` is `true` while the encounter loop is running. |

### Available in `targets` phase only

| Variable | Type | Description |
| :--- | :--- | :--- |
| `target` | table | The current target entity (same shape as `actor`). |
| `targets` | table | Results from previous steps in the `targets` phase for the current target, keyed by step name. |

### Available in `actor` phase only

| Variable | Type | Description |
| :--- | :--- | :--- |
| `actor_results` | table | Results from previous steps in the `actor` phase, keyed by step name. |

### Entity shape

Both `actor` and `target` have the same structure:

```lua
actor.id              -- "fighter"
actor.name            -- "Fighter"
actor.stats.str       -- 18
actor.stats.dex       -- 14
actor.resources.hp    -- 45
actor.resources.actions -- 1
actor.spent.actions   -- 0
actor.spent.hp        -- 10
actor.conditions      -- {"grappled", "prone"}
actor.proficiencies   -- {athletics = 1, perception = 1}
actor.statuses        -- {has_attacked = "true"}
actor.inventory       -- {}
actor.types           -- {"humanoid"}
actor.classes         -- {"fighter"}
```

---

## Built-in Commands

The engine provides several built-in commands that cannot be overridden by the manifest. These are available in every game:

- **`roll [dice: <expression>]`**: Evaluates a dice expression and returns the result (e.g., `roll dice: 1d20+5`).
- **`help [command: <name>]`**: Displays help documentation for all commands or a specific command.
- **`hint`**: Displays the hint for the last executed command.
- **`ask [options: <list>]`**: Requests input/choice from targets.
- **`allow`**: A GM-only command to approve a pending adjudication request.
- **`deny`**: A GM-only command to reject a pending adjudication request.
- **`adjudicate`**: Similar to `allow` (currently an alias).
- **`undo [steps: <N>] [turn: <N>] [round: <N>]`**: A GM-only command that rewinds the game state by undoing the last $N$ commands, or jumping back to the start of a specific round/turn.

---

## Built-in Functions

These functions are registered by the engine and available inside every `value` closure. They create **events** — instructions that the engine applies to the game state after the step completes.

---

### `roll`

Rolls dice using standard dice notation.

```lua
roll(dice_string)
```

| Argument | Type | Description |
| :--- | :--- | :--- |
| `dice_string` | string | A dice expression like `"1d20"`, `"2d6"`, `"4d8"` |

**Returns:** a number (the total of all dice rolled).

**Example:**

```lua
value = function() return roll("1d20") + mod(actor.stats.str) end
```

> **Note:** `roll()` does **not** create an event. It just returns a number. Use it inside other helper calls or as a plain step result.

---

### `loop`

Starts or stops a named loop. Loops track turn order (like combat rounds).

```lua
loop(name, active)
```

| Argument | Type | Description |
| :--- | :--- | :--- |
| `name` | string | The loop's identifier (e.g., `"encounter_start"`). |
| `active` | boolean | `true` to start the loop, `false` to stop it. |

**What it does:**

- When `active` is `true`: creates a new loop with the given name, initializes its actor list and turn order.
- When `active` is `false`: deactivates the loop and clears its actors.

**Example:**

```lua
-- Start the encounter loop
{ name = "start", value = function() return loop("encounter_start", true) end }

-- End the encounter loop
{ name = "stop", value = function() return loop("encounter_start", false) end }
```

---

### `loop_order`

Sets the sort direction for a loop's turn order.

```lua
loop_order(name, ascending)
```

| Argument | Type | Description |
| :--- | :--- | :--- |
| `name` | string | The loop's identifier. |
| `ascending` | boolean | `true` for ascending order (lowest first), `false` for descending (highest first). |

**Example:**

```lua
-- D&D initiative: highest goes first (descending)
{ name = "order", value = function() return loop_order("encounter_start", false) end }
```

---

### `loop_value`

Sets an actor's numeric value in a loop's turn order (e.g., initiative score).

```lua
loop_value(name, value)
```

| Argument | Type | Description |
| :--- | :--- | :--- |
| `name` | string | The loop's identifier. |
| `value` | number | The actor's order value (e.g., their initiative roll). |

**What it does:** Records the current actor's score in the named loop's turn order.

**Example:**

```lua
-- Roll initiative and register the result in the encounter loop
{ name = "roll_score", value = function()
    return loop_value("encounter_start", roll("1d20") + mod(actor.stats.dex))
end }
```

---

### `add_actor`

Adds one or more actors to the active loop.

```lua
add_actor(id_or_list)
```

| Argument | Type | Description |
| :--- | :--- | :--- |
| `id_or_list` | string or table | A single actor ID (`"fighter"`) or a list of IDs (`{"fighter", "goblin"}`). |

**Example:**

```lua
-- Add actors from the "with" parameter to the encounter
{ name = "add", value = function() return add_actor(command.with) end }
```

---

### `ask`

Sends a prompt to a target entity, asking them to choose from a list of options.

```lua
ask(target_id, option1, option2, ...)
```

| Argument | Type | Description |
| :--- | :--- | :--- |
| `target_id` | string | The ID of the entity being asked. |
| `option1`, `option2`, ... | strings | The choices available to the target. |

**What it does:** Creates an `AskIssuedEvent` that the TUI or Telegram bot presents to the target player.

**Example:**

```lua
-- Ask a target to roll initiative
{ name = "ask_init", value = function()
    return ask(target.id, "initiative")
end }

-- Ask a target to choose between two skill checks
{ name = "ask_save", value = function()
    local dc = game.contest.value
    return ask(target.id,
        "check skill: athletics dc: " .. tostring(dc),
        "check skill: acrobatics dc: " .. tostring(dc))
end }
```

---

### `condition`

Applies a condition to the current target (or actor, depending on the phase).

```lua
condition(condition_name)
```

| Argument | Type | Description |
| :--- | :--- | :--- |
| `condition_name` | string | The condition to add (e.g., `"grappled"`, `"prone"`, `"stunned"`). |

**Example:**

```lua
-- In the targets phase: apply "grappled" to each target
{ name = "grapple", value = function() return condition("grappled") end }
```

---

### `remove_condition`

Removes a condition from the current target (or actor).

```lua
remove_condition(condition_name)
```

| Argument | Type | Description |
| :--- | :--- | :--- |
| `condition_name` | string | The condition to remove. |

**Example:**

```lua
{ name = "ungrapple", value = function() return remove_condition("grappled") end }
```

---

### `spend`

Increments a spent resource counter for the acting entity.

```lua
spend(resource_key, amount)
```

| Argument | Type | Description |
| :--- | :--- | :--- |
| `resource_key` | string | The resource to spend (e.g., `"actions"`, `"arrows"`, `"speed"`). |
| `amount` | number | (Optional) The amount to spend. Defaults to 1. |

**What it does:** Adds 1 to `actor.spent[resource_key]`.

**Example:**

```lua
-- Consume one action
{ name = "consume", value = function() return spend("actions", 1) end }

-- Consume 5 arrows
{ name = "ammo", value = function() return spend("arrows", 5) end }
```

---

### `set_attr`

Sets an arbitrary attribute on an entity.

```lua
set_attr(section, key, value)
```

| Argument | Type | Description |
| :--- | :--- | :--- |
| `section` | string | The entity section to modify (`"stats"`, `"resources"`, `"spent"`, `"statuses"`). |
| `key` | string | The attribute name within that section. |
| `value` | any | The new value. |

**Example:**

```lua
-- Set an entity's HP to 50
{ name = "set_hp", value = function() return set_attr("resources", "hp", 50) end }

-- Mark that the actor has attacked this turn
{ name = "mark", value = function() return set_attr("statuses", "has_attacked", "true") end }
```

---

### `contest`

Records the result of a contested roll (e.g., grapple, shove). The value is stored in session metadata so subsequent steps can reference it.

```lua
contest(roll_value)
```

| Argument | Type | Description |
| :--- | :--- | :--- |
| `roll_value` | number | The total of the initiator's contested roll. |

**What it does:** Stores `{ actor = actor_id, value = roll_value }` in `metadata.contest`.

**Example:**

```lua
{ name = "contest", value = function()
    return contest(roll("1d20") + mod(actor.stats.str))
end }
```

Later steps can reference the contest result via `game.contest.value`.

---

### `check_result`

Records the outcome of an ability check or saving throw.

```lua
check_result(passed)
```

| Argument | Type | Description |
| :--- | :--- | :--- |
| `passed` | boolean | `true` if the check succeeded, `false` if it failed. |

**Example:**

```lua
{ name = "result", value = function()
    local total = roll("1d20") + mod(actor.stats.dex)
    return check_result(total >= command.dc)
end }
```

---

### `hint`

Displays a message to all players.

```lua
hint(message)
```

| Argument | Type | Description |
| :--- | :--- | :--- |
| `message` | string | The text to display. |

**Example:**

```lua
{ name = "announce", value = function()
    return hint(actor.id .. " rolled initiative: " .. tostring(game.roll_score))
end }
```

---

### `metadata`

Stores a key-value pair in the session's metadata. This persists across commands and can be read by any future step.

```lua
metadata(key, value)
```

| Argument | Type | Description |
| :--- | :--- | :--- |
| `key` | string | The metadata key. |
| `value` | any | The value to store. |

**Example:**

```lua
{ name = "record", value = function()
    return metadata("last_attacker", actor.id)
end }
```

---

### `emit`

Creates a **custom event** with an arbitrary type and payload. Use this when none of the built-in helpers fit your needs.

```lua
emit(event_type, payload)
```

| Argument | Type | Description |
| :--- | :--- | :--- |
| `event_type` | string | A name for your custom event type (e.g., `"arcane_blast"`). |
| `payload` | table | A table of data to store. The engine saves it in `metadata[event_type]`. |

**Example:**

```lua
{ name = "cast_spell", value = function()
    return emit("fire_bolt", {
        damage = roll("1d10"),
        caster = actor.id,
        target = target.id
    })
end }
```

After this event is applied, `metadata.fire_bolt` contains `{ damage = 7, caster = "wizard", target = "goblin" }`.

---

## Complete Example

Here is a minimal but complete manifest that defines an encounter system:

```lua
-- Section 1: Free Code
function mod(val)
    if not val then return -5 end
    return math.floor(val / 2) - 5
end

-- Section 2: Restrictions
restrictions = {
    gm_commands = { "encounter_start", "encounter_end" }
}

-- Section 3: Commands
commands = {
    encounter_start = {
        name = "encounter start",
        params = {
            { name = "with", type = "list<target>", required = false }
        },
        prereq = {
            {
                name = "check_conflict",
                value = function() return not is_encounter_start_active end,
                error = "an encounter is already active"
            }
        },
        hint = "Roll initiative for all actors.",
        help = "Starts a new combat encounter.",
        error = "encounter start [with: Actor1 [and: Actor2]*]",
        game = {
            { name = "create_loop", value = function() return loop("encounter_start", true) end },
            { name = "order",       value = function() return loop_order("encounter_start", false) end },
            { name = "add_actors",  value = function() return add_actor(command.with) end },
        },
        targets = {
            { name = "ask_init", value = function() return ask(target.id, "initiative") end }
        }
    },

    encounter_end = {
        name = "encounter end",
        prereq = {
            {
                name = "check_conflict",
                value = function() return is_encounter_start_active end,
                error = "no active encounter to end"
            }
        },
        hint = "Encounter has ended.",
        help = "Ends the current combat encounter.",
        error = "encounter end",
        game = {
            { name = "stop", value = function() return loop("encounter_start", false) end }
        }
    },

    initiative = {
        name = "initiative",
        prereq = {
            {
                name = "check_active",
                value = function() return is_encounter_start_active end,
                error = "no active encounter"
            }
        },
        hint = "Wait for your turn.",
        help = "Rolls initiative for the current actor.",
        error = "initiative",
        game = {
            { name = "roll_score", value = function()
                return loop_value("encounter_start", roll("1d20") + mod(actor.stats.dex))
            end },
            { name = "announce", value = function()
                return hint(actor.id .. " rolled initiative")
            end }
        }
    },
}
```
