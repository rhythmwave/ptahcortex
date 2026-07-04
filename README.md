# Ptahcortex

A lightweight Go framework for building production-ready AI agents with MCP (Model Context Protocol) tool calling.

## Why

Most AI agent frameworks are either too simple (single-turn chat) or too complex (heavy orchestration with lock-in). Ptahcortex sits in the middle:

- **MCP-native** — tools come from any MCP server, not hardcoded plugins
- **Production patterns** — retry, timeout, cost tracking, observability
- **Config-driven** — define agents in YAML, no code changes for new workflows
- **Go** — single binary, fast startup, easy deployment

## Architecture

```
┌─────────────────────────────────────────────────┐
│                  Ptahcortex Runtime                │
│                                                  │
│  ┌──────────┐  ┌──────────┐  ┌──────────────┐  │
│  │ Planner  │  │ Executor │  │  Reflector   │  │
│  │  (LLM)   │→│ (Tools)  │→│   (LLM)      │  │
│  └──────────┘  └──────────┘  └──────────────┘  │
│        │              │              │           │
│        └──────────────┼──────────────┘           │
│                       │                          │
│  ┌────────────────────┴───────────────────────┐  │
│  │            MCP Client Manager              │  │
│  │  ┌────────┐ ┌────────┐ ┌────────────────┐  │  │
│  │  │ Lexa   │ │ File   │ │ Custom Server  │  │  │
│  │  │ (code) │ │ System │ │ (your tools)   │  │  │
│  │  └────────┘ └────────┘ └────────────────┘  │  │
│  └─────────────────────────────────────────────┘  │
│                       │                          │
│  ┌────────────────────┴───────────────────────┐  │
│  │           LLM Provider Interface           │  │
│  │  ┌──────────┐ ┌──────────┐ ┌────────────┐  │  │
│  │  │ OpenAI   │ │ Anthropic│ │ Code Agent │  │  │
│  │  │ Compat   │ │ Compat   │ │ Proxy      │  │  │
│  │  └──────────┘ └──────────┘ └────────────┘  │  │
│  └─────────────────────────────────────────────┘  │
│                       │                          │
│  ┌────────────────────┴───────────────────────┐  │
│  │           OTel Observability               │  │
│  │  Traces │ Metrics │ Logs                   │  │
│  └─────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────┘
```

## Core Concepts

### Context Manager (Call-Aware Assembly)

Ptahcortex's differentiator: **different LLM calls get different context**.

Instead of sending everything to every LLM call, the Context Manager assembles messages based on call type:

- **Plan** — T0 (system + tools) + T1 (task) + T3 (summaries from previous iterations)
- **Sandbox Select** — minimal: just tool defs + sub-task
- **Sandbox Eval** — sub-task + truncated tool result
- **Reflect** — T0 + T1 + T2 (current summaries) + T3 (previous summaries)
- **Final** — T0 + T1 + all summaries

This achieves ~72% token savings at 20 iterations compared to naive approaches.

### Agent Loop (Plan → Execute → Reflect)

Every agent follows a three-phase loop:

1. **Plan** — LLM analyzes the task, decides which tools to call and in what order
2. **Execute** — Tools are called in sandboxed isolation (minimal context per tool call)
3. **Reflect** — LLM evaluates results, decides if more work is needed or if done

The loop runs up to `max_iterations` times. Each iteration is a full plan→execute→reflect cycle.

### MCP Tool Calling

Tools are not hardcoded. They come from MCP servers:

- **Lexa** — code intelligence (search, symbols, references)
- **Filesystem** — file read/write/search
- **Web** — fetch, scrape, search
- **Custom** — any stdio MCP server you write

Ptahcortex discovers tools at startup via `tools/list`, then makes them available to the LLM.

### Tool Orchestration

When the LLM requests multiple tools:

- **Independent calls** → executed in parallel (goroutines)
- **Sequential dependencies** → executed in order (output feeds next input)
- **Error handling** → retry with backoff, fallback to alternative tool
- **Timeout** → per-tool and per-iteration timeouts

## Project Structure

```
agentkit/
├── cmd/agentkit/          # CLI entrypoint
│   └── main.go
├── internal/
│   ├── agent/             # Agent loop (plan/execute/reflect)
│   │   ├── agent.go
│   │   ├── planner.go
│   │   ├── executor.go
│   │   ├── reflector.go
│   │   └── config.go
│   ├── mcp/               # MCP client (stdio JSON-RPC)
│   │   ├── client.go
│   │   ├── manager.go     # Multi-server manager
│   │   └── types.go
│   ├── llm/               # LLM provider interface
│   │   ├── provider.go    # Interface definition
│   │   ├── openai.go      # OpenAI-compatible
│   │   ├── anthropic.go   # Anthropic-compatible
│   │   └── proxy.go       # Code Agent Proxy client
│   ├── otel/              # Observability
│   │   ├── tracer.go
│   │   ├── metrics.go
│   │   └── middleware.go
│   └── tools/             # Tool execution engine
│       ├── executor.go
│       ├── parallel.go
│       └── retry.go
├── examples/
│   ├── code-reviewer/     # Review PRs with Lexa + LLM
│   ├── doc-qa/            # Document Q&A with filesystem MCP
│   └── task-planner/      # Multi-step task decomposition
├── configs/
│   ├── agent.yaml         # Agent definition
│   └── mcp-servers.yaml   # MCP server registry
├── docs/
│   ├── ARCHITECTURE.md    # Deep dive
│   ├── MCP.md             # MCP integration guide
│   ├── TOOLS.md           # Tool calling patterns
│   ├── OBSERVABILITY.md   # OTel setup
│   └── EXAMPLES.md        # Walkthrough of examples
└── go.mod
```

