# Ptahcortex + MiMo Benchmark Results

## Test: Security Audit (OAuth2)

### Ptahcortex + MiMo (Basic)

```
Task: Audit OAuth2 for security issues
Model: mimo-v2.5
Duration: 22.9 seconds
Tokens: 2,645 (833 plan + 1,812 analyze)
Findings: 3+ (JWT validation, token storage, etc.)
```

### Ptahcortex + MiMo (Lexa)

```
Task: Audit OAuth2 for security issues
Model: mimo-v2.5
Duration: 10.7 seconds
Tokens: 1,806 (995 plan + 811 analyze)
Findings: Detailed analysis
```

### Claude Code + MiMo

```
Task: Audit code for security vulnerabilities
Model: mimo-v2.5
Duration: 23 seconds
Findings: 8 (1 Critical, 2 High, 3 Medium, 2 Low)
```

## Comparison

| Metric | Ptahcortex (Basic) | Ptahcortex (Lexa) | Claude Code |
|--------|-------------------|-------------------|-------------|
| **Duration** | 22.9s | 10.7s | 23s |
| **Tokens** | 2,645 | 1,806 | ~2k (est.) |
| **Findings** | 3+ | Detailed | 8 |
| **Model** | mimo-v2.5 | mimo-v2.5 | mimo-v2.5 |

## Key Findings

### MiMo is VERY Fast
- **Plan phase:** 1.8-5 seconds (vs 10-15s for GLM-5.1)
- **Analyze phase:** 8-17 seconds (vs 30-40s for GLM-5.1)
- **Total:** 10-23 seconds (vs 52s-2min for GLM-5.1)

### Token Efficiency
- **Ptahcortex (Lexa):** 1,806 tokens (most efficient)
- **Ptahcortex (Basic):** 2,645 tokens
- **Claude Code:** ~2k tokens (estimated)

### Quality
- **Claude Code:** 8 findings (most)
- **Ptahcortex:** 3+ findings (detailed)

## Recommendation

**Use MiMo for:**
- Fast execution
- Quick reviews
- Cost efficiency

**Use GLM-5.1 for:**
- Deeper analysis
- More findings
- Complex tasks

## Next Steps

1. **Test with more complex tasks** — Refactoring, bug investigation
2. **Compare token costs** — MiMo vs GLM-5.1
3. **Optimize Ptahcortex** — Reduce token usage further
