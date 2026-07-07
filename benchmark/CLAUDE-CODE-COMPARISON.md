# Ptahcortex vs Claude Code Benchmark

## Test: Security Audit (Same Task)

### Claude Code + MiMo

```
Task: Audit code for security vulnerabilities
Model: mimo-v2.5
Duration: 23 seconds
Findings: 8 (1 Critical, 2 High, 3 Medium, 2 Low)
Code Patches: ✅ Yes (detailed)
```

### Ptahcortex + GLM-5.1

```
Task: Audit OAuth2 for security issues
Model: glm-5.1
Duration: 52 seconds (basic) / 2 minutes (Lexa)
Findings: 5-7 (2-3 High, 2 Medium, 1-2 Low)
Code Patches: ✅ Yes (detailed)
```

## Comparison

| Metric | Claude Code + MiMo | Ptahcortex (Basic) | Ptahcortex (Lexa) |
|--------|-------------------|-------------------|-------------------|
| **Duration** | 23s | 52s | 2 min |
| **Findings** | 8 | 5-7 | 5-7 |
| **High Severity** | 2 | 2-3 | 2-3 |
| **Code Patches** | ✅ | ✅ | ✅ |
| **Token Usage** | ~2k (est.) | 2,465 | ~3k |

## Key Findings

### Claude Code + MiMo
- ✅ **Fastest** (23 seconds)
- ✅ **Most findings** (8 issues)
- ✅ **Detailed code patches**
- ✅ **Uses MiMo endpoint**

### Ptahcortex
- ✅ **OTel observability** (traces, metrics)
- ✅ **Config-driven** (YAML agents)
- ✅ **MCP ecosystem** (22 tools)
- ✅ **Lexa integration** (code intelligence)

## Honest Assessment

**Claude Code + MiMo is faster and finds more issues** on simple tasks.

**Ptahcortex advantages:**
1. **Observability** — Full OTel traces
2. **Extensibility** — MCP tool ecosystem
3. **Configuration** — YAML-driven agents
4. **Production-ready** — Single binary, Go performance

**Claude Code advantages:**
1. **Speed** — Faster execution
2. **Quality** — More findings
3. **Simplicity** — Works out of box

## Recommendation

**Use Claude Code + MiMo for:**
- Quick security audits
- Simple code reviews
- When speed matters

**Use Ptahcortex for:**
- Production environments
- Complex multi-step tasks
- When observability matters
- When MCP tools needed

## Next Steps

1. **Test with more complex tasks** — Refactoring, bug investigation
2. **Measure actual token usage** — Not just estimates
3. **Compare with Aider** — Three-way comparison
4. **Optimize Ptahcortex** — Reduce token usage
