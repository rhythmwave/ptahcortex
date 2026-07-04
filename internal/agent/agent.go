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

// Agent runs the plan→execute→reflect loop with sandboxed tool reasoning.
type Agent struct {
	cfg       *config.Config
	llm       llm.Provider
	mcp       *mcp.Manager
	executor  *tools.Executor
	toolDefs  []llm.ToolDefinition
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

	a := &Agent{
		cfg:      cfg,
		llm:      provider,
		mcp:      mcpManager,
		executor: executor,
		tracer:   otel.NewTracer(true, cfg.Name),
		metrics:  otel.NewMetrics(true),
		ctxMgr:   ctx.NewManager(provider, mcpManager),
	}

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
	runSpan := a.tracer.Start(nil, "agent.run", map[string]any{
		"agent": a.cfg.Name,
		"task":  task,
	})
	defer runSpan.End()

	totalTokens := 0
	systemPrompt := a.systemPrompt()

	for iter := 0; iter < a.cfg.Agent.MaxIterations; iter++ {
		iterSpan := a.tracer.Start(nil, "agent.iteration", map[string]any{
			"iteration": iter + 1,
		})
		iterStart := time.Now()

		log.Printf("[agent] === iteration %d/%d ===", iter+1, a.cfg.Agent.MaxIterations)

		// ── PLAN ──
		// Build plan context: T0 + T1 + T3 (summaries from previous iterations)
		planMsgs := a.ctxMgr.BuildPlanContext(systemPrompt, task, a.toolDefs)
		planSpan := a.tracer.Start(nil, "agent.plan", nil)

		resp, err := a.llm.Chat(llm.ChatRequest{
			Messages:  planMsgs,
			Tools:     a.toolDefs,
			MaxTokens: a.cfg.LLM.MaxTokens,
			Model:     a.cfg.LLM.Model,
		})
		planSpan.End()

		if err != nil {
			iterSpan.End()
			return "", fmt.Errorf("llm error at iteration %d: %w", iter+1, err)
		}

		totalTokens += resp.Usage.TotalTokens
		a.ctxMgr.RecordTokens(ctx.CallPlan, resp.Usage.TotalTokens)
		a.metrics.RecordLLMCall(a.llm.Name(), a.cfg.LLM.Model, time.Since(iterStart), resp.Usage.TotalTokens)
		log.Printf("[agent] plan: %d tokens (total: %d)", resp.Usage.TotalTokens, totalTokens)

		if totalTokens >= a.cfg.Agent.MaxTokensPerRun {
			iterSpan.End()
			return "Token budget reached. Stopping.", nil
		}

		// No tool calls → agent is done, produce final answer
		if len(resp.ToolCalls) == 0 {
			log.Printf("[agent] no tool calls, producing final answer")
			iterSpan.End()
			a.metrics.RecordIteration(a.cfg.Name, iter+1, time.Since(iterStart), totalTokens)
			return resp.Content, nil
		}

		// ── EXECUTE (sandboxed) ──
		execSpan := a.tracer.Start(nil, "agent.execute", map[string]any{
			"tool_count": len(resp.ToolCalls),
		})

		// Build sub-tasks from tool calls
		subTasks := a.buildSubTasks(resp.ToolCalls)
		log.Printf("[agent] %d sub-tasks for sandbox", len(subTasks))

		// Execute through sandbox (isolated LLM calls, minimal context)
		a.ctxMgr.ExecuteSandboxed(subTasks, a.toolDefs)
		execSpan.End()

		// Log sandbox results
		for _, sr := range a.ctxMgr.CurrentIter() {
			log.Printf("[agent] sandbox: %s → %s", sr.ToolName, truncate(sr.Summary, 100))
		}

		// ── REFLECT ──
		reflectSpan := a.tracer.Start(nil, "agent.reflect", nil)
		reflectMsgs := a.ctxMgr.BuildReflectContext(systemPrompt, task, a.toolDefs)

		reflectResp, err := a.llm.Chat(llm.ChatRequest{
			Messages: reflectMsgs,
			MaxTokens: 500,
			Model:     a.cfg.LLM.Model,
		})
		reflectSpan.End()

		if err != nil {
			log.Printf("[agent] reflect error: %v", err)
		} else {
			totalTokens += reflectResp.Usage.TotalTokens
			a.ctxMgr.RecordTokens(ctx.CallReflect, reflectResp.Usage.TotalTokens)
			log.Printf("[agent] reflect: %d tokens — %s", reflectResp.Usage.TotalTokens, truncate(reflectResp.Content, 100))

			// If reflect says we're done, produce final answer
			if a.isDoneSignal(reflectResp.Content) {
				log.Printf("[agent] reflect signals done")
				a.ctxMgr.CommitIteration()
				iterSpan.End()
				a.metrics.RecordIteration(a.cfg.Name, iter+1, time.Since(iterStart), totalTokens)
				break
			}
		}

		// Commit iteration summaries for next plan call
		a.ctxMgr.CommitIteration()
		iterSpan.End()
		a.metrics.RecordIteration(a.cfg.Name, iter+1, time.Since(iterStart), totalTokens)
	}

	// ── FINAL ──
	log.Printf("[agent] producing final answer")
	finalMsgs := a.ctxMgr.BuildFinalContext(systemPrompt, task)

	finalResp, err := a.llm.Chat(llm.ChatRequest{
		Messages: finalMsgs,
		MaxTokens: a.cfg.LLM.MaxTokens,
		Model:     a.cfg.LLM.Model,
	})
	if err != nil {
		return "", fmt.Errorf("final llm error: %w", err)
	}

	totalTokens += finalResp.Usage.TotalTokens
	a.ctxMgr.RecordTokens(ctx.CallFinal, finalResp.Usage.TotalTokens)
	log.Printf("[agent] final: %d tokens, total run: %d tokens", finalResp.Usage.TotalTokens, totalTokens)
	log.Printf("[agent] token breakdown — plan: %d, sandbox: %d, reflect: %d, final: %d",
		a.ctxMgr.Stats().PlanTokens, a.ctxMgr.Stats().SandboxTokens,
		a.ctxMgr.Stats().ReflectTokens, a.ctxMgr.Stats().FinalTokens)

	return finalResp.Content, nil
}

// buildSubTasks converts tool calls into sub-task descriptions for the sandbox.
func (a *Agent) buildSubTasks(calls []llm.ToolCall) []string {
	var tasks []string
	for _, tc := range calls {
		// Parse arguments to build a readable sub-task
		var args map[string]any
		json.Unmarshal([]byte(tc.Function.Arguments), &args)

		subTask := fmt.Sprintf("Use %s", tc.Function.Name)
		if args != nil {
			// Extract key arguments for the sub-task description
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

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
