package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"dndsl/internal/session"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1).
			MarginBottom(1)

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#999999"))

	stateBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#874BFD")).
			Padding(1, 2)

	logBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#04B575")).
			Padding(0, 1)

	autocompleteStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#F25D94"))
)

type suggestion string

func (s suggestion) Title() string       { return string(s) }
func (s suggestion) Description() string { return "" }
func (s suggestion) FilterValue() string { return string(s) }

type replModel struct {
	app          *session.Session
	textInput    textinput.Model
	viewport     viewport.Model
	suggestions  list.Model
	history      []string
	historyIdx   int
	logContent   string
	width        int
	height       int
	worldName    string
	campaignName string
	showList     bool
}

func newREPLModel(app *session.Session, worldName, campaignName string) replModel {
	ti := textinput.New()
	ti.Placeholder = "Enter command (e.g., roll :by GM 1d20)..."
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 60

	vp := viewport.New(0, 0)
	vp.SetContent("Welcome to the DND-DSL Engine!\nType 'exit' to quit.")

	// Configure a minimalist list for autocomplete
	sugList := list.New([]list.Item{}, list.NewDefaultDelegate(), 50, 10)
	sugList.SetShowTitle(false)
	sugList.SetShowStatusBar(false)
	sugList.SetFilteringEnabled(false) // We filter manually
	sugList.SetShowHelp(false)

	return replModel{
		app:          app,
		textInput:    ti,
		viewport:     vp,
		suggestions:  sugList,
		history:      []string{},
		historyIdx:   -1,
		logContent:   "Welcome to the DND-DSL Engine!\nType 'exit' to quit.",
		worldName:    worldName,
		campaignName: campaignName,
	}
}

func (m replModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *replModel) updateSuggestions() {
	val := m.textInput.Value()
	var items []list.Item

	defer func() {
		m.suggestions.SetItems(items)
		m.showList = len(items) > 0
		if m.showList {
			m.suggestions.ResetSelected()
		}
	}()

	if val == "" {
		return
	}

	state := m.app.State()

	if strings.HasPrefix(strings.ToLower(val), "attack :by ") {
		prefix := val[11:]
		if !strings.Contains(prefix, " :") {
			// Suggest entity names
			for id := range state.Entities {
				if strings.HasPrefix(id, prefix) {
					items = append(items, suggestion("attack :by "+id+" :with "))
				}
			}
		} else if strings.Contains(prefix, " :with ") {
			parts := strings.SplitN(prefix, " :with ", 2)
			if len(parts) == 2 {
				actorID := strings.TrimSpace(parts[0])
				weaponPrefix := parts[1]
				if _, ok := state.Entities[actorID]; ok {
					if char, err := m.app.Loader().LoadCharacter(actorID); err == nil {
						for _, act := range char.Actions {
							if strings.HasPrefix(strings.ToLower(act.Name), strings.ToLower(weaponPrefix)) {
								items = append(items, suggestion(fmt.Sprintf("attack :by %s :with %s :to ", actorID, strings.ToLower(act.Name))))
							}
						}
					} else if monster, err := m.app.Loader().LoadMonster(actorID); err == nil {
						for _, act := range monster.Actions {
							if strings.HasPrefix(strings.ToLower(act.Name), strings.ToLower(weaponPrefix)) {
								items = append(items, suggestion(fmt.Sprintf("attack :by %s :with %s :to ", actorID, strings.ToLower(act.Name))))
							}
						}
					}
				}
			}
		} else if strings.Contains(prefix, " :to ") {
			parts := strings.SplitN(prefix, " :to ", 2)
			if len(parts) == 2 {
				targetPrefix := parts[1]
				baseCmd := val[:len(val)-len(targetPrefix)]
				for id := range state.Entities {
					if strings.HasPrefix(id, targetPrefix) {
						items = append(items, suggestion(baseCmd+id))
					}
				}
			}
		}
	} else if strings.HasPrefix(strings.ToLower(val), "ask :by gm :check ") {
		checkPrefix := val[18:]
		baseCmds := []string{"dex save", "str save", "con save", "int save", "wis save", "cha save", "athletics", "acrobatics", "stealth", "perception", "deception", "intimidation"}
		for _, c := range baseCmds {
			if strings.HasPrefix(c, checkPrefix) {
				items = append(items, suggestion("ask :by GM :check "+c+" :of "))
			}
		}
	} else if strings.HasPrefix(strings.ToLower(val), "encounter start :with ") {
		prefix := val[22:]
		baseCmd := val[:22]
		// It's hard to read local files smoothly on every keystroke, so if there's an active entity list we can suggest them,
		// but since encounter is starting, list is likely empty.
		_ = prefix
		_ = baseCmd
	} else {
		// Base commands
		cmds := []string{"roll :by ", "encounter start", "encounter end", "add ", "initiative :by ", "attack :by ", "damage :by ", "turn", "hint", "ask :by ", "check :by ", "exit", "quit"}
		for _, c := range cmds {
			if strings.HasPrefix(c, strings.ToLower(val)) && len(val) < len(c) {
				items = append(items, suggestion(c))
			}
		}
	}
}

