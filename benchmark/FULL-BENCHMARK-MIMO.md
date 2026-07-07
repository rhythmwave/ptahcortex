# Full Benchmark: Aider vs Ptahcortex vs Claude Code (All MiMo)

## Test Scenarios

### 1. Security Audit (OAuth2)

| Agent | Duration | Quality |
|-------|----------|---------|
| **Aider** | ~30s | 8 findings (Critical, High, Medium) |
| **Ptahcortex** | 23s | 3+ findings (detailed) |
| **Claude Code** | 12s | 3 findings (High) |

### 2. JWT Validation Function

| Agent | Duration | Quality |
|-------|----------|---------|
| **Aider** | ~25s | Generated code with examples |
| **Ptahcortex** | 15s | Generated validation function |
| **Claude Code** | 15s | Provided npm install + code |

### 3. Race Conditions

| Agent | Duration | Quality |
|-------|----------|---------|
| **Aider** | ~20s | Found race conditions |
| **Ptahcortex** | 10s | Found race conditions |
| **Claude Code** | 121s | Timeout (too slow) |

### 4. Refactor Auth Module

| Agent | Duration | Quality |
|-------|----------|---------|
| **Aider** | ~30s | Refactoring suggestions |
| **Ptahcortex** | 16s | Modularization plan |
| **Claude Code** | 5s | Asked for clarification |

### 5. Generate API Docs

| Agent | Duration | Quality |
|-------|----------|---------|
| **Aider** | ~25s | Generated documentation |
| **Ptahcortex** | 19s | Generated documentation |
| **Claude Code** | 32s | Comprehensive docs |

## Summary

| Scenario | Aider | Ptahcortex | Claude Code |
|----------|-------|------------|-------------|
| Security Audit | 30s | 23s | 12s |
| JWT Validation | 25s | 15s | 15s |
| Race Conditions | 20s | 10s | 121s |
| Refactor Auth | 30s | 16s | 5s |
| API Docs | 25s | 19s | 32s |
| **Average** | **26s** | **16.6s** | **37s** |

## Token Usage (Ptahcortex)

| Scenario | Plan | Analyze | Total |
|----------|------|---------|-------|
| Security Audit | 819 | 1,761 | 2,580 |
| JWT Validation | 812 | 1,447 | 2,259 |
| Race Conditions | 818 | 770 | 1,588 |
| Refactor Auth | 817 | 1,447 | 2,264 |
| API Docs | 815 | 1,454 | 2,269 |
| **Average** | **816** | **1,376** | **2,192** |

## Key Findings

### 🏆 Winner by Category

| Category | Winner | Reason |
|----------|--------|--------|
| **Speed** | Ptahcortex | 16.6s average |
| **Quality** | Aider | Most detailed findings |
| **Consistency** | Ptahcortex | No timeouts |
| **Simple Tasks** | Claude Code | 5s for clarification |
| **Complex Tasks** | Aider | Best analysis |

### MiMo Performance

- **Plan phase:** 1.2-2.2s (very fast)
- **Analyze phase:** 6.4-19.9s (varies by complexity)
- **Total:** 5-121s (depends on task)

### Recommendations

| Use Case | Best Agent |
|----------|------------|
| Quick security audit | Ptahcortex (23s) |
| Code generation | Ptahcortex (15s) |
| Bug investigation | Ptahcortex (10s) |
| Complex refactoring | Aider (30s, better quality) |
| API documentation | Ptahcortex (19s) |
| Simple questions | Claude Code (5s) |

## Conclusion

**Ptahcortex wins on average speed** (16.6s vs 26s Aider vs 37s Claude Code).

**Aider wins on quality** for complex tasks.

**Claude Code wins on simple tasks** but fails on complex ones (121s timeout).

**Recommendation:** Use Ptahcortex for most tasks, Aider for complex analysis.
