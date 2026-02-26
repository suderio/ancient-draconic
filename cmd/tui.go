package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/suderio/ancient-draconic/internal/manifest"

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
	app          *manifest.Session
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

func newREPLModel(app *manifest.Session, worldName, campaignName string) replModel {
	ti := textinput.New()
	ti.Placeholder = "Enter command (e.g., roll dice: 1d20)..."
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

	// Base hardcoded commands
	baseCmds := []string{"roll dice: ", "help ", "hint", "ask by: ", "adjudicate ", "allow", "deny", "exit", "quit"}

	// Dynamically pull loaded Manifest Commands
	mf := m.app.Manifest()
	for cmdKey, cmdDef := range mf.Commands {
		// Use display name (spaces) for autocomplete
		displayName := cmdDef.Name
		if displayName == "" {
			displayName = strings.ReplaceAll(cmdKey, "_", " ")
		}
		baseCmds = append(baseCmds, displayName+" ")
	}

	for _, c := range baseCmds {
		if strings.HasPrefix(strings.ToLower(c), strings.ToLower(val)) && len(val) < len(c) {
			items = append(items, suggestion(c))
		}
	}

	// Entity completion when typing "to: " or "by: "
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
				events, err := m.app.Execute(val)
				if err != nil {
					m.logContent += fmt.Sprintf("Error: %v", err)
				} else {
					for _, evt := range events {
						msg := evt.Message()
						if msg != "" {
							m.logContent += msg + "\n"
						}
					}
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
	stateView := "=== Game State ==="
	state := m.app.State()

	stateView += "\n\n"

	// Show active loops
	hasActiveLoop := false
	for name, loop := range state.Loops {
		if loop.Active {
			hasActiveLoop = true
			stateView += fmt.Sprintf("Loop: %s (active)\n", name)
			if len(loop.Order) > 0 {
				stateView += fmt.Sprintf("  Order: %s\n", strings.Join(orderToStrings(loop.Order), ", "))
			}
		}
	}
	if !hasActiveLoop {
		stateView += "No active loops.\n"
	}

	stateView += "\n"

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
			if maxHP > 0 {
				stateView += fmt.Sprintf(" - %s (%s): %d/%d HP%s\n", id, ent.Name, hp, maxHP, conds)
			} else {
				stateView += fmt.Sprintf(" - %s (%s)%s\n", id, ent.Name, conds)
			}
		}
	}

	return stateBoxStyle.Width(m.width - 4).Render(stateView)
}

// orderToStrings converts a loop order map to a sorted display list.
func orderToStrings(order map[string]int) []string {
	var result []string
	for actor, val := range order {
		result = append(result, fmt.Sprintf("%s(%d)", actor, val))
	}
	return result
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

func RunTUI(app *manifest.Session, worldDir, campaignDir string) error {
	m := newREPLModel(app, filepath.Base(worldDir), filepath.Base(campaignDir))
	p := tea.NewProgram(&m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}
