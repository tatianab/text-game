package engine

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"strings"
	"text/template"

	"github.com/google/generative-ai-go/genai"
	"github.com/tatianab/text-game/internal/models"
	"google.golang.org/api/option"
	"gopkg.in/yaml.v3"
)

//go:embed prompts/generate_world.txt
var generateWorldPrompt string

//go:embed prompts/process_turn.txt
var processTurnPrompt string

//go:embed prompts/summarize_history.txt
var summarizeHistoryPrompt string

type Engine struct {
	client *genai.Client
	model  *genai.GenerativeModel
}

func NewEngine(ctx context.Context, apiKey string) (*Engine, error) {
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, err
	}

	model := client.GenerativeModel("gemini-2.5-flash")
	// Ensure we get structured output as much as possible, though we'll parse YAML.
	return &Engine{
		client: client,
		model:  model,
	}, nil
}

func (e *Engine) Close() {
	e.client.Close()
}

func (e *Engine) GenerateWorld(ctx context.Context, hint string) (*models.GameSession, error) {
	tmpl, err := template.New("generate_world").Parse(generateWorldPrompt)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, struct{ Hint string }{Hint: hint}); err != nil {
		return nil, err
	}

	resp, err := e.model.GenerateContent(ctx, genai.Text(buf.String()))
	if err != nil {
		return nil, err
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no content returned from Gemini")
	}

	part := resp.Candidates[0].Content.Parts[0]
	text, ok := part.(genai.Text)
	if !ok {
		return nil, fmt.Errorf("unexpected response type from Gemini")
	}

	cleanYAML := strings.TrimSpace(string(text))
	cleanYAML = strings.TrimPrefix(cleanYAML, "```yaml")
	cleanYAML = strings.TrimPrefix(cleanYAML, "```")
	cleanYAML = strings.TrimSuffix(cleanYAML, "```")

	var respData struct {
		World           models.World    `yaml:"world"`
		InitialLocation models.Location `yaml:"initial_location"`
		State           models.GameState `yaml:"state"`
	}
	err = yaml.Unmarshal([]byte(cleanYAML), &respData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %v\nOutput was: %s", err, cleanYAML)
	}

	session := &models.GameSession{
		World:     respData.World,
		State:     respData.State,
		Locations: make(map[string]models.Location),
	}
	if respData.InitialLocation.Name != "" {
		session.Locations[respData.InitialLocation.Name] = respData.InitialLocation
	}

	return session, nil
}

