package models

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestGameSessionYAML(t *testing.T) {
	session := &GameSession{
		World: World{
			Title:         "The Hidden Manor",
			ShortName:     "hidden-manor",
			Description:   "A dark forest",
			Possibilities: []string{"look", "walk"},
			StateSchema:   "health and inventory",
			WinConditions: "Find the key",
		},
		State: GameState{
			Inventory: []string{"map"},
			Stats: map[string]string{"health": "100"},
			CurrentLocation: "Entrance",
			Health: "100",
			Progress: "0%",
		},
		History: GameHistory{
			Entries: []HistoryEntry{
				{PlayerAction: "look", Outcome: "You see trees."},
			},
		},
	}

	data, err := yaml.Marshal(session)
	if err != nil {
		t.Fatalf("Failed to marshal session: %v", err)
	}

	var session2 GameSession
	err = yaml.Unmarshal(data, &session2)
	if err != nil {
		t.Fatalf("Failed to unmarshal session: %v", err)
	}

	if session2.World.Description != session.World.Description {
		t.Errorf("Expected description %s, got %s", session.World.Description, session2.World.Description)
	}

	if len(session2.History.Entries) != 1 {
		t.Errorf("Expected 1 history entry, got %d", len(session2.History.Entries))
	}
}
