# Rule Evaluation Bridge (Phase 3)

This phase establishes the connection between the generic engine state and the CEL (Common Expression Language) registry. It enables the system to evaluate dynamic formulas defined in YAML data files instead of relying on hard-coded Go logic.

## Proposed Changes

### [internal/rules]

#### [NEW] [bridge.go](file:///home/paulo/org/projetos/dndsl/internal/rules/bridge.go)

- Implement `ContextFromEntity(entity *engine.Entity) map[string]any` to convert `engine.Entity` to a CEL-compatible map.
- Implement helper functions to build the global context (Actor, Target, Action).

#### [MODIFY] [env.go](file:///home/paulo/org/projetos/dndsl/internal/rules/env.go)

- Add or refine methods to simplify rule evaluation from commands.

### [internal/command]

#### [MODIFY] [attack.go](file:///home/paulo/org/projetos/dndsl/internal/command/attack.go)

- Use CEL to determine hit/miss if a formula is provided in the weapon data.
- Bind `actor` and `target` to the CEL environment.

#### [MODIFY] [check.go](file:///home/paulo/org/projetos/dndsl/internal/command/check.go)

- Use CEL to calculate modifiers or success/failure.

## Verification Plan

### Automated Tests

- `go test ./internal/rules/...` to verify mapping and basic evaluation.
- Update `internal/command/mechanics_test.go` to include scenarios with CEL-based weapons.

### Manual Verification

- Execute an attack command with a weapon that specifies a CEL formula (e.g., `hit_rule: "actor.stats.str + roll('1d20') >= target.stats.ac"`) if applicable.
