package engine

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"github.com/tatianab/text-game/internal/models"
	"google.golang.org/api/option"
	"gopkg.in/yaml.v3"
)

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
	prompt := fmt.Sprintf(`Create a text-based adventure game based on this hint: "%s".
If the hint is "random", pick a unique and interesting theme.

Use short, punchy paragraphs for the world description. 
Use double newlines between paragraphs for readability.
Use markdown **bold** to highlight important objects, locations, or actions.
Use double quotes "like this" for any spoken dialogue.

Output the initial game state in the following YAML format (use | for multi-line strings):

world:
  title: "The Title of the Game"
  short_name: "short-name-slug"
  description: |
    Detailed description of the world
  possibilities: ["action 1", "action 2"]
  state_schema: "Description of what stats and inventory items are tracked"
  stat_display_names: {"health": "Vitality", "mana": "Spirit Energy"} # Map machine keys to user-friendly names
  stat_polarities: {"health": "good", "mana": "good", "corruption": "bad"} # Define each stat as "good" (higher is better) or "bad" (lower is better)
  win_conditions: "Secret win conditions"
  lose_conditions: "Secret lose conditions (e.g., health reaches 0, specific fatal choices)"
initial_location:
  name: "Starting point"
  description: |
    Detailed description of the starting location
  people: ["Person 1", "Person 2"]
  objects: ["Object 1", "Object 2"]
state:
  inventory: []
  stats: {"health": "100", "mana": "50"}
  current_location: "Starting point"
  health: "100"
  progress: "0%%"

Return ONLY the YAML. No markdown formatting blocks like `+"```yaml"+`.`, hint)

	resp, err := e.model.GenerateContent(ctx, genai.Text(prompt))
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

	prompt := fmt.Sprintf(`You are the game master for a text-based adventure.
World Description: %s
Win Conditions: %s
Lose Conditions: %s
Known Locations:
%s
Current State:
  Location: %s
  Inventory: %v
  Stats: %v
  Health: %s
  Progress: %s

History of previous turns:
%s

The player takes the following action: "%s"

Based on the world rules and the player's action, describe what happens and update the game state.
Use short, punchy paragraphs for the description. 
Use double newlines between paragraphs for readability.
Use markdown **bold** to highlight important objects, locations, or actions.
Use double quotes "like this" for any spoken dialogue.

Output your response in the following YAML format (use | for multi-line strings):

outcome: |
  Narrative description of what happened
status: "PLAYING" # Set to "WON" or "LOST" if the game ends
discovered_location: # Optional: Include ONLY if a brand new location is discovered
  name: "Location Name"
  description: |
    Detailed description
  people: ["Person A"]
  objects: ["Object B"]
explanations:
  - "Narrative explanation of a change (e.g., 'Your Health decreased because you were struck.')"
changes: {"stat_name": "change_value", "item_added": "item_name"} # Briefly list side effects
state:
  inventory: ["updated", "list"]
  stats: {"stat": "value"}
  current_location: "Current location"
  health: "Updated health"
  progress: "Updated progress"

Return ONLY the YAML. No markdown formatting blocks.

If the player meets a Win or Lose condition, describe the final outcome clearly and set the status to "WON" or "LOST".`,
		session.World.Description,
		session.World.WinConditions,
		session.World.LoseConditions,
		knownLocations,
		session.State.CurrentLocation,
		session.State.Inventory,
		session.State.Stats,
		session.State.Health,
		session.State.Progress,
		historyText,
		action,
	)

	resp, err := e.model.GenerateContent(ctx, genai.Text(prompt))
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

	keepCount := 3
	toSummarize := session.History.Entries[:len(session.History.Entries)-keepCount]
	remaining := session.History.Entries[len(session.History.Entries)-keepCount:]

	historyToSummarize := ""
	for _, entry := range toSummarize {
		historyToSummarize += fmt.Sprintf("Action: %s\nOutcome: %s\n", entry.PlayerAction, entry.Outcome)
	}

	prompt := fmt.Sprintf(`The following is a list of actions and outcomes from a text-based adventure game.
Current Summary: %s\n\nNew events to add to the summary:\n%s\n\nProvide a concise, third-person summary of these events that captures the key plot points and state changes.\nReturn ONLY the summary text.`, session.History.Summary, historyToSummarize)

	resp, err := e.model.GenerateContent(ctx, genai.Text(prompt))
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