## Example Agent Config

```yaml
# configs/agent.yaml
name: code-reviewer
description: Reviews pull requests using code intelligence

llm:
  provider: openai          # or anthropic, proxy
  model: gpt-4o
  base_url: http://localhost:8080/v1  # Code Agent Proxy
  max_tokens: 4096

mcp_servers:
  - name: lexa
    command: /home/deploy/.local/bin/lexa
    args: ["serve"]
    cwd: /home/deploy/commit-reviewer-src

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

otel:
  enabled: true
  endpoint: localhost:4317
  service_name: agentkit-code-reviewer
```

## How It Differs From Existing Tools

| Feature | Ptahcortex | LangChain | CrewAI | AutoGen |
|---|---|---|---|---|
| Language | Go | Python | Python | Python |
| MCP native | ✅ | ❌ (plugins) | ❌ | ❌ |
| Single binary | ✅ | ❌ | ❌ | ❌ |
| OTel built-in | ✅ | Partial | ❌ | ❌ |
| Config-driven | ✅ | Code-heavy | Code-heavy | Code-heavy |
| Production ready | ✅ | 🟡 | 🟡 | 🟡 |
| Tool calling | MCP protocol | Custom | Custom | Custom |

## Integration with Existing Projects

Ptahcortex reuses and connects your existing work:

| Project | Relationship |
|---|---|
| **Code Agent Proxy** | LLM backend — Ptahcortex calls it for model inference |
| **Commit Reviewer** | First example agent — code review workflow |
| **Lexa MCP** | Primary MCP server — code intelligence tools |
| **OTel Grafana Demo** | Observability — traces flow to Grafana/Tempo |
| **OSS Finder** | Potential second agent — scan + score repos |

## Demo Agents

### 1. Code Reviewer (Primary)
- Input: GitHub PR webhook or manual trigger
- Tools: Lexa (code search, symbols, references)
- Flow: Analyze diff → find related code → assess impact → post review
- Already partially built in commit-reviewer

### 2. Document Q&A
- Input: Question about a codebase or docs
- Tools: Filesystem MCP (read, search, list)
- Flow: Parse question → search files → read relevant → answer with citations

### 3. Task Planner
- Input: Complex task description
- Tools: Any connected MCP servers
- Flow: Decompose → plan steps → execute step-by-step → reflect on results

## Development Phases

### Phase 1: Foundation (Week 1-2)
- [ ] Go module setup
- [ ] MCP client (stdio JSON-RPC) — reuse from commit-reviewer
- [ ] LLM provider interface + OpenAI-compatible implementation
- [ ] Basic agent loop (single iteration, no reflection)
- [ ] YAML config loading

### Phase 2: Tool Calling (Week 2-3)
- [ ] Tool execution engine (parallel + sequential)
- [ ] Retry with exponential backoff
- [ ] Timeout management (per-tool + per-iteration)
- [ ] Tool result parsing and error handling
- [ ] MCP multi-server manager

### Phase 3: Agent Intelligence (Week 3-4)
- [ ] Planner — LLM-based task decomposition
- [ ] Reflector — result evaluation + loop control
- [ ] Max iterations + token budget enforcement
- [ ] Context window management (truncate old tool results)

### Phase 4: Observability (Week 4)
- [ ] OTel traces: agent iteration → tool call → LLM latency
- [ ] Metrics: tokens used, tools called, iterations, errors
- [ ] Structured logging with trace correlation
- [ ] Example Grafana dashboard

### Phase 5: Examples + Docs (Week 5)
- [ ] Code Reviewer example (end-to-end)
- [ ] Document Q&A example
- [ ] Task Planner example
- [ ] Full documentation (architecture, MCP guide, examples)

### Phase 6: Production Hardening (Week 6)
- [ ] Graceful shutdown
- [ ] Health check endpoint
- [ ] Rate limiting (per-agent, per-LLM)
- [ ] Cost tracking (token → dollar estimation)
- [ ] CLI with run/config/health subcommands

## Success Criteria

1. **Works end-to-end** — Code Reviewer example reviews a real PR using Lexa
2. **MCP-native** — can connect to any stdio MCP server via config
3. **Observable** — every agent run produces traces visible in Grafana
4. **Production patterns** — retry, timeout, cost, rate limiting
5. **Portfolio-worthy** — clean code, good docs, deployable

## Target Role Alignment

This project directly demonstrates skills for **Senior AI Platform Engineer**:

- Agent orchestration patterns
- MCP protocol implementation
- LLM infrastructure (routing, fallback, cost)
- Observability (OTel traces, metrics)
- Production Go services
- Config-driven deployment
