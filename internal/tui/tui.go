package tui

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
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
	stateError
)

type model struct {
	state       sessionState
	engine      *engine.Engine
	session     *models.GameSession
	textInput   textinput.Model
	viewport    viewport.Model
	err         error
	gameLog     string
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
	ti := textinput.New()
	ti.Placeholder = "Enter a hint or 'random'..."
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 40

	return model{
		state:     stateInputHint,
		engine:    eng,
		textInput: ti,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
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
			return m, tea.Quit

		case tea.KeyEnter:
			if m.state == stateInputHint {
				hint := m.textInput.Value()
				if hint == "" {
					hint = "random"
				}
				m.state = stateLoading
				return m, m.generateWorld(hint)
			}
			if m.state == statePlaying {
				action := m.textInput.Value()
				if action == "" {
					return m, nil
				}
				m.textInput.Reset()

				if action == "/quit" {
					return m, tea.Quit
				}
				if action == "/restart" {
					m.state = stateInputHint
					m.gameLog = ""
					m.session = nil
					m.textInput.Placeholder = "Enter a hint or 'random'..."
					return m, nil
				}

				logWidth := int(float64(m.width) * 0.75)
				styledAction := userStyle.Width(logWidth).Render("> " + action)
				m.gameLog += "\n\n" + styledAction + "\n\n"
				m.viewport.SetContent(m.renderLog())
				m.viewport.GotoBottom()
				return m, m.processTurn(action)
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = int(float64(msg.Width) * 0.75)
		m.viewport.Height = msg.Height - 6
		if m.state == statePlaying {
			m.viewport.SetContent(m.renderLog())
		}

	case worldGeneratedMsg:
		m.session = msg.session
		m.state = statePlaying
		logWidth := int(float64(m.width) * 0.75)
		header := gameStyle.Bold(true).Render("World: " + m.session.State.CurrentLocation)
		description := gameStyle.Width(logWidth).Render(m.session.World.Description)
		m.gameLog = header + "\n\n" + description + "\n\n"
		if m.viewport.Width == 0 {
			m.viewport = viewport.New(logWidth, m.height-6)
		}
		m.viewport.SetContent(m.renderLog())
		m.textInput.Placeholder = "What do you do?"
		m.textInput.Reset()
		m.session.Save("current")
		return m, nil

	case turnProcessedMsg:
		if msg.err != nil {
			m.err = msg.err
			m.state = stateError
			return m, nil
		}
		m.lastOutcome = msg.outcome
		logWidth := int(float64(m.width) * 0.75)
		styledOutcome := gameStyle.Width(logWidth).Render(msg.outcome)
		m.gameLog += styledOutcome + "\n\n"
		m.viewport.SetContent(m.renderLog())
		m.viewport.GotoBottom()
		m.session.Save("current")
		return m, nil

	case errMsg:
		m.err = msg.err
		m.state = stateError
		return m, nil
	}

	if m.state == stateInputHint || m.state == statePlaying {
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m model) View() string {
	var s string

	switch m.state {
	case stateInputHint:
		s = fmt.Sprintf(
			"Welcome to the Text Game Generator!\n\n%s\n\n%s",
			"Give me a hint about the world you want to play in:",
			m.textInput.View(),
		)

	case stateLoading:
		s = "\n  Generating your world... please wait.\n"

	case statePlaying:
		logView := m.viewport.View()
		stateView := m.renderState()

		// Join log and state horizontally
		mainView := lipgloss.JoinHorizontal(lipgloss.Top,
			logView,
			stateView,
		)

		help := helpStyle.Render("Commands: /restart, /quit, or just type what you want to do.")

		s = lipgloss.JoinVertical(lipgloss.Left,
			mainView,
			"\n"+m.textInput.View(),
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
	for k, v := range state.Stats {
		if k != "health" && k != "progress" {
			stats += fmt.Sprintf("%s: %s\n", k, v)
		}
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
	return m.gameLog
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
