package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
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
	stateError
)

type logEntry struct {
	IsUser       bool
	IsSideEffect bool
	Text         string
}

type model struct {
	state       sessionState
	engine      *engine.Engine
	session     *models.GameSession
	textArea    textarea.Model
	viewport    viewport.Model
	spinner     spinner.Model
	err         error
	inputErr    string
	history     []logEntry
	width       int
	height      int
	lastOutcome string
	lastTabIdx  int
	lastSearch  string
	loadingTurn bool
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

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5F5F")).
			Bold(true)

	dialogueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#87D7AF")). // Light green/cyan
			Italic(true)

	boldStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF"))

	sideEffectStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#D7875F")). // Orange/Tan
			Italic(true)
)

func NewModel(eng *engine.Engine) model {
	ta := textarea.New()
	ta.Placeholder = "Enter a hint or 'random'..."
	ta.Focus()
	ta.CharLimit = 156
	ta.SetWidth(40)
	ta.SetHeight(1)
	ta.ShowLineNumbers = false

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return model{
		state:      stateInputHint,
		engine:     eng,
		textArea:   ta,
		spinner:    s,
		lastTabIdx: -1,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, m.spinner.Tick)
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
	case spinner.TickMsg:
		var sCmd tea.Cmd
		m.spinner, sCmd = m.spinner.Update(msg)
		return m, sCmd

	case tea.KeyMsg:
		if msg.Type != tea.KeyTab {
			m.lastTabIdx = -1
			m.lastSearch = ""
		}

		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit

		case tea.KeyTab:
			if m.state == stateInputHint {
				val := m.textArea.Value()
				if strings.HasPrefix(val, "/load ") {
					if m.lastSearch == "" {
						m.lastSearch = strings.TrimPrefix(val, "/load ")
					}
					
					saves, _ := models.ListSessions()
					var matches []string
					for _, s := range saves {
						if strings.HasPrefix(s, m.lastSearch) {
							matches = append(matches, s)
						}
					}

					if len(matches) > 0 {
						m.lastTabIdx = (m.lastTabIdx + 1) % len(matches)
						m.textArea.SetValue("/load " + matches[m.lastTabIdx])
						m.textArea.CursorEnd()
						return m, nil
					}
				}
			}

		case tea.KeyEnter:
			if m.state == stateInputHint {
				hint := strings.TrimSpace(m.textArea.Value())
				if strings.HasPrefix(hint, "/") {
					if strings.HasPrefix(hint, "/load ") {
						name := strings.TrimPrefix(hint, "/load ")
						session, err := models.LoadSession(name)
						if err != nil {
							m.inputErr = fmt.Sprintf("failed to load '%s': %v", name, err)
							m.textArea.Reset()
							return m, nil
						}
						m.session = session
						m.state = statePlaying
						// Reconstruct history
						m.history = nil
						m.history = append(m.history, logEntry{
							IsUser: false,
							Text:   fmt.Sprintf("%s\nLocation: %s\n\n%s", m.session.World.Title, m.session.State.CurrentLocation, m.session.World.Description),
						})
											for _, entry := range m.session.History.Entries {
												m.history = append(m.history, logEntry{IsUser: true, Text: entry.PlayerAction})
												m.history = append(m.history, logEntry{IsUser: false, Text: entry.Outcome})
												
																		if len(entry.Changes) > 0 {
																			m.history = append(m.history, logEntry{
																				IsSideEffect: true,
																				Text:         m.formatSideEffects(entry.Changes),
																			})
																		}											}
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
					if hint == "/quit" {
						return m, tea.Quit
					}
					// Unrecognized or malformed command on startup
					m.inputErr = fmt.Sprintf("unrecognized command: %s. Valid commands: /load <name>, /quit", hint)
					m.textArea.Reset()
					return m, nil
				}
				m.inputErr = ""
				if hint == "" {
					hint = "random"
				}
				m.state = stateLoading
				return m, tea.Batch(m.generateWorld(hint), m.spinner.Tick)
			}
			if m.state == statePlaying {
				if m.loadingTurn {
					return m, nil
				}
				action := strings.TrimSpace(m.textArea.Value())
				if action == "" {
					return m, nil
				}
				m.textArea.Reset()

				if strings.HasPrefix(action, "/") {
					if action == "/quit" {
						return m, tea.Quit
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

					// Unrecognized command during play
					errMsg := "Unrecognized command. Valid commands: /save <name>, /restart, /quit"
					if action == "/save" {
						errMsg = "Usage: /save <name>"
					}
					m.history = append(m.history, logEntry{IsUser: false, Text: errorStyle.Render(errMsg)})
					m.viewport.SetContent(m.renderLog())
					m.viewport.GotoBottom()
					return m, nil
				}

				m.history = append(m.history, logEntry{IsUser: true, Text: action})
				m.viewport.SetContent(m.renderLog())
				m.viewport.GotoBottom()
				m.loadingTurn = true
				return m, tea.Batch(m.processTurn(action), m.spinner.Tick)
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
		m.loadingTurn = false
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
		m.session.Save(m.session.World.ShortName)
		return m, nil

	case turnProcessedMsg:
		m.loadingTurn = false
		if msg.err != nil {
			m.err = msg.err
			m.state = stateError
			return m, nil
		}
		m.lastOutcome = msg.outcome
		m.history = append(m.history, logEntry{IsUser: false, Text: msg.outcome})

		// Check for side effects in the latest history entry
		if len(m.session.History.Entries) > 0 {
			last := m.session.History.Entries[len(m.session.History.Entries)-1]
			if len(last.Changes) > 0 {
				m.history = append(m.history, logEntry{
					IsSideEffect: true,
					Text:         m.formatSideEffects(last.Changes),
				})
			}
		}

		m.viewport.SetContent(m.renderLog())
		m.viewport.GotoBottom()
		m.session.Save(m.session.World.ShortName)
		return m, nil

	case errMsg:
		m.err = msg.err
		m.state = stateError
		return m, nil
	}

	if m.state == stateInputHint || m.state == statePlaying {
		m.textArea, cmd = m.textArea.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m model) View() string {
	var s string
	wrapStyle := lipgloss.NewStyle().Width(m.width)

	switch m.state {
	case stateInputHint:
		saves, _ := models.ListSessions()
		savesList := ""
		if len(saves) > 0 {
			savesList = "\nOr load a previous game: /load <name> (Press Tab to auto-complete)\nAvailable saves: " + strings.Join(saves, ", ") + "\n"
		}

		welcomeText := fmt.Sprintf(
			"Welcome to the Text Game Generator!\n\n%s\n%s",
			"Give me a hint about the world you want to play in (e.g., 'cyberpunk detective', 'zombie kitchen'):",
			savesList,
		)

		s = wrapStyle.Render(welcomeText)
		if m.inputErr != "" {
			s += "\n\n" + errorStyle.Render(m.inputErr)
		}
		s += "\n" + m.textArea.View()

	case stateLoading:
		s = fmt.Sprintf("\n  %s Generating your world... please wait.\n", m.spinner.View())

	case statePlaying:
		logView := m.viewport.View()
		stateView := m.renderState()

		// Join log and state horizontally
		mainView := lipgloss.JoinHorizontal(lipgloss.Top,
			logView,
			stateView,
		)

		help := helpStyle.Render("Commands: /save <name>, /restart, /quit, or just type what you want to do.")

		var inputArea string
		if m.loadingTurn {
			inputArea = fmt.Sprintf("\n  %s Thinking...", m.spinner.View())
		} else {
			inputArea = "\n" + m.textArea.View()
		}

		s = lipgloss.JoinVertical(lipgloss.Left,
			mainView,
			inputArea,
			"\n"+help,
		)

	case stateError:
		s = wrapStyle.Render(fmt.Sprintf("\n  Error: %v\n\nPress Esc to quit.", m.err))
	}

	return "\n" + s + "\n"
}

func (m model) renderState() string {
	if m.session == nil {
		return ""
	}

	world := m.session.World
	state := m.session.State

	stateWidth := int(float64(m.width) * 0.23) // Leave some room for padding
	wrapState := lipgloss.NewStyle().Width(stateWidth)

	// Title
	title := titleStyle.Render("TITLE") + "\n" + wrapState.Render(world.Title) + "\n\n"

	// Location
	location := titleStyle.Render("LOCATION") + "\n" + wrapState.Render(state.CurrentLocation) + "\n\n"

	// Stats
	statsTitle := titleStyle.Render("STATS") + "\n"
	
	healthName := "Health"
	if hn, ok := world.StatDisplayNames["health"]; ok {
		healthName = hn
	}
	progressName := "Progress"
	if pn, ok := world.StatDisplayNames["progress"]; ok {
		progressName = pn
	}
	
	stats := fmt.Sprintf("%s: %s\n%s: %s\n", healthName, state.Health, progressName, state.Progress)

	var keys []string
	for k := range state.Stats {
		if k != "health" && k != "progress" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	for _, k := range keys {
		name := k
		if dn, ok := world.StatDisplayNames[k]; ok {
			name = dn
		}
		stats += fmt.Sprintf("%s: %s\n", name, state.Stats[k])
	}
	stats += "\n"

	// Inventory
	invTitle := titleStyle.Render("INVENTORY") + "\n"
	inventory := ""
	if len(state.Inventory) == 0 {
		inventory = "(empty)"
	} else {
		for _, item := range state.Inventory {
			inventory += "- " + wrapState.Render(item) + "\n"
		}
	}

	content := title + location + statsTitle + stats + invTitle + inventory

	return stateStyle.Width(stateWidth).Height(m.viewport.Height).Render(content)
}

func (m model) renderLog() string {
	var b strings.Builder
	logWidth := int(float64(m.width) * 0.75)

	for i, entry := range m.history {
		var styled string
		if entry.IsUser {
			styled = userStyle.Width(logWidth).Render("> " + entry.Text)
		} else if entry.IsSideEffect {
			styled = sideEffectStyle.Width(logWidth).Render(entry.Text)
		} else {
			// Parse for bold and dialogue
			styled = m.styleGameText(entry.Text, logWidth)
		}
		b.WriteString(styled)
		
		if i < len(m.history)-1 {
			// If the NEXT entry is a side effect, use single newline
			if m.history[i+1].IsSideEffect {
				b.WriteString("\n")
			} else {
				b.WriteString("\n\n")
			}
		}
	}

	return b.String()
}

func (m model) formatSideEffects(changes map[string]string) string {
	var results []string
	for k, v := range changes {
		name := k
		if m.session != nil {
			if dn, ok := m.session.World.StatDisplayNames[k]; ok {
				name = dn
			}
		}
		results = append(results, fmt.Sprintf("%s: %s", name, v))
	}
	sort.Strings(results)
	return "Effects: " + strings.Join(results, ", ")
}

func (m model) styleGameText(text string, width int) string {
	var final strings.Builder
	var buf strings.Builder

	inBold := false
	inQuote := false

	getStyle := func(b, q bool) lipgloss.Style {
		if b && q {
			return boldStyle.Copy().Inherit(dialogueStyle)
		} else if b {
			return boldStyle
		} else if q {
			return dialogueStyle
		}
		return gameStyle
	}

	lastStyle := getStyle(false, false)

	flush := func() {
		if buf.Len() > 0 {
			final.WriteString(lastStyle.Render(buf.String()))
			buf.Reset()
		}
	}

	for i := 0; i < len(text); i++ {
		// Handle Bold **
		if i+1 < len(text) && text[i] == '*' && text[i+1] == '*' {
			flush()
			inBold = !inBold
			lastStyle = getStyle(inBold, inQuote)
			i++ // Skip second asterisk
			continue
		}

		// Handle Quote "
		if text[i] == '"' {
			if !inQuote {
				// Starting a quote
				flush()
				inQuote = true
				lastStyle = getStyle(inBold, true)
				buf.WriteByte('"')
			} else {
				// Ending a quote
				buf.WriteByte('"')
				flush()
				inQuote = false
				lastStyle = getStyle(inBold, false)
			}
			continue
		}

		buf.WriteByte(text[i])
	}
	flush()

	// Wrap the fully styled text
	return lipgloss.NewStyle().Width(width).Render(final.String())
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
