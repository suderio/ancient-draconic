# DnDSL

<div align="center">

![DnDSL Logo](assets/logo.png)

**The ultimate command-line engine for D&D 5e encounters.**

[![Go Report Card](https://goreportcard.com/badge/github.com/suderio/dndsl)](https://goreportcard.com/report/github.com/suderio/dndsl)
[![GoDoc](https://godoc.org/github.com/suderio/dndsl?status.svg)](https://godoc.org/github.com/suderio/dndsl)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Stability: Stable](https://img.shields.io/badge/Stability-Stable-green.svg)](https://github.com/suderio/dndsl/releases)
[![GitHub Actions Workflow Status](https://img.shields.io/github/actions/workflow/status/suderio/dndsl/go.yml)](https://github.com/suderio/dndsl/actions)
[![GitHub Release](https://img.shields.io/github/v/release/suderio/dndsl)](https://github.com/suderio/dndsl/releases)
[![GitHub Downloads](https://img.shields.io/github/downloads/suderio/dndsl/total)](https://github.com/suderio/dndsl/releases)

*Process combat, roll dice, and manage campaigns through a powerful, human-readable Domain Specific Language.*

[Explore the Docs](docs/) · [Report Bug](https://github.com/suderio/dndsl/issues) · [Request Feature](https://github.com/suderio/dndsl/issues)

</div>

---

![DnDSL TUI](assets/TUI.png)

---

## 📖 Introduction

**DnDSL** is a developer-centric tool for Dungeons & Dragons Game Masters. It provides a structured, terminal-based interface (powered by Bubble Tea) to manage complex combat encounters. By using an event-sourced architecture, every roll, hit, and turn is recorded in an immutable log, ensuring your campaign's history is safe and fully reproducible.

### Why DnDSL?

- **Speed**: No more clicking through complex UIs. Type `attack with: Longsword to: Orc_1` and let the engine handle the rest.
- **Precision**: Automated damage resistance, immunity, and vulnerability calculations.
- **Remote-First**: Built-in Telegram bot integration allows players to roll from their phones while the GM runs the engine on a server.
- **Customizable**: Purely YAML-based data layer. Adding a new monster or character is as simple as creating a file.

The **DnDSL** project is uniquely positioned to address "table-time" fatigue by bridging the gap between high-fidelity digital tools and the speed of rule-light systems. By prioritizing a text-based interface, you are offering a "low-friction" entry point that avoids the bloat of modern graphical Virtual Tabletops (VTTs) while keeping the mechanical depth of the 5e SRD.

---

### **The Problem: The "Crunch" vs. "Time" Paradox**

Tabletop RPGs are more popular than ever, but the "Table Time" required to play them is becoming a luxury few can afford. Between managing 400-page rulebooks, setting up complex 3D VTTs, and tracking HP for five different **Zombies**, the actual *playing* often takes a backseat to the *math*.

### **The Solution: DnDSL**

DnDSL is a Domain-Specific Language and engine designed to automate the heavy lifting of D&D 5e through a fast, stateless text interface.

- **Rule-Light Speed, Rule-Heavy Depth:** Perform complex actions like a **Young Green Dragon's Multiattack** or **Poison Breath** with a single line of text.
- **Event-Sourced "Time Travel":** Every roll, damage point, and level-up is a permanent record. Made a mistake? Use the `undo` command to instantly revert the world state.
- **Zero-Setup Persistence:** Sessions are saved as human-readable event logs. Close the CLI and resume your campaign months later exactly where you left off.
- **Mod-First Architecture:** Don't like a rule? Open the **YAML** files for your **Fighter** or **Rogue** and change it. The engine adapts to your homebrew instantly.

---

## 🛠 Strategic Approach to Market Saturation

To stand out in a saturated market, DnDSL focuses on three distinct pillars that traditional VTTs and rulebooks ignore:

### **1. The "Interface of Choice"**

While others chase 3D realism, DnDSL embraces the **efficiency of text**.

- **Text-Based Precision:** A text interface allows for rapid-fire combat without the "click-and-drag" fatigue of graphical maps.
- **Graphical Niceties:** Future updates will introduce TUI (Terminal User Interface) elements—like health bars and ASCII maps—that provide visual feedback without sacrificing the speed of a keyboard-driven workflow.

### **2. Automation of "Pending Choices"**

The engine acts as a digital Dungeon Master assistant. When a **Zombie** hits 0 HP, the engine doesn't just delete it; it pauses and prompts the user for the **Undead Fortitude** save, ensuring rules aren't forgotten in the heat of the moment.

---

## 🛠 Installation

### Pre-built Binaries

Download the latest release from the [Releases](https://github.com/suderio/dndsl/releases) page.

### From Source

```bash
git clone https://github.com/suderio/dndsl.git
cd dndsl
go build -o dndsl main.go
```

#### Prerequisites

- Go 1.25 or higher.

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
- [ ] Discord Bot Integration

---

## 🤝 Contributing

Contributions are what make the open-source community such an amazing place to learn, inspire, and create. Any contributions you make are **greatly appreciated**.

Please see [CONTRIBUTING.md](CONTRIBUTING.md) for details on our code of conduct and the process for submitting pull requests.

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

---

## 📜 License

Distributed under the MIT License. See [LICENSE](LICENSE) for more information.

<div align="center">
 Built with ❤️ by the DnDSL Team.
</div>
