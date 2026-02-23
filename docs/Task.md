# DnDSL Master Task List

| Phase | Description | Status |
|---|---|---|
| 01 | Project Description | [x] |
| 02 | Init Command | [x] |
| 03 | Persistence Layer | [x] |
| 04 | Game State Engine | [x] |
| 05 | Roll Command | [x] |
| 06 | Messaging & Engine Refactor | [x] |
| 07 | Encounter Commands | [x] |
| 08 | Initiative Rolls | [x] |
| 09 | Hierarchical Data Resolver | [x] |
| 10 | Attack/Damage/Turn Commands | [x] |
| 11 | Ask & Check Commands | [x] |
| 13 | TUI / REPL Shell | [x] |
| 14 | Damage Types & Defenses | [x] |
| 15 | Friendly Syntax Errors | [x] |
| 16 | Context-Aware Help | [x] |
| 17 | Telegram Bot Integration | [x] |
| 18 | Final Documentation & README | [x] |
| 20 | Syntax Refactor (Prepositions) | [x] |
| 21 | Justfile Coverage Task | [x] |
| 22 | SRD Data Internalization (Embedding) | [x] |
| 23 | Debug: Help Command Panic | [x] |
| 24 | Refactor: Valid Module Path | [x] |
| 25 | Refactor: TUI Autocomplete Refinement | [x] |
| 26 | Refactor: Syntax Cleanup in Docs and Strings | [x] |
| 27 | Refactor: Move dnd5eapi to internal/ | [x] |
| 28 | Refactor: TUI Layout & Autocomplete UX | [x] |
| 29 | Mechanics: Adjudication System (Freeze/Allow/Deny) | [x] |
| 30 | Mechanics: Action Economy (1 Action, 1 Bonus, 1 Reaction) | [x] |
| 31 | Mechanics: New Actions (Dodge, Grapple, Help) | [x] |
| 32 | Mechanics: Monster Recharge Logic | [x] |
| 33 | Remaining Actions: Dash, Disengage, Hide, Improvise, Influence, Ready, Search, Shove, Study, Utilize | [x] |
| 34 | Bonus Actions: Two Weapon Fighting | [x] |
| 35 | Reaction: Opportunity Attack | [x] |
| 36 | Cast Action | [ ] |

---

- [x] **Phase 28: Refactor: TUI Layout & Autocomplete UX**
- [x] **Phase 29: Mechanics: Adjudication System (Freeze/Allow/Deny)**
  - [x] Update `GameState` to support `PendingAdjudication` and system freeze
  - [x] Implement `adjudicate` command logic
  - [x] Implement `allow by: GM` and `deny by: GM` commands
  - [x] Update `Session.Execute` to enforce freeze
- [x] **Phase 30: Mechanics: Action Economy (1 Action, 1 Bonus, 1 Reaction)**
  - [x] Track action usage in `Entity` state
  - [x] Enforce usage limits in command execution
  - [x] Implement turn-start/end reset logic
- [x] **Phase 31: Mechanics: New Actions (Dodge, Grapple, Help)**
  - [x] Implement `dodge` command with condition effects
  - [x] Implement `grapple` command with targeted save and conditions
  - [x] Implement `help` command with check/attack advantage
  - [x] Integrate all with Adjudication system
- [x] **Phase 32: Mechanics: Monster Recharge Logic**
  - [x] Add `Recharge` field to `Monster` action schema
  - [x] Implement turn-start recharge roll logic
  - [x] Update engine to track spent recharge abilities
  - [x] Verify with integration tests
- [x] **Phase 33: Remaining Actions: Dash, Disengage, Hide, Improvise, Influence, Ready, Search, Shove, Study, Utilize**
  - [x] Implement standard actions with logging in `generic_action.go`
  - [x] Implement `Shove` with size restrictions and dynamic Saving Throw DC
  - [x] Add `Race` and `Size` support to data layer
  - [x] Update turn-reset logic for `Disengaged` condition
  - [x] Verify all actions with integration tests
- [x] **Phase 34: Bonus Actions: Two Weapon Fighting**
  - [x] Implement `attack off-hand` syntax
  - [x] Implement damage modifier stripping for off-hand attacks
  - [x] Enforce 5e rules: must have attacked previously this turn with a different weapon
  - [x] Enforce strict syntax order (`attack by: actor off-hand`)
- [x] **Phase 35: Reaction: Opportunity Attack**
  - [x] Implement `attack opportunity` syntax
  - [x] Integrate with Adjudication for GM approval
  - [x] Enforce reaction economy

---

## Future Goals

- [ ] Automatic Spell Slot Management
- [ ] Local Web Dashboard (Vite + React)
- [ ] Multi-platform GoReleaser parity
