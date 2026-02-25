package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/suderio/ancient-draconic/internal/session"

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
	ti.Placeholder = "Enter command (e.g., roll by: GM 1d20)..."
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 60

	vp := viewport.New(0, 0)
	vp.SetContent("Welcome to the Ancient Draconic Engine!\nType 'exit' to quit.")

	// Configure a minimalist list for autocomplete
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false
	delegate.SetHeight(1)
	delegate.SetSpacing(0)
	sugList := list.New([]list.Item{}, delegate, 50, 7) // Show up to 7 items
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
		logContent:   "Welcome to the Ancient Draconic Engine!\nType 'exit' to quit.",
		worldName:    worldName,
		campaignName: campaignName,
	}
}

func (m *replModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *replModel) updateSuggestions() {
	val := m.textInput.Value()
	var items []list.Item

	defer func() {
		m.suggestions.SetItems(items)
		m.showList = len(items) > 0
		if m.showList {
			// Use a more generous height for the list to avoid the pagination indicator (•••)
			// Small counts like 1-3 often need at least 4-5 total lines in the list model
			// to avoid clipping symbols depending on theme/styles.
			h := len(items)
			if h > 10 {
				h = 10
			}
			listHeight := h
			if listHeight > 0 && listHeight < 4 {
				listHeight = 4
			}
			m.suggestions.SetHeight(listHeight)
			m.suggestions.ResetSelected()
		}
	}()

	if val == "" {
		return
	}

	state := m.app.State()

	// Base Engine Commands & Contexts
	baseCmds := []string{"roll by: ", "encounter start", "encounter end", "add ", "initiative by: ", "turn", "hint", "ask by: ", "adjudicate ", "allow", "deny", "help ", "exit", "quit"}

	// Dynamically pull loaded Manifest Commands
	if manifest, err := m.app.Loader().LoadManifest(); err == nil {
		for cmdName := range manifest.Commands {
			baseCmds = append(baseCmds, fmt.Sprintf("%s by: ", cmdName))
		}
	}

	for _, c := range baseCmds {
		if strings.HasPrefix(strings.ToLower(c), strings.ToLower(val)) && len(val) < len(c) {
			items = append(items, suggestion(c))
		}
	}

	// Simple context-aware Entity completion when typing "to: " or "by: "
	if strings.Contains(strings.ToLower(val), " to: ") {
		parts := strings.SplitN(strings.ToLower(val), " to: ", 2)
		if len(parts) == 2 {
			targetPrefix := parts[1]
			baseStr := val[:len(val)-len(targetPrefix)]
			for id := range state.Entities {
				if strings.HasPrefix(strings.ToLower(id), strings.ToLower(targetPrefix)) {
					items = append(items, suggestion(baseStr+id))
				}
			}
		}
	} else if strings.Contains(strings.ToLower(val), " by: ") && !strings.Contains(strings.ToLower(val), " with: ") && !strings.Contains(strings.ToLower(val), " to: ") {
		parts := strings.SplitN(strings.ToLower(val), " by: ", 2)
		if len(parts) == 2 {
			actorPrefix := parts[1]
			baseStr := val[:len(val)-len(actorPrefix)]
			for id := range state.Entities {
				if strings.HasPrefix(strings.ToLower(id), strings.ToLower(actorPrefix)) {
					items = append(items, suggestion(baseStr+id+" "))
				}
			}
		}
	}
}

func (m *replModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		m.viewport.Height = msg.Height - 30 // Initial conservative estimate
		if m.viewport.Height < 5 {
			m.viewport.Height = 5
		}
		m.suggestions.SetWidth(msg.Width - 6)
	}

	m.viewport, vpCmd = m.viewport.Update(msg)

	// Calculate accurate heights for dynamic components
	titleH := lipgloss.Height(titleStyle.Render("Dummy"))
	stateH := lipgloss.Height(m.renderState())
	inputH := 1

	listAreaHeight := 0
	if m.showList {
		listAreaHeight = m.suggestions.Height() + 2 // +2 for autocompleteStyle borders
	}

	infoH := lipgloss.Height(infoStyle.Render("Dummy"))
	paddingH := 7

	// Total fixed overhead: title + state + input + listArea + info + padding + spacing
	overhead := titleH + stateH + inputH + listAreaHeight + infoH + paddingH + 4

	m.viewport.Height = m.height - overhead
	if m.viewport.Height < 4 {
		m.viewport.Height = 4
	}

	return m, tea.Batch(tiCmd, vpCmd, lsCmd)
}

