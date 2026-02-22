# Plan: Dynamic Escape DC

## Goal

Ensure the escape DC for a grapple matches the grappler's passive Save DC.

## Proposed Changes

- Store grappler ID in the condition: `grappledby:ID`.
- Update `escape` action to look up the grappler and calculate the correct DC.

## Verification

- Test case verifying Thorne's escape DC for a grappled target.
