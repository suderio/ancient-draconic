# Code Patterns and Implementation Standards

## Purpose

Define implementation patterns that keep behavior predictable, testable, and maintainable.

## Normative Rules

1. Global engineering and organization policies in `AGENTS.md` apply to all code changes.
2. Functions MUST prefer explicit inputs/outputs over implicit global state.
3. Side effects MUST be isolated behind interfaces declared in owner packages.
4. Error paths MUST preserve root cause context.
5. Secret material MUST never be logged or included in error messages.
6. Go sources MUST be `gofmt`-formatted and follow idiomatic package/export conventions.
7. Inline comments that only restate what the code already expresses MUST NOT be introduced; contributors MUST improve naming, structure, or tests to capture intent. Comments SHOULD only exist for exported-API documentation or compile-time directives.

## Lua Engine Conventions

### Sandbox Rules

1. Allowed Lua libraries: `base`, `table`, `string`, `math`.
2. Forbidden libraries: `os`, `io`, `debug`, `package` (beyond controlled `require`).
3. Go-bridged functions: `roll(dice)`, `mod(val)`, `help(topic)`.
4. All bridged functions MUST be registered at `LState` creation in `NewLuaEvaluator()`.

### Formula Patterns

1. Simple formulas SHOULD be strings: `formula = "not is_encounter_active"`.
2. Complex formulas (multi-step, table construction, conditionals) SHOULD be closures: `formula = function() ... end`.
3. The evaluator MUST handle both: type-switch on `string` → `L.DoString("return " .. formula)`, `*lua.LFunction` → `L.CallByParam()`.
4. The Go `GameStep.Formula` field is `any` to accommodate both types.

### Context Injection

Before each formula evaluation, the engine MUST set these Lua globals:

| Global | Type | Source |
| :--- | :--- | :--- |
| `actor` | table | Current acting entity |
| `target` | table | Current target entity (or `nil`) |
| `command` | table | Parsed command parameters |
| `game` | table | Results from game-phase steps |
| `targets` | table | Results from targets-phase steps |
| `actor_results` | table | Results from actor-phase steps |
| `is_<command>_active` | bool | Loop state flags |

### Type Conversion

1. `goValueToLua(any) LValue`: converts Go maps/slices/primitives → Lua tables/values.
2. `luaValueToGo(LValue) any`: converts Lua results → Go values for event construction.
3. `LNumber` → `int` (when integral) or `float64`.
4. `LTable` with sequential integer keys → `[]any`; otherwise → `map[string]any`.

### LState Lifecycle

1. One `LState` per `Session`.
2. `manifest.lua` is loaded once at session start via `L.DoFile()`.
3. Formula evaluations reuse the same `LState` with globals reset before each call.
4. Thread safety is provided by `sync.Mutex` on `Session.Execute()`.

### Error Handling

1. Lua `*ApiError` MUST be wrapped with formula context: step name, command name, phase.
2. Error messages MUST include Lua line numbers when available.
3. `nil` access in formulas MUST produce a clear error naming the missing variable.

## File Design Guidance

1. Co-locate contract/validator/mapper logic when change cadence is shared.
2. Separate CLI parsing/validation from execution side effects.
3. Keep boundary types in owner packages.

## Failure Modes

1. Hidden coupling through shared mutable state.
2. Mixed-concern files that combine CLI, implementation and integration logic.
3. Non-deterministic tests due to unstable ordering.
4. Error wrappers that drop actionable context.
5. Lua sandbox escape via unblocked standard library.
6. Formula evaluation leaking state between calls due to uncleared globals.
