package agent

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/rhythmwave/ptahcortex/internal/config"
	ctx "github.com/rhythmwave/ptahcortex/internal/context"
	"github.com/rhythmwave/ptahcortex/internal/llm"
	"github.com/rhythmwave/ptahcortex/internal/mcp"
	"github.com/rhythmwave/ptahcortex/internal/otel"
	"github.com/rhythmwave/ptahcortex/internal/tools"
)

// llmAdapter wraps the llm.Provider to implement ctx.SandboxLLMProvider.
type llmAdapter struct {
	provider llm.Provider
}

func (a *llmAdapter) Chat(req ctx.SandboxChatRequest) (*ctx.SandboxChatResponse, error) {
	// Convert context.Message to llm.Message
	var messages []llm.Message
	for _, m := range req.Messages {
		messages = append(messages, llm.Message{
			Role:       m.Role,
			Content:    m.Content,
			ToolCallID: m.ToolCallID,
		})
	}

	// Convert context.ToolDef to llm.ToolDefinition
	var tools []llm.ToolDefinition
	for _, t := range req.Tools {
		tools = append(tools, llm.ToolDefinition{
			Type: t.Type,
			Function: llm.ToolFunction{
				Name:        t.Function.Name,
				Description: t.Function.Description,
				Parameters:  t.Function.Parameters,
			},
		})
	}

	resp, err := a.provider.Chat(llm.ChatRequest{
		Messages:  messages,
		Tools:     tools,
		MaxTokens: req.MaxTokens,
		Model:     req.Model,
	})
	if err != nil {
		return nil, err
	}

	// Convert response
	var toolCalls []ctx.ToolCall
	for _, tc := range resp.ToolCalls {
		toolCalls = append(toolCalls, ctx.ToolCall{
			ID:   tc.ID,
			Type: tc.Type,
			Function: struct {
				Name      string
				Arguments string
			}{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			},
		})
	}

	return &ctx.SandboxChatResponse{
		Content:   resp.Content,
		ToolCalls: toolCalls,
		Usage: ctx.TokenUsage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}, nil
}

func (a *llmAdapter) Name() string {
	return a.provider.Name()
}

// mcpAdapter wraps mcp.Manager to implement ctx.ToolCaller.
type mcpAdapter struct {
	manager *mcp.Manager
}

func (a *mcpAdapter) CallTool(name string, arguments map[string]any) (*ctx.ToolResult, error) {
	result, err := a.manager.CallTool(name, arguments)
	if err != nil {
		return nil, err
	}
	return &ctx.ToolResult{
		CallID:  result.CallID,
		Content: result.Content,
		IsError: result.IsError,
	}, nil
}

// Agent runs the plan→execute→reflect loop with sandboxed tool reasoning.
type Agent struct {
	cfg       *config.Config
	llm       llm.Provider
	mcp       *mcp.Manager
	executor  *tools.Executor
	toolDefs  []ctx.ToolDef
	tracer    *otel.Tracer
	metrics   *otel.Metrics
	ctxMgr    *ctx.Manager
}

