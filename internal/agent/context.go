package agent

import (
	"fmt"
	"strings"
)

// ContextManager manages minimal context that grows based on interaction
type ContextManager struct {
	// Base context (always included)
	baseTokens int
	
	// Tracked context additions
	addedContext map[string]int
	
	// Token budget
	maxTokens int
	currentTokens int
}

// NewContextManager creates a minimal context manager
func NewContextManager(maxTokens int) *ContextManager {
	return &ContextManager{
		baseTokens:    200, // Minimal system prompt
		addedContext:  make(map[string]int),
		maxTokens:     maxTokens,
		currentTokens: 200,
	}
}

// GetMinimalPrompt returns the smallest possible prompt
func (cm *ContextManager) GetMinimalPrompt(task string) string {
	return fmt.Sprintf(`Task: %s

Use available tools to complete this task.
Return findings when done.`, task)
}

// ShouldAddContext checks if we should add more context
func (cm *ContextManager) ShouldAddContext(contextType string, tokens int) bool {
	// Don't add if already added
	if _, exists := cm.addedContext[contextType]; exists {
		return false
	}
	
	// Don't add if would exceed budget
	if cm.currentTokens+tokens > cm.maxTokens {
		return false
	}
	
	return true
}

// AddContext adds context and tracks tokens
func (cm *ContextManager) AddContext(contextType string, content string, tokens int) {
	if cm.ShouldAddContext(contextType, tokens) {
		cm.addedContext[contextType] = tokens
		cm.currentTokens += tokens
	}
}

// GetProgressivePrompt builds prompt that grows based on interaction
func (cm *ContextManager) GetProgressivePrompt(task string, iteration int, previousResults map[string]string) string {
	var prompt strings.Builder
	
	// Always: minimal task prompt
	prompt.WriteString(cm.GetMinimalPrompt(task))
	
	// Iteration 1+: Add tool results if available
	if iteration > 0 && len(previousResults) > 0 {
		if cm.ShouldAddContext("tool_results", 500) {
			prompt.WriteString("\n\nPrevious findings:\n")
			for k, v := range previousResults {
				// Truncate to save tokens
				if len(v) > 100 {
					v = v[:100] + "..."
				}
				prompt.WriteString(fmt.Sprintf("- %s: %s\n", k, v))
			}
			cm.AddContext("tool_results", "previous", 500)
		}
	}
	
	// Iteration 2+: Add code context if needed
	if iteration >= 2 {
		if cm.ShouldAddContext("code_context", 800) {
			prompt.WriteString("\n\nAnalyze the code and provide specific findings.")
			cm.AddContext("code_context", "analysis", 800)
		}
	}
	
	// Iteration 3+: Add full analysis request
	if iteration >= 3 {
		if cm.ShouldAddContext("full_analysis", 1000) {
			prompt.WriteString(`

Provide comprehensive analysis with:
1. Specific file paths and line numbers
2. Severity ratings (Critical/High/Medium/Low)
3. Attack vectors
4. Code patches`)
			cm.AddContext("full_analysis", "detailed", 1000)
		}
	}
	
	return prompt.String()
}

// GetTokenUsage returns current token usage
func (cm *ContextManager) GetTokenUsage() (current int, max int, percentage float64) {
	return cm.currentTokens, cm.maxTokens, float64(cm.currentTokens) / float64(cm.maxTokens) * 100
}
