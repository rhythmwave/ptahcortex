package agent

import (
	"fmt"
	"log"
	"regexp"
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
	prompt := fmt.Sprintf(`I need to search a codebase for this task: %s

What searches should I run? List them as:
- search: <query>
- outline: <path>
- read: <path>
- audit

Include 5-10 relevant searches.`, task)

	start := time.Now()
	span := a.tracer.Start(nil, "agent.llm_plan", map[string]any{
		"task_length": len(task),
	})

	resp, err := a.llm.Chat(llm.ChatRequest{
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
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
	log.Printf("[smart-agent] LLM response:\n%s", resp.Content)

	// Parse markdown response into queries
	queries := a.parseMarkdownResponse(resp.Content)
	return queries, nil
}

// parseMarkdownResponse parses LLM markdown response into tool calls
func (a *SmartAgent) parseMarkdownResponse(response string) []string {
	var queries []string
	seen := make(map[string]bool) // deduplicate
	
	// Match tool calls in backticks or plain text
	// Formats:
	//   `search: <query>`
	//   search: <query>
	//   - search: <query>
	toolPattern := regexp.MustCompile(`(?i)(search|outline|read|audit|callers|trace_deps)\s*:\s*['"]?([^'"\n\]]+)['"]?`)
	
	matches := toolPattern.FindAllStringSubmatch(response, -1)
	
	for _, match := range matches {
		tool := strings.ToLower(match[1])
		args := strings.TrimSpace(match[2])
		
		// Skip explanation text
		if strings.Contains(args, "*") || 
		   strings.Contains(args, "**") ||
		   strings.Contains(args, "<") ||
		   len(args) < 2 {
			continue
		}
		
		var query string
		// Clean up args - remove extra text after pipe or backtick
		if idx := strings.IndexAny(args, "|`\""); idx > 0 {
			args = strings.TrimSpace(args[:idx])
		}
		// Remove trailing punctuation
		args = strings.TrimRight(args, ".),;")
		
		switch tool {
		case "search":
			query = "text_search:" + args
		case "outline":
			query = "outline:" + args
		case "read":
			query = "read:" + args
		case "audit":
			query = "audit:"
		case "callers":
			query = "callers:" + args
		case "trace_deps":
			query = "trace_deps:" + args
		}
		
		// Deduplicate
		if query != "" && !seen[query] {
			seen[query] = true
			queries = append(queries, query)
		}
	}
	
	return queries
}

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
