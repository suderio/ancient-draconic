# Plan: Dynamic Grapple DC

## Goal

Calculate grapple DC dynamically using `8 + Strength Modifier + Proficiency Bonus`.

## Proposed Changes

- Update `ExecuteGrapple` to use `data.Loader`.
- Fetch attacker's Strength and Proficiency Bonus.
- Calculate DC accordingly.

## Verification

- Unit test in `internal/command/mechanics_test.go` verifying DC calculation for specific stats.
