# Ptahcortex Architecture

## Overview

Ptahcortex is a Go framework for building AI agents with MCP tool calling and call-aware context assembly.

**Key innovation:** Different LLM calls get different context based on their purpose, reducing token usage by ~72-90%.

## Core Principles

1. **MCP-native** — tools come from MCP servers, not hardcoded plugins
2. **Call-aware context** — different context per LLM call type
3. **Sandboxed tool reasoning** — isolated LLM calls for tool selection
4. **Summary flow** — only summaries flow up, raw results stay local
5. **Production-ready** — retry, timeout, observability, single binary

## System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Ptahcortex Runtime                        │
│                                                             │
│  ┌───────────────────────────────────────────────────────┐  │
│  │                  Context Manager                       │  │
│  │                                                       │  │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐           │  │
│  │  │ Assembler│  │ Sandbox  │  │ Summarizer│           │  │
│  │  │ (per     │  │ (isolated│  │ (summary  │           │  │
│  │  │ calltype)│  │  LLM)    │  │  flow)    │           │  │
│  │  └──────────┘  └──────────┘  └──────────┘           │  │
│  └───────────────────────────────────────────────────────┘  │
│                           │                                  │
│  ┌────────────────────────┴──────────────────────────────┐  │
│  │                  Agent Loop                            │  │
│  │                                                       │  │
│  │  Plan → Sandbox → Collect → Reflect → Final           │  │
│  │    │        │         │         │         │           │  │
│  │    │        │         │         │         │           │  │
│  │    ▼        ▼         ▼         ▼         ▼           │  │
│  │  T0+T1   T0+sub   summary   T0+T1+T2  T0+T1+all     │  │
│  │  +T3     +tools    only      +summaries +summaries    │  │
│  └───────────────────────────────────────────────────────┘  │
│                           │                                  │
│  ┌────────────────────────┴──────────────────────────────┐  │
│  │                 Tool Execution                         │  │
│  │                                                       │  │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐           │  │
│  │  │ Parallel │  │ Retry    │  │ Timeout  │           │  │
│  │  │ Executor │  │ (backoff)│  │ (per-tool)│           │  │
│  │  └──────────┘  └──────────┘  └──────────┘           │  │
│  └───────────────────────────────────────────────────────┘  │
│                           │                                  │
│  ┌────────────────────────┴──────────────────────────────┐  │
│  │               MCP Client Manager                      │  │
│  │                                                       │  │
│  │  ┌────────┐  ┌────────┐  ┌────────┐  ┌────────┐     │  │
│  │  │ Lexa   │  │ File   │  │ Web    │  │ Custom │     │  │
│  │  │ (code) │  │ System │  │ Search │  │ Server │     │  │
│  │  └────────┘  └────────┘  └────────┘  └────────┘     │  │
│  └───────────────────────────────────────────────────────┘  │
│                           │                                  │
│  ┌────────────────────────┴──────────────────────────────┐  │
│  │               LLM Provider Interface                   │  │
│  │                                                       │  │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐           │  │
│  │  │ OpenAI   │  │ Anthropic│  │ Proxy    │           │  │
│  │  │ Compat   │  │ Compat   │  │ (Code    │           │  │
│  │  │          │  │          │  │  Agent)  │           │  │
│  │  └──────────┘  └──────────┘  └──────────┘           │  │
│  └───────────────────────────────────────────────────────┘  │
│                           │                                  │
│  ┌────────────────────────┴──────────────────────────────┐  │
│  │               OTel Observability                       │  │
│  │                                                       │  │
│  │  Traces: agent.run → agent.iteration → tool.call      │  │
│  │  Metrics: tokens, latency, tool calls, errors         │  │
│  │  Logs: structured JSON with trace correlation         │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

## Context Engineering Stack

The context manager has four layers:

```
┌─────────────────────────────────────────────┐
│  4. Local LLM Selection                     │ ← Relevance filter (Phase 4)
├─────────────────────────────────────────────┤
│  3. Tool Sandboxing                         │ ← Isolated tool reasoning (Phase 2)
├─────────────────────────────────────────────┤
│  2. Call-Aware Assembly                     │ ← Different context per call type (Phase 1)
├─────────────────────────────────────────────┤
│  1. Summary Flow                            │ ← Only summaries flow up (Phase 3)
└─────────────────────────────────────────────┘
```

### Layer 1: Summary Flow

Raw tool results stay in the sandbox. Only concise summaries flow to the main loop.

```
Tool Result (10K chars) → Sandbox → Summary (500 chars) → Main Loop
```

### Layer 2: Call-Aware Assembly

Different LLM calls get different context based on their purpose.

| Call Type | Context Tiers | What's Included |
|-----------|--------------|-----------------|
| Plan | T0 + T1 + T3 | System + task + previous summaries |
| Sandbox Select | T0 + sub-task | System + sub-task + tool descriptions |
| Sandbox Eval | sub-task + result | Sub-task + tool result (truncated) |
| Reflect | T0 + T1 + T2 | System + task + current summaries |
| Final | T0 + T1 + all | System + task + all relevant summaries |

### Layer 3: Tool Sandboxing

Tool reasoning happens in isolated, cheap LLM calls. Main loop never sees raw tool output.

```
Main Loop → "Find error handling in main.go"
                ↓
Sandbox (isolated, minimal context):
  Select: "Use text_search"
  Call: text_search("error handling", "main.go")
  Eval: "Error handling uses structured errors with retry..."
                ↓
Main Loop ← Summary only
```

### Layer 4: Local LLM Selection (Optional)

