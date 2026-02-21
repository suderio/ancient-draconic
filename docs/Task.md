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

- [x] **Phase 22: SRD Data Internalization (Embedding)**
- [/] **Phase 23: Debug: Help Command Panic**
  - [ ] Reproduce panic in test/REPL
  - [ ] Fix index out of range in `internal/command/help.go`
  - [ ] Improve `IsFrozen()` logic in `internal/engine/state.go` (if needed)
  - [ ] Verify fix in REPL
| 22 | SRD Data Internalization (Embedding) | [x] |
| 23 | Debug: Help Command Panic | [/] |

---

## Future Goals

- [ ] Automatic Spell Slot Management
- [ ] Local Web Dashboard (Vite + React)
- [ ] Monster Recharge Logic
- [ ] Multi-platform GoReleaser parity