func (m replModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
		lsCmd tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit

		case tea.KeyUp:
			if m.showList {
				m.suggestions, lsCmd = m.suggestions.Update(msg)
			} else {
				if len(m.history) > 0 {
					if m.historyIdx == -1 {
						m.historyIdx = len(m.history) - 1
					} else if m.historyIdx > 0 {
						m.historyIdx--
					}
					m.textInput.SetValue(m.history[m.historyIdx])
					m.updateSuggestions()
				}
			}

		case tea.KeyDown:
			if m.showList {
				m.suggestions, lsCmd = m.suggestions.Update(msg)
			} else {
				if len(m.history) > 0 && m.historyIdx != -1 {
					if m.historyIdx < len(m.history)-1 {
						m.historyIdx++
						m.textInput.SetValue(m.history[m.historyIdx])
					} else {
						m.historyIdx = -1
						m.textInput.SetValue("")
					}
					m.updateSuggestions()
				}
			}

		case tea.KeyTab:
			if m.showList {
				if i, ok := m.suggestions.SelectedItem().(suggestion); ok {
					m.textInput.SetValue(string(i))
					m.textInput.SetCursor(len(string(i)))
					m.updateSuggestions()
				}
			}

		case tea.KeyEnter:
			val := strings.TrimSpace(m.textInput.Value())
			if val == "exit" || val == "quit" {
				return m, tea.Quit
			}

			if val != "" {
				// Prevent duplicate history entries
				if len(m.history) == 0 || m.history[len(m.history)-1] != val {
					m.history = append(m.history, val)
				}
				m.historyIdx = -1
				m.textInput.SetValue("")
				m.updateSuggestions()

				m.logContent += fmt.Sprintf("\n\n> %s\n", val)
				evt, err := m.app.Execute(val)
				if err != nil {
					m.logContent += fmt.Sprintf("Error: %v", err)
				} else if evt != nil {
					m.logContent += evt.Message()
				}

				m.viewport.SetContent(m.logContent)
				m.viewport.GotoBottom()
			}
		default:
			// Normal typing
			m.textInput, tiCmd = m.textInput.Update(msg)
			m.updateSuggestions()
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width - 4
		m.viewport.Height = msg.Height - 15 - 10 // Leave room for headers, input and list
		if m.viewport.Height < 5 {
			m.viewport.Height = 5
		}
	}

	m.viewport, vpCmd = m.viewport.Update(msg)
	return m, tea.Batch(tiCmd, vpCmd, lsCmd)
}

func (m replModel) View() string {
	if m.width == 0 {
		return "Initializing..."
	}

	title := titleStyle.Render(fmt.Sprintf(" DND-DSL Engine | %s / %s ", m.worldName, m.campaignName))

	// Render state
	stateView := "=== Active Encounter ==="
	state := m.app.State()

	if state.IsFrozen() {
		stateView += " [FROZEN]"
	}
	stateView += "\n\n"

	if len(state.TurnOrder) > 0 {
		currentActor := state.TurnOrder[state.CurrentTurn]
		stateView += fmt.Sprintf("Turn: %s\n\n", currentActor)
	}

	if len(state.Entities) == 0 {
		stateView += "No entities active."
	} else {
		for id, ent := range state.Entities {
			conds := ""
			if len(ent.Conditions) > 0 {
				conds = fmt.Sprintf(" [%s]", strings.Join(ent.Conditions, ", "))
			}
			stateView += fmt.Sprintf(" - %s (%s): %d/%d HP%s\n", id, ent.Name, ent.HP, ent.MaxHP, conds)
		}
	}

	if len(state.PendingChecks) > 0 {
		stateView += "\nPending Checks:\n"
		for id, req := range state.PendingChecks {
			stateView += fmt.Sprintf(" - %s requires %v (DC %d)\n", id, req.Check, req.DC)
		}
	}

	if state.PendingDamage != nil {
		stateView += fmt.Sprintf("\nPending Damage from %s:\n", state.PendingDamage.Attacker)
		for _, t := range state.PendingDamage.Targets {
			if state.PendingDamage.HitStatus[t] {
				stateView += fmt.Sprintf(" - %s (Hit)\n", t)
			} else {
				stateView += fmt.Sprintf(" - %s (Miss)\n", t)
			}
		}
	}

	stateBox := stateBoxStyle.Width(m.width - 4).Render(stateView)
	logBox := logBoxStyle.Width(m.width - 4).Render(m.viewport.View())

	var inputArea string
	if m.showList {
		inputArea = fmt.Sprintf("%s\n%s", m.textInput.View(), autocompleteStyle.Render(m.suggestions.View()))
	} else {
		inputArea = m.textInput.View()
	}

	return fmt.Sprintf(
		"%s\n%s\n%s\n\n%s\n%s",
		title,
		stateBox,
		logBox,
		inputArea,
		infoStyle.Render("(esc to quit, tab to complete, up/down history)"),
	)
}

func RunTUI(app *session.Session, worldDir, campaignDir string) error {
	p := tea.NewProgram(newREPLModel(app, filepath.Base(worldDir), filepath.Base(campaignDir)), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}
