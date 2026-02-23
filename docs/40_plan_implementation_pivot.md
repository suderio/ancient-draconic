
# 100_plan_implementation_pivot

We have a classic "Developer's Wall" in game engine design: the transition from a simple dice-roller to a full-system simulator. D&D 5e is a game of exceptions, and hard-coding every "if" for class features like *Sneak Attack* or *Undead Fortitude* will lead to a fragile codebase.

To keep **Ancient Draconic** maintainable, we need to move from **Hard-Coded Logic** to a **Data-Driven Rule Engine**.

Important: Every code shown here is *not* to be used as is. It is just an example of how the code should look like. I want you to use this as a reference to create the actual code that will be used in the project. Do not copy and paste code from here to the project.

---

## 1. The Architectural Shift: Logic as Data

Instead of Go code containing the rule (e.g., `if character.IsRogue && hasAdvantage { ... }`), the Go code should become a **Generic Evaluator** that runs small, isolated scripts attached to your YAML data.

### The Solution: CEL-Go (Common Expression Language)

Since you are using Go, **CEL-Go** is your best ally. It is a fast, safe, and "Google-native" expression language that allows you to store logic as strings in your YAML files.

* **The Logic:** You pass the `GameState` into a CEL environment, and it returns `true` or `false` (or a number).
* **The YAML integration:**

```yaml
# In rogue.yaml
special_abilities:
  - name: Sneak Attack
    # The rule is now data, not Go code!
    condition: "actor.class == 'Rogue' && (target.has_ally_adjacent || actor.has_advantage)"
    damage_dice: "1d6"

```

---

## 2. Decoupling "Intent" from "Resolution"

Your current collapse is likely caused by the `Execute()` or `Process()` functions trying to do too much. To fix this, we should split the "Rules" into three distinct, testable phases:

| Phase | Responsibility | Tool |
| --- | --- | --- |
| **Requirements** | Can I even do this? (e.g., "Do I have a spell slot?") | **CEL Expressions** |
| **Effects** | What happens? (e.g., "Deal 1d8 damage") | **Event Generator** |
| **Triggers** | Does this cause something else? (e.g., "Undead Fortitude") | **Observer Pattern** |

---

## 3. Improving Testability: The "Behavioral" Spec

If testing is getting hard, it’s because your tests are too coupled to the internal state. Use your **Combat Script** pattern to test "Behaviors" rather than "Functions".

### Example: Testing "Undead Fortitude"

Instead of testing the Go function that calculates the save, test the sequence:

1. **Command:** `attack zombie` (Damage = 20).
2. **Expectation:** Response Status is `PENDING_ROLL`.
3. **Command:** `roll zombie con-save` (Result = 18).
4. **Expectation:** Zombie HP is 1, not 0.

---

## 4. Antigravity Mission: Rule Engine Refactor

To stop the "if" statements from growing, instruct your Antigravity agent to implement a **Rule Registry**.

> **Mission: Refactor to Data-Driven Rules via CEL-Go**
>
> 1. **Integration:** Add `google/cel-go` to the project.
> 2. **Context Binding:** Create a `Mapping` that exposes the `GameState` and `Command` objects to CEL.
> 3. **The Ability Parser:** Update the YAML loader so that `special_abilities` can contain a `condition` string.
> 4. **Generic Processor:** Replace the hard-coded "if-rogue" logic with a loop that checks all `actor.Abilities` and evaluates their `condition` via CEL.
>
>

---

## 5. Why this works for the Magic System

Magic is the ultimate test of this system. A spell is just a YAML file with:

* **Requirement:** `actor.spell_slots[level] > 0`
* **Effect:** `target.hp -= roll(damage_dice)`
* **Side Effect:** `actor.spell_slots[level] -= 1`

---

## 6. How to implement this

To transition from hard-coded "if" statements to a data-driven system, we need to "bind" your Go structs (like the **Fighter** and **Rogue**) so that the **CEL-Go** engine can read them directly.

This allows you to write rules like `actor.dexterity > 15` or `target.type == 'undead'` inside your **YAML** files.

### 1. The Go-to-CEL Binding Logic

In **Antigravity**, instruct your agent to create a `rules` package. This package will handle the conversion of your game state into a format CEL understands.

