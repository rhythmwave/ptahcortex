package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rhythmwave/ptahcortex/internal/config"
	"github.com/rhythmwave/ptahcortex/internal/llm"
	"github.com/rhythmwave/ptahcortex/internal/mcp"
	"github.com/rhythmwave/ptahcortex/internal/otel"
	"github.com/rhythmwave/ptahcortex/internal/tools"
)

const (
	maxSubagents      = 3
	subagentTimeout   = 60 * time.Second
	maxSubagentTokens = 4096
)

// Subagent represents a subagent task
type Subagent struct {
	ID       string
	Task     string
	Output   string
	Error    error
	Duration time.Duration
}

// SmartAgent uses LLM planning with subagent support
type SmartAgent struct {
	cfg       *config.Config
	llm       llm.Provider
	mcp       *mcp.Manager
	basic     *tools.BasicTool
	tracer    *otel.Tracer
	metrics   *otel.Metrics
	collector *otel.MetricsCollector
	useLexa   bool
	isSub     bool // true if this is a subagent (no recursive spawning)
}

// NewSmartAgent creates a new smart agent
func NewSmartAgent(cfg *config.Config, provider llm.Provider, mcpManager *mcp.Manager, useLexa bool) *SmartAgent {
	return &SmartAgent{
		cfg:       cfg,
		llm:       provider,
		mcp:       mcpManager,
		basic:     tools.NewBasicTool(""),
		tracer:    otel.NewTracer(true, cfg.Name),
		metrics:   otel.NewMetrics(true),
		collector: otel.NewMetricsCollector(true, filepath.Join(os.TempDir(), "ptahcortex-metrics.jsonl")),
		useLexa:   useLexa,
	}
}

// NewSubAgent creates a subagent that won't spawn nested subagents
func NewSubAgent(cfg *config.Config, provider llm.Provider, mcpManager *mcp.Manager, useLexa bool) *SmartAgent {
	a := NewSmartAgent(cfg, provider, mcpManager, useLexa)
	a.isSub = true
	return a
}

// Run executes the agent
func (a *SmartAgent) Run(task string) (string, error) {
	start := time.Now()

	log.Printf("\n[agent] ═══════════════════════════════════════")
	log.Printf("[agent] ║ TASK: %s", truncate(task, 60))
	log.Printf("[agent] ║ SUBAGENT: %v", a.isSub)
	log.Printf("[agent] ═══════════════════════════════════════")

	var allResults map[string]string
	var err error

	if a.isSub {
		// Subagent: just execute tools directly, no sub-spawning
		allResults = a.executeDirectly(task)
	} else if isSimpleTask(task) {
		// Simple task: execute directly without subagents
		log.Printf("[agent] Simple task detected, skipping subagents")
		allResults = a.executeDirectly(task)
	} else {
		// Complex task: plan subagents, spawn in parallel, analyze
		subagentTasks, err := a.planSubagents(task)
		if err != nil {
			return "", fmt.Errorf("plan subagents: %w", err)
		}

		subagentResults := a.spawnParallel(subagentTasks)
		allResults = a.aggregate(subagentResults)
	}

	// Final analysis
	analysis, err := a.analyze(task, allResults)
	if err != nil {
		return "", fmt.Errorf("analyze: %w", err)
	}

	duration := time.Since(start)
	
	// Record metrics
	a.collector.RecordAgentRun(
		"smart",
		task,
		duration,
		len(analysis),
		1,
		err == nil,
		func() string { if err != nil { return err.Error() }; return "" }(),
	)
	
	log.Printf("\n[agent] ═══════════════════════════════════════")
	log.Printf("[agent] ║ COMPLETE in %v", duration)
	log.Printf("[agent] ═══════════════════════════════════════")

	return analysis, nil
}

// isSimpleTask detects tasks that don't need subagents
func isSimpleTask(task string) bool {
	simple := []string{
		"list", "find", "show", "read", "get",
		"count", "check", "verify", "display",
	}
	taskLower := strings.ToLower(task)
	for _, kw := range simple {
		if strings.HasPrefix(taskLower, kw) {
			return true
		}
	}
	return len(strings.Fields(task)) <= 5
}

