# Context Manager — Task Breakdown

## Overview

Four phases, each independently useful, each builds on the previous.

**Total estimated effort:** 8-12 days
**Start:** After current sprint
**Goal:** Reduce agent token usage by ~72-90%

---

## Phase 1: Call-Aware Assembly

**Effort:** 2-3 days
**Savings:** ~30% token reduction
**Risk:** None (pure code, no quality dependency)
**Confidence:** HIGH

### Tasks

| # | Task | Hours | Status |
|---|------|-------|--------|
| 1.1 | Define `CallType` enum (plan, eval, reflect, final) | 1h | ⬜ |
| 1.2 | Implement `ContextAssembler` — builds message list per call type | 3h | ⬜ |
| 1.3 | Define tier rules (T0-T4 inclusion per call type) | 1h | ⬜ |
| 1.4 | Wire `ContextAssembler` into agent loop | 2h | ⬜ |
| 1.5 | Add token counting per call type | 2h | ⬜ |
| 1.6 | Benchmark: run same task with/without assembly | 2h | ⬜ |
| 1.7 | Unit tests for assembler | 2h | ⬜ |

### Deliverables

```
internal/agent/
├── context.go          # ContextAssembler + CallType enum
├── assembler.go        # Message assembly per call type
└── context_test.go     # Unit tests
```

### Definition of Done

- [ ] CallType enum defined
- [ ] ContextAssembler returns different message sets per call type
- [ ] Agent loop uses ContextAssembler for all LLM calls
- [ ] Token usage tracked per call type
- [ ] Benchmark shows measurable savings
- [ ] Unit tests pass

### Benchmark Criteria

Run 5 tasks, compare:
- Total tokens without assembly
- Total tokens with assembly
- Savings percentage

Target: ≥20% token reduction

---

## Phase 2: Tool Sandboxing

**Effort:** 3-4 days
**Savings:** ~50% additional token reduction
**Risk:** Medium (quality is empirical)
**Confidence:** MEDIUM
**Depends on:** Phase 1

### Tasks

| # | Task | Hours | Status |
|---|------|-------|--------|
| 2.1 | Design `Sandbox` struct — isolated LLM call | 2h | ⬜ |
| 2.2 | Implement `SandboxResult` — summary + tools_used + confidence | 1h | ⬜ |
| 2.3 | Implement sandbox executor (select → call → eval) | 4h | ⬜ |
| 2.4 | Add parallel sandbox execution with semaphore | 2h | ⬜ |
| 2.5 | Implement summary extraction from sandbox output | 2h | ⬜ |
| 2.6 | Wire sandbox into agent loop (plan → sandbox → collect → reflect) | 3h | ⬜ |
| 2.7 | Benchmark: compare sandboxed vs direct tool calling | 2h | ⬜ |
| 2.8 | Unit tests for sandbox | 2h | ⬜ |

### Deliverables

```
internal/agent/
├── sandbox.go          # Sandbox executor
├── sandbox_result.go   # SandboxResult struct
├── summarizer.go       # Summary extraction
└── sandbox_test.go     # Unit tests
```

### Definition of Done

- [ ] Sandbox struct implemented
- [ ] Sandbox makes isolated LLM calls with minimal context
- [ ] Sandbox returns summaries (not raw results)
- [ ] Parallel sandbox execution works
- [ ] Agent loop uses sandbox for tool calls
- [ ] Benchmark shows measurable savings

### Benchmark Criteria

Run 10 tasks, compare:
- Tool selection accuracy (sandboxed vs direct)
- Token usage (sandboxed vs direct)
- Output quality (human evaluation)

Target: ≥40% token reduction, ≥80% tool selection accuracy

### Fallback

If sandbox quality is poor:
- Skip sandboxing
- Use call-aware assembly only (Phase 1, 30% savings)
- Document why sandboxing didn't work

---

## Phase 3: Summary Flow

**Effort:** 1-2 days
**Savings:** ~20% additional token reduction
**Risk:** Low-Medium (prompt-dependent)
**Confidence:** MEDIUM
**Depends on:** Phase 2

### Tasks

| # | Task | Hours | Status |
|---|------|-------|--------|
| 3.1 | Design summary prompt template | 1h | ⬜ |
| 3.2 | Implement `SummaryExtractor` — LLM-based summarization | 2h | ⬜ |
| 3.3 | Add summary storage per iteration | 1h | ⬜ |
| 3.4 | Wire summaries into plan/reflect calls | 2h | ⬜ |
| 3.5 | Configurable summary length (max_tokens) | 1h | ⬜ |
| 3.6 | Benchmark: compare full results vs summaries | 1h | ⬜ |

