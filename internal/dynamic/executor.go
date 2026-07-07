package dynamic

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/rhythmwave/ptahcortex/internal/mcp"
)

// Executor executes tool calls automatically
type Executor struct {
	mcp      *mcp.Manager
	detector *Detector
}

// NewExecutor creates a new executor
func NewExecutor(mcpManager *mcp.Manager, detector *Detector) *Executor {
	return &Executor{
		mcp:      mcpManager,
		detector: detector,
	}
}

// ExecuteTask executes all tools for a task automatically
func (e *Executor) ExecuteTask(task string) *ExecutionResult {
	start := time.Now()

	// 1. Detect tools
	mapping := e.detector.Detect(task)
	log.Printf("[auto] detected category: %s (source: %s)", mapping.Category, mapping.Source)

	// 2. Execute all tool calls
	results := make(map[string]string)
	for _, call := range mapping.ToolCalls {
		log.Printf("[auto] executing: %s %v", call.Tool, call.Args)
		result := e.mcp.Call(call.Tool, call.Args)
		results[call.Tool+":"+fmt.Sprintf("%v", call.Args)] = result
	}

	// 3. Aggregate results
	aggregated := e.aggregateResults(results)

	duration := time.Since(start)
	log.Printf("[auto] executed %d tools in %v", len(mapping.ToolCalls), duration)

	return &ExecutionResult{
		Mapping:   mapping,
		Results:   results,
		Aggregated: aggregated,
		Duration:  duration,
	}
}

// aggregateResults combines all tool results into a single string
func (e *Executor) aggregateResults(results map[string]string) string {
	var b strings.Builder

	for key, result := range results {
		parts := strings.SplitN(key, ":", 2)
		tool := parts[0]
		b.WriteString(fmt.Sprintf("=== %s ===\n%s\n\n", tool, result))
	}

	return b.String()
}

// ExecutionResult represents the result of executing a task
type ExecutionResult struct {
	Mapping    *TaskMapping
	Results    map[string]string
	Aggregated string
	Duration   time.Duration
}

// Summary returns a summary of the execution
func (r *ExecutionResult) Summary() string {
	return fmt.Sprintf("Category: %s | Source: %s | Tools: %d | Duration: %v",
		r.Mapping.Category,
		r.Mapping.Source,
		len(r.Mapping.ToolCalls),
		r.Duration,
	)
}
