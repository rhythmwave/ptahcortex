package agent

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/rhythmwave/agentkit/internal/config"
	"github.com/rhythmwave/agentkit/internal/llm"
	"github.com/rhythmwave/agentkit/internal/mcp"
)

// Agent runs the plan→execute→reflect loop.
type Agent struct {
	cfg     *config.Config
	llm     llm.Provider
	mcp     *mcp.Manager
	tools   []llm.ToolDefinition
}

// New creates an agent from config with connected MCP servers and LLM provider.
func New(cfg *config.Config, provider llm.Provider, mcpManager *mcp.Manager) *Agent {
	a := &Agent{
		cfg: cfg,
		llm: provider,
		mcp: mcpManager,
	}

	// Convert MCP tools to LLM tool definitions
	for _, t := range mcpManager.AllTools() {
		a.tools = append(a.tools, llm.ToolDefinition{
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
			Tools:     a.tools,
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

		// If no tool calls, we're done — return the content
		if len(resp.ToolCalls) == 0 {
			return resp.Content, nil
		}

		// Add assistant message (with tool calls) to history
		messages = append(messages, llm.Message{
			Role:      "assistant",
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		})

		// EXECUTE: call each tool
		for _, tc := range resp.ToolCalls {
			log.Printf("[agent] tool call: %s", tc.Function.Name)

			var args map[string]any
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
				log.Printf("[agent] bad tool args: %v", err)
				args = map[string]any{}
			}

			result, err := a.mcp.CallTool(tc.Function.Name, args)
			if err != nil {
				log.Printf("[agent] tool error: %v", err)
				messages = append(messages, llm.Message{
					Role:       "tool",
					ToolCallID: tc.ID,
					Content:    fmt.Sprintf("Error: %v", err),
				})
				continue
			}

			log.Printf("[agent] tool result: %d bytes, error=%v", len(result.Content), result.IsError)
			messages = append(messages, llm.Message{
				Role:       "tool",
				ToolCallID: tc.ID,
				Content:    result.Content,
			})
		}

		// REFLECT: ask LLM to evaluate results and decide next step
		// The next iteration's LLM call serves as reflection — it sees all
		// tool results and decides whether to continue or respond.
	}

	return "Reached maximum iterations.", nil
}

func (a *Agent) systemPrompt() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("You are %s. %s\n\n", a.cfg.Name, a.cfg.Description))
	b.WriteString("You have access to these tools. Use them to accomplish the user's task.\n")
	b.WriteString("After each tool call, evaluate the results. If you have enough information, respond directly.\n")
	b.WriteString("If you need more information, call additional tools.\n")
	return b.String()
}

// ToolCount returns the number of available tools.
func (a *Agent) ToolCount() int {
	return len(a.tools)
}
