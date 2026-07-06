# Ptahcortex — Development Roadmap

## Timeline: July 3 — August 14, 2026 (6 weeks)

**Goal:** Production-ready AI agent framework with MCP tool calling, deployed as portfolio project.

---

## Phase 1: Foundation + Context Manager (Week 1 — Jul 3-10)

**Objective:** Core MCP client + LLM provider + agent loop + Context Manager

| Task | Hours | Status |
|------|-------|--------|
| Go module setup + project structure | 1h | ✅ |
| MCP client (stdio JSON-RPC) — reuse commit-reviewer code | 3h | ✅ |
| LLM provider interface + OpenAI implementation | 3h | ✅ |
| Basic agent loop (single plan→execute→reflect) | 4h | ✅ |
| YAML config loading | 2h | ✅ |
| Unit tests for MCP client | 2h | ✅ |
| **Context Manager (call-type-aware assembly)** | 8h | ✅ |
| **Sandboxed tool reasoning** | 6h | ✅ |
| **Token tracking per call type** | 2h | ✅ |
| **Observability for context operations** | 3h | ✅ |
| **Benchmark suite** | 4h | ✅ |

**Deliverable:** ✅ Agent connects to Lexa (22 tools), runs with GLM-5.1/5.2, produces quality output.

**Actual Hours:** ~38h

---

## Phase 2: Tool Calling (Week 2 — Jul 10-17)

**Objective:** Robust tool execution engine

| Task | Hours | Status |
|------|-------|--------|
| Tool execution engine (parallel + sequential) | 4h | ✅ |
| Dependency detection between tool calls | 2h | ✅ |
| Retry with exponential backoff | 2h | ✅ |
| Timeout management (per-tool + per-iteration) | 2h | ✅ |
| Tool result truncation | 1h | ✅ |
| MCP multi-server manager | 3h | ✅ |
| Anthropic LLM provider | 2h | ✅ |

**Deliverable:** ✅ Agent can call multiple tools in parallel, handle errors, retry.

**Status:** All tasks completed in Phase 1.

---

## Phase 3: Agent Intelligence (Week 3 — Jul 17-24)

**Objective:** Multi-iteration agent with planning and reflection

| Task | Hours | Status |
|------|-------|--------|
| Planner — LLM-based task decomposition | 3h | ✅ |
| Reflector — result evaluation + loop control | 3h | ✅ |
| Max iterations + token budget enforcement | 2h | ✅ |
| Context window management (truncate old results) | 2h | ✅ |
| Agent state machine (idle→planning→executing→reflecting→done) | 3h | ✅ |
| Code Agent Proxy LLM provider | 2h | ✅ |

**Deliverable:** ✅ Agent runs multi-step workflows, reflects on results, stops when done.

**Status:** All tasks completed in Phase 1.

---

## Phase 4: Observability (Week 4 — Jul 24-31)

**Objective:** Full OTel integration

| Task | Hours | Status |
|------|-------|--------|
| OTel SDK setup (tracer + meter provider) | 2h | ✅ |
| Spans: agent.run, agent.iteration, agent.plan, agent.execute, agent.reflect | 3h | ✅ |
| Spans: mcp.call_tool, llm.chat | 2h | ✅ |
| Metrics: iterations, tool calls, tokens, latency | 2h | ✅ |
| Structured logging with trace correlation | 2h | ✅ |
| Example Grafana dashboard JSON | 2h | ⬜ |
| Integration with OTel Grafana Demo stack | 1h | ⬜ |

**Deliverable:** ✅ Every agent run produces traces visible in logs. Grafana dashboard pending.

**Status:** Core observability complete. Dashboard integration pending.

---

## Phase 5: Examples + Docs (Week 5 — Jul 31 — Aug 7)

**Objective:** Working examples and complete documentation

| Task | Hours | Status |
|------|-------|--------|
| Code Reviewer example (end-to-end with Lexa) | 4h | ✅ |
| Document Q&A example (filesystem MCP) | 3h | ⬜ |
| Task Planner example (multi-step reasoning) | 3h | ⬜ |
| README update with real usage examples | 2h | ✅ |
| API reference docs | 2h | ⬜ |
| Blog post: "Context Engineering for AI Agents" | 4h | ⬜ |

**Deliverable:** One working example (code-reviewer), README updated. More examples pending.

---

## Phase 6: Production Hardening (Week 6 — Aug 7-14)

**Objective:** Deploy-ready with production patterns

| Task | Hours | Status |
|------|-------|--------|
| Graceful shutdown (SIGTERM handling) | 1h | ⬜ |
| Health check endpoint (/health, /ready) | 1h | ⬜ |
| Rate limiting (per-agent, per-LLM) | 2h | ⬜ |
| Cost tracking (token → dollar estimation) | 2h | ⬜ |
| CLI with run/config/health subcommands | 2h | ⬜ |
| Dockerfile + docker-compose | 2h | ⬜ |
| Deploy to VPS + integrate with existing stack | 2h | ✅ |
| Portfolio case study page | 2h | ⬜ |

**Deliverable:** Deployed on VPS at /usr/local/bin/ptahcortex. Case study pending.

---

## Progress Summary

| Phase | Planned | Completed | Status |
|-------|---------|-----------|--------|
| Phase 1: Foundation | 15h | 38h | ✅ **Exceeded** |
| Phase 2: Tool Calling | 16h | 0h (done in P1) | ✅ **Merged into P1** |
| Phase 3: Agent Intelligence | 15h | 0h (done in P1) | ✅ **Merged into P1** |
| Phase 4: Observability | 14h | 11h | 🟡 80% |
| Phase 5: Examples + Docs | 18h | 6h | 🟡 33% |
| Phase 6: Production | 12h | 2h | 🟡 17% |
| **Total** | **90h** | **57h** | **63%** |

---

## Key Achievements

### 1. Context Manager (Differentiator)
- Call-type-aware context assembly (plan, sandbox_select, sandbox_eval, reflect, final)
- Sandboxed tool reasoning with minimal context per tool call
- **~60% token savings** vs naive approaches
- Full observability per call type

### 2. Benchmark Results (GLM-5.1)
- 14,353 total tokens for code review task
- 7 detailed findings with specific code locations
- Quality comparable to larger models

### 3. Model Comparison
- **GLM-5.1:** Better quality, more detailed output
- **GLM-5.2:** 18% fewer tokens, 45% faster

### 4. Production Deployment
- Binary at `/usr/local/bin/ptahcortex` on VPS
- Lexa MCP connected (22 tools)
- Working with ai.sumopod.com API

---

## Updated Timeline

Given the accelerated progress, the project is **2-3 weeks ahead of schedule**.

| Original | Revised |
|----------|---------|
| Phase 1-3: Weeks 1-3 | ✅ Done in Week 1 |
| Phase 4: Week 4 | 🟡 80% done |
| Phase 5: Week 5 | 🟡 33% done |
| Phase 6: Week 6 | 🟡 17% done |

**New Target:** Complete all phases by **July 24** (3 weeks early).

---

## Next Steps

1. **Week 2 (Jul 10-17):** Complete Phase 5 (examples + docs)
2. **Week 3 (Jul 17-24):** Complete Phase 6 (production hardening) + blog post
3. **Week 4 (Jul 24-31):** Portfolio case study + final polish

---

## Success Metrics

1. ✅ Code Reviewer example reviews code using Lexa
2. ✅ Any stdio MCP server works via config
3. ✅ Traces visible in logs for every agent run
4. ✅ Clean code, good docs, deployable binary
5. ⬜ Blog post published
6. ⬜ Case study on portfolio site
