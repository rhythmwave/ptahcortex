# Context Manager — Validation Assessment

## Purpose

This document provides an honest assessment of each component's feasibility, confidence level, and risks before implementation.

## Component Validation

### Phase 1: Call-Aware Assembly

| Aspect | Assessment |
|--------|-----------|
| Architecture | ✅ Simple — filter messages by call type, build different slices |
| Implementation | ✅ Pure Go code, no AI quality dependency |
| Quality dependency | ✅ None — it's data filtering, not model behavior |
| Risk | ✅ None |
| Confidence | **HIGH** |

**Why it works:** We're not changing what the LLM receives — we're choosing what NOT to send. If a planning call doesn't need tool results from 3 iterations ago, we simply don't include them. The LLM sees less noise, makes better decisions.

**Validation method:** Run same task with/without assembly, compare token counts and output quality.

---

### Phase 2: Tool Sandboxing

| Aspect | Assessment |
|--------|-----------|
| Architecture | ✅ Sound — isolated LLM calls with minimal context |
| Implementation | ✅ Mechanically simple — another LLM call with filtered context |
| Quality dependency | 🟡 Depends on sandbox LLM making good selections with minimal context |
| Risk | 🟡 Medium — quality is empirical, not guaranteed |
| Confidence | **MEDIUM** |

**Why it should work:** Tool selection with minimal context is a well-scoped task. "Find error handling in main.go" + tool list should be enough for the LLM to pick `text_search` or `read`. The sandbox doesn't need full history — it needs the sub-task and available tools.

**Uncertainty:** Can a sandbox LLM reliably make tool selections with only the sub-task + tool descriptions? In theory yes, but needs testing.

**Fallback:** If sandbox quality is poor, skip it. Call-aware assembly alone still gives ~30% savings.

**Validation method:** Compare sandboxed vs direct tool calling on 10 different tasks. Measure: tool selection accuracy, result quality, token usage.

---

### Phase 3: Summary Flow

| Aspect | Assessment |
|--------|-----------|
| Architecture | ✅ Proven — LLM summarization is widely used |
| Implementation | ✅ Straightforward — summarize tool results, store summaries |
| Quality dependency | 🟡 Prompt-dependent — summaries may lose important details |
| Risk | 🟡 Low-Medium — summarization quality varies |
| Confidence | **MEDIUM** |

**Why it works:** Summarization is a solved problem. We're just moving WHERE it happens — per-tool-result instead of at the end. The key is the summarization prompt.

**Uncertainty:** Do summaries retain enough information for the main loop to make good decisions? Depends on the task complexity and summary length.

**Fallback:** If summaries are too lossy, increase max summary length or skip summarization for critical results.

**Validation method:** Compare agent output quality with full results vs summaries on 5 different tasks.

---

### Phase 4: Local LLM Selection

| Aspect | Assessment |
|--------|-----------|
| Architecture | ✅ Sound — small model filters, large model executes |
| Implementation | ✅ Mechanically simple — Ollama API call + index filtering |
| Quality dependency | ⚠️ High — small models may not reliably judge relevance |
| Risk | ⚠️ High — hardware dependent, model quality varies |
| Confidence | **LOW** |

**Why it should work (in theory):** Context selection is a simpler task than generation. "Is this message relevant to this task?" is a yes/no question that small models should handle. The 32K-128K context constraint forces precise selection.

**Uncertainties:**
1. **Selection quality** — Can a 3B-4B model reliably judge relevance? Small models hallucinate, miss nuance, select wrong context.
2. **Latency** — 100ms per selection × 20 iterations = 2s added. CPU-bound inference could be 500ms+.
3. **Hardware** — Requires Ollama running. Not everyone has a GPU. CPU inference on 3B model = slow.
4. **Cost math** — "70% cost reduction" assumes selector is free. Time cost may outweigh dollar savings.

**Fallback:** If local LLM unavailable or poor quality, skip it entirely. Phases 1-3 still give 72% savings.

**Validation method:**
1. Test selection accuracy on 20 messages with known relevance
2. Measure latency on target hardware (CPU vs GPU)
3. Compare total cost with/without selector

---

## Savings Validation

### Claimed vs Proven

| Phase | Claimed Savings | Status | Basis |
|-------|----------------|--------|-------|
| 1. Call-Aware Assembly | 30% | 🟡 Theoretical | Removing messages = fewer tokens. Defensible but needs benchmarking. |
| 2. Tool Sandboxing | 50% | 🟡 Theoretical | Isolated calls = smaller context. Depends on sandbox quality. |
| 3. Summary Flow | 20% | 🟡 Theoretical | Summaries < raw results. Depends on summary quality. |
| 4. Local LLM Selection | 70% cost | ⚠️ Speculative | Free local model vs paid cloud. Depends on selector quality. |

**Combined 72% (Phases 1-3):** Plausible but needs empirical validation.
**Combined 90% (All phases):** Optimistic. Needs Phase 4 validation.

### Benchmark Plan

1. **Baseline:** Run 5 tasks without any optimization, record total tokens
2. **Phase 1:** Same tasks with call-aware assembly, record tokens
3. **Phase 2:** Same tasks with sandboxing, record tokens
4. **Phase 3:** Same tasks with summary flow, record tokens
5. **Compare:** Calculate actual savings per phase

---

## Risk Mitigation

| Risk | Mitigation |
|------|-----------|
| Sandbox quality poor | Fallback to call-aware assembly only (30% savings) |
| Summaries too lossy | Increase summary length or skip for critical results |
| Local LLM slow/bad | Skip Phase 4, Phases 1-3 still give 72% |
| Token math wrong | Benchmark early, adjust claims |
| Architecture too complex | Each phase is independently useful, stop at any phase |

---

## Decision Framework

**Start with Phase 1 because:**
- Zero risk (pure code, no quality dependency)
- Defensible savings (removing messages = fewer tokens)
- Foundation for all other phases
- Can benchmark immediately

**Decide on Phase 2 after Phase 1 benchmark:**
- If Phase 1 gives 30%+ savings → Phase 2 is worth pursuing
- If Phase 1 gives <15% savings → reassess approach

**Decide on Phase 3 after Phase 2 benchmark:**
- If sandbox quality is good → summaries will help
- If sandbox quality is poor → skip Phase 3

**Phase 4 is optional:**
- Only pursue if Phases 1-3 prove the concept
- Only if local LLM infrastructure is available
- Consider it a stretch goal, not a requirement

---

## Honest Summary

**What we know:**
- The architecture is sound
- Call-aware assembly will work (pure code)
- The concept of sandboxed tool reasoning is valid

**What we don't know:**
- Actual token savings (need benchmarks)
- Sandbox LLM quality with minimal context
- Summary quality and information loss
- Local LLM selection reliability

**What we should do:**
1. Implement Phase 1 (safe, no quality dependency)
2. Benchmark it
3. Decide Phase 2 based on results
4. Phase 4 is optional/stretch

**Bottom line:** The design is within reach. The architecture is valid. The savings need empirical validation. Phase 1 is the safest starting point.
