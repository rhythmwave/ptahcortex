package otel

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// MetricsCollector collects and persists metrics locally
type MetricsCollector struct {
	mu       sync.Mutex
	file     *os.File
	metrics  []MetricEntry
	enabled  bool
	logPath  string
}

// MetricEntry represents a single metric entry
type MetricEntry struct {
	Timestamp   time.Time      `json:"timestamp"`
	Type        string         `json:"type"` // "llm", "tool", "iteration", "agent"
	Agent       string         `json:"agent"`
	Model       string         `json:"model,omitempty"`
	Tokens      int            `json:"tokens,omitempty"`
	Duration    int64          `json:"duration_ms,omitempty"` // milliseconds
	Success     bool           `json:"success"`
	Error       string         `json:"error,omitempty"`
	Details     map[string]any `json:"details,omitempty"`
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(enabled bool, logPath string) *MetricsCollector {
	mc := &MetricsCollector{
		enabled: enabled,
		logPath: logPath,
	}
	
	if enabled && logPath != "" {
		// Create directory if needed
		dir := filepath.Dir(logPath)
		os.MkdirAll(dir, 0755)
		
		f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Printf("[metrics] failed to open log file: %v", err)
		} else {
			mc.file = f
		}
	}
	
	return mc
}

// RecordLLMCall records an LLM API call
func (mc *MetricsCollector) RecordLLMCall(provider, model string, duration time.Duration, tokens int, success bool, errStr string) {
	if !mc.enabled {
		return
	}
	
	entry := MetricEntry{
		Timestamp: time.Now(),
		Type:      "llm",
		Agent:     provider,
		Model:     model,
		Tokens:    tokens,
		Duration:  duration.Milliseconds(),
		Success:   success,
		Error:     errStr,
	}
	
	mc.write(entry)
}

// RecordToolCall records a tool call
func (mc *MetricsCollector) RecordToolCall(toolName string, duration time.Duration, success bool, errStr string) {
	if !mc.enabled {
		return
	}
	
	entry := MetricEntry{
		Timestamp: time.Now(),
		Type:      "tool",
		Agent:     toolName,
		Duration:  duration.Milliseconds(),
		Success:   success,
		Error:     errStr,
	}
	
	mc.write(entry)
}

// RecordAgentRun records a complete agent run
func (mc *MetricsCollector) RecordAgentRun(agentName string, task string, duration time.Duration, totalTokens int, iterations int, success bool, errStr string) {
	if !mc.enabled {
		return
	}
	
	entry := MetricEntry{
		Timestamp: time.Now(),
		Type:      "agent",
		Agent:     agentName,
		Tokens:    totalTokens,
		Duration:  duration.Milliseconds(),
		Success:   success,
		Error:     errStr,
		Details: map[string]any{
			"task":       task,
			"iterations": iterations,
		},
	}
	
	mc.write(entry)
}

// write persists a metric entry
func (mc *MetricsCollector) write(entry MetricEntry) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	mc.metrics = append(mc.metrics, entry)
	
	if mc.file != nil {
		data, err := json.Marshal(entry)
		if err != nil {
			return
		}
		mc.file.Write(append(data, '\n'))
		mc.file.Sync()
	}
}

// GetSummary returns a summary of collected metrics
func (mc *MetricsCollector) GetSummary() *MetricsSummary {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	summary := &MetricsSummary{
		TotalEntries: len(mc.metrics),
	}
	
	for _, entry := range mc.metrics {
		switch entry.Type {
		case "llm":
			summary.LLMCalls++
			summary.TotalTokens += entry.Tokens
			summary.TotalDuration += entry.Duration
			if !entry.Success {
				summary.LLMErrors++
			}
		case "tool":
			summary.ToolCalls++
			if !entry.Success {
				summary.ToolErrors++
			}
		case "agent":
			summary.AgentRuns++
			if !entry.Success {
				summary.AgentErrors++
			}
		}
	}
	
	return summary
}

// MetricsSummary represents aggregated metrics
type MetricsSummary struct {
	TotalEntries  int   `json:"total_entries"`
	LLMCalls      int   `json:"llm_calls"`
	LLMErrors     int   `json:"llm_errors"`
	ToolCalls     int   `json:"tool_calls"`
	ToolErrors    int   `json:"tool_errors"`
	AgentRuns     int   `json:"agent_runs"`
	AgentErrors   int   `json:"agent_errors"`
	TotalTokens   int   `json:"total_tokens"`
	TotalDuration int64 `json:"total_duration_ms"`
}

// FormatSummary returns a human-readable summary
func (s *MetricsSummary) FormatSummary() string {
	return fmt.Sprintf(`Metrics Summary:
  LLM Calls: %d (%d errors)
  Tool Calls: %d (%d errors)
  Agent Runs: %d (%d errors)
  Total Tokens: %d
  Total Duration: %dms`,
		s.LLMCalls, s.LLMErrors,
		s.ToolCalls, s.ToolErrors,
		s.AgentRuns, s.AgentErrors,
		s.TotalTokens,
		s.TotalDuration)
}

// Close closes the metrics collector
func (mc *MetricsCollector) Close() {
	if mc.file != nil {
		mc.file.Close()
	}
}
