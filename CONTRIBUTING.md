# Contributing to DND-DSL

First off, thank you for considering contributing to DND-DSL! It's people like you that make it a great tool for the community.

## Code of Conduct

By participating in this project, you are expected to uphold our Code of Conduct. Please be respectful and professional in all interactions.

## How Can I Contribute?

### Reporting Bugs

- **Check the FAQ**: See if your issue is already known.
- **Search existing issues**: Avoid duplicates.
- **Use the template**: Provide clear steps to reproduce, expected behavior, and your environment.

### Suggesting Enhancements

- **Explain the "Why"**: Describe the problem you're trying to solve.
- **Provide context**: How does this fit into the existing D&D 5e mechanics?

### Pull Requests

1. **Fork the repo** and create your branch from `main`.
2. **Install dependencies**: `go mod download`.
3. **Implement your changes**: Ensure you follow the project's coding style (defined in `.golangci.yml` if available).
4. **Add tests**: Every new feature or fix must have unit tests.
5. **Update documentation**: Follow the [Command Development Rules](task.md).
6. **Run formatting**: `go fmt ./...`.
7. **Submit**: Open a PR with a clear description of the impact.

## Development Rules

Every new command added to the DSL must include updates to:

1. **The `hint` command** (`internal/command/hint.go`): Provide contextual guidance.
2. **Error messages** (`internal/parser/errors.go`): Map syntax errors to friendly usage instructions.
3. **The `help` command** (`internal/command/help.go`): Add usage and summary to the registry.

## Technologies Used

- **Language**: Go (v1.25+)
- **Parser**: [Participle/v2](https://github.com/alecthomas/participle)
- **CLI**: [Cobra](https://github.com/spf13/cobra) & [Viper](https://github.com/spf13/viper)
- **TUI**: [Bubble Tea](https://github.com/charmbracelet/bubbletea) & [Lip Gloss](https://github.com/charmbracelet/lipgloss)
- **Persistence**: Event Sourcing (JSONL)

---

Happy rolling! 🎲
