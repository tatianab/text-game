package models

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	SaveDir            = ".saves"
	CurrentSaveVersion = "1"
)

type versionInfo struct {
	Version string `yaml:"version"`
}

func (s *GameSession) Save(name string) error {
	dir := filepath.Join(SaveDir, name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Save version.yaml
	vData, err := yaml.Marshal(versionInfo{Version: CurrentSaveVersion})
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "version.yaml"), vData, 0644); err != nil {
		return err
	}

	// Save world.yaml
	worldData, err := yaml.Marshal(s.World)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "world.yaml"), worldData, 0644); err != nil {
		return err
	}

	// Save state.yaml
	stateData, err := yaml.Marshal(s.State)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "state.yaml"), stateData, 0644); err != nil {
		return err
	}

	// Save history.yaml
	historyData, err := yaml.Marshal(s.History)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "history.yaml"), historyData, 0644); err != nil {
		return err
	}

	// Save locations
	if len(s.Locations) > 0 {
		locDir := filepath.Join(dir, "locations")
		if err := os.MkdirAll(locDir, 0755); err != nil {
			return err
		}
		for name, loc := range s.Locations {
			locData, err := yaml.Marshal(loc)
			if err != nil {
				return err
			}
			// Sanitize name for filename
			safeName := strings.ReplaceAll(strings.ToLower(name), " ", "-")
			if err := os.WriteFile(filepath.Join(locDir, safeName+".yaml"), locData, 0644); err != nil {
				return err
			}
		}
	}

	return nil
}

func LoadSession(name string) (*GameSession, error) {
	dir := filepath.Join(SaveDir, name)

	// Check version
	vData, err := os.ReadFile(filepath.Join(dir, "version.yaml"))
	if err != nil {
		return nil, fmt.Errorf("could not read version info (save may be too old): %v", err)
	}
	var vInfo versionInfo
	if err := yaml.Unmarshal(vData, &vInfo); err != nil {
		return nil, err
	}
	if vInfo.Version != CurrentSaveVersion {
		return nil, fmt.Errorf("incompatible save version: found %s, want %s", vInfo.Version, CurrentSaveVersion)
	}

	// Load world
	worldData, err := os.ReadFile(filepath.Join(dir, "world.yaml"))
	if err != nil {
		return nil, err
	}
	var world World
	if err := yaml.Unmarshal(worldData, &world); err != nil {
		return nil, err
	}

	// Load state
	stateData, err := os.ReadFile(filepath.Join(dir, "state.yaml"))
	if err != nil {
		return nil, err
	}
	var state GameState
	if err := yaml.Unmarshal(stateData, &state); err != nil {
		return nil, err
	}

	// Load history
	historyData, err := os.ReadFile(filepath.Join(dir, "history.yaml"))
	if err != nil {
		return nil, err
	}
	var history GameHistory
	if err := yaml.Unmarshal(historyData, &history); err != nil {
		return nil, err
	}

	// Load locations
	locations := make(map[string]Location)
	locDir := filepath.Join(dir, "locations")
	if _, err := os.Stat(locDir); err == nil {
		entries, err := os.ReadDir(locDir)
		if err == nil {
			for _, entry := range entries {
				if !entry.IsDir() && filepath.Ext(entry.Name()) == ".yaml" {
					locData, err := os.ReadFile(filepath.Join(locDir, entry.Name()))
					if err == nil {
						var loc Location
						if err := yaml.Unmarshal(locData, &loc); err == nil {
							locations[loc.Name] = loc
						}
					}
				}
			}
		}
	}

	return &GameSession{
		World:     world,
		State:     state,
		History:   history,
		Locations: locations,
	}, nil
}

func ListSessions() ([]string, error) {
	if _, err := os.Stat(SaveDir); os.IsNotExist(err) {
		return []string{}, nil
	}

	entries, err := os.ReadDir(SaveDir)
	if err != nil {
		return nil, err
	}

	var sessions []string
	for _, entry := range entries {
		if entry.IsDir() {
			// Check if version.yaml exists as a marker for a valid session
			vPath := filepath.Join(SaveDir, entry.Name(), "version.yaml")
			if _, err := os.Stat(vPath); err == nil {
				sessions = append(sessions, entry.Name())
			}
		}
	}
	return sessions, nil
}
