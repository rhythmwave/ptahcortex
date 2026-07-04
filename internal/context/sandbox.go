package context

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/rhythmwave/ptahcortex/internal/llm"
	"github.com/rhythmwave/ptahcortex/internal/mcp"
)

// Sandbox executes tool calls in isolated, cheap LLM calls with minimal context.
// Only summaries flow back to the main agent loop.
type Sandbox struct {
	provider   llm.Provider
	assembler  *Assembler
	manager    *mcp.Manager
	maxIter    int // max tool iterations per sandbox call
}

// NewSandbox creates a sandbox with the given LLM provider and MCP manager.
func NewSandbox(provider llm.Provider, assembler *Assembler, manager *mcp.Manager, maxIter int) *Sandbox {
	if maxIter <= 0 {
		maxIter = 3
	}
	return &Sandbox{
		provider:  provider,
		assembler: assembler,
		manager:   manager,
		maxIter:   maxIter,
	}
}

// ExecuteSubTask runs a sub-task in isolation: select tool → call → evaluate → summarize.
// Returns a SandboxResult with the summary.
func (s *Sandbox) ExecuteSubTask(subTask string, toolDefs []llm.ToolDefinition) (*SandboxResult, error) {
	log.Printf("[sandbox] executing sub-task: %s", truncate(subTask, 80))

	// Step 1: Select tool (minimal context)
	selectMsgs := s.assembler.AssembleSandboxSelect(subTask, toolDefs)
	selectResp, err := s.provider.Chat(llm.ChatRequest{
		Messages:  selectMsgs,
		Tools:     toolDefs,
		MaxTokens: 500,
	})
	if err != nil {
		return nil, fmt.Errorf("sandbox select: %w", err)
	}

	if len(selectResp.ToolCalls) == 0 {
		// No tool call — LLM answered directly
		return &SandboxResult{
			SubTask:    subTask,
			Summary:    selectResp.Content,
			TokensUsed: selectResp.Usage.TotalTokens,
		}, nil
	}

	// Step 2: Execute the selected tool(s)
	var allSummaries []string
	totalTokens := selectResp.Usage.TotalTokens

	for i, tc := range selectResp.ToolCalls {
		if i >= s.maxIter {
			break
		}

		toolName := tc.Function.Name
		log.Printf("[sandbox] calling tool: %s", toolName)

		var args map[string]any
		json.Unmarshal([]byte(tc.Function.Arguments), &args)

		// Execute via MCP manager directly
		result, err := s.manager.CallTool(toolName, args)
		if err != nil {
			log.Printf("[sandbox] tool error: %v", err)
			allSummaries = append(allSummaries, fmt.Sprintf("%s: Error: %v", toolName, err))
			continue
		}

		rawResult := result.Content
		if result.IsError {
			rawResult = "Error: tool returned error"
		}

		// Step 3: Evaluate result (minimal context)
		evalMsgs := s.assembler.AssembleSandboxEval(subTask, rawResult)
		evalResp, err := s.provider.Chat(llm.ChatRequest{
			Messages: evalMsgs,
			MaxTokens: 300,
		})
		if err != nil {
			log.Printf("[sandbox] eval error: %v, using raw", err)
			allSummaries = append(allSummaries, fmt.Sprintf("%s: %s", toolName, truncate(rawResult, 200)))
			continue
		}

		totalTokens += evalResp.Usage.TotalTokens
		summary := evalResp.Content
		if summary == "" {
			summary = truncate(rawResult, 200)
		}
		allSummaries = append(allSummaries, fmt.Sprintf("%s: %s", toolName, summary))
	}

	combinedSummary := strings.Join(allSummaries, "\n")
	log.Printf("[sandbox] sub-task complete: %d tokens, summary: %d chars", totalTokens, len(combinedSummary))

	return &SandboxResult{
		SubTask:    subTask,
		Summary:    combinedSummary,
		TokensUsed: totalTokens,
	}, nil
}

// ExecuteSubTasks runs multiple sub-tasks in parallel.
// Returns results in the same order as input.
func (s *Sandbox) ExecuteSubTasks(subTasks []string, toolDefs []llm.ToolDefinition) []*SandboxResult {
	results := make([]*SandboxResult, len(subTasks))

	// Run sequentially for now (parallel requires goroutines + sync)
	for i, st := range subTasks {
		result, err := s.ExecuteSubTask(st, toolDefs)
		if err != nil {
			log.Printf("[sandbox] sub-task %d failed: %v", i, err)
			results[i] = &SandboxResult{
				SubTask: st,
				Summary: fmt.Sprintf("Error: %v", err),
			}
		} else {
			results[i] = result
		}
	}

	return results
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
