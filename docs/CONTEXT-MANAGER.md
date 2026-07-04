# Context Manager Design

## The Problem

Every AI agent framework today has the same flaw: **context grows linearly with tool calls**.

```
Iteration 1: 1K tokens  (task + first tool result)
Iteration 2: 4K tokens  (task + both tool results)
Iteration 3: 9K tokens  (task + all three tool results)
Iteration 4: 16K tokens (task + all four tool results)
...
Iteration N: N² tokens  (quadratic growth)
```

This isn't just wasteful — it hits context window limits, forces truncation (losing information), and makes multi-step agents impractical for production.

### Why Current "Fixes" Don't Work

| Fix | What It Does | Why It Fails |
|-----|-------------|--------------|
| **User compaction** | Manually start new session | Breaks automation, requires human |
| **New session** | Reset context | Loses all history, agent forgets |
| **Truncation** | Cut old messages | Loses important context randomly |
| **Sliding window** | Keep last N messages | Drops relevant old context |
| **Summarization** | LLM summarizes old context | Costs tokens to save tokens, lossy |

All of these treat the symptom, not the cause.

### The Root Cause

**All context goes into one LLM call.** The agent loop sends everything — task, tools, all tool results, all history — to a single LLM call every iteration.

```
Current:  Everything → LLM → Response
```

## The Solution: Sandboxed Tool Reasoning

**Separate high-level reasoning from tool-level reasoning.** Tool calls happen in isolated, cheap LLM calls with minimal context. Only summaries flow back to the main loop.

```
Proposed:  Task + Tools → Main Loop
                ↓
           Sub-task → Tool Sandbox (isolated) → Summary
                ↓
           Main Loop ← Summary
```

## Architecture

```
┌──────────────────────────────────────────────────────┐
│                   Main Agent Loop                     │
│              (rich context, expensive)                 │
│                                                      │
│  ┌─────────┐   ┌─────────┐   ┌─────────┐           │
│  │  Plan   │──→│ Collect │──→│ Reflect │──→ Final   │
│  └────┬────┘   └────↑────┘   └─────────┘           │
│       │              │                               │
│       │    ┌─────────┴─────────┐                    │
│       │    │   Sandbox Results │                    │
│       │    │   (summaries only)│                    │
│       │    └─────────┬─────────┘                    │
│       │              │                               │
│  ┌────┴──────────────┴────────────────────────┐     │
│  │            Tool Sandbox Layer               │     │
│  │         (isolated, cheap LLM calls)         │     │
│  │                                             │     │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  │     │
│  │  │ Sandbox  │  │ Sandbox  │  │ Sandbox  │  │     │
│  │  │  Call 1  │  │  Call 2  │  │  Call 3  │  │     │
│  │  │          │  │          │  │          │  │     │
│  │  │ select→  │  │ select→  │  │ select→  │  │     │
│  │  │ call→    │  │ call→    │  │ call→    │  │     │
│  │  │ eval     │  │ eval     │  │ eval     │  │     │
│  │  └──────────┘  └──────────┘  └──────────┘  │     │
│  └─────────────────────────────────────────────┘     │
└──────────────────────────────────────────────────────┘
```

## Context Tiers

Not all context is equal. Different call types need different context:

| Tier | Content | When Included |
|------|---------|---------------|
| **T0** | System prompt + tool definitions | Always |
| **T1** | Original user task | Always |
| **T2** | Current iteration's tool results | When evaluating current work |
| **T3** | Previous iterations' summaries | Only for reflection/final answer |
| **T4** | Full raw history | Never (replaced by summaries) |

## Call Types

### 1. Plan Call
**When:** Start of each iteration
**Context:** T0 + T1 + T3 (summaries of previous iterations)
**Purpose:** Decide what to do next based on task + what's been learned

```
System: You are a code reviewer. [tools listed]
User: Review the error handling in main.go
Assistant: Previous findings: [summary of iteration 1, 2]
Assistant: I should search for error patterns and check the MCP client.
```

### 2. Sandbox Call — Select
**When:** Each sub-task from the plan
**Context:** T0 (minimal — just tool descriptions) + sub-task
**Purpose:** Choose which tool to call and with what arguments

```
System: You have these tools: [tool list only]
User: Find error handling in internal/mcp/client.go
Assistant: I'll use text_search with query "error" scoped to that file.
```

### 3. Sandbox Call — Evaluate
**When:** After tool returns a result
**Context:** Sub-task + tool result (truncated)
**Purpose:** Understand what the tool result means, extract key info

```
System: Summarize the tool result for the main agent.
User: Task: Find error handling. Tool result: [truncated output]
Assistant: The MCP client uses structured JSON-RPC errors. Key functions:
request() returns *jsonrpcResponse with Error field. Errors propagate
through CallTool() → Manager. No retry at client level.
```

### 4. Reflect Call
**When:** After all sandbox calls complete
**Context:** T0 + T1 + T2 (sandbox summaries) + T3 (previous summaries)
**Purpose:** Evaluate progress, decide if more work needed

