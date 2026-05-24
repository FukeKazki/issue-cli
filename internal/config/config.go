package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const FileName = "config.yaml"

type Config struct {
	Takt TaktConfig `yaml:"takt"`
}

type TaktConfig struct {
	Workflow    string `yaml:"workflow"`
	WorktreeDir string `yaml:"worktree-dir"`
}

func Load(issuesDir string) (*Config, error) {
	path := filepath.Join(issuesDir, FileName)
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Config{}, nil
		}
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &cfg, nil
}
