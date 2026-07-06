# Ptahcortex Benchmark: GLM-5.1

**Date:** 2026-07-06
**Model:** GLM-5.1 via ai.sumopod.com
**Agent:** Ptahcortex with Context Manager

## Test: Code Review (Error Handling)

| Metric | Value |
|--------|-------|
| Total Tokens | 14,353 |
| Iterations | 3 |
| Tool Calls | 10 |
| Duration | 2m 45s |

### Token Breakdown

| Phase | Tokens | Percentage |
|-------|--------|------------|
| Plan | 8,345 | 58.1% |
| Sandbox | 14,722 | (tool execution) |
| Reflect | 3,222 | 22.4% |
| Final | 2,786 | 19.4% |

### Quality Output

7 detailed findings with specific code locations:

1. **Unbounded Read / DoS Vulnerability** (High)
2. **Potential Deadlock / Stale Lock** (High)
3. **Missing Response ID Validation** (High)
4. **Unsafe Type Assertion** (Medium)
5. **Ignored Start Errors** (Medium)
6. **Silent Failure on Stop** (Low)
7. **Missing Context Cancellation** (Low)
