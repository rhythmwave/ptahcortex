package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/rhythmwave/ptahcortex/internal/mcp"
)

// Executor handles tool calls with parallel execution, retry, and timeout.
type Executor struct {
	manager     *mcp.Manager
	maxParallel int
	timeout     time.Duration
	retryPolicy RetryPolicy
}

// RetryPolicy configures retry behavior.
type RetryPolicy struct {
	MaxAttempts int
	BaseBackoff time.Duration
	MaxBackoff  time.Duration
}

// DefaultRetry returns sensible defaults.
func DefaultRetry() RetryPolicy {
	return RetryPolicy{
		MaxAttempts: 3,
		BaseBackoff: 1 * time.Second,
		MaxBackoff:  10 * time.Second,
	}
}

// NewExecutor creates a tool executor.
func NewExecutor(manager *mcp.Manager, maxParallel int, timeout time.Duration, retry RetryPolicy) *Executor {
	if maxParallel <= 0 {
		maxParallel = 5
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &Executor{
		manager:     manager,
		maxParallel: maxParallel,
		timeout:     timeout,
		retryPolicy: retry,
	}
}

// CallResult is the result of a single tool call.
type CallResult struct {
	CallID  string
	Name    string
	Content string
	IsError bool
	Latency time.Duration
	Error   error
}

// ExecuteCalls runs a batch of tool calls. Independent calls run in parallel.
func (e *Executor) ExecuteCalls(ctx context.Context, calls []mcp.ToolCall) []CallResult {
	if len(calls) == 0 {
		return nil
	}

	// Single call — no parallelism needed
	if len(calls) == 1 {
		return []CallResult{e.executeOne(ctx, calls[0])}
	}

	// Multiple calls — run in parallel with semaphore
	results := make([]CallResult, len(calls))
	sem := make(chan struct{}, e.maxParallel)
	var wg sync.WaitGroup

	for i, call := range calls {
		wg.Add(1)
		go func(i int, call mcp.ToolCall) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			results[i] = e.executeOne(ctx, call)
		}(i, call)
	}

	wg.Wait()
	return results
}

// executeOne calls a single tool with timeout and retry.
func (e *Executor) executeOne(ctx context.Context, call mcp.ToolCall) CallResult {
	var lastErr error

	for attempt := 0; attempt <= e.retryPolicy.MaxAttempts; attempt++ {
		if attempt > 0 {
			backoff := e.backoff(attempt)
			log.Printf("[tools] retry %d/%d for %s after %v", attempt, e.retryPolicy.MaxAttempts, call.Name, backoff)
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return CallResult{
					CallID:  call.ID,
					Name:    call.Name,
					IsError: true,
					Error:   ctx.Err(),
				}
			}
		}

		// Per-tool timeout
		toolCtx, cancel := context.WithTimeout(ctx, e.timeout)
		start := time.Now()

		resultCh := make(chan CallResult, 1)
		go func() {
			r, err := e.manager.CallTool(call.Name, call.Arguments)
			if err != nil {
				resultCh <- CallResult{
					CallID:  call.ID,
					Name:    call.Name,
					IsError: true,
					Error:   err,
					Latency: time.Since(start),
				}
				return
			}
			resultCh <- CallResult{
				CallID:  call.ID,
				Name:    call.Name,
				Content: r.Content,
				IsError: r.IsError,
				Latency: time.Since(start),
			}
		}()

		select {
		case result := <-resultCh:
			cancel()
			if !result.IsError {
				return result
			}
			lastErr = result.Error
			if result.Error == nil {
				// Tool returned isError=true but no Go error — don't retry
				return result
			}
		case <-toolCtx.Done():
			cancel()
			lastErr = fmt.Errorf("tool %s timeout after %v", call.Name, e.timeout)
		}
	}

	return CallResult{
		CallID:  call.ID,
		Name:    call.Name,
		IsError: true,
		Error:   fmt.Errorf("failed after %d attempts: %w", e.retryPolicy.MaxAttempts+1, lastErr),
		Latency: e.timeout * time.Duration(e.retryPolicy.MaxAttempts+1),
	}
}

// backoff calculates exponential backoff with the attempt number.
func (e *Executor) backoff(attempt int) time.Duration {
	d := e.retryPolicy.BaseBackoff * time.Duration(1<<uint(attempt-1))
	if d > e.retryPolicy.MaxBackoff {
		d = e.retryPolicy.MaxBackoff
	}
	return d
}

// FormatResultsForLLM converts tool results into messages for the LLM.
func FormatResultsForLLM(results []CallResult) []map[string]string {
	var messages []map[string]string
	for _, r := range results {
		content := r.Content
		if r.IsError {
			if r.Error != nil {
				content = fmt.Sprintf("Error: %v", r.Error)
			} else {
				content = "Error: tool returned error"
			}
		}
		messages = append(messages, map[string]string{
			"tool_call_id": r.CallID,
			"content":      content,
		})
	}
	return messages
}

// ParseToolCalls extracts tool calls from an LLM response's tool_calls JSON.
func ParseToolCalls(raw []byte) ([]mcp.ToolCall, error) {
	var calls []struct {
		ID   string `json:"id"`
		Type string `json:"type"`
		Fn   struct {
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		} `json:"function"`
	}
	if err := json.Unmarshal(raw, &calls); err != nil {
		return nil, err
	}

	var result []mcp.ToolCall
	for _, c := range calls {
		var args map[string]any
		json.Unmarshal([]byte(c.Fn.Arguments), &args)
		result = append(result, mcp.ToolCall{
			ID:        c.ID,
			Name:      c.Fn.Name,
			Arguments: args,
		})
	}
	return result, nil
}