```go
import (
    "github.com/google/cel-go/cel"
    "github.com/google/cel-go/checker/decls"
)

// Define the environment with your Domain Models
func CreateRuleEnv() (*cel.Env, error) {
    return cel.NewEnv(
        // Expose Character and Monster attributes to the rule engine
        cel.Declarations(
            decls.NewVar("actor", decls.NewObjectType("models.Entity")),
            decls.NewVar("target", decls.NewObjectType("models.Entity")),
            decls.NewVar("roll_result", decls.Int),
        ),
    )
}

```

### 2. Implementation: The "Ability" Evaluator

By using this binding, your `Process` function becomes a **Coordinator**. Instead of checking for **Sneak Attack** manually, it iterates through the Actor's abilities and evaluates their conditions.

| Hard-Coded (Current State) | Data-Driven (Future State) |
| --- | --- |
| `if actor.Class == "Rogue" { ... }` | `eval("ability.condition", context)` |
| `if target.HP <= 0 && isZombie { ... }` | `eval("special.trigger", context)` |
| **Maintenance:** High (requires re-compile) | **Maintenance:** Low (edit YAML) |

---

### 3. Mission Brief: Rule System Decoupling

Paste this into your **Antigravity** task window to begin the refactor:

> **Mission: Decouple Game Logic using CEL-Go**
>
> 1. **Binding:** Create a `rules` package that maps the `ActiveEntity` struct (HP, Stats, Type) to CEL variables.
> 2. **YAML Update:** Update the `Ability` struct to include a `Condition` string and an `Effect` string.
> 3. **The Logic Loop:** In the `engine`, create a loop that runs before every action to check for **Passive Requirements** (e.g., "Do I have the required Strength?") and **Active Triggers** (e.g., "Does this damage trigger Undead Fortitude?").
> 4. **Testing:** Create a test case where a custom YAML-defined ability is added to **Thorne Ironwill** without changing any Go code, and verify it executes via the `repl`.
>
>

---

### 4. Why this Solves the Magic System

When you reach the magic system, you won't need to write a `Fireball()` function. A spell will simply be an object with three CEL strings:

* **Requirement:** `actor.slots_level_3 > 0`
* **Targeting:** `targets.count <= 1`
* **Resolution:** `target.hp -= roll('8d6')`

This architecture allows you to scale to hundreds of spells while keeping your Go core clean and focused on **Event Sourcing** and **Persistence**.

## 7. Entity

To prevent the "if-statement collapse," the `Entity` struct needs to be structured so that **CEL-Go** can "reflect" over it easily. By using a standard Go struct with JSON tags that match your YAML keys, you create a seamless pipeline from your data files to your rule logic.

---

### 1. The `models.Entity` Blueprint

This struct is designed to hold the state of both **Monsters** and **Characters**. It provides the "Context" that your rule expressions will query.

```go
package models

// Entity represents any actor in the game (PC or Monster)
type Entity struct {
 Name           string         `json:"name"`
 Index          string         `json:"index"`
 Type           string         `json:"type"` // e.g., "undead", "humanoid"
 HP             int            `json:"hit_points"`
 MaxHP          int            `json:"max_hp"`
 ArmorClass     int            `json:"armor_class"` // Simplified for CEL evaluation
 
 // Ability scores are mapped for easy access: actor.stats.str
 Stats          map[string]int `json:"stats"` 
 
 // Track temporary states like advantage or conditions
 Conditions     []string       `json:"conditions"`
 
 // The list of abilities that contain the CEL logic
 SpecialAbilities []Ability    `json:"special_abilities"`
}

type Ability struct {
 Name      string `json:"name"`
 Condition string `json:"condition"` // The CEL expression: "target.hp <= 0"
 Effect    string `json:"effect"`    // Description or a second CEL script for resolution
}

```

---

### 2. Why this solves the complexity

By moving the stats into a `map[string]int`, you allow your **YAML** rules to stay readable. Instead of complex Go method calls, a rule for a **Rogue**'s *Sneak Attack* or a **Zombie**'s *Undead Fortitude* looks like natural language:

