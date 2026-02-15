package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tatianab/text-game/internal/engine"
	"github.com/tatianab/text-game/internal/models"
)

type sessionState int

const (
	stateInputHint sessionState = iota
	stateLoading
	statePlaying
	stateQuitting
	stateError
)

type logEntry struct {
	IsUser bool
	Text   string
}

type model struct {
	state       sessionState
	engine      *engine.Engine
	session     *models.GameSession
	textArea    textarea.Model
	viewport    viewport.Model
	err         error
	history     []logEntry
	width       int
	height      int
	lastOutcome string
}

var (
	userStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EEEEEE")).
			Background(lipgloss.Color("#5F5F87")).
			Bold(true).
			PaddingLeft(1)

	gameStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Italic(true)

	stateStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(lipgloss.Color("#3C3C3C")).
			PaddingLeft(2).
			Foreground(lipgloss.Color("#AAAAAA"))

	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFA500")).
			Bold(true).
			Underline(true)
)

func NewModel(eng *engine.Engine) model {
	ta := textarea.New()
	ta.Placeholder = "Enter a hint or 'random'..."
	ta.Focus()
	ta.CharLimit = 156
	ta.SetWidth(40)
	ta.SetHeight(1)
	ta.ShowLineNumbers = false

	return model{
		state:    stateInputHint,
		engine:   eng,
		textArea: ta,
	}
}

func (m model) Init() tea.Cmd {
	return textarea.Blink
}

type worldGeneratedMsg struct {
	session *models.GameSession
}

type turnProcessedMsg struct {
	outcome string
	err     error
}

