package agent

import (
	"strings"

	"github.com/rhythmwave/ptahcortex/internal/llm"
)

// ContextManager filters and optimizes messages before sending to LLM.
type ContextManager struct {
	maxToolResultLen int // max chars per tool result
	maxMessages      int // max messages to keep in history
}

// NewContextManager creates a context manager with sensible defaults.
func NewContextManager() *ContextManager {
	return &ContextManager{
		maxToolResultLen: 4000,  // ~1000 tokens
		maxMessages:      20,    // keep last 20 messages
	}
}

// Optimize filters and truncates messages to reduce token usage.
func (cm *ContextManager) Optimize(messages []llm.Message) []llm.Message {
	if len(messages) == 0 {
		return messages
	}

	// Step 1: Always keep system message
	var system llm.Message
	var rest []llm.Message
	for _, m := range messages {
		if m.Role == "system" {
			system = m
		} else {
			rest = append(rest, m)
		}
	}

	// Step 2: Truncate large tool results
	for i := range rest {
		if rest[i].Role == "tool" && len(rest[i].Content) > cm.maxToolResultLen {
			rest[i].Content = cm.truncateToolResult(rest[i].Content)
		}
	}

	// Step 3: Remove duplicate tool results (same tool_call_id, keep latest)
	rest = cm.deduplicateToolResults(rest)

	// Step 4: Drop empty assistant messages with no tool calls
	rest = cm.dropEmptyAssistant(rest)

	// Step 5: Trim to max messages (keep most recent)
	if len(rest) > cm.maxMessages {
		// Keep the first user message (original task) + recent messages
		rest = cm.trimHistory(rest)
	}

	// Rebuild with system message
	result := make([]llm.Message, 0, len(rest)+1)
	result = append(result, system)
	result = append(result, rest...)
	return result
}

// truncateToolResult shortens a tool result while preserving key info.
func (cm *ContextManager) truncateToolResult(content string) string {
	if len(content) <= cm.maxToolResultLen {
		return content
	}

	// Keep first 70% and last 20%, with summary in middle
	headLen := int(float64(cm.maxToolResultLen) * 0.7)
	tailLen := int(float64(cm.maxToolResultLen) * 0.2)

	head := content[:headLen]
	tail := content[len(content)-tailLen:]

	// Find a clean line break for head
	if idx := strings.LastIndex(head, "\n"); idx > headLen/2 {
		head = content[:idx]
	}
	// Find a clean line break for tail
	if idx := strings.Index(tail, "\n"); idx < tailLen/2 {
		tail = content[len(content)-tailLen+idx:]
	}

	omitted := len(content) - len(head) - len(tail)
	return head + "\n\n[... " + itoa(omitted) + " chars omitted ...]\n\n" + tail
}

// deduplicateToolResults keeps only the last result per tool_call_id.
func (cm *ContextManager) deduplicateToolResults(messages []llm.Message) []llm.Message {
	seen := make(map[string]int) // tool_call_id → index of last occurrence
	var toRemove []int

	for i, m := range messages {
		if m.Role == "tool" && m.ToolCallID != "" {
			if prev, exists := seen[m.ToolCallID]; exists {
				toRemove = append(toRemove, prev)
			}
			seen[m.ToolCallID] = i
		}
	}

	if len(toRemove) == 0 {
		return messages
	}

	removeSet := make(map[int]bool)
	for _, idx := range toRemove {
		removeSet[idx] = true
	}

	var result []llm.Message
	for i, m := range messages {
		if !removeSet[i] {
			result = append(result, m)
		}
	}
	return result
}

// dropEmptyAssistant removes assistant messages that have no content and no tool calls.
func (cm *ContextManager) dropEmptyAssistant(messages []llm.Message) []llm.Message {
	var result []llm.Message
	for _, m := range messages {
		if m.Role == "assistant" && m.Content == "" && len(m.ToolCalls) == 0 {
			continue
		}
		result = append(result, m)
	}
	return result
}

// trimHistory keeps the first user message + recent messages.
func (cm *ContextManager) trimHistory(messages []llm.Message) []llm.Message {
	// Find first user message (the original task)
	var firstUser *int
	for i, m := range messages {
		if m.Role == "user" {
			firstUser = &i
			break
		}
	}

	// Keep: first user + last (maxMessages-1) messages
	start := len(messages) - cm.maxMessages + 1
	if firstUser != nil && *firstUser < start {
		// Include first user message
		result := []llm.Message{messages[*firstUser]}
		result = append(result, messages[start:]...)
		return result
	}

	return messages[start:]
}

// itoa is a simple int-to-string helper (avoids strconv import for this small util).
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
