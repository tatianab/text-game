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
				m.gameLog += fmt.Sprintf("\n> %s\n", action)
				m.viewport.SetContent(m.renderLog())
				m.viewport.GotoBottom()
				return m, m.processTurn(action)
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - 6
		if m.state == statePlaying {
			m.viewport.SetContent(m.renderLog())
		}

	case worldGeneratedMsg:
		m.session = msg.session
		m.state = statePlaying
		m.gameLog = fmt.Sprintf("World: %s\n\n%s\n", m.session.State.CurrentLocation, m.session.World.Description)
		if m.viewport.Width == 0 {
			m.viewport = viewport.New(m.width, m.height-6)
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
		m.gameLog += fmt.Sprintf("\n%s\n", msg.outcome)
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
		stats := fmt.Sprintf("Health: %s | Progress: %s | Location: %s", 
			m.session.State.Health, m.session.State.Progress, m.session.State.CurrentLocation)
		
		s = lipgloss.JoinVertical(lipgloss.Left,
			m.viewport.View(),
			"\n"+stats,
			"\n"+m.textInput.View(),
		)

	case stateError:
		s = fmt.Sprintf("\n  Error: %v\n\nPress Esc to quit.", m.err)
	}

	return "\n" + s + "\n"
}

func (m model) renderLog() string {
	return lipgloss.NewStyle().Width(m.width).Render(m.gameLog)
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