Small local model filters context before cloud call.

```
All Context (50K tokens)
        ↓
Local LLM (3B, free): "Indices 0, 2, 5 are relevant"
        ↓
Filtered Context (15K tokens) → Cloud LLM
```

## Data Flow

```
User Task
    │
    ▼
┌─────────────────────────────────────────┐
│ Iteration Loop (max N times)            │
│                                         │
│  1. Plan: ContextAssembler(T0+T1+T3)   │
│     └─ LLM call → plan + sub-tasks      │
│                                         │
│  2. Sandbox: for each sub-task          │
│     ├─ Select: ContextAssembler(T0+sub) │
│     │  └─ LLM call → tool selection     │
│     ├─ Execute: MCP tool call           │
│     └─ Eval: ContextAssembler(sub+res)  │
│        └─ LLM call → summary            │
│                                         │
│  3. Collect: aggregate sandbox summaries │
│                                         │
│  4. Reflect: ContextAssembler(T0+T1+T2) │
│     └─ LLM call → done or continue      │
└─────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────┐
│ Final: ContextAssembler(T0+T1+all)      │
│ └─ LLM call → final answer              │
└─────────────────────────────────────────┘
    │
    ▼
Response to User
```

## Token Comparison

### Without Context Engineering

```
Iteration 1: 4,700 tokens
Iteration 2: 7,700 tokens
Iteration 3: 10,700 tokens
...
Iteration 20: 25,700 tokens
Total: ~650,000 tokens
```

### With Context Engineering (Phases 1-3)

```
Main calls:    20 × 2,200  =  44,000 tokens
Sandbox calls: 60 × 1,100  =  66,000 tokens
Reflect calls: 20 × 3,200  =  64,000 tokens
Final call:     1 × 5,700  =   5,700 tokens
Total:                        ~180,000 tokens
Savings:                      ~72%
```

### With Context Engineering (All Phases)

```
Same as above, but:
- Local LLM filters context before cloud calls
- Additional ~70% cost reduction on remaining tokens
- Net savings: ~90%+ cost reduction
```

## Configuration

```yaml
# Agent definition
name: code-reviewer
description: Reviews pull requests using code intelligence

# LLM settings
llm:
  provider: openai
  model: gpt-4o
  base_url: http://localhost:8080/v1
  max_tokens: 4096

# MCP servers
mcp_servers:
  - name: lexa
    command: /home/deploy/.local/bin/lexa
    args: ["mcp"]
    cwd: /home/deploy/commit-reviewer-src

# Tool execution
tools:
  max_parallel: 5
  timeout: 30s
  retry:
    max_attempts: 3
    backoff: 1s

# Agent loop
agent:
  max_iterations: 5
  max_tokens_per_run: 50000

# Context engineering
context:
  # Call-aware assembly
  assembly:
    enabled: true
    max_tool_result_len: 4000
    max_messages: 20
    keep_first_user: true

  # Tool sandboxing
  sandbox:
    enabled: true
    max_sandbox_iterations: 3
    summary_max_tokens: 500

  # Local LLM selection (optional)
  selector:
    enabled: false
    provider: ollama
    model: phi3-mini
    base_url: http://localhost:11434
    max_input_tokens: 8000
```

## Project Structure

```
ptahcortex/
├── cmd/ptahcortex/          # CLI entrypoint
│   └── main.go
├── internal/
│   ├── agent/               # Agent loop + context manager
│   │   ├── agent.go         # Main agent loop
│   │   ├── context.go       # CallType enum + ContextAssembler
│   │   ├── assembler.go     # Message assembly per call type
│   │   ├── sandbox.go       # Sandbox executor
│   │   ├── summarizer.go    # Summary extraction
│   │   └── local_selector.go # Local LLM selector (Phase 4)
│   ├── mcp/                 # MCP client (stdio JSON-RPC)
│   │   ├── client.go
│   │   └── manager.go
│   ├── llm/                 # LLM provider interface
│   │   └── provider.go      # OpenAI + Anthropic + Proxy
│   ├── tools/               # Tool execution engine
│   │   └── executor.go      # Parallel, retry, timeout
│   ├── otel/                # Observability
│   │   └── tracer.go        # Spans + metrics
│   └── config/              # Configuration
│       └── config.go        # YAML loader
├── configs/                 # Example agent configs
│   ├── agent.yaml
│   ├── code-reviewer.yaml
│   ├── doc-qa.yaml
│   └── task-planner.yaml
└── docs/                    # Documentation
    ├── ARCHITECTURE.md      # This file
    ├── CONTEXT-MANAGER.md   # Context manager design
    ├── CONTEXT-ENGINEERING.md # Landscape analysis
    ├── VALIDATION.md        # Feasibility assessment
    ├── TASK-BREAKDOWN.md    # Phased implementation plan
    ├── MCP.md               # MCP integration guide
    ├── TOOLS.md             # Tool calling patterns
    └── ROADMAP.md           # Development roadmap
```

## Related Documentation

- [Context Manager Design](CONTEXT-MANAGER.md) — detailed design with call types and tiers
- [Context Engineering Landscape](CONTEXT-ENGINEERING.md) — field analysis and positioning
- [Validation Assessment](VALIDATION.md) — honest confidence levels and risks
- [Task Breakdown](TASK-BREAKDOWN.md) — phased implementation plan
- [MCP Integration](MCP.md) — MCP protocol and server setup
- [Tool Calling Patterns](TOOLS.md) — parallel, retry, timeout patterns
- [Development Roadmap](ROADMAP.md) — 6-week timeline
