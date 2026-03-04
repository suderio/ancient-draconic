# Testing, Quality, and Security

## Purpose

Define quality gates and security invariants so behavior changes are verifiable and safe.

## In Scope

1. Test strategy by risk level.
2. Security and safety controls.
3. Required regression/acceptance coverage.
4. Release-readiness checks.

## Out of Scope

1. CI vendor configuration.
2. Runtime observability platform setup.
3. UI style concerns.

## Normative Rules

1. Every behavior change MUST add tests at the lowest effective layer.
2. High-risk orchestration or integration changes MUST include integration coverage.
3. CLI contract changes MUST include command-level success and failure tests.
4. Security-sensitive flows MUST include negative tests.
5. Deterministic ordering guarantees MUST be asserted.
6. Path traversal protections MUST be tested for repository and metadata access.
7. Secret values MUST never appear in logs, errors, or snapshots.
8. New normative rules SHOULD include an explicit matching test expectation.

## Data Contracts

Test layers:

1. Unit: pure transforms, normalization, metadata layering/template rendering, secret placeholder normalization.
2. Integration: workflows with fake endpoints.
3. E2E: CLI workflows using representative stacks and fixture trees.

Acceptance contracts:

1. Idempotency for repeated apply.
2. Typed error categories for all major failure classes.

## Required Scenario Coverage

1. CLI safeguards: validation errors, conflicting path inputs, and destructive-operation protections.

## Failure Modes

1. Tests pass locally with hidden non-determinism.
2. Changed behavior lacks regression coverage.
3. Security-sensitive paths bypass required safeguards.
4. Snapshot/log artifacts leak secret values.

## Lua Engine Testing

1. Sandbox isolation MUST be tested: verify `os`, `io`, `debug`, `loadfile` are inaccessible from formulas.
2. Formula error propagation MUST be tested: Lua syntax errors, nil-access, and type mismatches must produce structured errors with step/command names.
3. LState thread safety MUST be verified with `go test -race ./...`.
4. Event log replay consistency MUST be tested: persist events → rebuild state from log → verify state matches.
5. Both formula types (string and closure) MUST have coverage in executor tests.
6. Context injection MUST be tested: verify `actor`, `target`, `command`, `game`, `is_*_active` globals are correctly set and cleared between calls.