type errMsg struct {
	err error
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			if m.state == statePlaying {
				m.state = stateQuitting
				m.textArea.Reset()
				m.textArea.Placeholder = "Type a name to save, or press Enter to quit without saving..."
				m.textArea.SetHeight(1)
				return m, nil
			}
			return m, tea.Quit

		case tea.KeyEnter:
			if m.state == stateQuitting {
				action := strings.TrimSpace(m.textArea.Value())
				m.textArea.Reset()
				if action == "/cancel" {
					m.state = statePlaying
					m.textArea.Placeholder = "What do you do?"
					m.textArea.SetHeight(3)
					return m, nil
				}
				if action != "" {
					err := m.session.Save(action)
					if err != nil {
						m.err = err
						m.state = stateError
						return m, nil
					}
				}
				return m, tea.Quit
			}
			if m.state == stateInputHint {
				hint := strings.TrimSpace(m.textArea.Value())
				if strings.HasPrefix(hint, "/load ") {
					name := strings.TrimPrefix(hint, "/load ")
					session, err := models.LoadSession(name)
					if err != nil {
						m.err = err
						m.state = stateError
						return m, nil
					}
					m.session = session
					m.state = statePlaying
					// Reconstruct history
					m.history = nil
					m.history = append(m.history, logEntry{
						IsUser: false,
						Text:   fmt.Sprintf("World: %s\n\n%s", m.session.State.CurrentLocation, m.session.World.Description),
					})
					for _, entry := range m.session.History.Entries {
						m.history = append(m.history, logEntry{IsUser: true, Text: entry.PlayerAction})
						m.history = append(m.history, logEntry{IsUser: false, Text: entry.Outcome})
					}

					logWidth := int(float64(m.width) * 0.75)
					if m.viewport.Width == 0 {
						m.viewport = viewport.New(logWidth, m.height-8)
					}
					m.viewport.SetContent(m.renderLog())
					m.viewport.GotoBottom()
					m.textArea.Placeholder = "What do you do?"
					m.textArea.Reset()
					m.textArea.SetHeight(3)
					return m, nil
				}
				if hint == "" {
					hint = "random"
				}
				m.state = stateLoading
				return m, m.generateWorld(hint)
			}
			if m.state == statePlaying {
				action := strings.TrimSpace(m.textArea.Value())
				if action == "" {
					return m, nil
				}
				m.textArea.Reset()

				if action == "/quit" {
					m.state = stateQuitting
					m.textArea.Placeholder = "Type a name to save, or press Enter to quit without saving..."
					m.textArea.SetHeight(1)
					return m, nil
				}
				if action == "/restart" {
					m.state = stateInputHint
					m.history = nil
					m.session = nil
					m.textArea.Placeholder = "Enter a hint or 'random'..."
					m.textArea.SetHeight(1)
					return m, nil
				}
				if strings.HasPrefix(action, "/save ") {
					name := strings.TrimPrefix(action, "/save ")
					err := m.session.Save(name)
					if err != nil {
						m.history = append(m.history, logEntry{IsUser: false, Text: "Failed to save: " + err.Error()})
					} else {
						m.history = append(m.history, logEntry{IsUser: false, Text: "Game saved as '" + name + "'"})
					}
					m.viewport.SetContent(m.renderLog())
					m.viewport.GotoBottom()
					return m, nil
				}

				m.history = append(m.history, logEntry{IsUser: true, Text: action})
				m.viewport.SetContent(m.renderLog())
				m.viewport.GotoBottom()
				return m, m.processTurn(action)
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = int(float64(msg.Width) * 0.75)
		m.viewport.Height = msg.Height - 8
		m.textArea.SetWidth(msg.Width - 4)
		if m.state == statePlaying {
			m.viewport.SetContent(m.renderLog())
		}

	case worldGeneratedMsg:
		m.session = msg.session
		m.state = statePlaying
		m.history = append(m.history, logEntry{
			IsUser: false,
			Text:   fmt.Sprintf("%s\nLocation: %s\n\n%s", m.session.World.Title, m.session.State.CurrentLocation, m.session.World.Description),
		})

		logWidth := int(float64(m.width) * 0.75)
		if m.viewport.Width == 0 {
			m.viewport = viewport.New(logWidth, m.height-8)
		}
		m.viewport.SetContent(m.renderLog())
		m.textArea.Placeholder = "What do you do?"
		m.textArea.Reset()
		m.textArea.SetHeight(3)
		m.session.Save("current")
		m.session.Save(m.session.World.ShortName)
		return m, nil

	case turnProcessedMsg:
		if msg.err != nil {
			m.err = msg.err
			m.state = stateError
			return m, nil
		}
		m.lastOutcome = msg.outcome
		m.history = append(m.history, logEntry{IsUser: false, Text: msg.outcome})
		m.viewport.SetContent(m.renderLog())
		m.viewport.GotoBottom()
		m.session.Save("current")
		m.session.Save(m.session.World.ShortName)
		return m, nil

	case errMsg:
		m.err = msg.err
		m.state = stateError
		return m, nil
	}

	if m.state == stateInputHint || m.state == statePlaying || m.state == stateQuitting {
		m.textArea, cmd = m.textArea.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m model) View() string {
	var s string

	switch m.state {
	case stateInputHint:
		saves, _ := models.ListSessions()
		savesList := ""
		if len(saves) > 0 {
			savesList = "\nOr load a previous game: /load <name>\nAvailable saves: " + strings.Join(saves, ", ") + "\n"
		}

		s = fmt.Sprintf(
			"Welcome to the Text Game Generator!\n\n%s\n%s\n%s",
			"Give me a hint about the world you want to play in (e.g., 'cyberpunk detective', 'zombie kitchen'):",
			savesList,
			m.textArea.View(),
		)

	case stateLoading:
		s = "\n  Generating your world... please wait.\n"

	case stateQuitting:
		s = fmt.Sprintf(
			"Do you want to save your game before quitting?\n\n%s\n\n%s",
			"- To save and quit: Type a save name and press Enter\n- To quit without saving: Just press Enter\n- To go back to the game: Type /cancel and press Enter",
			m.textArea.View(),
		)

	case statePlaying:
		logView := m.viewport.View()
		stateView := m.renderState()

		// Join log and state horizontally
		mainView := lipgloss.JoinHorizontal(lipgloss.Top,
			logView,
			stateView,
		)

		help := helpStyle.Render("Commands: /save <name>, /restart, /quit, or just type what you want to do.")

		s = lipgloss.JoinVertical(lipgloss.Left,
			mainView,
			"\n"+m.textArea.View(),
			"\n"+help,
		)

	case stateError:
		s = fmt.Sprintf("\n  Error: %v\n\nPress Esc to quit.", m.err)
	}

	return "\n" + s + "\n"
}

func (m model) renderState() string {
	if m.session == nil {
		return ""
	}

	state := m.session.State

	// Location
	location := titleStyle.Render("LOCATION") + "\n" + state.CurrentLocation + "\n\n"

	// Stats
	statsTitle := titleStyle.Render("STATS") + "\n"
	stats := fmt.Sprintf("Health: %s\nProgress: %s\n", state.Health, state.Progress)

	var keys []string
	for k := range state.Stats {
		if k != "health" && k != "progress" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	for _, k := range keys {
		stats += fmt.Sprintf("%s: %s\n", k, state.Stats[k])
	}
	stats += "\n"

	// Inventory
	invTitle := titleStyle.Render("INVENTORY") + "\n"
	inventory := ""
	if len(state.Inventory) == 0 {
		inventory = "(empty)"
	} else {
		for _, item := range state.Inventory {
			inventory += "- " + item + "\n"
		}
	}

	content := location + statsTitle + stats + invTitle + inventory

	stateWidth := int(float64(m.width) * 0.23) // Leave some room for padding
	return stateStyle.Width(stateWidth).Height(m.viewport.Height).Render(content)
}

func (m model) renderLog() string {
	var b strings.Builder
	logWidth := int(float64(m.width) * 0.75)

	for i, entry := range m.history {
		var styled string
		if entry.IsUser {
			styled = userStyle.Width(logWidth).Render("> " + entry.Text)
		} else {
			styled = gameStyle.Width(logWidth).Render(entry.Text)
		}
		b.WriteString(styled)
		if i < len(m.history)-1 {
			b.WriteString("\n\n")
		}
	}

	return b.String()
}

func (m model) generateWorld(hint string) tea.Cmd {
	return func() tea.Msg {
		session, err := m.engine.GenerateWorld(context.Background(), hint)
		if err != nil {
			return errMsg{err}
		}
		return worldGeneratedMsg{session}
	}
}

func (m model) processTurn(action string) tea.Cmd {
	return func() tea.Msg {
		outcome, err := m.engine.ProcessTurn(context.Background(), m.session, action)
		return turnProcessedMsg{outcome, err}
	}
}

func Run(eng *engine.Engine) error {
	p := tea.NewProgram(NewModel(eng), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
