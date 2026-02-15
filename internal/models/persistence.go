package models

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const SaveDir = ".saves"

func (s *GameSession) Save(name string) error {
	if err := os.MkdirAll(SaveDir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(s)
	if err != nil {
		return err
	}

	path := filepath.Join(SaveDir, name+".yaml")
	return os.WriteFile(path, data, 0644)
}

func LoadSession(name string) (*GameSession, error) {
	path := filepath.Join(SaveDir, name+".yaml")
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

	files, err := os.ReadDir(SaveDir)
	if err != nil {
		return nil, err
	}

	var sessions []string
	for _, f := range files {
		if !f.IsDir() && filepath.Ext(f.Name()) == ".yaml" {
			sessions = append(sessions, strings.TrimSuffix(f.Name(), ".yaml"))
		}
	}
	return sessions, nil
}
