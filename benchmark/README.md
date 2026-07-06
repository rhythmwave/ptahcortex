# Ptahcortex Benchmark Suite

Comparing Ptahcortex's Context Manager against other OpenAI-compatible coding agents.

## Agents Compared

| Agent | Architecture | Context Strategy |
|-------|-------------|------------------|
| **Ptahcortex** | Sandboxed tool reasoning | Call-type-aware assembly (T0-T4) |
| **OpenCode** | Direct tool calling | Raw history accumulation |
| **Hermes** | Function calling | Sliding window |
| **Codex** | Multi-turn chat | Full context |

All agents use the same:
- **Model:** GLM-5.1 via ai.sumopod.com
- **MCP Server:** Lexa (22 code intelligence tools)
- **Target:** commit-reviewer-src codebase

## Benchmark Tasks

### Task 1: File Discovery (Easy)
```
List all Go files in the project
```
**Expected:** List of .go files with paths
**Tools:** files, glob

### Task 2: Symbol Search (Easy)
```
Find all functions named 'Call' or 'Start' in the codebase
```
**Expected:** Function definitions with locations
**Tools:** symbol_search, outline

### Task 3: Code Review (Medium)
```
Review error handling in the MCP client code
```
**Expected:** Structured review with findings
**Tools:** read, outline, text_search, callers

### Task 4: Dependency Analysis (Medium)
```
Trace all callers of the CallTool function
```
**Expected:** Call graph with locations
**Tools:** callers, trace_deps, read

### Task 5: Bug Investigation (Hard)
```
Find potential race conditions in concurrent code
```
**Expected:** Specific locations and explanations
**Tools:** text_search, read, outline, callers

## Metrics

| Metric | Why It Matters |
|--------|---------------|
| **Total Tokens** | Cost efficiency |
| **Tool Calls** | How many MCP invocations |
| **Duration** | Wall-clock time |
| **Iterations** | Agent loop count |
| **Quality Score** | Correctness (1-5) |
| **Token Efficiency** | Quality / Tokens (higher = better) |

## Running

```bash
# Run Ptahcortex benchmark
./run-ptahcortex.sh glm-5.1

# Run OpenCode benchmark (requires opencode installed)
./run-opencode.sh glm-5.1

# Compare results
./compare.sh
```

## Expected Results

Based on architecture analysis:

| Agent | Token Efficiency | Quality | Why |
|-------|-----------------|---------|-----|
| **Ptahcortex** | High | High | Sandboxed reasoning reduces noise |
| **OpenCode** | Medium | High | Direct but accumulates context |
| **Hermes** | Medium | Medium | Sliding window loses context |
| **Codex** | Low | High | Full context is expensive |

## Results

Results saved to `results/` as JSON.
