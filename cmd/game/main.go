package main

import (
	"context"
	"fmt"
	"os"

	"github.com/tatianab/text-game/internal/config"
	"github.com/tatianab/text-game/internal/engine"
	"github.com/tatianab/text-game/internal/models"
	"github.com/tatianab/text-game/internal/tui"
)

func main() {
	ctx := context.Background()

	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	models.SaveDir = cfg.SaveDir

	eng, err := engine.NewEngine(ctx, cfg.GeminiAPIKey)
	if err != nil {
		fmt.Printf("Error creating engine: %v\n", err)
		os.Exit(1)
	}
	defer eng.Close()

	if err := tui.Run(eng); err != nil {
		fmt.Printf("Error running TUI: %v\n", err)
		os.Exit(1)
	}
}
