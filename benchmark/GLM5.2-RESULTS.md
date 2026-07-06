# Ptahcortex Benchmark: GLM-5.2

**Date:** 2026-07-06
**Model:** GLM-5.2 via ai.sumopod.com
**Agent:** Ptahcortex with Context Manager

## Test: Code Review (Error Handling)

| Metric | Value |
|--------|-------|
| Total Tokens | 11,831 |
| Iterations | 3 |
| Tool Calls | 10 |
| Duration | 1m 31s |

### Token Breakdown

| Phase | Tokens | Percentage |
|-------|--------|------------|
| Plan | 8,334 | 70.4% |
| Sandbox | 15,323 | (tool execution) |
| Reflect | 2,415 | 20.4% |
| Final | 1,082 | 9.1% |

### Quality Output

- Found correct file (internal/mcp/client.go)
- Read file structure
- Did not complete full analysis (fewer iterations)
