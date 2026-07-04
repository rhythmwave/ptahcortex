package context

import (
	"log"

	"github.com/rhythmwave/ptahcortex/internal/llm"
	"github.com/rhythmwave/ptahcortex/internal/mcp"
)

// Manager coordinates context assembly across the agent loop.
// It tracks summaries across iterations and delegates to the assembler per call type.
type Manager struct {
	assembler *Assembler
	sandbox   *Sandbox
	manager   *mcp.Manager

	// State across iterations
	summaries    []string          // all summaries from previous iterations
	currentIter  []*SandboxResult  // summaries from current iteration
	stats        *ContextStats
}

// NewManager creates a context manager with optional sandbox.
func NewManager(provider llm.Provider, manager *mcp.Manager) *Manager {
	assembler := NewAssembler()
	return &Manager{
		assembler: assembler,
		sandbox:   NewSandbox(provider, assembler, manager, 3),
		manager:   manager,
		stats:     &ContextStats{},
	}
}

// NewManagerWithoutSandbox creates a context manager that skips sandboxing.
// Useful for simple tasks or when sandbox LLM is not available.
func NewManagerWithoutSandbox(manager *mcp.Manager) *Manager {
	return &Manager{
		assembler: NewAssembler(),
		manager:   manager,
		stats:     &ContextStats{},
	}
}

// BuildPlanContext assembles messages for a plan call.
func (m *Manager) BuildPlanContext(systemPrompt string, task string, tools []llm.ToolDefinition) []llm.Message {
	msgs := m.assembler.AssemblePlan(systemPrompt, task, m.summaries, tools)
	log.Printf("[context] plan: %d messages, %d previous summaries", len(msgs), len(m.summaries))
	return msgs
}

// BuildReflectContext assembles messages for a reflect call.
func (m *Manager) BuildReflectContext(systemPrompt string, task string, tools []llm.ToolDefinition) []llm.Message {
	msgs := m.assembler.AssembleReflect(systemPrompt, task, m.currentIter, m.summaries)
	log.Printf("[context] reflect: %d messages, %d current summaries, %d previous summaries",
		len(msgs), len(m.currentIter), len(m.summaries))
	return msgs
}

// BuildFinalContext assembles messages for the final answer call.
func (m *Manager) BuildFinalContext(systemPrompt string, task string) []llm.Message {
	allSummaries := append([]string{}, m.summaries...)
	for _, sr := range m.currentIter {
		allSummaries = append(allSummaries, sr.Summary)
	}
	msgs := m.assembler.AssembleFinal(systemPrompt, task, allSummaries)
	log.Printf("[context] final: %d messages, %d total summaries", len(msgs), len(allSummaries))
	return msgs
}

// ExecuteSandboxed runs sub-tasks through the sandbox and collects summaries.
func (m *Manager) ExecuteSandboxed(subTasks []string, toolDefs []llm.ToolDefinition) {
	results := m.sandbox.ExecuteSubTasks(subTasks, toolDefs)
	m.currentIter = results

	for _, r := range results {
		m.stats.AddRecord(CallSandboxSelect, r.TokensUsed) // approximate
		m.stats.AddRecord(CallSandboxEval, 0)
	}
}

// CommitIteration finalizes the current iteration's summaries.
// Call this after reflect to persist summaries for the next iteration.
func (m *Manager) CommitIteration() {
	for _, sr := range m.currentIter {
		m.summaries = append(m.summaries, sr.Summary)
	}
	m.currentIter = nil
	log.Printf("[context] iteration committed: %d total summaries", len(m.summaries))
}

// RecordTokens records token usage for a call type.
func (m *Manager) RecordTokens(ct CallType, tokens int) {
	m.stats.AddRecord(ct, tokens)
}

// CurrentIter returns the current iteration's sandbox results.
func (m *Manager) CurrentIter() []*SandboxResult {
	return m.currentIter
}

// Stats returns the current token usage stats.
func (m *Manager) Stats() *ContextStats {
	return m.stats
}

// Summaries returns all accumulated summaries.
func (m *Manager) Summaries() []string {
	return m.summaries
}
