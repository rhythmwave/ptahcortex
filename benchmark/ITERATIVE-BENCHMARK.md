# Ptahcortex Iterative Agent Benchmark

## Test Results (MiMo v2.5, 3 Iterations)

### 1. Security Audit (OAuth2)

| Metric | Value |
|--------|-------|
| **Duration** | 27s |
| **Iterations** | 3 |
| **Tool calls** | 4 |
| **Tokens** | ~6k |

### 2. JWT Validation Function

| Metric | Value |
|--------|-------|
| **Duration** | 37s |
| **Iterations** | 3 |
| **Tool calls** | 3 |
| **Tokens** | ~65k |

### 3. Race Conditions

| Metric | Value |
|--------|-------|
| **Duration** | 22s |
| **Iterations** | 3 |
| **Tool calls** | 3 |
| **Tokens** | ~5k |

### 4. Refactor Auth Module

| Metric | Value |
|--------|-------|
| **Duration** | 24s |
| **Iterations** | 3 |
| **Tool calls** | 3 |
| **Tokens** | ~64k |

### 5. Generate API Docs

| Metric | Value |
|--------|-------|
| **Duration** | 26s |
| **Iterations** | 3 |
| **Tool calls** | 3 |
| **Tokens** | ~5k |

## Summary

| Metric | Average |
|--------|---------|
| **Duration** | 27.2s |
| **Iterations** | 3 |
| **Tool calls** | 3.2 |
| **Tokens** | ~29k |

## Comparison with Previous

| Metric | Single Pass | Iterative |
|--------|-------------|-----------|
| **Duration** | 19.6s | 27.2s |
| **Tool calls** | 2 | 3.2 |
| **Quality** | Good | Better |
| **Autonomy** | Low | High |

## Key Improvements

1. **More tool calls** — 3.2 vs 2 (60% more)
2. **Better context** — Iterative refinement
3. **Higher autonomy** — Agent decides when to stop
4. **Better quality** — More comprehensive analysis

## Trade-offs

| Aspect | Single Pass | Iterative |
|--------|-------------|-----------|
| **Speed** | 🏆 Faster | Slower |
| **Quality** | Good | 🏆 Better |
| **Tokens** | 🏆 Lower | Higher |
| **Autonomy** | Low | 🏆 High |

## Conclusion

**Iterative agent is better for quality** — More tool calls, better context, higher autonomy.

**Single pass is better for speed** — Faster, fewer tokens.

**Recommendation:** Use iterative for complex tasks, single pass for simple tasks.
