package agent

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/rhythmwave/ptahcortex/internal/config"
	"github.com/rhythmwave/ptahcortex/internal/llm"
	"github.com/rhythmwave/ptahcortex/internal/mcp"
	"github.com/rhythmwave/ptahcortex/internal/tools"
)

// Agent runs the plan→execute→reflect loop.
type Agent struct {
	cfg       *config.Config
	llm       llm.Provider
	mcp       *mcp.Manager
	executor  *tools.Executor
	toolDefs  []llm.ToolDefinition
}

// New creates an agent from config with connected MCP servers and LLM provider.
func New(cfg *config.Config, provider llm.Provider, mcpManager *mcp.Manager) *Agent {
	// Parse timeout
	timeout := 30 * time.Second
	if cfg.Tools.Timeout != "" {
		if d, err := time.ParseDuration(cfg.Tools.Timeout); err == nil {
			timeout = d
		}
	}

	executor := tools.NewExecutor(
		mcpManager,
		cfg.Tools.MaxParallel,
		timeout,
		tools.DefaultRetry(),
	)

	a := &Agent{
		cfg:      cfg,
		llm:      provider,
		mcp:      mcpManager,
		executor: executor,
	}

	// Convert MCP tools to LLM tool definitions
	for _, t := range mcpManager.AllTools() {
		a.toolDefs = append(a.toolDefs, llm.ToolDefinition{
			Type: "function",
			Function: llm.ToolFunction{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.InputSchema,
			},
		})
	}

	return a
}

// Run executes the agent loop on a task and returns the final response.
func (a *Agent) Run(task string) (string, error) {
	messages := []llm.Message{
		{Role: "system", Content: a.systemPrompt()},
		{Role: "user", Content: task},
	}

	totalTokens := 0

	for iter := 0; iter < a.cfg.Agent.MaxIterations; iter++ {
		log.Printf("[agent] iteration %d/%d", iter+1, a.cfg.Agent.MaxIterations)

		// PLAN: ask LLM what to do
		resp, err := a.llm.Chat(llm.ChatRequest{
			Messages:  messages,
			Tools:     a.toolDefs,
			MaxTokens: a.cfg.LLM.MaxTokens,
			Model:     a.cfg.LLM.Model,
		})
		if err != nil {
			return "", fmt.Errorf("llm error at iteration %d: %w", iter+1, err)
		}

		totalTokens += resp.Usage.TotalTokens
		log.Printf("[agent] tokens: %d (total: %d)", resp.Usage.TotalTokens, totalTokens)

		// Check budget
		if totalTokens >= a.cfg.Agent.MaxTokensPerRun {
			log.Printf("[agent] token budget exceeded (%d/%d)", totalTokens, a.cfg.Agent.MaxTokensPerRun)
			return "Token budget reached. Stopping.", nil
		}

		// If no tool calls, we're done
		if len(resp.ToolCalls) == 0 {
			return resp.Content, nil
		}

		// Add assistant message with tool calls to history
		messages = append(messages, llm.Message{
			Role:      "assistant",
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		})

		// EXECUTE: call tools (parallel for independent, sequential for dependent)
		mcpCalls := toolCallsToMCP(resp.ToolCalls)
		log.Printf("[agent] executing %d tool calls", len(mcpCalls))

		results := a.executor.ExecuteCalls(nil, mcpCalls)

		// Log results
		for _, r := range results {
			if r.IsError {
				log.Printf("[agent] tool %s: ERROR (%v) [%v]", r.Name, r.Error, r.Latency)
			} else {
				log.Printf("[agent] tool %s: OK %d bytes [%v]", r.Name, len(r.Content), r.Latency)
			}
		}

		// Feed results back as tool messages
		for _, r := range results {
			content := r.Content
			if r.IsError {
				if r.Error != nil {
					content = fmt.Sprintf("Error: %v", r.Error)
				} else {
					content = "Error: tool returned error"
				}
			}
			messages = append(messages, llm.Message{
				Role:       "tool",
				ToolCallID: r.CallID,
				Content:    content,
			})
		}

		// REFLECT happens naturally in the next iteration's LLM call
	}

	return "Reached maximum iterations.", nil
}

func (a *Agent) systemPrompt() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("You are %s. %s\n\n", a.cfg.Name, a.cfg.Description))
	b.WriteString("You have access to tools. Use them to accomplish the user's task.\n")
	b.WriteString("After each tool call, evaluate the results. If you have enough information, respond directly.\n")
	b.WriteString("If you need more information, call additional tools.\n")
	b.WriteString("Be concise. Focus on results.\n")
	return b.String()
}

// ToolCount returns the number of available tools.
func (a *Agent) ToolCount() int {
	return len(a.toolDefs)
}

// toolCallsToMCP converts LLM tool calls to MCP tool calls.
func toolCallsToMCP(calls []llm.ToolCall) []mcp.ToolCall {
	var result []mcp.ToolCall
	for _, tc := range calls {
		var args map[string]any
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			args = map[string]any{}
		}
		result = append(result, mcp.ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: args,
		})
	}
	return result
}
