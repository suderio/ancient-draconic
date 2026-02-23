# Phase 7: Migrating Grapple and Shove to Generic Engine

Migrate `grapple` and `shove` commands to the manifest-driven engine, ensuring all existing tests pass and legacy handlers are removed.

## Proposed Changes

### [Component] Data & Manifest

#### [MODIFY] [manifest.yaml](file:///home/paulo/org/projetos/dndsl/data/manifest.yaml)

- Add `grapple` command definition:
  - Step 1: Check if already pending adjudication.
  - Step 2: Trigger `AdjudicationStarted` to get GM approval.
  - Step 3 (post-approval): Trigger `GrappleTaken`.
  - Step 4: Trigger `AskIssued` for the contested save (DC = 8 + Str mod + Prof).
- Add `shove` command definition:
  - Step 1: Validate sizing using `size_rank`.
  - Step 2: Consume action.
  - Step 3: Trigger `Hint` message.
  - Step 4: Trigger `AskIssued` for the save.

### [Component] Generic Command Engine

#### [MODIFY] [executor.go](file:///home/paulo/org/projetos/dndsl/internal/command/executor.go)

- Expand `mapManifestEvent` to support:
  - `AdjudicationStarted`
  - `GrappleTaken`
  - `AskIssued` (mapping `dc`, `check`, `fails`, `succeeds`)
  - `Hint`
- Update `ExecuteGenericCommand` to handle multi-stage execution for grapple (adjudication -> resolution).

#### [MODIFY] [bridge.go](file:///home/paulo/org/projetos/dndsl/internal/rules/bridge.go)

- Expose `size` and `is_frozen` to the CEL context.
- Ensure `state` metadata like `pending_adjudication` is accessible.

#### [MODIFY] [env.go](file:///home/paulo/org/projetos/dndsl/internal/rules/env.go)

- Add `size_rank(string)` function to CEL to facilitate size-based logic in the manifest.

### [Component] Session & Cleanup

#### [MODIFY] [session.go](file:///home/paulo/org/projetos/dndsl/internal/session/session.go)

- Route `astCmd.Grapple` and `astCmd.Shove` to `command.ExecuteGenericCommand`.

#### [DELETE] [grapple.go](file:///home/paulo/org/projetos/dndsl/internal/command/grapple.go)

#### [DELETE] [shove.go](file:///home/paulo/org/projetos/dndsl/internal/command/shove.go)

## Verification Plan

### Automated Tests

- `go test -v ./internal/command/mechanics_test.go`
- Specifically verify `TestAdjudicationFlow` (Grapple) and `TestExecuteShove`.

### Manual Verification

- N/A
