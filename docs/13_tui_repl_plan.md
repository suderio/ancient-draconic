# REPL TUI Plan

The goal is to upgrade the current barebones `bufio.Scanner` REPL loop in DnDSL into a fully-fledged Terminal User Interface (TUI). We want to introduce command history (Up/Down arrows), keybindings, and potentially a persistent dashboard of the GameState so the REPL isn't constantly spamming `hint` or status printouts into the terminal buffer.

Here is an analysis of the three leading alternatives in the Go ecosystem, their pros/cons, and a proposed implementation plan.

---

## Alternative 1: Bubble Tea (`github.com/charmbracelet/bubbletea`)

Bubble Tea (used with `bubbles` like `textinput` and `viewport`) provides a comprehensive Elm-inspired Model-View-Update architecture for TUIs.

* **Pros**:
  * **True UI**: We can split the terminal into panes. The top pane can persistently render the `=== Active Encounter ===` state, pending checks, and turn order. The bottom pane acts as the command input box with history.
  * Highly customizable, modern, and beautiful. Allows for colors, borders, and advanced layout (via `lipgloss`).
  * Event-driven: Fits well with our existing Command -> Event -> State architecture.
* **Cons**:
  * **Complexity**: Requires rewriting `cmd/repl.go` to fit the `Update(msg) (Model, Cmd)` loop rather than a simple `for` loop.
  * The REPL output (event log) must be manually rendered into a scrollable viewport instead of relying on standard `fmt.Println` scrolling.
* **Tradeoff**: High initial setup cost, but provides the ultimate "Game Engine" experience without spamming the CLI.

---

## Alternative 2: Go-Prompt (`github.com/c-bata/go-prompt`)

Go-Prompt is heavily inspired by `python-prompt-toolkit`.

* **Pros**:
  * **Auto-completion**: It natively supports rich drop-down menus for auto-complete. We could give the user instant suggestions like `[roll, encounter, attack, ask, hint]` as they type.
  * Built-in history and standard Emacs/Vim keybindings.
* **Cons**:
  * Still operates as a classic line-by-line CLI. State updates (HP changes) simply print to the terminal, pushing old text up.
  * The library hasn't seen major updates recently and can sometimes fight with standard `fmt.Print` calls.
* **Tradeoff**: Excellent for pure command discovery (if syntax is hard to remember), but visually it is just an upgraded prompt, not a "dashboard".

---

## Alternative 3: Readline (`github.com/chzyer/readline` or `x/term`)

A pure Go implementation of the GNU Readline interface.

* **Pros**:
  * **Drop-in Replacement**: We can literally replace `scanner.Scan()` with `rl.Readline()`.
  * Gives exactly what was asked: Command history (Up/Down) and standard keybindings (Ctrl-A, Ctrl-E, etc.).
  * Lowest risk and fastest implementation.
* **Cons**:
  * Bare minimum. No auto-complete prompts, no split-panes, no persistent visual formatting.
* **Tradeoff**: Quick and dirty. Solves the immediate pain point of navigating history but adds zero "wow" factor.

---

## Proposal: The Bubble Tea Dashboard Approach

Given this is a D&D Engine, the user experience heavily relies on tracking complex state (HP, conditions, turn order, blocked checks). Just printing strings down a terminal makes it hard to remember who is alive and who is next.

I propose we implement **Alternative 1: Bubble Tea**.

### The Bubble Tea Layout Plan

The terminal screen will be divided into three sections:

1. **Top Header**: The active World/Campaign names.
2. **Main Viewport**:
    * *Left Side*: The `Event.Message()` rolling log (what just happened).
    * *Right Side*: A persistent, bordered box showing the `GameState`: Entities, HP, Turn Order, and Pending Checks locks.
3. **Bottom Footer**: The `textinput` component acting as the REPL prompt (`> roll :by paulo...`), infused with history tracking.

### Implementation Steps

1. **Add Dependencies**: `go get github.com/charmbracelet/bubbletea github.com/charmbracelet/bubbles github.com/charmbracelet/lipgloss`
2. **Define the Model**: Create a `replModel` struct in `cmd/repl.go` holding our `app *session.Session`, a `textinput.Model`, a list of `history []string`, and a `viewport.Model` for logs.
3. **Update Loop**:
    * On `<Enter>`, extract text, push to history, call `app.Execute(text)`.
    * Retrieve generated `Event`, push its `.Message()` to the log viewport.
    * Let the `View()` function re-render the screen.
4. **Keybindings**: Up/Down arrow keys will cycle through the `history []string` and replace the `textinput.Value()`.
5. **Remove Stdout statements**: Ensure the `Execute` path no longer calls `fmt.Println` directly, routing everything to the Bubble Tea UI buffer.

This transforms the REPL into an actual interactive game console.

## Addendum: Autocomplete Complexity in Bubble Tea

While `bubbles/textinput` handles history natively with a bit of custom keybinding logic, it **does not** have a built-in drop-down menu for autocomplete out of the box (unlike `go-prompt`).

However, implementing autocomplete in Bubble Tea is relatively straightforward using the [bubbles/list](https://github.com/charmbracelet/bubbles/tree/master/list) component:

1. **Complexity Level**: Moderate.
2. **How it works**:
    * We watch the `textinput` value on every keystroke (`tea.KeyMsg`).
    * If the user types a command keyword (`encounter`, `attack`, `roll`), we can pass the prefix to a search function.
    * We toggle a floating `list.Model` underneath the `textinput` in the `View()` render.
    * The user can use `<Tab>` or `<Up/Down>` to navigate the `list` selections, and `<Enter>` to inject the selection back into the `textinput`.
3. **Data Sources**: We already have `data.Loader` wired into the REPL. This means our autocomplete engine can literally suggest live game data:
    * `"attack :by "` -> Autocompletes active `GameState.Entities` names.
    * `"attack :by paulo :with "` -> Autocompletes Paulo's configured weapons from his YAML.

**Verdict on Autocomplete**: Building a bespoke auto-completer in Bubble Tea takes more manual wiring than `go-prompt`, but because we own the `Update` loop, we can make it hyper-intelligent by tying it directly to our `GameState` rather than just static strings. We will add this to the Phase 13 roadmap.
