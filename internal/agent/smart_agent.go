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
	"github.com/rhythmwave/ptahcortex/internal/otel"
	"github.com/rhythmwave/ptahcortex/internal/tools"
)

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

// Run executes the agent with LLM planning
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
	log.Printf("[smart-agent] ═══════════════════════════════════════")

	// Step 1: LLM Plans what to do (1 call)
	log.Printf("\n[smart-agent] Step 1: LLM planning")
	toolCalls, err := a.plan(task)
	if err != nil {
		return "", fmt.Errorf("plan: %w", err)
	}
	log.Printf("[smart-agent] Planned %d tool calls", len(toolCalls))

	// Step 2: Execute tools (0 tokens)
	log.Printf("\n[smart-agent] Step 2: Executing tools")
	results := a.executeTools(toolCalls)
	log.Printf("[smart-agent] Executed %d tools", len(results))

	// Step 3: LLM Analyzes results (1 call)
	log.Printf("\n[smart-agent] Step 3: LLM analyzing results")
	analysis, err := a.analyze(task, results)
	if err != nil {
		return "", fmt.Errorf("analyze: %w", err)
	}

	duration := time.Since(start)
	log.Printf("\n[smart-agent] ═══════════════════════════════════════")
	log.Printf("[smart-agent] ║ COMPLETE")
	log.Printf("[smart-agent] ║ duration: %v", duration)
	log.Printf("[smart-agent] ║ tool calls: %d", len(toolCalls))
	log.Printf("[smart-agent] ═══════════════════════════════════════")

	return analysis, nil
}

// plan uses LLM to plan what tools to use
func (a *SmartAgent) plan(task string) ([]ToolCall, error) {
	// Build available tools
	toolsList := a.buildToolDefinitions()
	
	prompt := fmt.Sprintf(`I need to complete this task: %s

Use the available tools to accomplish this task.
You can use multiple tools in sequence.`, task)

	start := time.Now()
	span := a.tracer.Start(nil, "agent.llm_plan", map[string]any{
		"task_length": len(task),
	})

	resp, err := a.llm.Chat(llm.ChatRequest{
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
		Tools:     toolsList,
		MaxTokens: 1000,
		Model:     a.cfg.LLM.Model,
	})

	span.End()
	duration := time.Since(start)

	if err != nil {
		return nil, err
	}

	totalTokens := resp.Usage.TotalTokens
	a.metrics.RecordLLMCall(a.llm.Name(), a.cfg.LLM.Model, duration, totalTokens)

	log.Printf("[smart-agent] LLM plan: %d tokens, %v", totalTokens, duration)

	// Extract tool calls from response
	var toolCalls []ToolCall
	for _, tc := range resp.ToolCalls {
		var args map[string]any
		json.Unmarshal([]byte(tc.Function.Arguments), &args)
		
		toolCalls = append(toolCalls, ToolCall{
			Tool: tc.Function.Name,
			Args: args,
		})
	}

	return toolCalls, nil
}

// executeTools executes all tool calls
func (a *SmartAgent) executeTools(toolCalls []ToolCall) map[string]string {
	results := make(map[string]string)

	for _, tc := range toolCalls {
		log.Printf("[smart-agent] executing: %s %v", tc.Tool, tc.Args)
		
		var result string
		var err error

		switch tc.Tool {
		// Basic OS tools
		case "read_file":
			path, _ := tc.Args["path"].(string)
			result, err = a.basic.ReadFile(path)
		case "write_file":
			path, _ := tc.Args["path"].(string)
			content, _ := tc.Args["content"].(string)
			err = a.basic.WriteFile(path, content)
			if err == nil {
				result = "File written successfully"
			}
		case "exec":
			command, _ := tc.Args["command"].(string)
			result, err = a.basic.Exec(command)
		case "list_files":
			path, _ := tc.Args["path"].(string)
			result, err = a.basic.ListFiles(path)
		
		// Lexa tools (optional)
		case "search":
			if a.useLexa {
				query, _ := tc.Args["query"].(string)
				r, e := a.mcp.CallTool("text_search", map[string]any{"query": query})
				if e != nil {
					err = e
				} else {
					result = r.Content
				}
			} else {
				// Fallback to basic search
				query, _ := tc.Args["query"].(string)
				result, err = a.basic.Exec(fmt.Sprintf("grep -r %s .", query))
			}
		case "outline":
			if a.useLexa {
				path, _ := tc.Args["path"].(string)
				r, e := a.mcp.CallTool("outline", map[string]any{"path": path})
				if e != nil {
					err = e
				} else {
					result = r.Content
				}
			} else {
				// Fallback to basic outline
				path, _ := tc.Args["path"].(string)
				result, err = a.basic.Exec(fmt.Sprintf("head -50 %s", path))
			}
		case "audit":
			if a.useLexa {
				r, e := a.mcp.CallTool("audit", map[string]any{})
				if e != nil {
					err = e
				} else {
					result = r.Content
				}
			} else {
				result = "Audit requires Lexa"
			}
		default:
			err = fmt.Errorf("unknown tool: %s", tc.Tool)
		}

		if err != nil {
			log.Printf("[smart-agent] error: %v", err)
			results[tc.Tool] = fmt.Sprintf("Error: %v", err)
		} else {
			results[tc.Tool] = result
		}
	}

	return results
}

// analyze uses LLM to analyze all results
func (a *SmartAgent) analyze(task string, results map[string]string) (string, error) {
	// Build context from all results
	var context strings.Builder
	for tool, result := range results {
		context.WriteString(fmt.Sprintf("=== %s ===\n%s\n\n", tool, result))
	}

	prompt := fmt.Sprintf(`You are a code analyst. Analyze the following results and provide a comprehensive report.

TASK: %s

RESULTS:
%s

Provide:
1. Summary of findings
2. Specific issues with file paths and line numbers
3. Severity ratings (High/Medium/Low)
4. Recommendations for fixes
5. Code patches in diff format (if applicable)

Be concise but thorough.`, task, context.String())

	start := time.Now()
	span := a.tracer.Start(nil, "agent.llm_analyze", map[string]any{
		"task_length":    len(task),
		"results_length": context.Len(),
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

	log.Printf("[smart-agent] LLM analyze: %d tokens, %v", totalTokens, duration)

	return resp.Content, nil
}

// buildToolDefinitions builds tool definitions based on configuration
func (a *SmartAgent) buildToolDefinitions() []llm.ToolDefinition {
	var tools []llm.ToolDefinition

	// Basic OS tools (always available)
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
				Name:        "write_file",
				Description: "Write to a file",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"path": map[string]any{
							"type":        "string",
							"description": "File path",
						},
						"content": map[string]any{
							"type":        "string",
							"description": "File content",
						},
					},
					"required": []string{"path", "content"},
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

	// Lexa tools (optional)
	if a.useLexa {
		tools = append(tools,
			llm.ToolDefinition{
				Type: "function",
				Function: llm.ToolFunction{
					Name:        "search",
					Description: "Search code patterns in the codebase",
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
			llm.ToolDefinition{
				Type: "function",
				Function: llm.ToolFunction{
					Name:        "audit",
					Description: "Run architecture audit",
					Parameters:  map[string]any{},
				},
			},
		)
	}

	return tools
}

// ToolCall represents a tool call from LLM
type ToolCall struct {
	Tool string
	Args map[string]any
}
