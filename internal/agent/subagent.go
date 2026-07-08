package agent

import (
	"encoding/json"
	"fmt"
	"log"
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

// Run executes the agent with subagent support
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

	// Step 2: Execute subagents in parallel
	log.Printf("\n[smart-agent] Step 2: Executing subagents in parallel")
	subagentResults := a.executeSubagents(subagentTasks)
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

// executeSubagents executes subagents in parallel
func (a *SmartAgent) executeSubagents(tasks []Subagent) []Subagent {
	var wg sync.WaitGroup
	results := make([]Subagent, len(tasks))

	for i, task := range tasks {
		wg.Add(1)
		go func(idx int, t Subagent) {
			defer wg.Done()
			results[idx] = a.executeSubagent(t)
		}(i, task)
	}

	wg.Wait()
	return results
}

// executeSubagent executes a single subagent
func (a *SmartAgent) executeSubagent(task Subagent) Subagent {
	start := time.Now()
	
	log.Printf("[subagent-%s] Starting: %s", task.ID, task.Task)
	
	// Build tools for subagent (minimal context)
	toolsList := a.buildSubagentTools()
	
	prompt := fmt.Sprintf(`Complete this task: %s

Use the available tools to gather information.
Return the results as JSON:
{
  "findings": ["finding1", "finding2"],
  "files": ["file1.go", "file2.go"],
  "issues": [{"severity": "high", "description": "issue1"}]
}`, task.Task)

	resp, err := a.llm.Chat(llm.ChatRequest{
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
		Tools:     toolsList,
		MaxTokens: 1000,
		Model:     a.cfg.LLM.Model,
	})

	if err != nil {
		task.Error = err
		task.Duration = time.Since(start)
		log.Printf("[subagent-%s] Error: %v", task.ID, err)
		return task
	}

	// Execute tool calls
	results := make(map[string]string)
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

	task.Results = results
	task.Duration = time.Since(start)
	
	log.Printf("[subagent-%s] Completed in %v", task.ID, task.Duration)
	return task
}

// executeTool executes a single tool
func (a *SmartAgent) executeTool(name string, args map[string]any) (string, error) {
	switch name {
	case "read_file":
		path, _ := args["path"].(string)
		return a.basic.ReadFile(path)
	case "exec":
		command, _ := args["command"].(string)
		return a.basic.Exec(command)
	case "list_files":
		path, _ := args["path"].(string)
		return a.basic.ListFiles(path)
	case "search":
		if a.useLexa {
			query, _ := args["query"].(string)
			r, err := a.mcp.CallTool("text_search", map[string]any{"query": query})
			if err != nil {
				return "", err
			}
			return r.Content, nil
		}
		query, _ := args["query"].(string)
		return a.basic.Exec(fmt.Sprintf("grep -r %s .", query))
	case "outline":
		if a.useLexa {
			path, _ := args["path"].(string)
			r, err := a.mcp.CallTool("outline", map[string]any{"path": path})
			if err != nil {
				return "", err
			}
			return r.Content, nil
		}
		path, _ := args["path"].(string)
		return a.basic.Exec(fmt.Sprintf("head -50 %s", path))
	default:
		return "", fmt.Errorf("unknown tool: %s", name)
	}
}

// aggregateResults aggregates results from all subagents
func (a *SmartAgent) aggregateResults(subagents []Subagent) map[string]string {
	allResults := make(map[string]string)
	
	for _, sub := range subagents {
		if sub.Error != nil {
			allResults[fmt.Sprintf("subagent-%s-error", sub.ID)] = sub.Error.Error()
			continue
		}
		
		for tool, result := range sub.Results {
			key := fmt.Sprintf("subagent-%s-%s", sub.ID, tool)
			allResults[key] = result
		}
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

CODE RESULTS:
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

// buildSubagentTools builds tools for subagent (minimal context)
func (a *SmartAgent) buildSubagentTools() []llm.ToolDefinition {
	var tools []llm.ToolDefinition

	// Basic tools for subagent
	tools = append(tools,
		llm.ToolDefinition{
			Type: "function",
			Function: llm.ToolFunction{
				Name:        "read_file",
				Description: "Read file contents",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"path": map[string]any{
							"type":        "string",
							"description": "File path",
						},
					},
					"required": []string{"path"},
				},
			},
		},
		llm.ToolDefinition{
			Type: "function",
			Function: llm.ToolFunction{
				Name:        "exec",
				Description: "Execute a shell command",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"command": map[string]any{
							"type":        "string",
							"description": "Shell command to execute",
						},
					},
					"required": []string{"command"},
				},
			},
		},
		llm.ToolDefinition{
			Type: "function",
			Function: llm.ToolFunction{
				Name:        "list_files",
				Description: "List files in a directory",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"path": map[string]any{
							"type":        "string",
							"description": "Directory path",
						},
					},
				},
			},
		},
	)

	// Add Lexa tools if available
	if a.useLexa {
		tools = append(tools,
			llm.ToolDefinition{
				Type: "function",
				Function: llm.ToolFunction{
					Name:        "search",
					Description: "Search code patterns",
					Parameters: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"query": map[string]any{
								"type":        "string",
								"description": "Search query",
							},
						},
						"required": []string{"query"},
					},
				},
			},
			llm.ToolDefinition{
				Type: "function",
				Function: llm.ToolFunction{
					Name:        "outline",
					Description: "Get file structure",
					Parameters: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"path": map[string]any{
								"type":        "string",
								"description": "File path",
							},
						},
						"required": []string{"path"},
					},
				},
			},
		)
	}

	return tools
}

// truncate is defined in agent.go
