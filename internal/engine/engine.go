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
  win_conditions: "Secret win conditions"
  lose_conditions: "Secret lose conditions (e.g., health reaches 0, specific fatal choices)"
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

	var session models.GameSession
	err = yaml.Unmarshal([]byte(cleanYAML), &session)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %v\nOutput was: %s", err, cleanYAML)
	}

	return &session, nil
}

func (e *Engine) ProcessTurn(ctx context.Context, session *models.GameSession, action string) (string, string, error) {
	historyText := ""
	for _, entry := range session.History.Entries {
		historyText += fmt.Sprintf("Action: %s\nOutcome: %s\nStatus: %s\n", entry.PlayerAction, entry.Outcome, entry.Status)
		if len(entry.Changes) > 0 {
			historyText += fmt.Sprintf("Side Effects: %v\n", entry.Changes)
		}
		if len(entry.Inventory) > 0 {
			historyText += fmt.Sprintf("Inventory: %v\n", entry.Inventory)
		}
	}

	prompt := fmt.Sprintf(`You are the game master for a text-based adventure.
World Description: %s
Win Conditions: %s
Lose Conditions: %s
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
		return "", "", err
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", "", fmt.Errorf("no content returned from Gemini")
	}

	part := resp.Candidates[0].Content.Parts[0]
	text, ok := part.(genai.Text)
	if !ok {
		return "", "", fmt.Errorf("unexpected response type from Gemini")
	}

	cleanYAML := strings.TrimSpace(string(text))
	cleanYAML = strings.TrimPrefix(cleanYAML, "```yaml")
	cleanYAML = strings.TrimPrefix(cleanYAML, "```")
	cleanYAML = strings.TrimSuffix(cleanYAML, "```")

	type TurnResult struct {
		Outcome string            `yaml:"outcome"`
		Status  string            `yaml:"status"`
		Changes map[string]string `yaml:"changes"`
		State   models.GameState  `yaml:"state"`
	}

	var result TurnResult
	err = yaml.Unmarshal([]byte(cleanYAML), &result)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse turn YAML: %v\nOutput was: %s", err, cleanYAML)
	}

	// Update session
	session.State = result.State
	session.History.Entries = append(session.History.Entries, models.HistoryEntry{
		PlayerAction: action,
		Outcome:      result.Outcome,
		Status:       result.Status,
		Changes:      result.Changes,
		Inventory:    result.State.Inventory,
	})

	return result.Outcome, result.Status, nil
}
