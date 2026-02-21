# Phase 14: Damage Types and Defenses

Implement native support for damage types (fire, poison, etc.) and entity defenses (resistance, immunity, vulnerability).

## Proposed Changes

### 1. internal/data/models.go

- **[MODIFY]**: Add `Defense` struct and include it in `Character` and `Monster` definitions.
 - `Resistances []string`
 - `Immunities []string`
 - `Vulnerabilities []string`

### 2. internal/parser/ast.go

- **[MODIFY]**: Update `DamageCmd` to support multiple damage rolls.
 - `Rolls []*DamageRollExpr`
 - `DamageRollExpr` contains both `Dice` and optional `Type`.

### 3. internal/command/damage.go

- **[MODIFY]**: Update `ExecuteDamage` logic.
 - Traverse all provided damage instances.
 - Cross-reference the target's defenses based on the damage type.
 - Apply multipliers: x0 for immunity, x0.5 for resistance, x2 for vulnerability.
 - Sum the total damage after all modifiers are applied.

## Verification Plan

### Automated Tests

- `internal/command/damage_test.go`: Add `TestExecuteDamageWithDefenses` to verify modified arithmetic for all three defense types.
- Verify multiple damage types in a single command (e.g., `dice: 2d6 type: fire dice: 1d4 type: piercing`).
