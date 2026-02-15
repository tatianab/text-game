package models

// World represents the static (or semi-static) world definition.
type World struct {
	Description   string   `yaml:"description"`
	Possibilities []string `yaml:"possibilities"` // e.g., what sorts of actions a player can take
	StateSchema   string   `yaml:"state_schema"`  // description of what sort of state will be held
	WinConditions string   `yaml:"win_conditions"`
}

// GameState represents the current dynamic state of the game.
type GameState struct {
	Inventory       []string          `yaml:"inventory"`
	Stats           map[string]string `yaml:"stats"`
	CurrentLocation string            `yaml:"current_location"`
	Health          string            `yaml:"health"`
	Progress        string            `yaml:"progress"`
}

// HistoryEntry represents a single turn in the game.
type HistoryEntry struct {
	PlayerAction string `yaml:"player_action"`
	Outcome      string `yaml:"outcome"`
}

// GameHistory contains the abbreviated history of the game.
type GameHistory struct {
	Entries []HistoryEntry `yaml:"entries"`
}

// GameSession aggregates all game-related data.
type GameSession struct {
	World   World       `yaml:"world"`
	State   GameState   `yaml:"state"`
	History GameHistory `yaml:"history"`
}
