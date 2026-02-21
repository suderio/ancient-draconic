# Accurate Initiative Rolls Plan

## Overview

Currently, the `initiative` command and the automated monster initiative rolls in the `encounter` and `add` commands use a hardcoded default (e.g. `1d20+1` or `1d20+2`). The user requested that we calculate the genuine D&D 5e initiative roll, which is `1d20 + Dex Modifier`.

## Data Model Updates

To calculate this, the engine needs to parse the `Dexterity` attribute from the YAML definitions inside `data/characters/` and `data/monsters/`.

The `internal/data/models.go` file currently defines a generic `Monster` struct with attributes like `Dexterity int`. However, there is no `Character` struct yet.

We will:

1. Define a `Character` struct in `internal/data/models.go` that includes standard stats (or at least `Dexterity`).
2. Populate the `data/characters/paulo.yaml` and a second test character with a `dexterity` field.
3. Populate `data/monsters/goblin.yaml` with a `dexterity` field.

## Calculating the Modifier

In D&D 5e, the ability modifier is calculated as:
`(AbilityScore - 10) / 2` (rounded down).

We will create a helper function `CalculateModifier(score int) int` inside `internal/data/models.go` or a relevant utility file.

## Updating the Command Logic

The `CheckEntityLocally` function inside `internal/command/utils.go` currently only checks for file existence using `os.Stat`. We need to upgrade this function to actually unmarshal the YAML file and extract the `Dexterity` score.

`TargetRes` will be updated to carry the modifier:

```go
type TargetRes struct {
 Type     string // 'Character' | 'Monster'
 Name     string
 InitiativeMod int
}
```

The commands `ExecuteEncounter`, `ExecuteAdd`, and `ExecuteInitiative` will be updated to construct the strict parsing string dynamically:
`fmt.Sprintf("1d20%+d", res.InitiativeMod)`

## Action Plan

1. Update `models.go` with `Character` struct and `CalculateModifier` helper.
2. Update the dummy YAML files in `data/characters/` and `data/monsters/`.
3. Update `CheckEntityLocally` to decode the structs and compute `InitiativeMod`.
4. Replace hardcoded strings in `encounter.go`, `add.go`, and `initiative.go`.
5. Run unittests and verify via REPL.
