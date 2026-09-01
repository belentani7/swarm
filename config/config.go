package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds all swarm configuration.
type Config struct {
	SessionName  string  `yaml:"session_name"`
	AgentCLI     string  `yaml:"agent_cli"`
	BudgetLimit  float64 `yaml:"budget_limit"`
	RestartPolicy string `yaml:"restart_policy"`
	DataDir      string  `yaml:"data_dir"`
	Layout       string  `yaml:"layout"`
	MaxAgents    int     `yaml:"max_agents"`
	LogLines     int     `yaml:"log_lines"`
}

// Default returns the default configuration.
func Default() *Config {
	home, _ := os.UserHomeDir()
	return &Config{
		SessionName:   "swarm",
		AgentCLI:      "claude",
		BudgetLimit:   10.0,
		RestartPolicy: "on-failure",
		DataDir:       filepath.Join(home, ".swarm"),
		Layout:        "tiled",
		MaxAgents:     25,
		LogLines:      100,
	}
}

// Load reads configuration from ~/.swarm/config.yaml.
func Load() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Default(), err
	}

	cfg := Default()
	data, err := os.ReadFile(filepath.Join(home, ".swarm", "config.yaml"))
	if err != nil {
		return cfg, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return Default(), err
	}

	return cfg, nil
}

// Save writes configuration to ~/.swarm/config.yaml.
func (c *Config) Save() error {
	if err := os.MkdirAll(c.DataDir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(c.DataDir, "config.yaml"), data, 0644)
}
