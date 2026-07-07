# Full Benchmark: Ptahcortex vs Aider vs Claude Code (All MiMo)

## Test Results (MiMo v2.5)

### 1. Security Audit (OAuth2)

| Agent | Duration | Tokens | Quality |
|-------|----------|--------|---------|
| **Ptahcortex** | 19s | 2,668 | Good |
| **Aider** | 57s | N/A | Issues |
| **Claude Code** | 42s | N/A | Excellent |

### 2. JWT Validation Function

| Agent | Duration | Tokens | Quality |
|-------|----------|--------|---------|
| **Ptahcortex** | 20s | 2,937 | Good |
| **Aider** | 91s | N/A | Issues |
| **Claude Code** | 22s | N/A | Excellent |

### 3. Race Conditions

| Agent | Duration | Tokens | Quality |
|-------|----------|--------|---------|
| **Ptahcortex** | 20s | 3,141 | Good |
| **Aider** | 120s | N/A | Timeout |
| **Claude Code** | 72s | N/A | Good |

### 4. Refactor Auth Module

| Agent | Duration | Tokens | Quality |
|-------|----------|--------|---------|
| **Ptahcortex** | 20s | 2,710 | Good |
| **Aider** | N/A | N/A | Killed |
| **Claude Code** | 102s | N/A | Excellent |

### 5. Generate API Docs

| Agent | Duration | Tokens | Quality |
|-------|----------|--------|---------|
| **Ptahcortex** | 19s | 2,479 | Good |
| **Aider** | N/A | N/A | N/A |
| **Claude Code** | 42s | N/A | Excellent |

## Summary

| Metric | Ptahcortex | Aider | Claude Code |
|--------|------------|-------|-------------|
| **Average Duration** | 19.6s | 89.5s | 56s |
| **Success Rate** | 100% | 40% | 100% |
| **Quality** | Good | Issues | Excellent |
| **Consistency** | ✅ | ❌ | ✅ |

## Key Findings

### Ptahcortex (Fastest) 🏆
- ✅ **Fastest** — 19.6s average
- ✅ **Most consistent** — 100% success rate
- ✅ **Token efficient** — 2.4k-3.1k tokens
- ⚠️ **Quality** — Good but not best

### Aider (Issues)
- ❌ **Slow** — 89.5s average
- ❌ **Failures** — 60% failure rate
- ❌ **Summarization issues** — MiMo compatibility problems
- ⚠️ **Quality** — When it works, good

### Claude Code (Best Quality)
- ✅ **Quality** — Excellent findings
- ✅ **Success rate** — 100%
- ⚠️ **Speed** — 56s average (2.8x slower than Ptahcortex)
- ⚠️ **Token usage** — Unknown

## Recommendations

| Use Case | Best Agent |
|----------|------------|
| Speed-critical tasks | Ptahcortex |
| Quality-critical tasks | Claude Code |
| Production environments | Ptahcortex |
| Complex analysis | Claude Code |

## Conclusion

**Ptahcortex wins on speed** (19.6s vs 56s Claude Code).

**Claude Code wins on quality** (excellent findings).

**Aider has issues** with MiMo model compatibility.

**Recommendation:** Use Ptahcortex for speed, Claude Code for quality.
