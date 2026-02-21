# DnDSL

<div align="center">

![DnDSL Logo](assets/logo.png)

**The ultimate command-line engine for D&D 5e encounters.**

[![Go Report Card](https://goreportcard.com/badge/github.com/suderio/dndsl)](https://goreportcard.com/report/github.com/suderio/dndsl)
[![GoDoc](https://godoc.org/github.com/suderio/dndsl?status.svg)](https://godoc.org/github.com/suderio/dndsl)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Stability: Stable](https://img.shields.io/badge/Stability-Stable-green.svg)](#)

*Process combat, roll dice, and manage campaigns through a powerful, human-readable Domain Specific Language.*

[Explore the Docs](docs/01_project_description.md) · [Report Bug](https://github.com/suderio/dndsl/issues) · [Request Feature](https://github.com/suderio/dndsl/issues)

</div>

---

## 📖 Introduction

**DnDSL** is a developer-centric tool for Dungeons & Dragons Game Masters. It provides a structured, terminal-based interface (powered by Bubble Tea) to manage complex combat encounters. By using an event-sourced architecture, every roll, hit, and turn is recorded in an immutable log, ensuring your campaign's history is safe and fully reproducible.

### Why DnDSL?

- **Speed**: No more clicking through complex UIs. Type `attack with: Longsword to: Orc_1` and let the engine handle the rest.
- **Precision**: Automated damage resistance, immunity, and vulnerability calculations.
- **Remote-First**: Built-in Telegram bot integration allows players to roll from their phones while the GM runs the engine on a server.
- **Customizable**: Purely YAML-based data layer. Adding a new monster or character is as simple as creating a file.

---

## 🎨 Visuals

### Architecture Overview

The engine follows a strict Event Sourcing pattern to maintain high reliability and state predictability.

```mermaid
graph TD
    A[CLI / TUI REPL] --> B[Session Manager]
    T[Telegram Bot] --> B
    B --> C[Parser / Lexer]
    C --> D[Command Executor]
    D --> E[Event Log]
    E --> F[State Projector]
    F --> G[GameState]
    B --> G
```

### The TUI in Action

*(Placeholder: Upload a GIF of the Bubble Tea TUI here)*
> [!TIP]
> Use the `tab` key in the REPL for intelligent, context-aware autocomplete of characters, weapons, and targets!

---

## 🛠 Installation

### Prerequisites

- Go 1.25 or higher.

### From Source

```bash
git clone https://github.com/suderio/dndsl.git
cd dndsl
go build -o dndsl main.go
```

### Setup Your First Campaign

```bash
# Create a new world and campaign
./dndsl campaign create "SwordCoast" "LostMine"

# Add some participants
./dndsl add elara and: thorne

# Start the interactive REPL
./dndsl repl SwordCoast LostMine
```

---

## ⚔️ DSL Usage Examples

The language is designed to be self-documenting. Use the `help` command for contextual guidance.

**Roll for Initiative:**

```bash
initiative
```

**Attack and Damage:**

```bash
attack with: Longsword to: Goblin_A
damage with: Longsword dice: 1d8+3 type: slashing
```

**Universal Checks:**

```bash
ask check: athletics of: Elara dc: 15 fails: prone
check athletics
```

---

## 📱 Telegram Integration

Play from anywhere! Configure a bot to let your players interact with the campaign via Telegram.

1. **Register Bot**: `dndsl bot telegram --token YOUR_TOKEN`
2. **Link Campaign**: `dndsl campaign telegram [world] [campaign] --chat_id YOUR_CHAT_ID`
3. **Map Users**: `dndsl campaign telegram [world] [campaign] --user Elara:123456`
4. **Play**: Simply run the `repl` and your bot will start polling for messages starting with `/`.

---

## 🗺 Roadmap

- [x] Context-aware Autocomplete (REPL)
- [x] Damage Defenses (Resistance/Immunity)
- [x] Telegram Bot Polling
- [ ] Automatic Spell Slot Management
- [ ] Local Web Dashboard (Vite + React)
- [ ] Monster Recharge Logic

---

## 🤝 Contributing

Contributions are what make the open-source community such an amazing place to learn, inspire, and create. Any contributions you make are **greatly appreciated**.

Please see [CONTRIBUTING.md](CONTRIBUTING.md) for details on our code of conduct and the process for submitting pull requests.

---

## 📊 Project Stats & Activity (Dynamic Recommendations)

To maintain a vibrant profile, we recommend integrating the following in your GitHub README:

1. **[GitHub Readme Stats](https://github.com/anuraghazra/github-readme-stats)**: Display your most-used languages and overall activity levels.
2. **[WakaTime Weekly Stats](https://github.com/athul/waka-readme-stats)**: Show exactly how much time is spent crafting the DSL engine logic.
3. **[Contributor Faces](https://github.com/lineone/contributor-faces)**: Automatically display avatars of the brave souls contributing to the code.

---

## 📜 License

Distributed under the MIT License. See [LICENSE](LICENSE) for more information.

<div align="center">
  Built with ❤️ by the DnDSL Team.
</div>
