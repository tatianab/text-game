package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// Config holds the application configuration.
type Config struct {
	GeminiAPIKey string
	SaveDir      string
}

// LoadConfig loads the configuration from environment variables and defaults.
func LoadConfig() (*Config, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY environment variable is not set.\n\n" +
			"To play this game, you need a Google Gemini API key.\n" +
			"1. Get a free key at https://aistudio.google.com/app/apikey\n" +
			"2. Set it in your terminal: export GEMINI_API_KEY='your-key-here'\n" +
			"3. Run the game again.")
	}

	saveDir := os.Getenv("TEXT_GAME_SAVE_DIR")
	if saveDir == "" {
		configDir, err := os.UserConfigDir()
		if err != nil {
			// Fallback to local directory if we can't find config dir
			saveDir = ".saves"
		} else {
			saveDir = filepath.Join(configDir, "text-game", "saves")
		}
	}

	return &Config{
		GeminiAPIKey: apiKey,
		SaveDir:      saveDir,
	}, nil
}
