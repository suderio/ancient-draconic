# Plan: Phase 33 Standard Actions

## Goal

Implement missing standard actions (Dash, Disengage, Hide, etc.) and handle the "Shove" mechanic with size restrictions and saving throws.

## Proposed Changes

- Update `ExecuteAction` for simple actions and "Disengage".
- Implement `ExecuteShove` with size comparison and DC calculation.
- Add `Race` and `LoadRace` to the data layer.
- Add `Size` utility for categorical comparisons.
- Update turn logic to clear "Disengaged".

## Verification

- Tests for Shove (success/fail size restricted).
- Tests for Disengage condition.
