package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"github.com/tatianab/text-game/internal/config"
	"github.com/tatianab/text-game/internal/engine"
	"github.com/tatianab/text-game/internal/models"
	"google.golang.org/api/option"
)

const maxTurns = 10

func main() {
	ctx := context.Background()
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize the Game Engine (The "Game Master")
	gmEngine, err := engine.NewEngine(ctx, cfg.GeminiAPIKey)
	if err != nil {
		log.Fatalf("Failed to create GM engine: %v", err)
	}
	defer gmEngine.Close()

	// Initialize the Player LLM
	playerClient, err := genai.NewClient(ctx, option.WithAPIKey(cfg.GeminiAPIKey))
	if err != nil {
		log.Fatalf("Failed to create player client: %v", err)
	}
	defer playerClient.Close()
	playerModel := playerClient.GenerativeModel("gemini-2.5-flash")

	// 1. Get a theme from the Player LLM
	fmt.Println("--- Step 1: Requesting a theme from the Player LLM ---")
	themePrompt := "You are a player about to start a text-based adventure game. Provide a short, creative hint for a game theme (e.g., 'steampunk underwater city', 'noir detective in a world of cats'). Return ONLY the theme string."
	themeResp, err := playerModel.GenerateContent(ctx, genai.Text(themePrompt))
	if err != nil {
		log.Fatalf("Failed to get theme: %v", err)
	}
	theme := strings.TrimSpace(fmt.Sprintf("%v", themeResp.Candidates[0].Content.Parts[0]))
	fmt.Printf("Player chose theme: %s\n\n", theme)

	// 2. Generate the world
	fmt.Println("--- Step 2: Generating World ---")
	session, err := gmEngine.GenerateWorld(ctx, theme)
	if err != nil {
		log.Fatalf("Failed to generate world: %v", err)
	}
	fmt.Printf("Title: %s\n", session.World.Title)
	fmt.Printf("Initial Description: %s\n\n", session.World.Description)

	// 3. Play the game
	for turn := 1; turn <= maxTurns; turn++ {
		fmt.Printf("--- Turn %d ---\n", turn)

		// Ask Player LLM what to do
		action := getPlayerAction(ctx, playerModel, session)
		fmt.Printf("Player Action: %s\n", action)

		// Process Turn
		outcome, status, discovered, err := gmEngine.ProcessTurn(ctx, session, action)
		if err != nil {
			fmt.Printf("Error processing turn: %v\n", err)
			break
		}
		fmt.Printf("GM Outcome: %s\n", outcome)
		fmt.Printf("Status: %s\n", status)
		if discovered != "" {
			fmt.Printf("DISCOVERED: %s\n", discovered)
		}

		if len(session.History.Entries) > 0 {
			last := session.History.Entries[len(session.History.Entries)-1]
			for _, exp := range last.Explanations {
				fmt.Printf("Effect: %s\n", exp)
			}
		}

		fmt.Printf("Stats: Health=%s, Progress=%s, Inventory=%v\n\n", session.State.Health, session.State.Progress, session.State.Inventory)

		// Check for win/lose
		if status == "WON" {
			fmt.Println("Game Ended: Player Won!")
			break
		}
		if status == "LOST" {
			fmt.Println("Game Ended: Player Lost!")
			break
		}
	}
}

func getPlayerAction(ctx context.Context, model *genai.GenerativeModel, session *models.GameSession) string {
	historyText := ""
	for _, entry := range session.History.Entries {
		historyText += fmt.Sprintf("Action: %s\nOutcome: %s\nStatus: %s\n", entry.PlayerAction, entry.Outcome, entry.Status)
	}

	prompt := fmt.Sprintf(`You are playing a text-based adventure game.
World: %s
Current Location: %s
Inventory: %v
Stats: %v

History:
%s

What is your next action? Be creative but stay within the world's logic. Return ONLY the action string, no extra commentary.`,
		session.World.Description,
		session.State.CurrentLocation,
		session.State.Inventory,
		session.State.Stats,
		historyText,
	)

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "examine the area"
	}
	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "look around"
	}
	return strings.TrimSpace(fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0]))
}