### Deliverables

```
internal/agent/
├── summarizer.go       # SummaryExtractor
├── summary_store.go    # Summary storage per iteration
└── summarizer_test.go  # Unit tests
```

### Definition of Done

- [ ] SummaryExtractor produces concise summaries
- [ ] Summaries stored per iteration
- [ ] Plan/reflect calls use summaries instead of raw results
- [ ] Summary length configurable
- [ ] Benchmark shows measurable savings

### Benchmark Criteria

Run 5 tasks, compare:
- Agent output quality (full results vs summaries)
- Token usage (full results vs summaries)

Target: ≥15% token reduction, ≥90% output quality retention

### Fallback

If summaries are too lossy:
- Increase max summary length
- Skip summarization for critical results
- Use hybrid: summarize non-critical, keep raw for critical

---

## Phase 4: Local LLM Selection

**Effort:** 2-3 days
**Savings:** ~70% additional cost reduction
**Risk:** High (model-dependent, hardware-dependent)
**Confidence:** LOW
**Depends on:** Phase 1 (can run independently of Phases 2-3)

### Tasks

| # | Task | Hours | Status |
|---|------|-------|--------|
| 4.1 | Implement `LocalSelector` — Ollama/vLLM client | 3h | ⬜ |
| 4.2 | Design selection prompt template | 1h | ⬜ |
| 4.3 | Implement message filtering by indices | 1h | ⬜ |
| 4.4 | Add config (model, base_url, max_input_tokens) | 1h | ⬜ |
| 4.5 | Add fallback (if local LLM unavailable, skip) | 1h | ⬜ |
| 4.6 | Benchmark: selection accuracy on 20 messages | 2h | ⬜ |
| 4.7 | Benchmark: latency on target hardware | 1h | ⬜ |

### Deliverables

```
internal/agent/
├── local_selector.go   # LocalSelector client
├── selection_prompt.go # Prompt template
└── local_selector_test.go
```

### Definition of Done

- [ ] LocalSelector connects to Ollama/vLLM
- [ ] Selection prompt works (returns valid indices)
- [ ] Message filtering by indices works
- [ ] Fallback works (graceful degradation)
- [ ] Benchmark shows measurable cost savings

### Benchmark Criteria

Run 20 messages with known relevance, compare:
- Selection accuracy (correct/total)
- Latency per selection
- Total cost with/without selector

Target: ≥80% selection accuracy, ≤200ms latency

### Fallback

If local LLM is unavailable or poor quality:
- Skip Phase 4 entirely
- Phases 1-3 still give 72% savings
- Document limitations

---

## Phase Dependencies

```
Phase 1 (Assembly) ──→ Phase 2 (Sandbox) ──→ Phase 3 (Summaries)
      │
      └──────────────→ Phase 4 (Local Selector)
```

- Phase 1 is the foundation (no dependencies)
- Phase 2 depends on Phase 1 (uses call-aware assembly)
- Phase 3 depends on Phase 2 (uses sandbox for summarization)
- Phase 4 depends on Phase 1 only (can run independently)

## Decision Points

| After Phase | Decision | Criteria |
|-------------|----------|----------|
| 1 | Continue to Phase 2? | If Phase 1 savings ≥ 20% |
| 2 | Continue to Phase 3? | If sandbox quality ≥ 80% tool accuracy |
| 2 | Skip to Phase 4? | If sandbox quality poor but assembly works |
| 3 | Continue to Phase 4? | If summaries retain ≥ 90% quality |
| 3 | Stop? | If savings are sufficient (72%) |
| 4 | Done? | If local selector accuracy ≥ 80% |

## Total Timeline

| Phase | Duration | Cumulative | Savings |
|-------|----------|------------|---------|
| Phase 1 | 2-3 days | 2-3 days | 30% |
| Phase 2 | 3-4 days | 5-7 days | 65% |
| Phase 3 | 1-2 days | 6-9 days | 72% |
| Phase 4 | 2-3 days | 8-12 days | ~90% |

**Conservative estimate:** 8 days (stop after Phase 3)
**Optimistic estimate:** 12 days (all phases)
**Minimum viable:** 3 days (Phase 1 only)
