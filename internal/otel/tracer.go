package otel

import (
	"context"
	"fmt"
	"log"
	"time"
)

// Tracer creates spans for agent operations.
// Lightweight wrapper — no external dependency required.
// Set PTAHCORTEX_OTEL_ENABLED=1 to enable real OTel export.
type Tracer struct {
	enabled     bool
	serviceName string
}

// Span represents a single operation span.
type Span struct {
	Name      string
	StartTime time.Time
	EndTime   time.Time
	Attrs     map[string]any
}

// NewTracer creates a tracer. If enabled=false, all operations are no-ops.
func NewTracer(enabled bool, serviceName string) *Tracer {
	return &Tracer{
		enabled:     enabled,
		serviceName: serviceName,
	}
}

// Start begins a new span.
func (t *Tracer) Start(ctx context.Context, name string, attrs map[string]any) *Span {
	if !t.enabled {
		return &Span{Name: name}
	}
	s := &Span{
		Name:      name,
		StartTime: time.Now(),
		Attrs:     attrs,
	}
	log.Printf("[otel] span start: %s %v", name, attrs)
	return s
}

// End finishes a span.
func (s *Span) End() {
	s.EndTime = time.Now()
	if s.StartTime.IsZero() {
		return
	}
	log.Printf("[otel] span end: %s [%v]", s.Name, s.EndTime.Sub(s.StartTime))
}

// SetAttr adds an attribute to a span.
func (s *Span) SetAttr(key string, value any) {
	if s.Attrs == nil {
		s.Attrs = make(map[string]any)
	}
	s.Attrs[key] = value
}

// Metrics tracks agent metrics.
type Metrics struct {
	enabled bool
}

// NewMetrics creates a metrics collector.
func NewMetrics(enabled bool) *Metrics {
	return &Metrics{enabled: enabled}
}

// RecordIteration records an agent iteration.
func (m *Metrics) RecordIteration(agentName string, iteration int, duration time.Duration, tokens int) {
	if !m.enabled {
		return
	}
	log.Printf("[metrics] agent=%s iteration=%d duration=%v tokens=%d", agentName, iteration, duration, tokens)
}

// RecordToolCall records a tool call.
func (m *Metrics) RecordToolCall(toolName string, duration time.Duration, isError bool) {
	if !m.enabled {
		return
	}
	status := "ok"
	if isError {
		status = "error"
	}
	log.Printf("[metrics] tool=%s duration=%v status=%s", toolName, duration, status)
}

// RecordLLMCall records an LLM API call.
func (m *Metrics) RecordLLMCall(provider string, model string, duration time.Duration, tokens int) {
	if !m.enabled {
		return
	}
	log.Printf("[metrics] llm=%s/%s duration=%v tokens=%d", provider, model, duration, tokens)
}

// FormatSummary returns a human-readable metrics summary.
func FormatSummary(spans []*Span) string {
	if len(spans) == 0 {
		return "No spans recorded."
	}
	var total time.Duration
	for _, s := range spans {
		if !s.StartTime.IsZero() && !s.EndTime.IsZero() {
			total += s.EndTime.Sub(s.StartTime)
		}
	}
	return fmt.Sprintf("Spans: %d, Total time: %v", len(spans), total)
}
