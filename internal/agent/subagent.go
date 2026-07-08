package agent

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/rhythmwave/ptahcortex/internal/config"
	"github.com/rhythmwave/ptahcortex/internal/llm"
	"github.com/rhythmwave/ptahcortex/internal/mcp"
	"github.com/rhythmwave/ptahcortex/internal/otel"
	"github.com/rhythmwave/ptahcortex/internal/tools"
)

// Subagent represents a subagent for parallel execution
type Subagent struct {
	ID       string
	Task     string
	Results  map[string]string
	Error    error
	Duration time.Duration
	Output   string
}

// SmartAgent uses LLM planning for intelligent tool selection
type SmartAgent struct {
	cfg      *config.Config
	llm      llm.Provider
	mcp      *mcp.Manager
	basic    *tools.BasicTool
	tracer   *otel.Tracer
	metrics  *otel.Metrics
	useLexa  bool
}

// NewSmartAgent creates a new smart agent
func NewSmartAgent(cfg *config.Config, provider llm.Provider, mcpManager *mcp.Manager, useLexa bool) *SmartAgent {
	return &SmartAgent{
		cfg:     cfg,
		llm:     provider,
		mcp:     mcpManager,
		basic:   tools.NewBasicTool(""),
		tracer:  otel.NewTracer(true, cfg.Name),
		metrics: otel.NewMetrics(true),
		useLexa: useLexa,
	}
}

// Run executes the agent with parallel subagent spawning
func (a *SmartAgent) Run(task string) (string, error) {
	start := time.Now()
	runSpan := a.tracer.Start(nil, "agent.run", map[string]any{
		"agent": a.cfg.Name,
		"task":  task,
	})
	defer runSpan.End()

	log.Printf("\n[smart-agent] ═══════════════════════════════════════")
	log.Printf("[smart-agent] ║ TASK: %s", truncate(task, 60))
	log.Printf("[smart-agent] ║ LEXA: %v", a.useLexa)
	log.Printf("[smart-agent] ║ MAX ITERATIONS: %d", a.cfg.Agent.MaxIterations)
	log.Printf("[smart-agent] ═══════════════════════════════════════")

	// Step 1: Plan subagent tasks
	log.Printf("\n[smart-agent] Step 1: Planning subagent tasks")
	subagentTasks, err := a.planSubagents(task)
	if err != nil {
		return "", fmt.Errorf("plan subagents: %w", err)
	}
	log.Printf("[smart-agent] Planned %d subagent tasks", len(subagentTasks))

	// Step 2: Spawn parallel subagent processes
	log.Printf("\n[smart-agent] Step 2: Spawning parallel subagent processes")
	subagentResults := a.spawnParallelSubagents(subagentTasks)
	log.Printf("[smart-agent] Completed %d subagents", len(subagentResults))

	// Step 3: Aggregate results
	log.Printf("\n[smart-agent] Step 3: Aggregating results")
	allResults := a.aggregateResults(subagentResults)

	// Step 4: Main agent analysis
	log.Printf("\n[smart-agent] Step 4: Main agent analysis")
	analysis, err := a.analyze(task, allResults)
	if err != nil {
		return "", fmt.Errorf("analyze: %w", err)
	}

	duration := time.Since(start)
	log.Printf("\n[smart-agent] ═══════════════════════════════════════")
	log.Printf("[smart-agent] ║ COMPLETE")
	log.Printf("[smart-agent] ║ duration: %v", duration)
	log.Printf("[smart-agent] ║ subagents: %d", len(subagentTasks))
	log.Printf("[smart-agent] ═══════════════════════════════════════")

	return analysis, nil
}

// planSubagents uses LLM to plan subagent tasks
func (a *SmartAgent) planSubagents(task string) ([]Subagent, error) {
	prompt := fmt.Sprintf(`I need to complete this task: %s

Break this into 2-4 parallel subagent tasks.
Each subagent should handle a specific aspect of the task.

Return JSON array of subagent tasks:
[
  {"id": "1", "task": "Search for OAuth2 related files"},
  {"id": "2", "task": "Analyze token storage security"},
  {"id": "3", "task": "Check CSRF protection"}
]

Return ONLY the JSON array, no other text.`, task)

	resp, err := a.llm.Chat(llm.ChatRequest{
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
		MaxTokens: 500,
		Model:     a.cfg.LLM.Model,
	})

	if err != nil {
		return nil, err
	}

	// Parse JSON response
	var subagentTasks []Subagent
	if err := json.Unmarshal([]byte(resp.Content), &subagentTasks); err != nil {
		// Fallback to single task
		return []Subagent{
			{ID: "1", Task: task},
		}, nil
	}

	return subagentTasks, nil
}