```
System: You are a code reviewer.
User: Review error handling in main.go
Assistant: [sandbox summaries from this iteration]
Assistant: I've found the error handling patterns. The review is complete.
I can provide my final assessment.
```

### 5. Final Call
**When:** Agent decides it's done
**Context:** T0 + T1 + all sandbox summaries (this + previous iterations)
**Purpose:** Produce the final answer for the user

```
System: You are a code reviewer.
User: Review error handling in main.go
Assistant: [all relevant summaries]
Assistant: Here's my review of the error handling...
```

## Token Comparison

### Current Approach (8 iterations, 5 tool calls each)

```
Iter 1: system(500) + task(200) + tools(1000) + results(3000)     = 4,700
Iter 2: system(500) + task(200) + tools(1000) + prev(3000) + results(3000) = 7,700
Iter 3: system(500) + task(200) + tools(1000) + prev(6000) + results(3000) = 10,700
...
Iter 8: system(500) + task(200) + tools(1000) + prev(21000) + results(3000) = 25,700

Total: ~120,000 tokens
```

### Sandbox Approach (same work)

```
Main Plan:    system(500) + task(200) + tools(1000) + summaries(500) = 2,200
Sandbox 1:    tools(500) + subtask(100) + result(500) = 1,100
Sandbox 2:    tools(500) + subtask(100) + result(500) = 1,100
Sandbox 3:    tools(500) + subtask(100) + result(500) = 1,100
Sandbox 4:    tools(500) + subtask(100) + result(500) = 1,100
Sandbox 5:    tools(500) + subtask(100) + result(500) = 1,100
Main Reflect: system(500) + task(200) + summaries(2500) = 3,200
Main Final:   system(500) + task(200) + all_summaries(5000) = 5,700

Total: ~17,000 tokens per iteration × 8 = ~136,000 tokens
```

Wait — that's actually more? Let me recalculate. The sandbox approach has more calls but each is cheaper. The key difference is **sandbox calls don't grow** — they're always ~1,100 tokens regardless of iteration count.

### Corrected: 20 iterations, 3 tool calls each

**Current:**
```
Total = sum(1700 + 3000*i) for i in 1..20 = ~650,000 tokens
```

**Sandbox:**
```
Main calls: 20 × 2,200 = 44,000
Sandbox calls: 60 × 1,100 = 66,000
Reflect calls: 20 × 3,200 = 64,000
Final: 1 × 5,700 = 5,700
Total: ~180,000 tokens
```

**Savings: ~72% at 20 iterations.** The savings grow with more iterations.

## Implementation Plan

### Phase 1: Context Manager (Current Sprint)
- [ ] Define CallType enum (plan, sandbox_select, sandbox_eval, reflect, final)
- [ ] Implement message assembler per call type
- [ ] Add sandbox executor (isolated LLM call with minimal context)
- [ ] Summary extraction from sandbox results
- [ ] Wire into agent loop

### Phase 2: Token Tracking
- [ ] Track tokens per call type
- [ ] Compare against non-sandboxed baseline
- [ ] Log token savings per run

### Phase 3: Adaptive Sandbox
- [ ] Sandbox decides if it needs more tools (multi-tool sandbox)
- [ ] Sandbox can escalate to main loop ("I need more context")
- [ ] Main loop can request deeper sandbox analysis

## Configuration

```yaml
context:
  # Sandbox settings
  sandbox:
    enabled: true
    max_tool_result_len: 4000    # chars per tool result in sandbox
    max_sandbox_iterations: 3    # max tools per sandbox call

  # Main loop settings
  main:
    max_messages: 20             # max messages in main loop
    keep_first_user: true        # always include original task
    summary_tokens: 500          # max tokens per sandbox summary

  # Token budget
  budget:
    max_per_run: 50000           # total token limit
    warn_at_percent: 80          # warn at 80%
    sandbox_max_per_call: 1000   # max tokens per sandbox LLM call
```

## Key Design Decisions

1. **Summaries replace raw results** — Main loop never sees raw tool output, only sandbox summaries
2. **Sandbox is stateless** — Each sandbox call is independent, no shared state between sandboxes
3. **Main loop is stateful** — Keeps task + summaries across iterations
4. **Sandbox can be skipped** — Simple tasks don't need sandboxing (configurable)
5. **Sandbox calls are parallel** — Multiple sandboxes can run concurrently

## Open Questions

1. ~~Should sandbox use a cheaper/faster model than main loop?~~ → **Yes, see Local LLM Selector below**
2. Should sandbox results be cached? (same sub-task + same tools = same result)
3. How to handle sandbox failures? (fallback to raw result in main loop?)
4. Should the agent learn which tasks need sandboxing vs direct calls?

## Local LLM Context Selector

### Insight

The last piece of the context selection puzzle: **use a local LLM to decide what context is relevant** before sending to the main (cloud) LLM.

Local LLMs (Ollama, llama.cpp, vLLM) have smaller context windows:
- 32K tokens (most models)
- 64K tokens (some models)
- 128K tokens (top-end local models)

