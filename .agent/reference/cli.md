# CLI Commands, Input Grammar, Output, and Completion

## Purpose

Define user-facing CLI contract, command semantics, output stability, and completion behavior.

## Normative Rules

1. Input validation MUST fail fast with usage guidance and non-zero exit codes.
2. Human-readable output SHOULD be concise and deterministic.
3. Command aliases MUST not introduce ambiguous behavior.
4. Completion suggestions MUST be context-aware and deterministic.
5. Interactive command flows MUST only run when stdin/stdout are interactive terminals.
6. Help invocations (`--help`, `-h`) MUST render without requiring active-context resolution.

## Command Groups

### User-Facing Commands

| Command | Subcommands | Description |
| :--- | :--- | :--- |
| `repl` | — | Start the interactive TUI shell |
| `bot` | — | Start the Telegram bot (standalone) |
| `campaign` | `create`, `telegram` | Campaign management |
| `completion` | `bash`, `zsh`, `fish`, `powershell` | Shell completion scripts |
| `version` | — | Print version information |

### Hidden Commands

| Command | Description |
| :--- | :--- |
| `init` | Download SRD data from dnd5eapi (future: `--luafy`) |

### Global Flags

- `--debug`, `-d`: Enable debug output.
- `--verbose`, `-v`: Enable verbose output.
- `--help`, `-h`: Show help.

## Input Grammar (REPL)

Commands follow the pattern: `<command> [by: <actor>] [<key>: <value> [and <value>]*]*`

Examples:

- `roll dice: 2d6`
- `attack by: Fighter to: Goblin`
- `encounter start with: Fighter and Goblin`

Multi-word commands are joined with underscores internally (e.g., `encounter start` → `encounter_start`).

## Failure Modes

1. Missing required path argument.
2. Unsupported command/flag combination.