// executeDirectly runs tools without sub-spawning (for subagents)
func (a *SmartAgent) executeDirectly(task string) map[string]string {
	toolsList := a.buildTools()
	results := make(map[string]string)
	cm := NewContextManager(maxSubagentTokens)
	
	// Iterative execution with growing context
	for iteration := 0; iteration < 3; iteration++ {
		prompt := cm.GetProgressivePrompt(task, iteration, results)
		
		resp, err := a.llm.Chat(llm.ChatRequest{
			Messages: []llm.Message{
				{Role: "user", Content: prompt},
			},
			Tools:     toolsList,
			MaxTokens: maxSubagentTokens / 3,
			Model:     a.cfg.LLM.Model,
		})
		if err != nil {
			results["error"] = err.Error()
			break
		}

		// Execute tool calls
		for _, tc := range resp.ToolCalls {
			var args map[string]any
			json.Unmarshal([]byte(tc.Function.Arguments), &args)
			result, err := a.executeTool(tc.Function.Name, args)
			if err != nil {
				results[tc.Function.Name] = fmt.Sprintf("Error: %v", err)
			} else {
				results[tc.Function.Name] = result
			}
		}

		// If no tool calls, we're done
		if len(resp.ToolCalls) == 0 {
			results["analysis"] = resp.Content
			break
		}
	}

	return results
}

// planSubagents uses LLM to break task into parallel subtasks
func (a *SmartAgent) planSubagents(task string) ([]Subagent, error) {
	prompt := fmt.Sprintf(`Break this task into 2-3 parallel subtasks.
Return ONLY a JSON array, no explanations.

Task: %s

Format:
[{"id":"1","task":"subtask description"},{"id":"2","task":"subtask description"}]`, task)

	resp, err := a.llm.Chat(llm.ChatRequest{
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
		MaxTokens: 300,
		Model:     a.cfg.LLM.Model,
	})
	if err != nil {
		// Fallback to single task
		return []Subagent{{ID: "1", Task: task}}, nil
	}

	var tasks []Subagent
	if err := json.Unmarshal([]byte(resp.Content), &tasks); err != nil {
		return []Subagent{{ID: "1", Task: task}}, nil
	}

	// Limit to maxSubagents
	if len(tasks) > maxSubagents {
		tasks = tasks[:maxSubagents]
	}

	return tasks, nil
}

// spawnParallel spawns subagent processes in parallel with limits
func (a *SmartAgent) spawnParallel(tasks []Subagent) []Subagent {
	var wg sync.WaitGroup
	results := make([]Subagent, len(tasks))

	// Semaphore to limit concurrency
	sem := make(chan struct{}, maxSubagents)

	for i, task := range tasks {
		wg.Add(1)
		go func(idx int, t Subagent) {
			defer wg.Done()
			sem <- struct{}{}        // acquire
			defer func() { <-sem }() // release
			results[idx] = a.spawnSubagent(t)
		}(i, task)
	}

	wg.Wait()
	return results
}

// spawnSubagent spawns a single subagent process
func (a *SmartAgent) spawnSubagent(task Subagent) Subagent {
	start := time.Now()
	log.Printf("[subagent-%s] Starting: %s", task.ID, truncate(task.Task, 50))

	// Create temp config with --subagent flag (no nested spawning)
	configPath := filepath.Join(os.TempDir(), fmt.Sprintf("ptahcortex-sub-%s.yaml", task.ID))
	if err := a.writeConfig(configPath, task); err != nil {
		task.Error = err
		task.Duration = time.Since(start)
		return task
	}
	defer os.Remove(configPath)

	// Spawn with timeout and --subagent flag
	ctx, cancel := context.WithTimeout(context.Background(), subagentTimeout)
	defer cancel()

	execPath, _ := os.Executable()
	cmd := exec.CommandContext(ctx, execPath,
		"--config", configPath,
		"--subagent", // key: prevents recursive spawning
		"--task", task.Task,
	)

	output, err := cmd.CombinedOutput()
	task.Output = string(output)
	task.Duration = time.Since(start)

	if err != nil {
		task.Error = err
		log.Printf("[subagent-%s] Failed: %v (%v)", task.ID, err, task.Duration)
	} else {
		log.Printf("[subagent-%s] Done in %v", task.ID, task.Duration)
	}

	return task
}

