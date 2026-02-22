# Plan: Fix Grapple Bugs

## Goal

Fix issues with grapple resistance saving throws and escape condition removal.

## Proposed Changes

- **Engine**: Update `RollConsequence` to support `RemoveCondition`.
- **Logic**: Update `ExecuteCheck` to handle condition removal.
- **Grapple**: Update `ExecuteGrapple` to ask for Saving Throws.
- **Escape**: Implement a new `escape` command/action.

## Verification

- Reproduction test cases in `internal/command/grapple_bug_test.go`.
