# Plan - Phase 9: Utility and Encounter Migration

I will migrate the remaining hardcoded utility and encounter commands to the manifest-driven engine.

## Proposed Changes

### [Rules Bridge] [bridge.go](file:///home/paulo/org/projetos/dndsl/internal/rules/bridge.go)

- [MODIFY] Expose `state.PendingChecks` to the CEL context.

### [Manifest] [manifest.yaml](file:///home/paulo/org/projetos/dndsl/data/manifest.yaml)

- [NEW] Implement `help_action` (supports adjudication).
- [NEW] Implement `ask` (GM command to request rolls).
- [NEW] Implement `add` (adding actors to active encounter).
- [NEW] Implement `encounter` (start/end encounters).
- [MODIFY] Update `check` to pull DC from `pending_checks` if available.

### [Command Engine] [executor.go](file:///home/paulo/org/projetos/dndsl/internal/command/executor.go)

- [MODIFY] Update `mapManifestEvent` to handle new events.

### [Session] [session.go](file:///home/paulo/org/projetos/dndsl/internal/session/session.go)

- [MODIFY] Route all migrated commands.

## Verification Plan

- `go test -v ./internal/command/...`
- Manually verify dynamic DC logic.