func (e *Engine) ProcessTurn(ctx context.Context, session *models.GameSession, action string) (string, string, string, error) {
	// If history is too long, summarize it
	if len(session.History.Entries) > 8 {
		if err := e.SummarizeHistory(ctx, session); err != nil {
			// Log error but continue with full history for now
			fmt.Printf("Warning: failed to summarize history: %v\n", err)
		}
	}

	historyText := ""
	if session.History.Summary != "" {
		historyText = fmt.Sprintf("Summary of previous events: %s\n\n", session.History.Summary)
	}
	for _, entry := range session.History.Entries {
		historyText += fmt.Sprintf("Action: %s\nOutcome: %s\nStatus: %s\n", entry.PlayerAction, entry.Outcome, entry.Status)
		if len(entry.Changes) > 0 {
			historyText += fmt.Sprintf("Side Effects: %v\n", entry.Changes)
		}
		if len(entry.Inventory) > 0 {
			historyText += fmt.Sprintf("Inventory: %v\n", entry.Inventory)
		}
	}

	knownLocations := ""
	for name, loc := range session.Locations {
		knownLocations += fmt.Sprintf("- %s: %s (People: %v, Objects: %v)\n", name, loc.Description, loc.People, loc.Objects)
	}

	tmpl, err := template.New("process_turn").Parse(processTurnPrompt)
	if err != nil {
		return "", "", "", err
	}

	var buf bytes.Buffer
	data := struct {
		WorldDescription string
		WinConditions    string
		LoseConditions   string
		KnownLocations   string
		CurrentLocation  string
		Inventory        []string
		Stats            map[string]string
		Health           string
		Progress         string
		History          string
		Action           string
	}{
		WorldDescription: session.World.Description,
		WinConditions:    session.World.WinConditions,
		LoseConditions:   session.World.LoseConditions,
		KnownLocations:   knownLocations,
		CurrentLocation:  session.State.CurrentLocation,
		Inventory:        session.State.Inventory,
		Stats:            session.State.Stats,
		Health:           session.State.Health,
		Progress:         session.State.Progress,
		History:          historyText,
		Action:           action,
	}

	if err := tmpl.Execute(&buf, data); err != nil {
		return "", "", "", err
	}

	resp, err := e.model.GenerateContent(ctx, genai.Text(buf.String()))
	if err != nil {
		return "", "", "", err
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", "", "", fmt.Errorf("no content returned from Gemini")
	}

	part := resp.Candidates[0].Content.Parts[0]
	text, ok := part.(genai.Text)
	if !ok {
		return "", "", "", fmt.Errorf("unexpected response type from Gemini")
	}

	cleanYAML := strings.TrimSpace(string(text))
	cleanYAML = strings.TrimPrefix(cleanYAML, "```yaml")
	cleanYAML = strings.TrimPrefix(cleanYAML, "```")
	cleanYAML = strings.TrimSuffix(cleanYAML, "```")

	type TurnResult struct {
		Outcome            string            `yaml:"outcome"`
		Status             string            `yaml:"status"`
		DiscoveredLocation *models.Location  `yaml:"discovered_location"`
		Explanations       []string          `yaml:"explanations"`
		Changes            map[string]string `yaml:"changes"`
		State              models.GameState  `yaml:"state"`
	}

	var result TurnResult
	err = yaml.Unmarshal([]byte(cleanYAML), &result)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to parse turn YAML: %v\nOutput was: %s", err, cleanYAML)
	}

	// Update session
	session.State = result.State
	discoveredName := ""
	if result.DiscoveredLocation != nil && result.DiscoveredLocation.Name != "" {
		discoveredName = result.DiscoveredLocation.Name
		if session.Locations == nil {
			session.Locations = make(map[string]models.Location)
		}
		session.Locations[result.DiscoveredLocation.Name] = *result.DiscoveredLocation
	}
	session.History.Entries = append(session.History.Entries, models.HistoryEntry{
		PlayerAction: action,
		Outcome:      result.Outcome,
		Status:       result.Status,
		Explanations: result.Explanations,
		Changes:      result.Changes,
		Inventory:    result.State.Inventory,
	})

	return result.Outcome, result.Status, discoveredName, nil
}

func (e *Engine) SummarizeHistory(ctx context.Context, session *models.GameSession) error {
	if len(session.History.Entries) <= 5 {
		return nil
	}

	// We'll keep the last 3 entries and summarize the rest
	keepCount := 3
	toSummarize := session.History.Entries[:len(session.History.Entries)-keepCount]
	remaining := session.History.Entries[len(session.History.Entries)-keepCount:]

	historyToSummarize := ""
	for _, entry := range toSummarize {
		historyToSummarize += fmt.Sprintf("Action: %s\nOutcome: %s\n", entry.PlayerAction, entry.Outcome)
	}

	tmpl, err := template.New("summarize_history").Parse(summarizeHistoryPrompt)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	data := struct {
		CurrentSummary string
		NewEvents      string
	}{
		CurrentSummary: session.History.Summary,
		NewEvents:      historyToSummarize,
	}

	if err := tmpl.Execute(&buf, data); err != nil {
		return err
	}

	resp, err := e.model.GenerateContent(ctx, genai.Text(buf.String()))
	if err != nil {
		return err
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return fmt.Errorf("no content returned from Gemini during summarization")
	}

	part := resp.Candidates[0].Content.Parts[0]
	text, ok := part.(genai.Text)
	if !ok {
		return fmt.Errorf("unexpected response type from Gemini during summarization")
	}

	session.History.Summary = strings.TrimSpace(string(text))
	session.History.Entries = remaining
	return nil
}