* **Undead Fortitude Check:** `actor.type == 'undead' && roll_result < (5 + damage_taken)`.
* **Sneak Attack Requirement:** `actor.stats.dex > 10 && target.is_flanked`.

### 3. Updated Antigravity Mission Brief

Give this to your agent to implement the "Hydration" logic that fills this struct from your files:

> **Mission: Entity Hydration & CEL Contextualization**
>
> 1. **Refactor:** Update the `Entity` struct to match the provided blueprint, ensuring `AbilityScores` are flattened into a `map[string]int` for easier CEL access.
> 2. **Hydrator:** Implement a `LoadEntity(path string)` function that reads your **zombie.yaml** or **fighter.yaml** and populates this struct.
> 3. **The Context Wrapper:** Create a `GetContext(actor, target, roll)` function that returns a `map[string]any`. This map is what we will pass to `cel.Eval()` so the rule engine can see both participants in an action.
> 4. **Verification:** Write a test where you load the **young-green-dragon.yaml** and use a CEL expression to verify if its `intelligence` is greater than 15.
>
>

---

## 8. Generic System Simulator

Yes, transitioning to a **CEL-powered rule engine** effectively transforms your project from a hard-coded D&D tool into a **Generic RPG System Simulator**. By decoupling the mechanics (the "How") from the code (the "Engine"), you’ve created a programmable platform where the rules of the game are treated as content, not infrastructure.

Here is how this architecture achieves "Generic System" status:

---

### 1. Schema-Driven World Building

Because your `models.Entity` and `Ability` structs are generic, they don't "care" if they are loading a D&D **Fighter** or a Cyberpunk Netrunner.

* **Plug-and-Play Stats:** You can redefine the `Stats` map in your YAML from "Strength/Dexterity" to "Cool/Reflexes" without changing a single line of Go code.
* **System-Agnostic Actions:** A `Longsword` attack in D&D and a `Laser Pistol` shot in a sci-fi setting use the same `ActionTriggered` event structure; only the CEL `Effect` string changes.

### 2. Universal Resolution Logic

The `EvalRule` helper function acts as a universal judge. As long as the system involves "Requirements" and "Results," Ancient Draconic can simulate it.

* **Dice Flexibility:** Since you’ve implemented a `MockDice` and dice-rolling utility, you can handle  systems (D&D),  pools (Shadowrun), or percentile systems (Call of Cthulhu) just by updating the strings in your YAML.
* **Logic Isolation:** Complex inter-system rules, like the **Zombie’s Undead Fortitude**, are isolated within that specific creature's file, preventing "Rule Leakage" into the rest of the simulator.

### 3. The "Simulator" Workflow

As a generic simulator, the engine’s primary job is to maintain the **Event Log** (The History) and the **State** (The Present).

1. **Input:** A DSL command like `attack :by Elara :to Zombie`.
2. **Simulation:** The engine looks at `Elara`'s YAML, evaluates the CEL conditions for her abilities (like **Sneak Attack**), and checks if the **Zombie** has a reaction.
3. **Output:** A new set of events is appended to the campaign's `.jsonl` file, representing the "New Reality" of the world.

---

## 9. Implementation for the Generic Simulator

To finalize this "Generic" capability, instruct your Antigravity agent to implement the **`EvalRule`** helper:

```go
// EvalRule executes a CEL expression against the actor/target context.
func (e *Engine) EvalRule(expression string, context map[string]any) (bool, error) {
    // 1. Compile the expression found in the YAML (e.g., zombie.yaml)
    ast, iss := e.celEnv.Compile(expression)
    if iss.Err() != nil { return false, iss.Err() }

    // 2. Run the program with the current combat state
    program, _ := e.celEnv.Program(ast)
    out, _, err := program.Eval(context)
    if err != nil { return false, err }

    // 3. Return the boolean result (e.g., "Does the attack hit?")
    return out.Value().(bool), nil
}

```

## 10. Yaml Template

To prove that **Ancient Draconic** is now a **Generic RPG System Simulator**, we can define a character from a completely different genre—like a **Sci-Fi Pilot**—using the exact same YAML structure as your **Fighter** or **Zombie**.

The engine doesn't care if the stat is "Strength" or "Piloting"; it only cares that the **CEL expression** in the `condition` field resolves against the data provided.

