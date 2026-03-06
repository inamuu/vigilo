package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Slack SlackConfig `yaml:"slack"`
}

type SlackConfig struct {
	WebhookURL string `yaml:"webhook_url"`
}

func Load(explicitPath string) (Config, string, error) {
	candidates, err := candidatePaths(explicitPath)
	if err != nil {
		return Config{}, "", err
	}

	for _, path := range candidates {
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			if errors.Is(readErr, os.ErrNotExist) && explicitPath == "" {
				continue
			}

			return Config{}, "", fmt.Errorf("read config %q: %w", path, readErr)
		}

		var cfg Config
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return Config{}, "", fmt.Errorf("parse config %q: %w", path, err)
		}

		return cfg, path, nil
	}

	return Config{}, "", nil
}

func candidatePaths(explicitPath string) ([]string, error) {
	if explicitPath != "" {
		return []string{explicitPath}, nil
	}

	var paths []string
	if xdgConfigHome := os.Getenv("XDG_CONFIG_HOME"); xdgConfigHome != "" {
		paths = append(paths, filepath.Join(xdgConfigHome, "vigilo", "config.yaml"))
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve home directory: %w", err)
	}

	paths = append(paths, filepath.Join(homeDir, ".config", "vigilo", "config.yaml"))
	return paths, nil
}