// spawnParallelSubagents spawns independent processes for each subagent
func (a *SmartAgent) spawnParallelSubagents(tasks []Subagent) []Subagent {
	var wg sync.WaitGroup
	results := make([]Subagent, len(tasks))

	for i, task := range tasks {
		wg.Add(1)
		go func(idx int, t Subagent) {
			defer wg.Done()
			results[idx] = a.spawnSubagentProcess(t)
		}(i, task)
	}

	wg.Wait()
	return results
}

// spawnSubagentProcess spawns an independent process for a subagent
func (a *SmartAgent) spawnSubagentProcess(task Subagent) Subagent {
	start := time.Now()

	log.Printf("[subagent-%s] Spawning process: %s", task.ID, task.Task)

	// Create temporary config for subagent
	configPath := fmt.Sprintf("/tmp/ptahcortex-subagent-%s.yaml", task.ID)
	if err := a.createSubagentConfig(configPath, task); err != nil {
		task.Error = err
		task.Duration = time.Since(start)
		log.Printf("[subagent-%s] Error creating config: %v", task.ID, err)
		return task
	}
	defer os.Remove(configPath)

	// Spawn independent process
	cmd := exec.Command("/usr/local/bin/ptahcortex",
		"--config", configPath,
		"--smart",
		"--task", task.Task,
	)

	// Capture output
	output, err := cmd.CombinedOutput()
	if err != nil {
		task.Error = err
		task.Output = string(output)
		task.Duration = time.Since(start)
		log.Printf("[subagent-%s] Error: %v", task.ID, err)
		return task
	}

	task.Output = string(output)
	task.Duration = time.Since(start)

	log.Printf("[subagent-%s] Completed in %v", task.ID, task.Duration)
	return task
}

// createSubagentConfig creates a temporary config for a subagent
func (a *SmartAgent) createSubagentConfig(configPath string, task Subagent) error {
	// Escape task description for YAML
	escapedTask := strings.ReplaceAll(task.Task, "'", "''")
	escapedTask = strings.ReplaceAll(escapedTask, "\n", " ")
	
	config := fmt.Sprintf(`name: subagent-%s
description: "Subagent for: %s"

llm:
  provider: openai
  model: mimo-v2.5
  base_url: https://token-plan-sgp.xiaomimimo.com
  api_key: tp-s7emr8e5k8enrm5jnz2gs82d3hvhoq22vvt0sw110sipuhfm
  max_tokens: 4096

tools:
  max_parallel: 3
  timeout: 30s

agent:
  max_iterations: 3
  max_tokens_per_run: 20000
`, task.ID, escapedTask)

	return os.WriteFile(configPath, []byte(config), 0644)
}

// aggregateResults aggregates results from all subagents
func (a *SmartAgent) aggregateResults(subagents []Subagent) map[string]string {
	allResults := make(map[string]string)

	for _, sub := range subagents {
		if sub.Error != nil {
			allResults[fmt.Sprintf("subagent-%s-error", sub.ID)] = sub.Error.Error()
			continue
		}

		// Parse output for findings
		allResults[fmt.Sprintf("subagent-%s-output", sub.ID)] = sub.Output
	}

	return allResults
}

// analyze uses LLM to analyze all results
func (a *SmartAgent) analyze(task string, results map[string]string) (string, error) {
	// Build context from all results
	var context strings.Builder
	for key, result := range results {
		// Truncate long results
		truncated := result
		if len(truncated) > 500 {
			truncated = truncated[:500] + "..."
		}
		context.WriteString(fmt.Sprintf("=== %s ===\n%s\n\n", key, truncated))
	}

	prompt := fmt.Sprintf(`You are a senior security auditor. Analyze the following code for security vulnerabilities.

TASK: %s

SUBAGENT RESULTS:
%s

CRITICAL SECURITY CHECKS:
1. JWT/Token Security:
   - Token in URL query parameters (Critical)
   - Weak/default JWT secrets (Critical)
   - No algorithm restriction (Critical)
   - Token storage in localStorage (High)
   - No token revocation (High)

2. OAuth2 Security:
   - CSRF via missing state validation (High)
   - Open redirect via unvalidated redirect_uri (High)
   - Insecure token storage (High)

3. CORS/Headers:
   - CORS reflects any origin with credentials (Critical)
   - Missing security headers (Medium)

4. Session Management:
   - Session fixation (High)
   - Insecure session storage (High)

For EACH issue found:
- File path and exact line numbers
- Severity rating (Critical/High/Medium/Low)
- Attack vector explanation
- Code patch to fix

Be thorough. Find ALL critical vulnerabilities.`, task, context.String())

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

// truncate is defined in agent.go