// writeConfig creates a temp config for a subagent
func (a *SmartAgent) writeConfig(path string, task Subagent) error {
	escaped := strings.ReplaceAll(task.Task, `"`, `\"`)
	escaped = strings.ReplaceAll(escaped, "\n", " ")

	cfg := fmt.Sprintf(`name: subagent-%s
description: "Subagent: %s"
llm:
  provider: openai
  model: mimo-v2.5
  base_url: https://token-plan-sgp.xiaomimimo.com
  api_key: tp-s7emr8e5k8enrm5jnz2gs82d3hvhoq22vvt0sw110sipuhfm
  max_tokens: %d
tools:
  max_parallel: 2
  timeout: 20s
agent:
  max_iterations: 2
  max_tokens_per_run: 10000
`, task.ID, escaped, maxSubagentTokens)

	return os.WriteFile(path, []byte(cfg), 0644)
}

// aggregate combines results from all subagents
func (a *SmartAgent) aggregate(subagents []Subagent) map[string]string {
	results := make(map[string]string)
	for _, sub := range subagents {
		key := fmt.Sprintf("subagent-%s", sub.ID)
		if sub.Error != nil {
			results[key+"-error"] = sub.Error.Error()
		} else {
			results[key] = sub.Output
		}
	}
	return results
}

// analyze uses LLM to produce final analysis
func (a *SmartAgent) analyze(task string, results map[string]string) (string, error) {
	var ctx strings.Builder
	for k, v := range results {
		if len(v) > 800 {
			v = v[:800] + "..."
		}
		ctx.WriteString(fmt.Sprintf("=== %s ===\n%s\n\n", k, v))
	}

	prompt := fmt.Sprintf(`You are a senior security auditor. Analyze these results and provide a comprehensive report.

TASK: %s

RESULTS:
%s

For each finding include:
- File path and line numbers
- Severity (Critical/High/Medium/Low)
- Attack vector
- Code patch to fix

Be thorough and find ALL critical vulnerabilities.`, task, ctx.String())

	resp, err := a.llm.Chat(llm.ChatRequest{
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
		MaxTokens: a.cfg.LLM.MaxTokens,
		Model:     a.cfg.LLM.Model,
	})
	if err != nil {
		return "", err
	}

	return resp.Content, nil
}

// buildTools returns available tools
func (a *SmartAgent) buildTools() []llm.ToolDefinition {
	tools := []llm.ToolDefinition{
		{Type: "function", Function: llm.ToolFunction{
			Name: "read_file", Description: "Read file contents",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{"path": map[string]any{"type": "string", "description": "File path"}},
				"required": []string{"path"},
			},
		}},
		{Type: "function", Function: llm.ToolFunction{
			Name: "exec", Description: "Execute shell command",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{"command": map[string]any{"type": "string", "description": "Command"}},
				"required": []string{"command"},
			},
		}},
		{Type: "function", Function: llm.ToolFunction{
			Name: "list_files", Description: "List files in directory",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{"path": map[string]any{"type": "string", "description": "Directory path"}},
			},
		}},
	}

	if a.useLexa {
		tools = append(tools,
			llm.ToolDefinition{Type: "function", Function: llm.ToolFunction{
				Name: "search", Description: "Search code patterns (Lexa)",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{"query": map[string]any{"type": "string"}},
					"required": []string{"query"},
				},
			}},
			llm.ToolDefinition{Type: "function", Function: llm.ToolFunction{
				Name: "outline", Description: "Get file structure (Lexa)",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{"path": map[string]any{"type": "string"}},
					"required": []string{"path"},
				},
			}},
		)
	}

	return tools
}

// executeTool runs a single tool
func (a *SmartAgent) executeTool(name string, args map[string]any) (string, error) {
	switch name {
	case "read_file":
		p, _ := args["path"].(string)
		return a.basic.ReadFile(p)
	case "exec":
		c, _ := args["command"].(string)
		return a.basic.Exec(c)
	case "list_files":
		p, _ := args["path"].(string)
		return a.basic.ListFiles(p)
	case "search":
		if a.useLexa {
			q, _ := args["query"].(string)
			r, err := a.mcp.CallTool("text_search", map[string]any{"query": q})
			if err != nil {
				return "", err
			}
			return r.Content, nil
		}
		q, _ := args["query"].(string)
		return a.basic.Exec(fmt.Sprintf("grep -r '%s' . 2>/dev/null | head -20", q))
	case "outline":
		if a.useLexa {
			p, _ := args["path"].(string)
			r, err := a.mcp.CallTool("outline", map[string]any{"path": p})
			if err != nil {
				return "", err
			}
			return r.Content, nil
		}
		return "", fmt.Errorf("outline requires Lexa")
	default:
		return "", fmt.Errorf("unknown tool: %s", name)
	}
}

// truncate is in agent.go
