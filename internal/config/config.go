package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config is the top-level agent configuration.
type Config struct {
	Name        string        `yaml:"name"`
	Description string        `yaml:"description"`
	LLM         LLMConfig     `yaml:"llm"`
	MCPServers  []MCPServer   `yaml:"mcp_servers"`
	Tools       ToolsConfig   `yaml:"tools"`
	Agent       AgentConfig   `yaml:"agent"`
}

type LLMConfig struct {
	Provider   string `yaml:"provider"`    // openai, anthropic
	BaseURL    string `yaml:"base_url"`
	APIKey     string `yaml:"api_key"`
	Model      string `yaml:"model"`
	MaxTokens  int    `yaml:"max_tokens"`
}

type MCPServer struct {
	Name    string   `yaml:"name"`
	Command string   `yaml:"command"`
	Args    []string `yaml:"args"`
	CWD     string   `yaml:"cwd"`
}

type ToolsConfig struct {
	MaxParallel int    `yaml:"max_parallel"`
	Timeout     string `yaml:"timeout"`
}

type AgentConfig struct {
	MaxIterations   int `yaml:"max_iterations"`
	MaxTokensPerRun int `yaml:"max_tokens_per_run"`
}

// Load reads and parses a YAML config file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}

	// Defaults
	if cfg.LLM.MaxTokens == 0 {
		cfg.LLM.MaxTokens = 4096
	}
	if cfg.Tools.MaxParallel == 0 {
		cfg.Tools.MaxParallel = 5
	}
	if cfg.Agent.MaxIterations == 0 {
		cfg.Agent.MaxIterations = 5
	}
	if cfg.Agent.MaxTokensPerRun == 0 {
		cfg.Agent.MaxTokensPerRun = 50000
	}

	return &cfg, nil
}
