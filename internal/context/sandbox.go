package context

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
)

// Sandbox executes tool calls in isolated, cheap LLM calls with minimal context.
// Only summaries flow back to the main agent loop.
type Sandbox struct {
	provider   SandboxLLMProvider
	assembler  *Assembler
	manager    ToolCaller
	maxIter    int // max tool iterations per sandbox call
	tracer     Tracer
}

// SandboxLLMProvider is the interface for sandbox LLM calls.
type SandboxLLMProvider interface {
	Chat(req SandboxChatRequest) (*SandboxChatResponse, error)
	Name() string
}

// SandboxChatRequest is a request to the LLM for sandbox operations.
type SandboxChatRequest struct {
	Messages  []Message
	Tools     []ToolDef
	MaxTokens int
	Model     string
}

// SandboxChatResponse is the LLM response for sandbox operations.
type SandboxChatResponse struct {
	Content   string
	ToolCalls []ToolCall
	Usage     TokenUsage
}

// TokenUsage tracks token consumption.
type TokenUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// NewSandbox creates a sandbox with the given LLM provider and tool caller.
func NewSandbox(provider SandboxLLMProvider, assembler *Assembler, manager ToolCaller, maxIter int) *Sandbox {
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

// SetTracer sets the tracer for sandbox operations.
func (s *Sandbox) SetTracer(tracer Tracer) {
	s.tracer = tracer
}

// ExecuteSubTask runs a sub-task in isolation: select tool → call → evaluate → summarize.
// Returns a SandboxResult with the summary.
func (s *Sandbox) ExecuteSubTask(subTask string, toolDefs []ToolDef) (*SandboxResult, error) {
	start := time.Now()
	log.Printf("[sandbox] executing sub-task: %s", truncate(subTask, 80))

	// Step 1: Select tool (minimal context)
	selectStart := time.Now()
	selectMsgs := s.assembler.AssembleSandboxSelect(subTask, toolDefs)

	// Convert to provider format
	providerMsgs := make([]Message, len(selectMsgs))
	copy(providerMsgs, selectMsgs)

	selectResp, err := s.provider.Chat(SandboxChatRequest{
		Messages:  providerMsgs,
		Tools:     toolDefs,
		MaxTokens: 500,
	})
	selectDuration := time.Since(selectStart)

	if err != nil {
		return nil, fmt.Errorf("sandbox select: %w", err)
	}

	attrs := map[string]any{
		"sub_task":        truncate(subTask, 100),
		"select_tokens":   selectResp.Usage.TotalTokens,
		"select_duration": selectDuration.String(),
		"tool_calls":      len(selectResp.ToolCalls),
	}
	if s.tracer != nil {
		span := s.tracer.Start("sandbox.select", attrs)
		span.End()
	}

	log.Printf("[sandbox] select: %d tokens, %d tool calls, took %v",
		selectResp.Usage.TotalTokens, len(selectResp.ToolCalls), selectDuration)

	if len(selectResp.ToolCalls) == 0 {
		// No tool call — LLM answered directly
		totalDuration := time.Since(start)
		result := &SandboxResult{
			SubTask:    subTask,
			Summary:    selectResp.Content,
			TokensUsed: selectResp.Usage.TotalTokens,
		}

		log.Printf("[sandbox] direct answer: %d tokens, took %v", result.TokensUsed, totalDuration)
		return result, nil
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

		// Execute via tool caller
		toolStart := time.Now()
		result, err := s.manager.CallTool(toolName, args)
		toolDuration := time.Since(toolStart)

		if err != nil {
			log.Printf("[sandbox] tool error: %v", err)
			allSummaries = append(allSummaries, fmt.Sprintf("%s: Error: %v", toolName, err))

			if s.tracer != nil {
				s.tracer.Start("sandbox.tool_error", map[string]any{
					"tool":     toolName,
					"error":    err.Error(),
					"duration": toolDuration.String(),
				}).End()
			}
			continue
		}

		rawResult := result.Content
		if result.IsError {
			rawResult = "Error: tool returned error"
		}

		if s.tracer != nil {
			s.tracer.Start("sandbox.tool_call", map[string]any{
				"tool":        toolName,
				"result_size": len(rawResult),
				"duration":    toolDuration.String(),
				"is_error":    result.IsError,
			}).End()
		}

		// Step 3: Evaluate result (minimal context)
		evalStart := time.Now()
		evalMsgs := s.assembler.AssembleSandboxEval(subTask, rawResult)
		providerEvalMsgs := make([]Message, len(evalMsgs))
		copy(providerEvalMsgs, evalMsgs)

		evalResp, err := s.provider.Chat(SandboxChatRequest{
			Messages: providerEvalMsgs,
			MaxTokens: 300,
		})
		evalDuration := time.Since(evalStart)

		if err != nil {
			log.Printf("[sandbox] eval error: %v, using raw", err)
			allSummaries = append(allSummaries, fmt.Sprintf("%s: %s", toolName, truncate(rawResult, 200)))
			continue
		}

		totalTokens += evalResp.Usage.TotalTokens

		if s.tracer != nil {
			s.tracer.Start("sandbox.eval", map[string]any{
				"tool":         toolName,
				"eval_tokens":  evalResp.Usage.TotalTokens,
				"eval_duration": evalDuration.String(),
			}).End()
		}

		summary := evalResp.Content
		if summary == "" {
			summary = truncate(rawResult, 200)
		}
		allSummaries = append(allSummaries, fmt.Sprintf("%s: %s", toolName, summary))
	}

	combinedSummary := strings.Join(allSummaries, "\n")
	totalDuration := time.Since(start)

	attrs = map[string]any{
		"sub_task":      truncate(subTask, 100),
		"total_tokens":  totalTokens,
		"summary_len":   len(combinedSummary),
		"tools_called":  len(selectResp.ToolCalls),
		"total_time":    totalDuration.String(),
	}
	if s.tracer != nil {
		span := s.tracer.Start("sandbox.complete", attrs)
		span.End()
	}

	log.Printf("[sandbox] sub-task complete: %d tokens, summary: %d chars, took %v",
		totalTokens, len(combinedSummary), totalDuration)

	return &SandboxResult{
		SubTask:    subTask,
		Summary:    combinedSummary,
		TokensUsed: totalTokens,
	}, nil
}

// ExecuteSubTasks runs multiple sub-tasks sequentially.
// Returns results in the same order as input.
func (s *Sandbox) ExecuteSubTasks(subTasks []string, toolDefs []ToolDef) []*SandboxResult {
	results := make([]*SandboxResult, len(subTasks))

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
