# Tool Calling Patterns

## Overview

Ptahcortex supports multiple tool calling patterns based on dependency analysis. The LLM's tool calls are analyzed for dependencies, then executed optimally.

## Pattern 1: Parallel Independent Calls

When the LLM requests multiple tools with no dependencies:

```
LLM Response:
  - call search_code("error handling")
  - call search_code("retry logic")
  - call find_symbol("HandleError")
```

Ptahcortex executes all three in parallel:

```go
// Semaphore-limited parallel execution
func (e *ToolExecutor) executeParallel(ctx context.Context, calls []ToolCall) []ToolResult {
    results := make([]ToolResult, len(calls))
    sem := make(chan struct{}, e.maxParallel)
    var wg sync.WaitGroup

    for i, call := range calls {
        wg.Add(1)
        go func(i int, call ToolCall) {
            defer wg.Done()
            sem <- struct{}{}
            defer func() { <-sem }()
            results[i] = e.executeOne(ctx, call)
        }(i, call)
    }
    wg.Wait()
    return results
}
```

**Concurrency limit:** `max_parallel` (default 5) prevents overwhelming MCP servers.

## Pattern 2: Sequential Dependent Calls

When tool B needs output from tool A:

```
LLM Response:
  - call find_symbol("HandleError") → get file:line
  - call read_file({path: result[0].file}) → get source
  - call analyze_code({code: result[1].content}) → get analysis
```

Ptahcortex detects the dependency chain and executes in order:

```go
func (e *ToolExecutor) executeSequential(ctx context.Context, calls []ToolCall) []ToolResult {
    results := make([]ToolResult, len(calls))
    for i, call := range calls {
        // Substitute previous results into arguments
        resolved := e.resolveArgs(call, results[:i])
        results[i] = e.executeOne(ctx, resolved)
        if results[i].IsError {
            break // Stop chain on error
        }
    }
    return results
}
```

## Pattern 3: Retry with Backoff

Failed tools are retried with exponential backoff:

```go
type RetryPolicy struct {
    MaxAttempts int           // default: 3
    BaseBackoff time.Duration // default: 1s
    MaxBackoff  time.Duration // default: 30s
    Jitter      bool          // default: true
}

func (e *ToolExecutor) executeWithRetry(ctx context.Context, call ToolCall) ToolResult {
    var lastErr error
    for attempt := 0; attempt <= e.retryPolicy.MaxAttempts; attempt++ {
        if attempt > 0 {
            backoff := e.retryPolicy.Backoff(attempt)
            select {
            case <-time.After(backoff):
            case <-ctx.Done():
                return ToolResult{IsError: true, Error: "context cancelled"}
            }
        }
        result := e.executeOne(ctx, call)
        if !result.IsError {
            return result
        }
        lastErr = result.Error
    }
    return ToolResult{IsError: true, Error: fmt.Sprintf("failed after %d attempts: %v", e.retryPolicy.MaxAttempts+1, lastErr)}
}
```

## Pattern 4: Timeout Management

Two levels of timeout:

```yaml
tools:
  timeout: 30s          # Per-tool timeout
agent:
  iteration_timeout: 5m # Per-iteration timeout (all tools combined)
```

```go
// Per-tool timeout
func (e *ToolExecutor) executeOne(ctx context.Context, call ToolCall) ToolResult {
    toolCtx, cancel := context.WithTimeout(ctx, e.timeout)
    defer cancel()
    
    resultCh := make(chan ToolResult, 1)
    go func() {
        resultCh <- e.mcpManager.CallTool(toolCtx, call.Name, call.Arguments)
    }()
    
    select {
    case result := <-resultCh:
        return result
    case <-toolCtx.Done():
        return ToolResult{IsError: true, Error: "tool timeout"}
    }
}
```

## Pattern 5: Error Reporting to LLM

When a tool fails, the error is reported back to the LLM as a tool result:

```json
{
  "role": "tool",
  "tool_call_id": "call_123",
  "content": "Error: MCP server 'lexa' not responding after 3 attempts. Last error: connection refused"
}
```

The LLM can then:
- Try a different tool
- Adjust its approach
- Report the limitation to the user

## Pattern 6: Result Truncation

Large tool results are truncated to fit the context window:

```go
func (r *ToolResult) Truncate(maxTokens int) ToolResult {
    if r.TokenEstimate() <= maxTokens {
        return *r
    }
    // Truncate content and add note
    truncated := r.Content[:maxTokens*4] // rough char estimate
    return ToolResult{
        Content: truncated + "\n\n[Result truncated — showing first " + 
                 fmt.Sprintf("%d", maxTokens) + " tokens]",
        IsTruncated: true,
    }
}
```

## Tool Call ID Tracking

Every tool call gets a unique ID for correlation:

```go
type ToolCall struct {
    ID        string         // Unique ID (from LLM or generated)
    Name      string         // Tool name
    Arguments map[string]any // Tool arguments
    Server    string         // Which MCP server
}

type ToolResult struct {
    CallID  string // Matches ToolCall.ID
    Content string
    IsError bool
    Latency time.Duration
}
```

## Adding New Patterns

To add a new execution pattern:

1. Implement the `Pattern` interface:
```go
type Pattern interface {
    Match(calls []ToolCall) bool
    Execute(ctx context.Context, calls []ToolCall) []ToolResult
    Priority() int // Higher = checked first
}
```

2. Register in the executor:
```go
executor.RegisterPattern(&MyCustomPattern{})
```

3. Patterns are checked in priority order. First match wins.