// New creates an agent from config with connected MCP servers and LLM provider.
func New(cfg *config.Config, provider llm.Provider, mcpManager *mcp.Manager) *Agent {
	timeout := 30 * time.Second
	if cfg.Tools.Timeout != "" {
		if d, err := time.ParseDuration(cfg.Tools.Timeout); err == nil {
			timeout = d
		}
	}

	executor := tools.NewExecutor(mcpManager, cfg.Tools.MaxParallel, timeout, tools.DefaultRetry())
	tracer := otel.NewTracer(true, cfg.Name)

	// Create adapters for context manager
	llmAdapt := &llmAdapter{provider: provider}
	mcpAdapt := &mcpAdapter{manager: mcpManager}

	a := &Agent{
		cfg:      cfg,
		llm:      provider,
		mcp:      mcpManager,
		executor: executor,
		tracer:   tracer,
		metrics:  otel.NewMetrics(true),
		ctxMgr:   ctx.NewManager(llmAdapt, mcpAdapt),
	}

	// Wire tracer to context manager
	a.ctxMgr.SetTracer(ctx.NewLogTracer())

	// Convert MCP tools to context tool definitions
	for _, t := range mcpManager.AllTools() {
		a.toolDefs = append(a.toolDefs, ctx.ToolDef{
			Type: "function",
			Function: ctx.ToolFunction{
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
	runStart := time.Now()
	runSpan := a.tracer.Start(nil, "agent.run", map[string]any{
		"agent": a.cfg.Name,
		"task":  task,
	})
	defer runSpan.End()

	totalTokens := 0
	systemPrompt := a.systemPrompt()

	for iter := 0; iter < a.cfg.Agent.MaxIterations; iter++ {
		iterStart := time.Now()
		iterSpan := a.tracer.Start(nil, "agent.iteration", map[string]any{
			"iteration": iter + 1,
		})

		log.Printf("\n[agent] ═══════════════════════════════════════")
		log.Printf("[agent] ║ ITERATION %d/%d", iter+1, a.cfg.Agent.MaxIterations)
		log.Printf("[agent] ═══════════════════════════════════════")

		// ── PLAN ──
		planStart := time.Now()
		planMsgs := a.ctxMgr.BuildPlanContext(systemPrompt, task, a.toolDefs)

		// Convert context messages to LLM messages for the actual call
		var llmMessages []llm.Message
		for _, m := range planMsgs {
			llmMessages = append(llmMessages, llm.Message{
				Role:       m.Role,
				Content:    m.Content,
				ToolCallID: m.ToolCallID,
			})
		}

		planSpan := a.tracer.Start(nil, "agent.plan", map[string]any{
			"message_count": len(llmMessages),
		})

		resp, err := a.llm.Chat(llm.ChatRequest{
			Messages:  llmMessages,
			Tools:     a.toolDefsToLLM(a.toolDefs),
			MaxTokens: a.cfg.LLM.MaxTokens,
			Model:     a.cfg.LLM.Model,
		})
		planDuration := time.Since(planStart)
		planSpan.End()

		if err != nil {
			iterSpan.End()
			return "", fmt.Errorf("llm error at iteration %d: %w", iter+1, err)
		}

		totalTokens += resp.Usage.TotalTokens
		a.ctxMgr.RecordTokens(ctx.CallPlan, resp.Usage.TotalTokens)
		a.metrics.RecordLLMCall(a.llm.Name(), a.cfg.LLM.Model, planDuration, resp.Usage.TotalTokens)

		log.Printf("[agent] ┌─ PLAN ─────────────────────────────────")
		log.Printf("[agent] │ tokens: %d (total: %d)", resp.Usage.TotalTokens, totalTokens)
		log.Printf("[agent] │ tool_calls: %d", len(resp.ToolCalls))
		log.Printf("[agent] │ duration: %v", planDuration)
		if resp.Content != "" {
			log.Printf("[agent] │ response: %s", truncate(resp.Content, 150))
		}
		log.Printf("[agent] └─────────────────────────────────────────")

		if totalTokens >= a.cfg.Agent.MaxTokensPerRun {
			iterSpan.End()
			return "Token budget reached. Stopping.", nil
		}

		// No tool calls → agent is done
		if len(resp.ToolCalls) == 0 {
			log.Printf("[agent] no tool calls — ready for final answer")
			iterSpan.End()
			a.metrics.RecordIteration(a.cfg.Name, iter+1, time.Since(iterStart), totalTokens)
			break
		}

		// ── EXECUTE (sandboxed) ──
		execStart := time.Now()
		execSpan := a.tracer.Start(nil, "agent.execute", map[string]any{
			"tool_count": len(resp.ToolCalls),
		})

		subTasks := a.buildSubTasks(resp.ToolCalls)
		log.Printf("[agent] ┌─ EXECUTE (sandboxed) ──────────────────")
		log.Printf("[agent] │ sub-tasks: %d", len(subTasks))
		for i, st := range subTasks {
			log.Printf("[agent] │ [%d] %s", i+1, truncate(st, 80))
		}

		a.ctxMgr.ExecuteSandboxed(subTasks, a.toolDefs)
		execDuration := time.Since(execStart)
		execSpan.End()

		log.Printf("[agent] │ duration: %v", execDuration)
		log.Printf("[agent] └─────────────────────────────────────────")

		// Log sandbox results
		for i, sr := range a.ctxMgr.CurrentIter() {
			log.Printf("[agent]   sandbox[%d]: %s → %d tokens, %d chars",
				i, sr.ToolName, sr.TokensUsed, len(sr.Summary))
		}

		// ── REFLECT ──
		reflectStart := time.Now()
		reflectMsgs := a.ctxMgr.BuildReflectContext(systemPrompt, task, a.toolDefs)

		// Convert to LLM messages
		var llmReflectMsgs []llm.Message
		for _, m := range reflectMsgs {
			llmReflectMsgs = append(llmReflectMsgs, llm.Message{
				Role:       m.Role,
				Content:    m.Content,
				ToolCallID: m.ToolCallID,
			})
		}

		reflectSpan := a.tracer.Start(nil, "agent.reflect", map[string]any{
			"message_count": len(llmReflectMsgs),
		})

		reflectResp, err := a.llm.Chat(llm.ChatRequest{
			Messages: llmReflectMsgs,
			MaxTokens: 500,
			Model:     a.cfg.LLM.Model,
		})
		reflectDuration := time.Since(reflectStart)
		reflectSpan.End()

		if err != nil {
			log.Printf("[agent] ┌─ REFLECT ──────────────────────────────")
			log.Printf("[agent] │ error: %v", err)
			log.Printf("[agent] └─────────────────────────────────────────")
		} else {
			totalTokens += reflectResp.Usage.TotalTokens
			a.ctxMgr.RecordTokens(ctx.CallReflect, reflectResp.Usage.TotalTokens)

			log.Printf("[agent] ┌─ REFLECT ──────────────────────────────")
			log.Printf("[agent] │ tokens: %d (total: %d)", reflectResp.Usage.TotalTokens, totalTokens)
			log.Printf("[agent] │ duration: %v", reflectDuration)
			log.Printf("[agent] │ response: %s", truncate(reflectResp.Content, 150))
			log.Printf("[agent] └─────────────────────────────────────────")

			if a.isDoneSignal(reflectResp.Content) {
				log.Printf("[agent] ✓ reflect signals DONE")
				a.ctxMgr.CommitIteration()
				iterSpan.End()
				a.metrics.RecordIteration(a.cfg.Name, iter+1, time.Since(iterStart), totalTokens)
				break
			}
		}

		// Commit iteration summaries
		a.ctxMgr.CommitIteration()

		// Iteration summary
		iterDuration := time.Since(iterStart)
		stats := a.ctxMgr.Stats()
		log.Printf("[agent] ┌─ ITERATION %d SUMMARY ──────────────────", iter+1)
		log.Printf("[agent] │ duration: %v", iterDuration)
		log.Printf("[agent] │ tokens this iter: %d", resp.Usage.TotalTokens)
		log.Printf("[agent] │ total tokens: %d", totalTokens)
		log.Printf("[agent] │ summaries: %d", len(a.ctxMgr.Summaries()))
		log.Printf("[agent] │ breakdown — plan: %d, sandbox: %d, reflect: %d",
			stats.PlanTokens, stats.SandboxTokens, stats.ReflectTokens)
		log.Printf("[agent] └─────────────────────────────────────────")

		iterSpan.End()
		a.metrics.RecordIteration(a.cfg.Name, iter+1, iterDuration, totalTokens)
	}

	// ── FINAL ──
	finalStart := time.Now()
	log.Printf("\n[agent] ═══════════════════════════════════════")
	log.Printf("[agent] ║ FINAL ANSWER")
	log.Printf("[agent] ═══════════════════════════════════════")

	finalMsgs := a.ctxMgr.BuildFinalContext(systemPrompt, task)

	// Convert to LLM messages
	var llmFinalMsgs []llm.Message
	for _, m := range finalMsgs {
		llmFinalMsgs = append(llmFinalMsgs, llm.Message{
			Role:       m.Role,
			Content:    m.Content,
			ToolCallID: m.ToolCallID,
		})
	}

	finalSpan := a.tracer.Start(nil, "agent.final", map[string]any{
		"message_count": len(llmFinalMsgs),
	})

	finalResp, err := a.llm.Chat(llm.ChatRequest{
		Messages: llmFinalMsgs,
		MaxTokens: a.cfg.LLM.MaxTokens,
		Model:     a.cfg.LLM.Model,
	})
	finalDuration := time.Since(finalStart)
	finalSpan.End()

	if err != nil {
		return "", fmt.Errorf("final llm error: %w", err)
	}

	totalTokens += finalResp.Usage.TotalTokens
	a.ctxMgr.RecordTokens(ctx.CallFinal, finalResp.Usage.TotalTokens)

	stats := a.ctxMgr.Stats()
	log.Printf("[agent] ┌─ FINAL ─────────────────────────────────")
	log.Printf("[agent] │ tokens: %d", finalResp.Usage.TotalTokens)
	log.Printf("[agent] │ duration: %v", finalDuration)
	log.Printf("[agent] │ response length: %d chars", len(finalResp.Content))
	log.Printf("[agent] └─────────────────────────────────────────")
	log.Printf("[agent] ═══════════════════════════════════════════")
	log.Printf("[agent] ║ RUN COMPLETE")
	log.Printf("[agent] ║ total tokens: %d", totalTokens)
	log.Printf("[agent] ║ total time: %v", time.Since(runStart))
	log.Printf("[agent] ║ breakdown:")
	log.Printf("[agent] ║   plan:    %d tokens", stats.PlanTokens)
	log.Printf("[agent] ║   sandbox: %d tokens (%d calls)", stats.SandboxTokens, stats.SandboxCallCount)
	log.Printf("[agent] ║   reflect: %d tokens", stats.ReflectTokens)
	log.Printf("[agent] ║   final:   %d tokens", stats.FinalTokens)
	log.Printf("[agent] ═══════════════════════════════════════════")

	return finalResp.Content, nil
}

// toolDefsToLLM converts context ToolDefs to LLM ToolDefinitions.
func (a *Agent) toolDefsToLLM(defs []ctx.ToolDef) []llm.ToolDefinition {
	var result []llm.ToolDefinition
	for _, d := range defs {
		result = append(result, llm.ToolDefinition{
			Type: d.Type,
			Function: llm.ToolFunction{
				Name:        d.Function.Name,
				Description: d.Function.Description,
				Parameters:  d.Function.Parameters,
			},
		})
	}
	return result
}

// buildSubTasks converts tool calls into sub-task descriptions for the sandbox.
func (a *Agent) buildSubTasks(calls []llm.ToolCall) []string {
	var tasks []string
	for _, tc := range calls {
		var args map[string]any
		json.Unmarshal([]byte(tc.Function.Arguments), &args)

		subTask := fmt.Sprintf("Use %s", tc.Function.Name)
		if args != nil {
			if q, ok := args["query"].(string); ok {
				subTask += fmt.Sprintf(" with query %q", q)
			}
			if p, ok := args["path"].(string); ok {
				subTask += fmt.Sprintf(" on path %s", p)
			}
			if c, ok := args["command"].(string); ok {
				subTask += fmt.Sprintf(" running %s", truncate(c, 50))
			}
		}
		tasks = append(tasks, subTask)
	}
	return tasks
}

// isDoneSignal checks if the reflect response indicates the agent is done.
func (a *Agent) isDoneSignal(content string) bool {
	lower := strings.ToLower(content)
	signals := []string{
		"review is complete",
		"analysis is complete",
		"i have enough information",
		"ready to provide",
		"final assessment",
		"conclusion:",
		"here is my",
		"done",
		"complete",
	}
	for _, s := range signals {
		if strings.Contains(lower, s) {
			return true
		}
	}
	return false
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

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
