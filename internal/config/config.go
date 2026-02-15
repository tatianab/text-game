package config

import (
	"fmt"
	"os"
)

// Config holds the application configuration.
type Config struct {
	GeminiAPIKey string
}

// LoadConfig loads the configuration from environment variables.
func LoadConfig() (*Config, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY environment variable is not set")
	}

	return &Config{
		GeminiAPIKey: apiKey,
	}, nil
}
