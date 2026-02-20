# Project Description

This project will build a DnD engine and DSL to allow playing within the DnD rules.

The System Reference Document is in docs/SRD_CC_v5.1.pdf and is the ultimate reference of the rules.

## Mission Description

### **1. Objective**

Initialize a Go-based engine and CLI for a D&D 5e DSL. The project uses **Cobra** for command-line management, **Participle** for parsing a stateless DSL, and a foundational **Event Sourcing** model.

### **2. CLI Structure (Cobra)**

Implement a root command `dndsl` with the following sub-commands:

* **`repl`**: An interactive shell that accepts DSL strings, processes them into events, and prints the updated **Event Log** and **Calculated State** after every entry.
* **`start`**: (Placeholder) Initialize a new game session.
* **Global Flags**: `--debug` to show the raw Participle AST.

### **3. DSL Syntax Specification**

The DSL must support an optional `:by` delimiter for the actor to improve readability in complex commands.

**Standard Syntax:** `[Action] [Actor] [Argument...] [:Label Argument...]*`
**Explicit Actor Syntax:** `[Action] :by [Actor] [Argument...] [:Label Argument...]*`

**Examples to Support:**

* `attack :by Grog Goblin1 :with Greataxe`
* `cast :by Melf Protection from Evil and Good :to Ebenezer Scrooge`
* `roll :by Paulo 1d20 + 5`
* `undo`

#### Commented example (to be used in planning the implementation)

```text
encounter :by GM :with Giant Crocodile :and Giant Crocodile # start encounter with two giant-crocodile.yaml
# since giant-crocodile are 'monsters' not 'characters', run initiative for them
> Rolled initiative for giant-crocodile_1: 13
> Rolled initiative for giant-crocodile_2: 18
attack :by giant-crocodile_2 :with bite :to giant-crocodile_1
> giant_crocodile_2 hits giant_crocodile_1 with 25! Roll damage.
damage :by giant-crocodile_2 :with bite
> giant_crocodile_2 damages giant_crocodile_1 with 21 HP!
ask :by GM :check athletics :of giant_crocodile_1 :dc 16 :fails grappled # if giant_crocodile_1 fails, it has grappled condition
> GM has asked giant_crocodile_1 to make an athletics check
check :by giant_crocodile_1 athletics
> giant_crocodile_1 passes athletics check with 16!
attack :by giant-crocodile_2 :with tail :to giant-crocodile_1
> giant_crocodile_2 hits giant_crocodile_1 with 20! Roll damage.
damage :by giant-crocodile_2 :with bite
> giant_crocodile_2 damages giant_crocodile_1 with 18 HP!
turn :by giant_crocodile_2
> giant_crocodile_2 ended its turn. Now it is giant-crocodile_1.
# ... follows the same thing from giant-crocodile_1
turn :by giant_crocodile_1
> giant_crocodile_1 ended its turn. Starting a new round with giant_crocodile_2. # Since there are no more characteres, just two, this ends the round.
# ... the same thing until the GM ends the encounter
encounter :by GM end
> This encounter has ended.
```
These are examples, they are not to be implemented right away. The important part is that, beyond the mechanics of an encounter, all the information for the encounter resolution *must* be taken from the relevant yaml files in the data directory. In this example, the giant-crocodile.yaml file shows that the creature has multiattack, that one is a bite attack and the other is a tail attack. It also shows the Hit Points and Armor Class.

Optional decisions, like the one in the description of the bite attack, are to be adjucated by the GM. In this instance, the `ask` command for an athletics check was a decision of the GM, since these details cannot be properly read from the yaml files. The information that an athletics check is done with the Strenght attribute comes from the skills/athletics.yaml.

That does not mean everything is set and done in the yaml files. They may need some enrichment to allow correct interpretation of the game mechanics, but they are the starting point for the game resolution.

### **4. Technical Constraints**

1. **Lexer/Parser:** Use `participle/v2`. Configure the lexer to treat words starting with `:` as `PREP` tokens.
2. **Actor Resolution:** The parser must handle the actor name whether it follows the verb directly or follows the `:by` keyword.
3. **Event Sourcing:**

* Commands must not mutate state directly.
* `Engine.Process(cmd)` returns `[]Event`.
* `Store.Append(events)` adds to the log.
* `Projector.Build(log)` returns the current `GameState`.

### **5. Required Artifacts**

* **`cmd/dnd/main.go`**: Entry point using Cobra.
* **`internal/parser/`**: Participle grammar and Lexer logic.
* **`internal/engine/`**: `Event` interfaces and `GameState` projections.
* **`internal/repl/`**: The interactive loop logic for the `repl` command.

### **6. Implementation Steps for Agent**

1. **Initialize:** `go mod init dndsl` and `cobra-cli init`.
2. **Grammar:** Build the Participle struct. Use a slice for `Actor` or a specific `@PREP "by" Ident` capture to handle the `:by` logic.
3. **REPL:** Use a simple `bufio.Scanner` loop for the `repl` command.

Right now we will have only one command:

```text
Roll :by Someone1 3d6 + 1
> Someone1 rolled 12
```

This command rolls dice using the standard syntax for dice rolling. The dicing rolling should be implemented or added as a library, if a convenient one is found (preferable).

Important: the commands, prepositions (like :by) and parameters must be all case insensitive. :by Someone == :By someone.
