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
)

// SmartAgent uses LLM planning for intelligent tool selection
type SmartAgent struct {
	cfg     *config.Config
	llm     llm.Provider
	mcp     *mcp.Manager
	tracer  *otel.Tracer
	metrics *otel.Metrics
}

// NewSmartAgent creates a new smart agent
func NewSmartAgent(cfg *config.Config, provider llm.Provider, mcpManager *mcp.Manager) *SmartAgent {
	return &SmartAgent{
		cfg:     cfg,
		llm:     provider,
		mcp:     mcpManager,
		tracer:  otel.NewTracer(true, cfg.Name),
		metrics: otel.NewMetrics(true),
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
	log.Printf("[smart-agent] ═══════════════════════════════════════")

	// Step 1: LLM Plans what to search (1 call)
	log.Printf("\n[smart-agent] Step 1: LLM planning searches")
	queries, err := a.planSearches(task)
	if err != nil {
		return "", fmt.Errorf("plan searches: %w", err)
	}
	log.Printf("[smart-agent] Planned %d searches", len(queries))

	// Step 2: Execute ALL searches (Lexa, 0 tokens)
	log.Printf("\n[smart-agent] Step 2: Executing searches")
	results := a.executeSearches(queries)
	log.Printf("[smart-agent] Executed %d searches", len(results))

	// Step 3: LLM Analyzes results (1 call)
	log.Printf("\n[smart-agent] Step 3: LLM analyzing results")
	analysis, err := a.analyzeResults(task, results)
	if err != nil {
		return "", fmt.Errorf("analyze results: %w", err)
	}

	duration := time.Since(start)
	log.Printf("\n[smart-agent] ═══════════════════════════════════════")
	log.Printf("[smart-agent] ║ COMPLETE")
	log.Printf("[smart-agent] ║ duration: %v", duration)
	log.Printf("[smart-agent] ║ searches: %d", len(queries))
	log.Printf("[smart-agent] ═══════════════════════════════════════")

	return analysis, nil
}

// planSearches uses LLM to plan what to search
func (a *SmartAgent) planSearches(task string) ([]string, error) {
	// Define tools for the LLM to use
	tools := []llm.ToolDefinition{
		{
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
		{
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
		{
			Type: "function",
			Function: llm.ToolFunction{
				Name:        "read",
				Description: "Read file content",
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
		{
			Type: "function",
			Function: llm.ToolFunction{
				Name:        "audit",
				Description: "Run architecture audit",
				Parameters:  map[string]any{},
			},
		},
	}

	prompt := fmt.Sprintf("I need to search a codebase for this task: %s\n\nWhat searches should I run? Use the provided tools.", task)

	start := time.Now()
	span := a.tracer.Start(nil, "agent.llm_plan", map[string]any{
		"task_length": len(task),
	})

	resp, err := a.llm.Chat(llm.ChatRequest{
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
		Tools:     tools,
		MaxTokens: 500,
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
	var queries []string
	for _, tc := range resp.ToolCalls {
		var args map[string]any
		json.Unmarshal([]byte(tc.Function.Arguments), &args)
		
		switch tc.Function.Name {
		case "search":
			if query, ok := args["query"].(string); ok {
				queries = append(queries, "text_search:"+query)
			}
		case "outline":
			if path, ok := args["path"].(string); ok {
				queries = append(queries, "outline:"+path)
			}
		case "read":
			if path, ok := args["path"].(string); ok {
				queries = append(queries, "read:"+path)
			}
		case "audit":
			queries = append(queries, "audit:")
		}
	}
	
	return queries, nil
}

// parseMarkdownResponse parses LLM markdown response into tool calls
// Tool calling is now handled natively by the LLM API

// executeSearches executes all searches via Lexa
func (a *SmartAgent) executeSearches(queries []string) map[string]string {
	results := make(map[string]string)

	for _, query := range queries {
		parts := strings.SplitN(query, ":", 2)
		if len(parts) != 2 {
			continue
		}

		tool := parts[0]
		args := parts[1]

		log.Printf("[smart-agent] executing: %s %s", tool, args)

		var result string
		var err error

		switch tool {
		case "text_search":
			r, e := a.mcp.CallTool("text_search", map[string]any{
				"query": args,
			})
			if e != nil {
				err = e
			} else {
				result = r.Content
			}
		case "outline":
			r, e := a.mcp.CallTool("outline", map[string]any{
				"path": args,
			})
			if e != nil {
				err = e
			} else {
				result = r.Content
			}
		case "read":
			r, e := a.mcp.CallTool("read", map[string]any{
				"path": args,
			})
			if e != nil {
				err = e
			} else {
				result = r.Content
			}
		case "callers":
			r, e := a.mcp.CallTool("callers", map[string]any{
				"name": args,
			})
			if e != nil {
				err = e
			} else {
				result = r.Content
			}
		case "trace_deps":
			r, e := a.mcp.CallTool("trace_deps", map[string]any{
				"path": args,
			})
			if e != nil {
				err = e
			} else {
				result = r.Content
			}
		case "audit":
			r, e := a.mcp.CallTool("audit", map[string]any{})
			if e != nil {
				err = e
			} else {
				result = r.Content
			}
		default:
			// Try generic call
			r, e := a.mcp.CallTool(tool, map[string]any{
				"query": args,
			})
			if e != nil {
				err = e
			} else {
				result = r.Content
			}
		}

		if err != nil {
			log.Printf("[smart-agent] error: %v", err)
			results[query] = fmt.Sprintf("Error: %v", err)
		} else {
			results[query] = result
		}
	}

	return results
}

// analyzeResults uses LLM to analyze all search results
func (a *SmartAgent) analyzeResults(task string, results map[string]string) (string, error) {
	// Build context from all results
	var context strings.Builder
	for query, result := range results {
		context.WriteString(fmt.Sprintf("=== %s ===\n%s\n\n", query, result))
	}

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

Be concise but thorough.`, task, context.String())

	start := time.Now()
	span := a.tracer.Start(nil, "agent.llm_analyze", map[string]any{
		"task_length":    len(task),
		"results_length": context.Len(),
		"results_count":  len(results),
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