func (m *replModel) renderState() string {
	stateView := "=== Active Encounter ==="
	state := m.app.State()

	if state.IsFrozen() {
		stateView += " [FROZEN]"
	}
	stateView += "\n\n"

	if len(state.TurnOrder) > 0 && state.CurrentTurn >= 0 {
		currentActor := state.TurnOrder[state.CurrentTurn]
		stateView += fmt.Sprintf("Turn: %s\n\n", currentActor)
	} else if state.IsEncounterActive {
		stateView += "Turn: Setup (Waiting for initiatives)\n\n"
	}

	if len(state.Entities) == 0 {
		stateView += "No entities active."
	} else {
		for id, ent := range state.Entities {
			conds := ""
			if len(ent.Conditions) > 0 {
				conds = fmt.Sprintf(" [%s]", strings.Join(ent.Conditions, ", "))
			}
			hp := ent.Resources["hp"] - ent.Spent["hp"]
			maxHP := ent.Resources["hp"]
			stateView += fmt.Sprintf(" - %s (%s): %d/%d HP%s\n", id, ent.Name, hp, maxHP, conds)
		}
	}

	if pendingChecks, ok := state.Metadata["pending_checks"].(map[string]any); ok && len(pendingChecks) > 0 {
		stateView += "\nPending Checks:\n"
		for id, reqAny := range pendingChecks {
			if req, ok := reqAny.(map[string]any); ok {
				check, _ := req["check"].(string)
				dc, _ := req["dc"].(int)
				stateView += fmt.Sprintf(" - %s requires %v (DC %d)\n", id, check, dc)
			}
		}
	}

	if pendingDmg, ok := state.Metadata["pending_damage"].(map[string]any); ok && pendingDmg != nil {
		attacker, _ := pendingDmg["attacker"].(string)
		stateView += fmt.Sprintf("\nPending Damage from %s:\n", attacker)

		targets, _ := pendingDmg["targets"].([]string)
		hitStatus, _ := pendingDmg["hit_status"].(map[string]bool)

		for _, t := range targets {
			if hitStatus[t] {
				stateView += fmt.Sprintf(" - %s (Hit)\n", t)
			} else {
				stateView += fmt.Sprintf(" - %s (Miss)\n", t)
			}
		}
	}

	return stateBoxStyle.Width(m.width - 4).Render(stateView)
}

func (m *replModel) View() string {
	if m.width == 0 {
		return "Initializing..."
	}

	title := titleStyle.Render(fmt.Sprintf(" Ancient Draconic Engine | %s / %s ", m.worldName, m.campaignName))
	stateBox := m.renderState()
	logBox := logBoxStyle.Width(m.width - 4).Render(m.viewport.View())

	var inputArea string
	if m.showList {
		inputArea = fmt.Sprintf("%s\n%s", m.textInput.View(), autocompleteStyle.Render(m.suggestions.View()))
	} else {
		inputArea = m.textInput.View()
	}

	mainView := lipgloss.JoinVertical(lipgloss.Left,
		title,
		stateBox,
		logBox,
		"\n",
		inputArea,
		infoStyle.Render("(esc to quit, tab to complete, up/down history)"),
	)

	return mainView + strings.Repeat("\n", 7)
}

func RunTUI(app *session.Session, worldDir, campaignDir string) error {
	m := newREPLModel(app, filepath.Base(worldDir), filepath.Base(campaignDir))
	p := tea.NewProgram(&m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}
