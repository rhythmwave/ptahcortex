# Architecture

## Overview

AgentKit is a Go framework for building AI agents that use MCP (Model Context Protocol) for tool calling. The core design principles:

1. **Separation of concerns** — Agent logic, tool execution, LLM communication, and observability are independent layers
2. **Config-driven** — Agents are defined in YAML, not code
3. **MCP-first** — Tools are discovered and called via MCP protocol, not hardcoded
4. **Production-ready** — Built-in retry, timeout, cost tracking, OTel observability

## Core Components

### 1. Agent Loop

The agent loop is the heart of AgentKit. Each iteration follows:

```
┌─────────┐     ┌──────────┐     ┌───────────┐
│  Plan   │────→│ Execute  │────→│  Reflect  │
│  (LLM)  │     │ (Tools)  │     │   (LLM)   │
└─────────┘     └──────────┘     └───────────┘
     ↑                                   │
     └───────────── if needed ───────────┘
```

**Plan phase:**
- Send task + available tools + context to LLM
- LLM returns a plan: which tools to call, with what arguments
- Agent parses tool calls from LLM response

**Execute phase:**
- Independent tool calls run in parallel (goroutines)
- Dependent tools run sequentially (output feeds input)
- Each call has timeout + retry
- Results collected and formatted

**Reflect phase:**
- LLM evaluates tool results
- Decides: done (return result), or need more iterations
- If more: new plan with updated context
- Token budget checked before continuing

### 2. MCP Client

Implements the MCP protocol (stdio JSON-RPC 2.0):

```go
// Client manages a single MCP server connection
type Client struct {
    cmd     *exec.Cmd
    stdin   io.WriteCloser
    stdout  *bufio.Reader
    tools   []Tool
    nextID  int
}

// Key operations:
// - Initialize: handshake with server
// - ListTools: discover available tools
// - CallTool: execute a tool with arguments
// - Close: graceful shutdown
```

**Manager** handles multiple MCP servers:
- Starts each server process
- Aggregates tools from all servers
- Routes tool calls to correct server
- Handles server failures independently

### 3. LLM Provider Interface

```go
type Provider interface {
    Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
    Name() string
}

type ChatRequest struct {
    Messages  []Message
    Tools     []ToolDefinition
    MaxTokens int
    Model     string
}

type ChatResponse struct {
    Content   string
    ToolCalls []ToolCall
    Usage     TokenUsage
}
```

Implementations:
- **OpenAI** — `/v1/chat/completions` compatible
- **Anthropic** — `/v1/messages` compatible
- **Proxy** — routes through Code Agent Proxy (supports both APIs)

### 4. Tool Execution Engine

Handles the mechanics of calling tools:

- **Parallel execution** — independent tools run concurrently with semaphore limiting
- **Sequential execution** — dependent tools run in order
- **Retry** — exponential backoff with jitter
- **Timeout** — per-tool and per-iteration
- **Error handling** — structured errors with context

```go
type ToolExecutor struct {
    maxParallel int
    timeout     time.Duration
    retryPolicy RetryPolicy
    mcpManager  *mcp.Manager
}

func (e *ToolExecutor) Execute(ctx context.Context, calls []ToolCall) ([]ToolResult, error)
```

### 5. OTel Observability

Every component emits telemetry:

**Traces:**
- `agent.run` — full agent execution (parent span)
- `agent.iteration` — single plan→execute→reflect cycle
- `agent.plan` — LLM planning call
- `agent.execute` — tool execution batch
- `agent.reflect` — LLM reflection call
- `mcp.call_tool` — individual MCP tool call
- `llm.chat` — LLM API call

**Metrics:**
- `agentkit_iterations_total` — counter by agent/status
- `agentkit_tool_calls_total` — counter by tool/status
- `agentkit_llm_tokens_total` — counter by direction (input/output)
- `agentkit_llm_latency_seconds` — histogram
- `agentkit_tool_latency_seconds` — histogram by tool

**Logs:**
- Structured JSON with trace correlation
- Agent ID, iteration, tool name in every log line

## Data Flow

```
User Input
    │
    ▼
┌─────────────┐
│ Agent.Run() │
└──────┬──────┘
       │
       ▼
┌─────────────────────────────────────────┐
│ Iteration Loop (max N times)            │
│                                         │
│  1. Plan: LLM + tools + context → plan  │
│     └─ OTel: agent.plan span            │
│                                         │
│  2. Execute: parallel/sequential tools  │
│     ├─ OTel: agent.execute span         │
│     └─ OTel: mcp.call_tool per tool     │
│                                         │
│  3. Reflect: LLM evaluates results      │
│     ├─ OTel: agent.reflect span         │
│     └─ Decision: done or continue       │
└─────────────────────────────────────────┘
       │
       ▼
   Final Result
```

## Configuration

Two config files:

### agent.yaml — Agent Definition
```yaml
name: my-agent
description: What this agent does

llm:
  provider: openai
  model: gpt-4o
  base_url: http://localhost:8080/v1
  max_tokens: 4096

mcp_servers:
  - name: lexa
    command: /path/to/lexa
    args: ["serve"]
    cwd: /path/to/project

tools:
  max_parallel: 5
  timeout: 30s
  retry:
    max_attempts: 3
    backoff: 1s

agent:
  max_iterations: 5
  max_tokens_per_run: 50000
  reflect_after_each: true
```

### mcp-servers.yaml — MCP Server Registry
```yaml
servers:
  lexa:
    description: Code intelligence
    command: /home/deploy/.local/bin/lexa
    args: ["serve"]
    cwd: /home/deploy/commit-reviewer-src
    tools:
      - search_code
      - find_symbol
      - find_references

  filesystem:
    description: File operations
    command: npx
    args: ["-y", "@modelcontextprotocol/server-filesystem", "/data"]
    tools:
      - read_file
      - write_file
      - list_directory
```

## Error Handling

AgentKit handles errors at multiple levels:

1. **Tool errors** — retry, then report to LLM as tool error result
2. **LLM errors** — retry with backoff, fallback to alternative provider
3. **MCP server errors** — restart server, retry tool call
4. **Timeout errors** — cancel current iteration, report partial results
5. **Budget exceeded** — stop iteration loop, return best result so far

All errors are structured:

```go
type AgentError struct {
    Code     string
    Message  string
    Tool     string
    Iteration int
    Err      error
}
```

## Security Considerations

- MCP servers run as child processes with bounded permissions
- Tool arguments are validated against JSON schema before execution
- Token budgets prevent runaway LLM costs
- Rate limiting prevents abuse
- No arbitrary code execution — only MCP tool calls