---

## 11. System-Neutral YAML: The Sci-Fi Pilot (`pilot.yaml`)

This template uses custom stats and a "Tech-based" ability to show how the system-neutral logic works.

```yaml
name: Jax "Static" Vane
index: jax-static
type: humanoid
size: Medium
alignment: Chaotic Good
hit_points: 10
stats:
  reflexes: 16
  cool: 14
  technical: 12
  intelligence: 10
  luck: 8
proficiencies:
  - proficiency:
      index: skill-piloting
      name: 'Skill: Piloting'
    value: 5
actions:
  - name: Blaster Pistol
    attack_bonus: 6
    damage:
      - damage_dice: 2d6+3
        damage_type:
          index: energy
          name: Energy
    desc: 'Ranged Tech Attack: +6 to hit, range 50/100 ft. Hit: 10 (2d6 + 3) energy damage.'
special_abilities:
  - name: Emergency Burn
    # Generic CEL logic: checks a custom stat 'luck' instead of D&D stats
    condition: "actor.stats.luck > 0 && target.type == 'starship'"
    effect: "actor.stats.luck -= 1; target.position += 10"
    desc: "Burn a point of Luck to increase your ship's position by 10 units."
updated_at: "2026-02-22T21:26:00Z"

```

---

## 12. Why the "Simulator" can handle this

Because you are using **CEL-Go** and a **map-based state**, your Go code remains unchanged while the game system shifts.

* **Custom Stats:** The `Stats` map in your `Entity` struct handles "Reflexes" just as easily as "Strength".
* **Genre-Agnostic Actions:** When the user types `attack :by jax-static :to pirate_drone`, the engine simply looks for the "Blaster Pistol" entry in the `actions` slice, regardless of whether it's a sword or a laser.
* **Arbitrary Resource Tracking:** The `Emergency Burn` ability demonstrates that you can track any numerical resource (like "Luck") using the same logic you used for **Spell Slots** or the **Dragon's Recharge**.

---

## 13. The "Generic Simulator" Mission Brief

Give this final specification to your **Antigravity** agent to solidify the "Generic" nature of the engine:

> **Mission: Validate Multi-System Support**
>
> 1. **Test Suite Addition:** Create a new test file `internal/engine/generic_test.go`.
> 2. **The Sci-Fi Test:** Load the `pilot.yaml` and a mock `starship.yaml`.
> 3. **Cross-Genre Command:** Execute a DSL command `cast :by jax-static :action Emergency_Burn` and verify that the `Luck` stat in the resulting `GameState` decrements correctly.
> 4. **Flexible Lexer:** Ensure the **Participle** lexer is not hard-coded to D&D terms. It should accept any action name found in the loaded YAML files.
>
>

---

## 14. System Manifest

### 1. The Campaign Manifest (`campaign.yaml`)

This file sits at the root of your campaign directory. It allows the engine to pivot its entire logic—from a high-fantasy D&D setting to a grim-dark sci-fi world—without requiring a code change.

```yaml
name: "The Neon Horizon"
system: "Cyber-Sagas v1.0"
data_dir: "./data/sci-fi"
active_entities:
  - pilot.yaml
  - pirate_drone.yaml
# System-wide global variables
globals:
  gravity: 1.0
  radiation_level: low
# Default resolution logic for this system
resolution_logic:
  hit_formula: "actor.stats.reflexes + roll('1d20') >= target.stats.defense"
  damage_formula: "roll(action.damage_dice) - target.stats.armor"

```

---

### 2. Dynamic Rule Resolution via Manifests

With the manifest in place, your **CEL-Go** engine becomes truly context-aware. Instead of looking for a `dexterity` stat by name, it can use the `hit_formula` defined in the manifest to resolve actions.

* **Logic Swapping:** When you load a D&D campaign, the engine uses the `d20 + modifier` formula. When you load "The Neon Horizon," it automatically switches to the `reflexes + d20` formula defined in the manifest.
* **System-Specific Constraints:** You can add a `globals` map to the manifest to track world-wide effects, like "Gravity," which your **CEL** rules can then reference (e.g., `if (gravity > 1.0) { jump_height / 2 }`).
* **Encapsulated Growth:** To build a new RPG system, a user simply creates a new folder with its own `campaign.yaml` and a set of **YAML** entities; the engine handles the rest.

