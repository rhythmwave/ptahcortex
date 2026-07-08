package setup

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the agent configuration
type Config struct {
	Name        string      `yaml:"name"`
	Description string      `yaml:"description"`
	LLM         LLMConfig   `yaml:"llm"`
	MCPServers  []MCPServer `yaml:"mcp_servers,omitempty"`
	Tools       ToolsConfig `yaml:"tools"`
	Agent       AgentConfig `yaml:"agent"`
}

type LLMConfig struct {
	Provider  string `yaml:"provider"`
	Model     string `yaml:"model"`
	BaseURL   string `yaml:"base_url"`
	APIKey    string `yaml:"api_key"`
	MaxTokens int    `yaml:"max_tokens"`
}

type MCPServer struct {
	Name    string   `yaml:"name"`
	Command string   `yaml:"command"`
	Args    []string `yaml:"args,omitempty"`
	CWD     string   `yaml:"cwd,omitempty"`
}

type ToolsConfig struct {
	MaxParallel int    `yaml:"max_parallel"`
	Timeout     string `yaml:"timeout"`
}

type AgentConfig struct {
	MaxIterations   int `yaml:"max_iterations"`
	MaxTokensPerRun int `yaml:"max_tokens_per_run"`
}

// RunSetup runs the interactive setup wizard
func RunSetup() error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("═══════════════════════════════════════════")
	fmt.Println("  Ptahcortex Setup Wizard")
	fmt.Println("═══════════════════════════════════════════")
	fmt.Println()

	// Get config directory
	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, ".ptahcortex")
	os.MkdirAll(configDir, 0755)

	configPath := filepath.Join(configDir, "config.yaml")

	// Check if config exists
	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("Config already exists at: %s\n", configPath)
		fmt.Print("Overwrite? (y/N): ")
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			fmt.Println("Setup cancelled.")
			return nil
		}
	}

	config := Config{
		Name:        "ptahcortex",
		Description: "AI Agent with MCP tool calling",
		LLM: LLMConfig{
			Provider:  "openai",
			Model:     "mimo-v2.5",
			MaxTokens: 8192,
		},
		Tools: ToolsConfig{
			MaxParallel: 3,
			Timeout:     "30s",
		},
		Agent: AgentConfig{
			MaxIterations:   5,
			MaxTokensPerRun: 50000,
		},
	}

	// LLM Configuration
	fmt.Println("\n── LLM Configuration ──")
	fmt.Print("Provider (openai/anthropic) [openai]: ")
	provider, _ := reader.ReadString('\n')
	provider = strings.TrimSpace(provider)
	if provider == "" {
		provider = "openai"
	}
	config.LLM.Provider = provider

	fmt.Print("Model [mimo-v2.5]: ")
	model, _ := reader.ReadString('\n')
	model = strings.TrimSpace(model)
	if model == "" {
		model = "mimo-v2.5"
	}
	config.LLM.Model = model

	fmt.Print("API Base URL: ")
	baseURL, _ := reader.ReadString('\n')
	baseURL = strings.TrimSpace(baseURL)
	config.LLM.BaseURL = baseURL

	fmt.Print("API Key: ")
	apiKey, _ := reader.ReadString('\n')
	apiKey = strings.TrimSpace(apiKey)
	config.LLM.APIKey = apiKey

	// MCP Servers (optional)
	fmt.Println("\n── MCP Servers (optional) ──")
	fmt.Print("Add Lexa MCP server? (Y/n): ")
	addLexa, _ := reader.ReadString('\n')
	addLexa = strings.TrimSpace(strings.ToLower(addLexa))

	if addLexa != "n" && addLexa != "no" {
		lexaPath := findLexa()
		if lexaPath != "" {
			fmt.Printf("Found Lexa at: %s\n", lexaPath)
			config.MCPServers = append(config.MCPServers, MCPServer{
				Name:    "lexa",
				Command: lexaPath,
				Args:    []string{"mcp"},
			})
		} else {
			fmt.Print("Lexa binary path (leave empty to skip): ")
			lexaPath, _ = reader.ReadString('\n')
			lexaPath = strings.TrimSpace(lexaPath)
			if lexaPath != "" {
				config.MCPServers = append(config.MCPServers, MCPServer{
					Name:    "lexa",
					Command: lexaPath,
					Args:    []string{"mcp"},
				})
			}
		}
	}

	// Save config
	data, err := yaml.Marshal(&config)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	fmt.Printf("\n✓ Config saved to: %s\n", configPath)
	fmt.Println("\nUsage:")
	fmt.Printf("  ptahcortex --config %s --smart\n", configPath)
	fmt.Println("\nOr run without --task for interactive mode:")
	fmt.Printf("  ptahcortex --config %s --smart\n", configPath)

	return nil
}

// findLexa searches for Lexa binary in common locations
func findLexa() string {
	locations := []string{
		"/usr/local/bin/lexa",
		"/usr/bin/lexa",
		"/opt/lexa/lexa",
	}

	homeDir, _ := os.UserHomeDir()
	locations = append(locations,
		filepath.Join(homeDir, ".local/bin/lexa"),
		filepath.Join(homeDir, "bin/lexa"),
	)

	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			return loc
		}
	}

	return ""
}

// RunDoctor checks dependencies and configuration
func RunDoctor() error {
	fmt.Println("═══════════════════════════════════════════")
	fmt.Println("  Ptahcortex Doctor")
	fmt.Println("═══════════════════════════════════════════")
	fmt.Println()

	checks := []struct {
		name    string
		check   func() bool
		message string
	}{
		{"Config file", checkConfig, "Run 'ptahcortex init' to create config"},
		{"Lexa binary", checkLexa, "Install Lexa from github.com/anvia-hq/lexa"},
		{"Go runtime", checkGo, "Install Go from go.dev"},
		{"Network", checkNetwork, "Check internet connection"},
	}

	allPassed := true
	for _, c := range checks {
		if c.check() {
			fmt.Printf("  ✓ %s\n", c.name)
		} else {
			fmt.Printf("  ✗ %s - %s\n", c.name, c.message)
			allPassed = false
		}
	}

	fmt.Println()
	if allPassed {
		fmt.Println("All checks passed! Ready to use Ptahcortex.")
	} else {
		fmt.Println("Some checks failed. Fix issues above before using Ptahcortex.")
	}

	return nil
}

func checkConfig() bool {
	homeDir, _ := os.UserHomeDir()
	configPath := filepath.Join(homeDir, ".ptahcortex", "config.yaml")
	_, err := os.Stat(configPath)
	return err == nil
}

func checkLexa() bool {
	return findLexa() != ""
}

func checkGo() bool {
	_, err := os.Stat("/usr/local/go/bin/go")
	return err == nil
}

func checkNetwork() bool {
	_, err := os.Stat("/etc/resolv.conf")
	return err == nil
}
