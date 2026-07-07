package agent

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/rhythmwave/ptahcortex/internal/config"
	"github.com/rhythmwave/ptahcortex/internal/dynamic"
	"github.com/rhythmwave/ptahcortex/internal/llm"
	"github.com/rhythmwave/ptahcortex/internal/mcp"
	"github.com/rhythmwave/ptahcortex/internal/otel"
)

// AutoAgent uses dynamic tool mapping for efficient execution
type AutoAgent struct {
	cfg      *config.Config
	llm      llm.Provider
	mcp      *mcp.Manager
	executor *dynamic.Executor
	detector *dynamic.Detector
	tracer   *otel.Tracer
	metrics  *otel.Metrics
}

// NewAutoAgent creates a new auto agent
func NewAutoAgent(cfg *config.Config, provider llm.Provider, mcpManager *mcp.Manager) (*AutoAgent, error) {
	// Load tool mappings config
	configPath := "configs/tool-mappings.yaml"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		configPath = "/opt/ptahcortex/configs/tool-mappings.yaml"
	}

	detector, err := dynamic.NewDetector(configPath)
	if err != nil {
		return nil, fmt.Errorf("load tool mappings: %w", err)
	}

	executor := dynamic.NewExecutor(mcpManager, detector)

	return &AutoAgent{
		cfg:      cfg,
		llm:      provider,
		mcp:      mcpManager,
		executor: executor,
		detector: detector,
		tracer:   otel.NewTracer(true, cfg.Name),
		metrics:  otel.NewMetrics(true),
	}, nil
}

// Run executes the agent with dynamic tool mapping
func (a *AutoAgent) Run(task string) (string, error) {
	start := time.Now()
	runSpan := a.tracer.Start(nil, "agent.run", map[string]any{
		"agent": a.cfg.Name,
		"task":  task,
	})
	defer runSpan.End()

	log.Printf("\n[auto-agent] ═══════════════════════════════════════")
	log.Printf("[auto-agent] ║ TASK: %s", truncate(task, 60))
	log.Printf("[auto-agent] ═══════════════════════════════════════")

	// Step 1: Execute tools automatically (no LLM)
	log.Printf("\n[auto-agent] Step 1: Executing tools automatically")
	execResult := a.executor.ExecuteTask(task)
	log.Printf("[auto-agent] %s", execResult.Summary())

	// Step 2: LLM analyzes results (1 call)
	log.Printf("\n[auto-agent] Step 2: LLM analyzing results")
	analysis, err := a.analyzeWithLLM(task, execResult.Aggregated)
	if err != nil {
		return "", fmt.Errorf("llm analysis: %w", err)
	}

	duration := time.Since(start)
	log.Printf("\n[auto-agent] ═══════════════════════════════════════")
	log.Printf("[auto-agent] ║ COMPLETE")
	log.Printf("[auto-agent] ║ duration: %v", duration)
	log.Printf("[auto-agent] ║ category: %s", execResult.Mapping.Category)
	log.Printf("[auto-agent] ║ tools: %d", len(execResult.Mapping.ToolCalls))
	log.Printf("[auto-agent] ═══════════════════════════════════════")

	return analysis, nil
}

// analyzeWithLLM sends aggregated results to LLM for analysis
func (a *AutoAgent) analyzeWithLLM(task, aggregated string) (string, error) {
	prompt := fmt.Sprintf(`You are a code analyst. Analyze the following search results and provide a comprehensive report.

TASK: %s

SEARCH RESULTS:
%s

Provide:
1. Summary of findings
2. Specific issues with file paths and line numbers
3. Severity ratings (High/Medium/Low)
4. Recommendations for fixes
5. Code patches in diff format (if applicable)

Be concise but thorough.`, task, aggregated)

	start := time.Now()
	span := a.tracer.Start(nil, "agent.llm_analyze", map[string]any{
		"task_length": len(task),
		"results_length": len(aggregated),
	})

	resp, err := a.llm.Chat(llm.ChatRequest{
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
		MaxTokens: a.cfg.LLM.MaxTokens,
		Model:     a.cfg.LLM.Model,
	})

	span.End()
	duration := time.Since(start)

	if err != nil {
		return "", err
	}

	totalTokens := resp.Usage.TotalTokens
	a.metrics.RecordLLMCall(a.llm.Name(), a.cfg.LLM.Model, duration, totalTokens)

	log.Printf("[auto-agent] LLM analysis: %d tokens, %v", totalTokens, duration)

	return resp.Content, nil
}
