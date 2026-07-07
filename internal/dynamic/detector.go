package dynamic

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the tool mappings configuration
type Config struct {
	Version  int                    `yaml:"version"`
	Mappings map[string]*Mapping    `yaml:"mappings"`
	Commands map[string]string      `yaml:"commands"`
}

// Mapping represents a tool mapping for a category
type Mapping struct {
	Description string   `yaml:"description"`
	Tools       []string `yaml:"tools"`
	Keywords    []string `yaml:"keywords"`
	AutoDetect  bool     `yaml:"auto_detect"`
}

// ToolCall represents a tool call to execute
type ToolCall struct {
	Tool string
	Args map[string]any
}

// TaskMapping represents the detected mapping for a task
type TaskMapping struct {
	Category   string
	Mapping    *Mapping
	ToolCalls  []ToolCall
	Source     string // "command", "keyword", "dynamic"
}

// Detector detects which tools to use for a task
type Detector struct {
	config *Config
}

// NewDetector creates a new detector from config file
func NewDetector(configPath string) (*Detector, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return &Detector{config: &config}, nil
}

// Detect determines which tools to use for a task
func (d *Detector) Detect(task string) *TaskMapping {
	// 1. Check for command first
	if category, remaining := d.parseCommand(task); category != "" {
		return &TaskMapping{
			Category:  category,
			Mapping:   d.config.Mappings[category],
			ToolCalls: d.buildToolCalls(category, remaining),
			Source:    "command",
		}
	}

	// 2. Check config keywords
	if mapping, category := d.matchByKeywords(task); mapping != nil {
		return &TaskMapping{
			Category:  category,
			Mapping:   mapping,
			ToolCalls: d.buildToolCalls(category, task),
			Source:    "keyword",
		}
	}

	// 3. Default to review
	return &TaskMapping{
		Category:  "review",
		Mapping:   d.config.Mappings["review"],
		ToolCalls: d.buildToolCalls("review", task),
		Source:    "default",
	}
}

// parseCommand checks if task starts with a command
func (d *Detector) parseCommand(task string) (string, string) {
	parts := strings.SplitN(task, " ", 2)
	if len(parts) == 2 {
		if category, ok := d.config.Commands[parts[0]]; ok {
			return category, parts[1]
		}
	}
	return "", task
}

// matchByKeywords matches task to mapping by keywords
func (d *Detector) matchByKeywords(task string) (*Mapping, string) {
	taskLower := strings.ToLower(task)

	// Score each mapping
	bestScore := 0
	bestCategory := ""

	for category, mapping := range d.config.Mappings {
		score := 0
		for _, keyword := range mapping.Keywords {
			if strings.Contains(taskLower, keyword) {
				score++
			}
		}
		if score > bestScore {
			bestScore = score
			bestCategory = category
		}
	}

	if bestScore > 0 {
		return d.config.Mappings[bestCategory], bestCategory
	}

	return nil, ""
}

// buildToolCalls builds tool calls for a category and task
func (d *Detector) buildToolCalls(category, task string) []ToolCall {
	mapping := d.config.Mappings[category]
	if mapping == nil {
		return nil
	}

	// Extract search terms from task
	searchTerms := d.extractSearchTerms(task)

	var calls []ToolCall

	// Build tool calls based on category
	for _, tool := range mapping.Tools {
		switch tool {
		case "text_search":
			// Search for each term
			for _, term := range searchTerms {
				calls = append(calls, ToolCall{
					Tool: tool,
					Args: map[string]any{"query": term},
				})
			}
		case "outline", "read":
			// These need file paths - skip for now
			// Will be handled by Lexa pipeline
			continue
		case "callers", "trace_deps":
			// These need symbol names - skip for now
			// Will be handled by Lexa pipeline
			continue
		case "audit":
			calls = append(calls, ToolCall{
				Tool: tool,
				Args: map[string]any{},
			})
		}
	}

	return calls
}

// extractSearchTerms extracts search terms from task
func (d *Detector) extractSearchTerms(task string) []string {
	terms := []string{}
	taskLower := strings.ToLower(task)

	// Common code terms to search for
	codeTerms := []string{
		"exec.Command",
		"sql.Query",
		"mutex",
		"goroutine",
		"go func",
		"http.Handle",
		"error",
		"panic",
		"defer",
		"channel",
		"interface",
	}

	for _, term := range codeTerms {
		if strings.Contains(taskLower, strings.ToLower(term)) {
			terms = append(terms, term)
		}
	}

	// If no specific terms found, use task keywords
	if len(terms) == 0 {
		words := strings.Fields(taskLower)
		for _, word := range words {
			if len(word) > 3 { // Skip short words
				terms = append(terms, word)
			}
		}
	}

	return terms
}

// GetMapping returns the mapping for a category
func (d *Detector) GetMapping(category string) *Mapping {
	return d.config.Mappings[category]
}

// GetCategories returns all available categories
func (d *Detector) GetCategories() []string {
	categories := make([]string, 0, len(d.config.Mappings))
	for category := range d.config.Mappings {
		categories = append(categories, category)
	}
	return categories
}
