---
trigger: always_on
description: Maintain project documentation in docs/
---

# Project Documentation Rules

1. **Task List**:
    - The master task list is located at `docs/Task.md`.
    - Always update this file when tasks are completed or new tasks are added.
    - Do not rely solely on the ephemeral `task.md` artifact.

2. **Implementation Plans**:
    - When creating a new Implementation Plan, always save it as a markdown file in the `docs/` directory.
    - Use a descriptive naming convention: `docs/<phase>_<description>_plan.md` (e.g., `docs/plan_phase_5_refactoring.md`).
    - This ensures a permanent record of design decisions and plans.
3. **Code Documentation**:
    - Every code generation task must include or update code documentation (GoDoc).
    - Documentation must reflect:
        - What the code is doing.
        - Design decisions and rationale.
        - TODOs and future improvements.
        - Implementation details where complex.
    - Ensure comments are up-to-date with code changes.
