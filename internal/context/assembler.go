package context

import (
	"fmt"
	"strings"

	"github.com/rhythmwave/ptahcortex/internal/llm"
)

// Assembler builds message slices for different call types.
// It applies the context tier rules: each call type gets only the tiers it needs.
type Assembler struct {
	maxToolResultLen int // max chars per tool result in sandbox
	maxSummaryLen    int // max chars per summary
}

// NewAssembler creates an assembler with sensible defaults.
func NewAssembler() *Assembler {
	return &Assembler{
		maxToolResultLen: 4000,
		maxSummaryLen:    500,
	}
}

// AssemblePlan builds messages for a plan call.
// Context: T0 (system + tools) + T1 (task) + T3 (previous summaries)
func (a *Assembler) AssemblePlan(systemPrompt string, task string, summaries []string, tools []llm.ToolDefinition) []llm.Message {
	var msgs []llm.Message

	// T0: system prompt with tool list
	toolList := a.formatToolList(tools)
	msgs = append(msgs, llm.Message{
		Role:    "system",
		Content: systemPrompt + "\n\nAvailable tools:\n" + toolList,
	})

	// T1: task
	msgs = append(msgs, llm.Message{
		Role:    "user",
		Content: task,
	})

	// T3: previous summaries (if any)
	if len(summaries) > 0 {
		summaryBlock := a.buildSummaryBlock(summaries)
		msgs = append(msgs, llm.Message{
			Role:    "assistant",
			Content: "Previous findings:\n" + summaryBlock,
		})
	}

	return msgs
}

// AssembleSandboxSelect builds messages for a sandbox tool-selection call.
// Context: T0 (minimal tool defs) + sub-task only
func (a *Assembler) AssembleSandboxSelect(subTask string, tools []llm.ToolDefinition) []llm.Message {
	toolList := a.formatToolList(tools)

	return []llm.Message{
		{
			Role:    "system",
			Content: "You are a tool selector. Choose the best tool for the given sub-task.\n\nAvailable tools:\n" + toolList,
		},
		{
			Role:    "user",
			Content: subTask,
		},
	}
}

// AssembleSandboxEval builds messages for a sandbox evaluation call.
// Context: sub-task + truncated tool result
func (a *Assembler) AssembleSandboxEval(subTask string, toolResult string) []llm.Message {
	truncated := toolResult
	if len(truncated) > a.maxToolResultLen {
		truncated = truncated[:a.maxToolResultLen] + "\n[... truncated ...]"
	}

	return []llm.Message{
		{
			Role:    "system",
			Content: "Summarize the tool result for the main agent. Be concise. Extract key facts, patterns, and findings.",
		},
		{
			Role:    "user",
			Content: fmt.Sprintf("Sub-task: %s\n\nTool result:\n%s", subTask, truncated),
		},
	}
}

// AssembleReflect builds messages for a reflect call.
// Context: T0 + T1 + T2 (sandbox summaries from this iteration) + T3 (previous summaries)
func (a *Assembler) AssembleReflect(systemPrompt string, task string, iterationSummaries []*SandboxResult, previousSummaries []string) []llm.Message {
	var msgs []llm.Message

	// T0: system
	msgs = append(msgs, llm.Message{
		Role:    "system",
		Content: systemPrompt,
	})

	// T1: task
	msgs = append(msgs, llm.Message{
		Role:    "user",
		Content: task,
	})

	// T2: current iteration sandbox summaries
	if len(iterationSummaries) > 0 {
		var blocks []string
		for _, sr := range iterationSummaries {
			blocks = append(blocks, fmt.Sprintf("[%s via %s]: %s", sr.SubTask, sr.ToolName, sr.Summary))
		}
		msgs = append(msgs, llm.Message{
			Role:    "assistant",
			Content: "Current iteration findings:\n" + strings.Join(blocks, "\n\n"),
		})
	}

	// T3: previous summaries
	if len(previousSummaries) > 0 {
		summaryBlock := a.buildSummaryBlock(previousSummaries)
		msgs = append(msgs, llm.Message{
			Role:    "assistant",
			Content: "Previous findings:\n" + summaryBlock,
		})
	}

	return msgs
}

// AssembleFinal builds messages for the final answer call.
// Context: T0 + T1 + all summaries
func (a *Assembler) AssembleFinal(systemPrompt string, task string, allSummaries []string) []llm.Message {
	var msgs []llm.Message

	// T0: system
	msgs = append(msgs, llm.Message{
		Role:    "system",
		Content: systemPrompt,
	})

	// T1: task
	msgs = append(msgs, llm.Message{
		Role:    "user",
		Content: task,
	})

	// All summaries
	if len(allSummaries) > 0 {
		summaryBlock := a.buildSummaryBlock(allSummaries)
		msgs = append(msgs, llm.Message{
			Role:    "assistant",
			Content: "All findings from tool analysis:\n" + summaryBlock,
		})
	}

	return msgs
}

// formatToolList creates a compact tool list for the system prompt.
func (a *Assembler) formatToolList(tools []llm.ToolDefinition) string {
	var b strings.Builder
	for _, t := range tools {
		b.WriteString(fmt.Sprintf("- %s: %s\n", t.Function.Name, t.Function.Description))
	}
	return b.String()
}

// buildSummaryBlock joins summaries with numbering.
func (a *Assembler) buildSummaryBlock(summaries []string) string {
	var b strings.Builder
	for i, s := range summaries {
		b.WriteString(fmt.Sprintf("%d. %s\n", i+1, s))
	}
	return b.String()
}