---

### 3. Final Antigravity Mission Brief: The System Manifest

This is the final "Core Architecture" task for your agent. Once this is done, you have a platform that can simulate any tabletop game ever written.

> **Mission: Implementation of the System Manifest & Global State**
>
> 1. **Manifest Struct:** Create a `CampaignManifest` struct that includes `DataDir`, `ResolutionLogic`, and a `Globals` map.
> 2. **Manifest Loader:** Update the `dnd campaign load` command to first read this manifest and configure the **CEL-Go** environment accordingly.
> 3. **Dynamic Hit Resolution:** Refactor the `attack` logic to use the `hit_formula` from the manifest instead of a hard-coded "d20 + bonus" check.
> 4. **Integration Test:** Create a "Gravity Test" where a jump action succeeds in a "Low Gravity" campaign but fails in a "High Gravity" campaign using the same **YAML** character.
>
>

---

## 15. Commands

To handle wildly different command sets across RPG systems, the engine must move away from a fixed list of "Verbs" (like `attack` or `cast`) and instead treat **Commands as Registered Actions** within the System Manifest.

In this design, the **Parser** remains generic, but the **Resolution Logic** is mapped dynamically based on the current campaign's configuration.

---

### 1. Action-Based Command Mapping

Instead of the Go code defining what `attack` means, the **System Manifest** (`campaign.yaml`) defines a mapping between a **Verb** and a **Resolution Template**.

* **D&D 5e System:** Maps the `attack` verb to a "Targeted Roll" template that compares an attack roll against an Armor Class (AC).
* **Sci-Fi System:** Might map a `hack` verb to a "Skill vs. Difficulty" template that uses a `technical` stat against a firewall's `rating`.
* **Narrative System:** Could map a `persuade` verb to a simple "Success/Failure" roll without any target stats at all.

---

### 2. The "Action Template" Structure

To make this work, we introduce **Action Templates** in the manifest. These templates tell the engine which **CEL-Go** expressions to run when a specific verb is typed in the CLI.

```yaml
# In campaign.yaml
commands:
  - verb: "attack"
    template: "targeted_resolution"
    logic:
      roll: "actor.stats.str + roll('1d20')"
      threshold: "target.armor_class"
      on_success: "target.hp -= action.damage"
      
  - verb: "hack"
    template: "skill_check"
    logic:
      roll: "actor.stats.technical + roll('2d6')"
      threshold: "target.stats.firewall"
      on_success: "target.conditions.add('glitched')"

```

---

### 3. Handling Different Prepositions

Since you are using **Participle** for your DSL, we can make the prepositions (like `:to`, `:with`, `:using`) part of the Action Template.

* **D&D:** Uses `attack [Target] :with [Weapon]`.
* **Sci-Fi:** Might use `hack [System] :using [Program]`.

The engine uses the template to "validate" the incoming DSL string. If the user types `hack :with sword`, the engine checks the `hack` template, sees it requires `:using`, and returns an error message: *"Invalid argument: 'hack' requires ':using [program]'"*.

---

### 4. Antigravity Mission: Dynamic Command Dispatch

This is the "Final Boss" of the architecture refactor. Once implemented, your engine is no longer a D&D tool—it is a **Universal Command Processor**.

> **Mission: Implement Dynamic Action Templates**
>
> 1. **Registry:** Create an `ActionRegistry` that loads the `commands` section from the `campaign.yaml`.
> 2. **Parser Update:** Update the **Participle** grammar to accept *any* initial word as a `Verb`, then look up that verb in the Registry.
> 3. **Template Resolver:** Create a generic `ExecuteAction` function that:
>
> * Loads the CEL logic for the chosen verb.
> * Binds the `:labels` from the DSL to variables in the CEL context (e.g., `:with` becomes `action.item`).
> * Appends the resulting changes to the **Event Log**.
>
>
> 1. **Verification:** Verify that you can successfully run an `attack` in the D&D campaign and a `hack` in the Sci-Fi campaign without changing the Go source code.
>
>

---