This constraint is actually **a feature**, not a bug. It forces precise context selection.

### Architecture

```
User Task + Available Context
            │
            ▼
┌───────────────────────┐
│  Local LLM Selector   │ ← Small, fast, cheap (runs locally)
│  (32K-128K context)   │
│                       │
│  "Which messages are  │
│   relevant for this   │
│   specific LLM call?" │
└───────────┬───────────┘
            │
            ▼ (selected subset)
┌───────────────────────┐
│  Main LLM Call        │ ← Large, expensive (cloud API)
│  (only relevant ctx)  │
└───────────────────────┘
```

### How It Works

1. **Candidate assembly** — gather all potentially relevant messages (task, tool results, summaries)
2. **Selection prompt** — ask local LLM: "Given this task, which messages are relevant?"
3. **Filter** — local LLM returns indices/labels of relevant messages
4. **Assembly** — build final context from selected messages only
5. **Main call** — send filtered context to cloud LLM

### Selection Prompt

```
You are a context selector. Given a task and a list of messages,
return ONLY the message indices that are relevant to the task.

Task: {task}

Messages:
[0] System prompt
[1] User: Review error handling in main.go
[2] Tool result: Found 3 error patterns in client.go
[3] Tool result: File listing of /src
[4] Summary: Error handling uses structured errors with retry
[5] Tool result: Search results for "TODO" comments

Relevant indices (comma-separated): 0, 1, 2, 4
```

Local LLM returns: `0, 1, 2, 4`
Main LLM only sees those 4 messages instead of all 6.

### Why Local LLM?

| Aspect | Cloud LLM | Local LLM |
|--------|-----------|----------|
| Cost | $0.01-0.03 per call | Free (after hardware cost) |
| Latency | 200-500ms | 50-200ms (small model) |
| Context | 128K-1M | 32K-128K |
| Quality | High | Good enough for selection |
| Privacy | Data sent to cloud | Stays local |

The local LLM doesn't need to be smart. It needs to be **fast and precise** at filtering.

### Model Choices for Selection

| Model | Size | Context | Speed | Use Case |
|-------|------|---------|-------|----------|
| Phi-3 Mini | 3.8B | 128K | Fast | Best for selection (large context, small model) |
| Qwen2.5 3B | 3B | 32K | Very fast | Lightweight selection |
| Llama 3.2 3B | 3B | 128K | Fast | Good balance |
| Gemma 2 2B | 2B | 8K | Fastest | Simple filtering only |

### Configuration

```yaml
context:
  selector:
    enabled: true
    provider: ollama              # local LLM provider
    model: phi3-mini              # small model for selection
    base_url: http://localhost:11434
    max_input_tokens: 8000        # max tokens to send to selector
    strategy: relevance           # relevance | dependency | both
```

### Cost Impact

Without local selector:
- 20 iterations × cloud LLM = 20 API calls for context selection
- Cost: ~$0.20-0.60 per agent run

With local selector:
- 20 iterations × local LLM (free) + 20 cloud LLM (filtered)
- Cost: ~$0.06-0.18 per agent run (70% cost reduction)

### The Full Pipeline

```
┌──────────────────────────────────────────────────┐
│                 Agent Run                         │
│                                                  │
│  ┌──────────┐                                    │
│  │  Plan    │ ← Context: T0+T1+T3               │
│  └────┬─────┘                                    │
│       │                                          │
│       ▼                                          │
│  ┌──────────────────┐                            │
│  │ Local LLM Select │ ← "Which tools for this?" │
│  └────┬─────────────┘                            │
│       │                                          │
│       ▼                                          │
│  ┌──────────────────┐                            │
│  │ Tool Sandbox     │ ← Isolated, minimal ctx    │
│  └────┬─────────────┘                            │
│       │                                          │
│       ▼                                          │
│  ┌──────────────────┐                            │
│  │ Local LLM Filter │ ← "Which results matter?" │
│  └────┬─────────────┘                            │
│       │                                          │
│       ▼                                          │
│  ┌──────────┐                                    │
│  │ Reflect  │ ← Context: T0+T1+filtered T2      │
│  └────┬─────┘                                    │
│       │                                          │
│       ▼                                          │
│  ┌──────────┐                                    │
│  │ Final    │ ← Context: T0+T1+all summaries    │
│  └──────────┘                                    │
└──────────────────────────────────────────────────┘
```

### Summary

The local LLM acts as a **gatekeeper** between raw context and the cloud LLM:
- Selects what's relevant before the main call
- Runs for free (local) vs $0.01-0.03 per call (cloud)
- Forces precise context selection (small window = precision)
- Adds ~100ms latency (acceptable for the cost savings)

This completes the context engineering stack:
1. **Call-type-aware assembly** (different recipes per call)
2. **Tool sandboxing** (isolated tool reasoning)
3. **Local LLM selection** (relevance filtering before main call)
4. **Summary flow** (only summaries flow up)
