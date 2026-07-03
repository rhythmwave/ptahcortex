# Ptahcortex — Development Roadmap

## Timeline: July 3 — August 14, 2026 (6 weeks)

**Goal:** Production-ready AI agent framework with MCP tool calling, deployed as portfolio project.

---

## Phase 1: Foundation (Week 1 — Jul 3-10)

**Objective:** Core MCP client + LLM provider + basic agent loop

| Task | Hours | Status |
|------|-------|--------|
| Go module setup + project structure | 1h | ⬜ |
| MCP client (stdio JSON-RPC) — reuse commit-reviewer code | 3h | ⬜ |
| LLM provider interface + OpenAI implementation | 3h | ⬜ |
| Basic agent loop (single plan→execute→reflect) | 4h | ⬜ |
| YAML config loading | 2h | ⬜ |
| Unit tests for MCP client | 2h | ⬜ |

**Deliverable:** Agent can connect to Lexa, call one tool, get LLM response.

---

## Phase 2: Tool Calling (Week 2 — Jul 10-17)

**Objective:** Robust tool execution engine

| Task | Hours | Status |
|------|-------|--------|
| Tool execution engine (parallel + sequential) | 4h | ⬜ |
| Dependency detection between tool calls | 2h | ⬜ |
| Retry with exponential backoff | 2h | ⬜ |
| Timeout management (per-tool + per-iteration) | 2h | ⬜ |
| Tool result truncation | 1h | ⬜ |
| MCP multi-server manager | 3h | ⬜ |
| Anthropic LLM provider | 2h | ⬜ |

**Deliverable:** Agent can call multiple tools in parallel, handle errors, retry.

---

## Phase 3: Agent Intelligence (Week 3 — Jul 17-24)

**Objective:** Multi-iteration agent with planning and reflection

| Task | Hours | Status |
|------|-------|--------|
| Planner — LLM-based task decomposition | 3h | ⬜ |
| Reflector — result evaluation + loop control | 3h | ⬜ |
| Max iterations + token budget enforcement | 2h | ⬜ |
| Context window management (truncate old results) | 2h | ⬜ |
| Agent state machine (idle→planning→executing→reflecting→done) | 3h | ⬜ |
| Code Agent Proxy LLM provider | 2h | ⬜ |

**Deliverable:** Agent runs multi-step workflows, reflects on results, stops when done.

---

## Phase 4: Observability (Week 4 — Jul 24-31)

**Objective:** Full OTel integration

| Task | Hours | Status |
|------|-------|--------|
| OTel SDK setup (tracer + meter provider) | 2h | ⬜ |
| Spans: agent.run, agent.iteration, agent.plan, agent.execute, agent.reflect | 3h | ⬜ |
| Spans: mcp.call_tool, llm.chat | 2h | ⬜ |
| Metrics: iterations, tool calls, tokens, latency | 2h | ⬜ |
| Structured logging with trace correlation | 2h | ⬜ |
| Example Grafana dashboard JSON | 2h | ⬜ |
| Integration with OTel Grafana Demo stack | 1h | ⬜ |

**Deliverable:** Every agent run produces traces visible in Grafana/Tempo.

---

## Phase 5: Examples + Docs (Week 5 — Jul 31 — Aug 7)

**Objective:** Working examples and complete documentation

| Task | Hours | Status |
|------|-------|--------|
| Code Reviewer example (end-to-end with Lexa) | 4h | ⬜ |
| Document Q&A example (filesystem MCP) | 3h | ⬜ |
| Task Planner example (multi-step reasoning) | 3h | ⬜ |
| README update with real usage examples | 2h | ⬜ |
| API reference docs | 2h | ⬜ |
| Blog post: "Building AI Agents with MCP and Go" | 4h | ⬜ |

**Deliverable:** Three working examples, full docs, blog post draft.

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
| Deploy to VPS + integrate with existing stack | 2h | ⬜ |
| Portfolio case study page | 2h | ⬜ |

**Deliverable:** Deployed on VPS, case study on bayudn.pro, blog post published.

---

## Total Estimated Hours: ~120h

| Phase | Hours |
|-------|-------|
| Foundation | 15 |
| Tool Calling | 16 |
| Agent Intelligence | 15 |
| Observability | 14 |
| Examples + Docs | 18 |
| Production | 12 |
| **Total** | **~90-120h** |

---

## Dependencies

```
Phase 1 → Phase 2 → Phase 3 → Phase 4 → Phase 5 → Phase 6
                                    ↑
                          OTel Grafana Demo (exists)
```

## Risk Factors

| Risk | Impact | Mitigation |
|------|--------|------------|
| MCP protocol changes | Medium | Pin protocol version, abstract interface |
| LLM API instability | Low | Retry + fallback providers |
| Context window limits | Medium | Aggressive truncation + summarization |
| Scope creep | High | Phase-based delivery, stop at Phase 4 if needed |

## Success Metrics

1. ✅ Code Reviewer example reviews a real PR using Lexa
2. ✅ Any stdio MCP server works via config (no code changes)
3. ✅ Traces visible in Grafana for every agent run
4. ✅ Clean code, good docs, deployable binary
5. ✅ Blog post published, case study on portfolio
