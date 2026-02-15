package models

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const SaveDir = ".saves"

func (s *GameSession) Save(name string) error {
	dir := filepath.Join(SaveDir, name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(s)
	if err != nil {
		return err
	}

	path := filepath.Join(dir, "game.yaml")
	return os.WriteFile(path, data, 0644)
}

func LoadSession(name string) (*GameSession, error) {
	path := filepath.Join(SaveDir, name, "game.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var session GameSession
	if err := yaml.Unmarshal(data, &session); err != nil {
		return nil, err
	}

	return &session, nil
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
			gamePath := filepath.Join(SaveDir, entry.Name(), "game.yaml")
			if _, err := os.Stat(gamePath); err == nil {
				sessions = append(sessions, entry.Name())
			}
		}
	}
	return sessions, nil
}
