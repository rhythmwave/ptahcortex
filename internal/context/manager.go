package context

import (
	"log"
	"time"
)

// Tracer interface for context operations.
type Tracer interface {
	Start(name string, attrs map[string]any) *Span
}

// Span represents a traced operation.
type Span struct {
	Name      string
	StartTime time.Time
	EndTime   time.Time
	Attrs     map[string]any
}

// End finishes the span and logs it.
func (s *Span) End() {
	s.EndTime = time.Now()
	duration := s.EndTime.Sub(s.StartTime)
	log.Printf("[context-trace] %s [%v] %v", s.Name, duration, s.Attrs)
}

// LogTracer is a simple log-based tracer that implements the Tracer interface.
type LogTracer struct{}

// NewLogTracer creates a new log-based tracer.
func NewLogTracer() *LogTracer {
	return &LogTracer{}
}

// Start begins a new span.
func (t *LogTracer) Start(name string, attrs map[string]any) *Span {
	return &Span{
		Name:      name,
		StartTime: time.Now(),
		Attrs:     attrs,
	}
}

// Manager coordinates context assembly across the agent loop.
// It tracks summaries across iterations and delegates to the assembler per call type.
type Manager struct {
	assembler *Assembler
	sandbox   *Sandbox
	tracer    Tracer

	// State across iterations
	summaries    []string          // all summaries from previous iterations
	currentIter  []*SandboxResult  // summaries from current iteration
	stats        *ContextStats
}

// NewManager creates a context manager with optional sandbox.
func NewManager(provider SandboxLLMProvider, manager ToolCaller) *Manager {
	assembler := NewAssembler()
	return &Manager{
		assembler: assembler,
		sandbox:   NewSandbox(provider, assembler, manager, 3),
		stats:     &ContextStats{},
	}
}

// NewManagerWithoutSandbox creates a context manager that skips sandboxing.
func NewManagerWithoutSandbox(manager ToolCaller) *Manager {
	return &Manager{
		assembler: NewAssembler(),
		stats:     &ContextStats{},
	}
}

// SetTracer sets the tracer for context operations.
func (m *Manager) SetTracer(tracer Tracer) {
	m.tracer = tracer
}

// BuildPlanContext assembles messages for a plan call.
func (m *Manager) BuildPlanContext(systemPrompt string, task string, tools []ToolDef) []Message {
	start := time.Now()

	msgs := m.assembler.AssemblePlan(systemPrompt, task, m.summaries, tools)

	duration := time.Since(start)
	attrs := map[string]any{
		"call_type":       "plan",
		"message_count":   len(msgs),
		"summary_count":   len(m.summaries),
		"tool_count":      len(tools),
		"assembly_time_us": duration.Microseconds(),
	}

	if m.tracer != nil {
		span := m.tracer.Start("context.build_plan", attrs)
		span.End()
	}

	log.Printf("[context] plan: %d messages, %d previous summaries, %d tools, assembled in %v",
		len(msgs), len(m.summaries), len(tools), duration)
	return msgs
}

// BuildReflectContext assembles messages for a reflect call.
func (m *Manager) BuildReflectContext(systemPrompt string, task string, tools []ToolDef) []Message {
	start := time.Now()

	msgs := m.assembler.AssembleReflect(systemPrompt, task, m.currentIter, m.summaries)

	duration := time.Since(start)
	attrs := map[string]any{
		"call_type":          "reflect",
		"message_count":      len(msgs),
		"current_summaries":  len(m.currentIter),
		"prev_summaries":     len(m.summaries),
		"assembly_time_us":   duration.Microseconds(),
	}

	if m.tracer != nil {
		span := m.tracer.Start("context.build_reflect", attrs)
		span.End()
	}

	log.Printf("[context] reflect: %d messages, %d current summaries, %d previous summaries, assembled in %v",
		len(msgs), len(m.currentIter), len(m.summaries), duration)
	return msgs
}

// BuildFinalContext assembles messages for the final answer call.
func (m *Manager) BuildFinalContext(systemPrompt string, task string) []Message {
	start := time.Now()

	allSummaries := append([]string{}, m.summaries...)
	for _, sr := range m.currentIter {
		allSummaries = append(allSummaries, sr.Summary)
	}

	msgs := m.assembler.AssembleFinal(systemPrompt, task, allSummaries)

	duration := time.Since(start)
	attrs := map[string]any{
		"call_type":        "final",
		"message_count":    len(msgs),
		"total_summaries":  len(allSummaries),
		"assembly_time_us": duration.Microseconds(),
	}

	if m.tracer != nil {
		span := m.tracer.Start("context.build_final", attrs)
		span.End()
	}

	log.Printf("[context] final: %d messages, %d total summaries, assembled in %v",
		len(msgs), len(allSummaries), duration)
	return msgs
}

// ExecuteSandboxed runs sub-tasks through the sandbox and collects summaries.
func (m *Manager) ExecuteSandboxed(subTasks []string, toolDefs []ToolDef) {
	start := time.Now()

	results := m.sandbox.ExecuteSubTasks(subTasks, toolDefs)
	m.currentIter = results

	for _, r := range results {
		m.stats.AddRecord(CallSandboxSelect, r.TokensUsed)
		m.stats.AddRecord(CallSandboxEval, 0)
	}

	duration := time.Since(start)
	totalSandboxTokens := 0
	for _, r := range results {
		totalSandboxTokens += r.TokensUsed
	}

	attrs := map[string]any{
		"call_type":       "sandbox",
		"sub_task_count":  len(subTasks),
		"result_count":    len(results),
		"total_tokens":    totalSandboxTokens,
		"execution_time":  duration.String(),
	}

	if m.tracer != nil {
		span := m.tracer.Start("context.execute_sandboxed", attrs)
		span.End()
	}

	log.Printf("[context] sandbox: %d sub-tasks → %d results, %d tokens, took %v",
		len(subTasks), len(results), totalSandboxTokens, duration)
}

// CommitIteration finalizes the current iteration's summaries.
func (m *Manager) CommitIteration() {
	committed := len(m.currentIter)
	for _, sr := range m.currentIter {
		m.summaries = append(m.summaries, sr.Summary)
	}
	m.currentIter = nil

	attrs := map[string]any{
		"committed_count":  committed,
		"total_summaries":  len(m.summaries),
	}

	if m.tracer != nil {
		span := m.tracer.Start("context.commit_iteration", attrs)
		span.End()
	}

	log.Printf("[context] iteration committed: %d summaries added, %d total",
		committed, len(m.summaries))
}

// RecordTokens records token usage for a call type.
func (m *Manager) RecordTokens(ct CallType, tokens int) {
	m.stats.AddRecord(ct, tokens)

	attrs := map[string]any{
		"call_type": ct.String(),
		"tokens":    tokens,
		"running_total": m.stats.TotalTokens,
	}

	if m.tracer != nil {
		span := m.tracer.Start("context.record_tokens", attrs)
		span.End()
	}

	log.Printf("[context] tokens: %s=%d (total: %d)", ct.String(), tokens, m.stats.TotalTokens)
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
