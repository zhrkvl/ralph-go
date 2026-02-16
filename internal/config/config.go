package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	ConfigVersion         string         `toml:"configVersion"`
	MaxIterations         int            `toml:"maxIterations"`
	Agent                 string         `toml:"agent"`
	Tracker               string         `toml:"tracker"`
	AutoCommit            bool           `toml:"autoCommit"`
	SubagentTracingDetail string         `toml:"subagentTracingDetail"`
	AgentOptions          map[string]any `toml:"agentOptions"`
	TrackerOptions        map[string]any `toml:"trackerOptions"`
}

func DefaultConfig() *Config {
	return &Config{
		ConfigVersion:         "2.1",
		MaxIterations:         10,
		Agent:                 "amp",
		Tracker:               "json",
		AutoCommit:            true,
		SubagentTracingDetail: "full",
		AgentOptions:          map[string]any{},
		TrackerOptions:        map[string]any{},
	}
}

// Load reads .ralph-tui/config.toml from the given project dir.
// If missing, returns defaults.
func Load(projectDir string) (*Config, error) {
	path := filepath.Join(projectDir, ".ralph-tui", "config.toml")
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	if _, err := toml.Decode(string(data), cfg); err != nil {
		return nil, fmt.Errorf("parsing config.toml: %w", err)
	}
	return cfg, nil
}

// Save writes the config to .ralph-tui/config.toml.
func (c *Config) Save(projectDir string) error {
	dir := filepath.Join(projectDir, ".ralph-tui")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	path := filepath.Join(dir, "config.toml")
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(c)
}
